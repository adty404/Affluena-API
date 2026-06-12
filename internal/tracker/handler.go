package tracker

import (
	"errors"
	"net/http"
	"time"

	"affluena/internal/httpx"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	installments  *InstallmentRepository
	subscriptions *SubscriptionRepository
}

func NewHandler(installments *InstallmentRepository, subscriptions *SubscriptionRepository) *Handler {
	return &Handler{installments: installments, subscriptions: subscriptions}
}

type installmentRequest struct {
	Name               string            `json:"name" binding:"required"`
	WalletID           string            `json:"wallet_id" binding:"required"`
	CategoryID         string            `json:"category_id" binding:"required"`
	TotalAmountMinor   int64             `json:"total_amount_minor" binding:"required"`
	MonthlyAmountMinor int64             `json:"monthly_amount_minor" binding:"required"`
	TenorMonths        int               `json:"tenor_months" binding:"required"`
	RemainingMonths    *int              `json:"remaining_months"`
	StartDate          string            `json:"start_date" binding:"required"`
	DueDay             int               `json:"due_day" binding:"required"`
	Status             InstallmentStatus `json:"status"`
	Note               string            `json:"note"`
}

type subscriptionRequest struct {
	Name         string             `json:"name" binding:"required"`
	WalletID     string             `json:"wallet_id" binding:"required"`
	CategoryID   string             `json:"category_id" binding:"required"`
	AmountMinor  int64              `json:"amount_minor" binding:"required"`
	BillingCycle BillingCycle       `json:"billing_cycle" binding:"required"`
	NextDueDate  string             `json:"next_due_date" binding:"required"`
	Status       SubscriptionStatus `json:"status"`
	Note         string             `json:"note"`
}

type payRequest struct {
	PaidAt string `json:"paid_at"`
	Note   string `json:"note"`
}

func (h *Handler) CreateInstallment(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	installment, ok := bindInstallment(c)
	if !ok {
		return
	}

	created, err := h.installments.Create(c.Request.Context(), userID, installment)
	if err != nil {
		writeTrackerError(c, err, "installment not found")
		return
	}
	httpx.JSON(c, http.StatusCreated, created)
}

func (h *Handler) ListInstallments(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	installments, err := h.installments.List(c.Request.Context(), userID)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "list installments failed")
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"installments": installments})
}

func (h *Handler) GetInstallment(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	installment, err := h.installments.Get(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		writeTrackerError(c, err, "installment not found")
		return
	}
	httpx.JSON(c, http.StatusOK, installment)
}

func (h *Handler) UpdateInstallment(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	installment, ok := bindInstallment(c)
	if !ok {
		return
	}
	updated, err := h.installments.Update(c.Request.Context(), userID, c.Param("id"), installment)
	if err != nil {
		writeTrackerError(c, err, "installment not found")
		return
	}
	httpx.JSON(c, http.StatusOK, updated)
}

func (h *Handler) DeleteInstallment(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	if err := h.installments.Delete(c.Request.Context(), userID, c.Param("id")); err != nil {
		writeTrackerError(c, err, "installment not found")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) PayInstallment(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	paidAt, note, ok := bindPay(c)
	if !ok {
		return
	}
	payment, err := h.installments.Pay(c.Request.Context(), userID, c.Param("id"), paidAt, note)
	if err != nil {
		writeTrackerError(c, err, "installment not found")
		return
	}
	httpx.JSON(c, http.StatusCreated, payment)
}

func (h *Handler) CreateSubscription(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	subscription, ok := bindSubscription(c)
	if !ok {
		return
	}
	created, err := h.subscriptions.Create(c.Request.Context(), userID, subscription)
	if err != nil {
		writeTrackerError(c, err, "subscription not found")
		return
	}
	httpx.JSON(c, http.StatusCreated, created)
}

func (h *Handler) ListSubscriptions(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	subscriptions, err := h.subscriptions.List(c.Request.Context(), userID)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "list subscriptions failed")
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"subscriptions": subscriptions})
}

func (h *Handler) GetSubscription(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	subscription, err := h.subscriptions.Get(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		writeTrackerError(c, err, "subscription not found")
		return
	}
	httpx.JSON(c, http.StatusOK, subscription)
}

