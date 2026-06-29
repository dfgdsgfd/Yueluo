package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
	"yuem-go/backend-gin/internal/services"
)

var balanceCenterHelperHTTPClient = &http.Client{Timeout: 30 * time.Second}

func (h NativeHandlers) balanceCenterGet(ctx context.Context, path string, apiKey bool) (*balanceCenterResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(h.Config.Balance.APIURL, "/")+path, nil)
	if err != nil {
		return nil, err
	}
	if apiKey {
		req.Header.Set("X-API-Key", h.Config.Balance.APIKey)
	}
	return doBalanceRequest(req)
}

func (h NativeHandlers) balanceCenterPost(ctx context.Context, path string, body any) (*balanceCenterResponse, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(h.Config.Balance.APIURL, "/")+path, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", h.Config.Balance.APIKey)
	return doBalanceRequest(req)
}

func doBalanceRequest(req *http.Request) (*balanceCenterResponse, error) {
	resp, err := balanceCenterHelperHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result balanceCenterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (h NativeHandlers) writePurchaseError(c *gin.Context, err error, result *repositories.PurchaseContentResult) bool {
	if err == nil {
		return false
	}
	needed := 0.0
	if result != nil {
		needed = result.PaidAmount
	}
	switch {
	case errors.Is(err, repositories.ErrPurchasePostMissing):
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgPurchasePostMissing, nil)
	case errors.Is(err, repositories.ErrPurchaseOwnContent):
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgPurchaseOwnContent, nil)
	case errors.Is(err, repositories.ErrPurchaseNotPaidContent):
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgPurchaseNotPaid, nil)
	case errors.Is(err, repositories.ErrPurchaseInsufficient):
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, fmt.Sprintf("\u6708\u5e01\u4e0d\u8db3\uff0c\u9700\u8981 %.2f", needed), nil)
	case errors.Is(err, repositories.ErrPointsInsufficient):
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, fmt.Sprintf("\u79ef\u5206\u4e0d\u8db3\uff0c\u9700\u8981 %.2f", needed), nil)
	case errors.Is(err, repositories.ErrPurchasePaymentMethod):
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.payment_method_mismatch", nil)
	case errors.Is(err, repositories.ErrPurchaseInProgress):
		response.JSON(c, http.StatusConflict, response.CodeValidationError, "error.purchase_in_progress", nil)
	case errors.Is(err, services.ErrBalanceOperationBusy):
		response.JSON(c, http.StatusConflict, response.CodeConflict, "error.balance_operation_in_progress", nil)
	case errors.Is(err, services.ErrBalanceOperationUnknown):
		response.JSON(c, http.StatusConflict, response.CodeConflict, "error.balance_operation_unknown", nil)
	case errors.Is(err, services.ErrBalanceRemoteRejected):
		response.JSON(c, http.StatusPaymentRequired, response.CodeValidationError, "error.balance_rejected", gin.H{"required": needed})
	case errors.Is(err, repositories.ErrCouponWrongOwner):
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "\u8be5\u4f18\u60e0\u5238\u4e0d\u5c5e\u4e8e\u60a8", nil)
	case errors.Is(err, repositories.ErrCouponUsed):
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "\u8be5\u4f18\u60e0\u5238\u5df2\u4f7f\u7528", nil)
	case errors.Is(err, repositories.ErrCouponExpired):
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "\u8be5\u4f18\u60e0\u5238\u5df2\u8fc7\u671f", nil)
	case errors.Is(err, repositories.ErrCouponInactive), errors.Is(err, repositories.ErrCouponNotStarted):
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "\u8be5\u4f18\u60e0\u5238\u4e0d\u53ef\u7528", nil)
	case errors.Is(err, repositories.ErrCouponMinOrder):
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "\u8ba2\u5355\u91d1\u989d\u672a\u6ee1\u8db3\u4f18\u60e0\u5238\u4f7f\u7528\u6761\u4ef6", nil)
	case errors.Is(err, gorm.ErrRecordNotFound):
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "\u4f18\u60e0\u5238\u4e0d\u5b58\u5728", nil)
	default:
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
	}
	return true
}

func purchaseIntentErrorCode(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, repositories.ErrPurchaseInsufficient):
		return "insufficient_balance"
	case errors.Is(err, repositories.ErrPointsInsufficient):
		return "insufficient_points"
	case errors.Is(err, repositories.ErrPurchasePaymentMethod):
		return "payment_method_mismatch"
	default:
		return "purchase_failed"
	}
}

func (h NativeHandlers) writeCreatorWithdrawError(c *gin.Context, err error, amount float64) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, repositories.ErrCreatorWithdrawClosed):
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "\u63d0\u73b0\u529f\u80fd\u6682\u672a\u5f00\u653e", nil)
	case errors.Is(err, repositories.ErrCreatorAmountInvalid):
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "\u8bf7\u8f93\u5165\u6709\u6548\u7684\u63d0\u73b0\u91d1\u989d", nil)
	case errors.Is(err, repositories.ErrCreatorBelowMinimum):
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, fmt.Sprintf("\u6700\u4f4e\u63d0\u73b0\u91d1\u989d\u4e3a %.0f \u6708\u5e01", h.Config.Creator.MinWithdrawAmount), nil)
	case errors.Is(err, repositories.ErrCreatorBalanceLow):
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "\u6536\u76ca\u4f59\u989d\u4e0d\u8db3", nil)
	default:
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
	}
	return true
}

