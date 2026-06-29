package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/http/response"
)

func (h NativeHandlers) adminDashboardOverview(c *gin.Context) {
	now := time.Now()
	today := startOfDay(now)
	yesterday := today.AddDate(0, 0, -1)
	count := func(table string, where string, args ...any) int64 {
		var total int64
		query := h.DB.WithContext(c.Request.Context()).Table(table)
		if where != "" {
			query = query.Where(where, args...)
		}
		_ = query.Count(&total).Error
		return total
	}
	sum := func(table string, column string, where string, args ...any) float64 {
		var row struct {
			Total float64 `gorm:"column:total"`
		}
		query := h.DB.WithContext(c.Request.Context()).Table(table).Select("COALESCE(SUM(" + column + "), 0) AS total")
		if where != "" {
			query = query.Where(where, args...)
		}
		_ = query.Scan(&row).Error
		return row.Total
	}
	totalUsers := count("users", "")
	totalPosts := count("posts", "is_draft = ?", false)
	totalComments := count("comments", "")
	totalReports := count("reports", "")
	totalFeedback := count("feedback", "")
	totalAnnouncements := count("announcements", "")

	writeSuccess(c, matrixMsgOK, gin.H{
		"generated_at": now.UTC().Format(time.RFC3339),
		"metrics": []gin.H{
			{"key": "users", "label": "总用户数", "value": totalUsers, "delta": percentDelta(count("users", "created_at >= ?", today), count("users", "created_at >= ? AND created_at < ?", yesterday, today)), "tone": "red"},
			{"key": "posts", "label": "内容数", "value": totalPosts, "delta": percentDelta(count("posts", "is_draft = ? AND created_at >= ?", false, today), count("posts", "is_draft = ? AND created_at >= ? AND created_at < ?", false, yesterday, today)), "tone": "blue"},
			{"key": "comments", "label": "评论数", "value": totalComments, "delta": percentDelta(count("comments", "created_at >= ?", today), count("comments", "created_at >= ? AND created_at < ?", yesterday, today)), "tone": "green"},
			{"key": "reports", "label": "举报数", "value": totalReports, "delta": percentDelta(count("reports", "created_at >= ?", today), count("reports", "created_at >= ? AND created_at < ?", yesterday, today)), "tone": "amber"},
			{"key": "feedback", "label": "反馈数", "value": totalFeedback, "delta": percentDelta(count("feedback", "created_at >= ?", today), count("feedback", "created_at >= ? AND created_at < ?", yesterday, today)), "tone": "purple"},
			{"key": "announcements", "label": "公告数", "value": totalAnnouncements, "delta": 0, "tone": "slate"},
		},
		"pending": gin.H{
			"content_review": count("audit", "type IN ? AND status = ?", []int{3, 4}, 0),
			"audit":          count("audit", "type IN ? AND status = ?", []int{1, 2}, 0),
			"reports":        count("reports", "status = ?", "pending"),
			"withdraw":       count("withdraw_orders", "status = ?", "pending"),
			"feedback":       count("feedback", "status <> ?", "resolved"),
			"missing_covers": count("post_videos", "(cover_url IS NULL OR cover_url = '')"),
		},
		"finance": gin.H{
			"creator_total_earnings": sum("creator_earnings", "total_earnings", ""),
			"creator_balance":        sum("creator_earnings", "balance", ""),
			"withdraw_pending":       sum("withdraw_orders", "amount", "status = ?", "pending"),
			"withdraw_paid":          sum("withdraw_orders", "amount", "status = ?", "paid"),
		},
		"statuses": []gin.H{
			{"key": "ai_review", "label": "AI 审核", "value": ternaryAny(h.Settings != nil && (h.Settings.Bool("ai_username_review_enabled") || h.Settings.Bool("ai_content_review_enabled")), "开启", "关闭"), "tone": ternaryAny(h.Settings != nil && (h.Settings.Bool("ai_username_review_enabled") || h.Settings.Bool("ai_content_review_enabled")), "green", "slate"), "description": "用户名与内容自动审核"},
			{"key": "guest_access", "label": "访客限制", "value": ternaryAny(h.Settings != nil && h.Settings.Bool("guest_access_restricted"), "限制", "开放"), "tone": ternaryAny(h.Settings != nil && h.Settings.Bool("guest_access_restricted"), "amber", "green"), "description": "笔记、视频游客访问策略"},
			{"key": "database", "label": "数据库", "value": "正常", "tone": "green", "description": "当前请求已连接数据库"},
		},
	})
}

func (h NativeHandlers) adminDashboardTrends(c *gin.Context) {
	days := min(positiveIntQuery(c, "days", 7), 30)
	if days < 1 {
		days = 7
	}
	today := startOfDay(time.Now())
	labels := make([]string, 0, days)
	users := make([]int64, 0, days)
	posts := make([]int64, 0, days)
	comments := make([]int64, 0, days)
	reports := make([]int64, 0, days)
	income := make([]float64, 0, days)

	for i := days - 1; i >= 0; i-- {
		start := today.AddDate(0, 0, -i)
		end := start.AddDate(0, 0, 1)
		labels = append(labels, start.Format("2006-01-02"))
		users = append(users, h.adminCountRange(c, "users", "created_at", start, end, ""))
		posts = append(posts, h.adminCountRange(c, "posts", "created_at", start, end, "is_draft = false"))
		comments = append(comments, h.adminCountRange(c, "comments", "created_at", start, end, ""))
		reports = append(reports, h.adminCountRange(c, "reports", "created_at", start, end, ""))
		income = append(income, h.adminSumRange(c, "creator_earnings_log", "amount", "created_at", start, end, "amount > 0"))
	}

	writeSuccess(c, matrixMsgOK, gin.H{
		"labels":   labels,
		"users":    users,
		"posts":    posts,
		"comments": comments,
		"reports":  reports,
		"income":   income,
	})
}

