package app

import (
	"github.com/gin-gonic/gin"
	"time"
)

var value int

func RunGin(ip, port string) {
	//logger := log.NewLogger("[Client]", "logs/client.go", "debug")
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", gin.H{
			"value": value,
		})
	})

	go func() {
		for {
			time.Sleep(time.Second * 1)
			value++
		}
	}()

	r.Run(ip + ":" + port)
}
