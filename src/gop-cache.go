package main

import (
	"fmt"
	"sync"
)

type GopCache struct {
	cacheLength int
	mutex       sync.Mutex
	inputIndex  int
	Cache       []Packet
}

type GopCacheCopy struct {
	pktChan chan Packet
}

func NewGopCache() (cache GopCache) {
	const cacheLength int = 90
	cache = GopCache{Cache: make([]Packet, cacheLength), inputIndex: 0, cacheLength: cacheLength - 1}
	return
}

func (g *GopCache) Input(p Packet) (err error) {
	g.mutex.Lock()
	err = nil
	const maxKeyFramePackets = 6
	defer g.mutex.Unlock()
	if (g.inputIndex > maxKeyFramePackets || g.inputIndex == 0) && p.isKeyFrame() {
		g.inputIndex = 0
	}
	if g.inputIndex < g.cacheLength {
		g.Cache[g.inputIndex] = p
		g.inputIndex++
	} else {
		err = fmt.Errorf("GOP Cache is full")
	}
	return
}

func (g *GopCache) GetCurrent() (gopCacheCopy *GopCacheCopy) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	gopCacheCopy = NewGopCacheCopy(g)
	return
}
func NewGopCacheCopy(cache *GopCache) (copy *GopCacheCopy) {
	copy = &GopCacheCopy{pktChan: make(chan Packet, cache.cacheLength)}
	for i := 0; i < cache.inputIndex; i++ {
		copy.pktChan <- cache.Cache[i]
	}
	return
}

func (c GopCacheCopy) Get() (ok bool, packet Packet) {
	ok = true
	select {
	case packet = <-c.pktChan:

	default:
		ok = false
	}
	return
}
