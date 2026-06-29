package handlers

import (
	"context"
	"maps"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
)

const msgSearchInternal = "\u670d\u52a1\u5668\u5185\u90e8\u9519\u8bef"

type searchLoadOptions struct {
	Keyword            string
	Tag                string
	SearchType         string
	Page               int
	Limit              int
	CurrentUserID      int64
	PublicAccessOnly   bool
	ExcludeVideoGuests bool
}

func (h NativeHandlers) Search(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgSearchInternal, nil)
		return
	}

	keyword := c.DefaultQuery("keyword", "")
	tag := c.DefaultQuery("tag", "")
	searchType := c.DefaultQuery("type", "all")
	page := positiveIntQuery(c, "page", 1)
	limit := positiveIntQuery(c, "limit", 20)
	currentUserID := int64(0)
	if user, ok := currentUser(c); ok {
		currentUserID = user.ID
	}
	publicAccessOnly := publicAccessExemptOnly(c)
	excludeVideoGuests := currentUserID == 0 && h.Settings != nil && h.Settings.IsVideoGuestRestricted()

	if strings.TrimSpace(keyword) == "" && strings.TrimSpace(tag) == "" {
		c.JSON(http.StatusOK, gin.H{
			"code":    response.CodeSuccess,
			"message": "success",
			"data": gin.H{
				"keyword":  keyword,
				"tag":      tag,
				"type":     searchType,
				"data":     []gin.H{},
				"tagStats": []gin.H{},
				"pagination": gin.H{
					"page":  page,
					"limit": limit,
					"total": 0,
					"pages": 0,
				},
			},
		})
		return
	}

	opts := searchLoadOptions{
		Keyword:            keyword,
		Tag:                tag,
		SearchType:         searchType,
		Page:               page,
		Limit:              limit,
		CurrentUserID:      currentUserID,
		PublicAccessOnly:   publicAccessOnly,
		ExcludeVideoGuests: excludeVideoGuests,
	}
	repo := repositories.NewSearchRepository(h.DB)
	if currentUserID != 0 && strings.TrimSpace(keyword) != "" {
		recordKeyword := keyword
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = repo.RecordSearchHistory(ctx, currentUserID, recordKeyword)
		}()
	}

	cacheKey := ""
	if h.Redis != nil {
		cacheKey = h.cacheKeyWithVersions(cacheScopeSearch, []string{cacheScopePosts, cacheScopeUsers, cacheScopeInteractions, cacheScopeSettings}, currentUserID, keyword, tag, searchType, page, limit, publicAccessOnly, excludeVideoGuests)
		var cached gin.H
		if h.Redis.CacheGetJSON(c.Request.Context(), cacheKey, &cached) {
			c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": cached})
			return
		}
	}

	if h.Redis != nil && cacheKey != "" && h.SearchGroup != nil {
		value, err, _ := h.SearchGroup.Do(cacheKey, func() (any, error) {
			var cached gin.H
			if h.Redis.CacheGetJSON(c.Request.Context(), cacheKey, &cached) {
				return cached, nil
			}
			data, err := h.loadSearchData(c.Request.Context(), opts)
			if err == nil {
				_ = h.Redis.CacheSet(c.Request.Context(), cacheKey, data, cacheTTL(30))
			}
			return data, err
		})
		if err != nil {
			response.JSON(c, http.StatusInternalServerError, response.CodeError, msgSearchInternal, nil)
			return
		}
		if data, ok := value.(gin.H); ok {
			c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": data})
			return
		}
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgSearchInternal, nil)
		return
	}

	data, err := h.loadSearchData(c.Request.Context(), opts)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgSearchInternal, nil)
		return
	}
	if h.Redis != nil && cacheKey != "" {
		_ = h.Redis.CacheSet(c.Request.Context(), cacheKey, data, cacheTTL(30))
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": data})
}

