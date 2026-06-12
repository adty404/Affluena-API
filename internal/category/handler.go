package category

import (
	"net/http"

	"affluena/internal/httpx"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
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
	if !IsValidType(req.Type) {
		httpx.Error(c, http.StatusBadRequest, "invalid category type")
		return
	}

	category, err := h.repo.Create(c.Request.Context(), userID, req.Name, req.Type)
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

	categories, err := h.repo.List(c.Request.Context(), userID)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "list categories failed")
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"categories": categories})
}

func (h *Handler) Get(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	category, err := h.repo.Get(c.Request.Context(), userID, c.Param("id"))
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
	if !IsValidType(req.Type) {
		httpx.Error(c, http.StatusBadRequest, "invalid category type")
		return
	}

	category, err := h.repo.Update(c.Request.Context(), userID, c.Param("id"), req.Name, req.Type)
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

	if err := h.repo.Delete(c.Request.Context(), userID, c.Param("id")); err != nil {
		if NotFound(err) {
			httpx.Error(c, http.StatusNotFound, "category not found")
			return
		}
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}
