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

// @Summary      Crear un juego
// @Description  Crea un nuevo juego en la base de datos
// @Tags         games
// @Accept       json
// @Produce      json
// @Param        body  body      CreateGameRequest  true  "Datos del juego"
// @Success      201   {object}  Game
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /games [post]
func (h *handler) create(c *gin.Context) {
	var body CreateGameRequest

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

// @Summary      Listar juegos
// @Description  Retorna todos los juegos disponibles
// @Tags         games
// @Produce      json
// @Success      200  {array}   Game
// @Failure      500  {object}  map[string]string
// @Router       /games [get]
func (h *handler) list(c *gin.Context) {
	games, err := h.service.ListGames()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, games)
}
