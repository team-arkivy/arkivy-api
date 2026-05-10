package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zitadel/zitadel-go/v3/pkg/authorization"
	"github.com/zitadel/zitadel-go/v3/pkg/authorization/oauth"
	"github.com/zitadel/zitadel-go/v3/pkg/zitadel"
)

var authz *authorization.Authorizer[*oauth.IntrospectionContext]

// InitAuth inicializa el cliente de Zitadel.
// Llamar una sola vez al arrancar el servidor.
func InitAuth(domain, keyPath string) error {
	var err error
	authz, err = authorization.New(
		context.Background(),
		zitadel.New(domain),
		oauth.DefaultAuthorization(keyPath),
	)
	return err
}

// AuthMiddleware valida tokens JWT emitidos por Zitadel.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header requerido"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Formato inválido. Usar: Bearer <token>"})
			return
		}

		token := parts[1]

		ctx, err := authz.CheckAuthorization(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
			return
		}

		c.Set("userID", ctx.UserID())
		c.Next()
	}
}
