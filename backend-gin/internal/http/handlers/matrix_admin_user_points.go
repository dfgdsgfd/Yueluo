package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
)

type adminUserPointsRequest struct {
	Operation string  `json:"operation"`
	Amount    float64 `json:"amount"`
	Reason    string  `json:"reason"`
}

func (h NativeHandlers) AdminUserPointsUpdate(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	userID, ok := pathInt64(c, "id")
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.user_invalid_id", nil)
		return
	}

	var body adminUserPointsRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.invalid_request", nil)
		return
	}
	body.Operation = strings.ToLower(strings.TrimSpace(body.Operation))
	body.Reason = strings.TrimSpace(body.Reason)

	result, err := h.pointsRepo().AdminAdjustBalance(c.Request.Context(), userID, body.Operation, body.Amount, body.Reason)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			response.JSON(c, http.StatusNotFound, response.CodeNotFound, "error.user_not_found", nil)
		case errors.Is(err, repositories.ErrPointsInsufficient):
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.points_insufficient", nil)
		case errors.Is(err, repositories.ErrPointsAdjustmentInvalidOperation):
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.points_invalid_operation", nil)
		case errors.Is(err, repositories.ErrPointsAdjustmentInvalidAmount):
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.points_invalid_amount", nil)
		case errors.Is(err, repositories.ErrPointsAdjustmentNoChange):
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.points_no_change", nil)
		default:
			response.JSON(c, http.StatusInternalServerError, response.CodeError, "error.internal", nil)
		}
		return
	}

	actorID, actorDisplayID, actorType := requestUserActor(c)
	h.recordSecurityAudit(c, "admin", "user_points_adjustment", "success", "", http.StatusOK, actorID, actorType, actorDisplayID, gin.H{
		"target_uid":       result.UserID,
		"operation":        result.Operation,
		"previous_balance": result.PreviousBalance,
		"amount":           result.Amount,
		"balance_after":    result.BalanceAfter,
		"reason":           body.Reason,
	})
	writeSuccess(c, matrixMsgOK, gin.H{
		"uid":              result.UserID,
		"operation":        result.Operation,
		"previous_balance": result.PreviousBalance,
		"amount":           result.Amount,
		"balance_after":    result.BalanceAfter,
	})
}
