package handlers

import (
	"errors"
	"maps"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
	"yuem-go/backend-gin/internal/services"
)

const (
	msgBalanceInternal       = "\u670d\u52a1\u5668\u5185\u90e8\u9519\u8bef"
	msgBalanceNotConfigured  = "error.balance_center_not_configured"
	msgBalanceUserMissing    = "\u7528\u6237\u4e0d\u5b58\u5728"
	msgBalanceOAuthMissing   = "\u7528\u6237\u672a\u7ed1\u5b9aOAuth2\u8d26\u53f7\uff0c\u65e0\u6cd5\u4f7f\u7528\u94b1\u5305"
	msgBalanceGetFailed      = "\u83b7\u53d6\u6708\u5e01\u4f59\u989d\u5931\u8d25"
	msgPurchasePostIDMissing = "\u7f3a\u5c11\u5e16\u5b50ID"
	msgPurchasePostMissing   = "\u5e16\u5b50\u4e0d\u5b58\u5728"
	msgPurchaseOwnContent    = "\u4e0d\u80fd\u8d2d\u4e70\u81ea\u5df1\u7684\u5185\u5bb9"
	msgPurchaseNotPaid       = "\u8be5\u5185\u5bb9\u4e0d\u662f\u4ed8\u8d39\u5185\u5bb9"
	msgPurchaseOAuthMissing  = "\u7528\u6237\u672a\u7ed1\u5b9aOAuth2\u8d26\u53f7\uff0c\u65e0\u6cd5\u4f7f\u7528\u6708\u5e01\u652f\u4ed8"
	msgPurchaseOK            = "\u8d2d\u4e70\u6210\u529f\uff01"
	msgAlreadyPurchased      = "\u60a8\u5df2\u7ecf\u8d2d\u4e70\u8fc7\u6b64\u5185\u5bb9"
	msgCreatorInternal       = "\u670d\u52a1\u5668\u5185\u90e8\u9519\u8bef"
)

type balanceCenterResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

func (h NativeHandlers) BalanceConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    response.CodeSuccess,
		"data":    gin.H{"enabled": balanceCenterConfigured(h.Config.Balance)},
		"message": "success",
	})
}

func (h NativeHandlers) BalanceRechargeConfig(c *gin.Context) {
	if !balanceCenterConfigured(h.Config.Balance) {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgBalanceNotConfigured, nil)
		return
	}
	result, err := h.balanceCenterGet(c.Request.Context(), "/api/recharge-settings", false)
	if err != nil || !result.Success {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "\u83b7\u53d6\u5145\u503c\u914d\u7f6e\u5931\u8d25", nil)
		return
	}
	data := gin.H{}
	maps.Copy(data, result.Data)
	data["recharge_url"] = strings.TrimRight(h.Config.Balance.APIURL, "/") + "/recharge"
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": data, "message": "success"})
}

func (h NativeHandlers) BalanceLocalPoints(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgBalanceInternal, nil)
		return
	}
	points, err := repositories.NewBalanceRepository(h.DB).GetOrCreateUserPoints(c.Request.Context(), user.ID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgBalanceInternal, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{"points": points.Points}, "message": "success"})
}

func (h NativeHandlers) BalanceUserBalance(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgBalanceInternal, nil)
		return
	}
	wallet, err := repositories.NewWithdrawRepository(h.DB).GetOrCreateWallet(c.Request.Context(), user.ID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgBalanceInternal, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": response.CodeSuccess,
		"data": gin.H{
			"balance":       wallet.CashBalance,
			"cash_balance":  wallet.CashBalance,
			"vip_level":     0,
			"vip_expire_at": nil,
			"username":      nil,
			"is_active":     true,
		},
		"message": "success",
	})
}

type purchaseContentRequest struct {
	PostID        any    `json:"postId"`
	PaymentMethod string `json:"paymentMethod"`
	UserCouponID  any    `json:"user_coupon_id"`
}

