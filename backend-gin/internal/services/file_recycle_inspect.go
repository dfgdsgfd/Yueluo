package services

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

var (
	ErrFileRecycleItemNotFound   = errors.New("admin.fileRecycleBin.notFound")
	ErrFileRecycleUnsafePath     = errors.New("admin.fileRecycleBin.unsafePath")
	ErrFileRecycleMissingPath    = errors.New("admin.fileRecycleBin.fileMissing")
	ErrFileRecycleDirectory      = errors.New("admin.fileRecycleBin.directoryPreviewUnsupported")
	ErrFileRecyclePreviewUnknown = errors.New("admin.fileRecycleBin.previewUnsupported")
)

type FileRecyclePathState struct {
	Configured bool   `json:"configured"`
	Exists     bool   `json:"exists"`
	IsDir      bool   `json:"is_dir"`
	SizeBytes  int64  `json:"size_bytes"`
	FileCount  int64  `json:"file_count"`
	Unsafe     bool   `json:"unsafe"`
	Path       string `json:"path,omitempty"`
}

type FileRecycleDirEntry struct {
	Name      string `json:"name"`
	IsDir     bool   `json:"is_dir"`
	SizeBytes int64  `json:"size_bytes"`
}

type FileRecycleInspection struct {
	Item         domain.FileRecycleItem `json:"item"`
	Original     FileRecyclePathState   `json:"original"`
	Recycled     FileRecyclePathState   `json:"recycled"`
	Files        []FileRecycleDirEntry  `json:"files,omitempty"`
	Previewable  bool                   `json:"previewable"`
	Downloadable bool                   `json:"downloadable"`
	MIMEType     string                 `json:"mime_type,omitempty"`
	Filename     string                 `json:"filename,omitempty"`
	PreviewKind  string                 `json:"preview_kind,omitempty"`
}

type FileRecycleOpenResult struct {
	Inspection  *FileRecycleInspection
	File        *os.File
	Stat        os.FileInfo
	MIMEType    string
	Filename    string
	PreviewKind string
}

func (s *FileRecycleService) InspectItem(ctx context.Context, id int64) (*FileRecycleInspection, error) {
	if s == nil || s.db == nil || id <= 0 {
		return nil, ErrFileRecycleItemNotFound
	}
	var item domain.FileRecycleItem
	if err := s.db.WithContext(ctx).Where("id = ?", id).Take(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFileRecycleItemNotFound
		}
		return nil, err
	}
	inspection := &FileRecycleInspection{
		Item:     item,
		Original: s.inspectOriginalPath(item.OriginalPath, item.IsDir),
		Recycled: s.inspectRecycledPath(item.RecycledPath),
		Filename: fileRecycleDisplayName(item),
	}
	if inspection.Recycled.Exists && !inspection.Recycled.IsDir && !inspection.Recycled.Unsafe {
		inspection.MIMEType, inspection.PreviewKind = detectRecycleFileType(inspection.Recycled.Path)
		inspection.Downloadable = true
		inspection.Previewable = inspection.PreviewKind != "" && inspection.PreviewKind != "other"
	}
	if inspection.Recycled.Exists && inspection.Recycled.IsDir && !inspection.Recycled.Unsafe {
		inspection.Files = recycledDirEntries(inspection.Recycled.Path, 200)
	}
	return inspection, nil
}

func (s *FileRecycleService) OpenRecycledFile(ctx context.Context, id int64) (*FileRecycleOpenResult, error) {
	inspection, err := s.InspectItem(ctx, id)
	if err != nil {
		return nil, err
	}
	if inspection.Recycled.Unsafe {
		return nil, ErrFileRecycleUnsafePath
	}
	if !inspection.Recycled.Exists {
		return nil, ErrFileRecycleMissingPath
	}
	if inspection.Recycled.IsDir {
		return nil, ErrFileRecycleDirectory
	}
	file, err := os.Open(inspection.Recycled.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrFileRecycleMissingPath
		}
		return nil, err
	}
	stat, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	mimeType, previewKind := detectRecycleFileType(inspection.Recycled.Path)
	if _, err := file.Seek(0, 0); err != nil {
		_ = file.Close()
		return nil, err
	}
	return &FileRecycleOpenResult{
		Inspection:  inspection,
		File:        file,
		Stat:        stat,
		MIMEType:    mimeType,
		Filename:    inspection.Filename,
		PreviewKind: previewKind,
	}, nil
}

