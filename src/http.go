package main

import (
	"encoding/binary"
	"fmt"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/websocket"
	"io"
	"log"
	"net/http"
	"time"
)

var streams = NewStreams()

/*
 ReadBox
 Sometimes the ftyp and moov atoms at the start of the stream from ffmpeg  are combined in one packet. This
	function separates them if this occurs, so they can be put in their respective places and the moov
	atom analyes to get the codec data. THis is only used to handle those first two messages. From then on it doesn't
	matter if messages get appended to each other as they are going straight to mse (or ffmpeg for recordings)
*/

func ReadBox(readCloser io.ReadCloser, data []byte, queue chan Packet) (numOfByte int, err error) {
	numOfByte = 0
	if len(queue) > 0 {
		pckt := <-queue
		copy(data, pckt.pckt)
		var boxLen = binary.BigEndian.Uint32(data[:4])
		if boxLen > uint32(len(data)) {
			lenData := len(data)
			numOfByte, err = readCloser.Read(data[lenData:])
			if err != nil {
				return
			}
			numOfByte += lenData
		}
	} else {
		numOfByte, err = readCloser.Read(data[:cap(data)])
		if err != nil {
			return
		}
	}
	var boxLen = binary.BigEndian.Uint32(data[0:4])
	if boxLen < uint32(numOfByte) {
		var tmp = make([]byte, uint32(numOfByte)-boxLen)
		copy(tmp, data[boxLen:uint32(numOfByte)-boxLen])
		queue <- NewPacket(tmp)
		fmt.Printf("splitting packet boxLen = %d, numOfByte = %d\n", boxLen, numOfByte)
		data = data[:boxLen]
	}
	numOfByte = int(boxLen)
	return
}

func serveHTTP() {
	router := gin.Default()
	gin.SetMode(gin.DebugMode)
	router.LoadHTMLFiles("web/index.html")

	// For web page
	router.GET("/", func(c *gin.Context) {
		//path, err := os.Getwd()
		//if err != nil {
		//	log.Println(err)
		//}
		//fmt.Println(path)
		c.HTML(http.StatusOK, "index.html", gin.H{
			//	"suuid": c.Param("suuid"),
		})
	})
	// For ffmpeg to write to
	router.POST("/live/:suuid", func(c *gin.Context) {
		var req = c.Request
		suuid := req.FormValue("suuid")
		_, hasEntry := streams.StreamMap[suuid]
		if hasEntry {
			log.Printf("Cannot add %s, there is already an existing stream with that id", suuid)
			return
		}
		readCloser := req.Body

		streams.addInput(suuid)
		defer streams.removeInput(suuid)

		// TODO: Need to find the most efficient way to get a clean buffer
		data := make([]byte, 33000)
		queue := make(chan Packet, 1)

		// Set up the stream ready for connection from client, put in the ftyp, moov and codec data
		numOfByte, err := ReadBox(readCloser, data, queue)
		if err != nil {
			log.Println("Error reading the ftyp data for stream ", suuid, ":- ", err.Error())
			return
		}

		d := NewPacket(data[:numOfByte]) //make([]byte, numOfByte)
		err = streams.putFtyp(suuid, d)
		if err != nil {
			return
		}

		numOfByte, err = ReadBox(readCloser, data, queue)
		if err != nil {
			log.Println("Error reading the moov data for stream ", suuid, ":- ", err.Error())
			return
		}

		d = NewPacket(data[:numOfByte])
		err = streams.putMoov(suuid, d)
		if err == nil {
			err, _ := streams.getCodecsFromMoov(suuid)
			if err != nil {
				return
			}
		}
		// Empty the queue
		for len(queue) > 0 {
			_ = <-queue
		}
		for {
			data = data[:33000]
			numOfByte, err = readCloser.Read(data)
			d = NewPacket(data[:numOfByte])
			err = streams.put(suuid, d)

			if err != nil {
				log.Println("Error: " + err.Error())
				break
			} else if numOfByte == 0 {
				break
			}
			//log.Println(numOfByte, " bytes received")
		}
	})

	// For http connections from ffmpeg to read from (for recordings)
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
