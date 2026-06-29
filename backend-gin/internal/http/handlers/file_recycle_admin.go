package handlers

import (
	"errors"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/services"
)

func (h NativeHandlers) AdminFileRecycleBin(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	service := h.FileRecycle
	if service == nil {
		service = services.NewFileRecycleServiceWithSettings(h.DB, h.Config, nil, h.Settings)
	}
	method := matrixMethod(c)
	path := c.Request.URL.Path
	switch {
	case method == http.MethodGet && path == "/api/admin/file-recycle-bin":
		h.adminGenericList(c, adminResources["file-recycle-bin"])
	case method == http.MethodGet && strings.HasSuffix(path, "/inspect"):
		id, ok := fileRecycleID(c)
		if !ok {
			return
		}
		inspection, err := service.InspectItem(c.Request.Context(), id)
		if writeFileRecycleError(c, err) {
			return
		}
		writeSuccess(c, "admin.fileRecycleBin.inspectDone", inspection)
	case method == http.MethodGet && strings.HasSuffix(path, "/preview"):
		id, ok := fileRecycleID(c)
		if !ok {
			return
		}
		file, err := service.OpenRecycledFile(c.Request.Context(), id)
		if writeFileRecycleError(c, err) {
			return
		}
		if file.PreviewKind == "" || file.PreviewKind == "other" {
			_ = file.File.Close()
			response.JSON(c, http.StatusUnsupportedMediaType, response.CodeValidationError, "admin.fileRecycleBin.previewUnsupported", file.Inspection)
			return
		}
		serveFileRecycleFile(c, file, "inline")
	case method == http.MethodGet && strings.HasSuffix(path, "/download"):
		id, ok := fileRecycleID(c)
		if !ok {
			return
		}
		file, err := service.OpenRecycledFile(c.Request.Context(), id)
		if writeFileRecycleError(c, err) {
			return
		}
		serveFileRecycleFile(c, file, "attachment")
	case method == http.MethodDelete && path == "/api/admin/file-recycle-bin":
		ids := int64SliceFromAny(readBodyMap(c)["ids"])
		if len(ids) == 0 {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "admin.fileRecycleBin.idsRequired", nil)
			return
		}
		summary, err := service.PurgeIDs(c.Request.Context(), ids)
		if writeDBError(c, err, "") {
			return
		}
		h.bumpAdminResourceCacheVersions("file_recycle_items")
		writeSuccess(c, "admin.fileRecycleBin.purgeDone", gin.H{"summary": summary})
	case method == http.MethodDelete:
		idText := matrixParam(c, "id")
		if idText == "" {
			segments := adminSegments(c)
			if len(segments) >= 2 {
				idText = segments[1]
			}
		}
		id, err := strconv.ParseInt(idText, 10, 64)
		if err != nil || id <= 0 {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "admin.fileRecycleBin.invalidID", nil)
			return
		}
		summary, err := service.PurgeIDs(c.Request.Context(), []int64{id})
		if writeDBError(c, err, "") {
			return
		}
		h.bumpAdminResourceCacheVersions("file_recycle_items")
		writeSuccess(c, "admin.fileRecycleBin.purgeDone", gin.H{"summary": summary})
	case method == http.MethodPost && path == "/api/admin/file-recycle-bin/run-cleanup":
		result, err := service.RunCleanup(c.Request.Context(), 500, 100)
		if writeDBError(c, err, "") {
			return
		}
		h.bumpAdminResourceCacheVersions("file_recycle_items")
		writeSuccess(c, "admin.fileRecycleBin.cleanupDone", gin.H{
			"expired":     result.Expired,
			"orphan_dash": result.OrphanDASH,
		})
	default:
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "admin route not found", nil)
	}
}

func fileRecycleID(c *gin.Context) (int64, bool) {
	idText := matrixParam(c, "id")
	if idText == "" {
		segments := adminSegments(c)
		if len(segments) >= 2 {
			idText = segments[1]
		}
	}
	id, err := strconv.ParseInt(idText, 10, 64)
	if err != nil || id <= 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "admin.fileRecycleBin.invalidID", nil)
		return 0, false
	}
	return id, true
}

func writeFileRecycleError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, services.ErrFileRecycleItemNotFound):
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "admin.fileRecycleBin.notFound", nil)
	case errors.Is(err, services.ErrFileRecycleUnsafePath):
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "admin.fileRecycleBin.unsafePath", nil)
	case errors.Is(err, services.ErrFileRecycleMissingPath):
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "admin.fileRecycleBin.fileMissing", nil)
	case errors.Is(err, services.ErrFileRecycleDirectory):
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "admin.fileRecycleBin.directoryPreviewUnsupported", nil)
	case errors.Is(err, services.ErrFileRecyclePreviewUnknown):
		response.JSON(c, http.StatusUnsupportedMediaType, response.CodeValidationError, "admin.fileRecycleBin.previewUnsupported", nil)
	default:
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "admin.fileRecycleBin.inspectFailed", nil)
	}
	return true
}

func serveFileRecycleFile(c *gin.Context, file *services.FileRecycleOpenResult, disposition string) {
	defer file.File.Close()
	if file.MIMEType != "" {
		c.Header("Content-Type", file.MIMEType)
	}
	c.Header("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": file.Filename}))
	http.ServeContent(c.Writer, c.Request, file.Filename, file.Stat.ModTime(), file.File)
}
