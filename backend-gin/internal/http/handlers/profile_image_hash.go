package handlers

import (
	"context"
	"encoding/hex"
	"io"
	"path"
	"strconv"
	"strings"

	"github.com/zeebo/xxh3"

	"yuem-go/backend-gin/internal/domain"
)

const (
	profileImageAvatarColumn     = "avatar"
	profileImageBackgroundColumn = "background"
	profileImageAvatarHashColumn = "avatar_xxh3"
	profileImageBannerHashColumn = "background_xxh3"
)

type profileImageURLParts struct {
	FileType   string
	Column     string
	HashColumn string
	UserID     int64
	Ext        string
	Hash       string
	HasHash    bool
}

func profileImageXXH3(data []byte) string {
	sum := xxh3.Hash128(data).Bytes()
	return hex.EncodeToString(sum[:])
}

func profileImageLegacyFilename(userID int64, format string) string {
	return strconv.FormatInt(userID, 10) + "." + strings.TrimPrefix(format, ".")
}

func profileImageHashedURL(fileType string, userID int64, hash string, format string) string {
	return "/api/file/" + fileType + "/" + strconv.FormatInt(userID, 10) + "-" + strings.ToLower(hash) + "." + strings.TrimPrefix(format, ".")
}

func profileImageHashColumnForStorageColumn(column string) string {
	switch column {
	case profileImageAvatarColumn:
		return profileImageAvatarHashColumn
	case profileImageBackgroundColumn:
		return profileImageBannerHashColumn
	default:
		return ""
	}
}

func profileImageStorageColumnForType(fileType string) (string, string, bool) {
	switch fileType {
	case "avatar":
		return profileImageAvatarColumn, profileImageAvatarHashColumn, true
	case "banner":
		return profileImageBackgroundColumn, profileImageBannerHashColumn, true
	default:
		return "", "", false
	}
}

func parseProfileImageCanonicalURL(canonical string) (profileImageURLParts, bool) {
	rest := strings.TrimPrefix(strings.TrimSpace(canonical), "/api/file/")
	if rest == canonical || rest == "" {
		return profileImageURLParts{}, false
	}
	segments := strings.SplitN(rest, "/", 2)
	if len(segments) != 2 || strings.Contains(segments[1], "/") {
		return profileImageURLParts{}, false
	}
	column, hashColumn, ok := profileImageStorageColumnForType(segments[0])
	if !ok {
		return profileImageURLParts{}, false
	}
	filename := path.Base(segments[1])
	ext := path.Ext(filename)
	if ext == "" {
		return profileImageURLParts{}, false
	}
	stem := strings.TrimSuffix(filename, ext)
	idText := stem
	hash := ""
	hasHash := false
	if before, after, found := strings.Cut(stem, "-"); found {
		idText = before
		hash = strings.ToLower(after)
		if !validProfileImageHash(hash) {
			return profileImageURLParts{}, false
		}
		hasHash = true
	}
	userID, err := strconv.ParseInt(idText, 10, 64)
	if err != nil || userID <= 0 {
		return profileImageURLParts{}, false
	}
	return profileImageURLParts{
		FileType:   segments[0],
		Column:     column,
		HashColumn: hashColumn,
		UserID:     userID,
		Ext:        strings.ToLower(ext),
		Hash:       hash,
		HasHash:    hasHash,
	}, true
}

func validProfileImageHash(value string) bool {
	if len(value) != 16 && len(value) != 32 {
		return false
	}
	for _, ch := range value {
		if (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') {
			continue
		}
		return false
	}
	return true
}

func (parts profileImageURLParts) legacyFilename() string {
	return strconv.FormatInt(parts.UserID, 10) + parts.Ext
}

func (parts profileImageURLParts) legacyURL() string {
	return "/api/file/" + parts.FileType + "/" + parts.legacyFilename()
}

func (parts profileImageURLParts) hashedURL(hash string) string {
	return "/api/file/" + parts.FileType + "/" + strconv.FormatInt(parts.UserID, 10) + "-" + strings.ToLower(hash) + parts.Ext
}

func profileImageDiskSubPath(fileType string, subPath string) string {
	canonical := "/api/file/" + strings.Trim(fileType, "/") + "/" + strings.TrimLeft(subPath, "/")
	parts, ok := parseProfileImageCanonicalURL(canonical)
	if !ok || !parts.HasHash {
		return subPath
	}
	return parts.legacyFilename()
}

func (h NativeHandlers) profileImageStorageValue(raw string, column string) (string, any) {
	normalized := h.normalizeFileURLForStorage(raw)
	if normalized == "" {
		return "", nil
	}
	parts, ok := parseProfileImageCanonicalURL(normalized)
	if !ok || parts.Column != column {
		return normalized, nil
	}
	if parts.HasHash {
		return parts.hashedURL(parts.Hash), parts.Hash
	}
	hash, ok := h.profileImageHashFromDisk(parts)
	if !ok {
		return normalized, nil
	}
	return parts.hashedURL(hash), hash
}

