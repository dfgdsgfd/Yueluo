package handlers

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
	"yuem-go/backend-gin/internal/services"
)

const (
	msgUploadNoFile          = "\u6ca1\u6709\u4e0a\u4f20\u6587\u4ef6"
	msgUploadNoChunk         = "\u6ca1\u6709\u4e0a\u4f20\u5206\u7247"
	msgUploadMissingParams   = "\u7f3a\u5c11\u5fc5\u8981\u53c2\u6570"
	msgUploadOK              = "\u4e0a\u4f20\u6210\u529f"
	msgUploadFailed          = "\u4e0a\u4f20\u5931\u8d25"
	msgUploadImagehostFailed = "\u56fe\u5e8a\u4e0a\u4f20\u5931\u8d25"
	msgUploadUnsupported     = "\u4e0d\u652f\u6301\u7684\u4e0a\u4f20\u7b56\u7565"
	msgUploadOnlyImages      = "\u53ea\u5141\u8bb8\u4e0a\u4f20\u56fe\u7247\u6587\u4ef6"
	msgUploadOnlyVideo       = "\u53ea\u652f\u6301\u89c6\u9891\u6587\u4ef6"
	msgUploadOnlyAPK         = "\u53ea\u5141\u8bb8\u4e0a\u4f20 APK \u6216 APKS \u6587\u4ef6"
	msgUploadBadAttachment   = "\u4e0d\u652f\u6301\u7684\u9644\u4ef6\u7c7b\u578b"
	msgChunkVerifyOK         = "\u9a8c\u8bc1\u5b8c\u6210"
	msgChunkSaveOK           = "\u5206\u7247\u4e0a\u4f20\u6210\u529f"
	msgChunkSaveFailed       = "\u5206\u7247\u4fdd\u5b58\u5931\u8d25"
	msgChunkMergeFailed      = "\u5206\u7247\u5408\u5e76\u5931\u8d25"
	msgChunkIncomplete       = "\u5206\u7247\u4e0d\u5b8c\u6574\uff0c\u7f3a\u5c11: "
	msgVideoUploadOK         = "\u89c6\u9891\u4e0a\u4f20\u6210\u529f"
	msgImageUploadOK         = "\u56fe\u7247\u4e0a\u4f20\u6210\u529f"
	msgAPKUploadOK           = "APK\u4e0a\u4f20\u6210\u529f"
)

func (h NativeHandlers) UploadSingle(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgUploadNoFile, nil)
		return
	}
	if !strings.HasPrefix(file.Header.Get("Content-Type"), "image/") {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgUploadOnlyImages, nil)
		return
	}
	data, err := readMultipartFile(file)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgUploadFailed, nil)
		return
	}
	stored, errMsg, ok := h.storeImage(c, services.NormalizeImagePurpose(c.PostForm("purpose")), data, file.Filename, file.Header.Get("Content-Type"))
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, errMsg, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    response.CodeSuccess,
		"message": msgUploadOK,
		"data":    h.imageUploadResponse(stored, file.Filename, file.Size),
	})
}

func (h NativeHandlers) UploadAvatar(c *gin.Context) {
	h.uploadProfileImage(c, "avatar", "avatar", h.Config.Upload.AvatarDir, services.ImagePurposeAvatar, repositories.PointsTaskSetAvatar)
}

func (h NativeHandlers) UploadBanner(c *gin.Context) {
	h.uploadProfileImage(c, "background", "banner", h.Config.Upload.BannerDir, services.ImagePurposeBackground, repositories.PointsTaskSetBackground)
}

