package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
)

func (h NativeHandlers) AdminBalanceTransactions(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgBalanceInternal, nil)
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := min(positiveIntQuery(c, "limit", 20), 100)
	query := h.DB.WithContext(c.Request.Context()).Model(&domain.ExternalBalanceTransaction{})
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgBalanceInternal, nil)
		return
	}
	var rows []domain.ExternalBalanceTransaction
	if err := query.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&rows).Error; err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgBalanceInternal, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": gin.H{
		"list": rows, "pagination": paginationTotalPages(page, limit, total),
	}})
}

func (h NativeHandlers) AdminBalanceTransactionCompensate(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.invalid_transaction_id", nil)
		return
	}
	if h.Balance == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgBalanceInternal, nil)
		return
	}
	transaction, err := h.Balance.CompensateApplied(c.Request.Context(), id)
	if err != nil {
		h.writeBalanceCenterError(c, err, msgBalanceInternal)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "balance.transaction_compensated", "data": transaction})
}

func (h NativeHandlers) CreatorConfig(c *gin.Context) {
	cfg := h.Config.Creator
	c.JSON(http.StatusOK, gin.H{
		"code": response.CodeSuccess,
		"data": gin.H{
			"platformFeeRate":   cfg.PlatformFeeRate,
			"creatorShareRate":  1 - cfg.PlatformFeeRate,
			"withdrawEnabled":   cfg.WithdrawEnabled,
			"minWithdrawAmount": cfg.MinWithdrawAmount,
			"extendedEarnings": gin.H{
				"enabled":  cfg.ExtendedEarningsEnabled,
				"rates":    creatorRatesResponse(cfg.EarningsRates),
				"dailyCap": cfg.DailyExtendedCap,
			},
		},
		"message": "success",
	})
}

func (h NativeHandlers) CreatorOverview(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	repo := repositories.NewCreatorRepository(h.DB, h.Config.Creator)
	if h.Config.Creator.ExtendedEarningsEnabled {
		_, _ = repo.ClaimExtendedEarnings(c.Request.Context(), user.ID)
	}
	overview, err := repo.Overview(c.Request.Context(), user.ID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCreatorInternal, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{
		"balance":          overview.Earnings.Balance,
		"total_earnings":   overview.Earnings.TotalEarnings,
		"withdrawn_amount": overview.Earnings.WithdrawnAmount,
		"today_earnings":   overview.TodayEarnings,
		"month_earnings":   overview.MonthEarnings,
	}, "message": "success"})
}

func (h NativeHandlers) CreatorTrends(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	days := positiveIntQuery(c, "days", 7)
	load := func() (gin.H, error) {
		data, err := repositories.NewCreatorRepository(h.DB, h.Config.Creator).Trends(c.Request.Context(), user.ID, days)
		if err != nil {
			return nil, err
		}
		return creatorTrendsPayload(data), nil
	}
	if h.Redis != nil {
		cacheKey := h.cacheKeyWithVersions(cacheScopeCreator, []string{cacheScopePosts, cacheScopeInteractions, cacheScopeUsers}, user.ID, "trends", days)
		var cached gin.H
		_, err := h.Redis.CacheGetOrLoad(c.Request.Context(), cacheKey, &cached, cacheTTL(30), func() (any, error) {
			return load()
		})
		if err != nil {
			response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCreatorInternal, nil)
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": cached, "message": "success"})
		return
	}
	data, err := load()
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCreatorInternal, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": data, "message": "success"})
}

func creatorTrendsPayload(data *repositories.CreatorTrendData) gin.H {
	return gin.H{
		"days":      len(data.Labels),
		"labels":    data.Labels,
		"views":     data.Views,
		"likes":     data.Likes,
		"collects":  data.Collects,
		"comments":  data.Comments,
		"followers": data.Followers,
	}
}

func (h NativeHandlers) CreatorStats(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	days := positiveIntQuery(c, "days", 30)
	load := func() (gin.H, error) {
		data, err := repositories.NewCreatorRepository(h.DB, h.Config.Creator).Stats(c.Request.Context(), user.ID, days)
		if err != nil {
			return nil, err
		}
		return creatorStatsPayload(data), nil
	}
	if h.Redis != nil {
		cacheKey := h.cacheKeyWithVersions(cacheScopeCreator, []string{cacheScopePosts, cacheScopeInteractions, cacheScopeUsers}, user.ID, "stats", days)
		var cached gin.H
		_, err := h.Redis.CacheGetOrLoad(c.Request.Context(), cacheKey, &cached, cacheTTL(30), func() (any, error) {
			return load()
		})
		if err != nil {
			response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCreatorInternal, nil)
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": cached, "message": "success"})
		return
	}
	data, err := load()
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCreatorInternal, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": data, "message": "success"})
}

