package message_hub

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/EnderCHX/DSMS-go/internal/dstp"
	auth "github.com/EnderCHX/DSMS-go/utils/jwt"
	"github.com/EnderCHX/DSMS-go/utils/log"
	"go.uber.org/zap"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

var globMsg = make(chan struct {
	data []byte
	c    *client
}, 16)

var clientCloseNotify = make(chan *client, 16)

var logger *zap.Logger
var loggerInited = false
var mtx = sync.Mutex{}

func InitLogger() {
	mtx.Lock()
	if loggerInited {
		return
	}
	logger = log.NewLogger("[MESSAGE_HUB]", "log/message_hub.log", "debug")
	loggerInited = true
	defer mtx.Unlock()
}

type Hub struct {
	subscribers map[string]map[*client]struct{} // 订阅者 key: topic, value: clients set
	mtx         sync.Mutex
	server      *server
}

func NewHub(addr, port string) *Hub {
	listener, err := net.Listen("tcp", addr+":"+port)
	InitLogger()
	if err != nil {
		logger.Error(err.Error())
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	h := &Hub{
		subscribers: make(map[string]map[*client]struct{}),
		server: &server{
			listen:    listener,
			clients:   make(map[*client]struct{}),
			broadcast: make(chan []byte),
			mtx:       sync.Mutex{},
			ctx:       ctx,
			close:     cancel,
		},
		mtx: sync.Mutex{},
	}
	return h
}

func (h *Hub) start() {
	InitLogger()
	go h.server.start()
	for {
		select {
		case msg := <-globMsg:
			logger.Debug(fmt.Sprintf("%v -> msg: %s", msg.c.conn.RemoteAddr(), string(msg.data)))
			var data Msg
			err := json.Unmarshal(msg.data, &data)
			if err != nil {
				msg.c.send <- func() []byte {
					msg_, _ := json.Marshal(&Msg{
						Option: "error",
						Data:   json.RawMessage(`{"msg":"message format error, need json"}`),
					})
					return msg_
				}()
				continue
			}
			go h.handleMsg(data, msg.c)
		case msg := <-h.server.broadcast:
			for client := range h.server.clients {
				client.send <- msg
			}
		}
	}
}

func (h *Hub) Run() {
	go h.start()
}

type client struct {
	username string
	login    bool
	conn     *dstp.Conn
	send     chan []byte
	pong     chan struct{}
	ctx      context.Context
	close    context.CancelFunc
	closed   bool
	mtx      sync.Mutex
}

func newClient(conn *net.Conn) *client {
	con := dstp.NewConn(conn)

	ctx, cancel := context.WithCancel(context.Background())
	return &client{
		conn:   con,
		send:   make(chan []byte),
		ctx:    ctx,
		close:  cancel,
		mtx:    sync.Mutex{},
		closed: false,
		pong:   make(chan struct{}),
	}
}

func (c *client) Read() {
	defer func() {
		if err := recover(); err != nil {
			logger.Error(fmt.Sprintf("%v -> read error: %v", c.conn.RemoteAddr(), err))
		}
		defer c.Close()
	}()
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			data, type_, err := c.conn.Receive()
			if err != nil {
				if err == io.EOF {
					logger.Debug(fmt.Sprintf("%v -> disconnected", c.conn.RemoteAddr()))
					c.Close()
					return
				}
				logger.Error(fmt.Sprintf("%v -> read error: %v", c.conn.RemoteAddr(), err))
				c.Close()
				return
			}

			if type_ != 1 {
				continue
			}

			globMsg <- struct {
				data []byte
				c    *client
			}{
				data: data,
				c:    c,
			}
		}
	}
}

