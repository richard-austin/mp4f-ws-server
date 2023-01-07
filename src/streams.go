package main

import (
	"crypto/rand"
	"fmt"
	"sync"
)

type Stream struct {
	PcktStreams map[string]PacketStream // One packetStream for each client connected through the suuid
	mutex       sync.RWMutex
}

func (s *Stream) addClient(suuid string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.PcktStreams[suuid] = PacketStream{}
}

type StreamMap map[string]Stream
type Streams struct {
	mutex sync.RWMutex
	StreamMap
}

func NewStreams() *Streams {
	s := Streams{}
	s.StreamMap = StreamMap{}

	return &s
}

func (s *Streams) addInput(suuid string) {
	s.StreamMap[suuid] = Stream{PcktStreams: map[string]PacketStream{}}
}

func (s *Streams) removeInput(suuid string) {
	delete(s.StreamMap, suuid)
}

func (s *Streams) addClient(suuid string) (string, chan Packet) {
	cuuid := ""
	pkt := make(chan Packet)

	stream, ok := s.StreamMap[suuid]
	if ok {
		id := pseudoUUID()
		stream.PcktStreams[id] = PacketStream{}
		cuuid = id
		stream.PcktStreams[cuuid] = PacketStream{ps: pkt}
	} else {
		pkt = nil
	}
	return cuuid, pkt
}

func (s *Streams) deleteClient(suuid string, cuuid string) {
	stream, ok := s.StreamMap[suuid]
	if ok {
		delete(stream.PcktStreams, cuuid)
	}
}

func (s *Streams) put(suuid string, pckt Packet) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	var retVal error = nil
	stream, ok := s.StreamMap[suuid]
	if ok {
		for _, packetStream := range stream.PcktStreams {
			packetStream.ps <- pckt
		}

	} else {
		retVal = fmt.Errorf("No stream with name ", suuid, " was found")
	}
	return retVal
}

type PacketStream struct {
	ps chan Packet
}
type Packet struct {
	pckt []byte
}

func NewPacket(pckt []byte) Packet {
	b := make([]byte, len(pckt))
	copy(b, pckt)
	return Packet{pckt: b}
}

func pseudoUUID() (uuid string) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	uuid = fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return
}
