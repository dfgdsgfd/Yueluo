package handlers

import (
	"maps"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/http/response"
)

func (h NativeHandlers) adminGenericResource(c *gin.Context, resource adminResource, id string) {
	method := matrixMethod(c)
	if id == "" {
		id = matrixParam(c, "id")
	}
	switch method {
	case http.MethodGet:
		if id != "" {
			h.adminGenericDetail(c, resource, id)
			return
		}
		h.adminGenericList(c, resource)
	case http.MethodPost:
		h.adminGenericCreate(c, resource)
	case http.MethodPut, http.MethodPatch:
		if id == "" {
			h.adminGenericBulkUpdate(c, resource)
			return
		}
		h.adminGenericUpdate(c, resource, id)
	case http.MethodDelete:
		if id == "" {
			h.adminGenericBulkDelete(c, resource)
			return
		}
		h.adminGenericDelete(c, resource, id)
	default:
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "admin route not found", nil)
	}
}

func (h NativeHandlers) adminGenericList(c *gin.Context, resource adminResource) {
	page, limit, offset := adminPageLimit(c, resource)
	query := h.adminResourceQuery(c, resource)
	if keyword := firstNonEmpty(c.Query("keyword"), c.Query("search")); keyword != "" && len(resource.SearchFields) > 0 {
		query = applySearch(query, resource.SearchFields, keyword)
	}
	query = applyAdminFilters(c, query, resource)
	var total int64
	if err := h.adminResourceCountQuery(c, resource).Count(&total).Error; writeDBError(c, err, "") {
		return
	}
	var rows []map[string]any
	if err := query.Select(adminSelect(resource)).Order(adminOrder(c, resource)).Offset(offset).Limit(limit).Find(&rows).Error; writeDBError(c, err, "") {
		return
	}
	normalized := h.normalizeRows(rows, resource)
	writeAdminList(c, resource, normalized, page, limit, total)
}

func (h NativeHandlers) adminGenericDetail(c *gin.Context, resource adminResource, id string) {
	var row map[string]any
	err := h.adminResourceQuery(c, resource).Select(adminSelect(resource)).Where(adminIDColumn(resource)+" = ?", id).Take(&row).Error
	if writeDBError(c, err, "资源不存在") {
		return
	}
	writeSuccess(c, matrixMsgOK, h.normalizeRow(row, resource))
}

func (h NativeHandlers) adminGenericCreate(c *gin.Context, resource adminResource) {
	body := h.adminSanitizeBody(resource, readBodyMap(c))
	extra, err := adminPreprocessCreate(resource, body)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	if len(body) == 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "没有可创建的数据", nil)
		return
	}
	if err := h.DB.WithContext(c.Request.Context()).Table(resource.Table).Create(body).Error; writeDBError(c, err, "") {
		return
	}
	h.bumpAdminResourceCacheVersions(resource.Table)
	payload := h.normalizeRow(body, resource)
	maps.Copy(payload, extra)
	writeSuccess(c, "创建成功", payload)
}

func (h NativeHandlers) adminGenericUpdate(c *gin.Context, resource adminResource, id string) {
	body := h.adminSanitizeBody(resource, readBodyMap(c))
	if len(body) == 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "没有需要更新的数据", nil)
		return
	}
	if err := adminPreprocessUpdate(resource, body); err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	res := h.DB.WithContext(c.Request.Context()).Table(resource.Table).Where("id = ?", id).Updates(body)
	if writeDBError(c, res.Error, "") {
		return
	}
	if res.RowsAffected == 0 {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "资源不存在", nil)
		return
	}
	if resource.Table == "open_apis" {
		if numericID, ok := int64FromAny(id); ok {
			h.invalidateOpenAPIKeyIDs(numericID)
		}
	}
	h.bumpAdminResourceCacheVersions(resource.Table)
	writeSimpleSuccess(c, "更新成功")
}

func (h NativeHandlers) adminGenericDelete(c *gin.Context, resource adminResource, id string) {
	res := h.DB.WithContext(c.Request.Context()).Table(resource.Table).Where("id = ?", id).Delete(nil)
	if writeDBError(c, res.Error, "") {
		return
	}
	if res.RowsAffected == 0 {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "资源不存在", nil)
		return
	}
	if resource.Table == "open_apis" {
		if numericID, ok := int64FromAny(id); ok {
			h.invalidateOpenAPIKeyIDs(numericID)
		}
	}
	h.bumpAdminResourceCacheVersions(resource.Table)
	writeSimpleSuccess(c, "删除成功")
}

func (h NativeHandlers) adminGenericBulkDelete(c *gin.Context, resource adminResource) {
	ids := int64SliceFromAny(readBodyMap(c)["ids"])
	if len(ids) == 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "请提供要删除的ID列表", nil)
		return
	}
	res := h.DB.WithContext(c.Request.Context()).Table(resource.Table).Where("id IN ?", ids).Delete(nil)
	if writeDBError(c, res.Error, "") {
		return
	}
	if resource.Table == "open_apis" {
		h.invalidateOpenAPIKeyIDs(ids...)
	}
	h.bumpAdminResourceCacheVersions(resource.Table)
	writeSimpleSuccess(c, "成功删除 "+strconv.FormatInt(res.RowsAffected, 10)+" 条记录")
}

func (h NativeHandlers) adminGenericBulkUpdate(c *gin.Context, resource adminResource) {
	body := readBodyMap(c)
	ids := int64SliceFromAny(body["ids"])
	if len(ids) == 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "请提供要更新的ID列表", nil)
		return
	}
	delete(body, "ids")
	updates := h.adminSanitizeBody(resource, body)
	if len(updates) == 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "没有需要更新的数据", nil)
		return
	}
	if err := adminPreprocessUpdate(resource, updates); err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	res := h.DB.WithContext(c.Request.Context()).Table(resource.Table).Where("id IN ?", ids).Updates(updates)
	if writeDBError(c, res.Error, "") {
		return
	}
	if resource.Table == "open_apis" {
		h.invalidateOpenAPIKeyIDs(ids...)
	}
	h.bumpAdminResourceCacheVersions(resource.Table)
	writeSimpleSuccess(c, "成功更新 "+strconv.FormatInt(res.RowsAffected, 10)+" 条记录")
}