func (h NativeHandlers) purchaseOrderResponse(order repositories.PurchaseOrderBundle) gin.H {
	postTitle := "\u5df2\u5220\u9664\u5185\u5bb9"
	var postID any
	if order.Post != nil {
		postTitle = order.Post.Title
		postID = strconv.FormatInt(order.Post.ID, 10)
	}
	return gin.H{
		"id":             strconv.FormatInt(order.Purchase.ID, 10),
		"price":          order.Purchase.Price,
		"paid_amount":    order.Purchase.PaidAmount,
		"discount_rate":  order.Purchase.DiscountRate,
		"purchase_type":  order.Purchase.PurchaseType,
		"payment_method": normalizePaymentMethodForResponse(order.Purchase.PaymentMethod),
		"purchased_at":   order.Purchase.PurchasedAt,
		"post_id":        postID,
		"post_title":     postTitle,
		"post_cover":     h.signFileURLPtr(order.Cover),
	}
}

func purchasedAtValue(purchase *domain.UserPurchasedContent) any {
	if purchase == nil {
		return nil
	}
	return purchase.PurchasedAt
}

func creatorRatesResponse(r config.CreatorEarningsRates) gin.H {
	return gin.H{"perView": r.PerView, "perLike": r.PerLike, "perCollect": r.PerCollect, "perComment": r.PerComment, "perFollower": r.PerFollower}
}

func (h NativeHandlers) creatorEarningsLogResponse(item repositories.CreatorEarningsLogBundle) gin.H {
	log := item.Log
	return gin.H{
		"id":            expressBigInt(log.ID),
		"amount":        log.Amount,
		"gross_amount":  log.Amount + log.PlatformFee,
		"balance_after": log.BalanceAfter,
		"type":          log.Type,
		"platform_fee":  log.PlatformFee,
		"reason":        log.Reason,
		"source":        creatorSourceResponse(item.Source),
		"buyer":         h.creatorBuyerResponse(item.Buyer),
		"created_at":    log.CreatedAt,
	}
}

func creatorSourceResponse(post *domain.Post) any {
	if post == nil {
		return nil
	}
	return gin.H{"id": expressBigInt(post.ID), "title": post.Title}
}

func (h NativeHandlers) creatorBuyerResponse(user *domain.User) any {
	if user == nil {
		return nil
	}
	return gin.H{"id": expressBigInt(user.ID), "nickname": user.Nickname, "avatar": h.signFileURLPtr(user.Avatar), "user_id": user.UserID}
}

func (h NativeHandlers) paidContentResponse(item repositories.PaidContentBundle) gin.H {
	price := 0.0
	if item.Payment != nil {
		price = item.Payment.Price
	}
	return gin.H{
		"id":            expressBigInt(item.Post.ID),
		"title":         item.Post.Title,
		"type":          item.Post.Type,
		"cover":         h.signFileURLPtr(item.Cover),
		"price":         price,
		"view_count":    item.Post.ViewCount,
		"like_count":    item.Post.LikeCount,
		"collect_count": item.Post.CollectCount,
		"sales_count":   item.SalesCount,
		"total_revenue": item.TotalRevenue,
		"created_at":    item.Post.CreatedAt,
	}
}

func extendedEarningsResponse(value repositories.ExtendedEarnings) gin.H {
	return gin.H{
		"enabled":   value.Enabled,
		"rates":     creatorRatesResponse(value.Rates),
		"dailyCap":  value.DailyCap,
		"views":     countEarningsResponse(value.Views),
		"likes":     countEarningsResponse(value.Likes),
		"collects":  countEarningsResponse(value.Collects),
		"comments":  countEarningsResponse(value.Comments),
		"followers": countEarningsResponse(value.Followers),
		"total":     value.Total,
	}
}

func countEarningsResponse(value repositories.CountEarnings) gin.H {
	return gin.H{"count": value.Count, "earnings": value.Earnings}
}

func (h NativeHandlers) qualityRewardResponse(item repositories.QualityRewardBundle) gin.H {
	log := item.Log
	return gin.H{"id": expressBigInt(log.ID), "amount": log.Amount, "reason": log.Reason, "post": h.qualityRewardPostResponse(item.Post), "created_at": log.CreatedAt}
}

func (h NativeHandlers) qualityRewardPostResponse(post *repositories.QualityRewardPost) any {
	if post == nil {
		return nil
	}
	return gin.H{"id": expressBigInt(post.ID), "title": post.Title, "type": post.Type, "quality_level": post.QualityLevel, "cover": h.signFileURLPtr(post.Cover), "created_at": post.CreatedAt}
}

func qualityRewardStatsResponse(stats []repositories.QualityRewardStats) []gin.H {
	out := make([]gin.H, 0, len(stats))
	for _, stat := range stats {
		out = append(out, gin.H{"quality_label": stat.QualityLabel, "count": stat.Count, "total_amount": stat.TotalAmount})
	}
	return out
}

func floatFromMap(data map[string]any, key string) float64 {
	value, _ := float64FromAny(data[key])
	return value
}

func optionalInt64FromAny(value any) *int64 {
	parsed, ok := int64FromAny(value)
	if !ok || parsed <= 0 {
		return nil
	}
	return &parsed
}
