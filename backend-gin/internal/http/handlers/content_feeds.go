package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
)

var videoCenterHTTPClient = &http.Client{Timeout: 15 * time.Second}

func (h NativeHandlers) PostsRecommended(c *gin.Context) {
	h.writePostList(c, repositories.PostListOptions{Mode: "recommended"})
}

func (h NativeHandlers) PostsHot(c *gin.Context) {
	h.writePostList(c, repositories.PostListOptions{Mode: "hot", TimeRangeDays: positiveIntQuery(c, "timeRange", 7)})
}

func (h NativeHandlers) PostsVideoCenter(c *gin.Context) {
	page := positiveIntQuery(c, "page", 1)
	limit := positiveIntQuery(c, "limit", 20)
	if h.Config.PyVideo.VideoCenterAPIKey == "" || (h.Settings != nil && !h.Settings.Bool("video_center_enabled")) {
		writeEmptyVideoCenter(c, page, limit, "\u89c6\u9891\u4e2d\u5fc3\u672a\u914d\u7f6e")
		return
	}
	if user, ok := currentUser(c); ok && h.shouldHideVideoCenterForRequest(c, user) {
		writeEmptyVideoCenter(c, page, limit, "video_center_hidden_by_registration_time")
		return
	}
	upstream := strings.TrimRight(h.Config.PyVideo.UpstreamURL, "/")
	if upstream == "" {
		upstream = "https://v.yuelk.com"
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, upstream+"/pyvideo2/api/get_posts", nil)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "\u83b7\u53d6\u89c6\u9891\u4e2d\u5fc3\u6570\u636e\u5931\u8d25", nil)
		return
	}
	q := req.URL.Query()
	q.Set("page", strconv.Itoa(page))
	q.Set("per_page", strconv.Itoa(limit))
	q.Set("search", c.DefaultQuery("search", ""))
	q.Set("order", c.DefaultQuery("order", "desc"))
	req.URL.RawQuery = q.Encode()
	req.Header.Set("accept", "application/json")
	req.Header.Set("X-API-KEY", h.Config.PyVideo.VideoCenterAPIKey)

	resp, err := videoCenterHTTPClient.Do(req)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "\u83b7\u53d6\u89c6\u9891\u4e2d\u5fc3\u6570\u636e\u5931\u8d25", nil)
		return
	}
	defer resp.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "\u83b7\u53d6\u89c6\u9891\u4e2d\u5fc3\u6570\u636e\u5931\u8d25", nil)
		return
	}
	if success, _ := boolFromAny(body["success"]); !success {
		writeEmptyVideoCenter(c, page, limit, "success")
		return
	}
	rawPosts, _ := body["posts"].([]any)
	posts := make([]gin.H, 0, len(rawPosts))
	for _, item := range rawPosts {
		raw, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if enabled, ok := boolFromAny(raw["enable"]); ok && !enabled {
			continue
		}
		posts = append(posts, videoCenterPostResponse(raw, upstream))
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    response.CodeSuccess,
		"message": "success",
		"data": gin.H{
			"posts": posts,
			"pagination": gin.H{
				"page":  intFromAnyFallback(body["current_page"], page),
				"limit": intFromAnyFallback(body["posts_per_page"], limit),
				"total": intFromAnyFallback(body["total_posts"], 0),
				"pages": intFromAnyFallback(body["total_pages"], 0),
			},
		},
	})
}

func writeEmptyVideoCenter(c *gin.Context, page int, limit int, message string) {
	c.JSON(http.StatusOK, gin.H{
		"code":    response.CodeSuccess,
		"message": message,
		"data": gin.H{
			"posts":      []gin.H{},
			"pagination": gin.H{"page": page, "limit": limit, "total": 0, "pages": 0},
		},
	})
}

func (h NativeHandlers) PostsFriends(c *gin.Context) {
	h.writePostList(c, repositories.PostListOptions{Mode: "friends", Sort: c.DefaultQuery("sort", "time")})
}

func (h NativeHandlers) PostsFollowing(c *gin.Context) {
	h.writePostList(c, repositories.PostListOptions{Mode: "following", Sort: c.DefaultQuery("sort", "time")})
}

