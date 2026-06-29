package handlers

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
	"yuem-go/backend-gin/internal/services"
)

const (
	msgWithdrawInternal          = "\u670d\u52a1\u5668\u5185\u90e8\u9519\u8bef"
	msgWithdrawSavePaymentOK     = "\u6536\u6b3e\u7801\u4fdd\u5b58\u6210\u529f"
	msgWithdrawPaymentRequired   = "\u8bf7\u81f3\u5c11\u63d0\u4f9b\u4e00\u4e2a\u6536\u6b3e\u7801URL"
	msgWithdrawTypeInvalid       = "\u63d0\u73b0\u7c7b\u578b\u65e0\u6548\uff0c\u8bf7\u9009\u62e9 cash\uff08\u73b0\u91d1\uff09\u6216 moon_coin\uff08\u6708\u5e01\uff09"
	msgWithdrawAmountInvalid     = "\u8bf7\u8f93\u5165\u6709\u6548\u7684\u63d0\u73b0\u91d1\u989d"
	msgWithdrawMissingPayCode    = "\u8bf7\u5148\u4e0a\u4f20\u5fae\u4fe1\u6216\u652f\u4ed8\u5b9d\u6536\u6b3e\u7801"
	msgWithdrawPendingExists     = "\u60a8\u6709\u4e00\u7b14\u63d0\u73b0\u7533\u8bf7\u6b63\u5728\u5ba1\u6838\u4e2d\uff0c\u8bf7\u7b49\u5f85\u5ba1\u6838\u5b8c\u6210\u540e\u518d\u7533\u8bf7"
	msgWithdrawApplyOK           = "\u63d0\u73b0\u7533\u8bf7\u5df2\u63d0\u4ea4\uff0c\u8bf7\u7b49\u5f85\u5ba1\u6838"
	msgWithdrawOrderNotFound     = "\u63d0\u73b0\u8ba2\u5355\u4e0d\u5b58\u5728"
	msgWithdrawApproveOK         = "\u5ba1\u6838\u5df2\u901a\u8fc7"
	msgWithdrawRejectOK          = "\u5df2\u9a73\u56de\u63d0\u73b0\u7533\u8bf7\u5e76\u9000\u8fd8\u4f59\u989d"
	msgWithdrawPayoutOK          = "\u5df2\u6807\u8bb0\u4e3a\u6253\u6b3e\u5b8c\u6210"
	msgWithdrawDefaultRejectNote = "\u5ba1\u6838\u672a\u901a\u8fc7"
	msgWithdrawDefaultPaidNote   = "\u5df2\u5b8c\u6210\u6253\u6b3e"
)

func (h NativeHandlers) WithdrawWallet(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgWithdrawInternal, nil)
		return
	}
	wallet, err := repositories.NewWithdrawRepository(h.DB).GetOrCreateWallet(c.Request.Context(), user.ID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgWithdrawInternal, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": walletResponse(wallet), "message": "success"})
}

func (h NativeHandlers) WithdrawPaymentCode(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgWithdrawInternal, nil)
		return
	}
	code, err := repositories.NewWithdrawRepository(h.DB).PaymentCode(c.Request.Context(), user.ID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgWithdrawInternal, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": paymentCodeResponse(code), "message": "success"})
}

func (h NativeHandlers) WithdrawSavePaymentCode(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	var body map[string]any
	_ = c.ShouldBindJSON(&body)
	wechat, wechatOK := stringFromAny(body["wechat_url"])
	alipay, alipayOK := stringFromAny(body["alipay_url"])
	if (!wechatOK || strings.TrimSpace(wechat) == "") && (!alipayOK || strings.TrimSpace(alipay) == "") {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgWithdrawPaymentRequired, nil)
		return
	}
	updates := map[string]any{}
	if _, exists := body["wechat_url"]; exists {
		updates["wechat_url"] = nullableTrimmedString(wechat)
	}
	if _, exists := body["alipay_url"]; exists {
		updates["alipay_url"] = nullableTrimmedString(alipay)
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgWithdrawInternal, nil)
		return
	}
	code, err := repositories.NewWithdrawRepository(h.DB).SavePaymentCode(c.Request.Context(), user.ID, updates)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgWithdrawInternal, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": paymentCodeResponse(code), "message": msgWithdrawSavePaymentOK})
}

type withdrawApplyRequest struct {
	Amount any    `json:"amount"`
	Type   string `json:"type"`
}

