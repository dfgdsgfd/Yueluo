package handlers

import (
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
)

const (
	msgInvalidToken             = "\u65e0\u6548\u7684\u8bbf\u95ee\u4ee4\u724c"
	msgGetActivitiesFailed      = "\u83b7\u53d6\u6d3b\u52a8\u5217\u8868\u5931\u8d25"
	msgGetNotificationsFailed   = "\u83b7\u53d6\u901a\u77e5\u5217\u8868\u5931\u8d25"
	msgGetUnreadCountFailed     = "\u83b7\u53d6\u672a\u8bfb\u901a\u77e5\u6570\u91cf\u5931\u8d25"
	msgMarkAllReadFailed        = "\u5168\u90e8\u6807\u8bb0\u5df2\u8bfb\u5931\u8d25"
	msgMarkAllReadOK            = "\u5168\u90e8\u6807\u8bb0\u5df2\u8bfb\u6210\u529f"
	msgNotificationNotFound     = "\u901a\u77e5\u4e0d\u5b58\u5728"
	msgMarkReadFailed           = "\u6807\u8bb0\u5df2\u8bfb\u5931\u8d25"
	msgMarkReadOK               = "\u6807\u8bb0\u5df2\u8bfb\u6210\u529f"
	msgDeleteFailed             = "\u5220\u9664\u5931\u8d25"
	msgDeleteOK                 = "\u5220\u9664\u6210\u529f"
	msgGetSystemFailed          = "\u83b7\u53d6\u7cfb\u7edf\u901a\u77e5\u5217\u8868\u5931\u8d25"
	msgGetPopupFailed           = "\u83b7\u53d6\u5f39\u7a97\u901a\u77e5\u5931\u8d25"
	msgSystemNotificationAbsent = "\u7cfb\u7edf\u901a\u77e5\u4e0d\u5b58\u5728"
	msgConfirmFailed            = "\u786e\u8ba4\u5931\u8d25"
	msgConfirmOK                = "\u786e\u8ba4\u6210\u529f"
)

func (h NativeHandlers) NotificationActivities(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgGetActivitiesFailed, nil)
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := min(positiveIntQuery(c, "limit", 20), 50)
	total, activities, err := repositories.NewNotificationsRepository(h.DB).Activities(c.Request.Context(), page, limit)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgGetActivitiesFailed, nil)
		return
	}
	data := make([]gin.H, 0, len(activities))
	for _, activity := range activities {
		data = append(data, h.systemNotificationActivityResponse(activity))
	}
	c.JSON(http.StatusOK, gin.H{
		"code": response.CodeSuccess,
		"data": gin.H{
			"data":       data,
			"pagination": pagination(page, limit, total),
		},
		"message": "success",
	})
}

func (h NativeHandlers) Notifications(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgGetNotificationsFailed, nil)
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := positiveIntQuery(c, "limit", 20)
	total, notifications, err := repositories.NewNotificationsRepository(h.DB).List(c.Request.Context(), user.ID, repositories.NotificationListParams{
		Page:   page,
		Limit:  limit,
		Type:   c.Query("type"),
		IsRead: c.Query("is_read"),
	})
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgGetNotificationsFailed, nil)
		return
	}
	data := make([]gin.H, 0, len(notifications))
	for _, notification := range notifications {
		data = append(data, h.notificationResponse(notification))
	}
	c.JSON(http.StatusOK, gin.H{
		"code": response.CodeSuccess,
		"data": gin.H{
			"data":       data,
			"pagination": pagination(page, limit, total),
		},
		"message": "success",
	})
}

func (h NativeHandlers) NotificationUnreadCount(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgGetUnreadCountFailed, nil)
		return
	}
	cacheKey := ""
	if h.Redis != nil {
		cacheKey = h.cacheKeyWithVersions(cacheScopeNotifications, []string{cacheScopeSystemNotifications}, user.ID, "unread")
		var data gin.H
		_, err := h.Redis.CacheGetOrLoad(c.Request.Context(), cacheKey, &data, cacheTTL(10), func() (any, error) {
			notificationCount, systemCount, err := repositories.NewNotificationsRepository(h.DB).UnreadCounts(c.Request.Context(), user.ID)
			if err != nil {
				return nil, err
			}
			return notificationUnreadPayload(notificationCount, systemCount), nil
		})
		if err != nil {
			response.JSON(c, http.StatusInternalServerError, response.CodeError, msgGetUnreadCountFailed, nil)
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": data, "message": "success"})
		return
	}
	notificationCount, systemCount, err := repositories.NewNotificationsRepository(h.DB).UnreadCounts(c.Request.Context(), user.ID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgGetUnreadCountFailed, nil)
		return
	}
	data := notificationUnreadPayload(notificationCount, systemCount)
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": data, "message": "success"})
}

