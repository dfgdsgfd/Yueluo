package repositories

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yuem-go/backend-gin/internal/domain"
)

type NotificationsRepository struct {
	db *gorm.DB
}

type NotificationListParams struct {
	Page   int
	Limit  int
	Type   string
	IsRead string
}

type NotificationBundle struct {
	Notification       domain.Notification
	Sender             *domain.User
	Comment            *domain.Comment
	Detail             *string
	GiftCardRedemption *GiftCardRedemptionNotification
	PostTitle          *string
	PostCover          *string
}

type GiftCardRedemptionNotification struct {
	ProductName string
	FaceValue   string
	PointsSpent float64
	Code        string
	RedeemedAt  time.Time
}

type SystemNotificationBundle struct {
	Notification domain.SystemNotification
	Confirmation *domain.SystemNotificationConfirmation
}

func NewNotificationsRepository(db *gorm.DB) NotificationsRepository {
	return NotificationsRepository{db: db}
}

const NotificationTypeGiftCardRedeemed = 21

func (r NotificationsRepository) Activities(ctx context.Context, page, limit int) (int64, []domain.SystemNotification, error) {
	now := time.Now()
	query := r.db.WithContext(ctx).Model(&domain.SystemNotification{}).
		Where("type = ? AND is_active = ?", "activity", true).
		Where("(start_time IS NULL OR start_time <= ?) AND (end_time IS NULL OR end_time >= ?)", now, now)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		if isOptionalSystemNotificationSchemaError(err) {
			return 0, []domain.SystemNotification{}, nil
		}
		return 0, nil, err
	}
	var activities []domain.SystemNotification
	err := query.Select("id", "title", "content", "type", "content_format", "image_url", "link_url", "show_popup", "start_time", "end_time", "created_at").
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&activities).Error
	if isOptionalSystemNotificationSchemaError(err) {
		return 0, []domain.SystemNotification{}, nil
	}
	return total, activities, err
}

func isOptionalSystemNotificationSchemaError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "system_notifications") {
		return strings.Contains(message, "no such table") ||
			strings.Contains(message, "does not exist") ||
			strings.Contains(message, "doesn't exist") ||
			strings.Contains(message, "unknown column") ||
			strings.Contains(message, "no such column")
	}
	if strings.Contains(message, "unknown column") ||
		strings.Contains(message, "no such column") ||
		strings.Contains(message, "does not exist") {
		for _, column := range []string{"content_format", "image_url", "link_url", "show_popup", "start_time", "end_time"} {
			if strings.Contains(message, column) {
				return true
			}
		}
	}
	return false
}

func (r NotificationsRepository) List(ctx context.Context, userID int64, params NotificationListParams) (int64, []NotificationBundle, error) {
	query := r.db.WithContext(ctx).Model(&domain.Notification{}).Where("user_id = ?", userID)
	blocked, err := r.blockedUserIDs(ctx, userID)
	if err != nil {
		return 0, nil, err
	}
	if len(blocked) > 0 {
		query = query.Where("sender_id NOT IN ?", blocked)
	}
	if strings.TrimSpace(params.Type) != "" {
		types := parseTypeFilter(params.Type)
		if len(types) == 1 {
			query = query.Where("type = ?", types[0])
		} else if len(types) > 1 {
			query = query.Where("type IN ?", types)
		}
	}
	if params.IsRead != "" {
		read := params.IsRead == "true" || params.IsRead == "1"
		query = query.Where("is_read = ?", read)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	var notifications []domain.Notification
	if err := query.Order("created_at DESC").Offset((params.Page - 1) * params.Limit).Limit(params.Limit).Find(&notifications).Error; err != nil {
		return 0, nil, err
	}
	bundles, err := r.enrichNotifications(ctx, notifications)
	return total, bundles, err
}

func (r NotificationsRepository) UnreadCounts(ctx context.Context, userID int64) (int64, int64, error) {
	var notificationCount int64
	if err := r.db.WithContext(ctx).Model(&domain.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Count(&notificationCount).Error; err != nil {
		return 0, 0, err
	}
	var systemCount int64
	err := r.db.WithContext(ctx).Model(&domain.SystemNotification{}).
		Where("is_active = ?", true).
		Where("NOT EXISTS (SELECT 1 FROM system_notification_confirmations c WHERE c.notification_id = system_notifications.id AND c.user_id = ?)", userID).
		Count(&systemCount).Error
	return notificationCount, systemCount, err
}

func (r NotificationsRepository) MarkAllRead(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).Model(&domain.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Update("is_read", true).Error
}

func (r NotificationsRepository) MarkRead(ctx context.Context, userID, id int64) error {
	var notification domain.Notification
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&notification).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&domain.Notification{}).Where("id = ?", id).Update("is_read", true).Error
}