func (h NativeHandlers) Posts(c *gin.Context) {
	opts := repositories.PostListOptions{}
	if category := c.Query("category"); category != "" {
		if parsed, err := strconv.Atoi(category); err == nil {
			opts.CategoryID = &parsed
		}
	}
	if rawUserID := c.Query("user_id"); rawUserID != "" {
		userID, err := h.resolvePostUserID(c, rawUserID)
		if err != nil {
			_ = c.Error(err)
			response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
			return
		}
		if userID != nil {
			opts.UserID = userID
		}
	}
	opts.IsDraft = c.Query("is_draft") != "" && intQuery(c, "is_draft", 0) == 1
	h.writePostList(c, opts)
}

func (h NativeHandlers) PostsProtectionConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": gin.H{
		"enabled":          h.imageProtectionEnabled(),
		"maxContentLength": h.maxPostContentLength(),
		"noticeEnabled":    h.Settings == nil || h.Settings.Bool("image_protection_notice_enabled"),
		"selectAllEnabled": h.Settings == nil || h.Settings.Bool("image_select_all_enabled"),
		"maxImages":        h.maxPostImages(),
		"archive": gin.H{
			"enabled":   h.imageArchiveEnabled(),
			"threshold": h.imageArchiveThreshold(),
		},
		"paymentMethods": gin.H{
			"balance": h.paidContentPaymentMethodEnabled("balance"),
			"points":  h.paidContentPaymentMethodEnabled("points"),
		},
		"paymentMaxPrices": gin.H{
			"balance": h.paidContentMaxPrice("balance"),
			"points":  h.paidContentMaxPrice("points"),
		},
	}})
}

func (h NativeHandlers) PostDetail(c *gin.Context) {
	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
		return
	}
	currentUserID := currentUserID(c)
	repo := repositories.NewContentRepository(h.DB)
	bundle, err := repo.PostDetail(c.Request.Context(), postID, currentUserID)
	if errors.Is(err, repositories.ErrContentForbidden) {
		post, postErr := repo.PostExists(c.Request.Context(), postID)
		if postErr == nil && post != nil {
			switch post.Visibility {
			case repositories.VisibilityPrivate:
				response.JSON(c, http.StatusForbidden, response.CodeForbidden, "\u8be5\u7b14\u8bb0\u4e3a\u79c1\u5bc6\u7b14\u8bb0", nil)
			case repositories.VisibilityFriendsOnly:
				response.JSON(c, http.StatusForbidden, response.CodeForbidden, "\u8be5\u7b14\u8bb0\u4ec5\u4e92\u5173\u597d\u53cb\u53ef\u89c1", nil)
			default:
				response.JSON(c, http.StatusForbidden, response.CodeForbidden, "\u65e0\u6743\u67e5\u770b\u8be5\u7b14\u8bb0", nil)
			}
			return
		}
	}
	if h.writeContentError(c, err, false) {
		return
	}
	if bundle.Post.Type == repositories.PostTypeVideo && currentUserID == 0 && h.Settings != nil && h.Settings.IsVideoGuestRestricted() && !bundle.Post.PublicAccessExempt {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgLoginForContent, nil)
		return
	}
	purchased, liked, collected, err := h.postInteractionSets(c, currentUserID, []int64{postID})
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
		return
	}
	body := h.postDetailResponse(*bundle, currentUserID, purchased[postID], liked[postID], collected[postID])
	if currentUserID > 0 {
		body["points_award"] = h.awardPointsBestEffort(c, currentUserID, repositories.PointsTaskClick, postID, "点击奖励")
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": body})
}

func (h NativeHandlers) PostPurchaseUsers(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || postID <= 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgPostIDMissing, nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
		return
	}
	repo := repositories.NewContentRepository(h.DB)
	post, err := repo.PostExists(c.Request.Context(), postID)
	if h.writeContentError(c, err, false) {
		return
	}
	if post.UserID != user.ID {
		response.JSON(c, http.StatusForbidden, response.CodeForbidden, msgPostEditForbidden, nil)
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := min(positiveIntQuery(c, "limit", 20), 100)
	total, purchases, err := repositories.NewBalanceRepository(h.DB).PostPurchaseUsers(c.Request.Context(), postID, page, limit)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
		return
	}
	items := make([]gin.H, 0, len(purchases))
	for _, item := range purchases {
		items = append(items, h.postPurchaseUserResponse(item))
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": gin.H{
		"list":       items,
		"pagination": paginationTotalPages(page, limit, total),
	}})
}
