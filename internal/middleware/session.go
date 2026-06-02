package middleware

import (
	"net/http"

	"arkivy-api/internal/zitadel"

	"github.com/gin-gonic/gin"
)

// SessionMiddleware valida la sesión del usuario consultando Zitadel
func SessionMiddleware(zClient *zitadel.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.GetHeader("X-Session-Id")
		sessionToken := c.GetHeader("X-Session-Token")

		if sessionID == "" || sessionToken == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Sesión no proporcionada"})
			return
		}

		info, err := zClient.GetSession(c.Request.Context(), sessionID, sessionToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Sesión inválida o expirada"})
			return
		}

		if info.UserID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Sesión sin usuario verificado"})
			return
		}

		// Verificar que el password fue verificado
		if info.Factors.Password == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Autenticación incompleta"})
			return
		}

		// Obtener roles del usuario
		roles, err := zClient.GetUserRoles(c.Request.Context(), info.UserID)
		if err != nil {
			roles = []string{} // Sin roles no es fatal
		}

		// Inyectar en el contexto de Gin
		c.Set("userID", info.UserID)
		c.Set("loginName", info.LoginName)
		c.Set("displayName", info.DisplayName)
		c.Set("roles", roles)
		c.Next()
	}
}

// RequireRole verifica que el usuario tenga al menos uno de los roles requeridos
func RequireRole(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles, exists := c.Get("roles")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Sin roles asignados"})
			return
		}

		roles, ok := userRoles.([]string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Error leyendo roles"})
			return
		}

		for _, required := range requiredRoles {
			for _, userRole := range roles {
				if required == userRole {
					c.Next()
					return
				}
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Acceso denegado: rol insuficiente"})
	}
}
