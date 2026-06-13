package auth

import (
	"net/http"
	"strings"

	"affluena-api/internal/httpx"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(tokens *TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			httpx.Error(c, http.StatusUnauthorized, "authorization header is required")
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			httpx.Error(c, http.StatusUnauthorized, "bearer token is required")
			return
		}

		claims, err := tokens.ParseAccessToken(parts[1])
		if err != nil {
			httpx.Error(c, http.StatusUnauthorized, "invalid access token")
			return
		}

		httpx.SetUserID(c, claims.UserID)
		c.Next()
	}
}