func (c *client) Write() {
	defer func() {
		if err := recover(); err != nil {
			logger.Error(fmt.Sprintf("%v -> write error: %v", c.conn.RemoteAddr(), err))
		}
		defer c.Close()
	}()
	for {
		select {
		case msg := <-c.send:
			c.conn.Send(msg, true)
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *client) HeartBeat() {
	defer func() {
		if err := recover(); err != nil {
			logger.Error(fmt.Sprintf("%v -> heartbeat error: %v", c.conn.RemoteAddr(), err))
		}
		defer c.Close()
	}()
	logintimeout := 50 * time.Second
	heartbeatTime := 30 * time.Second
	timeout := 40 * time.Second
	loginTicker := time.NewTicker(logintimeout)
	heartTicker := time.NewTicker(heartbeatTime)
	timeoutTicker := time.NewTicker(timeout)
	for {
		select {
		case <-heartTicker.C:
			c.send <- func() []byte {
				msg_, _ := json.Marshal(&Msg{
					Option: "ping",
				})
				return msg_
			}()
			logger.Debug(fmt.Sprintf("ping -> %v", c.conn.RemoteAddr()))
		case <-c.pong:
			timeoutTicker.Reset(timeout)
			logger.Debug(fmt.Sprintf("%v -> pong", c.conn.RemoteAddr()))
		case <-timeoutTicker.C:
			clientCloseNotify <- c
			c.Close()
			logger.Debug(fmt.Sprintf("%v -> timeout, remove", c.conn.RemoteAddr()))
			return
		case <-loginTicker.C:
			if c.login {
				loginTicker.Stop()
			} else {
				c.send <- func() []byte {
					msg_, _ := json.Marshal(&Msg{
						Option: "error",
						Data:   json.RawMessage(`{"msg":"login timeout, please login"}`),
					})
					return msg_
				}()
				clientCloseNotify <- c
				c.Close()
				logger.Debug(fmt.Sprintf("%v -> login timeout, remove", c.conn.RemoteAddr()))
				return
			}
		}
	}
}

func (c *client) Close() {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if c.closed {
		return
	}
	c.close()
	c.closed = true
	c.conn.Close()
	clientCloseNotify <- c
}

type server struct {
	listen    net.Listener
	clients   map[*client]struct{}
	mtx       sync.Mutex
	broadcast chan []byte
	ctx       context.Context
	close     context.CancelFunc
}

func (s *server) start() {
	defer func() {
		if err := recover(); err != nil {
			logger.Error(fmt.Sprintf("%v -> server error: %v", s.listen.Addr(), err))
		}
		defer s.listen.Close()
	}()
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			default:
				c := <-clientCloseNotify
				s.mtx.Lock()
				delete(s.clients, c)
				s.mtx.Unlock()
				go func() {
					logger.Debug(fmt.Sprintf("%v clients connected", len(s.clients)))
					logger.Debug(fmt.Sprintf("%v", func() []net.Addr {
						var addrs []net.Addr
						for client := range s.clients {
							addrs = append(addrs, client.conn.RemoteAddr())
						}
						return addrs
					}()))
				}()
			}
		}
	}()
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			conn, err := s.listen.Accept()
			if err != nil {
				logger.Error(fmt.Sprintf("%v -> accept error: %v", s.listen.Addr(), err))
			}

			client := newClient(&conn)

			if _, ok := s.clients[client]; ok {
				client.Close()
				clientCloseNotify <- client
				continue
			}
			s.mtx.Lock()
			s.clients[client] = struct{}{}
			s.mtx.Unlock()

			logger.Debug(fmt.Sprintf("%v -> connected", conn.RemoteAddr()))
			logger.Debug(fmt.Sprintf("%v clients connected", len(s.clients)))
			logger.Debug(fmt.Sprintf("%v", func() []net.Addr {
				var addrs []net.Addr
				for client := range s.clients {
					addrs = append(addrs, client.conn.RemoteAddr())
				}
				return addrs
			}()))

			go client.Read()
			go client.Write()
			go client.HeartBeat()
		}
	}
}

type Msg struct {
	Option string          `json:"option"`
	Data   json.RawMessage `json:"data"`
}

