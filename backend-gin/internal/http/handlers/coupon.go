package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
)

const (
	msgCouponInternal             = "\u670d\u52a1\u5668\u5185\u90e8\u9519\u8bef"
	msgCouponCodeRequired         = "\u8bf7\u8f93\u5165\u4f18\u60e0\u5238\u7801"
	msgCouponCodeNotFound         = "\u4f18\u60e0\u5238\u7801\u4e0d\u5b58\u5728"
	msgCouponClaimed              = "\u60a8\u5df2\u9886\u53d6\u8fc7\u8be5\u4f18\u60e0\u5238"
	msgCouponOutOfStock           = "\u8be5\u4f18\u60e0\u5238\u5df2\u88ab\u9886\u5b8c"
	msgCouponClaimOK              = "\u9886\u53d6\u6210\u529f\uff01"
	msgCouponIDRequired           = "\u7f3a\u5c11\u4f18\u60e0\u5238ID"
	msgCouponUseOK                = "\u4f18\u60e0\u5238\u4f7f\u7528\u6210\u529f"
	msgCouponNotFound             = "\u4f18\u60e0\u5238\u4e0d\u5b58\u5728"
	msgCouponCodeExists           = "\u8be5\u4f18\u60e0\u5238\u7801\u5df2\u5b58\u5728"
	msgCouponCreateOK             = "\u4f18\u60e0\u5238\u521b\u5efa\u6210\u529f"
	msgCouponUpdateOK             = "\u66f4\u65b0\u6210\u529f"
	msgCouponDeleteOK             = "\u5220\u9664\u6210\u529f"
	msgCouponMissingFields        = "\u7f3a\u5c11\u5fc5\u586b\u5b57\u6bb5: name, type, value, start_time, end_time"
	msgCouponInvalidType          = "type \u5fc5\u987b\u4e3a amount \u6216 percent"
	msgCouponInvalidPercent       = "percent \u7c7b\u578b\u6298\u6263\u503c\u9700\u5728 1-100 \u4e4b\u95f4"
	msgCouponUsersRequired        = "\u8bf7\u63d0\u4f9b user_ids \u6570\u7ec4"
	msgCouponBalanceNotConfigured = "error.balance_center_not_configured"
)

func (h NativeHandlers) CouponMy(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCouponInternal, nil)
		return
	}
	bundles, err := repositories.NewCouponRepository(h.DB).MyCoupons(c.Request.Context(), user.ID, c.Query("status"))
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCouponInternal, nil)
		return
	}
	list := make([]gin.H, 0, len(bundles))
	for _, bundle := range bundles {
		list = append(list, userCouponResponse(bundle))
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{"list": list}, "message": "success"})
}

type couponClaimRequest struct {
	Code any `json:"code"`
}

func (h NativeHandlers) CouponClaim(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	var body couponClaimRequest
	_ = c.ShouldBindJSON(&body)
	code, ok := stringFromAny(body.Code)
	if !ok || strings.TrimSpace(code) == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgCouponCodeRequired, nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCouponInternal, nil)
		return
	}
	userCoupon, coupon, err := repositories.NewCouponRepository(h.DB).Claim(c.Request.Context(), user.ID, code)
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgCouponCodeNotFound, nil)
	case errors.Is(err, repositories.ErrCouponAlreadyClaimed):
		response.JSON(c, http.StatusConflict, response.CodeConflict, msgCouponClaimed, nil)
	case errors.Is(err, repositories.ErrCouponOutOfStock):
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgCouponOutOfStock, nil)
	case err != nil && coupon != nil:
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, couponWindowMessage(err), nil)
	case err != nil:
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCouponInternal, nil)
	default:
		c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{"id": strconv.FormatInt(userCoupon.ID, 10), "coupon_name": coupon.Name}, "message": msgCouponClaimOK})
	}
}

type couponValidateRequest struct {
	UserCouponID any `json:"user_coupon_id"`
	OrderAmount  any `json:"order_amount"`
}

func (h NativeHandlers) CouponValidate(c *gin.Context) {
	h.validateOrUseCoupon(c, false)
}

