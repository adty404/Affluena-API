package category

import (
	"context"
	"errors"
	"net/http"

	"affluena-api/internal/httpx"
	"affluena-api/internal/page"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	usecase categoryUseCase
}

type categoryUseCase interface {
	Create(ctx context.Context, userID string, input CreateCategoryInput) (Category, error)
	List(ctx context.Context, userID string, categoryType string, pagination page.Params) (page.Result[Category], error)
	Get(ctx context.Context, userID string, id string) (Category, error)
	Update(ctx context.Context, userID string, id string, input UpdateCategoryInput) (Category, error)
	Delete(ctx context.Context, userID string, id string) error
}

func NewHandler(usecase categoryUseCase) *Handler {
	return &Handler{usecase: usecase}
}

type categoryRequest struct {
	Name string `json:"name" binding:"required"`
	Type string `json:"type" binding:"required"`
}

func (h *Handler) Create(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	var req categoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	category, err := h.usecase.Create(c.Request.Context(), userID, CreateCategoryInput{Name: req.Name, Type: req.Type})
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(c, http.StatusCreated, category)
}

func (h *Handler) List(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	pagination, ok := httpx.ParsePage(c, "type_name_asc", categorySorts)
	if !ok {
		return
	}
	result, err := h.usecase.List(c.Request.Context(), userID, c.Query("type"), pagination)
	if err != nil {
		if errors.Is(err, ErrInvalidCategoryType) {
			httpx.Error(c, http.StatusBadRequest, err.Error())
			return
		}
		httpx.Error(c, http.StatusInternalServerError, "list categories failed")
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"categories": result.Items, "pagination": result.Pagination})
}

var categorySorts = map[string]struct{}{
	"type_name_asc":   {},
	"type_name_desc":  {},
	"name_asc":        {},
	"name_desc":       {},
	"created_at_desc": {},
	"created_at_asc":  {},
}

func (h *Handler) Get(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	category, err := h.usecase.Get(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		if NotFound(err) {
			httpx.Error(c, http.StatusNotFound, "category not found")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, "get category failed")
		return
	}
	httpx.JSON(c, http.StatusOK, category)
}

func (h *Handler) Update(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	var req categoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	category, err := h.usecase.Update(c.Request.Context(), userID, c.Param("id"), UpdateCategoryInput{Name: req.Name, Type: req.Type})
	if err != nil {
		if NotFound(err) {
			httpx.Error(c, http.StatusNotFound, "category not found")
			return
		}
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(c, http.StatusOK, category)
}

func (h *Handler) Delete(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	if err := h.usecase.Delete(c.Request.Context(), userID, c.Param("id")); err != nil {
		if NotFound(err) {
			httpx.Error(c, http.StatusNotFound, "category not found")
			return
		}
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}
