package httpx

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

func BindOptionalJSON(c *gin.Context, dest any, message string) bool {
	if c.Request.Body == nil || c.Request.ContentLength == 0 {
		return true
	}
	if err := c.ShouldBindJSON(dest); err != nil {
		if errors.Is(err, io.EOF) {
			return true
		}
		Error(c, http.StatusBadRequest, message)
		return false
	}
	return true
}
