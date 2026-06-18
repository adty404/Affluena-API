package goal

import (
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
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	goal, err := h.usecase.Create(c.Request.Context(), userID, input)
	if err != nil {
		httpx.WriteError(c, err)
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
		httpx.WriteError(c, err)
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
	id, ok := httpx.GetUUIDParam(c, "id")
	if !ok {
		return
	}
	goal, err := h.usecase.Get(c.Request.Context(), userID, id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, goal)
}

func (h *Handler) Update(c *gin.Context) {
	var input UpdateGoalInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	id, ok := httpx.GetUUIDParam(c, "id")
	if !ok {
		return
	}
	goal, err := h.usecase.Update(c.Request.Context(), userID, id, input)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, goal)
}

func (h *Handler) InviteMember(c *gin.Context) {
	var input InviteMemberInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	id, ok := httpx.GetUUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.usecase.InviteMember(c.Request.Context(), userID, id, input); err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"message": "invited"})
}

func (h *Handler) RespondInvite(c *gin.Context) {
	var input RespondInviteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	id, ok := httpx.GetUUIDParam(c, "id")
	if !ok {
		return
	}
	memberID, ok := httpx.GetUUIDParam(c, "user_id")
	if !ok {
		return
	}
	if err := h.usecase.RespondInvite(c.Request.Context(), userID, id, memberID, input); err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"message": "responded"})
}
