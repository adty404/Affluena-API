package httpx

import (
	"net/http"
	"strconv"

	"affluena-api/internal/page"

	"github.com/gin-gonic/gin"
)

const (
	DefaultPageLimit = 100
	MaxPageLimit     = 200
)

func ParsePage(c *gin.Context, defaultSort string, allowedSorts map[string]struct{}) (page.Params, bool) {
	limit, ok := parseNonNegativeInt(c, "limit", DefaultPageLimit)
	if !ok {
		return page.Params{}, false
	}
	if limit == 0 {
		Error(c, http.StatusBadRequest, "limit must be positive")
		return page.Params{}, false
	}
	if limit > MaxPageLimit {
		Error(c, http.StatusBadRequest, "limit must be at most 200")
		return page.Params{}, false
	}

	offset, ok := parseNonNegativeInt(c, "offset", 0)
	if !ok {
		return page.Params{}, false
	}

	sort := c.Query("sort")
	if sort == "" {
		sort = defaultSort
	}
	if _, ok := allowedSorts[sort]; !ok {
		Error(c, http.StatusBadRequest, "invalid sort")
		return page.Params{}, false
	}

	return page.Params{Limit: limit, Offset: offset, Sort: sort}, true
}

func parseNonNegativeInt(c *gin.Context, key string, defaultValue int) (int, bool) {
	value := c.Query(key)
	if value == "" {
		return defaultValue, true
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		Error(c, http.StatusBadRequest, key+" must be a non-negative integer")
		return 0, false
	}
	return parsed, true
}
