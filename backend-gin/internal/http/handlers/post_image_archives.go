package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
)

const postImageArchivePackageKind = "post_archive"

func (h NativeHandlers) schedulePostImageArchive(postID int64) {
	if postID <= 0 || h.DB == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_, _, _ = h.ensurePostImageArchive(ctx, postID)
	}()
}

func (h NativeHandlers) PostImageArchiveStatus(c *gin.Context) {
	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || postID <= 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgPostIDMissing, nil)
		return
	}
	userID := currentUserID(c)
	post, images, payment, canViewPaid, err := h.postImageArchiveContext(c.Request.Context(), postID, userID)
	if h.writeContentError(c, err, false) {
		return
	}
	threshold := h.imageArchiveThreshold()
	base := gin.H{
		"eligible":         h.imageArchiveEnabled() && len(images) > threshold,
		"enabled":          h.imageArchiveEnabled(),
		"imageCount":       len(images),
		"threshold":        threshold,
		"requiresPurchase": payment != nil && payment.Enabled && !canViewPaid,
	}
	if !h.imageArchiveEnabled() || len(images) <= threshold {
		base["status"] = "disabled"
		c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": base})
		return
	}
	if payment != nil && payment.Enabled && !canViewPaid {
		response.JSON(c, http.StatusPaymentRequired, response.CodeValidationError, "error.purchase_required", base)
		return
	}
	if postImagesContainProtected(images) {
		base["mode"] = "protected"
		base["status"] = "pending"
		if userID > 0 {
			if job, lookupErr := h.activeProtectedPackageJob(c, postID, userID); lookupErr == nil && job != nil {
				maps.Copy(base, h.protectedPackageResponse(*job, h.refreshProtectedPackageQueueEstimate(c, job)))
			}
		}
		c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": base})
		return
	}

	job, queueCount, err := h.ensurePostImageArchive(c.Request.Context(), post.ID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
		return
	}
	if job == nil {
		base["mode"] = "shared"
		base["status"] = "disabled"
		c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": base})
		return
	}
	maps.Copy(base, h.postImageArchiveResponse(*job, queueCount, len(images), threshold))
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": base})
}

func (h NativeHandlers) DownloadPostImageArchive(c *gin.Context) {
	jobID := strings.TrimSpace(c.Param("jobId"))
	var job domain.ImageProtectionJob
	err := h.DB.WithContext(c.Request.Context()).
		Where("job_id = ? AND package_kind = ?", jobID, postImageArchivePackageKind).
		First(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "error.image_archive_missing", nil)
		return
	}
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
		return
	}
	_, images, payment, canViewPaid, err := h.postImageArchiveContext(c.Request.Context(), job.PostID, currentUserID(c))
	if h.writeContentError(c, err, false) {
		return
	}
	if payment != nil && payment.Enabled && !canViewPaid {
		response.JSON(c, http.StatusPaymentRequired, response.CodeValidationError, "error.purchase_required", nil)
		return
	}
	if len(images) <= h.imageArchiveThreshold() || postImagesContainProtected(images) || postImageArchiveSignature(images) != job.SourceSignature {
		h.expirePostImageArchiveJob(c.Request.Context(), &job, "archive_source_changed")
		response.JSON(c, http.StatusGone, response.CodeValidationError, "error.image_archive_expired", nil)
		return
	}
	if job.Status != protectedPackageCompleted || job.PackagePath == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.image_archive_not_ready", nil)
		return
	}
	if !h.safePostImageArchivePath(job.PackagePath) {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "error.image_archive_missing", nil)
		return
	}
	file, err := os.Open(job.PackagePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			h.expirePostImageArchiveJob(c.Request.Context(), &job, "package_missing")
		}
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "error.image_archive_missing", nil)
		return
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
		return
	}
	filename := "post-" + strconv.FormatInt(job.PostID, 10) + "-images.zip"
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Header("Cache-Control", "private, no-store")
	http.ServeContent(c.Writer, c.Request, filename, stat.ModTime(), file)
}

