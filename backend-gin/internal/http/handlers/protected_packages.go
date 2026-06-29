package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
)

const (
	protectedPackageQueued     = "queued"
	protectedPackageProcessing = "processing"
	protectedPackageCompleted  = "completed"
	protectedPackageFailed     = "failed"
	protectedPackageExpired    = "expired"
)

func (h NativeHandlers) CreateProtectedPackage(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || postID <= 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgPostIDMissing, nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
		return
	}
	if !h.imageProtectionEnabled() {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.image_protection_disabled", nil)
		return
	}
	if h.Queue == nil || !h.Queue.Available() {
		response.JSON(c, http.StatusServiceUnavailable, response.CodeError, "error.image_protection_queue_unavailable", nil)
		return
	}
	post, images, payment, canViewPaid, err := h.protectedPackageContext(c, postID, user.ID)
	if h.writeContentError(c, err, false) {
		return
	}
	access := imageAccessForViewer(images, payment, canViewPaid, false)
	if access.ProtectedPackageImageCount <= 0 {
		if access.LockedProtectedImagesCount > 0 {
			response.JSON(c, http.StatusPaymentRequired, response.CodeValidationError, "error.purchase_required", gin.H{"lockedProtectedImagesCount": access.LockedProtectedImagesCount})
			return
		}
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.no_protected_images", nil)
		return
	}
	packageImages := access.ProtectedPackageImages
	if h.imageArchiveEnabled() && len(images) > h.imageArchiveThreshold() {
		if payment != nil && payment.Enabled && !canViewPaid {
			response.JSON(c, http.StatusPaymentRequired, response.CodeValidationError, "error.purchase_required", gin.H{"lockedImagesCount": access.HiddenPaidImagesCount})
			return
		}
		packageImages = sortedImages(images)
	}
	sourceSignature := postImageArchiveSignature(packageImages)
	existing, err := h.activeProtectedPackageJob(c, postID, user.ID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
		return
	}
	if existing != nil {
		if existing.ProtectedImageCount == len(packageImages) && existing.SourceSignature == sourceSignature && h.protectedPackageJobReusable(existing) {
			queueCount := h.refreshProtectedPackageQueueEstimate(c, existing)
			c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": h.protectedPackageResponse(*existing, queueCount)})
			return
		}
		h.expireProtectedPackageJob(c, existing, "package_missing")
	}

	now := time.Now()
	job := domain.ImageProtectionJob{
		JobID:               newProtectedPackageJobID(),
		PostID:              postID,
		UserID:              user.ID,
		AuthorID:            post.UserID,
		PackageKind:         "protected",
		SourceSignature:     sourceSignature,
		Status:              protectedPackageQueued,
		Progress:            0,
		ProtectedImageCount: len(packageImages),
		ProcessedImageCount: 0,
		CurrentStep:         "queued",
		Retryable:           true,
		CreatedAt:           now,
		UpdatedAt:           &now,
	}
	if err := h.DB.WithContext(c.Request.Context()).Create(&job).Error; err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
		return
	}
	info, err := h.Queue.EnqueueImageProtectionPackage(c.Request.Context(), job.JobID, postImageIDs(packageImages))
	if err != nil {
		message := err.Error()
		errorCode := "queue_unavailable"
		_ = h.DB.WithContext(c.Request.Context()).Model(&domain.ImageProtectionJob{}).Where("id = ?", job.ID).Updates(map[string]any{
			"status":        protectedPackageFailed,
			"error_message": &message,
			"error_code":    &errorCode,
			"retryable":     true,
			"updated_at":    time.Now(),
		}).Error
		response.JSON(c, http.StatusServiceUnavailable, response.CodeError, "error.image_protection_queue_unavailable", nil)
		return
	}
	if position, ok := intFromAny(info["queuePosition"]); ok {
		job.QueuePosition = position
	}
	if eta, ok := intFromAny(info["estimatedWaitSeconds"]); ok {
		job.EstimatedWaitSeconds = eta
	}
	queueCount, _ := intFromAny(info["queueCount"])
	_ = h.DB.WithContext(c.Request.Context()).Model(&domain.ImageProtectionJob{}).Where("id = ?", job.ID).Updates(map[string]any{"queue_position": job.QueuePosition, "estimated_wait_seconds": job.EstimatedWaitSeconds, "updated_at": time.Now()}).Error
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": h.protectedPackageResponse(job, queueCount)})
}

