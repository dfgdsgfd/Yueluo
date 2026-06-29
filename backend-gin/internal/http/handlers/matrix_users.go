package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/services"
)

func (h NativeHandlers) usersDispatch(c *gin.Context) {
	path := c.Request.URL.Path
	method := matrixMethod(c)
	if method == http.MethodGet && path == "/api/users/onboarding-config" {
		h.usersOnboardingConfig(c)
		return
	}
	user, ok := h.requireMatrixAuth(c)
	if !ok || !h.requireDB(c) {
		return
	}
	switch {
	case path == "/api/users/search" && method == http.MethodGet:
		h.usersSearch(c, user.ID)
	case path == "/api/users/history" && method == http.MethodPost:
		h.usersHistoryAdd(c, user.ID)
	case path == "/api/users/history" && method == http.MethodGet:
		h.usersHistoryList(c, user.ID)
	case path == "/api/users/history" && method == http.MethodDelete:
		h.usersHistoryClear(c, user.ID)
	case strings.HasPrefix(path, "/api/users/history/") && method == http.MethodDelete:
		h.usersHistoryDelete(c, user.ID)
	case path == "/api/users/onboarding-draft" && method == http.MethodGet:
		h.usersDraftGet(c, user.ID)
	case path == "/api/users/onboarding-draft" && method == http.MethodPut:
		h.usersDraftSave(c, user.ID)
	case path == "/api/users/onboarding" && method == http.MethodPost:
		h.usersOnboardingSubmit(c, user.ID)
	case path == "/api/users/privacy-settings" && method == http.MethodGet:
		h.usersPrivacyGet(c, user.ID)
	case path == "/api/users/privacy-settings" && method == http.MethodPut:
		h.usersPrivacyUpdate(c, user.ID)
	case path == "/api/users/toolbar/items" && method == http.MethodGet:
		h.usersToolbar(c)
	case path == "/api/users/verification/status" && method == http.MethodGet:
		h.usersVerificationStatus(c, user.ID)
	case path == "/api/users/verification" && method == http.MethodPost:
		h.usersVerificationCreate(c, user.ID)
	case path == "/api/users/verification/revoke" && method == http.MethodDelete:
		h.usersVerificationRevoke(c, user.ID)
	case path == "/api/users/api-keys" && method == http.MethodGet:
		h.usersAPIKeys(c, user.ID)
	case path == "/api/users/api-keys" && method == http.MethodPost:
		h.usersAPIKeyCreate(c, user.ID)
	case strings.HasPrefix(path, "/api/users/api-keys/") && method == http.MethodDelete:
		h.usersAPIKeyDelete(c, user.ID)
	case path == "/api/users" && method == http.MethodGet:
		h.usersList(c)
	default:
		h.usersByIDRoutes(c, user)
	}
}

func (h NativeHandlers) usersByIDRoutes(c *gin.Context, current *services.RequestUser) {
	id := matrixParam(c, "id")
	method := matrixMethod(c)
	path := c.Request.URL.Path
	switch {
	case strings.HasSuffix(path, "/personality-tags") && method == http.MethodGet:
		h.usersPersonality(c, id, current.ID)
	case method == http.MethodGet && strings.HasSuffix(path, "/follow-status"):
		h.usersFollowStatus(c, id, current.ID)
	case method == http.MethodPost && strings.HasSuffix(path, "/follow"):
		h.usersFollow(c, id, current.ID)
	case method == http.MethodDelete && strings.HasSuffix(path, "/follow"):
		h.usersUnfollow(c, id, current.ID)
	case method == http.MethodGet && strings.HasSuffix(path, "/following"):
		h.usersFollowList(c, id, true)
	case method == http.MethodGet && strings.HasSuffix(path, "/followers"):
		h.usersFollowList(c, id, false)
	case method == http.MethodGet && strings.HasSuffix(path, "/mutual-follows"):
		h.usersMutualFollows(c, id, current.ID)
	case method == http.MethodPost && strings.HasSuffix(path, "/block"):
		h.usersBlock(c, id, current.ID)
	case method == http.MethodDelete && strings.HasSuffix(path, "/block"):
		h.usersUnblock(c, id, current.ID)
	case method == http.MethodGet && strings.HasSuffix(path, "/block-status"):
		h.usersBlockStatus(c, id, current.ID)
	case method == http.MethodGet && strings.HasSuffix(path, "/posts"):
		h.usersPosts(c, id, current.ID)
	case method == http.MethodGet && strings.HasSuffix(path, "/collections"):
		h.usersCollections(c, id, current.ID)
	case method == http.MethodGet && strings.HasSuffix(path, "/likes"):
		h.usersLikes(c, id, current.ID)
	case method == http.MethodGet && strings.HasSuffix(path, "/stats"):
		h.usersStats(c, id)
	case method == http.MethodPut && strings.HasSuffix(path, "/password"):
		h.usersPassword(c, id, current.ID)
	case method == http.MethodGet:
		h.usersDetail(c, id, current.ID)
	case method == http.MethodPut:
		h.usersUpdate(c, id, current.ID)
	case method == http.MethodDelete:
		h.usersDelete(c, id, current.ID)
	default:
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "users route not found", nil)
	}
}
