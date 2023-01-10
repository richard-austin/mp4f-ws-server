package main

import (
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
	router.LoadHTMLGlob("../web/*")

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
			log.Println(numOfByte, " bytes received")
		}
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

func ws(ws *websocket.Conn) {

	defer ws.Close()
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
	if err != nil {
		return
	}
	//var start bool
	//err = websocket.Message.Send(ws, append([]byte{9}, "codecs"...))
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

	var pcktCount int = 0
	started := false
	for {
		var err error
		var data Packet

		if pcktCount == 0 {
			err, data = streams.getCodecs(suuid)
			if err != nil {
				log.Printf("Error getting codecs: %s", err.Error())
			}
			pcktCount++
		} else if pcktCount == 1 {
			err, data = streams.getFtyp(suuid)
			if err != nil {
				log.Printf("Error getting ftyp: %s", err.Error())
			}
			pcktCount++
		} else if pcktCount == 2 {
			err, data = streams.getMoov(suuid)
			if err != nil {
				log.Printf("Error getting moov: %s", err.Error())
			}
			pcktCount++
		} else {
			data = <-ch
			if !started && !data.isKeyFrame() {
				continue
			} else {
				started = true
			}
		}

		log.Println("Data received ", len(data.pckt), " bytes")
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
