package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/websocket"
	"net/http"
	"strings"
	"time"
)

var streams = NewStreams()

func serveHTTP() {
	router := gin.Default()
	gin.SetMode(gin.DebugMode)
	router.LoadHTMLFiles("web/index.gohtml")
	suuids := cameras.Suuids()
	// Get the name of the first stream
	var first Camera
	var firstStream string
	for _, first = range cameras.Cameras {
		for firstStream = range first.Streams {
			break
		}
		break
	}
	// For web page without suuid
	router.GET("/", func(c *gin.Context) {

		c.HTML(http.StatusOK, "index.gohtml", gin.H{
			"suuidMap": suuids,
			"suuid":    firstStream,
		})
	})

	// For web page with suuid
	router.GET("/:suuid", func(c *gin.Context) {

		c.HTML(http.StatusOK, "index.gohtml", gin.H{
			"suuidMap": suuids,
			"suuid":    c.Param("suuid"),
		})
	})

	// For ffmpeg to write to for live streaming (with suuid)
	router.POST("/live/:suuid", func(c *gin.Context) {
		req := c.Request
		suuid := req.FormValue("suuid")

		baseSuuid, isAudio := strings.CutSuffix(suuid, "a")

		stream, hasEntry := streams.StreamMap[baseSuuid]
		if hasEntry && (isAudio && stream.hasAudio || !isAudio && stream.hasVideo) {
			log.Errorf("Cannot add %s, there is already an existing stream with that id and media type", suuid)
			return
		}

		log.Infof("Input connected for %s", suuid)
		readCloser := req.Body

		streams.addStream(baseSuuid, isAudio)
		defer streams.removeStream(baseSuuid)

		data := make([]byte, 33000)

		d := NewPacket(data) //make([]byte, numOfByte)
		for {
			data = data[:33000]
			numOfByte, err := readCloser.Read(data)
			if err != nil {
				log.Errorf("Error reading the data feed for stream %s:- %s", suuid, err.Error())
				break
			}
			d = NewPacket(data[:numOfByte])

			if err != nil {
				log.Error(err)
			}
			err = streams.put(baseSuuid, d, isAudio)
			if err != nil {
				log.Errorf("Error putting the packet into stream %s:- %s", suuid, err.Error())
				break
			} else if numOfByte == 0 {
				break
			}
			log.Tracef("%d bytes received", numOfByte)
		}
	})

	// For ffmpeg to write to for recording (with rsuuid)
	router.POST("/recording/:rsuuid", func(c *gin.Context) {
		req := c.Request
		rsuuid := req.FormValue("rsuuid")

		_, hasEntry := streams.StreamMap[rsuuid]
		if hasEntry {
			log.Errorf("Cannot add %s, there is already an existing stream with that id and media type", rsuuid)
			return
		}

		log.Infof("Recording input connected for %s", rsuuid)
		readCloser := req.Body

		streams.addStream(rsuuid, false, true)
		defer streams.removeStream(rsuuid)

		data := make([]byte, 33000)

		d := NewPacket(data) //make([]byte, numOfByte)
		for {
			data = data[:33000]
			numOfByte, err := readCloser.Read(data)
			if err != nil {
				log.Errorf("Error reading the data feed for stream %s:- %s", rsuuid, err.Error())
				break
			}
			d = NewPacket(data[:numOfByte])

			if err != nil {
				log.Error(err)
			}
			err = streams.put(rsuuid, d, false)
			if err != nil {
				log.Errorf("Error putting the packet into stream %s:- %s", rsuuid, err.Error())
				break
			} else if numOfByte == 0 {
				break
			}
			log.Tracef("%d bytes received", numOfByte)
		}
	})

	router.StaticFS("/web", http.Dir("web"))

	// For http connections from ffmpeg to read from (for recordings)
	// This is the mpegts stream
	router.GET("/h/:rsuuid", func(c *gin.Context) {
		ServeHTTPStream(c.Writer, c.Request)
	})

	// For websocket connections
	router.GET("/ws/:suuid", func(c *gin.Context) {
		handler := websocket.Handler(ws)
		handler.ServeHTTP(c.Writer, c.Request)
	})

	addr := fmt.Sprintf(":%d", config.ServerPort)
	err := router.Run(addr)
	if err != nil {
		log.Errorln(err)
	}
}

