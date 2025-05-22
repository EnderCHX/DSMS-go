package test

import (
	"github.com/EnderCHX/DSMS-go/internal/dstp"
	"net"
	"testing"
	"time"
)

func TestDSTPs(t *testing.T) {
	listener, err := net.Listen("tcp", ":8888")
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
	t.Log(string(data))
	dstpConn.Send(data, true)
	dstpConn.Close()
}

func TestDSTPc(t *testing.T) {
	conn, err := net.Dial("tcp", "127.0.0.1:8888")
	if err != nil {
		t.Error(err.Error())
	}
	dstpConn := dstp.NewConn(&conn)
	dstpConn.Send([]byte("hello"), true)
	data, _, err := dstpConn.Receive()
	if err != nil {
		t.Error(err.Error())
	}
	t.Log(string(data))
}

func TestDSTPPing(t *testing.T) {
	listener, err := net.Listen("tcp", ":8888")
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
	conn, err := net.Dial("tcp", "192.168.86.11:8888")
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