func (s *FileRecycleService) inspectOriginalPath(path string, isDir bool) FileRecyclePathState {
	state := FileRecyclePathState{Configured: strings.TrimSpace(path) != ""}
	if !state.Configured {
		return state
	}
	safe, _ := s.safeRecycleSourcePath(path, isDir)
	if !safe {
		state.Unsafe = true
		state.Path = filepath.Clean(serviceAbsPath(path))
		return state
	}
	return inspectExistingRecyclePath(filepath.Clean(serviceAbsPath(path)))
}

func (s *FileRecycleService) inspectRecycledPath(path string) FileRecyclePathState {
	state := FileRecyclePathState{Configured: strings.TrimSpace(path) != ""}
	if !state.Configured {
		return state
	}
	safePath, err := s.safeRecycledPath(path)
	if err != nil {
		state.Unsafe = true
		state.Path = filepath.Clean(serviceAbsPath(path))
		return state
	}
	return inspectExistingRecyclePath(safePath)
}

func (s *FileRecycleService) safeRecycledPath(path string) (string, error) {
	path = filepath.Clean(serviceAbsPath(path))
	if path == "" {
		return "", ErrFileRecycleMissingPath
	}
	root := s.RecycleRoot()
	if root == "" || path == root || !withinRoot(path, root) {
		return "", ErrFileRecycleUnsafePath
	}
	if pathHasSymlinkComponent(root, path) {
		return "", ErrFileRecycleUnsafePath
	}
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return path, nil
		}
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", ErrFileRecycleUnsafePath
	}
	return path, nil
}

func pathHasSymlinkComponent(root string, path string) bool {
	root = filepath.Clean(serviceAbsPath(root))
	path = filepath.Clean(serviceAbsPath(path))
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return !errors.Is(err, os.ErrNotExist)
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 {
		return true
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return err != nil
	}
	current := root
	for part := range strings.SplitSeq(rel, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return !errors.Is(err, os.ErrNotExist)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return true
		}
	}
	return false
}

func (s *FileRecycleService) removeRecycledPath(path string, isDir bool) error {
	path, err := s.safeRecycledPath(path)
	if err != nil {
		return err
	}
	if path == "" {
		return nil
	}
	if isDir {
		return os.RemoveAll(path)
	}
	return os.Remove(path)
}

func inspectExistingRecyclePath(path string) FileRecyclePathState {
	state := FileRecyclePathState{Configured: true, Path: path}
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return state
		}
		state.Unsafe = true
		return state
	}
	if info.Mode()&os.ModeSymlink != 0 {
		state.Unsafe = true
		return state
	}
	count, size, err := pathStats(path)
	if err != nil {
		state.Unsafe = true
		return state
	}
	state.Exists = true
	state.IsDir = info.IsDir()
	state.FileCount = count
	state.SizeBytes = size
	return state
}

func recycledDirEntries(path string, limit int) []FileRecycleDirEntry {
	if limit <= 0 {
		limit = 200
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}
	out := make([]FileRecycleDirEntry, 0, minIntFileRecycle(len(entries), limit))
	for _, entry := range entries {
		if len(out) >= limit {
			break
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		out = append(out, FileRecycleDirEntry{Name: entry.Name(), IsDir: entry.IsDir(), SizeBytes: info.Size()})
	}
	return out
}

func detectRecycleFileType(path string) (string, string) {
	mimeType := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
	if mimeType == "" {
		mimeType = sniffRecycleFileType(path)
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	previewKind := "other"
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		previewKind = "image"
	case strings.HasPrefix(mimeType, "video/"):
		previewKind = "video"
	case strings.HasPrefix(mimeType, "audio/"):
		previewKind = "audio"
	case strings.HasPrefix(mimeType, "text/"):
		previewKind = "text"
	case strings.Contains(mimeType, "json") || strings.Contains(mimeType, "xml"):
		previewKind = "text"
	case mimeType == "application/pdf":
		previewKind = "pdf"
	}
	return mimeType, previewKind
}

func sniffRecycleFileType(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && n == 0 {
		return ""
	}
	return http.DetectContentType(buffer[:n])
}

func fileRecycleDisplayName(item domain.FileRecycleItem) string {
	for _, value := range []string{item.OriginalPath, item.OriginalURL, item.RecycledPath} {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		value = strings.TrimRight(strings.ReplaceAll(value, "\\", "/"), "/")
		if name := pathBase(value); name != "" && name != "." && name != "/" {
			return name
		}
	}
	return fmt.Sprintf("recycled-file-%d", item.ID)
}

func pathBase(value string) string {
	parts := strings.Split(value, "/")
	return parts[len(parts)-1]
}

func minIntFileRecycle(a, b int) int {
	if a < b {
		return a
	}
	return b
}
