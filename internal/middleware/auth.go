package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware valida tokens JWT emitidos por Zitadel.
//
// Por ahora es un placeholder estructurado listo para conectar con el
// introspection endpoint de Zitadel o validación local con JWKS.
//
// Cuando se integre Zitadel, aquí va:
//   - Obtener JWKS desde: https://<ZITADEL_DOMAIN>/oauth/v2/keys
//   - Validar firma, expiración y claims (iss, aud, sub)
//   - Inyectar el sub (userID) en el contexto de Gin
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

		// TODO: validar token con Zitadel JWKS cuando se configure
		// Por ahora solo verifica que no esté vacío
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token vacío"})
			return
		}

		// Cuando Zitadel esté integrado, inyectar el userID así:
		// c.Set("userID", claims.Subject)

		c.Next()
	}
}