func (h NativeHandlers) loadSearchData(ctx context.Context, opts searchLoadOptions) (gin.H, error) {
	repo := repositories.NewSearchRepository(h.DB)
	result := gin.H{}
	if opts.SearchType == "all" || opts.SearchType == "posts" || opts.SearchType == "videos" {
		var total int64
		var posts []repositories.SearchPostBundle
		var tagStats []repositories.SearchTagStat
		group, groupCtx := errgroup.WithContext(ctx)
		group.Go(func() error {
			var err error
			total, posts, err = repo.SearchPosts(groupCtx, repositories.SearchPostParams{
				Keyword:            opts.Keyword,
				Tag:                opts.Tag,
				Type:               opts.SearchType,
				Page:               opts.Page,
				Limit:              opts.Limit,
				PublicAccessOnly:   opts.PublicAccessOnly,
				ExcludeVideoGuests: opts.ExcludeVideoGuests,
			})
			return err
		})
		group.Go(func() error {
			var err error
			tagStats, err = repo.TagStats(groupCtx, repositories.SearchPostParams{
				Keyword:            opts.Keyword,
				Type:               opts.SearchType,
				PublicAccessOnly:   opts.PublicAccessOnly,
				ExcludeVideoGuests: opts.ExcludeVideoGuests,
			})
			return err
		})
		if err := group.Wait(); err != nil {
			return nil, err
		}

		postIDs := searchBundlePostIDs(posts)
		purchased, liked, collected, err := repositories.NewContentRepository(h.DB).InteractionSets(ctx, opts.CurrentUserID, postIDs)
		if err != nil {
			return nil, err
		}

		formattedPosts := make([]gin.H, 0, len(posts))
		for _, bundle := range posts {
			formattedPosts = append(formattedPosts, h.searchPostResponse(bundle, opts.CurrentUserID, purchased[bundle.Post.ID], liked[bundle.Post.ID], collected[bundle.Post.ID]))
		}
		stats := searchTagStatsResponse(tagStats)
		pageBody := pagination(opts.Page, opts.Limit, total)
		if opts.SearchType == "all" {
			result["data"] = formattedPosts
			result["tagStats"] = stats
			result["pagination"] = pageBody
		} else {
			result["posts"] = gin.H{
				"data":       formattedPosts,
				"tagStats":   stats,
				"pagination": pageBody,
			}
		}
	}

	if opts.SearchType == "users" {
		total, users, err := repo.SearchUsers(ctx, repositories.SearchUserParams{
			Keyword: opts.Keyword,
			Page:    opts.Page,
			Limit:   opts.Limit,
		})
		if err != nil {
			return nil, err
		}
		userIDs := searchBundleUserIDs(users)
		following, followers, err := repo.FollowSets(ctx, opts.CurrentUserID, userIDs)
		if err != nil {
			return nil, err
		}
		formattedUsers := make([]gin.H, 0, len(users))
		for _, bundle := range users {
			formattedUsers = append(formattedUsers, h.searchUserResponse(bundle, opts.CurrentUserID, following[bundle.User.ID], followers[bundle.User.ID]))
		}
		result["users"] = gin.H{
			"data":       formattedUsers,
			"pagination": pagination(opts.Page, opts.Limit, total),
		}
	}

	data := gin.H{"keyword": opts.Keyword, "tag": opts.Tag, "type": opts.SearchType}
	maps.Copy(data, result)
	return data, nil
}

func (h NativeHandlers) searchPostResponse(bundle repositories.SearchPostBundle, currentUserID int64, hasPurchased, liked, collected bool) gin.H {
	post := bundle.Post
	body := gin.H{
		"id":                   int(post.ID),
		"user_id":              int(post.UserID),
		"title":                post.Title,
		"content":              post.Content,
		"category_id":          post.CategoryID,
		"type":                 post.Type,
		"view_count":           post.ViewCount,
		"like_count":           post.LikeCount,
		"collect_count":        post.CollectCount,
		"comment_count":        post.CommentCount,
		"created_at":           post.CreatedAt,
		"is_draft":             post.IsDraft,
		"visibility":           post.Visibility,
		"public_access_exempt": post.PublicAccessExempt,
		"liked":                liked,
		"collected":            collected,
	}
	if body["visibility"] == "" {
		body["visibility"] = "public"
	}
	if bundle.User != nil {
		body["nickname"] = bundle.User.Nickname
		body["user_avatar"] = h.signFileURLPtr(bundle.User.Avatar)
		body["author_account"] = bundle.User.UserID
		body["avatar"] = h.signFileURLPtr(bundle.User.Avatar)
		body["author"] = bundle.User.Nickname
		body["location"] = bundle.User.Location
	} else {
		body["nickname"] = nil
		body["user_avatar"] = nil
		body["author_account"] = nil
		body["avatar"] = nil
		body["author"] = nil
		body["location"] = nil
	}

	imageItems := make([]searchImageItem, 0, len(bundle.Images))
	for _, image := range bundle.Images {
		imageItems = append(imageItems, searchImageItem{
			URL:           image.ImageURL,
			IsFreePreview: image.IsFreePreview,
			IsProtected:   image.IsProtected,
			SortOrder:     image.SortOrder,
		})
	}
	var video *domain.PostVideo
	if len(bundle.Videos) > 0 {
		video = &bundle.Videos[0]
	}
	h.protectSearchPostListItem(body, bundle.PaymentSetting, currentUserID != 0 && post.UserID == currentUserID, hasPurchased, video, imageItems)

	tags := make([]gin.H, 0, len(bundle.Tags))
	for _, tag := range bundle.Tags {
		tags = append(tags, gin.H{"id": tag.ID, "name": tag.Name})
	}
	body["tags"] = tags
	return body
}

type searchImageItem struct {
	URL           string
	IsFreePreview bool
	IsProtected   bool
	SortOrder     int
}