func postImageIDs(images []domain.PostImage) []int64 {
	ids := make([]int64, 0, len(images))
	for _, image := range images {
		ids = append(ids, image.ID)
	}
	return ids
}

func (h NativeHandlers) ProtectedPackageStatus(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	job, ok := h.protectedPackageJobForUser(c, user.ID)
	if !ok {
		return
	}
	queueCount := h.refreshProtectedPackageQueueEstimate(c, job)
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": h.protectedPackageResponse(*job, queueCount)})
}

func (h NativeHandlers) ProtectedPackageEvents(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	job, ok := h.protectedPackageJobForUser(c, user.ID)
	if !ok {
		return
	}
	c.Header("Content-Type", "application/x-ndjson")
	c.Header("Cache-Control", "no-cache, no-store")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	encoder := json.NewEncoder(c.Writer)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	heartbeat := time.NewTicker(2 * time.Second)
	defer heartbeat.Stop()
	lastUpdated := time.Time{}
	writeEvent := func(eventType string, current domain.ImageProtectionJob) bool {
		payload := h.protectedPackageResponse(current)
		payload["type"] = eventType
		if err := encoder.Encode(payload); err != nil {
			return false
		}
		c.Writer.Flush()
		return true
	}
	if !writeEvent(protectedPackageEventType(job.Status, false), *job) {
		return
	}
	if job.UpdatedAt != nil {
		lastUpdated = *job.UpdatedAt
	}
	if protectedPackageTerminal(job.Status) {
		return
	}
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-heartbeat.C:
			if !writeEvent("heartbeat", *job) {
				return
			}
		case <-ticker.C:
			var current domain.ImageProtectionJob
			if err := h.DB.WithContext(c.Request.Context()).Where("job_id = ? AND user_id = ?", job.JobID, user.ID).First(&current).Error; err != nil {
				return
			}
			if current.UpdatedAt != nil && !current.UpdatedAt.After(lastUpdated) {
				continue
			}
			job = &current
			if current.UpdatedAt != nil {
				lastUpdated = *current.UpdatedAt
			}
			if !writeEvent(protectedPackageEventType(current.Status, false), current) {
				return
			}
			if protectedPackageTerminal(current.Status) {
				return
			}
		}
	}
}

func protectedPackageTerminal(status string) bool {
	return status == protectedPackageCompleted || status == protectedPackageFailed || status == protectedPackageExpired
}

func protectedPackageEventType(status string, heartbeat bool) string {
	if heartbeat {
		return "heartbeat"
	}
	switch status {
	case protectedPackageCompleted:
		return "result"
	case protectedPackageFailed, protectedPackageExpired:
		return "error"
	default:
		return "progress"
	}
}

func (h NativeHandlers) DownloadProtectedPackage(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	job, ok := h.protectedPackageJobForUser(c, user.ID)
	if !ok {
		return
	}
	if job.Status != protectedPackageCompleted || job.PackagePath == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.protected_package_not_ready", nil)
		return
	}
	if job.ExpiresAt != nil && time.Now().After(*job.ExpiresAt) {
		response.JSON(c, http.StatusGone, response.CodeValidationError, "error.protected_package_expired", nil)
		return
	}
	file, err := os.Open(job.PackagePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			h.expireProtectedPackageJob(c, job, "package_missing")
		}
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "error.protected_package_missing", nil)
		return
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
		return
	}
	filename := "protected-post-" + strconv.FormatInt(job.PostID, 10) + "-" + job.JobID + ".zip"
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Header("Cache-Control", "private, no-store")
	http.ServeContent(c.Writer, c.Request, filepath.Base(filename), stat.ModTime(), file)
}

