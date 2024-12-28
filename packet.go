package waiter

import (
	"cmp"
	"fmt"
	"sync"
)

var IPPacketPool *PacketPool = &PacketPool{MTU: 1428}

type Packet struct {
	buf    []byte
	offset int
}

func NewPacket(offset, cap int) *Packet {
	if offset > cap {
		panic("short packet cap")
	}
	return &Packet{offset: offset, buf: make([]byte, offset, cap)}
}

// AsBytes get ip packet slice
func (p *Packet) AsBytes() []byte {
	return p.buf[p.offset:]
}

// Ver get ip packet version.
// return 4 or 6
func (p *Packet) Ver() uint8 {
	pkt := p.AsBytes()
	if pkt == nil {
		return 0
	}
	return pkt[0] >> 4
}

// Bytes get ip packet slice with a header
func (p *Packet) Bytes(offset int) []byte {
	if p.offset < offset {
		panic("short packet offset")
	}
	return p.buf[offset:]
}

// Write ip packet bytes
func (p *Packet) Write(b []byte) error {
	p.buf = append(p.buf, b...)
	return nil
}

// SetHeader set ip packet header
func (p *Packet) SetHeader(header []byte) error {
	if len(header) > p.offset {
		return fmt.Errorf("short packet offset")
	}
	copy(p.buf[:p.offset], header)
	return nil
}

// Reset clear ip packet slice
func (p *Packet) Reset() {
	p.buf = p.buf[:p.offset]
}

type PacketPool struct {
	MTU int

	pool     *sync.Pool
	poolInit sync.Once
}

func (pool *PacketPool) init() {
	pool.poolInit.Do(func() {
		pool.pool = &sync.Pool{New: func() any {
			return NewPacket(IPPacketOffset, cmp.Or(pool.MTU, (2<<15)-8-40-IPPacketOffset)+IPPacketOffset)
		}}
	})
}

func (pool *PacketPool) Get() *Packet {
	pool.init()
	return pool.pool.Get().(*Packet)
}

func (pool *PacketPool) Put(p *Packet) {
	pool.init()
	p.Reset()
	pool.pool.Put(p)
}
