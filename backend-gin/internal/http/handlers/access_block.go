package handlers

import (
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/services"
)

func (h NativeHandlers) AdminAccessBlockRules(c *gin.Context) {
	if _, ok := h.requireMatrixAdmin(c); !ok {
		return
	}
	if h.AccessBlock == nil {
		response.JSON(c, http.StatusServiceUnavailable, response.CodeError, "error.access_block_unavailable", nil)
		return
	}
	switch c.Request.Method {
	case http.MethodGet:
		h.adminAccessBlockList(c)
	case http.MethodPost:
		h.adminAccessBlockCreate(c)
	default:
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "error.route_not_found", nil)
	}
}

func (h NativeHandlers) AdminAccessBlockRule(c *gin.Context) {
	if _, ok := h.requireMatrixAdmin(c); !ok {
		return
	}
	if h.AccessBlock == nil {
		response.JSON(c, http.StatusServiceUnavailable, response.CodeError, "error.access_block_unavailable", nil)
		return
	}
	id, ok := accessBlockRuleID(c)
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.access_block_invalid_rule", nil)
		return
	}
	switch c.Request.Method {
	case http.MethodPut:
		h.adminAccessBlockUpdate(c, id)
	case http.MethodDelete:
		h.adminAccessBlockDelete(c, id)
	default:
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "error.route_not_found", nil)
	}
}

func (h NativeHandlers) AdminAccessBlockRulesBatch(c *gin.Context) {
	if _, ok := h.requireMatrixAdmin(c); !ok {
		return
	}
	if h.AccessBlock == nil {
		response.JSON(c, http.StatusServiceUnavailable, response.CodeError, "error.access_block_unavailable", nil)
		return
	}
	body := readBodyMap(c)
	items := accessBlockBatchItems(body)
	if len(items) == 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.access_block_empty_batch", nil)
		return
	}
	force, _ := boolFromAny(body["force"])
	results, err := h.AccessBlock.BatchCreate(c.Request.Context(), items, h.accessBlockCurrentInput(c), force)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, services.AccessBlockErrorKey(err), nil)
		return
	}
	created := 0
	for _, result := range results {
		if result.Rule != nil {
			created++
		}
	}
	writeSuccess(c, matrixMsgOK, gin.H{"results": results, "created": created, "failed": len(results) - created})
}

func (h NativeHandlers) AdminAccessBlockImportSources(c *gin.Context) {
	if _, ok := h.requireMatrixAdmin(c); !ok {
		return
	}
	if h.AccessBlock == nil {
		response.JSON(c, http.StatusServiceUnavailable, response.CodeError, "error.access_block_unavailable", nil)
		return
	}
	switch c.Request.Method {
	case http.MethodGet:
		sources, err := h.AccessBlock.ListImportSources(c.Request.Context())
		if err != nil {
			response.JSON(c, http.StatusInternalServerError, response.CodeError, "error.access_block_import_load_failed", nil)
			return
		}
		writeSuccess(c, matrixMsgOK, gin.H{"sources": sources})
	case http.MethodPost:
		source, err := h.AccessBlock.CreateImportSource(c.Request.Context(), accessBlockImportInputFromBody(readBodyMap(c)))
		if err != nil {
			response.JSON(c, accessBlockHTTPStatus(err), response.CodeValidationError, services.AccessBlockErrorKey(err), nil)
			return
		}
		writeSuccess(c, matrixMsgOK, gin.H{"source": source})
	default:
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "error.route_not_found", nil)
	}
}

func (h NativeHandlers) AdminAccessBlockImportSource(c *gin.Context) {
	if _, ok := h.requireMatrixAdmin(c); !ok {
		return
	}
	if h.AccessBlock == nil {
		response.JSON(c, http.StatusServiceUnavailable, response.CodeError, "error.access_block_unavailable", nil)
		return
	}
	id, ok := accessBlockRuleID(c)
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.access_block_invalid_rule", nil)
		return
	}
	switch c.Request.Method {
	case http.MethodPut:
		source, err := h.AccessBlock.UpdateImportSource(c.Request.Context(), id, accessBlockImportInputFromBody(readBodyMap(c)))
		if err != nil {
			response.JSON(c, accessBlockHTTPStatus(err), response.CodeValidationError, services.AccessBlockErrorKey(err), nil)
			return
		}
		writeSuccess(c, matrixMsgOK, gin.H{"source": source})
	case http.MethodDelete:
		if err := h.AccessBlock.DeleteImportSource(c.Request.Context(), id); err != nil {
			response.JSON(c, http.StatusInternalServerError, response.CodeError, "error.access_block_import_delete_failed", nil)
			return
		}
		writeSimpleSuccess(c, matrixMsgOK)
	default:
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "error.route_not_found", nil)
	}
}

