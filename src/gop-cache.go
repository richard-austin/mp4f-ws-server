package main

import (
	"fmt"
	"sync"
)

type GopCache struct {
	GopCacheUsed     bool
	cacheLength      int
	mutex            sync.Mutex
	inputIndex       int
	Cache            []Packet
	audioCacheLength int
	audioMutex       sync.Mutex
	audioInputIndex  int
	AudioCache       []Packet
}

type GopCacheSnapshot struct {
	pktChan chan Packet
}

func NewGopCache(used bool) (cache GopCache) {
	const cacheLength int = 90
	const audioCacheLength int = 600
	cache = GopCache{Cache: make([]Packet, cacheLength), cacheLength: cacheLength - 1, inputIndex: 0, AudioCache: make([]Packet, audioCacheLength), audioCacheLength: audioCacheLength, audioInputIndex: 0, GopCacheUsed: used}
	return
}

// Input
// Input packets to the GOP cache starting with the latest keyframe at index 0
func (g *GopCache) Input(p Packet) (err error) {
	err = nil
	if !g.GopCacheUsed {
		return
	}
	g.mutex.Lock()
	defer g.mutex.Unlock()
	if p.isKeyFrame() {
		g.inputIndex = 0
		g.audioInputIndex = 0
	}
	if g.inputIndex < g.cacheLength {
		g.Cache[g.inputIndex] = p
		g.inputIndex++
	} else {
		err = fmt.Errorf("GOP Cache is full")
	}
	return
}

func (g *GopCache) AudioInput(p Packet) (err error) {
	err = nil
	if !g.GopCacheUsed {
		return
	}
	g.audioMutex.Lock()
	defer g.audioMutex.Unlock()
	if g.audioInputIndex < g.audioCacheLength {
		g.AudioCache[g.audioInputIndex] = p
	} else {
		err = fmt.Errorf("audio GOP cache is full")
	}
	return
}

// GetSnapshot
// Create a new GOP cache snapshot from the current GOP cache, unless GOP cache is not enabled
// **
func (g *GopCache) GetSnapshot() (snapshot *GopCacheSnapshot) {
	if !g.GopCacheUsed {
		return
	}
	snapshot = newFeeder(g, false)
	return
}

func (g *GopCache) GetAudioSnapshot() (snapshot *GopCacheSnapshot) {
	if !g.GopCacheUsed {
		return
	}
	snapshot = newFeeder(g, true)
	return
}

// newFeeder
// Create a new GOP cache snapshot from the current GOP cache
// **
func newFeeder(g *GopCache, isAudio bool) (feeder *GopCacheSnapshot) {
	feeder = &GopCacheSnapshot{pktChan: make(chan Packet, g.cacheLength)}
	g.mutex.Lock()
	defer g.mutex.Unlock()
	if !isAudio {
		for _, pkt := range g.Cache[:g.inputIndex] {
			feeder.pktChan <- pkt
		}
	} else {
		for _, pkt := range g.AudioCache[:g.audioInputIndex] {
			feeder.pktChan <- pkt
		}
	}
	return
}

// Get
// Get: Get the live feed, prioritising the GOP cache snapshot content before sending live feed to the client
// **
func (s GopCacheSnapshot) Get(live chan Packet) (packet Packet) {
	select {
	case packet = <-s.pktChan:
	default:
		packet = <-live
	}
	return
}
