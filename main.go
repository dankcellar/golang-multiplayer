package main

import (
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	hub := newHub()
	go hub.run()

	router := gin.New()
	router.LoadHTMLGlob("public/*.html")

	router.GET("/room/:roomID", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	router.GET("/ws/:roomID", func(c *gin.Context) {
		roomID := c.Param("roomID")
		serveWs(hub, c.Writer, c.Request, roomID)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	router.Run(":" + port)
}
