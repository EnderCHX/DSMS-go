package app

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"fyne.io/fyne/v2"
	"github.com/EnderCHX/DSMS-go/internal/dstp"
	"github.com/EnderCHX/DSMS-go/utils/log"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var logger = log.NewLogger("[Client]", "log/client.log", "debug")

//go:embed resources
var fileFS embed.FS

var (
	client    *dstp.Conn
	send      = make(chan []byte)
	recv      = make(chan []byte)
	ctx       context.Context
	cancel    context.CancelFunc
	timestamp atomic.Uint64
	mtx       sync.Mutex
)

func init() {
	timestamp.Store(0)
	ctx, cancel = context.WithCancel(context.Background())
}

func connectServer(ip, port string) error {
	conn, err := net.Dial("tcp", ip+":"+port)
	if err != nil {
		return err
	}

	client = dstp.NewConn(&conn)

	go read()
	go write()
	go handleMsg()
	return err
}

func read() {
	for {
		data, type_, err := client.Receive()

		if err != nil {
			if err == io.EOF {
				logger.Error("Server closed")
				return
			} else {
				logger.Error("Read error: " + err.Error())
				return
			}
		}
		if type_ != 1 {
			continue
		}

		logger.Debug(string(data))
		recv <- data
	}
}

func write() {
	for {
		msg := <-send
		logger.Debug(string(msg))
		err := client.Send(msg, false)
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
