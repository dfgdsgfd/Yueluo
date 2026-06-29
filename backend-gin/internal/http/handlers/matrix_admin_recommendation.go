package handlers

import (
	"maps"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
)

type postRecommendConfigRow struct {
	ID                  int        `gorm:"column:id"`
	PostID              int64      `gorm:"column:post_id"`
	BoostScore          float64    `gorm:"column:boost_score"`
	IsPinned            bool       `gorm:"column:is_pinned"`
	IsSuppressed        bool       `gorm:"column:is_suppressed"`
	TargetUserID        *int64     `gorm:"column:target_user_id"`
	Reason              *string    `gorm:"column:reason"`
	IsActive            bool       `gorm:"column:is_active"`
	StartTime           *time.Time `gorm:"column:start_time"`
	EndTime             *time.Time `gorm:"column:end_time"`
	CreatedAt           time.Time  `gorm:"column:created_at"`
	UpdatedAt           *time.Time `gorm:"column:updated_at"`
	PostIDValue         *int64     `gorm:"column:post_id_value"`
	PostTitle           *string    `gorm:"column:post_title"`
	PostType            *int       `gorm:"column:post_type"`
	PostLikeCount       *int       `gorm:"column:post_like_count"`
	PostCollectCount    *int       `gorm:"column:post_collect_count"`
	PostViewCount       *int64     `gorm:"column:post_view_count"`
	PostUserID          *int64     `gorm:"column:post_user_id"`
	PostUserNickname    *string    `gorm:"column:post_user_nickname"`
	PostUserDisplayID   *string    `gorm:"column:post_user_display_id"`
	TargetUserIDValue   *int64     `gorm:"column:target_user_id_value"`
	TargetUserNickname  *string    `gorm:"column:target_user_nickname"`
	TargetUserDisplayID *string    `gorm:"column:target_user_display_id"`
	TargetUserAvatar    *string    `gorm:"column:target_user_avatar"`
	PostUserAvatar      *string    `gorm:"column:post_user_avatar"`
	PostCommentCount    *int       `gorm:"column:post_comment_count"`
	PostQualityLevel    *string    `gorm:"column:post_quality_level"`
	PostQualityReward   *float64   `gorm:"column:post_quality_reward"`
	PostQualityMarkedAt *time.Time `gorm:"column:post_quality_marked_at"`
	PostCreatedAt       *time.Time `gorm:"column:post_created_at"`
	PostIsDraft         *bool      `gorm:"column:post_is_draft"`
	PostCategoryName    *string    `gorm:"column:post_category_name"`
	PostFirstImageURL   *string    `gorm:"column:post_first_image_url"`
	PostFirstVideoCover *string    `gorm:"column:post_first_video_cover"`
	PostContent         *string    `gorm:"column:post_content"`
}

type recommendConfigRow struct {
	ID                 int        `gorm:"column:id"`
	UserID             *int64     `gorm:"column:user_id"`
	LikeWeight         float64    `gorm:"column:like_weight"`
	CollectWeight      float64    `gorm:"column:collect_weight"`
	ViewWeight         float64    `gorm:"column:view_weight"`
	CategoryWeight     float64    `gorm:"column:category_weight"`
	TagWeight          float64    `gorm:"column:tag_weight"`
	FollowingWeight    float64    `gorm:"column:following_weight"`
	MutualFollowWeight float64    `gorm:"column:mutual_follow_weight"`
	PopularityWeight   float64    `gorm:"column:popularity_weight"`
	InterestWeight     float64    `gorm:"column:interest_weight"`
	TimeDecayHalfLife  int        `gorm:"column:time_decay_half_life"`
	IsActive           bool       `gorm:"column:is_active"`
	CreatedAt          time.Time  `gorm:"column:created_at"`
	UpdatedAt          *time.Time `gorm:"column:updated_at"`
	UserIDValue        *int64     `gorm:"column:user_id_value"`
	UserDisplayID      *string    `gorm:"column:user_display_id"`
	UserNickname       *string    `gorm:"column:user_nickname"`
	UserAvatar         *string    `gorm:"column:user_avatar"`
}