func (h NativeHandlers) CouponUse(c *gin.Context) {
	h.validateOrUseCoupon(c, true)
}

func (h NativeHandlers) validateOrUseCoupon(c *gin.Context, consume bool) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	var body couponValidateRequest
	_ = c.ShouldBindJSON(&body)
	userCouponID, okID := int64FromAny(body.UserCouponID)
	if !okID {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgCouponIDRequired, nil)
		return
	}
	orderAmount, _ := float64FromAny(body.OrderAmount)
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCouponInternal, nil)
		return
	}
	repo := repositories.NewCouponRepository(h.DB)
	var validation *repositories.CouponValidation
	var err error
	if consume {
		validation, err = repo.Use(c.Request.Context(), user.ID, userCouponID, orderAmount)
	} else {
		validation, err = repo.Validate(c.Request.Context(), user.ID, userCouponID, orderAmount)
	}
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCouponInternal, nil)
		return
	}
	if !validation.Valid {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, validation.Message, nil)
		return
	}
	message := "success"
	if consume {
		message = msgCouponUseOK
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    response.CodeSuccess,
		"data":    gin.H{"discount": validation.Discount, "final_amount": validation.FinalAmount},
		"message": message,
	})
}

func (h NativeHandlers) CouponAdminList(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCouponInternal, nil)
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := positiveIntQuery(c, "limit", 20)
	total, coupons, issuedCounts, err := repositories.NewCouponRepository(h.DB).AdminList(c.Request.Context(), page, limit, c.Query("keyword"))
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCouponInternal, nil)
		return
	}
	list := make([]gin.H, 0, len(coupons))
	for _, coupon := range coupons {
		list = append(list, couponAdminResponse(coupon, issuedCounts[coupon.ID]))
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    response.CodeSuccess,
		"data":    gin.H{"list": list, "pagination": paginationTotalPages(page, limit, total)},
		"message": "success",
	})
}

type couponAdminRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Code        any    `json:"code"`
	Type        string `json:"type"`
	Value       any    `json:"value"`
	MinOrder    any    `json:"min_order"`
	MaxDiscount any    `json:"max_discount"`
	StartTime   any    `json:"start_time"`
	EndTime     any    `json:"end_time"`
	TotalCount  any    `json:"total_count"`
	IsActive    any    `json:"is_active"`
}

func (h NativeHandlers) CouponAdminCreate(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCouponInternal, nil)
		return
	}
	var body couponAdminRequest
	_ = c.ShouldBindJSON(&body)
	coupon, errMsg, ok := couponFromRequest(body, false)
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, errMsg, nil)
		return
	}
	created, err := repositories.NewCouponRepository(h.DB).Create(c.Request.Context(), coupon)
	if errors.Is(err, repositories.ErrCouponCodeExists) {
		response.JSON(c, http.StatusConflict, response.CodeConflict, msgCouponCodeExists, nil)
		return
	}
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCouponInternal, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{"id": strconv.FormatInt(created.ID, 10), "name": created.Name}, "message": msgCouponCreateOK})
}

func (h NativeHandlers) CouponAdminUpdate(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCouponInternal, nil)
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgCouponNotFound, nil)
		return
	}
	var body couponAdminRequest
	_ = c.ShouldBindJSON(&body)
	updates, code, errMsg, ok := couponUpdatesFromRequest(body)
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, errMsg, nil)
		return
	}
	err = repositories.NewCouponRepository(h.DB).Update(c.Request.Context(), id, updates, code)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgCouponNotFound, nil)
		return
	}
	if errors.Is(err, repositories.ErrCouponCodeExists) {
		response.JSON(c, http.StatusConflict, response.CodeConflict, msgCouponCodeExists, nil)
		return
	}
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCouponInternal, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{"id": strconv.FormatInt(id, 10)}, "message": msgCouponUpdateOK})
}

