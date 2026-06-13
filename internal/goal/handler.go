package goal

import (
	"errors"
	"net/http"

	"affluena-api/internal/httpx"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	usecase *Usecase
}

func NewHandler(usecase *Usecase) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) Create(c *gin.Context) {
	var input CreateGoalInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	goal, err := h.usecase.Create(c.Request.Context(), userID, input)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(c, http.StatusCreated, goal)
}

func (h *Handler) List(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	goals, err := h.usecase.List(c.Request.Context(), userID)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	// ensure not null json array
	if goals == nil {
		goals = make([]Goal, 0)
	}
	httpx.JSON(c, http.StatusOK, goals)
}

func (h *Handler) Get(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	id := c.Param("id")
	goal, err := h.usecase.Get(c.Request.Context(), userID, id)
	if NotFound(err) {
		httpx.Error(c, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(c, http.StatusOK, goal)
}

func (h *Handler) InviteMember(c *gin.Context) {
	var input InviteMemberInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	id := c.Param("id")
	if err := h.usecase.InviteMember(c.Request.Context(), userID, id, input); err != nil {
		if NotFound(err) {
			httpx.Error(c, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, ErrNotAuthorized) {
			httpx.Error(c, http.StatusForbidden, err.Error())
			return
		}
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"message": "invited"})
}

func (h *Handler) RespondInvite(c *gin.Context) {
	var input RespondInviteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	id := c.Param("id")
	if err := h.usecase.RespondInvite(c.Request.Context(), userID, id, c.Param("user_id"), input); err != nil {
		if NotFound(err) {
			httpx.Error(c, http.StatusNotFound, err.Error())
			return
		}
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"message": "responded"})
}
