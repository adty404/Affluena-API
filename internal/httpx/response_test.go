package httpx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetUUIDParam_Valid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/550e8400-e29b-41d4-a716-446655440000", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "550e8400-e29b-41d4-a716-446655440000"}}

	got, ok := GetUUIDParam(c, "id")
	assert.True(t, ok)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", got)
}

func TestGetUUIDParam_InvalidFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/not-a-uuid", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "not-a-uuid"}}

	_, ok := GetUUIDParam(c, "id")
	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, c.Writer.Status())
}

func TestGetUUIDParam_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: ""}}

	_, ok := GetUUIDParam(c, "id")
	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, c.Writer.Status())
}

func TestWriteError_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	handled := WriteError(c, errors.New("wallet not found"))
	assert.True(t, handled)
	assert.Equal(t, http.StatusNotFound, c.Writer.Status())
}

func TestWriteError_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	handled := WriteError(c, errors.New("unauthorized"))
	assert.True(t, handled)
	assert.Equal(t, http.StatusUnauthorized, c.Writer.Status())
}

func TestWriteError_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	handled := WriteError(c, errors.New("not authorized"))
	assert.True(t, handled)
	assert.Equal(t, http.StatusForbidden, c.Writer.Status())
}

func TestWriteError_Conflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	handled := WriteError(c, errors.New("email already exists"))
	assert.True(t, handled)
	assert.Equal(t, http.StatusConflict, c.Writer.Status())
}

func TestWriteError_Internal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	handled := WriteError(c, errors.New("database connection failed"))
	assert.True(t, handled)
	assert.Equal(t, http.StatusInternalServerError, c.Writer.Status())
}

func TestWriteError_PublicError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	handled := WriteError(c, NewPublicError("custom message", http.StatusTeapot))
	assert.True(t, handled)
	assert.Equal(t, http.StatusTeapot, c.Writer.Status())
}

func TestWriteError_Nil(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	handled := WriteError(c, nil)
	assert.False(t, handled)
}

func TestIsNotFound(t *testing.T) {
	assert.True(t, IsNotFound(ErrNotFound))
	assert.True(t, IsNotFound(errors.New("wallet not found")))
	assert.False(t, IsNotFound(errors.New("other error")))
	assert.False(t, IsNotFound(nil))
}

func TestIsUnauthorized(t *testing.T) {
	assert.True(t, IsUnauthorized(errors.New("unauthorized")))
	assert.True(t, IsUnauthorized(errors.New("Invalid email or password")))
	assert.False(t, IsUnauthorized(errors.New("other error")))
	assert.False(t, IsUnauthorized(nil))
}

func TestIsForbidden(t *testing.T) {
	assert.True(t, IsForbidden(errors.New("forbidden")))
	assert.True(t, IsForbidden(errors.New("not authorized")))
	assert.False(t, IsForbidden(errors.New("other error")))
	assert.False(t, IsForbidden(nil))
}

func TestIsConflict(t *testing.T) {
	assert.True(t, IsConflict(ErrConflict))
	assert.True(t, IsConflict(errors.New("already exists")))
	assert.False(t, IsConflict(errors.New("other error")))
	assert.False(t, IsConflict(nil))
}

func TestIsValidation(t *testing.T) {
	assert.True(t, IsValidation(ErrValidation))
	assert.True(t, IsValidation(errors.New("invalid input")))
	assert.False(t, IsValidation(errors.New("other error")))
	assert.False(t, IsValidation(nil))
}

func TestWriteError_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	handled := WriteError(c, errors.New("invalid param"))
	assert.True(t, handled)
	assert.Equal(t, http.StatusBadRequest, c.Writer.Status())
}