func creatorStatsPayload(data *repositories.CreatorStatsData) gin.H {
	return gin.H{
		"window_days":  data.Days,
		"generated_at": data.GeneratedAt.UTC().Format(time.RFC3339Nano),
		"fans":         data.Fans,
		"post_totals":  data.PostTotals,
		"interactions": data.Interactions,
	}
}

func (h NativeHandlers) CreatorEarningsLog(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := min(positiveIntQuery(c, "limit", 20), 100)
	total, logs, err := repositories.NewCreatorRepository(h.DB, h.Config.Creator).EarningsLog(c.Request.Context(), user.ID, page, limit, c.Query("type"))
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCreatorInternal, nil)
		return
	}
	list := make([]gin.H, 0, len(logs))
	for _, item := range logs {
		list = append(list, h.creatorEarningsLogResponse(item))
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{"list": list, "pagination": paginationTotalPages(page, limit, total)}, "message": "success"})
}

func (h NativeHandlers) CreatorPaidContent(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := min(positiveIntQuery(c, "limit", 20), 100)
	total, posts, err := repositories.NewCreatorRepository(h.DB, h.Config.Creator).PaidContent(c.Request.Context(), user.ID, page, limit)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCreatorInternal, nil)
		return
	}
	list := make([]gin.H, 0, len(posts))
	for _, item := range posts {
		list = append(list, h.paidContentResponse(item))
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{"list": list, "pagination": paginationTotalPages(page, limit, total)}, "message": "success"})
}

type creatorWithdrawRequest struct {
	Amount any `json:"amount"`
}

func (h NativeHandlers) CreatorWithdraw(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	var body creatorWithdrawRequest
	_ = c.ShouldBindJSON(&body)
	amount, ok := float64FromAny(body.Amount)
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "\u8bf7\u8f93\u5165\u6709\u6548\u7684\u63d0\u73b0\u91d1\u989d", nil)
		return
	}
	bonusAmount, activeBonus, err := h.pointsRepo().CreatorWithdrawBonus(c.Request.Context(), user.ID, amount)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCreatorInternal, nil)
		return
	}
	newEarnings, newPoints, err := repositories.NewCreatorRepository(h.DB, h.Config.Creator).WithdrawToPoints(c.Request.Context(), user.ID, amount, bonusAmount)
	if h.writeCreatorWithdrawError(c, err, amount) {
		return
	}
	bonusPercent := 0.0
	if activeBonus != nil {
		bonusPercent = activeBonus.BonusPercent
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{
		"amount":             amount,
		"bonusAmount":        bonusAmount,
		"bonusPercent":       bonusPercent,
		"newEarningsBalance": newEarnings,
		"newPointsBalance":   newPoints,
	}, "message": "\u63d0\u73b0\u6210\u529f"})
}

func (h NativeHandlers) CreatorClaimIncentive(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	result, err := repositories.NewCreatorRepository(h.DB, h.Config.Creator).ClaimExtendedEarnings(c.Request.Context(), user.ID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCreatorInternal, nil)
		return
	}
	if !result.Success {
		c.JSON(http.StatusBadRequest, gin.H{"code": response.CodeValidationError, "message": result.Message, "data": gin.H{"alreadyClaimed": result.AlreadyClaimed, "noEarnings": result.NoEarnings}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{"earnings": extendedEarningsResponse(result.Earnings), "newBalance": result.NewBalance, "details": result.Details}, "message": result.Message})
}

func (h NativeHandlers) CreatorQualityRewards(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := min(positiveIntQuery(c, "limit", 20), 100)
	total, totalAmount, items, stats, err := repositories.NewCreatorRepository(h.DB, h.Config.Creator).QualityRewards(c.Request.Context(), user.ID, page, limit)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgCreatorInternal, nil)
		return
	}
	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		list = append(list, h.qualityRewardResponse(item))
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{"list": list, "total_earnings": totalAmount, "stats": qualityRewardStatsResponse(stats), "pagination": paginationTotalPages(page, limit, total)}, "message": "success"})
}