func (h NativeHandlers) CouponAdminDelete(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCouponInternal, nil)
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgCouponNotFound, nil)
		return
	}
	err = repositories.NewCouponRepository(h.DB).Delete(c.Request.Context(), id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgCouponNotFound, nil)
		return
	}
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCouponInternal, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": msgCouponDeleteOK})
}

type couponIssueRequest struct {
	TargetType string `json:"target_type"`
	UserIDs    any    `json:"user_ids"`
	UserID     any    `json:"user_id"`
}

func (h NativeHandlers) CouponAdminIssue(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCouponInternal, nil)
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgCouponNotFound, nil)
		return
	}
	var body couponIssueRequest
	_ = c.ShouldBindJSON(&body)
	targetType := body.TargetType
	if targetType == "" {
		targetType = "users"
	}
	userIDs, errMsg, ok := h.resolveCouponIssueUsers(c.Request.Context(), targetType, body.UserIDs, body.UserID)
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, errMsg, nil)
		return
	}
	issued, skipped, err := repositories.NewCouponRepository(h.DB).Issue(c.Request.Context(), id, user.ID, targetType, userIDs)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgCouponNotFound, nil)
		return
	}
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCouponInternal, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    response.CodeSuccess,
		"data":    gin.H{"issued_count": len(issued), "issued": issued, "skipped": skipped},
		"message": "\u6210\u529f\u53d1\u653e " + strconv.Itoa(len(issued)) + " \u5f20\uff0c\u8df3\u8fc7 " + strconv.Itoa(len(skipped)) + " \u4e2a",
	})
}

func (h NativeHandlers) CouponAdminUsages(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCouponInternal, nil)
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgCouponNotFound, nil)
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := positiveIntQuery(c, "limit", 20)
	total, usages, err := repositories.NewCouponRepository(h.DB).Usages(c.Request.Context(), id, page, limit, c.Query("status"))
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCouponInternal, nil)
		return
	}
	list := make([]gin.H, 0, len(usages))
	for _, usage := range usages {
		list = append(list, h.couponUsageResponse(usage))
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{"list": list, "pagination": paginationTotalPages(page, limit, total)}, "message": "success"})
}

func (h NativeHandlers) CouponAdminStats(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCouponInternal, nil)
		return
	}
	stats, err := repositories.NewCouponRepository(h.DB).Stats(c.Request.Context())
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCouponInternal, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{
		"totalCoupons":  stats.TotalCoupons,
		"activeCoupons": stats.ActiveCoupons,
		"totalIssued":   stats.TotalIssued,
		"totalUsed":     stats.TotalUsed,
	}, "message": "success"})
}

func (h NativeHandlers) resolveCouponIssueUsers(ctx context.Context, targetType string, rawIDs any, rawID any) ([]int64, string, bool) {
	repo := repositories.NewCouponRepository(h.DB)
	switch targetType {
	case "all":
		users, err := repo.ActiveUsers(ctx, false)
		if err != nil {
			return nil, msgCouponInternal, false
		}
		return usersToIDs(users), "", true
	case "vip1", "vip2":
		if !balanceCenterConfigured(h.Config.Balance) {
			return nil, msgCouponBalanceNotConfigured, false
		}
		minLevel := 1
		if targetType == "vip2" {
			minLevel = 2
		}
		users, err := repo.ActiveUsers(ctx, true)
		if err != nil {
			return nil, msgCouponInternal, false
		}
		ids := h.filterVIPUsers(ctx, users, minLevel)
		return ids, "", true
	default:
		ids := int64SliceFromAny(rawIDs)
		if len(ids) == 0 {
			if id, ok := int64FromAny(rawID); ok {
				ids = []int64{id}
			}
		}
		if len(ids) == 0 {
			return nil, msgCouponUsersRequired, false
		}
		return ids, "", true
	}
}

func userCouponResponse(bundle repositories.UserCouponBundle) gin.H {
	uc := bundle.UserCoupon
	return gin.H{
		"id":         strconv.FormatInt(uc.ID, 10),
		"coupon_id":  strconv.FormatInt(uc.CouponID, 10),
		"status":     uc.Status,
		"used_at":    uc.UsedAt,
		"created_at": uc.CreatedAt,
		"coupon":     couponUserResponse(bundle.Coupon),
	}
}