func (h NativeHandlers) ensurePostImageArchive(ctx context.Context, postID int64) (*domain.ImageProtectionJob, int, error) {
	if h.DB == nil || postID <= 0 {
		return nil, 0, nil
	}
	var post domain.Post
	if err := h.DB.WithContext(ctx).Select("id", "user_id", "is_draft").Where("id = ?", postID).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	var images []domain.PostImage
	if err := h.DB.WithContext(ctx).Where("post_id = ?", postID).Order("sort_order ASC, id ASC").Find(&images).Error; err != nil {
		return nil, 0, err
	}
	if post.IsDraft || !h.imageArchiveEnabled() || len(images) <= h.imageArchiveThreshold() || postImagesContainProtected(images) {
		h.expirePostImageArchiveJobs(ctx, postID, "archive_not_eligible")
		return nil, 0, nil
	}

	signature := postImageArchiveSignature(images)
	var active domain.ImageProtectionJob
	err := h.DB.WithContext(ctx).
		Where("post_id = ? AND package_kind = ? AND status IN ?", postID, postImageArchivePackageKind, []string{protectedPackageQueued, protectedPackageProcessing, protectedPackageCompleted}).
		Order("created_at DESC").
		First(&active).Error
	if err == nil {
		if active.SourceSignature == signature && active.ProtectedImageCount == len(images) && h.protectedPackageJobReusable(&active) {
			position, eta, queueCount := 0, 0, 0
			if h.Queue != nil && (active.Status == protectedPackageQueued || active.Status == protectedPackageProcessing) {
				position, eta, queueCount = h.Queue.ImageProtectionQueueEstimate(ctx, active.JobID)
				active.QueuePosition = position
				active.EstimatedWaitSeconds = eta
			}
			return &active, queueCount, nil
		}
		h.expirePostImageArchiveJob(ctx, &active, "archive_source_changed")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, 0, err
	}

	now := time.Now()
	job := domain.ImageProtectionJob{
		JobID:               newProtectedPackageJobID(),
		PostID:              postID,
		UserID:              0,
		AuthorID:            post.UserID,
		PackageKind:         postImageArchivePackageKind,
		SourceSignature:     signature,
		Status:              protectedPackageQueued,
		Progress:            0,
		ProtectedImageCount: len(images),
		CurrentStep:         "queued",
		Retryable:           true,
		CreatedAt:           now,
		UpdatedAt:           &now,
	}
	if err := h.DB.WithContext(ctx).Create(&job).Error; err != nil {
		return nil, 0, err
	}
	if h.Queue == nil || !h.Queue.Available() {
		message := "image archive queue unavailable"
		errorCode := "queue_unavailable"
		job.Status = protectedPackageFailed
		job.ErrorMessage = &message
		job.ErrorCode = &errorCode
		_ = h.DB.WithContext(ctx).Model(&domain.ImageProtectionJob{}).Where("id = ?", job.ID).Updates(map[string]any{
			"status":        job.Status,
			"error_message": job.ErrorMessage,
			"error_code":    job.ErrorCode,
			"updated_at":    time.Now(),
		}).Error
		return &job, 0, nil
	}
	info, err := h.Queue.EnqueuePostImageArchive(ctx, job.JobID, postImageIDs(images))
	if err != nil {
		message := err.Error()
		errorCode := "queue_unavailable"
		job.Status = protectedPackageFailed
		job.ErrorMessage = &message
		job.ErrorCode = &errorCode
		_ = h.DB.WithContext(ctx).Model(&domain.ImageProtectionJob{}).Where("id = ?", job.ID).Updates(map[string]any{
			"status":        job.Status,
			"error_message": job.ErrorMessage,
			"error_code":    job.ErrorCode,
			"updated_at":    time.Now(),
		}).Error
		return &job, 0, nil
	}
	if position, ok := intFromAny(info["queuePosition"]); ok {
		job.QueuePosition = position
	}
	if eta, ok := intFromAny(info["estimatedWaitSeconds"]); ok {
		job.EstimatedWaitSeconds = eta
	}
	queueCount, _ := intFromAny(info["queueCount"])
	_ = h.DB.WithContext(ctx).Model(&domain.ImageProtectionJob{}).Where("id = ?", job.ID).Updates(map[string]any{
		"queue_position":         job.QueuePosition,
		"estimated_wait_seconds": job.EstimatedWaitSeconds,
		"updated_at":             time.Now(),
	}).Error
	return &job, queueCount, nil
}

