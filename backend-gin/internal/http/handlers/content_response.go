package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
	"yuem-go/backend-gin/internal/services"
)

func (h NativeHandlers) basePostResponse(bundle repositories.PostBundle) gin.H {
	post := bundle.Post
	visibility := post.Visibility
	if visibility == "" {
		visibility = repositories.VisibilityPublic
	}
	qualityLevel := normalizedQualityLevel(post.QualityLevel)
	body := gin.H{
		"id":                   int(post.ID),
		"user_id":              int(post.UserID),
		"title":                post.Title,
		"content":              post.Content,
		"category_id":          post.CategoryID,
		"category":             nil,
		"type":                 post.Type,
		"view_count":           post.ViewCount,
		"like_count":           post.LikeCount,
		"collect_count":        post.CollectCount,
		"comment_count":        post.CommentCount,
		"created_at":           post.CreatedAt,
		"is_draft":             post.IsDraft,
		"visibility":           visibility,
		"public_access_exempt": post.PublicAccessExempt,
		"quality_level":        qualityLevel,
		"quality_reward":       post.QualityReward,
		"quality_marked_at":    post.QualityMarkedAt,
		"original_incentive":   postHasOriginalIncentive(post),
		"nickname":             nil,
		"user_avatar":          nil,
		"author_account":       nil,
		"author_auto_id":       nil,
		"location":             nil,
		"verified":             nil,
		"avatar":               nil,
		"author":               nil,
	}
	if bundle.Category != nil {
		body["category"] = bundle.Category.Name
	}
	if bundle.User != nil {
		body["nickname"] = bundle.User.Nickname
		body["user_avatar"] = h.signFileURLPtr(bundle.User.Avatar)
		body["author_account"] = bundle.User.UserID
		body["author_auto_id"] = int(bundle.User.ID)
		body["location"] = bundle.User.Location
		body["verified"] = bundle.User.Verified
		body["avatar"] = h.signFileURLPtr(bundle.User.Avatar)
		body["author"] = bundle.User.Nickname
	}
	return body
}

func (h NativeHandlers) protectPostListItem(post gin.H, payment *domain.PostPaymentSetting, isAuthor, hasPurchased bool, videos []domain.PostVideo, images []domain.PostImage) {
	paid := payment != nil && payment.Enabled
	protect := paid && !isAuthor && !hasPurchased
	hideAll := payment != nil && payment.HideAll
	postType, _ := post["type"].(int)
	if postType == repositories.PostTypeVideo {
		var video *domain.PostVideo
		if len(videos) > 0 {
			video = &videos[0]
		}
		if video != nil && video.CoverURL != nil {
			cover := h.signFileURL(*video.CoverURL)
			post["images"] = []string{cover}
			post["image"] = cover
		} else {
			post["images"] = []string{}
			post["image"] = nil
		}
		if protect {
			switch {
			case hideAll:
				post["video_url"] = nil
				post["preview_video_url"] = nil
			case video != nil && video.PreviewVideoURL != nil && *video.PreviewVideoURL != "":
				post["video_url"] = nil
				post["preview_video_url"] = h.signFileURLPtr(video.PreviewVideoURL)
			case payment != nil && payment.PreviewDuration > 0 && video != nil:
				post["video_url"] = h.signFileURL(video.VideoURL)
				post["preview_video_url"] = nil
			default:
				post["video_url"] = nil
				post["preview_video_url"] = nil
			}
		} else if video != nil {
			post["video_url"] = h.signFileURL(video.VideoURL)
			post["preview_video_url"] = nil
		} else {
			post["video_url"] = nil
			post["preview_video_url"] = nil
		}
	} else {
		canViewPaid := !paid || isAuthor || hasPurchased
		access := imageAccessForViewer(images, payment, canViewPaid, false)
		sorted := access.DirectImages
		var cover any
		if len(sorted) > 0 {
			cover = h.signFileURL(sorted[0].ImageURL)
		}
		post["images"] = h.listImagesResponse(sorted)
		post["image"] = cover
		applyImageAccessMetadata(post, access)
	}
	post["isPaidContent"] = paid
	post["paymentSettings"] = paymentSettingsResponse(payment)
	post["hasPurchased"] = hasPurchased || isAuthor
	_ = protect
	_ = hideAll
}

