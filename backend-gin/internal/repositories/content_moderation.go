package repositories

import (
	"context"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

func (r ContentRepository) ApprovePendingComment(ctx context.Context, commentID int64) (domain.Comment, error) {
	var out domain.Comment
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var comment domain.Comment
		if err := tx.Where("id = ?", commentID).First(&comment).Error; err != nil {
			return err
		}
		if comment.IsPublic && comment.AuditStatus == 1 {
			out = comment
			return nil
		}
		if err := tx.Model(&domain.Comment{}).Where("id = ?", comment.ID).Updates(map[string]any{"audit_status": 1, "is_public": true}).Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.Post{}).Where("id = ?", comment.PostID).UpdateColumn("comment_count", gorm.Expr("comment_count + ?", 1)).Error; err != nil {
			return err
		}
		var post domain.Post
		if err := tx.Select("id", "user_id").Where("id = ?", comment.PostID).First(&post).Error; err != nil {
			return err
		}
		var parent domain.Comment
		if comment.ParentID != nil {
			_ = tx.Select("id", "user_id").Where("id = ?", *comment.ParentID).First(&parent).Error
		}
		if comment.ParentID != nil && parent.ID != 0 && parent.UserID != comment.UserID {
			targetID := comment.PostID
			commentID := comment.ID
			if err := tx.Create(&domain.Notification{UserID: parent.UserID, SenderID: comment.UserID, Type: 4, Title: "回复了你的评论", TargetID: &targetID, CommentID: &commentID}).Error; err != nil {
				return err
			}
		} else if comment.ParentID == nil && post.UserID != comment.UserID {
			targetID := comment.PostID
			commentID := comment.ID
			if err := tx.Create(&domain.Notification{UserID: post.UserID, SenderID: comment.UserID, Type: 3, Title: "评论了你的笔记", TargetID: &targetID, CommentID: &commentID}).Error; err != nil {
				return err
			}
		}
		comment.AuditStatus = 1
		comment.IsPublic = true
		out = comment
		return nil
	})
	return out, err
}

func (r ContentRepository) RejectPendingComment(ctx context.Context, commentID int64) error {
	return r.db.WithContext(ctx).Model(&domain.Comment{}).Where("id = ?", commentID).Updates(map[string]any{"audit_status": 2, "is_public": false}).Error
}

func (r ContentRepository) MarkCommentAIModerated(ctx context.Context, commentID int64, auditStatus int) error {
	if auditStatus == 0 {
		auditStatus = 1
	}
	return r.db.WithContext(ctx).Model(&domain.Comment{}).Where("id = ?", commentID).Update("audit_status", auditStatus).Error
}

func (r ContentRepository) RejectCommentAfterAIModeration(ctx context.Context, commentID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var comment domain.Comment
		if err := tx.Select("id", "post_id", "is_public").Where("id = ?", commentID).First(&comment).Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.Comment{}).Where("id = ?", commentID).Updates(map[string]any{"audit_status": 2, "is_public": false}).Error; err != nil {
			return err
		}
		if comment.IsPublic {
			return tx.Model(&domain.Post{}).Where("id = ? AND comment_count > 0", comment.PostID).UpdateColumn("comment_count", gorm.Expr("comment_count - ?", 1)).Error
		}
		return nil
	})
}

func (r ContentRepository) DeleteCommentAfterAIModeration(ctx context.Context, commentID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var comment domain.Comment
		if err := tx.Select("id", "post_id").Where("id = ?", commentID).First(&comment).Error; err != nil {
			return err
		}
		visibleCount, err := countVisibleCommentRecursive(ctx, tx, commentID)
		if err != nil {
			return err
		}
		if _, err := deleteCommentRecursive(ctx, tx, commentID); err != nil {
			return err
		}
		if visibleCount > 0 {
			return tx.Model(&domain.Post{}).
				Where("id = ?", comment.PostID).
				UpdateColumn("comment_count", gorm.Expr("CASE WHEN comment_count >= ? THEN comment_count - ? ELSE 0 END", visibleCount, visibleCount)).Error
		}
		return nil
	})
}

func (r ContentRepository) ApprovePendingPost(ctx context.Context, postID int64, visibility string) (domain.Post, error) {
	var out domain.Post
	if visibility == "" {
		visibility = VisibilityPublic
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var post domain.Post
		if err := tx.Where("id = ?", postID).First(&post).Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.Post{}).Where("id = ?", post.ID).Updates(map[string]any{"audit_status": 1, "visibility": normalizeVisibility(visibility)}).Error; err != nil {
			return err
		}
		if !post.IsDraft && normalizeVisibility(visibility) == VisibilityPublic {
			if err := createFollowerNewPostNotifications(ctx, tx, post.UserID, post.ID, post.Title); err != nil {
				return err
			}
		}
		post.AuditStatus = 1
		post.Visibility = normalizeVisibility(visibility)
		out = post
		return nil
	})
	return out, err
}

func (r ContentRepository) PrivatePendingPost(ctx context.Context, postID int64) error {
	return r.db.WithContext(ctx).Model(&domain.Post{}).Where("id = ?", postID).Updates(map[string]any{"audit_status": 2, "visibility": VisibilityPrivate}).Error
}

func (r ContentRepository) MarkPostAIModerated(ctx context.Context, postID int64, auditStatus int) error {
	if auditStatus == 0 {
		auditStatus = 1
	}
	return r.db.WithContext(ctx).Model(&domain.Post{}).Where("id = ?", postID).Update("audit_status", auditStatus).Error
}

func (r ContentRepository) PrivatePostAfterAIModeration(ctx context.Context, postID int64) error {
	return r.db.WithContext(ctx).Model(&domain.Post{}).Where("id = ?", postID).Updates(map[string]any{"audit_status": 2, "visibility": VisibilityPrivate}).Error
}