func (h NativeHandlers) postImageArchiveContext(ctx context.Context, postID, userID int64) (domain.Post, []domain.PostImage, *domain.PostPaymentSetting, bool, error) {
	repo := repositories.NewContentRepository(h.DB)
	post, err := repo.PostExists(ctx, postID)
	if err != nil {
		return domain.Post{}, nil, nil, false, err
	}
	canView, err := repo.CanViewPost(ctx, post.UserID, post.Visibility, userID)
	if err != nil {
		return domain.Post{}, nil, nil, false, err
	}
	if !canView {
		return domain.Post{}, nil, nil, false, repositories.ErrContentForbidden
	}
	var images []domain.PostImage
	if err := h.DB.WithContext(ctx).Where("post_id = ?", postID).Order("sort_order ASC, id ASC").Find(&images).Error; err != nil {
		return domain.Post{}, nil, nil, false, err
	}
	var payment domain.PostPaymentSetting
	var paymentPtr *domain.PostPaymentSetting
	paid := false
	if err := h.DB.WithContext(ctx).Where("post_id = ?", postID).First(&payment).Error; err == nil && payment.Enabled {
		paymentPtr = &payment
		paid = true
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Post{}, nil, nil, false, err
	}
	canViewPaid := !paid || post.UserID == userID
	if paid && !canViewPaid && userID > 0 {
		purchase, err := repositories.NewBalanceRepository(h.DB).CheckPurchase(ctx, userID, postID)
		if err != nil {
			return domain.Post{}, nil, nil, false, err
		}
		canViewPaid = purchase != nil
	}
	return *post, images, paymentPtr, canViewPaid, nil
}

func (h NativeHandlers) postImageArchiveResponse(job domain.ImageProtectionJob, queueCount, imageCount, threshold int) gin.H {
	payload := h.protectedPackageResponse(job, queueCount)
	payload["mode"] = "shared"
	payload["eligible"] = true
	payload["enabled"] = true
	payload["imageCount"] = imageCount
	payload["threshold"] = threshold
	payload["requiresPurchase"] = false
	if job.Status == protectedPackageCompleted && job.PackagePath != "" {
		payload["downloadUrl"] = "/api/image-archives/" + job.JobID + "/download"
	}
	return payload
}

func (h NativeHandlers) expirePostImageArchiveJobs(ctx context.Context, postID int64, errorCode string) {
	var jobs []domain.ImageProtectionJob
	if err := h.DB.WithContext(ctx).
		Where("post_id = ? AND package_kind = ? AND status <> ?", postID, postImageArchivePackageKind, protectedPackageExpired).
		Find(&jobs).Error; err != nil {
		return
	}
	for idx := range jobs {
		h.expirePostImageArchiveJob(ctx, &jobs[idx], errorCode)
	}
}

func (h NativeHandlers) expirePostImageArchiveJob(ctx context.Context, job *domain.ImageProtectionJob, errorCode string) {
	if job == nil || h.DB == nil {
		return
	}
	if job.PackagePath != "" && h.safePostImageArchivePath(job.PackagePath) {
		_ = os.RemoveAll(filepath.Dir(job.PackagePath))
	}
	now := time.Now()
	_ = h.DB.WithContext(ctx).Model(&domain.ImageProtectionJob{}).Where("id = ?", job.ID).Updates(map[string]any{
		"status":       protectedPackageExpired,
		"package_path": "",
		"expires_at":   now,
		"updated_at":   now,
		"retryable":    true,
		"error_code":   errorCode,
	}).Error
}

func (h NativeHandlers) safePostImageArchivePath(path string) bool {
	root := filepath.Join(serviceAbsPathForHandler(h.Config.Upload.RootDir), "archives")
	absolute, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(root, absolute)
	return err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func serviceAbsPathForHandler(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(absolute)
}

func postImagesContainProtected(images []domain.PostImage) bool {
	for _, image := range repositories.NormalizePostImagesForAccess(images) {
		if image.IsProtected {
			return true
		}
	}
	return false
}

func postImageArchiveSignature(images []domain.PostImage) string {
	hash := sha256.New()
	for _, image := range sortedImages(images) {
		_, _ = fmt.Fprintf(hash, "%d\x00%d\x00%s\x00%t\x00%t\n", image.ID, image.SortOrder, image.ImageURL, image.IsFreePreview, image.IsProtected)
	}
	return hex.EncodeToString(hash.Sum(nil))
}
