package main

import (
	"fmt"
	"time"
)

func main() {
	bro0 := make(chan struct{})
	bro1 := make(chan struct{})

	go func() {
		for {
			time.Sleep(time.Second * 1)
			<-bro0
			fmt.Println("你看看你后面呢")
			bro1 <- struct{}{}
		}
	}()

	go func() {
		for {
			time.Sleep(time.Second * 1)
			<-bro1
			fmt.Println("你再看看你后面呢")
			bro0 <- struct{}{}
		}
	}()
	bro0 <- struct{}{}

	select {}
}
