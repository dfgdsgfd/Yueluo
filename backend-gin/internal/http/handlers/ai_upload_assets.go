package handlers

import (
	"strings"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/services"
)

func (h NativeHandlers) touchAIAnalysisUploadAssets(c *gin.Context, userID int64, images []services.AIImageInput) {
	if h.UploadAssets == nil || userID <= 0 || len(images) == 0 {
		return
	}
	urls := make([]string, 0, len(images))
	for _, image := range images {
		if urlValue := h.normalizeFileURLForStorage(image.URL); strings.TrimSpace(urlValue) != "" {
			urls = append(urls, urlValue)
		}
	}
	_ = h.UploadAssets.TouchAIAnalysis(c.Request.Context(), userID, urls)
}

func (h NativeHandlers) touchAIJobUploadAssets(c *gin.Context, actor services.AIActor, req services.AIRequest) {
	if h.UploadAssets == nil || actor.ID == nil || *actor.ID <= 0 || len(req.Images) == 0 {
		return
	}
	switch strings.TrimSpace(req.Type) {
	case services.AITaskPublishTitleGenerate, services.AITaskPublishDetailGenerate:
	default:
		return
	}
	urls := make([]string, 0, len(req.Images))
	for _, image := range req.Images {
		if urlValue := h.normalizeFileURLForStorage(image.URL); strings.TrimSpace(urlValue) != "" {
			urls = append(urls, urlValue)
		}
	}
	_ = h.UploadAssets.TouchAIAnalysis(c.Request.Context(), *actor.ID, urls)
}
