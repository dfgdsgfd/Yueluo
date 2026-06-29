package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
)

type profileAwardTask struct {
	taskType string
	reason   string
}

func (h NativeHandlers) awardProfileTasksBestEffort(c *gin.Context, userID int64, tasks []profileAwardTask) []repositories.AwardResult {
	awards := make([]repositories.AwardResult, 0, len(tasks))
	seen := map[string]bool{}
	for _, task := range tasks {
		if task.taskType == "" || seen[task.taskType] {
			continue
		}
		seen[task.taskType] = true
		award := h.awardPointsBestEffort(c, userID, task.taskType, userID, task.reason)
		if award != nil && award.Awarded {
			awards = append(awards, *award)
		}
	}
	return awards
}

func (h NativeHandlers) usersDelete(c *gin.Context, id string, currentUserID int64) {
	target, ok := h.userByDisplayID(c, id)
	if !ok {
		return
	}
	if currentUserID != target.ID {
		response.JSON(c, http.StatusForbidden, response.CodeForbidden, "只能删除自己的账号", nil)
		return
	}
	if err := h.DB.WithContext(c.Request.Context()).Model(&domain.User{}).Where("id = ?", target.ID).Update("is_active", false).Error; writeDBError(c, err, "") {
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "账号删除成功", "success": true})
}

func (h NativeHandlers) usersList(c *gin.Context) {
	page, limit, offset := pageLimit(c, 20)
	query := h.DB.WithContext(c.Request.Context()).Model(&domain.User{})
	var total int64
	if err := query.Count(&total).Error; writeDBError(c, err, "") {
		return
	}
	var users []domain.User
	if err := query.Select("id", "user_id", "nickname", "avatar", "bio", "location", "follow_count", "fans_count", "like_count", "created_at", "verified").Order("created_at DESC").Offset(offset).Limit(limit).Find(&users).Error; writeDBError(c, err, "") {
		return
	}
	items, err := h.usersDecorated(c, users, currentUserID(c))
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	writeSuccess(c, matrixMsgOK, gin.H{"users": items, "pagination": matrixPagination(page, limit, total)})
}

func (h NativeHandlers) userByDisplayID(c *gin.Context, id string) (domain.User, bool) {
	var user domain.User
	query := h.DB.WithContext(c.Request.Context())
	err := query.Where("user_id = ?", id).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if numeric, parseErr := strconv.ParseInt(id, 10, 64); parseErr == nil {
			err = query.Where("id = ?", numeric).First(&user).Error
		}
	}
	if writeDBError(c, err, "用户不存在") {
		return domain.User{}, false
	}
	return user, true
}

func (h NativeHandlers) usersByIDs(c *gin.Context, ids []int64) ([]domain.User, error) {
	if len(ids) == 0 {
		return []domain.User{}, nil
	}
	var users []domain.User
	err := h.DB.WithContext(c.Request.Context()).Where("id IN ?", uniqueInt64Local(ids)).Find(&users).Error
	return users, err
}

func (h NativeHandlers) usersDecorated(c *gin.Context, users []domain.User, currentUserID int64) ([]gin.H, error) {
	out := make([]gin.H, 0, len(users))
	ids := make([]int64, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.ID)
	}
	postCounts, err := h.postCounts(c, ids)
	if err != nil {
		return nil, err
	}
	following := map[int64]bool{}
	followers := map[int64]bool{}
	if currentUserID != 0 && len(ids) > 0 {
		var rows []domain.Follow
		if err := h.DB.WithContext(c.Request.Context()).Where("follower_id = ? AND following_id IN ?", currentUserID, ids).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			following[row.FollowingID] = true
		}
		rows = nil
		if err := h.DB.WithContext(c.Request.Context()).Where("follower_id IN ? AND following_id = ?", ids, currentUserID).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			followers[row.FollowerID] = true
		}
	}
	for _, user := range users {
		item := gin.H{
			"id":           int(user.ID),
			"user_id":      user.UserID,
			"nickname":     user.Nickname,
			"avatar":       h.signFileURLPtr(user.Avatar),
			"bio":          user.Bio,
			"location":     user.Location,
			"follow_count": user.FollowCount,
			"fans_count":   user.FansCount,
			"like_count":   user.LikeCount,
			"created_at":   user.CreatedAt,
			"verified":     user.Verified,
			"post_count":   postCounts[user.ID],
			"isFollowing":  following[user.ID],
			"isMutual":     following[user.ID] && followers[user.ID],
			"buttonType":   buttonType(currentUserID, user.ID, following[user.ID], followers[user.ID]),
		}
		out = append(out, item)
	}
	return out, nil
}

