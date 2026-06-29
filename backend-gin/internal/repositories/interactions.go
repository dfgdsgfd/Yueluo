package repositories

import (
	"context"
	"time"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

const (
	NotificationLikePost    = 1
	NotificationLikeComment = 2
)

type InteractionsRepository struct {
	db                      *gorm.DB
	notificationSuppression NotificationSuppressionConfig
}

type ToggleLikeResult struct {
	Liked                 bool
	NotificationUserID    int64
	NotificationTargetID  int64
	NotificationCommentID *int64
	NotificationType      int
}

func NewInteractionsRepository(db *gorm.DB, configs ...NotificationSuppressionConfig) InteractionsRepository {
	repo := InteractionsRepository{db: db}
	if len(configs) > 0 {
		repo.notificationSuppression = configs[0]
	}
	return repo
}

func (r InteractionsRepository) ToggleLike(ctx context.Context, userID int64, targetType int, targetID int64) (ToggleLikeResult, error) {
	var result ToggleLikeResult
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing domain.Like
		err := tx.Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).First(&existing).Error
		if err == nil {
			if err := tx.Delete(&existing).Error; err != nil {
				return err
			}
			return decrementLikeCounters(tx, targetType, targetID)
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}

		like := domain.Like{UserID: userID, TargetType: targetType, TargetID: targetID}
		if err := tx.Create(&like).Error; err != nil {
			return err
		}
		notificationUserID, notificationTargetID, notificationCommentID, notificationType, err := incrementLikeCounters(tx, targetType, targetID)
		if err != nil {
			return err
		}
		result = ToggleLikeResult{
			Liked:                 true,
			NotificationUserID:    notificationUserID,
			NotificationTargetID:  notificationTargetID,
			NotificationCommentID: notificationCommentID,
			NotificationType:      notificationType,
		}
		if notificationUserID != 0 && notificationUserID != userID {
			notification := likeNotification(notificationUserID, userID, notificationType, notificationTargetID, notificationCommentID)
			if err := createNotificationIfAllowed(ctx, tx, notification, r.notificationSuppression); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return ToggleLikeResult{}, err
	}
	return result, nil
}

func (r InteractionsRepository) RemoveLike(ctx context.Context, userID int64, targetType int, targetID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing domain.Like
		if err := tx.Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).First(&existing).Error; err != nil {
			return err
		}
		if err := tx.Delete(&existing).Error; err != nil {
			return err
		}
		return decrementLikeCounters(tx, targetType, targetID)
	})
}

func (r InteractionsRepository) ToggleDislike(ctx context.Context, userID int64, postID int64) (bool, error) {
	disliked := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing domain.Dislike
		err := tx.Where("user_id = ? AND post_id = ?", userID, postID).First(&existing).Error
		if err == nil {
			return tx.Delete(&existing).Error
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
		disliked = true
		return tx.Create(&domain.Dislike{UserID: userID, PostID: postID}).Error
	})
	return disliked, err
}

func (r InteractionsRepository) HasDislike(ctx context.Context, userID int64, postID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Dislike{}).Where("user_id = ? AND post_id = ?", userID, postID).Count(&count).Error
	return count > 0, err
}

func (r InteractionsRepository) ReportExists(ctx context.Context, reporterID int64, targetType string, targetID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Report{}).
		Where("reporter_id = ? AND target_type = ? AND target_id = ?", reporterID, targetType, targetID).
		Count(&count).Error
	return count > 0, err
}

func (r InteractionsRepository) CreateReport(ctx context.Context, report domain.Report) (*domain.Report, error) {
	report.Status = "pending"
	if err := r.db.WithContext(ctx).Create(&report).Error; err != nil {
		return nil, err
	}
	return &report, nil
}

func incrementLikeCounters(tx *gorm.DB, targetType int, targetID int64) (int64, int64, *int64, int, error) {
	if targetType == 1 {
		var post domain.Post
		if err := tx.Select("id", "user_id").Where("id = ?", targetID).First(&post).Error; err != nil {
			return 0, 0, nil, 0, err
		}
		if err := tx.Model(&domain.Post{}).Where("id = ?", targetID).UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error; err != nil {
			return 0, 0, nil, 0, err
		}
		if err := tx.Model(&domain.User{}).Where("id = ?", post.UserID).UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error; err != nil {
			return 0, 0, nil, 0, err
		}
		return post.UserID, targetID, nil, NotificationLikePost, nil
	}

	var comment domain.Comment
	if err := tx.Select("id", "post_id", "user_id").Where("id = ?", targetID).First(&comment).Error; err != nil {
		return 0, 0, nil, 0, err
	}
	if err := tx.Model(&domain.Comment{}).Where("id = ?", targetID).UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error; err != nil {
		return 0, 0, nil, 0, err
	}
	commentID := targetID
	return comment.UserID, comment.PostID, &commentID, NotificationLikeComment, nil
}

func decrementLikeCounters(tx *gorm.DB, targetType int, targetID int64) error {
	if targetType == 1 {
		var post domain.Post
		if err := tx.Select("id", "user_id").Where("id = ?", targetID).First(&post).Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.Post{}).Where("id = ?", targetID).UpdateColumn("like_count", gorm.Expr("like_count - ?", 1)).Error; err != nil {
			return err
		}
		return tx.Model(&domain.User{}).Where("id = ?", post.UserID).UpdateColumn("like_count", gorm.Expr("like_count - ?", 1)).Error
	}
	return tx.Model(&domain.Comment{}).Where("id = ?", targetID).UpdateColumn("like_count", gorm.Expr("like_count - ?", 1)).Error
}

type NotificationSuppressionConfig struct {
	Enabled       bool
	WindowSeconds int
	Threshold     int
	Now           func() time.Time
}

func createNotificationIfAllowed(ctx context.Context, tx *gorm.DB, notification domain.Notification, cfg NotificationSuppressionConfig) error {
	if tx == nil {
		return nil
	}
	now := time.Now
	if cfg.Now != nil {
		now = cfg.Now
	}
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = now()
	}
	if !cfg.Enabled || cfg.WindowSeconds <= 0 || cfg.Threshold <= 0 {
		return tx.WithContext(ctx).Create(&notification).Error
	}
	since := now().Add(-time.Duration(cfg.WindowSeconds) * time.Second)
	var count int64
	if err := tx.WithContext(ctx).Model(&domain.Notification{}).
		Where("user_id = ? AND sender_id = ? AND type = ? AND created_at >= ?", notification.UserID, notification.SenderID, notification.Type, since).
		Count(&count).Error; err != nil {
		return err
	}
	if count >= int64(cfg.Threshold) {
		return nil
	}
	return tx.WithContext(ctx).Create(&notification).Error
}

func likeNotification(userID int64, senderID int64, notificationType int, targetID int64, commentID *int64) domain.Notification {
	title := "赞了你的笔记"
	if notificationType == NotificationLikeComment {
		title = "赞了你的评论"
	}
	return domain.Notification{
		UserID:    userID,
		SenderID:  senderID,
		Type:      notificationType,
		Title:     title,
		TargetID:  &targetID,
		CommentID: commentID,
		IsRead:    false,
	}
}
