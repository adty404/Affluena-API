package splitbill

import (
	"context"
	"net/http"
	"time"

	"affluena-api/internal/httpx"

	"github.com/gin-gonic/gin"
)

type splitBillUseCase interface {
	SplitExpense(ctx context.Context, userID string, input SplitTransactionInput) (SplitTransactionResponse, error)
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
