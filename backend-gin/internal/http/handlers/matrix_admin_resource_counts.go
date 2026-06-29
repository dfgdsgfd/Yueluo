package handlers

import (
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h NativeHandlers) adminResourceCountQuery(c *gin.Context, resource adminResource) *gorm.DB {
	countResource := resource
	if canUseBaseTableAdminCount(c, resource) {
		countResource.Joins = nil
	}
	query := h.adminResourceQuery(c, countResource)
	if keyword := firstNonEmpty(c.Query("keyword"), c.Query("search")); keyword != "" && len(resource.SearchFields) > 0 {
		query = applySearch(query, resource.SearchFields, keyword)
	}
	return applyAdminFilters(c, query, resource)
}

func canUseBaseTableAdminCount(c *gin.Context, resource adminResource) bool {
	if len(resource.Joins) == 0 || firstNonEmpty(c.Query("keyword"), c.Query("search")) != "" {
		return false
	}
	for key, values := range c.Request.URL.Query() {
		if len(values) == 0 || values[0] == "" {
			continue
		}
		switch key {
		case "page", "limit", "pageSize", "keyword", "search", "sortField", "sortOrder":
			continue
		}
		filter, ok := resource.Filters[key]
		if !ok {
			continue
		}
		if adminFilterRequiresJoin(resource, filter) {
			return false
		}
	}
	return true
}

func adminFilterRequiresJoin(resource adminResource, filter adminFilter) bool {
	column := strings.TrimSpace(filter.Column)
	if column == "" || resource.Alias == "" {
		return false
	}
	if strings.HasPrefix(column, resource.Alias+".") {
		return false
	}
	if !strings.Contains(column, ".") {
		return false
	}
	return true
}
