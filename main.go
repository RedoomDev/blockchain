package main

import (
	"github.com/joho/godotenv"
	"github.com/gin-gonic/gin"
	"fmt"
	"os"
)


func main() {
	godotenv.Load(".env")
	mongo := os.Getenv("MONGO_URL")

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
		r.Run(":5000")
}