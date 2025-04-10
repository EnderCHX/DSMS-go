package message_hub

type MessageHub struct {
	// 订阅者
	subscribers map[string][]chan string
}