func couponUserResponse(coupon *domain.Coupon) any {
	if coupon == nil {
		return nil
	}
	return gin.H{
		"id":           strconv.FormatInt(coupon.ID, 10),
		"name":         coupon.Name,
		"description":  coupon.Description,
		"type":         coupon.Type,
		"value":        coupon.Value,
		"min_order":    coupon.MinOrder,
		"max_discount": coupon.MaxDiscount,
		"start_time":   coupon.StartTime,
		"end_time":     coupon.EndTime,
		"is_active":    coupon.IsActive,
	}
}

func couponAdminResponse(coupon domain.Coupon, issuedCount int64) gin.H {
	return gin.H{
		"id":           strconv.FormatInt(coupon.ID, 10),
		"name":         coupon.Name,
		"description":  coupon.Description,
		"code":         coupon.Code,
		"type":         coupon.Type,
		"value":        coupon.Value,
		"min_order":    coupon.MinOrder,
		"max_discount": coupon.MaxDiscount,
		"start_time":   coupon.StartTime,
		"end_time":     coupon.EndTime,
		"total_count":  coupon.TotalCount,
		"used_count":   coupon.UsedCount,
		"issued_count": issuedCount,
		"is_active":    coupon.IsActive,
		"created_at":   coupon.CreatedAt,
	}
}

func (h NativeHandlers) couponUsageResponse(bundle repositories.UserCouponBundle) gin.H {
	uc := bundle.UserCoupon
	var user any
	if bundle.User != nil {
		user = gin.H{"id": strconv.FormatInt(bundle.User.ID, 10), "user_id": bundle.User.UserID, "nickname": bundle.User.Nickname, "avatar": h.signFileURLPtr(bundle.User.Avatar)}
	}
	var issuedBy any
	if uc.IssuedBy != nil {
		issuedBy = strconv.FormatInt(*uc.IssuedBy, 10)
	}
	return gin.H{"id": strconv.FormatInt(uc.ID, 10), "user_id": strconv.FormatInt(uc.UserID, 10), "status": uc.Status, "used_at": uc.UsedAt, "issued_by": issuedBy, "created_at": uc.CreatedAt, "user": user}
}

func couponFromRequest(body couponAdminRequest, partial bool) (domain.Coupon, string, bool) {
	if !partial && (body.Name == "" || body.Type == "" || body.Value == nil || body.StartTime == nil || body.EndTime == nil) {
		return domain.Coupon{}, msgCouponMissingFields, false
	}
	value, ok := float64FromAny(body.Value)
	if !ok {
		return domain.Coupon{}, msgCouponMissingFields, false
	}
	if body.Type != "amount" && body.Type != "percent" {
		return domain.Coupon{}, msgCouponInvalidType, false
	}
	if body.Type == "percent" && (value <= 0 || value > 100) {
		return domain.Coupon{}, msgCouponInvalidPercent, false
	}
	start, okStart := timeFromAny(body.StartTime)
	end, okEnd := timeFromAny(body.EndTime)
	if !okStart || !okEnd {
		return domain.Coupon{}, msgCouponMissingFields, false
	}
	minOrder, _ := float64FromAny(body.MinOrder)
	totalCount := -1
	if parsed, ok := intFromAny(body.TotalCount); ok {
		totalCount = parsed
	}
	active := true
	if parsed, ok := boolFromAny(body.IsActive); ok {
		active = parsed
	}
	var code *string
	if raw, ok := stringFromAny(body.Code); ok && strings.TrimSpace(raw) != "" {
		normalized := strings.ToUpper(strings.TrimSpace(raw))
		code = &normalized
	}
	var maxDiscount *float64
	if parsed, ok := float64FromAny(body.MaxDiscount); ok {
		maxDiscount = &parsed
	}
	var description *string
	if strings.TrimSpace(body.Description) != "" {
		description = &body.Description
	}
	return domain.Coupon{Name: body.Name, Description: description, Code: code, Type: body.Type, Value: value, MinOrder: minOrder, MaxDiscount: maxDiscount, StartTime: start, EndTime: end, TotalCount: totalCount, IsActive: active}, "", true
}