func protectPostDetail(post gin.H, payment *domain.PostPaymentSetting) {
	hideAll := payment != nil && payment.HideAll
	if _, preset := post["totalImagesCount"]; !preset {
		if images, ok := post["images"].([]gin.H); ok && len(images) > 0 {
			post["totalImagesCount"] = len(images)
			paidCount := 0
			filtered := []gin.H{}
			for _, image := range images {
				if image["isFreePreview"] == false {
					paidCount++
					continue
				}
				filtered = append(filtered, image)
			}
			post["hiddenPaidImagesCount"] = paidCount
			if hideAll {
				post["images"] = []gin.H{}
				post["hiddenPaidImagesCount"] = len(images)
			} else {
				post["images"] = filtered
			}
		}
	}
	if post["type"] == repositories.PostTypeVideo {
		if hideAll {
			post["video_url"] = nil
			post["preview_video_url"] = nil
			post["videos"] = protectedVideosResponse(post["videos"], false)
		} else if post["preview_video_url"] != nil {
			post["video_url"] = nil
			post["videos"] = protectedVideosResponse(post["videos"], true)
		} else if payment == nil || payment.PreviewDuration <= 0 {
			post["video_url"] = nil
			post["videos"] = protectedVideosResponse(post["videos"], false)
		}
	}
	post["attachment"] = nil
	if hideAll {
		post["content"] = ""
		post["contentTruncated"] = true
		post["contentHidden"] = true
	} else if content, ok := post["content"].(string); ok && len([]rune(content)) > 100 {
		post["content"] = string([]rune(content)[:100]) + "..."
		post["contentTruncated"] = true
	}
}

func protectedVideosResponse(value any, keepPreview bool) any {
	rows, ok := value.([]gin.H)
	if !ok {
		return value
	}
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		item := gin.H{"cover_url": row["cover_url"], "video_url": nil}
		if keepPreview {
			item["preview_video_url"] = row["preview_video_url"]
		}
		out = append(out, item)
	}
	return out
}

func tagsResponse(tags []domain.Tag) []gin.H {
	out := make([]gin.H, 0, len(tags))
	for _, tag := range tags {
		out = append(out, gin.H{"id": tag.ID, "name": tag.Name})
	}
	return out
}

func sortedImages(images []domain.PostImage) []domain.PostImage {
	out := append([]domain.PostImage(nil), images...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return out[i].ID < out[j].ID
	})
	return out
}

type imageAccessSummary struct {
	DirectImages               []domain.PostImage
	ProtectedPackageImages     []domain.PostImage
	TotalImagesCount           int
	HiddenPaidImagesCount      int
	ProtectedImagesCount       int
	ProtectedFreeImagesCount   int
	ProtectedPaidImagesCount   int
	ProtectedPackageImageCount int
	LockedProtectedImagesCount int
}

func imageAccessForViewer(images []domain.PostImage, payment *domain.PostPaymentSetting, canViewPaid, allowProtectedDirect bool) imageAccessSummary {
	summary := imageAccessSummary{TotalImagesCount: len(images)}
	paid := payment != nil && payment.Enabled
	hideAll := payment != nil && payment.HideAll
	for _, image := range repositories.NormalizePostImagesForAccess(images) {
		if image.IsProtected {
			summary.ProtectedImagesCount++
			if image.IsFreePreview || !paid {
				summary.ProtectedFreeImagesCount++
			} else {
				summary.ProtectedPaidImagesCount++
			}
			if !paid || canViewPaid || image.IsFreePreview {
				summary.ProtectedPackageImageCount++
				summary.ProtectedPackageImages = append(summary.ProtectedPackageImages, image)
			} else {
				summary.HiddenPaidImagesCount++
				summary.LockedProtectedImagesCount++
			}
			if allowProtectedDirect {
				summary.DirectImages = append(summary.DirectImages, image)
			}
			continue
		}
		if paid && !canViewPaid {
			if hideAll || !image.IsFreePreview {
				summary.HiddenPaidImagesCount++
				continue
			}
		}
		summary.DirectImages = append(summary.DirectImages, image)
	}
	return summary
}