func (h NativeHandlers) BalancePurchaseContent(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	var body purchaseContentRequest
	_ = c.ShouldBindJSON(&body)
	postID, ok := int64FromAny(body.PostID)
	if !ok || postID <= 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgPurchasePostIDMissing, nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgBalanceInternal, nil)
		return
	}

	repo := repositories.NewBalanceRepository(h.DB)
	probe, err := repo.PurchaseQuote(c.Request.Context(), repositories.PurchaseContentInput{
		UserID:        user.ID,
		PostID:        postID,
		PaymentMethod: body.PaymentMethod,
	})
	if h.writePurchaseError(c, err, probe) {
		return
	}
	actualMethod := normalizePaymentMethodForResponse(probe.PaymentSetting.PaymentMethod)
	if probe.Already {
		c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{
			"orderId":          probe.PurchaseID,
			"status":           "completed",
			"alreadyPurchased": true,
			"paymentMethod":    actualMethod,
			"messageKey":       "purchase.already_purchased",
		}, "message": msgAlreadyPurchased})
		return
	}
	if !h.paidContentPaymentMethodEnabled(actualMethod) {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.payment_method_disabled", gin.H{"paymentMethod": actualMethod})
		return
	}
	if actualMethod == "points" {
		reservation, err := repo.ReservePurchaseIntent(c.Request.Context(), repositories.PurchaseContentInput{
			UserID:        user.ID,
			PostID:        postID,
			PaymentMethod: "points",
		}, probe.Price, probe.PaidAmount)
		if h.writePurchaseError(c, err, probe) {
			return
		}
		if reservation.Completed {
			c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{
				"orderId":          reservation.PurchaseID,
				"status":           "completed",
				"alreadyPurchased": true,
				"paymentMethod":    "points",
				"messageKey":       "purchase.already_purchased",
			}, "message": msgAlreadyPurchased})
			return
		}
		result, err := repo.PurchaseContent(c.Request.Context(), repositories.PurchaseContentInput{
			UserID:        user.ID,
			PostID:        postID,
			PaymentMethod: body.PaymentMethod,
		})
		if h.writePurchaseError(c, err, result) {
			_ = repo.FailPurchaseIntent(c.Request.Context(), reservation.Intent.ID, purchaseIntentErrorCode(err))
			return
		}
		_ = repo.CompletePurchaseIntent(c.Request.Context(), reservation.Intent.ID)
		h.bumpCacheVersions(cacheScopePosts, cacheScopeSearch, cacheScopeInteractions)
		c.JSON(http.StatusOK, gin.H{
			"code": response.CodeSuccess,
			"data": gin.H{
				"orderId":          result.PurchaseID,
				"status":           "completed",
				"alreadyPurchased": result.Already,
				"messageKey":       "purchase.completed",
				"postId":           strconv.FormatInt(postID, 10),
				"price":            result.Price,
				"paidAmount":       result.PaidAmount,
				"discountRate":     result.DiscountRate,
				"couponDiscount":   result.CouponDiscount,
				"paymentMethod":    "points",
				"authorEarnings":   result.AuthorEarnings,
				"platformFee":      result.PlatformFee,
			},
			"message": msgPurchaseOK,
		})
		return
	}

	userCouponID := optionalInt64FromAny(body.UserCouponID)
	wallet, err := repositories.NewWithdrawRepository(h.DB).GetOrCreateWallet(c.Request.Context(), user.ID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgBalanceInternal, nil)
		return
	}
	quote, err := repo.PurchaseQuote(c.Request.Context(), repositories.PurchaseContentInput{
		UserID:        user.ID,
		PostID:        postID,
		PaymentMethod: "balance",
		VIPLevel:      0,
		UserCouponID:  userCouponID,
	})
	if h.writePurchaseError(c, err, quote) {
		return
	}
	if quote.Already {
		c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{
			"orderId":          quote.PurchaseID,
			"status":           "completed",
			"alreadyPurchased": true,
			"paymentMethod":    "balance",
			"messageKey":       "purchase.already_purchased",
		}, "message": msgAlreadyPurchased})
		return
	}
	if quote.PaidAmount > wallet.CashBalance {
		h.writePurchaseError(c, repositories.ErrPurchaseInsufficient, quote)
		return
	}
	reservation, err := repo.ReservePurchaseIntent(c.Request.Context(), repositories.PurchaseContentInput{
		UserID:        user.ID,
		PostID:        postID,
		PaymentMethod: "balance",
	}, quote.Price, quote.PaidAmount)
	if h.writePurchaseError(c, err, quote) {
		return
	}
	if reservation.Completed {
		c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{
			"orderId":          reservation.PurchaseID,
			"status":           "completed",
			"alreadyPurchased": true,
			"paymentMethod":    "balance",
			"messageKey":       "purchase.already_purchased",
		}, "message": msgAlreadyPurchased})
		return
	}

	var result *repositories.PurchaseContentResult
	err = h.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		var commitErr error
		result, commitErr = repositories.NewBalanceRepository(tx).PurchaseContent(c.Request.Context(), repositories.PurchaseContentInput{
			UserID:             user.ID,
			PostID:             postID,
			PaymentMethod:      "balance",
			VIPLevel:           0,
			UserCouponID:       userCouponID,
			PlatformFeeRate:    h.Config.Creator.PlatformFeeRate,
			UseInternalBalance: true,
		})
		if commitErr != nil {
			return commitErr
		}
		return repositories.NewBalanceRepository(tx).CompletePurchaseIntent(c.Request.Context(), reservation.Intent.ID)
	})
	if h.writePurchaseError(c, err, result) {
		_ = repo.FailPurchaseIntent(c.Request.Context(), reservation.Intent.ID, purchaseIntentErrorCode(err))
		return
	}
	h.bumpCacheVersions(cacheScopePosts, cacheScopeSearch, cacheScopeInteractions)
	c.JSON(http.StatusOK, gin.H{
		"code": response.CodeSuccess,
		"data": gin.H{
			"orderId":          result.PurchaseID,
			"status":           "completed",
			"alreadyPurchased": result.Already,
			"messageKey":       "purchase.completed",
			"postId":           strconv.FormatInt(postID, 10),
			"price":            result.Price,
			"paidAmount":       result.PaidAmount,
			"discountRate":     result.DiscountRate,
			"couponDiscount":   result.CouponDiscount,
			"paymentMethod":    "balance",
			"authorEarnings":   result.AuthorEarnings,
			"platformFee":      result.PlatformFee,
			"balanceAfter":     result.BalanceAfter,
		},
		"message": msgPurchaseOK,
	})
}

