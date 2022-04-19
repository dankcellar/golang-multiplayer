package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	secret := "m9kaxFePwArUwRs53qaOsSoFP6bjpFD6"
	hub := newHub()
	go hub.run()

	router := gin.New()
	router.LoadHTMLGlob("public/*.html")

	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Server running",
		})
	})

	router.GET("/status/:roomId", func(c *gin.Context) {
	})

	router.GET("/chat/:roomId", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	router.GET("/room/:roomId", func(c *gin.Context) {
		room := c.Param("roomId")
		ipAddr := c.ClientIP()
		h := hmac.New(sha256.New, []byte(secret))
		h.Write([]byte(ipAddr))
		userToken := hex.EncodeToString(h.Sum(nil))
		serveWs(hub, c.Writer, c.Request, room, userToken)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	router.Run(":" + port)
}
