package middleware

import (
	"net/http"

	"arkivy-api/internal/zitadel"

	"github.com/gin-gonic/gin"
)

// SessionMiddleware validates the user session by querying Zitadel
func SessionMiddleware(zClient *zitadel.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.GetHeader("X-Session-Id")
		sessionToken := c.GetHeader("X-Session-Token")

		if sessionID == "" || sessionToken == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session not provided"})
			return
		}

		info, err := zClient.GetSession(c.Request.Context(), sessionID, sessionToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired session"})
			return
		}

		if info.UserID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session without verified user"})
			return
		}

		// Verify that the password was authenticated
		if info.Factors.Password == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Incomplete authentication"})
			return
		}

		// Get user roles
		roles, err := zClient.GetUserRoles(c.Request.Context(), info.UserID)
		if err != nil {
			roles = []string{}
		}

		// Inject into Gin context
		c.Set("userID", info.UserID)
		c.Set("loginName", info.LoginName)
		c.Set("displayName", info.DisplayName)
		c.Set("roles", roles)
		c.Next()
	}
}

// RequireRole verifies that the user has at least one of the required roles
func RequireRole(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles, exists := c.Get("roles")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "No roles assigned"})
			return
		}

		roles, ok := userRoles.([]string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Error reading roles"})
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

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied: insufficient role"})
	}
}
