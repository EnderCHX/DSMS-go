package dstp

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

/*
分布式仿真传输协议
Distributed Simulation Transport Protocol (DSTP)
数据包格式：
|开始标记|控制标记|消息id|数据长度|数据|继续标识|数据长度|数据|结束标记|
控制标记:1字节，|0|0|0|是否为ping包|ping包0为pong1为ping|是否需要应答ack|是否为ack包0否1是|是否分段|
ping包: |0x01|00010000|0x03| pong包: |0x01|00011000|0x03|
ack包: |0x01|00000010|应答消息id|0x03|
需要应答的分段的数据包: |0x01|00000101|消息id|数据长度|数据|继续标识|数据长度|数据|0x03|
*/

const (
	dataStart    byte = 0x01
	dataContinue byte = 0x02
	dataEnd      byte = 0x03

	ctrlSeg     byte = 0b00000001
	ctrlIfPing  byte = 0b00010000
	ctrlPing    byte = 0b00001000
	ctrlNeedAck byte = 0b00000100
	ctrlIfAck   byte = 0b00000010
)

type Conn struct {
	conn   *net.Conn
	closed atomic.Bool
	ackMap sync.Map
	mtx    sync.Mutex
}

func (c *Conn) sendData(data []byte, needAck bool, waitAck bool, ackMessageId uint32) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if c.closed.Load() {
		return io.ErrClosedPipe
	}

	needSegment := false

	segment := len(data)/((1<<17)-1) + 1

	if segment > 1 {
		needSegment = true
	}

	_, err := (*c.conn).Write([]byte{dataStart})
	if err != nil {
		return err
	}

	var controlData byte
	controlData = 0b00000000

	if needAck {
		controlData |= 0b00000100
	}
	if needSegment {
		controlData |= 0b00000001
	}
	_, err = (*c.conn).Write([]byte{controlData})
	if err != nil {
		return err
	}

	var messageId uint32

	if waitAck {
		messageId = ackMessageId
	} else {
		messageId = uint32(time.Now().UnixNano() + rand.Int63())
	}
	messageIdBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(messageIdBuf, messageId)
	_, err = (*c.conn).Write(messageIdBuf)
	if err != nil {
		return err
	}

	if segment <= 1 {
		dataLen := len(data)
		dataLenBuf := make([]byte, 2)
		binary.BigEndian.PutUint16(dataLenBuf, uint16(dataLen))
		_, err = (*c.conn).Write(dataLenBuf)
		if err != nil {
			return err
		}

		_, err = (*c.conn).Write(data)
		if err != nil {
			return err
		}

		_, err = (*c.conn).Write([]byte{dataEnd})
		return err
	}

	for i := 0; i < segment; i++ {
		dataLen := ((1 << 17) - 1)
		dataLenBuf := make([]byte, 2)
		binary.BigEndian.PutUint16(dataLenBuf, uint16(dataLen))
		_, err = (*c.conn).Write(dataLenBuf)
		if err != nil {
			return err
		}
		_, err = (*c.conn).Write(data[i*((1<<17)-1) : (i+1)*((1<<17)-1)])
		if err != nil {
			return err
		}

		if i == segment-1 {
			if needAck && !waitAck {
				go func() {
					for i := 0; i < 3; i++ {
						time.Sleep(10 * time.Second)
						if _, ok := c.ackMap.Load(messageId); ok {
							c.ackMap.Delete(messageId)
							break
						} else {
							c.sendData(data, true, true, messageId)
						}
					}
				}()
			}
			_, err = (*c.conn).Write([]byte{dataEnd})
			return err
		} else {
			_, err = (*c.conn).Write([]byte{dataContinue})
			if err != nil {
				return err
			}
		}
	}
	return fmt.Errorf("invalid data start")
}

func (c *Conn) sendAck(messageId uint32) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if c.closed.Load() {
		return io.ErrClosedPipe
	}

	_, err := (*c.conn).Write([]byte{dataStart})
	if err != nil {
		return err
	}
	_, err = (*c.conn).Write([]byte{ctrlIfAck})
	if err != nil {
		return err
	}
	messageIdBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(messageIdBuf, messageId)
	_, err = (*c.conn).Write(messageIdBuf)
	if err != nil {
		return err
	}
	_, err = (*c.conn).Write([]byte{dataEnd})
	return err
}

func (c *Conn) sendPing() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if c.closed.Load() {
		return io.ErrClosedPipe
	}

	_, err := (*c.conn).Write([]byte{dataStart})
	if err != nil {
		return err
	}
	ctrlData := ctrlIfPing | ctrlPing
	_, err = (*c.conn).Write([]byte{ctrlData})
	if err != nil {
		return err
	}
	_, err = (*c.conn).Write([]byte{dataEnd})
	return err
}

