package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/url"
	"strconv"
	"strings"
	"time"

	"yuem-go/backend-gin/internal/config"
)

const (
	FileSignatureExpiryParam = "pvimg_exp"
	FileSignatureParam       = "sign"
	DefaultFileSigningTTL    = 15 * time.Minute
)

func CanonicalFileURL(fileType, subPath string) (string, bool) {
	fileType = strings.Trim(fileType, "/")
	subPath = strings.TrimLeft(subPath, "/")
	if fileType == "" || strings.ContainsAny(fileType, `\/`) || invalidFileSubPath(subPath) {
		return "", false
	}
	return "/api/file/" + fileType + "/" + subPath, true
}

func SignFileURL(raw string, cfg config.Config) string {
	return SignFileURLAt(raw, cfg, time.Now())
}

func SignFileURLAt(raw string, cfg config.Config, now time.Time) string {
	canonicalPath, parsed, ok := LocalFilePathFromURL(raw)
	if !ok {
		return raw
	}
	expiry := now.Add(FileSigningTTL(cfg)).Unix()
	signature := FileSignature(canonicalPath, expiry, cfg)
	if parsed != nil && parsed.IsAbs() {
		out := *parsed
		out.Path = canonicalPath
		out.RawQuery = ""
		out.Fragment = ""
		query := out.Query()
		query.Set(FileSignatureExpiryParam, strconv.FormatInt(expiry, 10))
		query.Set(FileSignatureParam, signature)
		out.RawQuery = query.Encode()
		return out.String()
	}
	return canonicalPath + "?" + FileSignatureExpiryParam + "=" + strconv.FormatInt(expiry, 10) + "&" + FileSignatureParam + "=" + url.QueryEscape(signature)
}

func NormalizeFileURLForStorage(raw string) string {
	canonicalPath, _, ok := LocalFilePathFromURL(raw)
	if ok {
		return canonicalPath
	}
	return strings.TrimSpace(raw)
}

func VerifyFileURLSignature(canonicalPath, expiryText, signature string, cfg config.Config, now time.Time) bool {
	if canonicalPath == "" || expiryText == "" || signature == "" {
		return false
	}
	expiry, err := strconv.ParseInt(expiryText, 10, 64)
	if err != nil || expiry <= 0 || now.Unix() > expiry {
		return false
	}
	expected := FileSignature(canonicalPath, expiry, cfg)
	return subtle.ConstantTimeCompare([]byte(signature), []byte(expected)) == 1
}

func FileSignature(canonicalPath string, expiry int64, cfg config.Config) string {
	secret := FileSigningSecret(cfg)
	if secret == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(canonicalPath))
	mac.Write([]byte{'\n'})
	mac.Write([]byte(strconv.FormatInt(expiry, 10)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func FileSigningSecret(cfg config.Config) string {
	if value := strings.TrimSpace(cfg.Upload.FileSigning.Secret); value != "" {
		return value
	}
	return strings.TrimSpace(cfg.Auth.JWTSecret)
}

func FileSigningTTL(cfg config.Config) time.Duration {
	if cfg.Upload.FileSigning.TTL > 0 {
		return cfg.Upload.FileSigning.TTL
	}
	return DefaultFileSigningTTL
}

func LocalFilePathFromURL(raw string) (string, *url.URL, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil, false
	}
	parsed, err := url.Parse(trimmed)
	var pathValue string
	var parsedURL *url.URL
	if err == nil && parsed.Scheme != "" && parsed.Host != "" {
		pathValue = parsed.Path
		copied := *parsed
		parsedURL = &copied
	} else if err == nil && strings.HasPrefix(parsed.Path, "/") {
		pathValue = parsed.Path
	} else {
		pathValue = trimmed
		if strings.HasPrefix(pathValue, "api/file/") {
			pathValue = "/" + pathValue
		}
	}
	if !strings.HasPrefix(pathValue, "/api/file/") {
		return "", nil, false
	}
	rest := strings.TrimPrefix(pathValue, "/api/file/")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 {
		return "", nil, false
	}
	canonicalPath, ok := CanonicalFileURL(parts[0], parts[1])
	if !ok {
		return "", nil, false
	}
	return canonicalPath, parsedURL, true
}

func invalidFileSubPath(path string) bool {
	if path == "" || strings.Contains(path, "\\") {
		return true
	}
	segments := strings.SplitSeq(path, "/")
	for segment := range segments {
		if segment == "" || segment == ".." {
			return true
		}
	}
	return false
}
