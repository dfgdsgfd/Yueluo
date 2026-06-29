package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
	"yuem-go/backend-gin/internal/services"
)

const (
	msgFeedbackChooseImage    = "\u8bf7\u9009\u62e9\u56fe\u7247\u6587\u4ef6"
	msgFeedbackChooseVideo    = "\u8bf7\u9009\u62e9\u89c6\u9891\u6587\u4ef6"
	msgFeedbackImageFailed    = "\u56fe\u7247\u4e0a\u4f20\u5931\u8d25"
	msgFeedbackVideoFailed    = "\u89c6\u9891\u4e0a\u4f20\u5931\u8d25"
	msgFeedbackContentEmpty   = "\u53cd\u9988\u5185\u5bb9\u4e0d\u80fd\u4e3a\u7a7a"
	msgFeedbackContentTooLong = "\u53cd\u9988\u5185\u5bb9\u4e0d\u80fd\u8d85\u8fc72000\u5b57"
	msgFeedbackImagesInvalid  = "\u56fe\u7247\u683c\u5f0f\u9519\u8bef"
	msgFeedbackImagesTooMany  = "\u6700\u591a\u4e0a\u4f209\u5f20\u56fe\u7247"
	msgFeedbackSubmitOK       = "\u53cd\u9988\u63d0\u4ea4\u6210\u529f\uff0c\u611f\u8c22\u60a8\u7684\u610f\u89c1\uff01"
	msgFeedbackSubmitFailed   = "\u63d0\u4ea4\u5931\u8d25\uff0c\u8bf7\u7a0d\u540e\u91cd\u8bd5"
	msgFeedbackGetFailed      = "\u83b7\u53d6\u5931\u8d25"
	msgFeedbackNotFound       = "\u53cd\u9988\u4e0d\u5b58\u5728"
	msgFeedbackForbidden      = "\u65e0\u6743\u67e5\u770b"
)

func (h NativeHandlers) FeedbackUploadImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgFeedbackChooseImage, nil)
		return
	}
	if !strings.HasPrefix(file.Header.Get("Content-Type"), "image/") {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgUploadOnlyImages, nil)
		return
	}
	data, err := readMultipartFile(file)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgFeedbackImageFailed, nil)
		return
	}
	stored, errMsg, ok := h.storeImage(c, services.ImagePurposeFeedback, data, file.Filename, file.Header.Get("Content-Type"))
	if !ok {
		if errMsg == "" {
			errMsg = msgFeedbackImageFailed
		}
		response.JSON(c, http.StatusBadRequest, response.CodeError, errMsg, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": h.imageUploadResponse(stored, file.Filename, file.Size), "message": msgUploadOK})
}

func (h NativeHandlers) FeedbackUploadVideo(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgFeedbackChooseVideo, nil)
		return
	}
	if !strings.HasPrefix(file.Header.Get("Content-Type"), "video/") {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgUploadOnlyVideo, nil)
		return
	}
	data, err := readMultipartFile(file)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgFeedbackVideoFailed, nil)
		return
	}
	url, _, errMsg, ok := h.storeVideo(data, file.Filename, file.Header.Get("Content-Type"))
	if !ok {
		if errMsg == "" {
			errMsg = msgFeedbackVideoFailed
		}
		response.JSON(c, http.StatusBadRequest, response.CodeError, errMsg, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{"url": url, "signedUrl": h.signFileURL(url)}, "message": msgUploadOK})
}

type feedbackCreateRequest struct {
	Content  string `json:"content"`
	Images   any    `json:"images"`
	VideoURL string `json:"video_url"`
}

