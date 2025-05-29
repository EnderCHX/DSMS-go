package test

import (
	"github.com/EnderCHX/DSMS-go/internal/dstp"
	"github.com/bytedance/sonic"
	"net"
	"sync"
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

var access_token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InRlc3QyIiwicm9sZSI6IlVTRVIiLCJhdmF0YXIiOiIiLCJzaWduYXR1cmUiOiLns7vnu5_ljp_oo4Xnrb7lkI3vvIEiLCJpc3MiOiJjaHhjLmNjIiwiZXhwIjoxNzQ4MTYwMzc5LCJpYXQiOjE3NDgxNTY3Nzl9.Zuqw5xAgdg_nndzmShEG9yjleziAbWg0TqtQt6RkFTM"
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
	for i := 0; i < 10000; i++ {
		wg.Add(1)
		go func() {
			dstpConn.Send(msgTest, true)
			wg.Done()
		}()
	}
	wg.Wait()
	dstpConn.Close()
}

func TestReceive(t *testing.T) {
	conn, err := net.Dial("tcp", "127.0.0.1:1314")
	if err != nil {
		t.Error(err)
	}
	dstpConn := dstp.NewConn(&conn)
	dstpConn.Send(login, true)
	dstpConn.Send(subTest, true)
	var timeCount time.Time
	once := sync.Once{}
	count := 0
	for {
		data, _, err := dstpConn.Receive()
		if err != nil {
			t.Error(err)
		}
		if TopicIsTest(data) {
			once.Do(func() {
				timeCount = time.Now()
			})
			count++
		}
		if count == 10000 {
			t.Log("received", count, "messages", "in", time.Since(timeCount))
			break
		}
	}
}

func TestReceive2(t *testing.T) {
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

func TopicIsTest(data []byte) bool {
	topic, _ := sonic.Get(data, "data", "topic")
	topicStr, _ := topic.String()
	if topicStr == "test" {
		return true
	} else {
		return false
	}
}