func (h NativeHandlers) adminDashboardHotContent(c *gin.Context) {
	var rows []struct {
		ID            int64     `gorm:"column:id"`
		Title         string    `gorm:"column:title"`
		Type          int       `gorm:"column:type"`
		ViewCount     int64     `gorm:"column:view_count"`
		LikeCount     int       `gorm:"column:like_count"`
		CollectCount  int       `gorm:"column:collect_count"`
		CommentCount  int       `gorm:"column:comment_count"`
		CreatedAt     time.Time `gorm:"column:created_at"`
		Nickname      *string   `gorm:"column:nickname"`
		UserDisplayID *string   `gorm:"column:user_display_id"`
		CoverURL      *string   `gorm:"column:cover_url"`
	}
	err := h.DB.WithContext(c.Request.Context()).
		Table("posts p").
		Select(`p.id, p.title, p.type, p.view_count, p.like_count, p.collect_count, p.comment_count, p.created_at,
			u.nickname, u.user_id AS user_display_id,
			COALESCE((SELECT image_url FROM post_images pi WHERE pi.post_id = p.id ORDER BY pi.id ASC LIMIT 1), (SELECT cover_url FROM post_videos pv WHERE pv.post_id = p.id ORDER BY pv.id ASC LIMIT 1)) AS cover_url`).
		Joins("LEFT JOIN users u ON u.id = p.user_id").
		Where("p.is_draft = ?", false).
		Order("(p.like_count * 5 + p.collect_count * 4 + p.comment_count * 6 + p.view_count) DESC, p.created_at DESC").
		Limit(8).
		Scan(&rows).Error
	if writeDBError(c, err, "") {
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, gin.H{
			"id":              row.ID,
			"title":           row.Title,
			"type":            row.Type,
			"view_count":      row.ViewCount,
			"like_count":      row.LikeCount,
			"collect_count":   row.CollectCount,
			"comment_count":   row.CommentCount,
			"created_at":      row.CreatedAt,
			"nickname":        row.Nickname,
			"user_display_id": row.UserDisplayID,
			"cover_url":       h.signFileURLPtr(row.CoverURL),
			"heat":            row.ViewCount + int64(row.LikeCount*5+row.CollectCount*4+row.CommentCount*6),
		})
	}
	writeSuccess(c, matrixMsgOK, gin.H{"items": items})
}

func (h NativeHandlers) adminCountRange(c *gin.Context, table string, column string, start time.Time, end time.Time, extra string) int64 {
	var total int64
	query := h.DB.WithContext(c.Request.Context()).Table(table).Where(column+" >= ? AND "+column+" < ?", start, end)
	if extra != "" {
		query = query.Where(extra)
	}
	_ = query.Count(&total).Error
	return total
}

func (h NativeHandlers) adminSumRange(c *gin.Context, table string, sumColumn string, timeColumn string, start time.Time, end time.Time, extra string) float64 {
	var row struct {
		Total float64 `gorm:"column:total"`
	}
	query := h.DB.WithContext(c.Request.Context()).Table(table).Select("COALESCE(SUM("+sumColumn+"), 0) AS total").Where(timeColumn+" >= ? AND "+timeColumn+" < ?", start, end)
	if extra != "" {
		query = query.Where(extra)
	}
	_ = query.Scan(&row).Error
	return row.Total
}

func startOfDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func percentDelta(today int64, yesterday int64) float64 {
	if yesterday == 0 {
		if today == 0 {
			return 0
		}
		return 100
	}
	value := (float64(today-yesterday) / float64(yesterday)) * 100
	rounded, _ := strconv.ParseFloat(strconv.FormatFloat(value, 'f', 1, 64), 64)
	return rounded
}

func (h NativeHandlers) AdminDashboardOverview(c *gin.Context) {
	if _, ok := h.requireMatrixAdmin(c); !ok || !h.requireDB(c) {
		return
	}
	h.adminDashboardOverview(c)
}

func (h NativeHandlers) AdminDashboardTrends(c *gin.Context) {
	if _, ok := h.requireMatrixAdmin(c); !ok || !h.requireDB(c) {
		return
	}
	h.adminDashboardTrends(c)
}

func (h NativeHandlers) AdminDashboardHotContent(c *gin.Context) {
	if _, ok := h.requireMatrixAdmin(c); !ok || !h.requireDB(c) {
		return
	}
	h.adminDashboardHotContent(c)
}

func (h NativeHandlers) AdminDashboardMissing(c *gin.Context) {
	response.JSON(c, http.StatusNotFound, response.CodeNotFound, "admin dashboard route not found", nil)
}
