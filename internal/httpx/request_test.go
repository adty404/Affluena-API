package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBindOptionalJSONAllowsEmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	var req struct {
		Note string `json:"note"`
	}
	if !BindOptionalJSON(c, &req, "invalid request body") {
		t.Fatal("expected empty body to bind successfully")
	}
}

func TestBindOptionalJSONRejectsInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{"))
	c.Request.Header.Set("content-type", "application/json")

	var req struct {
		Note string `json:"note"`
	}
	if BindOptionalJSON(c, &req, "invalid request body") {
		t.Fatal("expected invalid JSON to fail")
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 response, got %d", recorder.Code)
	}
}
