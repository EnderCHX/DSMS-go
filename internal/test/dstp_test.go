package test

import (
	"github.com/EnderCHX/DSMS-go/internal/dstp"
	"net"
	"testing"
	"time"
)

func TestDSTPs(t *testing.T) {
	listener, err := net.Listen("tcp", "0.0.0.0:20252")
	if err != nil {
		t.Error(err.Error())
	}
	tcpConn, err := listener.Accept()
	if err != nil {
		t.Error(err.Error())
	}
	dstpConn := dstp.NewConn(&tcpConn)
	data, _, err := dstpConn.Receive()
	if err != nil {
		t.Error(err.Error())
	}
	t.Log(data)
	t.Log(string(data))
	err = dstpConn.Send(data, true)
	if err != nil {
		t.Error(err.Error())
	}
	time.Sleep(1 * time.Second)
	dstpConn.Close()
}

func TestDSTPc(t *testing.T) {
	conn, err := net.Dial("tcp", "127.0.0.1:20252")
	if err != nil {
		t.Error(err.Error())
	}
	dstpConn := dstp.NewConn(&conn)
	err = dstpConn.Send([]byte("hello"), true)
	if err != nil {
		t.Error(err.Error())
	}
	for {
		data, type_, err := dstpConn.Receive()
		if err != nil {
			t.Error(err.Error())
		}
		if type_ != 1 {
			continue
		}
		t.Log(data)
		t.Log(string(data))
		break
	}
}

func TestDSTPPing(t *testing.T) {
	listener, err := net.Listen("tcp", "0.0.0.0:20256")
	if err != nil {
		t.Error(err.Error())
	}
	tcpConn, err := listener.Accept()
	if err != nil {
		t.Error(err.Error())
	}
	dstpConn := dstp.NewConn(&tcpConn)
	_, type_, err := dstpConn.Receive()
	if err != nil {
		t.Error(err.Error())
	}
	t.Log(type_)
	dstpConn.Close()
}

func TestDSTPDelay(t *testing.T) {
	conn, err := net.Dial("tcp", "127.0.0.1:20256")
	if err != nil {
		t.Error(err.Error())
	}
	dstpConn := dstp.NewConn(&conn)
	timeNow := time.Now()
	dstpConn.Ping()
	_, type_, err := dstpConn.Receive()
	if err != nil {
		t.Error(err.Error())
	}
	t.Log(time.Since(timeNow), type_)
}
