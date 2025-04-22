package main

import (
	"github.com/EnderCHX/DSMS-go/internal/message_hub"
)

func main() {
	hub := message_hub.NewHub("0.0.0.0", "1314")
	hub.Run()

	select {}
}
