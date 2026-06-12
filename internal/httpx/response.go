package httpx

import "github.com/gin-gonic/gin"

type errorResponse struct {
	Error string `json:"error"`
}

func JSON(c *gin.Context, status int, value any) {
	c.JSON(status, value)
}

func Error(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, errorResponse{Error: message})
}