func (h NativeHandlers) AdminAccessBlockImportSourceSync(c *gin.Context) {
	if _, ok := h.requireMatrixAdmin(c); !ok {
		return
	}
	if h.AccessBlock == nil {
		response.JSON(c, http.StatusServiceUnavailable, response.CodeError, "error.access_block_unavailable", nil)
		return
	}
	id, ok := accessBlockRuleID(c)
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.access_block_invalid_rule", nil)
		return
	}
	body := readBodyMap(c)
	force, _ := boolFromAny(body["force"])
	result, err := h.AccessBlock.SyncImportSource(c.Request.Context(), id, services.AccessBlockImportSyncOptions{
		Current: h.accessBlockCurrentInput(c),
		Force:   force,
		Manual:  true,
	})
	if err != nil {
		response.JSON(c, accessBlockHTTPStatus(err), response.CodeValidationError, services.AccessBlockErrorKey(err), gin.H{"source": result.Source})
		return
	}
	writeSuccess(c, matrixMsgOK, gin.H{"source": result.Source, "count": result.Count})
}

func (h NativeHandlers) adminAccessBlockList(c *gin.Context) {
	rules, err := h.AccessBlock.List(c.Request.Context())
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "error.access_block_load_failed", nil)
		return
	}
	writeSuccess(c, matrixMsgOK, gin.H{
		"rules":    rules,
		"disabled": h.AccessBlock.Disabled(),
	})
}

func (h NativeHandlers) adminAccessBlockCreate(c *gin.Context) {
	rule, err := h.AccessBlock.Create(c.Request.Context(), accessBlockInputFromBody(readBodyMap(c)), h.accessBlockCurrentInput(c))
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrInvalidDB) {
			status = http.StatusInternalServerError
		}
		response.JSON(c, status, response.CodeValidationError, services.AccessBlockErrorKey(err), nil)
		return
	}
	writeSuccess(c, matrixMsgOK, gin.H{"rule": rule})
}

func (h NativeHandlers) adminAccessBlockUpdate(c *gin.Context, id int64) {
	rule, err := h.AccessBlock.Update(c.Request.Context(), id, accessBlockInputFromBody(readBodyMap(c)), h.accessBlockCurrentInput(c))
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, gorm.ErrInvalidDB) {
			status = http.StatusInternalServerError
		}
		response.JSON(c, status, response.CodeValidationError, services.AccessBlockErrorKey(err), nil)
		return
	}
	writeSuccess(c, matrixMsgOK, gin.H{"rule": rule})
}

func (h NativeHandlers) adminAccessBlockDelete(c *gin.Context, id int64) {
	if err := h.AccessBlock.Delete(c.Request.Context(), id); err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "error.access_block_delete_failed", nil)
		return
	}
	writeSimpleSuccess(c, matrixMsgOK)
}

func (h NativeHandlers) accessBlockCurrentInput(c *gin.Context) services.AccessBlockMatchInput {
	return services.AccessBlockMatchInput{
		IP:        accessBlockClientIP(c, h.Config.Server.ClientIPHeaders),
		UserAgent: c.Request.UserAgent(),
	}
}

func accessBlockRuleID(c *gin.Context) (int64, bool) {
	value := strings.TrimSpace(c.Param("id"))
	if value == "" {
		value = matrixParam(c, "id")
	}
	id, err := strconv.ParseInt(value, 10, 64)
	return id, err == nil && id > 0
}

func accessBlockInputFromBody(body map[string]any) services.AccessBlockRuleInput {
	input := services.AccessBlockRuleInput{
		Kind:        toString(firstPresent(body, "kind", "type")),
		MatchType:   toString(firstPresent(body, "match_type", "matchType")),
		Pattern:     toString(body["pattern"]),
		Action:      toString(body["action"]),
		RedirectURL: toString(firstPresent(body, "redirect_url", "redirectUrl")),
		Note:        toString(body["note"]),
	}
	if value, ok := boolFromAny(body["enabled"]); ok {
		input.Enabled = &value
	}
	if value, ok := intFromAny(body["priority"]); ok {
		input.Priority = &value
	}
	if value, ok := intFromAny(firstPresent(body, "status_code", "statusCode")); ok {
		input.StatusCode = value
	}
	input.Force, _ = boolFromAny(body["force"])
	return input
}

