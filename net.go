package pooh

import (
	"encoding/binary"
	"io"
	"net"
	"time"
)

const (
	PacketTimeout  = 30 * time.Second
	HttpBufferSize = 8 * 1024
)

func NewConn(c net.Conn) Conn {
	if conn, ok := c.(Conn); ok {
		return conn
	}
	return &connImpl{
		Conn: c,
		buf:  make([]byte, 0, HttpBufferSize),
	}
}

type Conn interface {
	net.Conn
	Byte() (byte, error)
	Bytes(n int) ([]byte, error)
	Int(size int) (i int, err error)
	Uint16() (u uint16, err error)
	Uint32() (u uint32, err error)
	Uint64() (u uint64, err error)
	Frame(size int) ([]byte, error)
	WriteFrame(p []byte, size int) error
	Preload() (p []byte, err error)
	Preplace(replace []byte)
	Reset()
}

type connImpl struct {
	net.Conn
	buf   []byte
	index int
}

func (c *connImpl) Read(b []byte) (n int, err error) {
	switch {
	case c.buf == nil:
		return c.Conn.Read(b)
	case c.index < len(c.buf):
		n = copy(b, c.buf[c.index:])
		c.index += n
	default:
		n, err = c.Conn.Read(b)
		if err != nil {
			return
		}
		c.index += n
		if c.index > cap(c.buf) {
			c.buf = nil
			c.index = 0
		} else {
			c.buf = append(c.buf, b[:n]...)
		}
	}
	return
}

func (c *connImpl) Close() error {
	return Close(c.Conn)
}

func (c *connImpl) Byte() (byte, error) {
	b, err := c.Bytes(1)
	return b[0], err
}

func (c *connImpl) Bytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := io.ReadFull(c, b)
	return b, err
}

func (c *connImpl) Int(size int) (i int, err error) {
	bs, err := c.Bytes(size)
	if err != nil {
		return
	}
	i = Int(bs)
	return
}

func (c *connImpl) Uint16() (u uint16, err error) {
	bs, err := c.Bytes(2)
	if err != nil {
		return
	}
	u = binary.BigEndian.Uint16(bs)
	return
}

func (c *connImpl) Uint32() (u uint32, err error) {
	bs, err := c.Bytes(4)
	if err != nil {
		return
	}
	u = binary.BigEndian.Uint32(bs)
	return
}

func (c *connImpl) Uint64() (u uint64, err error) {
	bs, err := c.Bytes(8)
	if err != nil {
		return
	}
	u = binary.BigEndian.Uint64(bs)
	return
}

func (c *connImpl) Frame(size int) ([]byte, error) {
	bs, err := c.Bytes(size)
	if err != nil {
		return nil, err
	}
	return c.Bytes(Int(bs))
}

func (c *connImpl) WriteFrame(p []byte, size int) error {
	_, err := c.Conn.Write(append(Int2Bytes(len(p), size), p...))
	return err
}

func (c *connImpl) Preload() (p []byte, err error) {
	buf := make([]byte, HttpBufferSize)
	n, err := c.Read(buf)
	if err != nil {
		return
	}
	p = buf[:n]
	c.Reset()
	return
}

func (c *connImpl) Preplace(replace []byte) {
	need := len(replace) + len(c.buf) - c.index
	if need > cap(c.buf) {
		buf := make([]byte, 0, need)
		buf = append(buf, replace...)
		buf = append(buf, c.buf[c.index:]...)
		c.buf = buf
	} else {
		copy(c.buf[len(replace):need], c.buf[c.index:])
		copy(c.buf, replace)
		c.buf = c.buf[:need]
	}
	c.Reset()
}

func (c *connImpl) Reset() {
	c.index = 0
}