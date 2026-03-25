package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "API de Arkivy funcionando",
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	_ = r.Run(":" + port)
}
