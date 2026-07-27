package organizations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Handler struct {
	db *mongo.Database
}

func NewHandler(db *mongo.Database) *Handler {
	return &Handler{db: db}
}

// ListUsers handles GET /users
// @Summary List users in the caller's organization
// @Tags Organizations
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {array} User
// @Failure 401 {object} map[string]string
// @Router /users [get]
func (h *Handler) ListUsers(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Session not provided"})
		return
	}

	_, org, err := GetUserWithOrganization(c.Request.Context(), h.db, userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User has no organization"})
		return
	}

	users, err := ListUsersByOrg(c.Request.Context(), h.db, org.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error listing users"})
		return
	}

	c.JSON(http.StatusOK, users)
}
