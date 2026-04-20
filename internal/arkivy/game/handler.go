package game

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type handler struct {
	service GameService
}

func newHandler(svc GameService) *handler {
	return &handler{service: svc}
}

func (h *handler) create(c *gin.Context) {
	var body struct {
		Name   string `json:"name" binding:"required"`
		Amount int    `json:"amount" binding:"min=0"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	game, err := h.service.CreateGame(body.Name, body.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, game)
}

func (h *handler) list(c *gin.Context) {
	games, err := h.service.ListGames()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, games)
}