func (h NativeHandlers) protectedPackageContext(c *gin.Context, postID, userID int64) (domain.Post, []domain.PostImage, *domain.PostPaymentSetting, bool, error) {
	repo := repositories.NewContentRepository(h.DB)
	post, err := repo.PostExists(c.Request.Context(), postID)
	if err != nil {
		return domain.Post{}, nil, nil, false, err
	}
	canView, err := repo.CanViewPost(c.Request.Context(), post.UserID, post.Visibility, userID)
	if err != nil {
		return domain.Post{}, nil, nil, false, err
	}
	if !canView {
		return domain.Post{}, nil, nil, false, repositories.ErrContentForbidden
	}
	var images []domain.PostImage
	if err := h.DB.WithContext(c.Request.Context()).Where("post_id = ?", postID).Order("sort_order ASC, id ASC").Find(&images).Error; err != nil {
		return domain.Post{}, nil, nil, false, err
	}
	var payment domain.PostPaymentSetting
	var paymentPtr *domain.PostPaymentSetting
	paid := false
	if err := h.DB.WithContext(c.Request.Context()).Where("post_id = ?", postID).First(&payment).Error; err == nil && payment.Enabled {
		paymentPtr = &payment
		paid = true
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Post{}, nil, nil, false, err
	}
	canViewPaid := !paid || post.UserID == userID
	if paid && !canViewPaid {
		purchase, err := repositories.NewBalanceRepository(h.DB).CheckPurchase(c.Request.Context(), userID, postID)
		if err != nil {
			return domain.Post{}, nil, nil, false, err
		}
		canViewPaid = purchase != nil
	}
	return *post, images, paymentPtr, canViewPaid, nil
}

func (h NativeHandlers) activeProtectedPackageJob(c *gin.Context, postID, userID int64) (*domain.ImageProtectionJob, error) {
	var job domain.ImageProtectionJob
	now := time.Now()
	err := h.DB.WithContext(c.Request.Context()).
		Where("post_id = ? AND user_id = ? AND (package_kind = ? OR package_kind = '') AND status IN ?", postID, userID, "protected", []string{protectedPackageQueued, protectedPackageProcessing, protectedPackageCompleted}).
		Where("expires_at IS NULL OR expires_at > ?", now).
		Order("created_at DESC").
		First(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (h NativeHandlers) protectedPackageJobReusable(job *domain.ImageProtectionJob) bool {
	if job == nil {
		return false
	}
	if job.Status != protectedPackageCompleted {
		return true
	}
	if job.PackagePath == "" {
		return false
	}
	stat, err := os.Stat(job.PackagePath)
	return err == nil && !stat.IsDir()
}

func (h NativeHandlers) expireProtectedPackageJob(c *gin.Context, job *domain.ImageProtectionJob, errorCode string) {
	if job == nil || h.DB == nil {
		return
	}
	if job.PackagePath != "" {
		_ = os.Remove(job.PackagePath)
	}
	now := time.Now()
	_ = h.DB.WithContext(c.Request.Context()).Model(&domain.ImageProtectionJob{}).Where("id = ?", job.ID).Updates(map[string]any{
		"status":       protectedPackageExpired,
		"package_path": "",
		"expires_at":   now,
		"updated_at":   now,
		"retryable":    true,
		"error_code":   errorCode,
	}).Error
	job.Status = protectedPackageExpired
	job.PackagePath = ""
	job.ExpiresAt = &now
	job.UpdatedAt = &now
	job.Retryable = true
	job.ErrorCode = &errorCode
}

func (h NativeHandlers) protectedPackageJobForUser(c *gin.Context, userID int64) (*domain.ImageProtectionJob, bool) {
	jobID := c.Param("jobId")
	var job domain.ImageProtectionJob
	err := h.DB.WithContext(c.Request.Context()).Where("job_id = ? AND user_id = ?", jobID, userID).First(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "error.protected_package_missing", nil)
		return nil, false
	}
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
		return nil, false
	}
	return &job, true
}

