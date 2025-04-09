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
			"suuidMap":            suuids,
			"suuid":               firstStream,
			"defaultLatencyLimit": config.DefaultLatencyLimit,
		})
	})

	// For web page with suuid
	router.GET("/:suuid", func(c *gin.Context) {

		c.HTML(http.StatusOK, "index.gohtml", gin.H{
			"suuidMap":            suuids,
			"suuid":               c.Param("suuid"),
			"defaultLatencyLimit": config.DefaultLatencyLimit,
		})
	})

	// For ffmpeg to write to
	router.POST("/live/:suuid", func(c *gin.Context) {
		req := c.Request
		suuid := req.FormValue("suuid")
		_, hasEntry := streams.StreamMap[suuid]
		if hasEntry {
			log.Errorf("Cannot add %s, there is already an existing stream with that id", suuid)
			return
		}

		log.Infof("Input connected for %s", suuid)
		readCloser := req.Body

		streams.addStream(suuid)
		defer streams.removeStream(suuid)

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
			err = streams.put(suuid, d)
			if err != nil {
				log.Errorf("Error putting the packet into stream %s:- %s", suuid, err.Error())
				break
			} else if numOfByte == 0 {
				break
			}
			log.Tracef("%d bytes received", numOfByte)
		}
	})
	log.Info("Point 1")
	router.StaticFS("/web", http.Dir("web"))

	// For http connections from ffmpeg to read from (for recordings)
	// This does not send the codec info
	router.GET("/h/:suuid", func(c *gin.Context) {
		ServeHTTPStream(c.Writer, c.Request)
	})
	log.Info("Point 2")
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

func ServeHTTPStream(w http.ResponseWriter, r *http.Request) {
	log.Info("In ServeHTTPStream")

	defer func() { r.Close = true }()
	suuid := r.FormValue("suuid")

	log.Infof("Request %s", suuid)
	cuuid, ch := streams.addClient(suuid)
	if ch == nil {
		return
	}
	log.Infof("number of cuuid's = %d", len(streams.StreamMap[suuid].PcktStreams))
	defer streams.deleteClient(suuid, cuuid)

	//	started := false
	//	stream := streams.StreamMap[suuid]
	//gopCache := stream.gopCache.GetCurrent()
	//gopCacheUsed := stream.gopCache.GopCacheUsed
	for {
		var data Packet

		//if gopCacheUsed {
		//	data = gopCache.Get(ch)
		//	started = true
		//} else {
		data = <-ch
		//if !started {
		//	if data.isKeyFrame() {
		//		started = true
		//	} else {
		//		continue
		//	}
		//}
		//}
		bytes, err := w.Write(data.pckt)
		if err != nil {
			// Warning only as it could be because the client disconnected
			log.Warnf("writing to client for %s:= %s", suuid, err.Error())
			break
		}
		log.Tracef("Data sent to http client for %s:- %d bytes", suuid, bytes)

	}
}

func ws(ws *websocket.Conn) {
	defer func() {
		err := ws.Close()
		if err != nil {
			log.Warnf("closing websocket:- %s", err.Error())
		}
	}()
	suuid := ws.Request().FormValue("suuid")

	log.Infof("Request %s", suuid)
	err := ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		log.Errorf("Error in SetWriteDeadline %s", err.Error())
		return
	}
	cuuid, ch := streams.addClient(suuid)
	if ch == nil {
		return
	}
	defer streams.deleteClient(suuid, cuuid)
	log.Infof("number of cuuid's = %d", len(streams.StreamMap[suuid].PcktStreams))

	// Send the header information (codecs, ftyp and moov)
	var data Packet
	if !strings.HasSuffix(suuid, "a") {
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

	stream := streams.StreamMap[suuid]
	gopCache := stream.gopCache.GetSnapshot()
	gopCacheUsed := stream.gopCache.GopCacheUsed
	// Main loop to
	started := false
	for {
		if gopCacheUsed {
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
