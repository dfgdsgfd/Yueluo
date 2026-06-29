package handlers

import (
	"context"
	"net/url"
	"strings"
	"time"

	"yuem-go/backend-gin/internal/services"
)

const (
	fileSignatureExpiryParam = services.FileSignatureExpiryParam
	fileSignatureParam       = services.FileSignatureParam
)

func (h NativeHandlers) canonicalFileURL(fileType, subPath string) (string, bool) {
	return services.CanonicalFileURL(fileType, subPath)
}

func (h NativeHandlers) signFileURL(raw string) string {
	return h.signFileURLAt(raw, time.Now())
}

func (h NativeHandlers) signFileURLAt(raw string, now time.Time) string {
	return services.SignFileURLAt(raw, h.Config, now)
}

func (h NativeHandlers) normalizeFileURLForStorage(raw string) string {
	return services.NormalizeFileURLForStorage(raw)
}

func (h NativeHandlers) signFileURLPtr(raw *string) *string {
	if raw == nil {
		return nil
	}
	signed := h.signFileURL(h.ensureProfileImageHashedURL(context.Background(), *raw))
	return &signed
}

func (h NativeHandlers) signAnyFileURL(raw any) any {
	if raw == nil {
		return nil
	}
	text := strings.TrimSpace(toString(raw))
	if text == "" {
		return nil
	}
	text = h.ensureProfileImageHashedURL(context.Background(), text)
	return h.signFileURL(text)
}

func (h NativeHandlers) normalizeFileURLPtrForStorage(raw *string) *string {
	if raw == nil {
		return nil
	}
	normalized := h.normalizeFileURLForStorage(*raw)
	return &normalized
}

func (h NativeHandlers) verifyFileURLSignature(canonicalPath, expiryText, signature string, now time.Time) bool {
	return services.VerifyFileURLSignature(canonicalPath, expiryText, signature, h.Config, now)
}

func (h NativeHandlers) fileSignature(canonicalPath string, expiry int64) string {
	return services.FileSignature(canonicalPath, expiry, h.Config)
}

func (h NativeHandlers) fileSigningSecret() string {
	return services.FileSigningSecret(h.Config)
}

func (h NativeHandlers) fileSigningTTL() time.Duration {
	return services.FileSigningTTL(h.Config)
}

func (h NativeHandlers) localFilePathFromURL(raw string) (string, *url.URL, bool) {
	return services.LocalFilePathFromURL(raw)
}