func (h NativeHandlers) FeedbackCreate(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	var body feedbackCreateRequest
	_ = c.ShouldBindJSON(&body)
	content := sanitizeMarkdownSubmittedText(body.Content)
	if content == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgFeedbackContentEmpty, nil)
		return
	}
	if len([]rune(content)) > 2000 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgFeedbackContentTooLong, nil)
		return
	}
	images, errMsg, ok := h.feedbackImages(body.Images)
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, errMsg, nil)
		return
	}
	var rawImages datatypes.JSON
	if len(images) > 0 {
		encoded, _ := json.Marshal(images)
		rawImages = datatypes.JSON(encoded)
	}
	var videoURL *string
	if strings.TrimSpace(body.VideoURL) != "" {
		trimmed := h.normalizeFileURLForStorage(body.VideoURL)
		videoURL = &trimmed
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgFeedbackSubmitFailed, nil)
		return
	}
	feedback, err := repositories.NewFeedbackRepository(h.DB).Create(c.Request.Context(), domain.Feedback{
		UserID:   user.ID,
		Content:  content,
		Images:   rawImages,
		VideoURL: videoURL,
		Status:   "pending",
	})
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgFeedbackSubmitFailed, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{"id": strconv.FormatInt(feedback.ID, 10)}, "message": msgFeedbackSubmitOK})
}

func (h NativeHandlers) FeedbackMine(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgFeedbackGetFailed, nil)
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := min(positiveIntQuery(c, "limit", 10), 50)
	total, list, err := repositories.NewFeedbackRepository(h.DB).ListMine(c.Request.Context(), user.ID, page, limit)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgFeedbackGetFailed, nil)
		return
	}
	data := make([]gin.H, 0, len(list))
	for _, item := range list {
		data = append(data, h.feedbackListResponse(item))
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{"data": data, "pagination": pagination(page, limit, total)}, "message": "success"})
}

func (h NativeHandlers) FeedbackDetail(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgFeedbackNotFound, nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgFeedbackGetFailed, nil)
		return
	}
	feedback, owned, err := repositories.NewFeedbackRepository(h.DB).FindOwned(c.Request.Context(), id, user.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgFeedbackNotFound, nil)
		return
	}
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgFeedbackGetFailed, nil)
		return
	}
	if !owned {
		response.JSON(c, http.StatusForbidden, response.CodeForbidden, msgFeedbackForbidden, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": h.feedbackDetailResponse(*feedback), "message": "success"})
}

func (h NativeHandlers) feedbackImages(value any) ([]string, string, bool) {
	if value == nil {
		return nil, "", true
	}
	values, ok := value.([]any)
	if !ok {
		return nil, msgFeedbackImagesInvalid, false
	}
	if len(values) > 9 {
		return nil, msgFeedbackImagesTooMany, false
	}
	out := []string{}
	for _, item := range values {
		text, ok := item.(string)
		if ok && strings.TrimSpace(text) != "" {
			out = append(out, h.normalizeFileURLForStorage(text))
		}
	}
	return out, "", true
}

func (h NativeHandlers) feedbackListResponse(item domain.Feedback) gin.H {
	return gin.H{
		"id":          strconv.FormatInt(item.ID, 10),
		"content":     item.Content,
		"images":      h.feedbackImagesJSON(item.Images),
		"video_url":   h.signFileURLPtr(item.VideoURL),
		"status":      item.Status,
		"admin_reply": item.AdminReply,
		"replied_at":  item.RepliedAt,
		"created_at":  item.CreatedAt,
	}
}

func (h NativeHandlers) feedbackDetailResponse(item domain.Feedback) gin.H {
	body := h.feedbackListResponse(item)
	body["user_id"] = strconv.FormatInt(item.UserID, 10)
	body["updated_at"] = item.UpdatedAt
	return body
}

func (h NativeHandlers) feedbackImagesJSON(raw datatypes.JSON) any {
	if len(raw) == 0 {
		return nil
	}
	var out []string
	if json.Unmarshal(raw, &out) == nil {
		for i := range out {
			out[i] = h.signFileURL(out[i])
		}
		return out
	}
	var anyOut any
	if json.Unmarshal(raw, &anyOut) == nil {
		return anyOut
	}
	return nil
}