func applyImageAccessMetadata(post gin.H, summary imageAccessSummary) {
	post["totalImagesCount"] = summary.TotalImagesCount
	post["hiddenPaidImagesCount"] = summary.HiddenPaidImagesCount
	post["protectedImagesCount"] = summary.ProtectedImagesCount
	post["protectedFreeImagesCount"] = summary.ProtectedFreeImagesCount
	post["protectedPaidImagesCount"] = summary.ProtectedPaidImagesCount
	post["protectedPackageImageCount"] = summary.ProtectedPackageImageCount
	post["lockedProtectedImagesCount"] = summary.LockedProtectedImagesCount
	post["protectedPackageAvailable"] = summary.ProtectedPackageImageCount > 0
	post["protectedPackageRequired"] = summary.ProtectedImagesCount > 0
}

func (h NativeHandlers) applyImageArchiveMetadata(post gin.H, summary imageAccessSummary, payment *domain.PostPaymentSetting, canViewPaid bool) {
	enabled := h.imageArchiveEnabled()
	threshold := h.imageArchiveThreshold()
	eligible := enabled && summary.TotalImagesCount > threshold
	mode := "shared"
	if summary.ProtectedImagesCount > 0 {
		mode = "protected"
	}
	post["imageArchiveEnabled"] = enabled
	post["imageArchiveEligible"] = eligible
	post["imageArchiveThreshold"] = threshold
	post["imageArchiveMode"] = mode
	post["imageArchiveRequiresPurchase"] = eligible && payment != nil && payment.Enabled && !canViewPaid
}

func (h NativeHandlers) imageProtectionEnabled() bool {
	return h.Settings != nil && h.Settings.Bool("image_protection_enabled")
}

func (h NativeHandlers) maxPostImages() int {
	if h.Settings == nil {
		return defaultMaxPostImages
	}
	return min(max(h.Settings.Int("image_post_max_count", defaultMaxPostImages), 1), 500)
}

func (h NativeHandlers) maxPostContentLength() int {
	if h.Settings == nil {
		return defaultMaxPostContent
	}
	return min(max(h.Settings.Int("post_content_max_length", defaultMaxPostContent), 1), 1_000_000)
}

func (h NativeHandlers) rejectPostContentOverLimit(c *gin.Context, content string) bool {
	currentLength := postContentLength(content)
	maxLength := h.maxPostContentLength()
	if currentLength <= maxLength {
		return false
	}
	response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.post_content_limit", gin.H{
		"currentLength": currentLength,
		"maxLength":     maxLength,
	})
	return true
}

func (h NativeHandlers) postResourceSectionPosition() string {
	if h.Settings != nil && h.Settings.String("post_resource_section_position") == "after_content" {
		return "after_content"
	}
	return "before_content"
}

func (h NativeHandlers) imageArchiveEnabled() bool {
	return h.Settings == nil || h.Settings.Bool("image_archive_enabled")
}

func (h NativeHandlers) imageArchiveThreshold() int {
	threshold := 25
	if h.Settings != nil {
		threshold = h.Settings.Int("image_archive_threshold", threshold)
	}
	return min(max(threshold, 1), h.maxPostImages())
}

func (h NativeHandlers) paidContentPaymentMethodEnabled(method string) bool {
	if h.Settings == nil {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(method)) {
	case "balance":
		return h.Settings.Bool("paid_content_balance_enabled")
	case "points":
		return h.Settings.Bool("paid_content_points_enabled")
	default:
		return false
	}
}