func accessBlockImportInputFromBody(body map[string]any) services.AccessBlockImportSourceInput {
	input := services.AccessBlockImportSourceInput{
		URL:         toString(body["url"]),
		Action:      toString(body["action"]),
		RedirectURL: toString(firstPresent(body, "redirect_url", "redirectUrl")),
		Note:        toString(body["note"]),
	}
	if value, ok := boolFromAny(body["enabled"]); ok {
		input.Enabled = &value
	}
	if value, ok := intFromAny(body["priority"]); ok {
		input.Priority = &value
	}
	if value, ok := intFromAny(firstPresent(body, "status_code", "statusCode")); ok {
		input.StatusCode = value
	}
	if value, ok := intFromAny(firstPresent(body, "update_interval_seconds", "updateIntervalSeconds")); ok {
		input.UpdateIntervalSeconds = value
	}
	return input
}

func accessBlockHTTPStatus(err error) int {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return http.StatusNotFound
	case errors.Is(err, gorm.ErrInvalidDB):
		return http.StatusInternalServerError
	default:
		return http.StatusBadRequest
	}
}

func accessBlockBatchItems(body map[string]any) []services.AccessBlockBatchItem {
	items := []services.AccessBlockBatchItem{}
	if rawRules, ok := body["rules"].([]any); ok {
		for idx, raw := range rawRules {
			if rule, ok := raw.(map[string]any); ok {
				items = append(items, services.AccessBlockBatchItem{Input: accessBlockInputFromBody(rule), Line: idx + 1})
			}
		}
		return items
	}
	base := accessBlockInputFromBody(body)
	rawPatterns := strings.TrimSpace(toString(firstPresent(body, "patterns", "batch", "text")))
	if rawPatterns == "" && base.Pattern != "" {
		rawPatterns = base.Pattern
	}
	line := 0
	for rawLine := range strings.SplitSeq(rawPatterns, "\n") {
		line++
		pattern := strings.TrimSpace(rawLine)
		if pattern == "" || strings.HasPrefix(pattern, "#") {
			continue
		}
		next := base
		next.Pattern = pattern
		items = append(items, services.AccessBlockBatchItem{Input: next, Line: line})
	}
	return items
}

func accessBlockClientIP(c *gin.Context, headers []string) string {
	if c == nil || c.Request == nil {
		return ""
	}
	for _, header := range accessBlockClientIPHeaders(headers) {
		if ip := accessBlockIPFromHeaderValue(header, c.GetHeader(header)); ip != "" {
			return ip
		}
	}
	if host, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
		if ip := accessBlockNormalizeIP(host); ip != "" {
			return ip
		}
	}
	return c.ClientIP()
}

func accessBlockClientIPHeaders(headers []string) []string {
	if len(headers) == 0 {
		return []string{"X-Forwarded-For", "X-Real-IP", "CF-Connecting-IP"}
	}
	out := make([]string, 0, len(headers))
	for _, header := range headers {
		if text := strings.TrimSpace(header); text != "" {
			out = append(out, text)
		}
	}
	if len(out) == 0 {
		return []string{"X-Forwarded-For", "X-Real-IP", "CF-Connecting-IP"}
	}
	return out
}

func accessBlockIPFromHeaderValue(header string, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.EqualFold(header, "Forwarded") {
		for group := range strings.SplitSeq(value, ",") {
			for part := range strings.SplitSeq(group, ";") {
				key, raw, ok := strings.Cut(strings.TrimSpace(part), "=")
				if ok && strings.EqualFold(strings.TrimSpace(key), "for") {
					if ip := accessBlockNormalizeIP(raw); ip != "" {
						return ip
					}
				}
			}
		}
		return ""
	}
	for part := range strings.SplitSeq(value, ",") {
		if ip := accessBlockNormalizeIP(part); ip != "" {
			return ip
		}
	}
	return ""
}

func accessBlockNormalizeIP(value string) string {
	value = strings.TrimSpace(strings.Trim(value, `"`))
	if value == "" || strings.EqualFold(value, "unknown") {
		return ""
	}
	if ip := net.ParseIP(value); ip != nil {
		return ip.String()
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		host = strings.Trim(host, "[]")
		if ip := net.ParseIP(host); ip != nil {
			return ip.String()
		}
	}
	value = strings.Trim(value, "[]")
	if ip := net.ParseIP(value); ip != nil {
		return ip.String()
	}
	return ""
}