// ServeHTTPStream For recording from
// Recording command example which seems to work well. The tempo filter compensates for the tempo filter used to keep libe audio and video in sync:-
// ffmpeg -y -f alaw -ar 48000 -i http://localhost:8081/h/stream?suuid=cam1-stream1a -f h264 -i http://localhost:8081/h/stream?suuid=cam1-stream1 -f mp4 -af atempo=0.9804 test.mp4
// or to correct the frame rate (PTZ camera) Not sure if genpts before the audio input makes any odds.
// ffmpeg -y -f alaw -fflags +genpts -i http://localhost:8081/h/stream?suuid=cam1-stream1a -f hevc -fflags +genpts -r 11 -i http://localhost:8081/h/stream?suuid=cam1-stream1 -f mp4 test.mp4
func ServeHTTPStream(w http.ResponseWriter, r *http.Request) {
	log.Info("In ServeHTTPStream")

	defer func() { r.Close = true }()
	rsuuid := r.FormValue("rsuuid")

	log.Infof("Request %s", rsuuid)
	cuuid, ch := streams.addClient(rsuuid, false)
	if ch == nil {
		return
	}
	log.Infof("number of cuuid's = %d", len(streams.StreamMap[rsuuid].PcktStreams))
	defer streams.deleteClient(rsuuid, cuuid, false)

	for {
		var data Packet
		data = <-ch
		bytes, err := w.Write(data.pckt)
		if err != nil {
			// Warning only as it could be because the client disconnected
			log.Warnf("writing to client for %s:= %s", rsuuid, err.Error())
			break
		}
		log.Tracef("Data sent to http client for %s:- %d bytes", rsuuid, bytes)
	}
}

// ws For live streaming connection
func ws(ws *websocket.Conn) {
	defer func() {
		err := ws.Close()
		log.Warn("Closing the websocket")
		if err != nil {
			log.Warnf("closing websocket:- %s", err.Error())
		}
	}()
	suuid := ws.Request().FormValue("suuid")
	baseSuuid, isAudio := strings.CutSuffix(suuid, "a")

	log.Infof("Request %s", suuid)
	err := ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		log.Errorf("Error in SetWriteDeadline %s", err.Error())
		return
	}
	cuuid, ch := streams.addClient(baseSuuid, isAudio)
	if ch == nil {
		return
	}
	defer streams.deleteClient(baseSuuid, cuuid, isAudio)
	log.Infof("number of cuuid's = %d", len(streams.StreamMap[baseSuuid].PcktStreams))

	// Send the header information (codec)
	var data Packet
	if !isAudio {
		err, data = streams.getCodec(suuid)
		if err != nil {
			log.Errorf("Error getting codecs: %s", err.Error())
			return
		}
		err = websocket.Message.Send(ws, data.pckt)
		if err != nil {
			log.Errorf("Error writing codec: %s", err.Error())
			return
		}
	}

	go func() {
		for {
			var message string
			err := websocket.Message.Receive(ws, &message)
			if err != nil {
				_ = ws.Close()
				return
			}
		}
	}()

	stream := streams.StreamMap[baseSuuid]
	var gopCache *GopCacheSnapshot
	if !isAudio { // Audio GOP cache not used for live streams, only recordings
		gopCache = stream.gopCache.GetSnapshot()
	}
	gopCacheUsed := stream.gopCache.GopCacheUsed
	// Main loop to send data to the browser
	started := isAudio // Always started for audio as we don't wait for a keyframe
	for {
		if gopCacheUsed && !isAudio {
			data = gopCache.Get(ch)
			started = true
		} else {
			data = <-ch
			if !started {
				if data.isKeyFrame() {
					started = true
				} else {
					continue
				}
			}
		}

		err = ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err != nil {
			log.Warnf("calling SetWriteDeadline:- %s", err.Error())
			return
		}
		err = websocket.Message.Send(ws, data.pckt)
		if err != nil {
			log.Warnf("calling Send:- %s", err.Error())
			return
		}
		log.Tracef("Data sent to client %d bytes", len(data.pckt))
	}
}