func (h NativeHandlers) postCounts(c *gin.Context, ids []int64) (map[int64]int64, error) {
	out := map[int64]int64{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		UserID int64 `gorm:"column:user_id"`
		Count  int64 `gorm:"column:count"`
	}
	err := h.DB.WithContext(c.Request.Context()).Model(&domain.Post{}).
		Select("user_id, COUNT(*) AS count").
		Where("user_id IN ? AND is_draft = ?", uniqueInt64Local(ids), false).
		Group("user_id").Scan(&rows).Error
	for _, row := range rows {
		out[row.UserID] = row.Count
	}
	return out, err
}

func (h NativeHandlers) postsForIDs(c *gin.Context, ids []int64, currentUserID int64) ([]gin.H, error) {
	ids = uniqueInt64Local(ids)
	if len(ids) == 0 {
		return []gin.H{}, nil
	}
	var posts []domain.Post
	if err := h.DB.WithContext(c.Request.Context()).Where("id IN ? AND is_draft = ?", ids, false).Find(&posts).Error; err != nil {
		return nil, err
	}
	bundles, err := repositories.NewContentRepository(h.DB).PostBundlesForMatrix(c.Request.Context(), posts)
	if err != nil {
		return nil, err
	}
	purchased, liked, collected, err := h.postInteractionSets(c, currentUserID, ids)
	if err != nil {
		return nil, err
	}
	byID := map[int64]gin.H{}
	for _, bundle := range bundles {
		byID[bundle.Post.ID] = h.postListResponse(bundle, currentUserID, purchased[bundle.Post.ID], liked[bundle.Post.ID], collected[bundle.Post.ID])
	}
	out := make([]gin.H, 0, len(ids))
	for _, id := range ids {
		if post, ok := byID[id]; ok {
			out = append(out, post)
		}
	}
	return out, nil
}

func (h NativeHandlers) followStatusData(c *gin.Context, currentUserID int64, targetID int64) (gin.H, error) {
	if currentUserID == 0 {
		return gin.H{"followed": false, "isFollowing": false, "isMutual": false, "buttonType": "follow"}, nil
	}
	var rows []domain.Follow
	err := h.DB.WithContext(c.Request.Context()).
		Where("(follower_id = ? AND following_id = ?) OR (follower_id = ? AND following_id = ?)", currentUserID, targetID, targetID, currentUserID).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	isFollowing := false
	isFollower := false
	for _, row := range rows {
		if row.FollowerID == currentUserID && row.FollowingID == targetID {
			isFollowing = true
		}
		if row.FollowerID == targetID && row.FollowingID == currentUserID {
			isFollower = true
		}
	}
	return gin.H{"followed": isFollowing, "isFollowing": isFollowing, "isMutual": isFollowing && isFollower, "buttonType": buttonType(currentUserID, targetID, isFollowing, isFollower)}, nil
}

func (h NativeHandlers) blockStatus(c *gin.Context, currentUserID int64, targetID int64) (bool, bool, error) {
	if currentUserID == 0 {
		return false, false, nil
	}
	var rows []domain.Blacklist
	err := h.DB.WithContext(c.Request.Context()).
		Where("(blocker_id = ? AND blocked_id = ?) OR (blocker_id = ? AND blocked_id = ?)", currentUserID, targetID, targetID, currentUserID).
		Find(&rows).Error
	if err != nil {
		return false, false, err
	}
	blocked := false
	blockedBy := false
	for _, row := range rows {
		if row.BlockerID == currentUserID {
			blocked = true
		}
		if row.BlockerID == targetID {
			blockedBy = true
		}
	}
	return blocked, blockedBy, nil
}

func (h NativeHandlers) removeMutualFollows(tx *gorm.DB, a int64, b int64) error {
	var rows []domain.Follow
	if err := tx.Where("(follower_id = ? AND following_id = ?) OR (follower_id = ? AND following_id = ?)", a, b, b, a).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := tx.Delete(&row).Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.User{}).Where("id = ? AND follow_count > 0", row.FollowerID).UpdateColumn("follow_count", gorm.Expr("follow_count - 1")).Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.User{}).Where("id = ? AND fans_count > 0", row.FollowingID).UpdateColumn("fans_count", gorm.Expr("fans_count - 1")).Error; err != nil {
			return err
		}
	}
	return nil
}

func buttonType(currentUserID, targetID int64, isFollowing, isFollower bool) string {
	switch {
	case currentUserID != 0 && currentUserID == targetID:
		return "self"
	case isFollowing && isFollower:
		return "mutual"
	case isFollowing:
		return "unfollow"
	case isFollower:
		return "back"
	default:
		return "follow"
	}
}

