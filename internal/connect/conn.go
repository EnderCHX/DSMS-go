package connect

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync/atomic"
)

const (
	dataStart    byte = 0x01
	dataContinue byte = 0x02
	dataEnd      byte = 0x03
)

type Conn struct {
	conn   net.Conn
	closed atomic.Bool
}

type connWriter struct {
	conn    *Conn
	started bool
	sent    atomic.Int64
}

func (c *connWriter) Write(p []byte) (n int, err error) {
	//没有发送开始符号则发送
	if !c.started {
		_, err = c.conn.conn.Write([]byte{dataStart})
		if err != nil {
			return 0, err
		}
		c.started = true
		c.sent.Add(1)
	}

	//发送数据（续）
	_, err = c.conn.conn.Write([]byte{dataContinue})
	if err != nil {
		return 0, err
	}
	c.sent.Add(1)

	dataLenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(dataLenBuf, uint32(len(p)))
	_, err = c.conn.conn.Write(dataLenBuf)
	if err != nil {
		return 0, err
	}
	c.sent.Add(4)
	_, err = c.conn.conn.Write(p)
	if err != nil {
		return 0, err
	}
	c.sent.Add(int64(len(p)))
	n = int(c.sent.Load())
	return
}

func (c *connWriter) Close() error {
	_, err := c.conn.conn.Write([]byte{dataEnd})
	return err
}

func (c *Conn) Send() (io.WriteCloser, error) {
	if c.closed.Load() {
		return nil, io.ErrClosedPipe
	}
	return &connWriter{
		conn:    c,
		started: false,
		sent:    atomic.Int64{},
	}, nil
}

type connReader struct {
	conn   *Conn
	eof    atomic.Bool
	buffer []byte
}

func (c *connReader) Read(p []byte) (n int, err error) {
	if c.eof.Load() {
		return 0, io.EOF
	}
	buf := make([]byte, 1)
	_, err = io.ReadFull(c.conn.conn, buf)
	if err != nil {
		return 0, err
	}
	if buf[0] != dataStart && buf[0] != dataContinue && buf[0] != dataEnd {
		return 0, fmt.Errorf("invalid data start")
	}

	if buf[0] == dataEnd {
		c.eof.Store(true)
		return 0, io.EOF
	}

	lenBuf := make([]byte, 4)
	_, err = io.ReadFull(c.conn.conn, lenBuf)
	if err != nil {
		return 0, err
	}
	dataLen := binary.BigEndian.Uint32(lenBuf)

	if dataLen > 0 {
		c.buffer = make([]byte, dataLen)
		_, err = io.ReadFull(c.conn.conn, c.buffer)
		if err != nil {
			return 0, err
		}
	}
	return copy(p, c.buffer), nil
}

func (c *Conn) Receive() (io.Reader, error) {
	if c.closed.Load() {
		return nil, io.ErrClosedPipe
	}
	return &connReader{
		conn:   c,
		eof:    atomic.Bool{},
		buffer: []byte{},
	}, nil
}

func (c *Conn) Close() {
	if c.closed.CompareAndSwap(false, true) {
		c.conn.Close()
	}
}

func NewConn(conn net.Conn) *Conn {
	return &Conn{
		conn:   conn,
		closed: atomic.Bool{},
	}
}
