package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/websocket"
	"log"
	"net/http"
	"time"
)

var streams = NewStreams()

func serveHTTP() {
	router := gin.Default()
	gin.SetMode(gin.DebugMode)
	router.LoadHTMLFiles("../web/index.html")

	// For web page
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			//	"suuid": c.Param("suuid"),
		})
	})
	// For ffmpeg to write to
	router.POST("/live/:suuid", func(c *gin.Context) {
		var req = c.Request
		suuid := req.FormValue("suuid")
		readCloser := req.Body
		streams.addInput(suuid)
		defer streams.removeInput(suuid)

		// TODO: Need to find the most efficient way to get a clean buffer
		data := make([]byte, 100000)
		pcktCount := 0

		for {
			data = data[:100000]
			numOfByte, err := readCloser.Read(data)
			if err != nil {
				log.Println("Error in read loop for stream ", suuid, ":- ", err.Error())
				break
			}
			data = data[:numOfByte]
			d := NewPacket(data[:numOfByte]) //make([]byte, numOfByte)
			if pcktCount == 0 {
				err = streams.putFtyp(suuid, d)
				if err != nil {
					return
				}
				pcktCount++
			} else if pcktCount == 1 {
				err = streams.putMoov(suuid, d)
				if err == nil {
					err, _ := streams.getCodecsFromMoov(suuid)
					if err != nil {
						return
					}
					pcktCount++
				}
			} else {
				err = streams.put(suuid, d)
			}

			if err != nil {
				log.Println("Error: " + err.Error())
				break
			}
			//log.Println(numOfByte, " bytes received")
		}
	})

	// For http connections from ffmpeg
	// This does not send the codec info ahead of ftyp and moov
	router.GET("/h/:suuid", func(c *gin.Context) {
		ServeHTTPStream(c.Writer, c.Request)
	})

	// For websocket connections from mse
	router.GET("/ws/:suuid", func(c *gin.Context) {
		handler := websocket.Handler(ws)
		handler.ServeHTTP(c.Writer, c.Request)
	})
	err := router.Run(":8081")
	if err != nil {
		log.Fatalln(err)
	}
}

func ServeHTTPStream(w http.ResponseWriter, r *http.Request) {
	defer func() { r.Close = true }()
	suuid := r.FormValue("suuid")

	log.Println("Request", suuid)
	//if !Config.ext(suuid) {
	//	log.Println("Stream Not Found")
	//	return
	//}
	cuuid, ch := streams.addClient(suuid)
	log.Printf("number of cuuid's = %d", len(streams.StreamMap[suuid].PcktStreams))
	defer streams.deleteClient(suuid, cuuid)
	//err = websocket.Message.Send(ws, append([]byte{9}, "codecs"...))

	err, data := streams.getFtyp(suuid)
	if err != nil {
		log.Printf("Error getting ftyp: %s", err.Error())
	}
	_, err = w.Write(data.pckt)
	if err != nil {
		log.Printf("Error writing ftyp: %s", err.Error())
		return
	}
	err, data = streams.getMoov(suuid)
	if err != nil {
		log.Printf("Error getting moov: %s", err.Error())
	}
	_, err = w.Write(data.pckt)
	if err != nil {
		log.Printf("Error writing moov: %s", err.Error())
	}

	started := false
	for {
		var data Packet

		data = <-ch
		if !started && !data.isKeyFrame() {
			continue
		} else {
			started = true
			_, err = w.Write(data.pckt)
			if err != nil {
				break
			}
		}
	}
}

func ws(ws *websocket.Conn) {
	defer func() {
		err := ws.Close()
		if err != nil {
			_ = fmt.Errorf("Error closing websocket %s", err.Error())
		}
	}()
	suuid := ws.Request().FormValue("suuid")

	log.Println("Request", suuid)
	//if !Config.ext(suuid) {
	//	log.Println("Stream Not Found")
	//	return
	//}
	err := ws.SetWriteDeadline(time.Now().Add(50 * time.Second))
	cuuid, ch := streams.addClient(suuid)
	defer streams.deleteClient(suuid, cuuid)
	//err = websocket.Message.Send(ws, init)
	var data Packet
	err, data = streams.getCodecs(suuid)
	if err != nil {
		log.Printf("Error getting codecs: %s", err.Error())
		return
	}
	err = websocket.Message.Send(ws, data.pckt)
	if err != nil {
		log.Printf("Error writing codecs: %s", err.Error())
		return
	}
	err, data = streams.getFtyp(suuid)
	if err != nil {
		log.Printf("Error getting ftyp: %s", err.Error())
		return
	}
	err = websocket.Message.Send(ws, data.pckt)
	if err != nil {
		log.Printf("Error writing ftyp: %s", err.Error())
		return
	}

	err, data = streams.getMoov(suuid)
	if err != nil {
		log.Printf("Error getting moov: %s", err.Error())
	}
	err = websocket.Message.Send(ws, data.pckt)
	if err != nil {
		log.Printf("Error writing moov: %s", err.Error())
	}

	// Main loop to send moof and mdat atoms
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

	started := false
	for {
		var err error
		data = <-ch
		if !started && !data.isKeyFrame() {
			continue
		} else {
			started = true
		}

		//log.Println("Data received ", len(data.pckt), " bytes")
		err = ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err != nil {
			return
		}
		err = websocket.Message.Send(ws, data.pckt)
		if err != nil {
			return
		}
	}
}
