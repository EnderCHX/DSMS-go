package test

import (
	"github.com/EnderCHX/DSMS-go/internal/auth"
	"github.com/EnderCHX/DSMS-go/internal/dstp"
	"github.com/bytedance/sonic"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var msgTest, _ = sonic.Marshal(map[string]any{
	"option": "publish",
	"data": map[string]any{
		"topic": "test",
		"data":  "hello",
	},
})

var subTest, _ = sonic.Marshal(map[string]any{
	"option": "subscribe",
	"data": map[string]any{
		"topic": "test",
	},
})

var access_token = func() string {
	username := "test"
	password := "test"
	_, access_token, err := auth.Login(username, password)
	if err != nil {
		panic(err)
	}
	return access_token
}()

var login, _ = sonic.Marshal(map[string]any{
	"option": "login",
	"data": map[string]any{
		"access_token": access_token,
	},
})

func TestSend(t *testing.T) {
	conn, err := net.Dial("tcp", "127.0.0.1:1314")
	if err != nil {
		t.Error(err)
	}
	dstpConn := dstp.NewConn(&conn)
	dstpConn.Send(login, true)
	wg := &sync.WaitGroup{}
	start := time.Now()
	for i := 0; i < 10000; i++ {
		wg.Add(1)
		go func() {
			dstpConn.Send(msgTest, true)
			wg.Done()
		}()
	}
	wg.Wait()
	t.Log("sent", 10000, "messages", "in", time.Since(start))
	time.Sleep(time.Second)
	dstpConn.Close()
}

func TestReceive(t *testing.T) {
	conn, err := net.Dial("tcp", "127.0.0.1:1314")
	if err != nil {
		t.Error(err)
	}
	dstpConn := dstp.NewConn(&conn)
	dstpConn.Send(login, true)
	time.Sleep(time.Second)
	dstpConn.Send(subTest, true)
	var timeCount time.Time
	once := sync.Once{}
	count := atomic.Uint64{}
	t.Log("receiving")
	for {
		data, _, err := dstpConn.Receive()
		if err != nil {
			t.Error(err)
		}
		if TopicIsTest(data, dstpConn) {
			once.Do(func() {
				timeCount = time.Now()
			})
			count.Add(1)
		}
		t.Log(count.Load())
		if count.Load() == 10000 {
			t.Log("received", count.Load(), "messages", "in", time.Since(timeCount))
			break
		}
	}
}

func TestReceive2(t *testing.T) {
	t.Parallel()

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		TestReceive(t)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		TestReceive(t)
		wg.Done()
	}()

	wg.Wait()
}

func TopicIsTest(data []byte, conn *dstp.Conn) bool {
	topic, _ := sonic.Get(data, "data", "topic")
	topicStr, _ := topic.String()
	option, _ := sonic.Get(data, "option")
	optionStr, _ := option.String()
	if optionStr == "ping" {
		conn.Send([]byte("{\"option\":\"pong\"}"), true)
	}
	if topicStr == "test" {
		return true
	}
	return false
}