func (h NativeHandlers) paidContentMaxPrice(method string) float64 {
	switch strings.ToLower(strings.TrimSpace(method)) {
	case "points":
		if h.Settings == nil {
			return services.DefaultPaidContentPointsMaxPrice
		}
		maxPrice := h.Settings.Int("paid_content_points_max_price", services.DefaultPaidContentPointsMaxPrice)
		if maxPrice <= 0 {
			return services.DefaultPaidContentPointsMaxPrice
		}
		return float64(maxPrice)
	case "balance":
		if h.Settings == nil {
			return services.DefaultPaidContentBalanceMaxPrice
		}
		maxPrice := h.Settings.Int("paid_content_balance_max_price", services.DefaultPaidContentBalanceMaxPrice)
		if maxPrice <= 0 {
			return services.DefaultPaidContentBalanceMaxPrice
		}
		return float64(maxPrice)
	default:
		return 0
	}
}

func (h NativeHandlers) rejectDisabledImagePaymentMethod(c *gin.Context, payment *repositories.PaymentSettingsInput) bool {
	if payment == nil || h.paidContentPaymentMethodEnabled(payment.PaymentMethod) {
		return false
	}
	response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.payment_method_disabled", gin.H{"paymentMethod": normalizePaymentMethodForResponse(payment.PaymentMethod)})
	return true
}

func (h NativeHandlers) rejectImagePaymentPriceOverLimit(c *gin.Context, payment *repositories.PaymentSettingsInput) bool {
	if payment == nil || !payment.Enabled {
		return false
	}
	method := normalizePaymentMethodForResponse(payment.PaymentMethod)
	maxPrice := h.paidContentMaxPrice(method)
	if maxPrice <= 0 || payment.Price <= maxPrice {
		return false
	}
	response.JSON(c, http.StatusBadRequest, response.CodeValidationError, fmt.Sprintf("error.paid_content_%s_price_limit", method), gin.H{"paymentMethod": method, "maxPrice": maxPrice})
	return true
}

func (h NativeHandlers) rejectImageProtectionWhenDisabled(c *gin.Context, images []repositories.PostImageInput) bool {
	if h.imageProtectionEnabled() {
		return false
	}
	for _, image := range images {
		if image.IsProtected {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.image_protection_disabled", nil)
			return true
		}
	}
	return false
}

func (h NativeHandlers) rejectImageProtectionUpdateWhenDisabled(c *gin.Context, postID int64, images []repositories.PostImageInput) bool {
	if h.imageProtectionEnabled() {
		return false
	}
	var existing []domain.PostImage
	if err := h.DB.WithContext(c.Request.Context()).Where("post_id = ? AND is_protected = ?", postID, true).Find(&existing).Error; err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
		return true
	}
	protectedURLs := make(map[string]struct{}, len(existing))
	for _, image := range existing {
		protectedURLs[strings.TrimSpace(image.ImageURL)] = struct{}{}
	}
	for _, image := range images {
		if !image.IsProtected {
			continue
		}
		if _, ok := protectedURLs[strings.TrimSpace(image.URL)]; !ok {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.image_protection_disabled", nil)
			return true
		}
	}
	return false
}

func (h NativeHandlers) rejectInvalidImagePayment(c *gin.Context, images []repositories.PostImageInput, payment *repositories.PaymentSettingsInput) bool {
	if !hasPaidPostImages(images) {
		return false
	}
	if !validImagePaymentSettings(payment) {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.image_payment_settings_required", nil)
		return true
	}
	if h.rejectDisabledImagePaymentMethod(c, payment) {
		return true
	}
	if h.rejectImagePaymentPriceOverLimit(c, payment) {
		return true
	}
	return false
}