func couponUpdatesFromRequest(body couponAdminRequest) (map[string]any, *string, string, bool) {
	updates := map[string]any{}
	var codeForCheck *string
	if body.Name != "" {
		updates["name"] = body.Name
	}
	if body.Description != "" {
		updates["description"] = body.Description
	}
	if body.Code != nil {
		raw, _ := stringFromAny(body.Code)
		if strings.TrimSpace(raw) == "" {
			updates["code"] = nil
		} else {
			normalized := strings.ToUpper(strings.TrimSpace(raw))
			updates["code"] = normalized
			codeForCheck = &normalized
		}
	}
	if body.Type != "" {
		if body.Type != "amount" && body.Type != "percent" {
			return nil, nil, msgCouponInvalidType, false
		}
		updates["type"] = body.Type
	}
	if body.Value != nil {
		value, ok := float64FromAny(body.Value)
		if !ok {
			return nil, nil, msgCouponMissingFields, false
		}
		if body.Type == "percent" && (value <= 0 || value > 100) {
			return nil, nil, msgCouponInvalidPercent, false
		}
		updates["value"] = value
	}
	if body.MinOrder != nil {
		value, _ := float64FromAny(body.MinOrder)
		updates["min_order"] = value
	}
	if body.MaxDiscount != nil {
		value, ok := float64FromAny(body.MaxDiscount)
		if ok {
			updates["max_discount"] = value
		} else {
			updates["max_discount"] = nil
		}
	}
	if body.StartTime != nil {
		value, ok := timeFromAny(body.StartTime)
		if !ok {
			return nil, nil, msgCouponMissingFields, false
		}
		updates["start_time"] = value
	}
	if body.EndTime != nil {
		value, ok := timeFromAny(body.EndTime)
		if !ok {
			return nil, nil, msgCouponMissingFields, false
		}
		updates["end_time"] = value
	}
	if body.TotalCount != nil {
		value, ok := intFromAny(body.TotalCount)
		if ok {
			updates["total_count"] = value
		}
	}
	if body.IsActive != nil {
		value, ok := boolFromAny(body.IsActive)
		if ok {
			updates["is_active"] = value
		}
	}
	return updates, codeForCheck, "", true
}

func paginationTotalPages(page, limit int, total int64) gin.H {
	body := pagination(page, limit, total)
	body["totalPages"] = body["pages"]
	delete(body, "pages")
	return body
}

func couponWindowMessage(err error) string {
	switch {
	case errors.Is(err, repositories.ErrCouponInactive):
		return "\u8be5\u4f18\u60e0\u5238\u5df2\u505c\u7528"
	case errors.Is(err, repositories.ErrCouponNotStarted):
		return "\u8be5\u4f18\u60e0\u5238\u5c1a\u672a\u5f00\u59cb"
	case errors.Is(err, repositories.ErrCouponExpired):
		return "\u8be5\u4f18\u60e0\u5238\u5df2\u8fc7\u671f"
	default:
		return "\u4f18\u60e0\u5238\u4e0d\u53ef\u7528"
	}
}

func usersToIDs(users []domain.User) []int64 {
	out := make([]int64, 0, len(users))
	for _, user := range users {
		out = append(out, user.ID)
	}
	return out
}

func int64SliceFromAny(value any) []int64 {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]int64, 0, len(values))
	for _, item := range values {
		if parsed, ok := int64FromAny(item); ok {
			out = append(out, parsed)
		}
	}
	return out
}

func stringFromAny(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		return typed, true
	default:
		return "", false
	}
}

func boolFromAny(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes":
			return true, true
		case "false", "0", "no":
			return false, true
		}
	default:
		return false, false
	}
	return false, false
}

func timeFromAny(value any) (time.Time, bool) {
	raw, ok := stringFromAny(value)
	if !ok || strings.TrimSpace(raw) == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}
