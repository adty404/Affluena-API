package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUserIDContextRoundTrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	if _, ok := UserID(c); ok {
		t.Fatal("expected missing user id")
	}

	SetUserID(c, "user-1")
	got, ok := UserID(c)
	if !ok || got != "user-1" {
		t.Fatalf("expected user-1, got %q ok=%v", got, ok)
	}
}

func TestMustUserIDWritesUnauthorizedWhenMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	if _, ok := MustUserID(c); ok {
		t.Fatal("expected missing user id to fail")
	}
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}