func (c *Conn) sendPong() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if c.closed.Load() {
		return io.ErrClosedPipe
	}

	_, err := (*c.conn).Write([]byte{dataStart})
	if err != nil {
		return err
	}
	ctrlData := ctrlIfPing
	_, err = (*c.conn).Write([]byte{ctrlData})
	if err != nil {
		return err
	}
	_, err = (*c.conn).Write([]byte{dataEnd})
	return err
}

// 接收数据包 return 数据
// 数据类型int
// 1 普通消息
// 2 ping包
// 3 pong包
// 4 ack应答
func (c *Conn) receiveData() ([]byte, int, error) {
	var data []byte

	startBuf := make([]byte, 1)
	_, err := io.ReadFull((*c.conn), startBuf)
	if err != nil {
		return nil, 0, err
	}

	if startBuf[0] != dataStart {
		return nil, 0, fmt.Errorf("invalid data start")
	}

	ctrlBuf := make([]byte, 1)
	_, err = io.ReadFull((*c.conn), ctrlBuf)
	if err != nil {
		return nil, 0, err
	}

	ctrlData := ctrlBuf[0]

	if ctrlData&ctrlIfPing == 1 {
		if ctrlData&ctrlPing == 1 {
			err := c.sendPong()
			if err != nil {
				return nil, 2, err
			}
			endBuf := make([]byte, 1)
			_, err = io.ReadFull((*c.conn), endBuf)
			return nil, 2, err
		} else {
			endBuf := make([]byte, 1)
			_, err = io.ReadFull((*c.conn), endBuf)
			return nil, 3, err
		}
	}

	messageIdBuf := make([]byte, 4)
	if ctrlData&ctrlIfAck == 1 {
		_, err = io.ReadFull((*c.conn), messageIdBuf)
		if err != nil {
			return nil, 4, err
		}
		messageId := binary.BigEndian.Uint32(messageIdBuf)
		endBuf := make([]byte, 1)
		_, err = io.ReadFull((*c.conn), endBuf)
		c.ackMap.Store(messageId, struct{}{})
		return messageIdBuf, 4, err
	}

	needAck := ctrlData&ctrlNeedAck == 1
	_, err = io.ReadFull((*c.conn), messageIdBuf)
	if err != nil {
		return nil, 0, err
	}
	messageId := binary.BigEndian.Uint32(messageIdBuf)
	needSegment := ctrlData&ctrlSeg == 1

	for {
		if needSegment {
			dataLenBuf := make([]byte, 2)
			_, err := io.ReadFull((*c.conn), dataLenBuf)
			if err != nil {
				return nil, 1, err
			}
			dataLen := binary.BigEndian.Uint16(dataLenBuf)
			dataT := make([]byte, dataLen)
			_, err = io.ReadFull((*c.conn), dataT)
			if err != nil {
				return nil, 1, err
			}
			data = append(data, dataT...)
		} else {
			dataLenBuf := make([]byte, 2)
			_, err := io.ReadFull((*c.conn), dataLenBuf)
			if err != nil {
				return nil, 1, err
			}
			dataLen := binary.BigEndian.Uint16(dataLenBuf)
			data = make([]byte, dataLen)
			_, err = io.ReadFull((*c.conn), data)
			if err != nil {
				return nil, 1, err
			}
		}
		buf := make([]byte, 1)
		_, err := io.ReadFull((*c.conn), buf)
		if err != nil {
			return nil, 1, err
		}
		if buf[0] == dataEnd {
			if needAck {
				err := c.sendAck(messageId)
				if err != nil {
					return nil, 1, err
				}
			}
			return data, 1, nil
		} else if buf[0] == dataContinue {
			continue
		} else if buf[0] == dataStart {
			return nil, 1, fmt.Errorf("invalid data start")
		} else {
			return nil, 1, fmt.Errorf("invalid data end")
		}
	}
}

func (c *Conn) Send(data []byte, needAck bool) error {
	return c.sendData(data, needAck, false, 0)
}

func (c *Conn) Ping() error {
	return c.sendPing()
}

func (c *Conn) Pong() error {
	return c.sendPong()
}

func (c *Conn) Receive() (data []byte, type_ int, err error) {
	data, type_, err = c.receiveData()
	return
}

func (c *Conn) Close() {
	if c.closed.CompareAndSwap(false, true) {
		(*c.conn).Write([]byte{dataEnd})
		(*c.conn).Close()
	}
}

func (c *Conn) IsClosed() bool {
	return c.closed.Load()
}

func (c *Conn) RemoteAddr() net.Addr {
	return (*c.conn).RemoteAddr()
}

func NewConn(conn *net.Conn) *Conn {
	return &Conn{
		conn:   conn,
		closed: atomic.Bool{},
		ackMap: sync.Map{},
		mtx:    sync.Mutex{},
	}
}