func (h NativeHandlers) protectSearchPostListItem(post gin.H, paymentSetting *domain.PostPaymentSetting, isAuthor bool, hasPurchased bool, videoData *domain.PostVideo, imageItems []searchImageItem) {
	paid := paymentSetting != nil && paymentSetting.Enabled
	protect := paid && !isAuthor && !hasPurchased
	hideAll := paymentSetting != nil && paymentSetting.HideAll
	postType, _ := post["type"].(int)

	if postType == 2 {
		if videoData != nil && videoData.CoverURL != nil {
			cover := h.signFileURL(*videoData.CoverURL)
			post["images"] = []string{cover}
			post["image"] = cover
		} else {
			post["images"] = []string{}
			post["image"] = nil
		}
		previewDuration := 0
		if paymentSetting != nil {
			previewDuration = paymentSetting.PreviewDuration
		}
		if protect {
			switch {
			case hideAll:
				post["video_url"] = nil
				post["preview_video_url"] = nil
			case videoData != nil && videoData.PreviewVideoURL != nil && *videoData.PreviewVideoURL != "":
				post["video_url"] = nil
				post["preview_video_url"] = h.signFileURLPtr(videoData.PreviewVideoURL)
			case previewDuration > 0 && videoData != nil:
				post["video_url"] = h.signFileURL(videoData.VideoURL)
				post["preview_video_url"] = nil
			default:
				post["video_url"] = nil
				post["preview_video_url"] = nil
			}
		} else if videoData != nil {
			post["video_url"] = h.signFileURL(videoData.VideoURL)
			post["preview_video_url"] = nil
		} else {
			post["video_url"] = nil
			post["preview_video_url"] = nil
		}
	} else {
		images := sortedSearchImages(imageItems)
		directImages := make([]searchImageItem, 0, len(images))
		protectedCount := 0
		hiddenPaidCount := 0
		for _, image := range images {
			if image.IsProtected {
				protectedCount++
				if protect && !image.IsFreePreview {
					hiddenPaidCount++
				}
				continue
			}
			if protect && (hideAll || !image.IsFreePreview) {
				hiddenPaidCount++
				continue
			}
			directImages = append(directImages, image)
		}
		var cover any
		if len(directImages) > 0 {
			cover = h.signFileURL(directImages[0].URL)
		}
		post["images"] = h.searchImagesResponse(directImages)
		post["image"] = cover
		post["totalImagesCount"] = len(images)
		post["hiddenPaidImagesCount"] = hiddenPaidCount
		post["protectedImagesCount"] = protectedCount
		post["protectedPackageRequired"] = protectedCount > 0
	}

	post["isPaidContent"] = paid
	if paymentSetting != nil {
		post["paymentSettings"] = gin.H{
			"enabled":          paymentSetting.Enabled,
			"freePreviewCount": paymentSetting.FreePreviewCount,
			"previewDuration":  paymentSetting.PreviewDuration,
			"price":            paymentSetting.Price,
			"hideAll":          paymentSetting.HideAll,
		}
	} else {
		post["paymentSettings"] = nil
	}
	post["hasPurchased"] = hasPurchased
}

func sortedSearchImages(images []searchImageItem) []searchImageItem {
	out := append([]searchImageItem(nil), images...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return out[i].URL < out[j].URL
	})
	if len(out) > 0 {
		out[0].IsFreePreview = true
		out[0].IsProtected = false
	}
	return out
}

func (h NativeHandlers) searchImagesResponse(images []searchImageItem) []gin.H {
	out := make([]gin.H, 0, len(images))
	for _, image := range images {
		out = append(out, gin.H{"url": h.signFileURL(image.URL), "isFreePreview": image.IsFreePreview})
	}
	return out
}

func (h NativeHandlers) searchUserResponse(bundle repositories.SearchUserBundle, currentUserID int64, isFollowing, isFollower bool) gin.H {
	user := bundle.User
	buttonType := "follow"
	isMutual := false
	switch {
	case currentUserID != 0 && user.ID == currentUserID:
		buttonType = "self"
	case isFollowing && isFollower:
		buttonType = "mutual"
		isMutual = true
	case isFollowing:
		buttonType = "unfollow"
	case isFollower:
		buttonType = "back"
	}
	return gin.H{
		"id":           int(user.ID),
		"user_id":      user.UserID,
		"nickname":     user.Nickname,
		"avatar":       h.signFileURLPtr(user.Avatar),
		"bio":          user.Bio,
		"location":     user.Location,
		"follow_count": user.FollowCount,
		"fans_count":   user.FansCount,
		"like_count":   user.LikeCount,
		"created_at":   user.CreatedAt,
		"verified":     user.Verified,
		"post_count":   bundle.PostCount,
		"isFollowing":  isFollowing,
		"isMutual":     isMutual,
		"buttonType":   buttonType,
	}
}

func searchTagStatsResponse(stats []repositories.SearchTagStat) []gin.H {
	out := make([]gin.H, 0, len(stats))
	for _, stat := range stats {
		out = append(out, gin.H{"id": stat.Name, "label": stat.Name, "count": stat.Count})
	}
	return out
}

func searchBundlePostIDs(bundles []repositories.SearchPostBundle) []int64 {
	out := make([]int64, 0, len(bundles))
	for _, bundle := range bundles {
		out = append(out, bundle.Post.ID)
	}
	return out
}

func searchBundleUserIDs(bundles []repositories.SearchUserBundle) []int64 {
	out := make([]int64, 0, len(bundles))
	for _, bundle := range bundles {
		out = append(out, bundle.User.ID)
	}
	return out
}
