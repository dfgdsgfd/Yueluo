package handlers

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

type adminLogRange struct {
	Start  time.Time
	End    time.Time
	Bucket string
}

func (h NativeHandlers) AdminAccessLogs(c *gin.Context) {
	if _, ok := h.requireMatrixAdmin(c); !ok || !h.requireDB(c) {
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := min(positiveIntQuery(c, "limit", 30), 100)
	if limit <= 0 {
		limit = 30
	}
	query := h.DB.WithContext(c.Request.Context()).Model(&domain.AccessLog{})
	query = applyAccessLogFilters(c, query)
	var total int64
	if err := query.Count(&total).Error; writeDBError(c, err, "") {
		return
	}
	var rows []domain.AccessLog
	err := query.Order("created_at DESC, id DESC").Offset((page - 1) * limit).Limit(limit).Find(&rows).Error
	if writeDBError(c, err, "") {
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, h.accessLogMap(row))
	}
	writeSuccess(c, matrixMsgOK, gin.H{
		"items":      items,
		"pagination": matrixPagination(page, limit, total),
		"status":     h.auditLogRuntimeStatus(),
	})
}

func (h NativeHandlers) AdminSecurityAuditLogs(c *gin.Context) {
	if _, ok := h.requireMatrixAdmin(c); !ok || !h.requireDB(c) {
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := min(positiveIntQuery(c, "limit", 30), 100)
	if limit <= 0 {
		limit = 30
	}
	query := h.DB.WithContext(c.Request.Context()).Model(&domain.SecurityAuditLog{})
	query = applySecurityAuditLogFilters(c, query)
	var total int64
	if err := query.Count(&total).Error; writeDBError(c, err, "") {
		return
	}
	var rows []domain.SecurityAuditLog
	err := query.Order("created_at DESC, id DESC").Offset((page - 1) * limit).Limit(limit).Find(&rows).Error
	if writeDBError(c, err, "") {
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, h.securityAuditLogMap(row))
	}
	writeSuccess(c, matrixMsgOK, gin.H{
		"items":      items,
		"pagination": matrixPagination(page, limit, total),
		"status":     h.auditLogRuntimeStatus(),
	})
}

type adminPointsAuditRow struct {
	ID                    int64      `gorm:"column:id"`
	UserID                int64      `gorm:"column:user_id"`
	UserDisplayID         *string    `gorm:"column:user_display_id"`
	Nickname              *string    `gorm:"column:nickname"`
	Amount                float64    `gorm:"column:amount"`
	BalanceAfter          float64    `gorm:"column:balance_after"`
	Type                  string     `gorm:"column:type"`
	Reason                *string    `gorm:"column:reason"`
	PostID                *int64     `gorm:"column:post_id"`
	PostTitle             *string    `gorm:"column:post_title"`
	PurchaseID            *int64     `gorm:"column:purchase_id"`
	EntryRole             string     `gorm:"column:entry_role"`
	PaymentMethod         string     `gorm:"column:payment_method"`
	CounterpartyUserID    *int64     `gorm:"column:counterparty_user_id"`
	CounterpartyDisplayID *string    `gorm:"column:counterparty_display_id"`
	CounterpartyNickname  *string    `gorm:"column:counterparty_nickname"`
	CreatedAt             time.Time  `gorm:"column:created_at"`
	UserCreatedAt         *time.Time `gorm:"column:user_created_at"`
}