func (h *Handler) UpdateSubscription(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	subscription, ok := bindSubscription(c)
	if !ok {
		return
	}
	updated, err := h.subscriptions.Update(c.Request.Context(), userID, c.Param("id"), subscription)
	if err != nil {
		writeTrackerError(c, err, "subscription not found")
		return
	}
	httpx.JSON(c, http.StatusOK, updated)
}

func (h *Handler) DeleteSubscription(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	if err := h.subscriptions.Delete(c.Request.Context(), userID, c.Param("id")); err != nil {
		writeTrackerError(c, err, "subscription not found")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) PaySubscription(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	paidAt, note, ok := bindPay(c)
	if !ok {
		return
	}
	payment, err := h.subscriptions.Pay(c.Request.Context(), userID, c.Param("id"), paidAt, note)
	if err != nil {
		writeTrackerError(c, err, "subscription not found")
		return
	}
	httpx.JSON(c, http.StatusCreated, payment)
}

func bindInstallment(c *gin.Context) (Installment, bool) {
	var req installmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return Installment{}, false
	}
	if req.TotalAmountMinor <= 0 || req.MonthlyAmountMinor <= 0 || req.TenorMonths <= 0 {
		httpx.Error(c, http.StatusBadRequest, "amounts and tenor_months must be positive")
		return Installment{}, false
	}
	if req.DueDay < 1 || req.DueDay > 31 {
		httpx.Error(c, http.StatusBadRequest, "due_day must be between 1 and 31")
		return Installment{}, false
	}
	remainingMonths, status, err := ResolveInstallmentRemainingAndStatus(req.TenorMonths, req.RemainingMonths, req.Status)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return Installment{}, false
	}
	startDate, err := parseDate(req.StartDate)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "start_date must use YYYY-MM-DD")
		return Installment{}, false
	}

	return Installment{
		Name:               req.Name,
		WalletID:           req.WalletID,
		CategoryID:         req.CategoryID,
		TotalAmountMinor:   req.TotalAmountMinor,
		MonthlyAmountMinor: req.MonthlyAmountMinor,
		TenorMonths:        req.TenorMonths,
		RemainingMonths:    remainingMonths,
		StartDate:          startDate,
		DueDay:             req.DueDay,
		Status:             status,
		Note:               req.Note,
	}, true
}

func bindSubscription(c *gin.Context) (Subscription, bool) {
	var req subscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return Subscription{}, false
	}
	if req.AmountMinor <= 0 {
		httpx.Error(c, http.StatusBadRequest, "amount_minor must be positive")
		return Subscription{}, false
	}
	if !IsValidBillingCycle(req.BillingCycle) {
		httpx.Error(c, http.StatusBadRequest, "invalid billing cycle")
		return Subscription{}, false
	}
	if req.Status == "" {
		req.Status = SubscriptionStatusActive
	}
	if !IsValidSubscriptionStatus(req.Status) {
		httpx.Error(c, http.StatusBadRequest, "invalid subscription status")
		return Subscription{}, false
	}
	nextDueDate, err := parseDate(req.NextDueDate)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "next_due_date must use YYYY-MM-DD")
		return Subscription{}, false
	}

	return Subscription{
		Name:         req.Name,
		WalletID:     req.WalletID,
		CategoryID:   req.CategoryID,
		AmountMinor:  req.AmountMinor,
		BillingCycle: req.BillingCycle,
		NextDueDate:  nextDueDate,
		Status:       req.Status,
		Note:         req.Note,
	}, true
}

func bindPay(c *gin.Context) (time.Time, string, bool) {
	var req payRequest
	if !httpx.BindOptionalJSON(c, &req, "invalid request body") {
		return time.Time{}, "", false
	}
	paidAt := time.Now().UTC()
	if req.PaidAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.PaidAt)
		if err != nil {
			httpx.Error(c, http.StatusBadRequest, "paid_at must be RFC3339")
			return time.Time{}, "", false
		}
		paidAt = parsed.UTC()
	}
	return paidAt, req.Note, true
}

func parseDate(value string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC), nil
}

func writeTrackerError(c *gin.Context, err error, notFoundMessage string) {
	if NotFound(err) {
		httpx.Error(c, http.StatusNotFound, notFoundMessage)
		return
	}
	if errors.Is(err, errInactiveSubscription) {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.Error(c, http.StatusBadRequest, err.Error())
}
