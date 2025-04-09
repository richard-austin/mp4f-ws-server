package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	log "github.com/sirupsen/logrus"
	"sync"
)

type Stream struct {
	hasAudio         bool
	hasVideo         bool
	gopCache         GopCache
	PcktStreams      map[string]*PacketStream // One packetStream for each client connected through the suuid
	AudioPcktStreams map[string]*PacketStream // Separate set of streams for audio
}
type StreamMap map[string]*Stream
type Streams struct {
	mutex sync.RWMutex
	StreamMap
}

func NewStreams() *Streams {
	s := Streams{}
	s.StreamMap = StreamMap{}

	return &s
}

func (s *Streams) addStream(suuid string, isAudio bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	stream := &Stream{PcktStreams: map[string]*PacketStream{}, AudioPcktStreams: map[string]*PacketStream{}, gopCache: NewGopCache(config.GopCache)}
	if isAudio {
		stream.hasAudio = true
	} else {
		stream.hasVideo = true
	}

	s.StreamMap[suuid] = stream
}

func (s *Streams) removeStream(suuid string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	_, ok := s.StreamMap[suuid]
	if ok {
		delete(s.StreamMap, suuid)
	}
}

func (s *Streams) addClient(suuid string, isAudio bool) (cuuid string, pkt chan Packet) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	stream, ok := s.StreamMap[suuid]
	if ok {
		cuuid = pseudoUUID()
		pktStream := NewPacketStream()
		if !isAudio {
			stream.PcktStreams[cuuid] = &pktStream
		} else {
			stream.AudioPcktStreams[cuuid] = &pktStream
		}
		pkt = pktStream.ps
	} else {
		pkt = nil
	}
	return
}

func (s *Streams) deleteClient(suuid string, cuuid string, isAudio bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	stream, ok := s.StreamMap[suuid]
	if ok {
		if isAudio {
			delete(stream.AudioPcktStreams, cuuid)
		} else {
			delete(stream.PcktStreams, cuuid)
		}

	}
}

func (s *Streams) put(suuid string, pckt Packet, isAudio bool) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	var retVal error = nil
	stream, ok := s.StreamMap[suuid]
	if ok {
		if !isAudio {
			err := stream.gopCache.Input(pckt)
			if err != nil {
				_ = fmt.Errorf(err.Error())
			}
			for _, packetStream := range stream.PcktStreams {
				length := len(packetStream.ps)
				log.Tracef("%s channel length = %d", suuid, length)
				select {
				case packetStream.ps <- pckt:
				default:
					{
						retVal = fmt.Errorf("client channel for %s has reached capacity (%d)", suuid, length)
					}
				}
			}
		} else {
			err := stream.gopCache.AudioInput(pckt)
			if err != nil {
				_ = fmt.Errorf(err.Error())
			}
			for _, packetStream := range stream.AudioPcktStreams {
				length := len(packetStream.ps)
				log.Tracef("%s audio channel length = %d", suuid, length)
				select {
				case packetStream.ps <- pckt:
				default:
					{
						retVal = fmt.Errorf("client channel for %s has reached capacity (%d)", suuid, length)
					}
				}
			}
		}

	} else {
		retVal = fmt.Errorf("no stream with name %s was found", suuid)
	}
	return retVal
}

func (s *Streams) getGOPCache(suuid string) (err error, gopCache *GopCacheSnapshot) {
	gopCache = nil
	stream, ok := s.StreamMap[suuid]
	if !ok {
		err = fmt.Errorf("no stream for %s in getGOPCache", suuid)
		return
	}
	gopCache = stream.gopCache.GetSnapshot()
	return
}

func (s *Streams) getCodec(suuid string) (err error, pckt Packet) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	codec, err := codecs.getCodecString(suuid)
	pckt.pckt = append([]byte{0x09}, []byte(codec)...)
	return
}

type PacketStream struct {
	ps chan Packet
	//	gopCacheCopy *GopCacheCopy
}

func NewPacketStream() (packetStream PacketStream) {
	packetStream = PacketStream{ps: make(chan Packet, 300)}
	return
}

type Packet struct {
	pckt []byte
}

func NewPacket(pckt []byte) Packet {
	b := make([]byte, len(pckt))
	copy(b, pckt)
	return Packet{pckt: b}
}

var hevcStart = []byte{0x00, 0x00, 0x01}
var h264Start = []byte{0x00, 0x00, 0x00, 0x01}
var h264KeyFrame1 = []byte{0x67, 0x64}
var h264KeyFrame2 = []byte{0x27, 0x64}
var h264KeyFrame3 = []byte{0x61, 0x88}

func (p Packet) isKeyFrame() (retVal bool) {
	retVal = false
	if bytes.Equal(p.pckt[:len(h264Start)], h264Start) {
		// H264 header
		retVal = bytes.Equal(p.pckt[4:6], h264KeyFrame1)
		if !retVal {
			retVal = bytes.Equal(p.pckt[4:6], h264KeyFrame2)
		}
		if !retVal {
			retVal = bytes.Equal(p.pckt[4:6], h264KeyFrame3)
		}
	} else if bytes.Equal(p.pckt[:len(hevcStart)], hevcStart) {
		// HEVC header
		theByte := p.pckt[3]
		retVal = theByte == 0x40
		theByte = (theByte >> 1) & 0x3f
		retVal = theByte == 0x19 || theByte == 0x20
	}
	return
}

func pseudoUUID() (uuid string) {
	const pseudoUUIDLen int = 16
	b := make([]byte, pseudoUUIDLen)
	_, err := rand.Read(b)
	if err != nil {
		log.Errorf("Error in pseudoUUID: %s", err.Error())
		return
	}
	uuid = fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return
}