func (h NativeHandlers) AdminPointsAuditLogs(c *gin.Context) {
	if _, ok := h.requireMatrixAdmin(c); !ok || !h.requireDB(c) {
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := min(positiveIntQuery(c, "limit", 30), 100)
	if limit <= 0 {
		limit = 30
	}
	window := parseAdminLogRange(c)
	query := h.DB.WithContext(c.Request.Context()).
		Table("points_log AS pl").
		Joins("LEFT JOIN users u ON u.id = pl.user_id").
		Joins("LEFT JOIN posts p ON p.id = pl.post_id").
		Joins("LEFT JOIN users cu ON cu.id = pl.counterparty_user_id").
		Where("pl.created_at >= ? AND pl.created_at < ?", window.Start, window.End)
	if logType := strings.TrimSpace(c.Query("type")); logType != "" {
		query = query.Where("pl.type = ?", logType)
	}
	if userID := strings.TrimSpace(c.Query("user_id")); userID != "" {
		query = query.Where("pl.user_id = ? OR u.user_id = ?", int64FromString(userID), userID)
	}
	if strings.EqualFold(strings.TrimSpace(c.Query("anomaly")), "only") {
		query = query.Where("(pl.balance_after < 0 OR pl.amount = 0 OR u.id IS NULL)")
	}
	if like, ok := logKeywordLike(c.Query("keyword")); ok {
		query = query.Where(
			"pl.type LIKE ? OR COALESCE(pl.reason, '') LIKE ? OR COALESCE(u.user_id, '') LIKE ? OR COALESCE(u.nickname, '') LIKE ? OR COALESCE(p.title, '') LIKE ? OR COALESCE(cu.user_id, '') LIKE ? OR COALESCE(cu.nickname, '') LIKE ? OR "+logTextCast(h.DB, "pl.user_id")+" LIKE ?",
			like, like, like, like, like, like, like, like,
		)
	}
	var total int64
	if err := query.Count(&total).Error; writeDBError(c, err, "") {
		return
	}
	var rows []adminPointsAuditRow
	err := query.
		Select("pl.id, pl.user_id, u.user_id AS user_display_id, u.nickname, pl.amount, pl.balance_after, pl.type, pl.reason, pl.post_id, p.title AS post_title, pl.purchase_id, pl.entry_role, pl.payment_method, pl.counterparty_user_id, cu.user_id AS counterparty_display_id, cu.nickname AS counterparty_nickname, pl.created_at, u.created_at AS user_created_at").
		Order("pl.created_at DESC, pl.id DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Scan(&rows).Error
	if writeDBError(c, err, "") {
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		entryRole := normalizedLedgerEntryRole(row.EntryRole, row.Type, row.Amount)
		items = append(items, gin.H{
			"id":              row.ID,
			"user_id":         row.UserID,
			"user_display_id": row.UserDisplayID,
			"nickname":        row.Nickname,
			"amount":          row.Amount,
			"balance_after":   row.BalanceAfter,
			"type":            row.Type,
			"reason":          row.Reason,
			"post_id":         row.PostID,
			"post_title":      row.PostTitle,
			"purchase_id":     row.PurchaseID,
			"entry_role":      entryRole,
			"payment_method":  row.PaymentMethod,
			"counterparty": gin.H{
				"user_id":         row.CounterpartyUserID,
				"user_display_id": row.CounterpartyDisplayID,
				"nickname":        row.CounterpartyNickname,
			},
			"is_anomaly": row.BalanceAfter < 0 || row.Amount == 0 || row.UserCreatedAt == nil,
			"created_at": row.CreatedAt,
		})
	}
	writeSuccess(c, matrixMsgOK, gin.H{
		"items":      items,
		"pagination": matrixPagination(page, limit, total),
	})
}

func normalizedLedgerEntryRole(entryRole, entryType string, amount float64) string {
	if value := strings.TrimSpace(entryRole); value != "" {
		return value
	}
	switch strings.ToLower(strings.TrimSpace(entryType)) {
	case "paid_content_purchase":
		return "buyer_debit"
	case "paid_content_sale", "content_sale":
		return "author_credit"
	}
	if amount < 0 {
		return "debit"
	}
	return "credit"
}

type adminBalanceAuditRow struct {
	ID                    int64      `gorm:"column:id"`
	LedgerSource          string     `gorm:"column:ledger_source"`
	OperationKey          string     `gorm:"column:operation_key"`
	UserID                int64      `gorm:"column:user_id"`
	UserDisplayID         *string    `gorm:"column:user_display_id"`
	Nickname              *string    `gorm:"column:nickname"`
	OAuth2ID              *int64     `gorm:"column:oauth2_id"`
	Amount                float64    `gorm:"column:amount"`
	Reason                string     `gorm:"column:reason"`
	Status                string     `gorm:"column:status"`
	RemoteBalanceAfter    *float64   `gorm:"column:remote_balance_after"`
	CompensationAmount    *float64   `gorm:"column:compensation_amount"`
	Attempts              int        `gorm:"column:attempts"`
	LastError             *string    `gorm:"column:last_error"`
	PostID                *int64     `gorm:"column:post_id"`
	PostTitle             *string    `gorm:"column:post_title"`
	PurchaseID            *int64     `gorm:"column:purchase_id"`
	EntryRole             string     `gorm:"column:entry_role"`
	PaymentMethod         string     `gorm:"column:payment_method"`
	PlatformFee           float64    `gorm:"column:platform_fee"`
	CounterpartyUserID    *int64     `gorm:"column:counterparty_user_id"`
	CounterpartyDisplayID *string    `gorm:"column:counterparty_display_id"`
	CounterpartyNickname  *string    `gorm:"column:counterparty_nickname"`
	CreatedAt             time.Time  `gorm:"column:created_at"`
	UpdatedAt             *time.Time `gorm:"column:updated_at"`
	AppliedAt             *time.Time `gorm:"column:applied_at"`
	CompletedAt           *time.Time `gorm:"column:completed_at"`
	UserCreatedAt         *time.Time `gorm:"column:user_created_at"`
}

func (h NativeHandlers) AdminBalanceAuditLogs(c *gin.Context) {
	if _, ok := h.requireMatrixAdmin(c); !ok || !h.requireDB(c) {
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := min(positiveIntQuery(c, "limit", 30), 100)
	if limit <= 0 {
		limit = 30
	}
	window := parseAdminLogRange(c)
	union := h.DB.WithContext(c.Request.Context()).Raw(`
		SELECT
			ebt.id,
			'external_balance' AS ledger_source,
			ebt.operation_key,
			ebt.user_id,
			u.user_id AS user_display_id,
			u.nickname,
			ebt.oauth2_id,
			ebt.amount,
			ebt.reason,
			ebt.status,
			ebt.remote_balance_after,
			ebt.compensation_amount,
			ebt.attempts,
			ebt.last_error,
			ebt.post_id,
			p.title AS post_title,
			ebt.purchase_id,
			ebt.entry_role,
			ebt.payment_method,
			ebt.platform_fee,
			ebt.counterparty_user_id,
			cu.user_id AS counterparty_display_id,
			cu.nickname AS counterparty_nickname,
			ebt.created_at,
			ebt.updated_at,
			ebt.applied_at,
			ebt.completed_at,
			u.created_at AS user_created_at
		FROM external_balance_transactions ebt
		LEFT JOIN users u ON u.id = ebt.user_id
		LEFT JOIN posts p ON p.id = ebt.post_id
		LEFT JOIN users cu ON cu.id = ebt.counterparty_user_id
		WHERE ebt.created_at >= ? AND ebt.created_at < ?
		UNION ALL
		SELECT
			cel.id,
			'creator_earnings' AS ledger_source,
			'' AS operation_key,
			cel.user_id,
			u.user_id AS user_display_id,
			u.nickname,
			NULL AS oauth2_id,
			cel.amount,
			COALESCE(cel.reason, '') AS reason,
			'completed' AS status,
			cel.balance_after AS remote_balance_after,
			NULL AS compensation_amount,
			1 AS attempts,
			NULL AS last_error,
			CASE WHEN cel.source_type = 'post' THEN cel.source_id ELSE NULL END AS post_id,
			p.title AS post_title,
			NULL AS purchase_id,
			'author_credit' AS entry_role,
			'balance' AS payment_method,
			cel.platform_fee,
			cel.buyer_id AS counterparty_user_id,
			cu.user_id AS counterparty_display_id,
			cu.nickname AS counterparty_nickname,
			cel.created_at,
			NULL AS updated_at,
			NULL AS applied_at,
			cel.created_at AS completed_at,
			u.created_at AS user_created_at
		FROM creator_earnings_log cel
		LEFT JOIN users u ON u.id = cel.user_id
		LEFT JOIN posts p ON p.id = cel.source_id AND cel.source_type = 'post'
		LEFT JOIN users cu ON cu.id = cel.buyer_id
		WHERE cel.type = 'content_sale' AND cel.created_at >= ? AND cel.created_at < ?
	`, window.Start, window.End, window.Start, window.End)
	query := h.DB.WithContext(c.Request.Context()).Table("(?) AS ledger", union)
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("ledger.status = ?", status)
	}
	if userID := strings.TrimSpace(c.Query("user_id")); userID != "" {
		query = query.Where("ledger.user_id = ? OR ledger.oauth2_id = ? OR ledger.user_display_id = ?", int64FromString(userID), int64FromString(userID), userID)
	}
	if strings.EqualFold(strings.TrimSpace(c.Query("anomaly")), "only") {
		query = query.Where("(ledger.status IN ? OR ledger.last_error IS NOT NULL OR ledger.attempts > 1 OR ledger.user_created_at IS NULL)", []string{"unknown", "failed", "compensated"})
	}
	if like, ok := logKeywordLike(c.Query("keyword")); ok {
		query = query.Where(
			"ledger.operation_key LIKE ? OR ledger.reason LIKE ? OR COALESCE(ledger.last_error, '') LIKE ? OR COALESCE(ledger.user_display_id, '') LIKE ? OR COALESCE(ledger.nickname, '') LIKE ? OR COALESCE(ledger.post_title, '') LIKE ? OR COALESCE(ledger.counterparty_display_id, '') LIKE ? OR COALESCE(ledger.counterparty_nickname, '') LIKE ? OR "+logTextCast(h.DB, "ledger.oauth2_id")+" LIKE ?",
			like, like, like, like, like, like, like, like, like,
		)
	}
	var total int64
	if err := query.Count(&total).Error; writeDBError(c, err, "") {
		return
	}
	var rows []adminBalanceAuditRow
	err := query.
		Select("ledger.*").
		Order("ledger.created_at DESC, ledger.id DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Scan(&rows).Error
	if writeDBError(c, err, "") {
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		anomaly := balanceAuditAnomaly(domain.ExternalBalanceTransaction{
			Status: row.Status, Attempts: row.Attempts, LastError: row.LastError,
		}) || row.UserCreatedAt == nil
		items = append(items, gin.H{
			"id":                   row.ID,
			"ledger_source":        row.LedgerSource,
			"operation_key":        row.OperationKey,
			"user_id":              row.UserID,
			"user_display_id":      row.UserDisplayID,
			"nickname":             row.Nickname,
			"oauth2_id":            row.OAuth2ID,
			"amount":               row.Amount,
			"reason":               row.Reason,
			"status":               row.Status,
			"remote_balance_after": row.RemoteBalanceAfter,
			"compensation_amount":  row.CompensationAmount,
			"attempts":             row.Attempts,
			"last_error":           row.LastError,
			"post_id":              row.PostID,
			"post_title":           row.PostTitle,
			"purchase_id":          row.PurchaseID,
			"entry_role":           normalizedLedgerEntryRole(row.EntryRole, "", row.Amount),
			"payment_method":       row.PaymentMethod,
			"platform_fee":         row.PlatformFee,
			"counterparty": gin.H{
				"user_id":         row.CounterpartyUserID,
				"user_display_id": row.CounterpartyDisplayID,
				"nickname":        row.CounterpartyNickname,
			},
			"is_anomaly":   anomaly,
			"created_at":   row.CreatedAt,
			"updated_at":   row.UpdatedAt,
			"applied_at":   row.AppliedAt,
			"completed_at": row.CompletedAt,
		})
	}
	writeSuccess(c, matrixMsgOK, gin.H{
		"items":      items,
		"pagination": matrixPagination(page, limit, total),
	})
}

func balanceAuditAnomaly(row domain.ExternalBalanceTransaction) bool {
	status := strings.ToLower(strings.TrimSpace(row.Status))
	return status == "unknown" || status == "failed" || status == "compensated" ||
		row.LastError != nil || row.Attempts > 1
}

func logTextCast(db *gorm.DB, column string) string {
	if db != nil && db.Dialector != nil && db.Dialector.Name() == "mysql" {
		return "CAST(" + column + " AS CHAR)"
	}
	return "CAST(" + column + " AS TEXT)"
}
