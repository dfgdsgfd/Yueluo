package handlers

import (
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
	pointsDailyCapKey = "points_daily_cap"
	defaultDailyCap   = 50
)

type pointsTaskConfigRequest struct {
	TaskType    string  `json:"task_type"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Points      float64 `json:"points"`
	DailyLimit  int     `json:"daily_limit"`
	IsDailyTask *bool   `json:"is_daily_task"`
	IsActive    *bool   `json:"is_active"`
	SortOrder   int     `json:"sort_order"`
}

type pointsAchievementRuleRequest struct {
	Name                string  `json:"name"`
	TriggerType         string  `json:"trigger_type"`
	ThresholdValue      int     `json:"threshold_value"`
	PointsReward        float64 `json:"points_reward"`
	CreatorBonusPercent float64 `json:"creator_bonus_percent"`
	BonusDays           int     `json:"bonus_days"`
	Description         *string `json:"description"`
	IsActive            *bool   `json:"is_active"`
}

type giftCardProductRequest struct {
	Name           string  `json:"name"`
	Description    *string `json:"description"`
	FaceValue      *string `json:"face_value"`
	PointsRequired float64 `json:"points_required"`
	IsActive       *bool   `json:"is_active"`
	SortOrder      int     `json:"sort_order"`
}

type giftCardImportRequest struct {
	Text  string `json:"text"`
	Codes string `json:"codes"`
}

type pointsSettingsRequest struct {
	DailyCap any `json:"daily_cap"`
}

type pointsMaintenanceRequest struct {
	Reason string `json:"reason"`
}

func (h NativeHandlers) PointsOverview(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	overview, err := h.pointsRepo().Overview(c.Request.Context(), user.ID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": h.pointsOverviewResponse(overview)})
}

func (h NativeHandlers) PointsLogs(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := boundedLimit(c, 20, 100)
	total, rows, err := h.pointsRepo().Logs(c.Request.Context(), user.ID, page, limit)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, item := range rows {
		items = append(items, pointsLogResponse(item.Log))
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": gin.H{"list": items, "pagination": paginationTotalPages(page, limit, total)}})
}

func (h NativeHandlers) PointsRedeemGiftCard(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	productID, err := strconv.ParseInt(c.Param("productId"), 10, 64)
	if err != nil || productID <= 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "礼品卡不存在", nil)
		return
	}
	bundle, err := h.pointsRepo().RedeemGiftCard(c.Request.Context(), user.ID, productID)
	if h.writeGiftCardRedeemError(c, err) {
		return
	}
	h.bumpCacheVersions(cacheScopeNotifications)
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "兑换成功", "data": giftCardRedemptionResponse(*bundle, true)})
}

func (h NativeHandlers) PointsRedemptions(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := boundedLimit(c, 20, 100)
	total, rows, err := h.pointsRepo().Redemptions(c.Request.Context(), &user.ID, page, limit)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, item := range rows {
		items = append(items, giftCardRedemptionResponse(item, true))
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": gin.H{"list": items, "pagination": paginationTotalPages(page, limit, total)}})
}

func (h NativeHandlers) PointsAdminStats(c *gin.Context) {
	stats, err := h.pointsRepo().Stats(c.Request.Context())
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": gin.H{
		"total_users":        stats.TotalUsers,
		"total_points":       stats.TotalPoints,
		"today_awarded":      stats.TodayAwarded,
		"total_redeemed":     stats.TotalRedeemed,
		"available_cards":    stats.AvailableCards,
		"active_tasks":       stats.ActiveTasks,
		"active_bonus_users": stats.ActiveBonusUsers,
	}})
}

func (h NativeHandlers) PointsAdminSettings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": gin.H{"daily_cap": h.pointsDailyCap()}})
}

func (h NativeHandlers) PointsAdminUpdateSettings(c *gin.Context) {
	var body pointsSettingsRequest
	_ = c.ShouldBindJSON(&body)
	value, ok := float64FromAny(body.DailyCap)
	if !ok || value < 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "每日积分上限不能小于 0", nil)
		return
	}
	if h.Settings == nil || !h.Settings.Set(c.Request.Context(), pointsDailyCapKey, value) {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "保存积分设置失败", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "积分设置已保存", "data": gin.H{"daily_cap": value}})
}

func (h NativeHandlers) PointsAdminTasks(c *gin.Context) {
	rows, err := h.pointsRepo().TaskConfigs(c.Request.Context(), false)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, taskConfigResponse(row, nil))
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": gin.H{"list": items}})
}

func (h NativeHandlers) PointsAdminCreateTask(c *gin.Context) {
	var body pointsTaskConfigRequest
	_ = c.ShouldBindJSON(&body)
	input := taskConfigFromRequest(body)
	if strings.TrimSpace(input.TaskType) == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "任务类型不能为空", nil)
		return
	}
	row, err := h.pointsRepo().SaveTaskConfig(c.Request.Context(), 0, input)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "任务已创建", "data": taskConfigResponse(*row, nil)})
}

func (h NativeHandlers) PointsAdminUpdateTask(c *gin.Context) {
	id, ok := pathInt64(c, "id")
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "任务不存在", nil)
		return
	}
	var body pointsTaskConfigRequest
	_ = c.ShouldBindJSON(&body)
	input := taskConfigFromRequest(body)
	row, err := h.pointsRepo().SaveTaskConfig(c.Request.Context(), id, input)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "任务已更新", "data": taskConfigResponse(*row, nil)})
}

func (h NativeHandlers) PointsAdminDeleteTask(c *gin.Context) {
	id, ok := pathInt64(c, "id")
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "任务不存在", nil)
		return
	}
	if err := h.pointsRepo().DeleteTaskConfig(c.Request.Context(), id); err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "任务已删除"})
}

func (h NativeHandlers) PointsAdminClearBalances(c *gin.Context) {
	var body pointsMaintenanceRequest
	_ = c.ShouldBindJSON(&body)
	result, err := h.pointsRepo().ClearAllBalances(c.Request.Context(), body.Reason)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "全部积分已清空", "data": pointsMaintenanceResponse(result)})
}

func (h NativeHandlers) PointsAdminResetTaskProgress(c *gin.Context) {
	result, err := h.pointsRepo().ResetTaskProgress(c.Request.Context())
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "积分任务进度已重置", "data": pointsMaintenanceResponse(result)})
}

func (h NativeHandlers) PointsAdminAchievementRules(c *gin.Context) {
	rows, err := h.pointsRepo().AchievementRules(c.Request.Context(), false)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, achievementRuleResponse(row))
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": gin.H{"list": items}})
}

func (h NativeHandlers) PointsAdminCreateAchievementRule(c *gin.Context) {
	var body pointsAchievementRuleRequest
	_ = c.ShouldBindJSON(&body)
	input := achievementRuleFromRequest(body)
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.TriggerType) == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "规则名称和触发类型不能为空", nil)
		return
	}
	row, err := h.pointsRepo().SaveAchievementRule(c.Request.Context(), 0, input)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "规则已创建", "data": achievementRuleResponse(*row)})
}

func (h NativeHandlers) PointsAdminUpdateAchievementRule(c *gin.Context) {
	id, ok := pathInt64(c, "id")
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "规则不存在", nil)
		return
	}
	var body pointsAchievementRuleRequest
	_ = c.ShouldBindJSON(&body)
	row, err := h.pointsRepo().SaveAchievementRule(c.Request.Context(), id, achievementRuleFromRequest(body))
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "规则已更新", "data": achievementRuleResponse(*row)})
}

func (h NativeHandlers) PointsAdminDeleteAchievementRule(c *gin.Context) {
	id, ok := pathInt64(c, "id")
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "规则不存在", nil)
		return
	}
	if err := h.pointsRepo().DeleteAchievementRule(c.Request.Context(), id); err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "规则已删除"})
}

func (h NativeHandlers) PointsAdminGiftCardProducts(c *gin.Context) {
	rows, err := h.pointsRepo().GiftCardProducts(c.Request.Context(), false)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, giftCardProductResponse(row))
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": gin.H{"list": items}})
}

func (h NativeHandlers) PointsAdminCreateGiftCardProduct(c *gin.Context) {
	var body giftCardProductRequest
	_ = c.ShouldBindJSON(&body)
	input := giftCardProductFromRequest(body)
	if strings.TrimSpace(input.Name) == "" || input.PointsRequired <= 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "礼品卡名称和兑换积分不能为空", nil)
		return
	}
	row, err := h.pointsRepo().SaveGiftCardProduct(c.Request.Context(), 0, input)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "礼品卡已创建", "data": giftCardProductResponse(repositories.GiftCardProductStock{Product: *row})})
}

func (h NativeHandlers) PointsAdminUpdateGiftCardProduct(c *gin.Context) {
	id, ok := pathInt64(c, "id")
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "礼品卡不存在", nil)
		return
	}
	var body giftCardProductRequest
	_ = c.ShouldBindJSON(&body)
	row, err := h.pointsRepo().SaveGiftCardProduct(c.Request.Context(), id, giftCardProductFromRequest(body))
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "礼品卡已更新", "data": giftCardProductResponse(repositories.GiftCardProductStock{Product: *row})})
}

func (h NativeHandlers) PointsAdminDeleteGiftCardProduct(c *gin.Context) {
	id, ok := pathInt64(c, "id")
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "礼品卡不存在", nil)
		return
	}
	if err := h.pointsRepo().DeleteGiftCardProduct(c.Request.Context(), id); err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "礼品卡已删除"})
}

func (h NativeHandlers) PointsAdminImportGiftCardCodes(c *gin.Context) {
	productID, ok := pathInt64(c, "id")
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "礼品卡不存在", nil)
		return
	}
	var body giftCardImportRequest
	_ = c.ShouldBindJSON(&body)
	text := body.Text
	if text == "" {
		text = body.Codes
	}
	result, err := h.pointsRepo().ImportGiftCardCodes(c.Request.Context(), productID, text)
	if errors.Is(err, repositories.ErrGiftCardImportEmpty) {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "请粘贴一行一个的礼品卡卡密", nil)
		return
	}
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "卡密导入完成", "data": gin.H{"imported": result.Imported, "skipped": result.Skipped, "batch": result.Batch}})
}

func (h NativeHandlers) PointsAdminGiftCardCodes(c *gin.Context) {
	productID, ok := pathInt64(c, "id")
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "礼品卡不存在", nil)
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := boundedLimit(c, 20, 100)
	total, rows, err := h.pointsRepo().GiftCardCodes(c.Request.Context(), productID, c.Query("status"), page, limit)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, giftCardCodeResponse(row))
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": gin.H{"list": items, "pagination": paginationTotalPages(page, limit, total)}})
}

func (h NativeHandlers) PointsAdminRedemptions(c *gin.Context) {
	page := positiveIntQuery(c, "page", 1)
	limit := boundedLimit(c, 20, 100)
	total, rows, err := h.pointsRepo().Redemptions(c.Request.Context(), nil, page, limit)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, item := range rows {
		items = append(items, giftCardRedemptionResponse(item, true))
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": gin.H{"list": items, "pagination": paginationTotalPages(page, limit, total)}})
}

func (h NativeHandlers) pointsRepo() repositories.PointsRepository {
	return repositories.NewPointsRepository(h.DB, h.pointsDailyCap())
}

func (h NativeHandlers) pointsDailyCap() float64 {
	if h.Settings == nil {
		return defaultDailyCap
	}
	return float64(h.Settings.Int(pointsDailyCapKey, defaultDailyCap))
}

func (h NativeHandlers) awardPointsBestEffort(c *gin.Context, userID int64, taskType string, targetID any, reason string) *repositories.AwardResult {
	if h.DB == nil || userID <= 0 {
		return nil
	}
	return h.pointsRepo().AwardBestEffort(c.Request.Context(), repositories.AwardInput{
		UserID:    userID,
		TaskType:  taskType,
		TargetKey: repositories.PointsTaskTarget(taskType, targetID),
		Reason:    reason,
	})
}

func (h NativeHandlers) evaluateAchievementsBestEffort(c *gin.Context, userID int64) []repositories.AwardResult {
	if h.DB == nil || userID <= 0 {
		return nil
	}
	awards, err := h.pointsRepo().EvaluateAchievements(c.Request.Context(), userID)
	if err != nil {
		return nil
	}
	return awards
}

func (h NativeHandlers) pointsOverviewResponse(overview *repositories.PointsOverview) gin.H {
	tasks := make([]gin.H, 0, len(overview.Tasks))
	for _, task := range overview.Tasks {
		tasks = append(tasks, taskConfigResponse(task.Config, &task))
	}
	giftCards := make([]gin.H, 0, len(overview.GiftCards))
	for _, card := range overview.GiftCards {
		giftCards = append(giftCards, giftCardProductResponse(card))
	}
	var activeBonus any
	if overview.ActiveBonus != nil {
		activeBonus = creatorBonusResponse(*overview.ActiveBonus)
	}
	return gin.H{
		"points":       overview.Points,
		"today_earned": overview.TodayEarned,
		"daily_cap":    overview.DailyCap,
		"tasks":        tasks,
		"gift_cards":   giftCards,
		"active_bonus": activeBonus,
		"generated_at": overview.GeneratedAt,
	}
}

func taskConfigResponse(row domain.PointsTaskConfig, progress *repositories.PointsTaskProgress) gin.H {
	body := gin.H{
		"id":            expressBigInt(row.ID),
		"task_type":     row.TaskType,
		"name":          row.Name,
		"description":   row.Description,
		"points":        row.Points,
		"daily_limit":   row.DailyLimit,
		"is_daily_task": row.IsDailyTask,
		"is_active":     row.IsActive,
		"sort_order":    row.SortOrder,
		"created_at":    row.CreatedAt,
		"updated_at":    row.UpdatedAt,
	}
	if progress != nil {
		body["completed"] = progress.Completed
		body["awarded_points"] = progress.AwardedPoints
		body["remaining_count"] = progress.RemainingCount
		body["reached_limit"] = progress.ReachedLimit
	}
	return body
}

func pointsMaintenanceResponse(result *repositories.PointsMaintenanceResult) gin.H {
	if result == nil {
		return gin.H{}
	}
	return gin.H{
		"affected_users":       result.AffectedUsers,
		"deleted_events":       result.DeletedEvents,
		"deleted_stats":        result.DeletedStats,
		"deleted_achievements": result.DeletedAchievements,
		"deleted_bonuses":      result.DeletedBonuses,
	}
}

func achievementRuleResponse(row domain.PointsAchievementRule) gin.H {
	return gin.H{
		"id":                    expressBigInt(row.ID),
		"name":                  row.Name,
		"trigger_type":          row.TriggerType,
		"threshold_value":       row.ThresholdValue,
		"points_reward":         row.PointsReward,
		"creator_bonus_percent": row.CreatorBonusPercent,
		"bonus_days":            row.BonusDays,
		"description":           row.Description,
		"is_active":             row.IsActive,
		"created_at":            row.CreatedAt,
		"updated_at":            row.UpdatedAt,
	}
}

func giftCardProductResponse(row repositories.GiftCardProductStock) gin.H {
	product := row.Product
	return gin.H{
		"id":              expressBigInt(product.ID),
		"name":            product.Name,
		"description":     product.Description,
		"face_value":      product.FaceValue,
		"points_required": product.PointsRequired,
		"is_active":       product.IsActive,
		"sort_order":      product.SortOrder,
		"available_stock": row.AvailableStock,
		"redeemed_stock":  row.RedeemedStock,
		"created_at":      product.CreatedAt,
		"updated_at":      product.UpdatedAt,
	}
}

func giftCardCodeResponse(row domain.GiftCardCode) gin.H {
	return gin.H{
		"id":            expressBigInt(row.ID),
		"product_id":    expressBigInt(row.ProductID),
		"code":          row.Code,
		"status":        row.Status,
		"import_batch":  row.ImportBatch,
		"redemption_id": expressBigIntPtr(row.RedemptionID),
		"user_id":       expressBigIntPtr(row.UserID),
		"redeemed_at":   row.RedeemedAt,
		"created_at":    row.CreatedAt,
		"updated_at":    row.UpdatedAt,
	}
}

func giftCardRedemptionResponse(row repositories.GiftCardRedemptionBundle, includeCode bool) gin.H {
	item := row.Redemption
	body := gin.H{
		"id":            expressBigInt(item.ID),
		"user_id":       expressBigInt(item.UserID),
		"product_id":    expressBigInt(item.ProductID),
		"code_id":       expressBigInt(item.CodeID),
		"points_spent":  item.PointsSpent,
		"balance_after": item.BalanceAfter,
		"status":        item.Status,
		"created_at":    item.CreatedAt,
	}
	if includeCode {
		body["code"] = item.CodeSnapshot
	}
	if row.Product != nil {
		body["product"] = giftCardProductResponse(repositories.GiftCardProductStock{Product: *row.Product})
	}
	return body
}

func pointsLogResponse(log domain.PointsLog) gin.H {
	return gin.H{
		"id":            expressBigInt(log.ID),
		"amount":        log.Amount,
		"balance_after": log.BalanceAfter,
		"type":          log.Type,
		"reason":        log.Reason,
		"created_at":    log.CreatedAt,
	}
}

func creatorBonusResponse(row domain.UserCreatorBonus) gin.H {
	return gin.H{
		"id":            expressBigInt(row.ID),
		"rule_id":       expressBigIntPtr(row.RuleID),
		"bonus_percent": row.BonusPercent,
		"is_active":     row.IsActive,
		"starts_at":     row.StartsAt,
		"expires_at":    row.ExpiresAt,
	}
}

func taskConfigFromRequest(body pointsTaskConfigRequest) domain.PointsTaskConfig {
	isDaily := true
	if body.IsDailyTask != nil {
		isDaily = *body.IsDailyTask
	}
	isActive := true
	if body.IsActive != nil {
		isActive = *body.IsActive
	}
	return domain.PointsTaskConfig{
		TaskType:    body.TaskType,
		Name:        body.Name,
		Description: body.Description,
		Points:      body.Points,
		DailyLimit:  body.DailyLimit,
		IsDailyTask: isDaily,
		IsActive:    isActive,
		SortOrder:   body.SortOrder,
	}
}

func achievementRuleFromRequest(body pointsAchievementRuleRequest) domain.PointsAchievementRule {
	isActive := true
	if body.IsActive != nil {
		isActive = *body.IsActive
	}
	return domain.PointsAchievementRule{
		Name:                body.Name,
		TriggerType:         body.TriggerType,
		ThresholdValue:      body.ThresholdValue,
		PointsReward:        body.PointsReward,
		CreatorBonusPercent: body.CreatorBonusPercent,
		BonusDays:           body.BonusDays,
		Description:         body.Description,
		IsActive:            isActive,
	}
}

func giftCardProductFromRequest(body giftCardProductRequest) domain.GiftCardProduct {
	isActive := true
	if body.IsActive != nil {
		isActive = *body.IsActive
	}
	return domain.GiftCardProduct{
		Name:           body.Name,
		Description:    body.Description,
		FaceValue:      body.FaceValue,
		PointsRequired: body.PointsRequired,
		IsActive:       isActive,
		SortOrder:      body.SortOrder,
	}
}

func (h NativeHandlers) writeGiftCardRedeemError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, repositories.ErrGiftCardProductNotFound), errors.Is(err, gorm.ErrRecordNotFound):
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "礼品卡不存在", nil)
	case errors.Is(err, repositories.ErrGiftCardProductInactive):
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "该礼品卡暂不可兑换", nil)
	case errors.Is(err, repositories.ErrGiftCardOutOfStock):
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "礼品卡库存不足", nil)
	case errors.Is(err, repositories.ErrPointsInsufficient):
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "积分余额不足", nil)
	default:
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
	}
	return true
}

func boundedLimit(c *gin.Context, fallback, max int) int {
	limit := positiveIntQuery(c, "limit", fallback)
	if limit > max {
		return max
	}
	return limit
}

func pathInt64(c *gin.Context, key string) (int64, bool) {
	value, err := strconv.ParseInt(c.Param(key), 10, 64)
	return value, err == nil && value > 0
}

func expressBigIntPtr(value *int64) any {
	if value == nil {
		return nil
	}
	return expressBigInt(*value)
}

func dateJSON(value time.Time) string {
	return value.Format("2006-01-02")
}
