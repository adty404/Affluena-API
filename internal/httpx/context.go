package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const userIDKey = "user_id"

func SetUserID(c *gin.Context, userID string) {
	c.Set(userIDKey, userID)
}

func UserID(c *gin.Context) (string, bool) {
	value, exists := c.Get(userIDKey)
	if !exists {
		return "", false
	}
	userID, ok := value.(string)
	return userID, ok && userID != ""
}

func MustUserID(c *gin.Context) (string, bool) {
	userID, ok := UserID(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return "", false
	}
	return userID, true
}
