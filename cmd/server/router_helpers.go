package main

import "github.com/gin-gonic/gin"

var allowedCORSOrigins = map[string]struct{}{
	"http://localhost:3000":            {},
	"http://127.0.0.1:3000":            {},
	"https://payment-kita.excitech.id": {},
	"https://paymentkita.netlify.app":  {},
	"https://api-dompet-ku.excitech.id": {},
}

func applyCORSMiddleware(r *gin.Engine) {
	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if _, ok := allowedCORSOrigins[origin]; ok {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else if origin == "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-PK-Key, X-PK-Timestamp, X-PK-Signature")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})
}

func registerHealthRoute(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "payment-kita-backend",
			"version": "0.2.0",
		})
	})
}