func (h NativeHandlers) WithdrawApply(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	var body withdrawApplyRequest
	_ = c.ShouldBindJSON(&body)
	if body.Type != "cash" && body.Type != "moon_coin" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgWithdrawTypeInvalid, nil)
		return
	}
	amount, ok := float64FromAny(body.Amount)
	if !ok || amount <= 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgWithdrawAmountInvalid, nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgWithdrawInternal, nil)
		return
	}
	if body.Type == "moon_coin" {
		h.withdrawToRemoteMoonCoin(c, user.ID, amount)
		return
	}
	order, err := repositories.NewWithdrawRepository(h.DB).Apply(c.Request.Context(), user.ID, amount, body.Type)
	switch {
	case errors.Is(err, repositories.ErrWithdrawMissingPayment):
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgWithdrawMissingPayCode, nil)
	case errors.Is(err, repositories.ErrWithdrawPendingExists):
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgWithdrawPendingExists, nil)
	case err != nil:
		var balanceErr repositories.InsufficientBalanceError
		if errors.As(err, &balanceErr) {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, fmt.Sprintf("\u53ef\u63d0\u73b0\u4f59\u989d\u4e0d\u8db3\uff0c\u5f53\u524d\u4f59\u989d: %.2f", balanceErr.Balance), nil)
			return
		}
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
	default:
		c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{
			"orderId": expressBigInt(order.ID),
			"amount":  amount,
			"type":    body.Type,
			"status":  "pending",
		}, "message": msgWithdrawApplyOK})
	}
}

func (h NativeHandlers) withdrawToRemoteMoonCoin(c *gin.Context, userID int64, amount float64) {
	if h.Balance == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgWithdrawInternal, nil)
		return
	}
	repo := repositories.NewWithdrawRepository(h.DB)
	order, err := repo.PrepareMoonCoinTransfer(c.Request.Context(), userID, amount)
	if err != nil {
		var balanceErr repositories.InsufficientBalanceError
		if errors.As(err, &balanceErr) {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.creator_balance_insufficient", gin.H{"balance": balanceErr.Balance})
			return
		}
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgWithdrawInternal, nil)
		return
	}
	_, oauth2ID, err := h.Balance.UserByInternalID(c.Request.Context(), userID)
	if err != nil {
		_ = repo.FailMoonCoinTransfer(c.Request.Context(), order.ID, err.Error())
		h.writeBalanceCenterError(c, err, msgBalanceGetFailed)
		return
	}
	mutation, err := h.Balance.ChangeBalance(c.Request.Context(), services.BalanceMutationInput{
		OperationKey: "creator_earnings:" + strconv.FormatInt(order.ID, 10) + ":credit",
		UserID:       userID,
		OAuth2ID:     oauth2ID,
		Amount:       amount,
		Reason:       "creator earnings transfer order=" + strconv.FormatInt(order.ID, 10),
		LocalCommit: func(tx *gorm.DB, _ *domain.ExternalBalanceTransaction) error {
			return repositories.NewWithdrawRepository(tx).FinalizeMoonCoinTransferTx(c.Request.Context(), tx, order.ID)
		},
	})
	if err != nil {
		if !errors.Is(err, services.ErrBalanceOperationUnknown) {
			_ = repo.FailMoonCoinTransfer(c.Request.Context(), order.ID, err.Error())
		}
		h.writeBalanceCenterError(c, err, msgWithdrawInternal)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{
		"orderId":      expressBigInt(order.ID),
		"amount":       amount,
		"type":         "moon_coin",
		"status":       "paid",
		"balanceAfter": mutation.BalanceAfter,
	}, "message": "withdraw.moon_coin_completed"})
}

func (h NativeHandlers) WithdrawOrders(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgWithdrawInternal, nil)
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := min(positiveIntQuery(c, "limit", 20), 100)
	total, orders, err := repositories.NewWithdrawRepository(h.DB).UserOrders(c.Request.Context(), user.ID, c.Query("status"), page, limit)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgWithdrawInternal, nil)
		return
	}
	list := make([]gin.H, 0, len(orders))
	for _, order := range orders {
		list = append(list, withdrawOrderResponse(order))
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{"list": list, "pagination": paginationTotalPages(page, limit, total)}, "message": "success"})
}

func (h NativeHandlers) WithdrawAdminOrders(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgWithdrawInternal, nil)
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := min(positiveIntQuery(c, "limit", 20), 100)
	total, orders, err := repositories.NewWithdrawRepository(h.DB).AdminOrders(c.Request.Context(), c.Query("status"), c.Query("keyword"), page, limit)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgWithdrawInternal, nil)
		return
	}
	list := make([]gin.H, 0, len(orders))
	for _, order := range orders {
		list = append(list, h.withdrawAdminOrderResponse(order))
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{"list": list, "pagination": paginationTotalPages(page, limit, total)}, "message": "success"})
}

type withdrawAdminActionRequest struct {
	Remark *string `json:"remark"`
}

func (h NativeHandlers) WithdrawAdminApprove(c *gin.Context) {
	h.withdrawAdminUpdate(c, "approve")
}

func (h NativeHandlers) WithdrawAdminReject(c *gin.Context) {
	h.withdrawAdminUpdate(c, "reject")
}

func (h NativeHandlers) WithdrawAdminPayout(c *gin.Context) {
	h.withdrawAdminUpdate(c, "payout")
}

