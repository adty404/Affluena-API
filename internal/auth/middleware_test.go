package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"affluena-api/internal/httpx"

	"github.com/gin-gonic/gin"
)

func TestAuthMiddlewareRejectsMissingMalformedAndInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokens := NewTokenManager("secret", time.Minute, time.Hour)

	cases := []struct {
		name   string
		header string
	}{
		{name: "missing header", header: ""},
		{name: "missing bearer token", header: "Bearer"},
		{name: "wrong scheme", header: "Basic token"},
		{name: "invalid token", header: "Bearer invalid"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(AuthMiddleware(tokens))
			router.GET("/protected", func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", recorder.Code)
			}
		})
	}
}

func TestAuthMiddlewareSetsUserIDForValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokens := NewTokenManager("secret", time.Minute, time.Hour)
	token, _, err := tokens.IssueAccessToken(User{ID: "user-1", Email: "user@example.com"}, time.Now().UTC())
	if err != nil {
		t.Fatalf("IssueAccessToken returned error: %v", err)
	}

	router := gin.New()
	router.Use(AuthMiddleware(tokens))
	router.GET("/protected", func(c *gin.Context) {
		userID, ok := httpx.UserID(c)
		if !ok {
			t.Fatal("expected user id in context")
		}
		c.String(http.StatusOK, userID)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if recorder.Body.String() != "user-1" {
		t.Fatalf("expected user-1 body, got %q", recorder.Body.String())
	}
}

func TestTokenManagerRejectsExpiredToken(t *testing.T) {
	tokens := NewTokenManager("secret", -time.Minute, time.Hour)
	token, _, err := tokens.IssueAccessToken(User{ID: "user-1", Email: "user@example.com"}, time.Now().UTC().Add(-time.Hour))
	if err != nil {
		t.Fatalf("IssueAccessToken returned error: %v", err)
	}

	if _, err := tokens.ParseAccessToken(token); err == nil {
		t.Fatal("expected expired token to fail")
	}
}
