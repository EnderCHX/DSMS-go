package app

import (
	"github.com/EnderCHX/DSMS-go/utils/log"
	"github.com/gin-gonic/gin"
)

func RunGin(ip, port string) {
	logger := log.NewLogger("[Client]", "logs/client.go", "debug")
	r := gin.New()
	r.Use(gin.Recovery(), log.GinZapLogger(logger))

	r.LoadHTMLGlob("resources/html/**/*")

	r.GET("/", func(c *gin.Context) {

	})

	r.Run(ip + ":" + port)
}
