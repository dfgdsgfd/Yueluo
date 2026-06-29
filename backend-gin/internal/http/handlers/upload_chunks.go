package handlers

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func (h NativeHandlers) chunkDir(identifier string) (string, bool) {
	if invalidIdentifier(identifier) {
		return "", false
	}
	root := absPath(h.Config.Upload.Video.Chunk.TempDir)
	return filepath.Join(root, identifier), true
}

func (h NativeHandlers) chunkPath(identifier string, chunkNumber int) (string, bool) {
	dir, ok := h.chunkDir(identifier)
	if !ok || chunkNumber <= 0 {
		return "", false
	}
	return filepath.Join(dir, "chunk_"+strconv.Itoa(chunkNumber)), true
}

func (h NativeHandlers) saveChunk(identifier string, chunkNumber int, data []byte) bool {
	dir, ok := h.chunkDir(identifier)
	if !ok {
		return false
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false
	}
	chunkPath, ok := h.chunkPath(identifier, chunkNumber)
	if !ok {
		return false
	}
	if err := os.WriteFile(chunkPath, data, 0644); err != nil {
		return false
	}
	metaPath := filepath.Join(dir, "meta.json")
	meta := map[string]any{"createdAt": time.Now().UnixMilli(), "chunks": map[string]any{}}
	if raw, err := os.ReadFile(metaPath); err == nil {
		_ = json.Unmarshal(raw, &meta)
	}
	chunks, _ := meta["chunks"].(map[string]any)
	if chunks == nil {
		chunks = map[string]any{}
	}
	chunks[strconv.Itoa(chunkNumber)] = map[string]any{"uploadedAt": time.Now().UnixMilli(), "size": len(data)}
	meta["chunks"] = chunks
	encoded, _ := json.Marshal(meta)
	_ = os.WriteFile(metaPath, encoded, 0644)
	return true
}

func (h NativeHandlers) verifyChunk(identifier string, chunkNumber int, expectedMD5 string) (bool, bool) {
	path, ok := h.chunkPath(identifier, chunkNumber)
	if !ok {
		return false, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false, false
	}
	if expectedMD5 != "" {
		sum := md5.Sum(data)
		if hex.EncodeToString(sum[:]) != expectedMD5 {
			_ = os.Remove(path)
			return false, false
		}
	}
	return true, true
}

func (h NativeHandlers) checkUploadComplete(identifier string, totalChunks int) ([]int, []int, bool) {
	uploaded := []int{}
	missing := []int{}
	for i := 1; i <= totalChunks; i++ {
		path, ok := h.chunkPath(identifier, i)
		if ok {
			if _, err := os.Stat(path); err == nil {
				uploaded = append(uploaded, i)
				continue
			}
		}
		missing = append(missing, i)
	}
	return uploaded, missing, len(missing) == 0
}

func (h NativeHandlers) mergeChunksToFile(identifier string, totalChunks int, filename string, uploadDir string, urlType string) (string, string, string, bool) {
	_, missing, complete := h.checkUploadComplete(identifier, totalChunks)
	if !complete {
		return "", "", msgChunkIncomplete + joinInts(missing), false
	}
	dir := absPath(uploadDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", "", err.Error(), false
	}
	unique := uniqueUploadFilename([]byte(identifier+strconv.FormatInt(time.Now().UnixMilli(), 10)), filename)
	outPath := filepath.Join(dir, unique)
	workDir, ok := h.chunkDir(identifier)
	if !ok {
		return "", "", msgChunkMergeFailed, false
	}
	mergedPath := filepath.Join(workDir, "merged.part")
	out, err := os.Create(mergedPath)
	if err != nil {
		return "", "", err.Error(), false
	}
	merged := false
	defer func() {
		if !merged {
			_ = os.Remove(mergedPath)
		}
	}()
	for i := 1; i <= totalChunks; i++ {
		path, ok := h.chunkPath(identifier, i)
		if !ok {
			_ = out.Close()
			return "", "", msgChunkMergeFailed, false
		}
		in, err := os.Open(path)
		if err != nil {
			_ = out.Close()
			return "", "", err.Error(), false
		}
		_, copyErr := io.Copy(out, in)
		_ = in.Close()
		if copyErr != nil {
			_ = out.Close()
			return "", "", copyErr.Error(), false
		}
	}
	if err := out.Close(); err != nil {
		return "", "", err.Error(), false
	}
	finalPart := outPath + ".part-" + identifier
	_ = os.Remove(finalPart)
	if err := copyFileContents(mergedPath, finalPart); err != nil {
		_ = os.Remove(finalPart)
		return "", "", err.Error(), false
	}
	if err := os.Rename(finalPart, outPath); err != nil {
		_ = os.Remove(finalPart)
		return "", "", err.Error(), false
	}
	merged = true
	h.cleanupChunks(identifier)
	return outPath, "/api/file/" + urlType + "/" + unique, "", true
}

