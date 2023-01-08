package main

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"sync"
)

type Stream struct {
	ftyp        Packet
	moov        Packet
	PcktStreams map[string]PacketStream // One packetStream for each client connected through the suuid
	mutex       sync.RWMutex
}

func (s *Stream) addFtype(pckt Packet) {
	s.ftyp = pckt
}

func (s *Stream) addMoov(pckt Packet) {
	s.moov = pckt
}

//	func (s *Stream) addClient(suuid string) {
//		s.mutex.Lock()
//		defer s.mutex.Unlock()
//
//		s.PcktStreams[suuid] = PacketStream{}
//	}
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
	pkt := make(chan Packet, 300)

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

func (s *Streams) putFtyp(suuid string, pckt Packet) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	var retVal error = nil
	stream, ok := s.StreamMap[suuid]
	if ok {
		stream.ftyp = pckt
		s.StreamMap[suuid] = stream
	} else {
		retVal = fmt.Errorf("Stream ", suuid, " not found")
	}
	return retVal
}

func (s *Streams) putMoov(suuid string, pckt Packet) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	var retVal error = nil
	stream, ok := s.StreamMap[suuid]
	if ok {
		stream.moov = pckt
		s.StreamMap[suuid] = stream
	} else {
		retVal = fmt.Errorf("Stream ", suuid, " not found")
	}
	return retVal
}

func (s *Streams) getFtyp(suuid string) (error, Packet) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	var retVal error = nil
	stream, ok := s.StreamMap[suuid]
	if !ok {
		retVal = fmt.Errorf("Stream ", suuid, " not found")
	} else if stream.ftyp.pckt == nil {
		retVal = fmt.Errorf("No ftyp for stream ", suuid)
	}
	return retVal, stream.ftyp

}

func (s *Streams) getMoov(suuid string) (error, Packet) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	var retVal error = nil
	stream, ok := s.StreamMap[suuid]
	if !ok {
		retVal = fmt.Errorf("Stream ", suuid, " not found")
	} else if stream.moov.pckt == nil {
		retVal = fmt.Errorf("No moov for stream ", suuid)
	}
	return retVal, stream.moov
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

func (p Packet) isKeyFrame() (retVal bool) {
	// [moof [mfhd] [traf [tfhd] [tfdt] [trun]]]
	retVal = false
	traf := getSubBox(p, "traf")
	if traf == nil {
		retVal = false
	}

	trun := getSubBox(Packet{pckt: traf}, "trun")
	if trun == nil {
		retVal = false
	}
	flags := trun[10:14]

	retVal = flags[1]&4 == 4
	return
}

func getBox(pckt Packet, index int) (len int, name string) {
	len = int(binary.BigEndian.Uint32(pckt.pckt[index : index+4]))
	name = string(pckt.pckt[index+4 : index+8])
	return
}

func getSubBox(pckt Packet, boxName string) (sub_box []byte) {
	index := 0
	len, _ := getBox(pckt, index)

	index += 8

	for i := index; i < len; {
		length, nam := getBox(pckt, i)

		if nam == boxName {
			sub_box = Packet{pckt: pckt.pckt[i : i+length]}.pckt
		}
		i += length
	}
	return
}

func pseudoUUID() (uuid string) {
	const pseudoUUIDLen int = 16
	b := make([]byte, pseudoUUIDLen)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	uuid = fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return
}