func notificationUnreadPayload(notificationCount, systemCount int64) gin.H {
	return gin.H{
		"notification_count":        notificationCount,
		"system_notification_count": systemCount,
		"total":                     notificationCount + systemCount,
	}
}

func (h NativeHandlers) MarkAllNotificationsRead(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgMarkAllReadFailed, nil)
		return
	}
	if err := repositories.NewNotificationsRepository(h.DB).MarkAllRead(c.Request.Context(), user.ID); err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgMarkAllReadFailed, nil)
		return
	}
	h.bumpCacheVersions(cacheScopeNotifications)
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": msgMarkAllReadOK})
}

func (h NativeHandlers) MarkNotificationRead(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgNotificationNotFound, nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgMarkReadFailed, nil)
		return
	}
	err = repositories.NewNotificationsRepository(h.DB).MarkRead(c.Request.Context(), user.ID, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgNotificationNotFound, nil)
		return
	}
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgMarkReadFailed, nil)
		return
	}
	h.bumpCacheVersions(cacheScopeNotifications)
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": msgMarkReadOK})
}

func (h NativeHandlers) DeleteNotification(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgNotificationNotFound, nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgDeleteFailed, nil)
		return
	}
	err = repositories.NewNotificationsRepository(h.DB).Delete(c.Request.Context(), user.ID, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgNotificationNotFound, nil)
		return
	}
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgDeleteFailed, nil)
		return
	}
	h.bumpCacheVersions(cacheScopeNotifications)
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": msgDeleteOK})
}

func (h NativeHandlers) SystemNotifications(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgGetSystemFailed, nil)
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := positiveIntQuery(c, "limit", 20)
	total, notifications, err := repositories.NewNotificationsRepository(h.DB).SystemList(c.Request.Context(), user.ID, page, limit, c.Query("type"))
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgGetSystemFailed, nil)
		return
	}
	data := make([]gin.H, 0, len(notifications))
	for _, notification := range notifications {
		data = append(data, h.systemNotificationResponse(notification))
	}
	c.JSON(http.StatusOK, gin.H{
		"code": response.CodeSuccess,
		"data": gin.H{
			"data":       data,
			"pagination": pagination(page, limit, total),
		},
		"message": "success",
	})
}

func (h NativeHandlers) PopupSystemNotifications(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgGetPopupFailed, nil)
		return
	}
	notifications, err := repositories.NewNotificationsRepository(h.DB).PopupSystem(c.Request.Context(), user.ID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgGetPopupFailed, nil)
		return
	}
	data := make([]gin.H, 0, len(notifications))
	for _, notification := range notifications {
		data = append(data, h.systemNotificationRawResponse(notification))
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": data, "message": "success"})
}

func (h NativeHandlers) ConfirmSystemNotification(c *gin.Context) {
	h.modifySystemNotification(c, true)
}

func (h NativeHandlers) DismissSystemNotification(c *gin.Context) {
	h.modifySystemNotification(c, false)
}

func (h NativeHandlers) modifySystemNotification(c *gin.Context, confirm bool) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgSystemNotificationAbsent, nil)
		return
	}
	if h.DB == nil {
		if confirm {
			response.JSON(c, http.StatusInternalServerError, response.CodeError, msgConfirmFailed, nil)
			return
		}
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgDeleteFailed, nil)
		return
	}
	repo := repositories.NewNotificationsRepository(h.DB)
	if confirm {
		err = repo.ConfirmSystem(c.Request.Context(), user.ID, id)
	} else {
		err = repo.DismissSystem(c.Request.Context(), user.ID, id)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgSystemNotificationAbsent, nil)
		return
	}
	if err != nil {
		if confirm {
			response.JSON(c, http.StatusInternalServerError, response.CodeError, msgConfirmFailed, nil)
			return
		}
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgDeleteFailed, nil)
		return
	}
	if confirm {
		h.bumpCacheVersions(cacheScopeSystemNotifications, cacheScopeNotifications)
		c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": msgConfirmOK})
		return
	}
	h.bumpCacheVersions(cacheScopeSystemNotifications, cacheScopeNotifications)
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": msgDeleteOK})
}

