package app

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"fyne.io/fyne/v2"
	"github.com/EnderCHX/DSMS-go/internal/connect"
	"github.com/EnderCHX/DSMS-go/utils/log"
	"io"
	"net"
	"sync/atomic"
	"time"
)

var logger = log.NewLogger("[Client]", "logs/client.log", "debug")

//go:embed resources
var fileFS embed.FS

var (
	client    *connect.Conn
	writer    io.WriteCloser
	reader    io.Reader
	send      = make(chan []byte)
	recv      = make(chan []byte)
	ctx       context.Context
	cancel    context.CancelFunc
	timestamp atomic.Uint64
)

func init() {
	timestamp.Store(0)
}

func connectServer(ip, port string) error {
	conn, err := net.Dial("tcp", ip+":"+port)
	if err != nil {
		return err
	}

	client = connect.NewConn(&conn)

	writer, err = client.Send()
	if err != nil {
		return err
	}

	reader, err = client.Receive()
	if err != nil {
		return err
	}
	go read()
	go write()
	go handleMsg()
	return err
}

func read() {
	for {
		buf := make([]byte, 1024)
		n, err := reader.Read(buf)
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Println("Server closed")
				return
			}
			logger.Error("Read error: " + err.Error())
			return
		}
		logger.Debug(string(buf[:n]))
		recv <- buf[:n]
	}
}

func write() {
	for {
		msg := <-send
		logger.Debug(string(msg))
		_, err := writer.Write(msg)
		if err != nil {
			logger.Error("Write error: " + err.Error())
			return
		}
	}
}

func handleMsg() {
	for {
		msg := <-recv
		fyne.Do(func() {
			messageList.Append(append([]byte(time.Now().Format("2006-01-02 15:04:05 ")), msg...))
			list.ScrollToBottom()
		})
		logger.Debug(string(msg))
		var data Msg
		err := json.Unmarshal(msg, &data)
		if err != nil {
			continue
		}
		switch data.Option {
		case "ping":
			send <- func() []byte {
				msg_, _ := json.Marshal(Msg{
					Option: "pong",
				})
				return msg_
			}()
		case "publish":
			logger.Info(fmt.Sprintf("%s", data.Data))
		}
	}
}

type Msg struct {
	Option string          `json:"option"`
	Data   json.RawMessage `json:"data"`
}
