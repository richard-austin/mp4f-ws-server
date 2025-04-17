package main

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	log "github.com/sirupsen/logrus"
	"sync"
)

type Stream struct {
	ftyp             Packet
	moov             Packet
	hasAudio         bool
	hasVideo         bool
	gopCache         GopCache
	bucketBrigade    BucketBrigade
	PcktStreams      map[string]*PacketStream // One packetStream for each client connected through the suuid
	AudioPcktStreams map[string]*PacketStream // Separate set of streams for audio
	IsRecording      bool
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

func (s *Streams) addStream(suuid string, isAudio bool, isRecording ...bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	gopCacheEnabled := false
	if len(isRecording) == 0 || isRecording[0] == false {
		gopCacheEnabled = config.GopCache
	} else {
		isAudio = false
	}
	stream := &Stream{PcktStreams: map[string]*PacketStream{}, AudioPcktStreams: map[string]*PacketStream{}, gopCache: NewGopCache(gopCacheEnabled), bucketBrigade: NewBucketBrigade( /*streamC.PreambleFrames*/ 40)}
	stream.hasAudio = isAudio
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

func (s *Streams) put(suuid string, pckt Packet, isAudio bool, isRecording ...bool) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	var retVal error = nil
	stream, ok := s.StreamMap[suuid]
	if ok {
		if len(isRecording) > 0 && isRecording[0] {
			err := stream.gopCache.RecordingInput(pckt)
			if err != nil {
				_ = fmt.Errorf(err.Error())
			}
			err = stream.bucketBrigade.Input(pckt)
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

		} else if !isAudio {
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

func (s *Streams) putFtyp(suuid string, pckt Packet) (retVal error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	retVal = nil
	// Check it is actually a ftyp
	val := getSubBox(pckt, "ftyp")
	if val == nil {
		retVal = fmt.Errorf("The packet recieved in putFtyp was not a ftyp")
		return
	} else {
		stream, ok := s.StreamMap[suuid]
		if ok {
			stream.ftyp = pckt
			s.StreamMap[suuid] = stream
		} else {
			retVal = fmt.Errorf("Stream %s not found", suuid)
		}
	}
	return
}

func (s *Streams) putMoov(suuid string, pckt Packet) (retVal error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	retVal = nil
	// Check it is actually a moov
	val := getSubBox(pckt, "moov")
	if val == nil {
		retVal = fmt.Errorf("The packet recieved in putMoov was not a moov")
		return
	} else {
		stream, ok := s.StreamMap[suuid]
		if ok {
			stream.moov = pckt
			s.StreamMap[suuid] = stream
		} else {
			retVal = fmt.Errorf("Stream %s not found", suuid)
		}
	}
	return
}

func (p Packet) isFmp4KeyFrame() (retVal bool) {
	// [moof [mfhd] [traf [tfhd] [tfdt] [moof]]]
	retVal = false
	moof := getSubBox(p, "moof")
	if moof == nil {
		log.Warnf("moof was nil in isKeyFrame")
		return
	}
	flags := moof[3:5]

	retVal = flags[0] == 0x68 || flags[0] == 0xb4
	//	log.Infof("flags = 0x%x%c, %t", flags[0], flags[1], retVal)
	return
}

func (p Packet) isMoof() (retVal bool) {
	retVal = false
	if len(p.pckt) > 20 {
		moof := getSubBox(p, "moof")
		retVal = moof != nil
	}
	return
}

func getSubBox(pckt Packet, boxName string) (sub_box []byte) {
	searchData := pckt.pckt
	searchTerm := []byte(boxName)
	idx := bytes.Index(searchData, searchTerm)

	if idx >= 4 {
		length := int(binary.BigEndian.Uint32(searchData[idx-4 : idx]))
		sub_box = searchData[idx-4 : length+idx-4]

	} else {
		sub_box = nil
	}
	return
}

func (s *Streams) getFtyp(suuid string) (err error, ftyp Packet) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	err = nil
	stream, ok := s.StreamMap[suuid]

	log.Infof("ok = %t", ok)
	if !ok {
		err = fmt.Errorf("stream %s not found", suuid)
	} else if stream.ftyp.pckt == nil {
		err = fmt.Errorf("no ftyp for stream %s", suuid)
	} else {
		ftyp = stream.ftyp
	}
	return
}

func (s *Streams) getMoov(suuid string) (error, Packet) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	var retVal error = nil
	stream, ok := s.StreamMap[suuid]
	if !ok {
		retVal = fmt.Errorf("stream %s not found", suuid)
	} else if stream.moov.pckt == nil {
		retVal = fmt.Errorf("no moov for stream %s", suuid)
	}

	return retVal, stream.moov
}