func (h NativeHandlers) notificationResponse(bundle repositories.NotificationBundle) gin.H {
	n := bundle.Notification
	body := gin.H{
		"id":         n.ID,
		"user_id":    n.UserID,
		"sender_id":  n.SenderID,
		"type":       n.Type,
		"title":      n.Title,
		"target_id":  n.TargetID,
		"comment_id": n.CommentID,
		"is_read":    n.IsRead,
		"created_at": n.CreatedAt,
		"sender":     h.notificationSenderResponse(bundle.Sender),
		"comment":    notificationCommentResponse(bundle.Comment),
		"detail":     bundle.Detail,
		"post_cover": h.signFileURLPtr(bundle.PostCover),
		"post_title": bundle.PostTitle,
	}
	if bundle.GiftCardRedemption != nil {
		body["gift_card_redemption"] = giftCardNotificationResponse(*bundle.GiftCardRedemption)
	}
	return body
}

func giftCardNotificationResponse(detail repositories.GiftCardRedemptionNotification) gin.H {
	return gin.H{
		"product_name": detail.ProductName,
		"face_value":   detail.FaceValue,
		"points_spent": detail.PointsSpent,
		"code":         detail.Code,
		"redeemed_at":  detail.RedeemedAt,
	}
}

func (h NativeHandlers) notificationSenderResponse(user *domain.User) any {
	if user == nil {
		return nil
	}
	return gin.H{"id": user.ID, "nickname": user.Nickname, "avatar": h.signFileURLPtr(user.Avatar), "user_id": user.UserID}
}

func notificationCommentResponse(comment *domain.Comment) any {
	if comment == nil {
		return nil
	}
	return gin.H{"id": comment.ID, "content": comment.Content, "post_id": comment.PostID, "parent_id": comment.ParentID}
}

func (h NativeHandlers) systemNotificationResponse(bundle repositories.SystemNotificationBundle) gin.H {
	body := h.systemNotificationRawResponse(bundle.Notification)
	body["is_read"] = bundle.Confirmation != nil
	if bundle.Confirmation != nil {
		body["confirmed_at"] = bundle.Confirmation.ConfirmedAt
	} else {
		body["confirmed_at"] = nil
	}
	return body
}

func (h NativeHandlers) systemNotificationActivityResponse(notification domain.SystemNotification) gin.H {
	return gin.H{
		"id":             notification.ID,
		"title":          notification.Title,
		"content":        notification.Content,
		"type":           notification.Type,
		"content_format": notification.ContentFormat,
		"image_url":      h.signFileURLPtr(notification.ImageURL),
		"link_url":       notification.LinkURL,
		"show_popup":     notification.ShowPopup,
		"start_time":     notification.StartTime,
		"end_time":       notification.EndTime,
		"created_at":     notification.CreatedAt,
	}
}

func (h NativeHandlers) systemNotificationRawResponse(notification domain.SystemNotification) gin.H {
	return gin.H{
		"id":             notification.ID,
		"title":          notification.Title,
		"content":        notification.Content,
		"type":           notification.Type,
		"content_format": notification.ContentFormat,
		"image_url":      h.signFileURLPtr(notification.ImageURL),
		"link_url":       notification.LinkURL,
		"show_popup":     notification.ShowPopup,
		"is_active":      notification.IsActive,
		"start_time":     notification.StartTime,
		"end_time":       notification.EndTime,
		"created_at":     notification.CreatedAt,
		"updated_at":     notification.UpdatedAt,
	}
}

func positiveIntQuery(c *gin.Context, key string, fallback int) int {
	value := intQuery(c, key, fallback)
	if value < 1 {
		return fallback
	}
	return value
}

func pagination(page, limit int, total int64) gin.H {
	return gin.H{
		"page":  page,
		"limit": limit,
		"total": total,
		"pages": int(math.Ceil(float64(total) / float64(limit))),
	}
}