func (h NativeHandlers) rejectInvalidUpdatedImagePayment(c *gin.Context, postID int64, input repositories.UpdatePostInput) bool {
	if !input.ImagesSet && !input.PaymentSet {
		return false
	}
	images := input.Images
	if !input.ImagesSet {
		var existingImages []domain.PostImage
		if err := h.DB.WithContext(c.Request.Context()).Where("post_id = ?", postID).Order("sort_order ASC, id ASC").Find(&existingImages).Error; err != nil {
			response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
			return true
		}
		images = make([]repositories.PostImageInput, 0, len(existingImages))
		for _, image := range existingImages {
			images = append(images, repositories.PostImageInput{
				URL:                 image.ImageURL,
				WatermarkTraceToken: image.WatermarkTraceToken,
				IsFreePreview:       image.IsFreePreview,
				IsProtected:         image.IsProtected,
				SortOrder:           image.SortOrder,
			})
		}
	}
	images = repositories.NormalizePostImageInputs(images)
	if !hasPaidPostImages(images) {
		return false
	}
	payment := input.PaymentSettings
	if !input.PaymentSet {
		var existing domain.PostPaymentSetting
		if err := h.DB.WithContext(c.Request.Context()).Where("post_id = ? AND enabled = ?", postID, true).First(&existing).Error; err == nil {
			payment = &repositories.PaymentSettingsInput{
				Enabled:       existing.Enabled,
				PaymentMethod: existing.PaymentMethod,
				Price:         existing.Price,
			}
		}
	}
	if !validImagePaymentSettings(payment) {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.image_payment_settings_required", nil)
		return true
	}
	if h.rejectDisabledImagePaymentMethod(c, payment) {
		return true
	}
	if h.rejectImagePaymentPriceOverLimit(c, payment) {
		return true
	}
	return false
}

func hasPaidPostImages(images []repositories.PostImageInput) bool {
	for _, image := range images {
		if !image.IsFreePreview {
			return true
		}
	}
	return false
}

func validImagePaymentSettings(payment *repositories.PaymentSettingsInput) bool {
	if payment == nil || !payment.Enabled || payment.Price <= 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(payment.PaymentMethod)) {
	case "balance", "points":
		return true
	default:
		return false
	}
}

func normalizeImagePaymentSettings(images []repositories.PostImageInput, payment *repositories.PaymentSettingsInput) *repositories.PaymentSettingsInput {
	free, paid := repositories.PostImageAccessCounts(images)
	if paid == 0 {
		return &repositories.PaymentSettingsInput{Enabled: false, FreePreviewCount: free}
	}
	if payment == nil {
		return nil
	}
	normalized := *payment
	normalized.FreePreviewCount = free
	normalized.HideAll = false
	return &normalized
}

func normalizePaymentMethodForResponse(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "points":
		return "points"
	default:
		return "balance"
	}
}

func (h NativeHandlers) listImagesResponse(images []domain.PostImage) []gin.H {
	out := make([]gin.H, 0, len(images))
	for _, image := range images {
		out = append(out, gin.H{"url": h.signFileURL(image.ImageURL), "watermarkTraceToken": image.WatermarkTraceToken, "isFreePreview": image.IsFreePreview, "isProtected": image.IsProtected, "sortOrder": image.SortOrder})
	}
	return out
}

func (h NativeHandlers) detailImagesResponse(images []domain.PostImage) []gin.H {
	images = sortedImages(images)
	out := make([]gin.H, 0, len(images))
	for _, image := range images {
		out = append(out, gin.H{"id": int(image.ID), "url": h.signFileURL(image.ImageURL), "watermarkTraceToken": image.WatermarkTraceToken, "isFreePreview": image.IsFreePreview, "isProtected": image.IsProtected, "sortOrder": image.SortOrder})
	}
	return out
}

func (h NativeHandlers) videosResponse(videos []domain.PostVideo) []gin.H {
	out := make([]gin.H, 0, len(videos))
	for _, video := range videos {
		out = append(out, gin.H{
			"id":                int(video.ID),
			"video_url":         h.signFileURL(video.VideoURL),
			"cover_url":         h.signFileURLPtr(video.CoverURL),
			"dash_url":          h.signFileURLPtr(video.DashURL),
			"preview_video_url": h.signFileURLPtr(video.PreviewVideoURL),
		})
	}
	return out
}