func copyFileContents(source, destination string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func (h NativeHandlers) mergeChunksToBytes(identifier string, totalChunks int) ([]byte, string, bool) {
	_, missing, complete := h.checkUploadComplete(identifier, totalChunks)
	if !complete {
		return nil, msgChunkIncomplete + joinInts(missing), false
	}
	buf := bytes.Buffer{}
	for i := 1; i <= totalChunks; i++ {
		path, ok := h.chunkPath(identifier, i)
		if !ok {
			return nil, msgChunkMergeFailed, false
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err.Error(), false
		}
		buf.Write(data)
	}
	h.cleanupChunks(identifier)
	return buf.Bytes(), "", true
}

func (h NativeHandlers) cleanupChunks(identifier string) {
	if dir, ok := h.chunkDir(identifier); ok {
		_ = os.RemoveAll(dir)
	}
}

func bindChunkMerge(c *gin.Context) chunkMergeRequest {
	var body chunkMergeRequest
	_ = c.ShouldBindJSON(&body)
	if body.Identifier == "" {
		body.Identifier = c.PostForm("identifier")
	}
	if body.Filename == "" {
		body.Filename = c.PostForm("filename")
	}
	if body.TotalChunks == nil {
		body.TotalChunks = c.PostForm("totalChunks")
	}
	if body.Purpose == "" {
		body.Purpose = c.PostForm("purpose")
	}
	return body
}

func readMultipartFile(file *multipart.FileHeader) ([]byte, error) {
	opened, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer opened.Close()
	return io.ReadAll(opened)
}

func uniqueUploadFilename(data []byte, filename string) string {
	ext := filepath.Ext(filename)
	sum := md5.Sum(data)
	return strconv.FormatInt(time.Now().UnixMilli(), 10) + "_" + hex.EncodeToString(sum[:]) + ext
}

func invalidIdentifier(value string) bool {
	return value == "" || strings.Contains(value, "..") || strings.ContainsAny(value, `/\`)
}

func allowedAttachmentType(value string) bool {
	if strings.HasPrefix(value, "audio/") {
		return true
	}
	switch value {
	case "application/zip", "application/x-zip-compressed", "application/x-rar-compressed", "application/x-7z-compressed", "application/gzip", "application/x-tar", "application/pdf", "application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "application/vnd.ms-excel", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "application/vnd.ms-powerpoint", "application/vnd.openxmlformats-officedocument.presentationml.presentation", "text/plain":
		return true
	default:
		return false
	}
}

func allowedAttachmentFile(filename, contentType string) bool {
	if allowedAttachmentType(contentType) {
		return true
	}
	if strings.EqualFold(filepath.Ext(filename), ".mobileconfig") {
		return contentType == "" || contentType == "application/octet-stream" || contentType == "application/x-apple-aspen-config" || contentType == "text/xml" || contentType == "application/xml"
	}
	return false
}

func joinInts(values []int) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Itoa(value))
	}
	return strings.Join(parts, ", ")
}
