package tag

import (
	"context"
	"net/http"

	"affluena-api/internal/httpx"
	"affluena-api/internal/page"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	usecase tagUseCase
}

type tagUseCase interface {
	Create(ctx context.Context, userID string, input CreateTagInput) (Tag, error)
	List(ctx context.Context, userID string, pagination page.Params) (page.Result[Tag], error)
	Get(ctx context.Context, userID string, id string) (Tag, error)
	Update(ctx context.Context, userID string, id string, input UpdateTagInput) (Tag, error)
	Delete(ctx context.Context, userID string, id string) error
}

func NewHandler(usecase tagUseCase) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) Create(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	var input CreateTagInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	tag, err := h.usecase.Create(c.Request.Context(), userID, input)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	httpx.JSON(c, http.StatusCreated, tag)
}

func (h *Handler) List(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	pagination, ok := httpx.ParsePage(c, "created_at_desc", tagSorts)
	if !ok {
		return
	}

	result, err := h.usecase.List(c.Request.Context(), userID, pagination)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "list tags failed")
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"tags": result.Items, "pagination": result.Pagination})
}

var tagSorts = map[string]struct{}{
	"created_at_desc": {},
	"created_at_asc":  {},
	"name_asc":        {},
	"name_desc":       {},
}

func (h *Handler) Get(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	id, ok := httpx.GetUUIDParam(c, "id")
	if !ok {
		return
	}

	tag, err := h.usecase.Get(c.Request.Context(), userID, id)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, tag)
}

func (h *Handler) Update(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	id, ok := httpx.GetUUIDParam(c, "id")
	if !ok {
		return
	}

	var input UpdateTagInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	tag, err := h.usecase.Update(c.Request.Context(), userID, id, input)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, tag)
}

func (h *Handler) Delete(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	id, ok := httpx.GetUUIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.usecase.Delete(c.Request.Context(), userID, id); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func writeError(c *gin.Context, err error) {
	if NotFound(err) {
		httpx.Error(c, http.StatusNotFound, "resource not found")
		return
	}
	httpx.Error(c, http.StatusBadRequest, "invalid request")
}