func (h NativeHandlers) ensureProfileImageHashedURL(ctx context.Context, raw string) string {
	normalized := h.normalizeFileURLForStorage(raw)
	parts, ok := parseProfileImageCanonicalURL(normalized)
	if !ok {
		return raw
	}
	if parts.HasHash {
		return parts.hashedURL(parts.Hash)
	}
	if h.DB == nil {
		return raw
	}
	hash, ok := h.profileImageHashFromDisk(parts)
	if !ok {
		return raw
	}
	hashedURL := parts.hashedURL(hash)
	updates := map[string]any{
		parts.Column:     hashedURL,
		parts.HashColumn: hash,
	}
	legacyURL := parts.legacyURL()
	result := h.DB.WithContext(ctx).Model(&domain.User{}).
		Where("id = ? AND ("+parts.Column+" = ? OR "+parts.Column+" = ?)", parts.UserID, legacyURL, strings.TrimPrefix(legacyURL, "/")).
		Updates(updates)
	if result.Error != nil || result.RowsAffected == 0 {
		return raw
	}
	return hashedURL
}

func (h NativeHandlers) profileImageHashFromDisk(parts profileImageURLParts) (string, bool) {
	file, _, _, ok := h.openUploadFile(parts.FileType, parts.legacyFilename())
	if !ok {
		return "", false
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return "", false
	}
	return profileImageXXH3(data), true
}

func (h NativeHandlers) BackfillProfileImageHashes(ctx context.Context, batchSize int) (int64, error) {
	if h.DB == nil {
		return 0, nil
	}
	if batchSize <= 0 {
		batchSize = 200
	}
	var totalChanged int64
	var lastID int64
	for {
		if err := ctx.Err(); err != nil {
			return totalChanged, err
		}
		var users []domain.User
		err := h.DB.WithContext(ctx).
			Where("id > ?", lastID).
			Where(profileImageBackfillWhereClause()).
			Order("id ASC").
			Limit(batchSize).
			Find(&users).Error
		if err != nil {
			return totalChanged, err
		}
		if len(users) == 0 {
			return totalChanged, nil
		}
		for _, user := range users {
			lastID = user.ID
			updates := map[string]any{}
			if value, ok := h.profileImageBackfillValue(stringPtrValue(user.Avatar), profileImageAvatarColumn); ok {
				updates[profileImageAvatarColumn] = value.URL
				updates[profileImageAvatarHashColumn] = value.Hash
			}
			if value, ok := h.profileImageBackfillValue(stringPtrValue(user.Background), profileImageBackgroundColumn); ok {
				updates[profileImageBackgroundColumn] = value.URL
				updates[profileImageBannerHashColumn] = value.Hash
			}
			if len(updates) == 0 {
				continue
			}
			res := h.DB.WithContext(ctx).Model(&domain.User{}).Where("id = ?", user.ID).Updates(updates)
			if res.Error != nil {
				return totalChanged, res.Error
			}
			totalChanged += res.RowsAffected
		}
		if len(users) < batchSize {
			return totalChanged, nil
		}
	}
}

func profileImageBackfillWhereClause() string {
	return `((
		avatar IS NOT NULL
		AND avatar <> ''
		AND (avatar_xxh3 IS NULL OR avatar_xxh3 = '')
		AND (avatar LIKE '/api/file/avatar/%' OR avatar LIKE 'api/file/avatar/%' OR avatar LIKE '%://%/api/file/avatar/%')
	) OR (
		background IS NOT NULL
		AND background <> ''
		AND (background_xxh3 IS NULL OR background_xxh3 = '')
		AND (background LIKE '/api/file/banner/%' OR background LIKE 'api/file/banner/%' OR background LIKE '%://%/api/file/banner/%')
	))`
}

type profileImageBackfillValue struct {
	URL  string
	Hash string
}

func (h NativeHandlers) profileImageBackfillValue(raw string, column string) (profileImageBackfillValue, bool) {
	normalized := h.normalizeFileURLForStorage(raw)
	parts, ok := parseProfileImageCanonicalURL(normalized)
	if !ok || parts.Column != column {
		return profileImageBackfillValue{}, false
	}
	if parts.HasHash {
		return profileImageBackfillValue{URL: parts.hashedURL(parts.Hash), Hash: parts.Hash}, true
	}
	hash, ok := h.profileImageHashFromDisk(parts)
	if !ok {
		return profileImageBackfillValue{}, false
	}
	return profileImageBackfillValue{URL: parts.hashedURL(hash), Hash: hash}, true
}
