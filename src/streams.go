package main

type Streams struct {
	Streams map[string]*PacketStream
}

func NewStreams() *Streams {
	s := Streams{Streams: make(map[string]*PacketStream)}

	return &s
}

type PacketStream struct {
	ps chan []byte
}

func NewPacketStream() *PacketStream {
	ps := PacketStream{ps: make(chan []byte)}
	return &ps
}

func (self *Streams) addPcktStream(uri string) {
	self.Streams[uri] = NewPacketStream()
}