type postQualityRow struct {
	ID              int64      `gorm:"column:id"`
	UserID          int64      `gorm:"column:user_id"`
	Title           string     `gorm:"column:title"`
	Content         string     `gorm:"column:content"`
	Type            int        `gorm:"column:type"`
	ViewCount       int64      `gorm:"column:view_count"`
	LikeCount       int        `gorm:"column:like_count"`
	CollectCount    int        `gorm:"column:collect_count"`
	CommentCount    int        `gorm:"column:comment_count"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	IsDraft         bool       `gorm:"column:is_draft"`
	QualityLevel    string     `gorm:"column:quality_level"`
	QualityMarkedAt *time.Time `gorm:"column:quality_marked_at"`
	QualityReward   *float64   `gorm:"column:quality_reward"`
	UserDisplayID   *string    `gorm:"column:user_display_id"`
	Nickname        *string    `gorm:"column:nickname"`
	ImageCover      *string    `gorm:"column:image_cover"`
	VideoCover      *string    `gorm:"column:video_cover"`
}

func (h NativeHandlers) adminRecommendationPostConfigs(c *gin.Context) {
	id := matrixParam(c, "id")
	switch matrixMethod(c) {
	case http.MethodGet:
		h.adminRecommendationPostConfigList(c)
	case http.MethodPost:
		h.adminRecommendationPostConfigCreate(c)
	case http.MethodPut:
		h.adminRecommendationPostConfigUpdate(c, id)
	case http.MethodDelete:
		h.adminRecommendationPostConfigDelete(c, id)
	default:
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "admin route not found", nil)
	}
}

func (h NativeHandlers) adminRecommendationPostConfigList(c *gin.Context) {
	page, limit, offset := pageLimit(c, 20)
	build := func() *gorm.DB {
		query := h.DB.WithContext(c.Request.Context()).
			Table("post_recommend_configs prc").
			Joins("LEFT JOIN posts p ON p.id = prc.post_id").
			Joins("LEFT JOIN users u ON u.id = p.user_id").
			Joins("LEFT JOIN users tu ON tu.id = prc.target_user_id")
		if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
			like := "%" + keyword + "%"
			if postID, err := strconv.ParseInt(keyword, 10, 64); err == nil && postID > 0 {
				query = query.Where("prc.post_id = ? OR p.title LIKE ? OR p.content LIKE ? OR u.nickname LIKE ? OR u.user_id LIKE ? OR prc.reason LIKE ?", postID, like, like, like, like, like)
			} else {
				query = query.Where("p.title LIKE ? OR p.content LIKE ? OR u.nickname LIKE ? OR u.user_id LIKE ? OR prc.reason LIKE ?", like, like, like, like, like)
			}
		}
		return query
	}
	var total int64
	if err := build().Count(&total).Error; writeDBError(c, err, "") {
		return
	}
	var rows []postRecommendConfigRow
	err := build().Select(`prc.id, prc.post_id, prc.boost_score, prc.is_pinned, prc.is_suppressed, prc.target_user_id, prc.reason, prc.is_active, prc.start_time, prc.end_time, prc.created_at, prc.updated_at,
		p.id AS post_id_value, p.title AS post_title, p.content AS post_content, p.type AS post_type, p.like_count AS post_like_count, p.collect_count AS post_collect_count, p.view_count AS post_view_count,
		(SELECT image_url FROM post_images pi WHERE pi.post_id = p.id ORDER BY pi.id ASC LIMIT 1) AS post_first_image_url,
		(SELECT cover_url FROM post_videos pv WHERE pv.post_id = p.id ORDER BY pv.id ASC LIMIT 1) AS post_first_video_cover,
		u.id AS post_user_id, u.nickname AS post_user_nickname, u.user_id AS post_user_display_id,
		tu.id AS target_user_id_value, tu.nickname AS target_user_nickname, tu.user_id AS target_user_display_id, tu.avatar AS target_user_avatar`).
		Order("prc.created_at DESC").Offset(offset).Limit(limit).Scan(&rows).Error
	if writeDBError(c, err, "") {
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, h.postRecommendConfigMap(row))
	}
	writeSuccess(c, matrixMsgOK, gin.H{"items": items, "total": total, "page": page, "limit": limit, "pages": totalPages(page, limit, total)})
}

func (h NativeHandlers) adminRecommendationPostConfigCreate(c *gin.Context) {
	body := readBodyMap(c)
	postID, ok := int64FromAny(body["post_id"])
	if !ok || postID <= 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "帖子ID不能为空", nil)
		return
	}
	if !h.adminPostExists(c, postID) {
		return
	}
	var existing int64
	if err := h.DB.WithContext(c.Request.Context()).Model(&domain.PostRecommendConfig{}).Where("post_id = ?", postID).Count(&existing).Error; writeDBError(c, err, "") {
		return
	}
	if existing > 0 {
		response.JSON(c, http.StatusConflict, response.CodeConflict, "该帖子已有推荐配置，请使用更新接口", nil)
		return
	}
	row := postRecommendDataFromBody(body, true)
	row["post_id"] = postID
	now := time.Now()
	row["created_at"] = now
	row["updated_at"] = now
	if err := h.DB.WithContext(c.Request.Context()).Table("post_recommend_configs").Create(row).Error; writeDBError(c, err, "") {
		return
	}
	h.bumpCacheVersions(cacheScopePosts)
	created, ok := h.loadPostRecommendConfigByPost(c, postID)
	if !ok {
		return
	}
	writeSuccess(c, "帖子推荐配置已创建", h.postRecommendConfigMap(created))
}

func (h NativeHandlers) adminRecommendationPostConfigUpdate(c *gin.Context, id string) {
	row := postRecommendDataFromBody(readBodyMap(c), false)
	if len(row) == 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "没有提供任何可更新字段", nil)
		return
	}
	row["updated_at"] = time.Now()
	res := h.DB.WithContext(c.Request.Context()).Table("post_recommend_configs").Where("id = ?", id).Updates(row)
	if writeDBError(c, res.Error, "") {
		return
	}
	h.bumpCacheVersions(cacheScopePosts)
	updated, ok := h.loadPostRecommendConfigByID(c, id)
	if !ok {
		return
	}
	writeSuccess(c, "帖子推荐配置已更新", h.postRecommendConfigMap(updated))
}

func (h NativeHandlers) adminRecommendationPostConfigDelete(c *gin.Context, id string) {
	if err := h.DB.WithContext(c.Request.Context()).Where("id = ?", id).Delete(&domain.PostRecommendConfig{}).Error; writeDBError(c, err, "") {
		return
	}
	h.bumpCacheVersions(cacheScopePosts)
	writeSimpleSuccess(c, "帖子推荐配置已删除")
}

func (h NativeHandlers) adminRecommendationUserConfigs(c *gin.Context) {
	id := matrixParam(c, "id")
	switch matrixMethod(c) {
	case http.MethodGet:
		h.adminRecommendationUserConfigList(c)
	case http.MethodPost:
		h.adminRecommendationUserConfigSave(c)
	case http.MethodPut:
		h.adminRecommendationUserConfigUpdate(c, id)
	case http.MethodDelete:
		h.adminRecommendationUserConfigDelete(c, id)
	default:
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "admin route not found", nil)
	}
}

func (h NativeHandlers) adminRecommendationUserConfigList(c *gin.Context) {
	page, limit, offset := pageLimit(c, 20)
	build := func() *gorm.DB {
		query := h.DB.WithContext(c.Request.Context()).
			Table("recommend_configs rc").
			Joins("LEFT JOIN users u ON u.id = rc.user_id").
			Where("rc.user_id IS NOT NULL")
		if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
			query = query.Where("u.nickname LIKE ?", "%"+keyword+"%")
		}
		return query
	}
	var total int64
	if err := build().Count(&total).Error; writeDBError(c, err, "") {
		return
	}
	var rows []recommendConfigRow
	err := build().Select(`rc.id, rc.user_id, rc.like_weight, rc.collect_weight, rc.view_weight, rc.category_weight, rc.tag_weight, rc.following_weight, rc.mutual_follow_weight, rc.popularity_weight, rc.interest_weight, rc.time_decay_half_life, rc.is_active, rc.created_at, rc.updated_at,
		u.id AS user_id_value, u.user_id AS user_display_id, u.nickname AS user_nickname, u.avatar AS user_avatar`).
		Order("rc.created_at DESC").Offset(offset).Limit(limit).Scan(&rows).Error
	if writeDBError(c, err, "") {
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, h.recommendConfigMap(row))
	}
	writeSuccess(c, matrixMsgOK, gin.H{"items": items, "total": total, "page": page, "limit": limit, "pages": totalPages(page, limit, total)})
}

func (h NativeHandlers) adminRecommendationUserConfigSave(c *gin.Context) {
	body := readBodyMap(c)
	userID, ok := int64FromAny(body["user_id"])
	if !ok || userID <= 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "用户ID不能为空", nil)
		return
	}
	if !h.adminUserExists(c, userID) {
		return
	}
	row := recommendConfigDataFromBody(body)
	row["user_id"] = userID
	now := time.Now()
	row["updated_at"] = now
	createRow := maps.Clone(row)
	createRow["created_at"] = now
	if err := h.DB.WithContext(c.Request.Context()).Table("recommend_configs").Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(row),
	}).Create(createRow).Error; writeDBError(c, err, "") {
		return
	}
	h.bumpCacheVersions(cacheScopePosts)
	updated, ok := h.loadRecommendConfigByUser(c, userID)
	if !ok {
		return
	}
	writeSuccess(c, "用户推荐配置已保存", h.recommendConfigMap(updated))
}

func (h NativeHandlers) adminRecommendationUserConfigUpdate(c *gin.Context, id string) {
	row := recommendConfigDataFromBody(readBodyMap(c))
	delete(row, "user_id")
	if len(row) == 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "没有提供任何可更新字段", nil)
		return
	}
	row["updated_at"] = time.Now()
	res := h.DB.WithContext(c.Request.Context()).Table("recommend_configs").Where("id = ?", id).Updates(row)
	if writeDBError(c, res.Error, "") {
		return
	}
	h.bumpCacheVersions(cacheScopePosts)
	updated, ok := h.loadRecommendConfigByID(c, id)
	if !ok {
		return
	}
	writeSuccess(c, "用户推荐配置已更新", h.recommendConfigMap(updated))
}

func (h NativeHandlers) adminRecommendationUserConfigDelete(c *gin.Context, id string) {
	if err := h.DB.WithContext(c.Request.Context()).Where("id = ?", id).Delete(&domain.RecommendConfig{}).Error; writeDBError(c, err, "") {
		return
	}
	h.bumpCacheVersions(cacheScopePosts)
	writeSimpleSuccess(c, "用户推荐配置已删除")
}