func (r NotificationsRepository) Delete(ctx context.Context, userID, id int64) error {
	var notification domain.Notification
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&notification).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Delete(&notification).Error
}

func (r NotificationsRepository) SystemList(ctx context.Context, userID int64, page, limit int, notificationType string) (int64, []SystemNotificationBundle, error) {
	query := r.db.WithContext(ctx).Model(&domain.SystemNotification{}).
		Where("is_active = ?", true).
		Where("NOT EXISTS (SELECT 1 FROM system_notification_confirmations c WHERE c.notification_id = system_notifications.id AND c.user_id = ? AND c.is_dismissed = ?)", userID, true)
	if notificationType != "" {
		query = query.Where("type = ?", notificationType)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	var notifications []domain.SystemNotification
	if err := query.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&notifications).Error; err != nil {
		return 0, nil, err
	}
	confirmations, err := r.confirmationsByNotificationID(ctx, userID, notificationIDs(notifications))
	if err != nil {
		return 0, nil, err
	}
	bundles := make([]SystemNotificationBundle, 0, len(notifications))
	for _, notification := range notifications {
		bundles = append(bundles, SystemNotificationBundle{
			Notification: notification,
			Confirmation: confirmations[notification.ID],
		})
	}
	return total, bundles, nil
}

func (r NotificationsRepository) PopupSystem(ctx context.Context, userID int64) ([]domain.SystemNotification, error) {
	var notifications []domain.SystemNotification
	err := r.db.WithContext(ctx).Where("is_active = ? AND show_popup = ?", true, true).
		Where("NOT EXISTS (SELECT 1 FROM system_notification_confirmations c WHERE c.notification_id = system_notifications.id AND c.user_id = ?)", userID).
		Order("created_at DESC").
		Find(&notifications).Error
	return notifications, err
}

func (r NotificationsRepository) ConfirmSystem(ctx context.Context, userID, notificationID int64) error {
	if err := r.ensureSystemNotification(ctx, notificationID); err != nil {
		return err
	}
	confirmation := domain.SystemNotificationConfirmation{NotificationID: notificationID, UserID: userID}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "notification_id"}, {Name: "user_id"}},
		DoNothing: true,
	}).Create(&confirmation).Error
}

func (r NotificationsRepository) DismissSystem(ctx context.Context, userID, notificationID int64) error {
	if err := r.ensureSystemNotification(ctx, notificationID); err != nil {
		return err
	}
	confirmation := domain.SystemNotificationConfirmation{NotificationID: notificationID, UserID: userID, IsDismissed: true}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "notification_id"}, {Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{"is_dismissed": true}),
	}).Create(&confirmation).Error
}

