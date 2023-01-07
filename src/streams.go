package main

type Streams struct {
	Streams map[string]*PacketStream
}

func NewStreams() *Streams {
	s := Streams{Streams: make(map[string]*PacketStream)}

	return &s
}

type PacketStream struct {
	ps chan Packet
}

func NewPacketStream() *PacketStream {
	ps := PacketStream{ps: make(chan Packet)}
	return &ps
}

func (self *Streams) addPcktStream(uri string) {
	self.Streams[uri] = NewPacketStream()
}

type Packet struct {
	pckt []byte
}

func NewPacket(pckt []byte) Packet {
	b := make([]byte, len(pckt))
	copy(b, pckt)
	return Packet{pckt: b}
}