func (h NativeHandlers) refreshProtectedPackageQueueEstimate(c *gin.Context, job *domain.ImageProtectionJob) int {
	if job == nil || h.Queue == nil || (job.Status != protectedPackageQueued && job.Status != protectedPackageProcessing) {
		return 0
	}
	position, eta, queueCount := h.Queue.ImageProtectionQueueEstimate(c.Request.Context(), job.JobID)
	if position == job.QueuePosition && eta == job.EstimatedWaitSeconds {
		return queueCount
	}
	job.QueuePosition = position
	job.EstimatedWaitSeconds = eta
	_ = h.DB.WithContext(c.Request.Context()).Model(&domain.ImageProtectionJob{}).Where("id = ?", job.ID).Updates(map[string]any{"queue_position": position, "estimated_wait_seconds": eta, "updated_at": time.Now()}).Error
	return queueCount
}

func (h NativeHandlers) protectedPackageResponse(job domain.ImageProtectionJob, queueCounts ...int) gin.H {
	downloadURL := any(nil)
	if job.Status == protectedPackageCompleted && job.PackagePath != "" {
		downloadURL = "/api/protected-packages/" + job.JobID + "/download"
	}
	queuePosition := any(nil)
	estimatedWaitSeconds := any(nil)
	if job.QueuePosition > 0 {
		queuePosition = job.QueuePosition
	}
	if job.EstimatedWaitSeconds > 0 {
		estimatedWaitSeconds = job.EstimatedWaitSeconds
	}
	queueCount := 0
	if len(queueCounts) > 0 {
		queueCount = queueCounts[0]
	}
	elapsedSeconds := 0
	if job.StartedAt != nil {
		end := time.Now()
		if job.FinishedAt != nil {
			end = *job.FinishedAt
		}
		elapsedSeconds = max(0, int(end.Sub(*job.StartedAt).Seconds()))
	}
	return gin.H{
		"jobId":                     job.JobID,
		"postId":                    job.PostID,
		"status":                    job.Status,
		"progress":                  job.Progress,
		"queuePosition":             queuePosition,
		"estimatedWaitSeconds":      estimatedWaitSeconds,
		"estimatedRemainingSeconds": job.EstimatedRemainingSeconds,
		"elapsedSeconds":            elapsedSeconds,
		"queueCount":                queueCount,
		"protectedImageCount":       job.ProtectedImageCount,
		"processedImageCount":       job.ProcessedImageCount,
		"currentImageIndex":         protectedPackageCurrentImageIndex(job),
		"currentStep":               job.CurrentStep,
		"activeProfile":             job.ActiveProfile,
		"heartbeatAt":               job.HeartbeatAt,
		"downloadUrl":               downloadURL,
		"errorMessage":              job.ErrorMessage,
		"errorCode":                 job.ErrorCode,
		"retryable":                 job.Retryable,
		"expiresAt":                 job.ExpiresAt,
		"createdAt":                 job.CreatedAt,
		"updatedAt":                 job.UpdatedAt,
	}
}

func protectedPackageCurrentImageIndex(job domain.ImageProtectionJob) int {
	total := max(0, job.ProtectedImageCount)
	processed := min(max(0, job.ProcessedImageCount), total)
	if total == 0 {
		return 0
	}
	switch job.Status {
	case protectedPackageProcessing:
		if processed >= total {
			return total
		}
		switch job.CurrentStep {
		case "", "preparing", "queued", "image_completed", "image_skipped", "completed":
			return processed
		default:
			return processed + 1
		}
	case protectedPackageQueued:
		return processed
	default:
		return processed
	}
}

func newProtectedPackageJobID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(buf[:])
}
