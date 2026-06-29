package handlers

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/services"
)

func (h NativeHandlers) UploadChunkConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    response.CodeSuccess,
		"message": "\u83b7\u53d6\u5206\u7247\u914d\u7f6e\u6210\u529f",
		"data": gin.H{
			"chunkSize":           h.Config.Upload.Video.Chunk.ChunkSize,
			"maxFileSize":         h.Config.Upload.Video.MaxSizeBytes,
			"imageMaxSize":        100 * 1024 * 1024,
			"imageChunkThreshold": 2 * 1024 * 1024,
		},
	})
}

func (h NativeHandlers) UploadChunkVerify(c *gin.Context) {
	identifier := c.Query("identifier")
	chunkNumber := positiveIntQuery(c, "chunkNumber", 0)
	if identifier == "" || chunkNumber == 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgUploadMissingParams, nil)
		return
	}
	exists, valid := h.verifyChunk(identifier, chunkNumber, c.Query("md5"))
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": msgChunkVerifyOK, "data": gin.H{"exists": exists, "valid": valid}})
}

func (h NativeHandlers) UploadChunk(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgUploadNoChunk, nil)
		return
	}
	identifier := c.PostForm("identifier")
	chunkNumber, errNumber := strconv.Atoi(c.PostForm("chunkNumber"))
	totalChunks, errTotal := strconv.Atoi(c.PostForm("totalChunks"))
	if identifier == "" || errNumber != nil || errTotal != nil || chunkNumber <= 0 || totalChunks <= 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgUploadMissingParams, nil)
		return
	}
	data, err := readMultipartFile(file)
	if err != nil || !h.saveChunk(identifier, chunkNumber, data) {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgChunkSaveFailed, nil)
		return
	}
	uploaded, _, complete := h.checkUploadComplete(identifier, totalChunks)
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": msgChunkSaveOK, "data": gin.H{
		"chunkNumber": chunkNumber,
		"uploaded":    len(uploaded),
		"total":       totalChunks,
		"complete":    complete,
	}})
}

type chunkMergeRequest struct {
	Identifier  string `json:"identifier"`
	TotalChunks any    `json:"totalChunks"`
	Filename    string `json:"filename"`
	Purpose     string `json:"purpose"`
}

func (h NativeHandlers) UploadChunkMerge(c *gin.Context) {
	body := bindChunkMerge(c)
	total, ok := intFromAny(body.TotalChunks)
	if body.Identifier == "" || body.Filename == "" || !ok || total <= 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgUploadMissingParams, nil)
		return
	}
	filePath, url, errMsg, ok := h.mergeChunksToFile(body.Identifier, total, body.Filename, h.Config.Upload.Video.LocalUploadDir, "videos")
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, errMsg, nil)
		return
	}
	transcodingJob, transcoding := h.enqueueVideoTranscoding(c.Request.Context(), services.VideoTranscodingInput{
		VideoURL:   url,
		SourcePath: filePath,
	})
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": msgVideoUploadOK, "data": gin.H{
		"originalname":   body.Filename,
		"url":            url,
		"signedUrl":      h.signFileURL(url),
		"filePath":       filePath,
		"coverUrl":       nil,
		"coverSignedUrl": nil,
		"transcoding":    transcoding,
		"transcodingJob": transcodingJob,
		"videoInfo":      nil,
	}})
}

func (h NativeHandlers) UploadChunkMergeImage(c *gin.Context) {
	body := bindChunkMerge(c)
	total, ok := intFromAny(body.TotalChunks)
	if body.Identifier == "" || body.Filename == "" || !ok || total <= 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgUploadMissingParams, nil)
		return
	}
	data, errMsg, ok := h.mergeChunksToBytes(body.Identifier, total)
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, errMsg, nil)
		return
	}
	stored, errMsg, ok := h.storeImage(c, services.NormalizeImagePurpose(body.Purpose), data, body.Filename, contentTypeForPath(body.Filename))
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, errMsg, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": msgImageUploadOK, "data": h.imageUploadResponse(stored, body.Filename, int64(len(data)))})
}

func (h NativeHandlers) UploadChunkMergeAPK(c *gin.Context) {
	body := bindChunkMerge(c)
	total, ok := intFromAny(body.TotalChunks)
	ext := strings.ToLower(filepath.Ext(body.Filename))
	if body.Identifier == "" || body.Filename == "" || !ok || total <= 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgUploadMissingParams, nil)
		return
	}
	if ext != ".apk" && ext != ".apks" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgUploadOnlyAPK, nil)
		return
	}
	_, url, errMsg, ok := h.mergeChunksToFile(body.Identifier, total, body.Filename, "uploads/apk", "apk")
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, errMsg, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": msgAPKUploadOK, "data": gin.H{"originalname": body.Filename, "url": url, "signedUrl": h.signFileURL(url)}})
}

func (h NativeHandlers) UploadAttachment(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgUploadNoFile, nil)
		return
	}
	if !allowedAttachmentFile(file.Filename, file.Header.Get("Content-Type")) {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgUploadBadAttachment, nil)
		return
	}
	data, err := readMultipartFile(file)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgUploadFailed, nil)
		return
	}
	url, _, errMsg, ok := h.storeLocalFilePreservingName(data, h.Config.Upload.Attachment.LocalUploadDir, "attachments", file.Filename)
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, errMsg, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": msgUploadOK, "data": gin.H{"originalname": file.Filename, "size": file.Size, "url": url, "signedUrl": h.signFileURL(url)}})
}

func (h NativeHandlers) UploadAPK(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgUploadNoFile, nil)
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".apk" && ext != ".apks" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgUploadOnlyAPK, nil)
		return
	}
	data, err := readMultipartFile(file)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgUploadFailed, nil)
		return
	}
	url, _, errMsg, ok := h.storeLocalFile(data, "uploads/apk", "apk", file.Filename)
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, errMsg, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": msgUploadOK, "data": gin.H{"originalname": file.Filename, "size": file.Size, "url": url, "signedUrl": h.signFileURL(url)}})
}
