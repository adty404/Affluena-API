package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetUUIDParam retrieves and validates a UUID path parameter.
// Returns the UUID string and true if valid, false otherwise.
// If invalid, writes a 400 error response.
func GetUUIDParam(c *gin.Context, name string) (string, bool) {
	value := c.Param(name)
	if value == "" {
		Error(c, http.StatusBadRequest, name+" is required")
		return "", false
	}
	if _, err := uuid.Parse(value); err != nil {
		Error(c, http.StatusBadRequest, "invalid "+name+" format")
		return "", false
	}
	return value, true
}