func (h *Hub) handleMsg(msg Msg, c *client) {
	switch msg.Option {
	case "subscribe":
		if !c.login {
			return
		}
		type Data struct {
			Topic string `json:"topic"`
		}
		var data Data
		json.Unmarshal(msg.Data, &data)
		if data.Topic == "" {
			return
		}
		h.mtx.Lock()
		defer h.mtx.Unlock()
		if _, ok := h.subscribers[data.Topic]; !ok {
			h.subscribers[data.Topic] = make(map[*client]struct{})
		}
		h.subscribers[data.Topic][c] = struct{}{}
	case "unsubscribe":
		if !c.login {
			return
		}
		type Data struct {
			Topic string `json:"topic"`
		}
		var data Data
		json.Unmarshal(msg.Data, &data)

		if data.Topic == "" {
			return
		}
		h.mtx.Lock()
		defer h.mtx.Unlock()
		if _, ok := h.subscribers[data.Topic]; !ok {
			return
		}
		delete(h.subscribers[data.Topic], c)
	case "publish":
		if !c.login {
			return
		}
		type Data struct {
			Topic    string          `json:"topic"`
			Data     json.RawMessage `json:"data"`
			FromUser string          `json:"from_user"`
		}
		var data Data
		json.Unmarshal(msg.Data, &data)
		if data.Topic == "" {
			return
		}
		data.FromUser = c.username
		data_, _ := json.Marshal(data)
		msg.Data = data_
		msg_, _ := json.Marshal(msg)
		for client := range h.subscribers[data.Topic] {
			if client.closed {
				logger.Debug(fmt.Sprintf("%v -> client closed", client.conn.RemoteAddr()))
				client.mtx.Lock()
				delete(h.subscribers[data.Topic], client)
				client.mtx.Unlock()
				continue
			}
			if !client.login {
				continue
			}
			client.send <- msg_
		}
	case "pong":
		c.pong <- struct{}{}
	case "login":
		type Data struct {
			AccessToken string `json:"access_token"`
		}
		var data Data
		json.Unmarshal(msg.Data, &data)
		if data.AccessToken == "" {
			c.send <- func() []byte {
				msg_, _ := json.Marshal(&Msg{
					Option: "error",
					Data:   json.RawMessage(`{"error":"access token is empty"}`),
				})
				logger.Debug(fmt.Sprintf("%v -> : %v", c.conn.RemoteAddr(), "登录失败"))
				return msg_
			}()
			return
		} else {
			payload, err := auth.VerifyToken(data.AccessToken, os.Getenv("ACCESS_SECRET"))
			if err != nil {
				c.send <- func() []byte {
					msg_, _ := json.Marshal(&Msg{
						Option: "error",
						Data:   json.RawMessage(`{"error":"access token is invalid"}`),
					})
					return msg_
				}()
				return
			} else {
				c.mtx.Lock()
				defer c.mtx.Unlock()
				if c.login {
					c.send <- func() []byte {
						msg_, _ := json.Marshal(&Msg{
							Option: "error",
							Data:   json.RawMessage(`{"error":"already login"}`),
						})
						return msg_
					}()
					return
				} else {
					c.login = true
					c.username = payload.Username
					c.send <- func() []byte {
						msg_, _ := json.Marshal(&Msg{
							Option: "info",
							Data:   json.RawMessage(`{"info":"login success"}`),
						})
						logger.Debug(fmt.Sprintf("%v -> : %v", c.conn.RemoteAddr(), "登录成功"))
						return msg_
					}()
					return
				}
			}
		}
	default:
		logger.Error(fmt.Sprintf("%v -> unknown option: %v", c.conn.RemoteAddr(), msg.Option))
		c.send <- func() []byte {
			msg_, _ := json.Marshal(&Msg{
				Option: "error",
				Data:   json.RawMessage(`{"error":"unknown option"}`),
			})
			return msg_
		}()
	}
}