func (h NativeHandlers) withdrawAdminUpdate(c *gin.Context, action string) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgWithdrawInternal, nil)
		return
	}
	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgWithdrawOrderNotFound, nil)
		return
	}
	var body withdrawAdminActionRequest
	_ = c.ShouldBindJSON(&body)
	repo := repositories.NewWithdrawRepository(h.DB)
	switch action {
	case "approve":
		order, err := repo.Approve(c.Request.Context(), orderID, body.Remark)
		if h.writeWithdrawAdminActionError(c, err, "approve") {
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{"id": expressBigInt(order.ID), "status": order.Status}, "message": msgWithdrawApproveOK})
	case "reject":
		if err := repo.Reject(c.Request.Context(), orderID, body.Remark); h.writeWithdrawAdminActionError(c, err, "reject") {
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{"id": expressBigInt(orderID), "status": "rejected"}, "message": msgWithdrawRejectOK})
	case "payout":
		order, err := repo.Payout(c.Request.Context(), orderID, body.Remark)
		if h.writeWithdrawAdminActionError(c, err, "payout") {
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{"id": expressBigInt(order.ID), "status": order.Status}, "message": msgWithdrawPayoutOK})
	}
}

func (h NativeHandlers) writeWithdrawAdminActionError(c *gin.Context, err error, action string) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgWithdrawOrderNotFound, nil)
		return true
	}
	if errors.Is(err, repositories.ErrWithdrawInvalidStatus) {
		message := "\u8ba2\u5355\u5f53\u524d\u72b6\u6001\u65e0\u6cd5\u64cd\u4f5c"
		switch action {
		case "approve":
			message = "\u8ba2\u5355\u5f53\u524d\u72b6\u6001\u65e0\u6cd5\u5ba1\u6838"
		case "reject":
			message = "\u8ba2\u5355\u5f53\u524d\u72b6\u6001\u65e0\u6cd5\u9a73\u56de"
		case "payout":
			message = "\u53ea\u6709\u5df2\u901a\u8fc7\u7684\u8ba2\u5355\u624d\u80fd\u6807\u8bb0\u4e3a\u5df2\u6253\u6b3e"
		}
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, message, nil)
		return true
	}
	response.JSON(c, http.StatusInternalServerError, response.CodeError, msgWithdrawInternal, nil)
	return true
}

func walletResponse(wallet *domain.UserWallet) gin.H {
	return gin.H{
		"cash_balance":  wallet.CashBalance,
		"total_income":  wallet.TotalIncome,
		"frozen_amount": wallet.FrozenAmount,
	}
}

func paymentCodeResponse(code *domain.UserPaymentCode) gin.H {
	if code == nil {
		return gin.H{"wechat_url": nil, "alipay_url": nil}
	}
	return gin.H{"wechat_url": code.WechatURL, "alipay_url": code.AlipayURL}
}

func withdrawOrderResponse(order domain.WithdrawOrder) gin.H {
	return gin.H{
		"id":         expressBigInt(order.ID),
		"amount":     order.Amount,
		"type":       order.Type,
		"status":     order.Status,
		"remark":     order.Remark,
		"created_at": order.CreatedAt,
		"updated_at": order.UpdatedAt,
	}
}

func (h NativeHandlers) withdrawAdminOrderResponse(bundle repositories.WithdrawAdminOrder) gin.H {
	var userID any
	var userUID any
	var nickname any
	var avatar any
	if bundle.User != nil {
		userID = expressBigInt(bundle.User.ID)
		userUID = bundle.User.UserID
		nickname = bundle.User.Nickname
		avatar = h.signFileURLPtr(bundle.User.Avatar)
	}
	var wechat any
	var alipay any
	if bundle.PayCode != nil {
		wechat = h.signFileURLPtr(bundle.PayCode.WechatURL)
		alipay = h.signFileURLPtr(bundle.PayCode.AlipayURL)
	}
	order := bundle.Order
	return gin.H{
		"id":         expressBigInt(order.ID),
		"user_id":    userID,
		"user_uid":   userUID,
		"nickname":   nickname,
		"avatar":     avatar,
		"amount":     order.Amount,
		"type":       order.Type,
		"status":     order.Status,
		"remark":     order.Remark,
		"wechat_url": wechat,
		"alipay_url": alipay,
		"created_at": order.CreatedAt,
		"updated_at": order.UpdatedAt,
	}
}

func nullableTrimmedString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func expressBigInt(value int64) any {
	const maxSafeInteger = int64(9007199254740991)
	if value <= maxSafeInteger && value >= -maxSafeInteger {
		return value
	}
	return strconv.FormatInt(value, 10)
}

func totalPages(page, limit int, total int64) int {
	if limit <= 0 {
		return 0
	}
	return int(math.Ceil(float64(total) / float64(limit)))
}
