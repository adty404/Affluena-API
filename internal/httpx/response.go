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

// WriteError maps domain errors to appropriate HTTP status codes.
// Returns true if the error was handled and response was sent.
func WriteError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}

	// Check PublicError first
	var pubErr PublicError
	if errors.As(err, &pubErr) {
		Error(c, pubErr.HTTPStatus, pubErr.Message)
		return true
	}

	// Check common error patterns
	errMsg := err.Error()
	lowerMsg := strings.ToLower(errMsg)

	// Unauthorized
	if strings.Contains(lowerMsg, "unauthorized") || strings.Contains(lowerMsg, "invalid email or password") {
		Error(c, http.StatusUnauthorized, "unauthorized access")
		return true
	}

	// Forbidden
	if strings.Contains(lowerMsg, "forbidden") || strings.Contains(lowerMsg, "not authorized") || strings.Contains(lowerMsg, "only creator") {
		Error(c, http.StatusForbidden, "forbidden action")
		return true
	}

	// Not found
	if strings.Contains(lowerMsg, "not found") || errors.Is(err, ErrNotFound) {
		Error(c, http.StatusNotFound, "resource not found")
		return true
	}

	// Conflict
	if strings.Contains(lowerMsg, "already exists") || strings.Contains(lowerMsg, "conflict") || strings.Contains(lowerMsg, "duplicate") {
		Error(c, http.StatusConflict, "conflict in resource state")
		return true
	}

	// Validation errors (bad request)
	if strings.HasPrefix(lowerMsg, "invalid") || strings.HasPrefix(lowerMsg, "required") ||
		strings.Contains(lowerMsg, "must be") || strings.Contains(lowerMsg, "cannot") {
		Error(c, http.StatusBadRequest, "invalid request")
		return true
	}

	// Default: internal server error with generic message
	Error(c, http.StatusInternalServerError, "internal server error")
	return true
}

// InternalError sanitizes internal errors before sending to client.
// Deprecated: Use WriteError instead for better error mapping.
func InternalError(c *gin.Context, err error) {
	WriteError(c, err)
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

// Common domain errors that can be checked with errors.Is
var (
	ErrNotFound = errors.New("resource not found")
)

// IsNotFound checks if an error is a not found error.
func IsNotFound(err error) bool {
	return err != nil && (errors.Is(err, ErrNotFound) || strings.Contains(err.Error(), "not found"))
}

// IsUnauthorized checks if an error is an unauthorized error.
func IsUnauthorized(err error) bool {
	if err == nil {
		return false
	}
	lowerMsg := strings.ToLower(err.Error())
	return strings.Contains(lowerMsg, "unauthorized") || strings.Contains(lowerMsg, "invalid email or password")
}

// IsForbidden checks if an error is a forbidden error.
func IsForbidden(err error) bool {
	if err == nil {
		return false
	}
	lowerMsg := strings.ToLower(err.Error())
	return strings.Contains(lowerMsg, "forbidden") || strings.Contains(lowerMsg, "not authorized")
}
