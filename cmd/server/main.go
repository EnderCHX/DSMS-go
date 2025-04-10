package main

import (
	"fmt"
	"net"
)

func main() {
	tab := byte('\r')
	enter := byte('\n')

	socket, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		panic(err)
	}
	defer socket.Close()
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := socket.Read(buf)
			if err != nil {
				panic(err)
			}
			println(string(buf[:n]))
		}
	}()

	go func() {
		for {
			var msg string
			fmt.Scan(&msg)
			data := []byte(msg)
			data = append(data, tab, enter)
			_, err := socket.Write(data)
			if err != nil {
				panic(err)
			}
		}
	}()

	select {}
}
