package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/config"
)

func TestAdminMediaUploadPreservesOriginalFilename(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uploadDir := t.TempDir()
	tempDir := t.TempDir()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	filename := "公司 Wi-Fi.mobileconfig"
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write([]byte("custom media bytes")); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/admin/media-library", &body)
	context.Request.Header.Set("Content-Type", writer.FormDataContentType())
	handler := NativeHandlers{Config: config.Config{Upload: config.UploadConfig{
		Attachment: config.UploadAttachmentConfig{LocalUploadDir: uploadDir},
		Temp:       config.UploadTempConfig{RootDir: tempDir},
	}}}

	handler.adminMediaUpload(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var responseBody struct {
		Data struct {
			OriginalName string `json:"originalname"`
			URL          string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if responseBody.Data.OriginalName != filename ||
		responseBody.Data.URL != "/api/file/attachments/%E5%85%AC%E5%8F%B8%20Wi-Fi.mobileconfig" {
		t.Fatalf("unexpected upload response: %#v", responseBody.Data)
	}
	files, err := os.ReadDir(uploadDir)
	if err != nil {
		t.Fatalf("read upload directory: %v", err)
	}
	if len(files) != 1 || files[0].Name() != filename {
		t.Fatalf("uploaded files = %#v, want original filename %q", files, filename)
	}
}