func (h NativeHandlers) uploadProfileImage(c *gin.Context, column, fileType, uploadDir string, purpose services.ImagePurpose, awardTask string) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgUploadNoFile, nil)
		return
	}
	if !strings.HasPrefix(file.Header.Get("Content-Type"), "image/") {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgUploadOnlyImages, nil)
		return
	}
	data, err := readMultipartFile(file)
	if err != nil {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgUploadFailed, nil)
		return
	}
	processor := h.Images
	if processor == nil {
		secret := h.Config.WebP.HiddenWatermark.Secret
		if secret == "" {
			secret = h.Config.Upload.FileSigning.Secret
		}
		processor = services.NewImageProcessorWithRemote(
			h.Settings,
			secret,
			h.Config.Upload.Image.MaxSizeBytes,
			nil,
			services.HiddenWatermarkRemoteClientConfigFromConfig(h.Config),
		)
	}
	processed, err := processor.Process(c.Request.Context(), services.ProcessImageInput{
		Data: data, Filename: file.Filename, ContentType: file.Header.Get("Content-Type"), Purpose: purpose,
		User: &services.RequestUser{ID: user.ID, UserID: user.UserID, Nickname: user.Nickname, Username: user.Username},
	})
	if err != nil {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, err.Error(), nil)
		return
	}
	workDir, err := h.profileImageWorkDir()
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgUploadFailed, nil)
		return
	}
	defer os.RemoveAll(workDir)
	stagedPath := filepath.Join(workDir, "profile."+processed.Format)
	if err := os.WriteFile(stagedPath, processed.Data, 0600); err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgUploadFailed, nil)
		return
	}
	dir := absPath(uploadDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgUploadFailed, nil)
		return
	}
	hash := profileImageXXH3(processed.Data)
	filename := profileImageLegacyFilename(user.ID, processed.Format)
	finalPath := filepath.Join(dir, filename)
	partPath := finalPath + ".part-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := copyFileContents(stagedPath, partPath); err != nil {
		_ = os.Remove(partPath)
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgUploadFailed, nil)
		return
	}
	defer os.Remove(partPath)
	backupPath := finalPath + ".backup-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	backupCreated := false
	installed := false
	var oldURL string
	canonicalURL := profileImageHashedURL(fileType, user.ID, hash, processed.Format)
	hashColumn := profileImageHashColumnForStorageColumn(column)
	err = h.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		var current domain.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", user.ID).First(&current).Error; err != nil {
			return err
		}
		if column == "avatar" {
			oldURL = stringPtrValue(current.Avatar)
		} else {
			oldURL = stringPtrValue(current.Background)
		}
		if _, statErr := os.Stat(finalPath); statErr == nil {
			if err := os.Rename(finalPath, backupPath); err != nil {
				return err
			}
			backupCreated = true
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return statErr
		}
		if err := os.Rename(partPath, finalPath); err != nil {
			return err
		}
		installed = true
		updates := map[string]any{column: canonicalURL}
		if hashColumn != "" {
			updates[hashColumn] = hash
		}
		return tx.Model(&domain.User{}).Where("id = ?", user.ID).Updates(updates).Error
	})
	if err != nil {
		if installed {
			_ = os.Remove(finalPath)
		}
		if backupCreated {
			_ = os.Rename(backupPath, finalPath)
		}
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgUploadFailed, nil)
		return
	}
	_ = os.Remove(backupPath)
	h.removeObsoleteProfileImage(oldURL, finalPath)
	awards := h.awardProfileTasksBestEffort(c, user.ID, []profileAwardTask{{taskType: awardTask, reason: "profile image updated"}})
	stored := storedImage{URL: canonicalURL, Processed: processed}
	payload := h.imageUploadResponse(stored, file.Filename, file.Size)
	if len(awards) > 0 {
		payload["points_awards"] = awards
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": msgUploadOK, "data": payload})
}

func (h NativeHandlers) profileImageWorkDir() (string, error) {
	if h.TempStorage != nil {
		return h.TempStorage.NewWorkDir("profile-media")
	}
	root := absPath(h.Config.Upload.Temp.RootDir)
	if err := os.MkdirAll(root, 0700); err != nil {
		return "", err
	}
	return os.MkdirTemp(root, "profile-media-")
}

func (h NativeHandlers) removeObsoleteProfileImage(raw, currentPath string) {
	canonical, _, ok := h.localFilePathFromURL(raw)
	if !ok {
		return
	}
	parts := strings.SplitN(strings.TrimPrefix(canonical, "/api/file/"), "/", 2)
	if len(parts) != 2 || (parts[0] != "avatar" && parts[0] != "banner") {
		return
	}
	fileType := parts[0]
	name := profileImageDiskSubPath(fileType, filepath.Base(canonical))
	for _, dir := range h.uploadTypeDirs(fileType) {
		candidate := filepath.Join(dir, name)
		if filepath.Clean(candidate) != filepath.Clean(currentPath) {
			_ = os.Remove(candidate)
		}
	}
}

