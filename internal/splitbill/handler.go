package splitbill

import (
	"context"
	"errors"
	"net/http"
	"time"

	"affluena-api/internal/httpx"
	"affluena-api/internal/page"

	"github.com/gin-gonic/gin"
)

type splitBillUseCase interface {
	SplitExpense(ctx context.Context, userID string, input SplitTransactionInput) (SplitTransactionResponse, error)
	List(ctx context.Context, userID string, status string, pagination page.Params) (page.Result[SplitBillSummary], error)
	Get(ctx context.Context, userID string, transactionID string) (SplitBillDetail, error)
}

var splitBillSorts = map[string]struct{}{
	"transaction_at_desc": {},
	"transaction_at_asc":  {},
}

type Handler struct {
	usecase splitBillUseCase
}

func NewHandler(usecase splitBillUseCase) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) Split(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	var req SplitTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request payload")
		return
	}

	var transactionAt time.Time
	var err error
	if req.TransactionAt != "" {
		transactionAt, err = time.Parse(time.RFC3339, req.TransactionAt)
		if err != nil {
			httpx.Error(c, http.StatusBadRequest, "invalid transaction_at format")
			return
		}
	} else {
		transactionAt = time.Now().UTC()
	}

	input := SplitTransactionInput{
		WalletID:         req.WalletID,
		CategoryID:       req.CategoryID,
		TotalAmountMinor: req.TotalAmountMinor,
		TransactionAt:    transactionAt,
		Note:             req.Note,
		TagIDs:           req.TagIDs,
		Splits:           req.Splits,
	}

	resp, err := h.usecase.SplitExpense(c.Request.Context(), userID, input)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.JSON(c, http.StatusCreated, resp)
}

// List returns the user's split bills. Optional ?status=ongoing|settled.
func (h *Handler) List(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	status := c.Query("status")
	if status != "" && status != SplitBillStatusOngoing && status != SplitBillStatusSettled {
		httpx.Error(c, http.StatusBadRequest, "status must be ongoing or settled")
		return
	}

	pagination, ok := httpx.ParsePage(c, "transaction_at_desc", splitBillSorts)
	if !ok {
		return
	}

	result, err := h.usecase.List(c.Request.Context(), userID, status, pagination)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "list split bills failed")
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"split_bills": result.Items, "pagination": result.Pagination})
}

// Get returns a single split bill by its origination transaction id.
func (h *Handler) Get(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	id, ok := httpx.GetUUIDParam(c, "id")
	if !ok {
		return
	}

	detail, err := h.usecase.Get(c.Request.Context(), userID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, "split bill not found")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, "get split bill failed")
		return
	}
	httpx.JSON(c, http.StatusOK, detail)
}