func privacyMap(user domain.User) gin.H {
	return gin.H{
		"privacy_birthday":        user.PrivacyBirthday,
		"privacy_age":             user.PrivacyAge,
		"privacy_zodiac":          user.PrivacyZodiac,
		"privacy_mbti":            user.PrivacyMBTI,
		"privacy_custom_fields":   jsonValue(user.PrivacyCustomFields),
		"ai_auto_comment_enabled": user.AIAutoCommentEnabled,
	}
}

func personalityMap(user domain.User, owner bool) gin.H {
	custom := jsonValue(user.CustomFields)
	if !owner {
		privacy, _ := jsonValue(user.PrivacyCustomFields).(map[string]any)
		if fields, ok := custom.(map[string]any); ok && privacy != nil {
			filtered := gin.H{}
			for key, value := range fields {
				if raw, ok := privacy[key]; ok {
					if visible, ok := boolFromAny(raw); ok && !visible {
						continue
					}
				}
				filtered[key] = value
			}
			custom = filtered
		}
	}
	return gin.H{
		"gender":        user.Gender,
		"zodiac_sign":   ternaryAny(owner || user.PrivacyZodiac, user.ZodiacSign, nil),
		"mbti":          ternaryAny(owner || user.PrivacyMBTI, user.MBTI, nil),
		"education":     user.Education,
		"major":         user.Major,
		"interests":     jsonValue(user.Interests),
		"custom_fields": custom,
	}
}

func (h NativeHandlers) auditMap(row domain.Audit) gin.H {
	return gin.H{
		"id":           int(row.ID),
		"user_id":      int(row.UserID),
		"type":         row.Type,
		"content":      row.Content,
		"status":       row.Status,
		"reason":       row.Reason,
		"audit_result": h.signVerificationAuditResult(jsonValue(row.AuditResult)),
		"created_at":   row.CreatedAt,
		"audit_time":   row.AuditTime,
	}
}

func historyPostIDs(rows []domain.BrowsingHistory) []int64 {
	out := make([]int64, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.PostID)
	}
	return out
}

func collectionPostIDs(rows []domain.Collection) []int64 {
	out := make([]int64, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.PostID)
	}
	return out
}

func likePostIDs(rows []domain.Like) []int64 {
	out := make([]int64, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.TargetID)
	}
	return out
}

func sanitizedStringMap(value any, maxKey int, maxValue int) gin.H {
	raw, ok := value.(map[string]any)
	if !ok {
		return gin.H{}
	}
	out := gin.H{}
	for key, item := range raw {
		key = sanitizePlainSubmittedText(key)
		if key == "" || len([]rune(key)) > maxKey {
			continue
		}
		text := sanitizePlainSubmittedText(toString(item))
		if len([]rune(text)) > maxValue {
			continue
		}
		out[key] = text
	}
	return out
}

func sanitizedStringSlice(value any, maxItems int, maxValue int) []string {
	raw := parseStringSlice(value)
	out := make([]string, 0, len(raw))
	seen := map[string]bool{}
	for _, item := range raw {
		text := sanitizePlainSubmittedText(item)
		if text == "" || seen[text] {
			continue
		}
		if maxValue > 0 && len([]rune(text)) > maxValue {
			continue
		}
		seen[text] = true
		out = append(out, text)
		if maxItems > 0 && len(out) >= maxItems {
			break
		}
	}
	return out
}

func zodiacSign(t time.Time) string {
	month := int(t.Month())
	day := t.Day()
	switch {
	case (month == 1 && day >= 20) || (month == 2 && day <= 18):
		return "水瓶座"
	case (month == 2 && day >= 19) || (month == 3 && day <= 20):
		return "双鱼座"
	case (month == 3 && day >= 21) || (month == 4 && day <= 19):
		return "白羊座"
	case (month == 4 && day >= 20) || (month == 5 && day <= 20):
		return "金牛座"
	case (month == 5 && day >= 21) || (month == 6 && day <= 21):
		return "双子座"
	case (month == 6 && day >= 22) || (month == 7 && day <= 22):
		return "巨蟹座"
	case (month == 7 && day >= 23) || (month == 8 && day <= 22):
		return "狮子座"
	case (month == 8 && day >= 23) || (month == 9 && day <= 22):
		return "处女座"
	case (month == 9 && day >= 23) || (month == 10 && day <= 23):
		return "天秤座"
	case (month == 10 && day >= 24) || (month == 11 && day <= 22):
		return "天蝎座"
	case (month == 11 && day >= 23) || (month == 12 && day <= 21):
		return "射手座"
	default:
		return "摩羯座"
	}
}

func ternaryAny(cond bool, yes any, no any) any {
	if cond {
		return yes
	}
	return no
}

func uniqueInt64Local(values []int64) []int64 {
	seen := map[int64]bool{}
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if value == 0 || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