func (h NativeHandlers) attachmentResponse(attachments []domain.PostAttachment) any {
	if len(attachments) == 0 {
		return nil
	}
	item := attachments[0]
	return gin.H{"url": h.signFileURL(item.AttachmentURL), "filename": item.Filename, "filesize": item.Filesize}
}

func paymentSettingsResponse(payment *domain.PostPaymentSetting) any {
	if payment == nil {
		return nil
	}
	return gin.H{"enabled": payment.Enabled, "paymentType": payment.PaymentType, "paymentMethod": normalizePaymentMethodForResponse(payment.PaymentMethod), "freePreviewCount": payment.FreePreviewCount, "previewDuration": payment.PreviewDuration, "price": payment.Price, "hideAll": payment.HideAll}
}

func (h NativeHandlers) recommendedUsersResponse(users []repositories.RecommendedUser) []gin.H {
	out := make([]gin.H, 0, len(users))
	for _, item := range users {
		user := item.User
		out = append(out, gin.H{"id": int(user.ID), "user_id": user.UserID, "nickname": user.Nickname, "avatar": h.signFileURLPtr(user.Avatar), "bio": user.Bio, "fans_count": user.FansCount, "verified": user.Verified, "post_count": item.PostCount})
	}
	return out
}

func videoCenterPostResponse(raw map[string]any, upstream string) gin.H {
	id := intFromAnyFallback(raw["id"], 0)
	image := rewriteVideoCenterProxyURL(stringFromMap(raw, "first_image"), upstream)
	return gin.H{
		"id":                "vc_" + strconv.Itoa(id),
		"user_id":           intFromAnyFallback(raw["author_uid"], 0),
		"title":             stringFromMap(raw, "title"),
		"content":           "",
		"category_id":       nil,
		"category":          nil,
		"type":              repositories.PostTypeVideoCenter,
		"view_count":        intFromAnyFallback(raw["purchases_count"], 0),
		"like_count":        0,
		"collect_count":     0,
		"comment_count":     0,
		"created_at":        raw["created_at"],
		"is_draft":          false,
		"visibility":        repositories.VisibilityPublic,
		"nickname":          "\u89c6\u9891\u4e2d\u5fc3",
		"user_avatar":       nil,
		"author_account":    nil,
		"author_auto_id":    nil,
		"verified":          0,
		"avatar":            nil,
		"author":            "\u89c6\u9891\u4e2d\u5fc3",
		"images":            videoCenterImages(image),
		"cover_url":         nilIfEmpty(image),
		"preview_video_url": nilIfEmpty(stringFromMap(raw, "preview_video_url")),
		"duration":          intFromAnyFallback(raw["duration"], 0),
		"friendy_date":      stringFromMap(raw, "friendy_date"),
		"tags":              []gin.H{},
		"liked":             false,
		"collected":         false,
		"video_center_url":  strings.TrimRight(upstream, "/") + "/pyvideo2/play_with_post_id?post_id=" + strconv.Itoa(id),
	}
}

func rewriteVideoCenterProxyURL(value string, upstream string) string {
	if value == "" {
		return value
	}
	upstream = strings.TrimRight(upstream, "/")
	if after, ok := strings.CutPrefix(value, upstream); ok {
		return "/api/pyvideo-api-proxy" + after
	}
	return value
}

func videoCenterImages(value string) []gin.H {
	if value == "" {
		return []gin.H{}
	}
	return []gin.H{{"url": value}}
}

func nilIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func stringFromMap(raw map[string]any, key string) string {
	value, _ := raw[key].(string)
	return value
}

func intFromAnyFallback(value any, fallback int) int {
	if parsed, ok := intFromAny(value); ok {
		return parsed
	}
	return fallback
}

func postBundleIDs(bundles []repositories.PostBundle) []int64 {
	out := make([]int64, 0, len(bundles))
	for _, bundle := range bundles {
		out = append(out, bundle.Post.ID)
	}
	return out
}
