package category

import (
	"context"
	"net/http"

	"affluena-api/internal/httpx"
	"affluena-api/internal/page"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	Reorder(ctx context.Context, userID string, ids []string) error
}

func NewHandler(usecase categoryUseCase) *Handler {
	return &Handler{usecase: usecase}
}

type categoryRequest struct {
	Name     string  `json:"name" binding:"required"`
	Type     string  `json:"type" binding:"required"`
	ParentID *string `json:"parent_id"`
	Icon     string  `json:"icon"`
	Color    string  `json:"color"`
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
	category, err := h.usecase.Create(c.Request.Context(), userID, CreateCategoryInput{Name: req.Name, Type: req.Type, ParentID: req.ParentID, Icon: req.Icon, Color: req.Color})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.JSON(c, http.StatusCreated, category)
}

func (h *Handler) List(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	pagination, ok := httpx.ParsePage(c, "position_asc", categorySorts)
	if !ok {
		return
	}
	result, err := h.usecase.List(c.Request.Context(), userID, c.Query("type"), pagination)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"categories": result.Items, "pagination": result.Pagination})
}

var categorySorts = map[string]struct{}{
	"position_asc":    {},
	"type_name_asc":   {},
	"type_name_desc":  {},
	"name_asc":        {},
	"name_desc":       {},
	"created_at_desc": {},
	"created_at_asc":  {},
}

type reorderCategoriesRequest struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

func (h *Handler) Reorder(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	var req reorderCategoriesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	for _, id := range req.IDs {
		if _, err := uuid.Parse(id); err != nil {
			httpx.Error(c, http.StatusBadRequest, "invalid category id format")
			return
		}
	}
	if err := h.usecase.Reorder(c.Request.Context(), userID, req.IDs); err != nil {
		httpx.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
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

	category, err := h.usecase.Get(c.Request.Context(), userID, id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, category)
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

	var req categoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	category, err := h.usecase.Update(c.Request.Context(), userID, id, UpdateCategoryInput{Name: req.Name, Type: req.Type, ParentID: req.ParentID, Icon: req.Icon, Color: req.Color})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, category)
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
		httpx.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
