package main

import (
	"github.com/EnderCHX/DSMS-go/internal/message_hub"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}
	hub := message_hub.NewHub("0.0.0.0", "8080")
	hub.Run()

	select {}
}
