package main

import (
	"github.com/EnderCHX/DSMS-go/internal/app"
)

type Msg struct {
	Option string `json:"option"`
	Data   string `json:"data"`
}

func main() {
	//tab := byte('\r')
	//enter := byte('\n')
	//fmt.Println(int('\r'))
	app.Run()

	//msg := Msg{
	//	Option: "send",
	//	Data:   "hello world",
	//}
	//
	//data, _ := json.Marshal(msg)
	//fmt.Println(string(data))
	//
	//socket, err := net.Dial("tcp", "127.0.0.1:8080")
	//if err != nil {
	//	panic(err)
	//}
	//defer socket.Close()
	//go func() {
	//	buf := make([]byte, 1024)
	//	for {
	//		n, err := socket.Read(buf)
	//		if err != nil {
	//			panic(err)
	//		}
	//		println(string(buf[:n]))
	//	}
	//}()
	//
	//go func() {
	//	for {
	//		var msg string
	//		fmt.Scan(&msg)
	//		data := []byte(msg + "\n")
	//		_, err := socket.Write(data)
	//		if err != nil {
	//			panic(err)
	//		}
	//	}
	//}()
	//
	//select {}
}