func (h NativeHandlers) UploadMultiple(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil || form == nil || len(form.File["files"]) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "data": nil, "message": msgUploadNoFile})
		return
	}
	files := form.File["files"]
	maxFiles := h.maxPostImages()
	if len(files) > maxFiles {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.upload_image_limit", gin.H{"maxFiles": maxFiles})
		return
	}
	uploaded := []gin.H{}
	errorsList := []gin.H{}
	purpose := services.NormalizeImagePurpose(c.PostForm("purpose"))
	for index, file := range files {
		if !strings.HasPrefix(file.Header.Get("Content-Type"), "image/") {
			errorsList = append(errorsList, gin.H{"batchIndex": index, "file": file.Filename, "error": msgUploadOnlyImages})
			continue
		}
		data, err := readMultipartFile(file)
		if err != nil {
			errorsList = append(errorsList, gin.H{"batchIndex": index, "file": file.Filename, "error": msgUploadFailed})
			continue
		}
		stored, errMsg, ok := h.storeImage(c, purpose, data, file.Filename, file.Header.Get("Content-Type"))
		if !ok {
			errorsList = append(errorsList, gin.H{"batchIndex": index, "file": file.Filename, "error": errMsg})
			continue
		}
		item := h.imageUploadResponse(stored, file.Filename, file.Size)
		item["batchIndex"] = index
		uploaded = append(uploaded, item)
	}
	if len(uploaded) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "data": nil, "message": "\u6240\u6709\u56fe\u7247\u4e0a\u4f20\u5931\u8d25"})
		return
	}
	message := "\u6240\u6709\u56fe\u7247\u4e0a\u4f20\u6210\u529f"
	if len(errorsList) > 0 {
		message = strconv.Itoa(len(uploaded)) + "\u5f20\u4e0a\u4f20\u6210\u529f\uff0c" + strconv.Itoa(len(errorsList)) + "\u5f20\u5931\u8d25"
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"uploaded":     uploaded,
			"errors":       errorsList,
			"total":        len(form.File["files"]),
			"successCount": len(uploaded),
			"errorCount":   len(errorsList),
		},
		"message": message,
	})
}

func (h NativeHandlers) UploadVideo(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "\u6ca1\u6709\u4e0a\u4f20\u89c6\u9891\u6587\u4ef6", nil)
		return
	}
	if !strings.HasPrefix(file.Header.Get("Content-Type"), "video/") {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgUploadOnlyVideo, nil)
		return
	}
	data, err := readMultipartFile(file)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgUploadFailed, nil)
		return
	}
	url, filePath, errMsg, ok := h.storeVideo(data, file.Filename, file.Header.Get("Content-Type"))
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, errMsg, nil)
		return
	}
	var coverURL any
	var coverSignedURL any
	if thumbnail, err := c.FormFile("thumbnail"); err == nil && thumbnail != nil {
		thumbData, readErr := readMultipartFile(thumbnail)
		if readErr == nil {
			if stored, _, ok := h.storeImage(c, services.ImagePurposeCover, thumbData, thumbnail.Filename, thumbnail.Header.Get("Content-Type")); ok {
				coverURL = stored.URL
				coverSignedURL = h.signFileURL(stored.URL)
			}
		}
	}
	coverValue, _ := coverURL.(string)
	transcodingJob, transcoding := h.enqueueVideoTranscoding(c.Request.Context(), services.VideoTranscodingInput{
		VideoURL:   url,
		SourcePath: filePath,
		CoverURL:   coverValue,
	})
	c.JSON(http.StatusOK, gin.H{
		"code":    response.CodeSuccess,
		"message": msgUploadOK,
		"data": gin.H{
			"originalname":   file.Filename,
			"size":           file.Size,
			"url":            url,
			"signedUrl":      h.signFileURL(url),
			"filePath":       filePath,
			"coverUrl":       coverURL,
			"coverSignedUrl": coverSignedURL,
			"transcoding":    transcoding,
			"transcodingJob": transcodingJob,
		},
	})
}
