package debt

import (
	"context"
	"errors"
	"net/http"
	"time"

	"affluena-api/internal/httpx"
	"affluena-api/internal/page"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	usecase debtUseCase
}

type debtUseCase interface {
	Create(ctx context.Context, userID string, input DebtInput) (Debt, error)
	List(ctx context.Context, userID string, pagination page.Params) (page.Result[Debt], error)
	Get(ctx context.Context, userID string, id string) (Debt, error)
	Update(ctx context.Context, userID string, id string, update DebtUpdate) (Debt, error)
	Delete(ctx context.Context, userID string, id string) error
	Pay(ctx context.Context, userID string, id string, amountMinor int64, paidAt time.Time, note string) (DebtPayment, error)
}

func NewHandler(usecase debtUseCase) *Handler {
	return &Handler{usecase: usecase}
}

type createRequest struct {
	Type                   DebtType `json:"type" binding:"required"`
	CounterpartyName       string   `json:"counterparty_name" binding:"required"`
	WalletID               string   `json:"wallet_id" binding:"required"`
	DisbursementCategoryID string   `json:"disbursement_category_id" binding:"required"`
	PaymentCategoryID      string   `json:"payment_category_id" binding:"required"`
	PrincipalAmountMinor   int64    `json:"principal_amount_minor" binding:"required"`
	OpenedAt               string   `json:"opened_at"`
	DueDate                string   `json:"due_date"`
	Note                   string   `json:"note"`
}

type updateRequest struct {
	CounterpartyName string     `json:"counterparty_name" binding:"required"`
	DueDate          string     `json:"due_date"`
	Status           DebtStatus `json:"status"`
	Note             string     `json:"note"`
}

type payRequest struct {
	AmountMinor int64  `json:"amount_minor" binding:"required"`
	PaidAt      string `json:"paid_at"`
	Note        string `json:"note"`
}

func (h *Handler) Create(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	input, ok := bindCreate(c)
	if !ok {
		return
	}
	created, err := h.usecase.Create(c.Request.Context(), userID, input)
	if err != nil {
		writeError(c, err, "debt not found")
		return
	}
	httpx.JSON(c, http.StatusCreated, created)
}

func (h *Handler) List(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	pagination, ok := httpx.ParsePage(c, "opened_at_desc", debtSorts)
	if !ok {
		return
	}
	result, err := h.usecase.List(c.Request.Context(), userID, pagination)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "list debts failed")
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"debts": result.Items, "pagination": result.Pagination})
}

var debtSorts = map[string]struct{}{
	"opened_at_desc": {},
	"opened_at_asc":  {},
	"due_date_asc":   {},
	"due_date_desc":  {},
	"amount_desc":    {},
	"amount_asc":     {},
}

func (h *Handler) Get(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	debt, err := h.usecase.Get(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		writeError(c, err, "debt not found")
		return
	}
	httpx.JSON(c, http.StatusOK, debt)
}

func (h *Handler) Update(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	update, ok := bindUpdate(c)
	if !ok {
		return
	}
	updated, err := h.usecase.Update(c.Request.Context(), userID, c.Param("id"), update)
	if err != nil {
		writeError(c, err, "debt not found")
		return
	}
	httpx.JSON(c, http.StatusOK, updated)
}

func (h *Handler) Delete(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	if err := h.usecase.Delete(c.Request.Context(), userID, c.Param("id")); err != nil {
		writeError(c, err, "debt not found")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Pay(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	amountMinor, paidAt, note, ok := bindPay(c)
	if !ok {
		return
	}
	payment, err := h.usecase.Pay(c.Request.Context(), userID, c.Param("id"), amountMinor, paidAt, note)
	if err != nil {
		writeError(c, err, "debt not found")
		return
	}
	httpx.JSON(c, http.StatusCreated, payment)
}

func bindCreate(c *gin.Context) (DebtInput, bool) {
	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return DebtInput{}, false
	}
	if !IsValidDebtType(req.Type) {
		httpx.Error(c, http.StatusBadRequest, "invalid debt type")
		return DebtInput{}, false
	}
	if req.PrincipalAmountMinor <= 0 {
		httpx.Error(c, http.StatusBadRequest, "principal_amount_minor must be positive")
		return DebtInput{}, false
	}
	openedAt := time.Now().UTC()
	if req.OpenedAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.OpenedAt)
		if err != nil {
			httpx.Error(c, http.StatusBadRequest, "opened_at must be RFC3339")
			return DebtInput{}, false
		}
		openedAt = parsed.UTC()
	}
	dueDate, ok := parseOptionalDate(c, req.DueDate, "due_date")
	if !ok {
		return DebtInput{}, false
	}
	return DebtInput{
		Type:                   req.Type,
		CounterpartyName:       req.CounterpartyName,
		WalletID:               req.WalletID,
		DisbursementCategoryID: req.DisbursementCategoryID,
		PaymentCategoryID:      req.PaymentCategoryID,
		PrincipalAmountMinor:   req.PrincipalAmountMinor,
		OpenedAt:               openedAt,
		DueDate:                dueDate,
		Note:                   req.Note,
	}, true
}

func bindUpdate(c *gin.Context) (DebtUpdate, bool) {
	var req updateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return DebtUpdate{}, false
	}
	if req.Status != "" && !IsValidDebtStatus(req.Status) {
		httpx.Error(c, http.StatusBadRequest, "invalid debt status")
		return DebtUpdate{}, false
	}
	dueDate, ok := parseOptionalDate(c, req.DueDate, "due_date")
	if !ok {
		return DebtUpdate{}, false
	}
	return DebtUpdate{
		CounterpartyName: req.CounterpartyName,
		DueDate:          dueDate,
		Status:           req.Status,
		Note:             req.Note,
	}, true
}

func bindPay(c *gin.Context) (int64, time.Time, string, bool) {
	var req payRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return 0, time.Time{}, "", false
	}
	if req.AmountMinor <= 0 {
		httpx.Error(c, http.StatusBadRequest, "amount_minor must be positive")
		return 0, time.Time{}, "", false
	}
	paidAt := time.Now().UTC()
	if req.PaidAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.PaidAt)
		if err != nil {
			httpx.Error(c, http.StatusBadRequest, "paid_at must be RFC3339")
			return 0, time.Time{}, "", false
		}
		paidAt = parsed.UTC()
	}
	return req.AmountMinor, paidAt, req.Note, true
}

func parseOptionalDate(c *gin.Context, value string, field string) (*time.Time, bool) {
	if value == "" {
		return nil, true
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, field+" must use YYYY-MM-DD")
		return nil, false
	}
	date := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
	return &date, true
}

func writeError(c *gin.Context, err error, notFoundMessage string) {
	if NotFound(err) {
		httpx.Error(c, http.StatusNotFound, notFoundMessage)
		return
	}
	if errors.Is(err, errInvalidPayment) ||
		errors.Is(err, errOverpayment) ||
		errors.Is(err, errInactiveDebt) ||
		errors.Is(err, errInvalidDebtType) ||
		errors.Is(err, errInvalidDebtStatus) ||
		errors.Is(err, errInvalidDebtAction) {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.Error(c, http.StatusBadRequest, err.Error())
}