func (h NativeHandlers) writeBalanceCenterError(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, services.ErrBalanceOAuth2Missing):
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgBalanceOAuthMissing, nil)
	case errors.Is(err, services.ErrBalanceOperationBusy):
		response.JSON(c, http.StatusConflict, response.CodeConflict, "error.balance_operation_in_progress", nil)
	case errors.Is(err, services.ErrBalanceOperationUnknown):
		response.JSON(c, http.StatusConflict, response.CodeConflict, "error.balance_operation_unknown", nil)
	case errors.Is(err, services.ErrBalanceRemoteRejected):
		response.JSON(c, http.StatusPaymentRequired, response.CodeValidationError, "error.balance_rejected", nil)
	default:
		response.JSON(c, http.StatusBadGateway, response.CodeError, fallback, nil)
	}
}

func balanceCenterConfigured(cfg config.BalanceCenterConfig) bool {
	return strings.TrimSpace(cfg.APIURL) != "" && strings.TrimSpace(cfg.APIKey) != ""
}

func (h NativeHandlers) BalanceOrders(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := positiveIntQuery(c, "limit", 20)
	total, orders, err := repositories.NewBalanceRepository(h.DB).Orders(c.Request.Context(), user.ID, page, limit)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgBalanceInternal, nil)
		return
	}
	list := make([]gin.H, 0, len(orders))
	for _, order := range orders {
		list = append(list, h.purchaseOrderResponse(order))
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{"list": list, "pagination": paginationTotalPages(page, limit, total)}, "message": "success"})
}

func (h NativeHandlers) BalanceCheckPurchase(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	postID, err := strconv.ParseInt(c.Param("postId"), 10, 64)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgBalanceInternal, nil)
		return
	}
	purchase, err := repositories.NewBalanceRepository(h.DB).CheckPurchase(c.Request.Context(), user.ID, postID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgBalanceInternal, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{"hasPurchased": purchase != nil, "purchasedAt": purchasedAtValue(purchase)}, "message": "success"})
}
