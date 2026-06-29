package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/localization"
	"yuem-go/backend-gin/internal/repositories"
)

const taxonomyCacheTTL = 5 * time.Minute

func (h NativeHandlers) Tags(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "error.internal", nil)
		return
	}
	loader := func() (any, error) {
		return repositories.NewTaxonomyRepository(h.DB).ListTags(c.Request.Context())
	}
	value, err := h.cached("tags:all", taxonomyCacheTTL, loader)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "服务器内部错误", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": value})
}

func (h NativeHandlers) HotTags(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "服务器内部错误", nil)
		return
	}
	limit := intQuery(c, "limit", 10)
	cacheKey := fmt.Sprintf("tags:hot:%d", limit)
	loader := func() (any, error) {
		return repositories.NewTaxonomyRepository(h.DB).HotTags(c.Request.Context(), limit)
	}
	value, err := h.cached(cacheKey, taxonomyCacheTTL, loader)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "服务器内部错误", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": value})
}

func (h NativeHandlers) Categories(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "服务器内部错误", nil)
		return
	}
	locale := localization.ResolveRequest(c.Request)
	loader := func() (any, error) {
		rows, err := repositories.NewTaxonomyRepository(h.DB).ListCategories(c.Request.Context())
		return localizedCategories(rows, locale), err
	}
	value, err := h.cached("categories:all:"+locale, taxonomyCacheTTL, loader)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "服务器内部错误", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": value})
}

func (h NativeHandlers) HotCategories(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "服务器内部错误", nil)
		return
	}
	limit := intQuery(c, "limit", 10)
	locale := localization.ResolveRequest(c.Request)
	cacheKey := fmt.Sprintf("categories:hot:%s:%d", locale, limit)
	loader := func() (any, error) {
		rows, err := repositories.NewTaxonomyRepository(h.DB).HotCategories(c.Request.Context(), limit)
		return localizedCategories(rows, locale), err
	}
	value, err := h.cached(cacheKey, taxonomyCacheTTL, loader)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "服务器内部错误", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": value})
}

type createCategoryRequest struct {
	Name          string            `json:"name"`
	CategoryTitle string            `json:"category_title"`
	Translations  map[string]string `json:"translations"`
}

func (h NativeHandlers) CreateCategory(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "error.internal", nil)
		return
	}
	var body createCategoryRequest
	_ = c.ShouldBindJSON(&body)
	name := strings.TrimSpace(body.Name)
	if name == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "category.name_required", nil)
		return
	}

	repo := repositories.NewTaxonomyRepository(h.DB)
	exists, err := repo.CategoryNameExists(c.Request.Context(), name)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "error.internal", nil)
		return
	}
	if exists {
		response.JSON(c, http.StatusConflict, response.CodeConflict, "category.name_exists", nil)
		return
	}

	var title *string
	trimmedTitle := strings.TrimSpace(body.CategoryTitle)
	if trimmedTitle != "" {
		titleExists, err := repo.CategoryTitleExists(c.Request.Context(), trimmedTitle)
		if err != nil {
			response.JSON(c, http.StatusInternalServerError, response.CodeError, "error.internal", nil)
			return
		}
		if titleExists {
			response.JSON(c, http.StatusConflict, response.CodeConflict, "category.identifier_exists", nil)
			return
		}
		title = &trimmedTitle
	}

	translations := completeCategoryTranslations(body.Translations, name, trimmedTitle)
	encodedTranslations, _ := json.Marshal(translations)
	category, err := repo.CreateCategory(c.Request.Context(), domain.Category{Name: name, CategoryTitle: title, Translations: encodedTranslations})
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "error.internal", nil)
		return
	}
	if h.Cache != nil {
		h.Cache.InvalidatePrefix("categories:")
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": localizedCategory(*category, localization.ResolveRequest(c.Request)), "message": "category.created"})
}

func localizedCategories(rows []domain.Category, locale string) []gin.H {
	out := make([]gin.H, 0, len(rows))
	for _, category := range rows {
		out = append(out, localizedCategory(category, locale))
	}
	return out
}

func localizedCategory(category domain.Category, locale string) gin.H {
	translations := map[string]string{}
	_ = json.Unmarshal(category.Translations, &translations)
	displayName := strings.TrimSpace(translations[locale])
	if displayName == "" {
		displayName = strings.TrimSpace(translations[localization.Default])
	}
	if displayName == "" && category.CategoryTitle != nil {
		displayName = strings.TrimSpace(*category.CategoryTitle)
	}
	if displayName == "" {
		displayName = category.Name
	}
	return gin.H{
		"id": category.ID, "name": category.Name, "category_title": category.CategoryTitle,
		"translations": translations, "display_name": displayName, "use_count": category.UseCount, "created_at": category.CreatedAt,
	}
}

func completeCategoryTranslations(input map[string]string, name, legacyTitle string) map[string]string {
	fallback := strings.TrimSpace(legacyTitle)
	if fallback == "" {
		fallback = strings.TrimSpace(name)
	}
	out := map[string]string{}
	for _, locale := range localization.Supported {
		value := strings.TrimSpace(input[locale])
		if value == "" {
			value = fallback
		}
		out[locale] = value
	}
	if strings.TrimSpace(input[localization.Default]) == "" {
		out[localization.Default] = strings.TrimSpace(name)
	}
	return out
}

func (h NativeHandlers) cached(key string, ttl time.Duration, loader func() (any, error)) (any, error) {
	if h.Cache == nil {
		return loader()
	}
	return h.Cache.GetOrSet(key, ttl, loader)
}