func (r NotificationsRepository) ensureSystemNotification(ctx context.Context, id int64) error {
	var count int64
	if err := r.db.WithContext(ctx).Model(&domain.SystemNotification{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r NotificationsRepository) blockedUserIDs(ctx context.Context, userID int64) ([]int64, error) {
	var rows []domain.Blacklist
	if err := r.db.WithContext(ctx).Where("blocker_id = ?", userID).Select("blocked_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]int64, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.BlockedID)
	}
	return out, nil
}

func (r NotificationsRepository) enrichNotifications(ctx context.Context, notifications []domain.Notification) ([]NotificationBundle, error) {
	if len(notifications) == 0 {
		return []NotificationBundle{}, nil
	}
	senderIDs := make([]int64, 0, len(notifications))
	commentIDs := []int64{}
	redemptionIDs := []int64{}
	targetIDs := []int64{}
	for _, notification := range notifications {
		senderIDs = append(senderIDs, notification.SenderID)
		if notification.CommentID != nil {
			commentIDs = append(commentIDs, *notification.CommentID)
		}
		if notification.Type == NotificationTypeGiftCardRedeemed && notification.TargetID != nil {
			redemptionIDs = append(redemptionIDs, *notification.TargetID)
			continue
		}
		if notification.TargetID != nil {
			targetIDs = append(targetIDs, *notification.TargetID)
		}
	}
	senders, err := r.users(ctx, senderIDs)
	if err != nil {
		return nil, err
	}
	comments, err := r.comments(ctx, commentIDs)
	if err != nil {
		return nil, err
	}
	postInfo, err := r.postCoverInfo(ctx, targetIDs)
	if err != nil {
		return nil, err
	}
	redemptions, err := r.giftCardRedemptionDetails(ctx, redemptionIDs, giftCardNotificationUsers(notifications))
	if err != nil {
		return nil, err
	}
	bundles := make([]NotificationBundle, 0, len(notifications))
	for _, notification := range notifications {
		var title *string
		var cover *string
		var detail *string
		var giftCardRedemption *GiftCardRedemptionNotification
		if notification.TargetID != nil {
			if notification.Type == NotificationTypeGiftCardRedeemed {
				giftCardRedemption = redemptions[*notification.TargetID]
				if giftCardRedemption != nil {
					legacyDetail := giftCardRedemption.legacyDetail()
					detail = &legacyDetail
				}
			} else {
				info := postInfo[*notification.TargetID]
				title = info.Title
				cover = info.Cover
			}
		}
		var comment *domain.Comment
		if notification.CommentID != nil {
			comment = comments[*notification.CommentID]
		}
		bundles = append(bundles, NotificationBundle{
			Notification:       notification,
			Sender:             senders[notification.SenderID],
			Comment:            comment,
			Detail:             detail,
			GiftCardRedemption: giftCardRedemption,
			PostTitle:          title,
			PostCover:          cover,
		})
	}
	return bundles, nil
}

type postCoverInfo struct {
	Title *string
	Cover *string
}

func (r NotificationsRepository) postCoverInfo(ctx context.Context, ids []int64) (map[int64]postCoverInfo, error) {
	out := map[int64]postCoverInfo{}
	if len(ids) == 0 {
		return out, nil
	}
	var posts []domain.Post
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Select("id", "title").Find(&posts).Error; err != nil {
		return nil, err
	}
	var images []domain.PostImage
	if err := r.db.WithContext(ctx).
		Where("post_id IN ?", ids).
		Select("post_id", "image_url", "sort_order").
		Order("post_id ASC, sort_order ASC, id ASC").
		Find(&images).Error; err != nil {
		return nil, err
	}
	covers := map[int64]string{}
	for _, image := range images {
		if _, ok := covers[image.PostID]; !ok {
			covers[image.PostID] = image.ImageURL
		}
	}
	for _, post := range posts {
		title := post.Title
		info := postCoverInfo{Title: &title}
		if cover, ok := covers[post.ID]; ok {
			info.Cover = &cover
		}
		out[post.ID] = info
	}
	return out, nil
}

func (r NotificationsRepository) giftCardRedemptionDetails(ctx context.Context, ids []int64, userByRedemptionID map[int64]int64) (map[int64]*GiftCardRedemptionNotification, error) {
	out := map[int64]*GiftCardRedemptionNotification{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		ID          int64
		Code        string
		CreatedAt   time.Time
		Name        *string
		FaceValue   *string
		PointsSpent float64
		UserID      int64
	}
	err := r.db.WithContext(ctx).Table("gift_card_redemptions AS r").
		Select("r.id, r.user_id, r.code_snapshot AS code, r.created_at, r.points_spent, p.name, p.face_value").
		Joins("LEFT JOIN gift_card_products p ON p.id = r.product_id").
		Where("r.id IN ?", uniqueInt64(ids)).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		expectedUserID := userByRedemptionID[row.ID]
		if expectedUserID <= 0 || row.UserID != expectedUserID {
			continue
		}
		productName := strings.TrimSpace(stringPtrValue(row.Name))
		if productName == "" {
			productName = "礼品卡"
		}
		faceValue := strings.TrimSpace(stringPtrValue(row.FaceValue))
		if faceValue == "" {
			faceValue = "礼品卡"
		}
		out[row.ID] = &GiftCardRedemptionNotification{
			ProductName: productName,
			FaceValue:   faceValue,
			PointsSpent: row.PointsSpent,
			Code:        row.Code,
			RedeemedAt:  row.CreatedAt,
		}
	}
	return out, nil
}

func (detail GiftCardRedemptionNotification) legacyDetail() string {
	return fmt.Sprintf("你兑换的%s已发放。\n\n商品：%s\n面值：%s\n消耗积分：%s\n卡密：%s\n兑换时间：%s", detail.ProductName, detail.ProductName, detail.FaceValue, formatNotificationAmount(detail.PointsSpent), detail.Code, detail.RedeemedAt.Format("2006-01-02 15:04"))
}

func formatNotificationAmount(value float64) string {
	if value == float64(int64(value)) {
		return strconv.FormatInt(int64(value), 10)
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", value), "0"), ".")
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func giftCardNotificationUsers(notifications []domain.Notification) map[int64]int64 {
	out := map[int64]int64{}
	for _, notification := range notifications {
		if notification.Type != NotificationTypeGiftCardRedeemed || notification.TargetID == nil {
			continue
		}
		out[*notification.TargetID] = notification.UserID
	}
	return out
}

func (r NotificationsRepository) users(ctx context.Context, ids []int64) (map[int64]*domain.User, error) {
	out := map[int64]*domain.User{}
	if len(ids) == 0 {
		return out, nil
	}
	var users []domain.User
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Select("id", "nickname", "avatar", "user_id").Find(&users).Error; err != nil {
		return nil, err
	}
	for i := range users {
		out[users[i].ID] = &users[i]
	}
	return out, nil
}

func (r NotificationsRepository) comments(ctx context.Context, ids []int64) (map[int64]*domain.Comment, error) {
	out := map[int64]*domain.Comment{}
	if len(ids) == 0 {
		return out, nil
	}
	var comments []domain.Comment
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Select("id", "content", "post_id").Find(&comments).Error; err != nil {
		return nil, err
	}
	for i := range comments {
		out[comments[i].ID] = &comments[i]
	}
	return out, nil
}

func (r NotificationsRepository) confirmationsByNotificationID(ctx context.Context, userID int64, ids []int64) (map[int64]*domain.SystemNotificationConfirmation, error) {
	out := map[int64]*domain.SystemNotificationConfirmation{}
	if len(ids) == 0 {
		return out, nil
	}
	var confirmations []domain.SystemNotificationConfirmation
	if err := r.db.WithContext(ctx).Where("user_id = ? AND notification_id IN ?", userID, ids).Find(&confirmations).Error; err != nil {
		return nil, err
	}
	for i := range confirmations {
		out[confirmations[i].NotificationID] = &confirmations[i]
	}
	return out, nil
}

func notificationIDs(notifications []domain.SystemNotification) []int64 {
	out := make([]int64, 0, len(notifications))
	for _, notification := range notifications {
		out = append(out, notification.ID)
	}
	return out
}

func parseTypeFilter(value string) []int {
	parts := strings.Split(value, ",")
	out := []int{}
	for _, part := range parts {
		parsed, err := strconv.Atoi(strings.TrimSpace(part))
		if err == nil {
			out = append(out, parsed)
		}
	}
	return out
}
