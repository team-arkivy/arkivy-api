package game

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func RegisterRoutes(rg *gin.RouterGroup, db *mongo.Database) {
	repo := NewRepository(db)
	svc := NewService(repo)
	h := newHandler(svc)

	games := rg.Group("/games")
	games.POST("", h.create)
	games.GET("", h.list)
}
