package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/config"
)

func TestUploadAttachmentPreservesOriginalFilename(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uploadDir := t.TempDir()
	tempDir := t.TempDir()
	filename := "月度 报告.pdf"
	content := []byte("pdf-data")

	recorder, handler := uploadAttachmentForTest(t, uploadDir, tempDir, filename, "application/pdf", content)

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
	if responseBody.Data.OriginalName != filename {
		t.Fatalf("original name = %q, want %q", responseBody.Data.OriginalName, filename)
	}
	if responseBody.Data.URL != "/api/file/attachments/%E6%9C%88%E5%BA%A6%20%E6%8A%A5%E5%91%8A.pdf" {
		t.Fatalf("url = %q, want encoded original filename", responseBody.Data.URL)
	}
	stored, err := os.ReadFile(filepath.Join(uploadDir, filename))
	if err != nil {
		t.Fatalf("read stored attachment: %v", err)
	}
	if !bytes.Equal(stored, content) {
		t.Fatalf("stored content = %q, want %q", stored, content)
	}

	router := gin.New()
	router.GET("/api/file/*filepath", handler.FileAccess)
	fileRecorder := httptest.NewRecorder()
	fileRequest := httptest.NewRequest(http.MethodGet, responseBody.Data.URL, nil)
	router.ServeHTTP(fileRecorder, fileRequest)
	if fileRecorder.Code != http.StatusOK || !bytes.Equal(fileRecorder.Body.Bytes(), content) {
		t.Fatalf("public attachment response = (%d, %q), want (200, %q)", fileRecorder.Code, fileRecorder.Body.Bytes(), content)
	}
}

func TestUploadAttachmentPreservesMobileconfigOriginalFilename(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uploadDir := t.TempDir()
	tempDir := t.TempDir()
	filename := "公司 Wi-Fi.mobileconfig"
	content := []byte("mobileconfig-data")

	recorder, _ := uploadAttachmentForTest(t, uploadDir, tempDir, filename, "application/x-apple-aspen-config", content)

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
	if responseBody.Data.OriginalName != filename {
		t.Fatalf("original name = %q, want %q", responseBody.Data.OriginalName, filename)
	}
	if responseBody.Data.URL != "/api/file/attachments/%E5%85%AC%E5%8F%B8%20Wi-Fi.mobileconfig" {
		t.Fatalf("url = %q, want encoded original filename", responseBody.Data.URL)
	}
	if _, err := os.Stat(filepath.Join(uploadDir, filename)); err != nil {
		t.Fatalf("stored mobileconfig with original filename: %v", err)
	}
}

func TestUploadAttachmentAcceptsMobileconfigOctetStream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder, _ := uploadAttachmentForTest(t, t.TempDir(), t.TempDir(), "profile.mobileconfig", "application/octet-stream", []byte("mobileconfig-data"))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
}

func uploadAttachmentForTest(t *testing.T, uploadDir, tempDir, filename, contentType string, content []byte) (*httptest.ResponseRecorder, NativeHandlers) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/upload/attachment", &body)
	context.Request.Header.Set("Content-Type", writer.FormDataContentType())
	handler := NativeHandlers{Config: config.Config{Upload: config.UploadConfig{
		Attachment: config.UploadAttachmentConfig{LocalUploadDir: uploadDir},
		Temp:       config.UploadTempConfig{RootDir: tempDir},
	}}}

	handler.UploadAttachment(context)
	return recorder, handler
}
