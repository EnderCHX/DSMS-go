package main

import (
	"github.com/EnderCHX/DSMS-go/internal/app"
)

type Msg struct {
	Option string `json:"option"`
	Data   string `json:"data"`
}

func main() {
	app.Run()
}
