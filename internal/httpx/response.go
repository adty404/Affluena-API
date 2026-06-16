package httpx

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type errorResponse struct {
	Error string `json:"error"`
}

func JSON(c *gin.Context, status int, value any) {
	c.JSON(status, value)
}

func Error(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, errorResponse{Error: message})
}

// InternalError sanitizes internal errors before sending to client.
// For server errors (5xx), it returns a generic message.
// For client errors (4xx), it returns the original message if safe.
func InternalError(c *gin.Context, err error) {
	if err == nil {
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	// Check if it's a PublicError - expose it with its status
	var pubErr PublicError
	if errors.As(err, &pubErr) {
		Error(c, pubErr.HTTPStatus, pubErr.Message)
		return
	}

	// For other errors, check if the message is safe to expose
	if shouldExposeError(err.Error()) {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// Generic internal error message
	Error(c, http.StatusInternalServerError, "internal server error")
}

// shouldExposeError determines if an error message is safe to expose.
func shouldExposeError(msg string) bool {
	safePrefixes := []string{
		"invalid", "not found", "already exists", "required",
		"must be", "cannot", "unauthorized", "forbidden",
		"email and password", "invalid email or password",
	}

	lowerMsg := strings.ToLower(msg)
	for _, prefix := range safePrefixes {
		if strings.HasPrefix(lowerMsg, prefix) || strings.Contains(lowerMsg, prefix) {
			return true
		}
	}
	return false
}

// PublicError wraps an error that is safe to expose to clients.
type PublicError struct {
	Message    string
	HTTPStatus int
}

func (e PublicError) Error() string {
	return e.Message
}

// NewPublicError creates a new PublicError with the given message and status.
func NewPublicError(message string, status int) PublicError {
	return PublicError{Message: message, HTTPStatus: status}
}
