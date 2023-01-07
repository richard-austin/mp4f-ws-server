package main

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/net/websocket"
	"log"
	"time"
)

var ch = make(chan Packet)

func serveHTTP() {
	router := gin.Default()
	gin.SetMode(gin.DebugMode)

	router.POST("/live/s1", func(c *gin.Context) {
		var req = c.Request
		readCloser := req.Body
		// TODO: Need to find the most efficient way to get a clean buffer
		data := make([]byte, 100000)
		for {
			data = data[:100000]
			numOfByte, err := readCloser.Read(data)
			data = data[:numOfByte]
			d := NewPacket(data[:numOfByte]) //make([]byte, numOfByte)
			//copy(d, data) // TODO: Would copy introduce inefficiencies?
			ch <- d
			if err != nil {
				log.Println("Error: " + err.Error())
				break
			}
			log.Println(numOfByte, " bytes received")
			//	c.Writer.WriteHeader(http.StatusOK)
		}
	})

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
				ws.Close()
				return
			}
		}
	}()

	for {
		data := <-ch

		log.Println("Data received ", len(data.pckt), " bytes")
		err = ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err != nil {
			return
		}
		err := websocket.Message.Send(ws, data.pckt)
		if err != nil {
			return
		}
	}
	//noVideo := time.NewTimer(10 * time.Second)
	//for {
	//	select {
	//	case <-noVideo.C:

	//		log.Println("noVideo")
	//		return
	//	case pck := <-ch:
	//		if pck.IsKeyFrame {
	//			noVideo.Reset(10 * time.Second)
	//			start = true
	//		}
	//		if !start {
	//			continue
	//		}
	//		if ready {
	//			err = ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
	//			if err != nil {
	//				return
	//			}
	//			err := websocket.Message.Send(ws, buf)
	//			if err != nil {
	//				return
	//			}
	//		}
	//	}
	// }
}
