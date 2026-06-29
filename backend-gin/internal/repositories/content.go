package repositories

import (
	"context"
	"errors"
	"sort"
	"strings"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

const (
	PostTypeImage       = 1
	PostTypeVideo       = 2
	PostTypeVideoCenter = 3

	VisibilityPublic      = "public"
	VisibilityPrivate     = "private"
	VisibilityFriendsOnly = "friends_only"

	PostQualityNone = "none"
)

type ContentRepository struct {
	db                      *gorm.DB
	notificationSuppression NotificationSuppressionConfig
}

type CommentBundle struct {
	Comment    domain.Comment
	User       *domain.User
	Liked      bool
	ReplyCount int64
}

type PostBundle struct {
	Post                domain.Post
	User                *domain.User
	Category            *domain.Category
	Images              []domain.PostImage
	Videos              []domain.PostVideo
	Attachments         []domain.PostAttachment
	Tags                []domain.Tag
	PaymentSetting      *domain.PostPaymentSetting
	RecommendationScore *float64
	ScoreBreakdown      map[string]float64
}

type PostListOptions struct {
	Page               int
	Limit              int
	CategoryID         *int
	Type               *int
	IsDraft            bool
	UserID             *int64
	CurrentUserID      int64
	Sort               string
	Mode               string
	TimeRangeDays      int
	ExcludeVideoGuests bool
	PublicAccessOnly   bool
	RecommendConfig    any
	BlockedUserIDs     []int64
}

type PostListResult struct {
	Total            int64
	Posts            []PostBundle
	HasFriends       *bool
	HasFollowing     *bool
	RecommendedUsers []RecommendedUser
	Debug            map[string]any
}

type RecommendedUser struct {
	User      domain.User
	PostCount int64
}

type CreatePostInput struct {
	UserID          int64
	Title           string
	Content         string
	CategoryID      *int
	Type            int
	Images          []PostImageInput
	Video           *PostVideoInput
	Attachment      *PostAttachmentInput
	Tags            []string
	IsDraft         bool
	PaymentSettings *PaymentSettingsInput
	Visibility      string
	AuditStatus     int
	AuditResult     datatypes.JSON
	HoldSideEffects bool
}

type UpdatePostInput struct {
	UserID          int64
	PostID          int64
	Title           *string
	Content         *string
	CategoryIDSet   bool
	CategoryID      *int
	Type            *int
	ImagesSet       bool
	Images          []PostImageInput
	VideoSet        bool
	Video           *PostVideoInput
	AttachmentSet   bool
	Attachment      *PostAttachmentInput
	TagsSet         bool
	Tags            []string
	IsDraft         *bool
	PaymentSet      bool
	PaymentSettings *PaymentSettingsInput
	Visibility      *string
}

type PostImageInput struct {
	URL                 string
	WatermarkTraceToken string
	IsFreePreview       bool
	IsProtected         bool
	SortOrder           int
}

type PostVideoInput struct {
	URL      string
	CoverURL *string
}

type PostAttachmentInput struct {
	URL      string
	Filename string
	Filesize int64
}

type PaymentSettingsInput struct {
	Enabled          bool
	PaymentType      string
	PaymentMethod    string
	Price            float64
	FreePreviewCount int
	PreviewDuration  int
	HideAll          bool
}

// NormalizePostImageInputs makes the persisted image order authoritative and
// reserves the first image as the public, directly viewable post cover.
func NormalizePostImageInputs(images []PostImageInput) []PostImageInput {
	normalized := make([]PostImageInput, 0, len(images))
	for index, image := range images {
		if strings.TrimSpace(image.URL) == "" {
			continue
		}
		if image.SortOrder <= 0 {
			image.SortOrder = index + 1
		}
		normalized = append(normalized, image)
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		return normalized[i].SortOrder < normalized[j].SortOrder
	})
	for index := range normalized {
		normalized[index].SortOrder = index + 1
	}
	if len(normalized) > 0 {
		normalized[0].IsFreePreview = true
		normalized[0].IsProtected = false
	}
	return normalized
}

func PostImageAccessCounts(images []PostImageInput) (free, paid int) {
	for _, image := range NormalizePostImageInputs(images) {
		if image.IsFreePreview {
			free++
		} else {
			paid++
		}
	}
	return free, paid
}

func NormalizePostImagesForAccess(images []domain.PostImage) []domain.PostImage {
	normalized := append([]domain.PostImage(nil), images...)
	sort.SliceStable(normalized, func(i, j int) bool {
		if normalized[i].SortOrder != normalized[j].SortOrder {
			return normalized[i].SortOrder < normalized[j].SortOrder
		}
		return normalized[i].ID < normalized[j].ID
	})
	if len(normalized) > 0 {
		normalized[0].IsFreePreview = true
		normalized[0].IsProtected = false
	}
	return normalized
}

type CreateCommentResult struct {
	Comment CommentBundle
}

type CreateCommentOptions struct {
	AuditResult     datatypes.JSON
	AuditStatus     int
	IsPublic        *bool
	HoldSideEffects bool
}

var (
	ErrContentPostMissing     = errors.New("post missing")
	ErrContentParentMissing   = errors.New("parent comment missing")
	ErrContentCommentMissing  = errors.New("comment missing")
	ErrContentForbidden       = errors.New("forbidden")
	ErrContentNoFriends       = errors.New("no friends")
	ErrContentNoFollowing     = errors.New("no following")
	ErrContentAlreadyExists   = errors.New("already exists")
	ErrContentInvalidArgument = errors.New("invalid argument")
)

func NewContentRepository(db *gorm.DB, configs ...NotificationSuppressionConfig) ContentRepository {
	repo := ContentRepository{db: db}
	if len(configs) > 0 {
		repo.notificationSuppression = configs[0]
	}
	return repo
}

func (r ContentRepository) Comments(ctx context.Context, postID int64, parentID *int64, currentUserID int64, page, limit int, asc bool, withReplyCount bool) (int64, []CommentBundle, error) {
	query := r.visibleCommentsQuery(ctx, currentUserID)
	if parentID != nil {
		query = query.Where("parent_id = ?", *parentID)
	} else {
		query = query.Where("post_id = ? AND parent_id IS NULL", postID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	order := "created_at DESC"
	if asc {
		order = "created_at ASC"
	}
	var comments []domain.Comment
	if err := query.Order(order).Offset((page - 1) * limit).Limit(limit).Find(&comments).Error; err != nil {
		return 0, nil, err
	}
	bundles, err := r.commentBundles(ctx, comments, currentUserID, withReplyCount)
	return total, bundles, err
}

func (r ContentRepository) CreateComment(ctx context.Context, userID, postID int64, parentID *int64, content string, options ...CreateCommentOptions) (*CreateCommentResult, error) {
	opts := CreateCommentOptions{}
	if len(options) > 0 {
		opts = options[0]
	}
	var created domain.Comment
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		effectivePostID := postID
		var parent domain.Comment
		if parentID != nil {
			if err := tx.Select("id", "post_id", "user_id").Where("id = ?", *parentID).First(&parent).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrContentParentMissing
				}
				return err
			}
			if parent.PostID > 0 {
				effectivePostID = parent.PostID
			}
		}
		var post domain.Post
		if err := tx.Select("id", "user_id").Where("id = ?", effectivePostID).First(&post).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrContentPostMissing
			}
			return err
		}
		created = domain.Comment{
			PostID:      effectivePostID,
			UserID:      userID,
			ParentID:    parentID,
			Content:     content,
			AuditStatus: 1,
			IsPublic:    true,
		}
		if opts.AuditStatus != 0 {
			created.AuditStatus = opts.AuditStatus
		}
		if opts.IsPublic != nil {
			created.IsPublic = *opts.IsPublic
		}
		if len(opts.AuditResult) > 0 {
			created.AuditResult = opts.AuditResult
		}
		if err := tx.Create(&created).Error; err != nil {
			return err
		}
		if opts.HoldSideEffects || !created.IsPublic || created.AuditStatus != 1 {
			return nil
		}
		if err := tx.Model(&domain.Post{}).Where("id = ?", effectivePostID).UpdateColumn("comment_count", gorm.Expr("comment_count + ?", 1)).Error; err != nil {
			return err
		}
		if parentID != nil && parent.UserID != userID {
			targetID := effectivePostID
			commentID := created.ID
			if err := tx.Create(&domain.Notification{UserID: parent.UserID, SenderID: userID, Type: 4, Title: "回复了你的评论", TargetID: &targetID, CommentID: &commentID}).Error; err != nil {
				return err
			}
		} else if parentID == nil && post.UserID != userID {
			targetID := effectivePostID
			commentID := created.ID
			if err := tx.Create(&domain.Notification{UserID: post.UserID, SenderID: userID, Type: 3, Title: "评论了你的笔记", TargetID: &targetID, CommentID: &commentID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	bundles, err := r.commentBundles(ctx, []domain.Comment{created}, userID, true)
	if err != nil {
		return nil, err
	}
	if len(bundles) == 0 {
		return &CreateCommentResult{}, nil
	}
	return &CreateCommentResult{Comment: bundles[0]}, nil
}

func (r ContentRepository) DeleteComment(ctx context.Context, userID, commentID int64) (int64, int64, error) {
	var postID int64
	var deleted int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var comment domain.Comment
		if err := tx.Select("id", "post_id", "user_id").Where("id = ?", commentID).First(&comment).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrContentCommentMissing
			}
			return err
		}
		if comment.UserID != userID {
			return ErrContentForbidden
		}
		postID = comment.PostID
		count, err := deleteCommentRecursive(ctx, tx, commentID)
		if err != nil {
			return err
		}
		deleted = count
		return tx.Model(&domain.Post{}).Where("id = ?", postID).UpdateColumn("comment_count", gorm.Expr("comment_count - ?", deleted)).Error
	})
	return deleted, postID, err
}

func (r ContentRepository) PostExists(ctx context.Context, postID int64) (*domain.Post, error) {
	var post domain.Post
	err := r.db.WithContext(ctx).Where("id = ?", postID).Select("id", "user_id", "type", "visibility", "public_access_exempt").First(&post).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrContentPostMissing
	}
	return &post, err
}

func (r ContentRepository) ListPosts(ctx context.Context, opts PostListOptions) (*PostListResult, error) {
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.Limit < 1 {
		opts.Limit = 20
	}
	if opts.TimeRangeDays < 1 {
		opts.TimeRangeDays = 7
	}

	switch opts.Mode {
	case "friends":
		return r.friendPosts(ctx, opts)
	case "following":
		return r.followingPosts(ctx, opts)
	case "hot":
		return r.hotPosts(ctx, opts)
	case "recommended":
		return r.recommendedPosts(ctx, opts)
	default:
		return r.standardPosts(ctx, opts)
	}
}

func (r ContentRepository) PostDetail(ctx context.Context, postID, currentUserID int64) (*PostBundle, error) {
	var post domain.Post
	if err := r.db.WithContext(ctx).Where("id = ?", postID).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContentPostMissing
		}
		return nil, err
	}
	canView, err := r.CanViewPost(ctx, post.UserID, post.Visibility, currentUserID)
	if err != nil {
		return nil, err
	}
	if !canView {
		return nil, ErrContentForbidden
	}
	if err := r.db.WithContext(ctx).Model(&domain.Post{}).Where("id = ?", postID).UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error; err != nil {
		return nil, err
	}
	post.ViewCount++
	bundles, err := r.postBundles(ctx, []domain.Post{post})
	if err != nil {
		return nil, err
	}
	if len(bundles) == 0 {
		return nil, ErrContentPostMissing
	}
	return &bundles[0], nil
}

func (r ContentRepository) PostBundlesForMatrix(ctx context.Context, posts []domain.Post) ([]PostBundle, error) {
	return r.postBundles(ctx, posts)
}

func (r ContentRepository) CreatePost(ctx context.Context, input CreatePostInput) (int64, error) {
	if input.Type == 0 {
		input.Type = PostTypeImage
	}
	input.Visibility = normalizeVisibility(input.Visibility)
	var postID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		post := postFromCreateInput(input)
		if err := tx.Create(&post).Error; err != nil {
			return err
		}
		postID = post.ID
		if err := createPostChildren(ctx, tx, postID, input.UserID, input.Images, input.Video, input.Attachment, input.PaymentSettings); err != nil {
			return err
		}
		if err := replacePostTags(ctx, tx, postID, nil, input.Tags); err != nil {
			return err
		}
		if input.CategoryID != nil {
			if err := tx.Model(&domain.Category{}).Where("id = ?", *input.CategoryID).UpdateColumn("use_count", gorm.Expr("use_count + ?", 1)).Error; err != nil {
				return err
			}
		}
		if !input.IsDraft && !input.HoldSideEffects && post.AuditStatus == 1 && post.Visibility == VisibilityPublic {
			if err := createFollowerNewPostNotifications(ctx, tx, input.UserID, postID, input.Title); err != nil {
				return err
			}
		}
		return nil
	})
	return postID, err
}

func postFromCreateInput(input CreatePostInput) domain.Post {
	auditStatus := input.AuditStatus
	if auditStatus == 0 && len(input.AuditResult) == 0 {
		auditStatus = 1
	}
	return domain.Post{
		UserID:       input.UserID,
		Title:        input.Title,
		Content:      input.Content,
		CategoryID:   input.CategoryID,
		Type:         input.Type,
		IsDraft:      input.IsDraft,
		Visibility:   input.Visibility,
		AuditStatus:  auditStatus,
		AuditResult:  input.AuditResult,
		QualityLevel: PostQualityNone,
	}
}

func (r ContentRepository) UpdatePost(ctx context.Context, input UpdatePostInput) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var post domain.Post
		if err := tx.Select("id", "user_id", "category_id").Where("id = ?", input.PostID).First(&post).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrContentPostMissing
			}
			return err
		}
		if post.UserID != input.UserID {
			return ErrContentForbidden
		}
		updates := map[string]any{}
		if input.Title != nil {
			updates["title"] = *input.Title
		}
		if input.Content != nil {
			updates["content"] = *input.Content
		}
		if input.CategoryIDSet {
			updates["category_id"] = input.CategoryID
		}
		if input.Type != nil {
			updates["type"] = *input.Type
		}
		if input.IsDraft != nil {
			updates["is_draft"] = *input.IsDraft
		}
		if input.Visibility != nil {
			updates["visibility"] = normalizeVisibility(*input.Visibility)
		}
		if len(updates) > 0 {
			if err := tx.Model(&domain.Post{}).Where("id = ?", input.PostID).Updates(updates).Error; err != nil {
				return err
			}
		}
		if input.ImagesSet {
			input.Images = NormalizePostImageInputs(input.Images)
			if err := PreparePostImageWatermarkTraceRebind(ctx, tx, input.PostID, input.Images); err != nil {
				return err
			}
			if err := tx.Where("post_id = ?", input.PostID).Delete(&domain.PostImage{}).Error; err != nil {
				return err
			}
			if err := createImages(ctx, tx, input.PostID, input.UserID, input.Images); err != nil {
				return err
			}
		}
		if input.VideoSet {
			if err := tx.Where("post_id = ?", input.PostID).Delete(&domain.PostVideo{}).Error; err != nil {
				return err
			}
			if input.Video != nil {
				if err := tx.Create(&domain.PostVideo{PostID: input.PostID, VideoURL: input.Video.URL, CoverURL: input.Video.CoverURL}).Error; err != nil {
					return err
				}
			}
		}
		if input.AttachmentSet {
			if err := tx.Where("post_id = ?", input.PostID).Delete(&domain.PostAttachment{}).Error; err != nil {
				return err
			}
			if input.Attachment != nil {
				if err := createAttachment(ctx, tx, input.PostID, *input.Attachment); err != nil {
					return err
				}
			}
		}
		if input.ImagesSet || input.VideoSet || input.AttachmentSet {
			if err := bindUploadAssetsToPost(ctx, tx, input.PostID, input.UserID, input.Images, input.Video, input.Attachment); err != nil {
				return err
			}
		}
		if input.TagsSet {
			if err := replacePostTags(ctx, tx, input.PostID, &input.PostID, input.Tags); err != nil {
				return err
			}
		}
		if input.CategoryIDSet {
			oldID := post.CategoryID
			newID := input.CategoryID
			if oldID != nil && (newID == nil || *newID != *oldID) {
				if err := tx.Model(&domain.Category{}).Where("id = ?", *oldID).UpdateColumn("use_count", gorm.Expr("use_count - ?", 1)).Error; err != nil {
					return err
				}
			}
			if newID != nil && (oldID == nil || *newID != *oldID) {
				if err := tx.Model(&domain.Category{}).Where("id = ?", *newID).UpdateColumn("use_count", gorm.Expr("use_count + ?", 1)).Error; err != nil {
					return err
				}
			}
		}
		if input.PaymentSet {
			if err := tx.Where("post_id = ?", input.PostID).Delete(&domain.PostPaymentSetting{}).Error; err != nil {
				return err
			}
			if input.PaymentSettings != nil && input.PaymentSettings.Enabled {
				free, paid := PostImageAccessCounts(input.Images)
				if input.ImagesSet {
					input.PaymentSettings.FreePreviewCount = free
					input.PaymentSettings.Enabled = paid > 0
					input.PaymentSettings.HideAll = false
				}
				if !input.PaymentSettings.Enabled {
					return nil
				}
				if err := createPayment(ctx, tx, input.PostID, *input.PaymentSettings); err != nil {
					return err
				}
			}
		} else if input.ImagesSet {
			free, paid := PostImageAccessCounts(input.Images)
			if paid == 0 {
				if err := tx.Where("post_id = ?", input.PostID).Delete(&domain.PostPaymentSetting{}).Error; err != nil {
					return err
				}
			} else if err := tx.Model(&domain.PostPaymentSetting{}).Where("post_id = ?", input.PostID).Updates(map[string]any{
				"free_preview_count": free,
				"hide_all":           false,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r ContentRepository) DeletePost(ctx context.Context, userID, postID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var post domain.Post
		if err := tx.Select("id", "user_id").Where("id = ?", postID).First(&post).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrContentPostMissing
			}
			return err
		}
		if post.UserID != userID {
			return ErrContentForbidden
		}
		var postTags []domain.PostTag
		if err := tx.Where("post_id = ?", postID).Select("tag_id").Find(&postTags).Error; err != nil {
			return err
		}
		tagIDs := make([]int, 0, len(postTags))
		for _, tag := range postTags {
			tagIDs = append(tagIDs, tag.TagID)
		}
		if len(tagIDs) > 0 {
			if err := tx.Model(&domain.Tag{}).Where("id IN ?", uniqueInt(tagIDs)).UpdateColumn("use_count", gorm.Expr("use_count - ?", 1)).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("post_id = ?", postID).Delete(&domain.PostImage{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", postID).Delete(&domain.ImageWatermarkTrace{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", postID).Delete(&domain.PostVideo{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", postID).Delete(&domain.PostAttachment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", postID).Delete(&domain.PostPaymentSetting{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", postID).Delete(&domain.PostTag{}).Error; err != nil {
			return err
		}
		return tx.Delete(&post).Error
	})
}

func (r ContentRepository) ToggleCollection(ctx context.Context, userID, postID int64) (bool, error) {
	collected := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var post domain.Post
		if err := tx.Select("id", "user_id").Where("id = ?", postID).First(&post).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrContentPostMissing
			}
			return err
		}
		var existing domain.Collection
		err := tx.Where("user_id = ? AND post_id = ?", userID, postID).First(&existing).Error
		if err == nil {
			if err := tx.Delete(&existing).Error; err != nil {
				return err
			}
			return tx.Model(&domain.Post{}).Where("id = ?", postID).UpdateColumn("collect_count", gorm.Expr("collect_count - ?", 1)).Error
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Create(&domain.Collection{UserID: userID, PostID: postID}).Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.Post{}).Where("id = ?", postID).UpdateColumn("collect_count", gorm.Expr("collect_count + ?", 1)).Error; err != nil {
			return err
		}
		collected = true
		if post.UserID != userID {
			targetID := postID
			notification := domain.Notification{UserID: post.UserID, SenderID: userID, Type: 6, Title: "收藏了你的笔记", TargetID: &targetID}
			if err := createNotificationIfAllowed(ctx, tx, notification, r.notificationSuppression); err != nil {
				return err
			}
		}
		return nil
	})
	return collected, err
}

func (r ContentRepository) InteractionSets(ctx context.Context, userID int64, postIDs []int64) (map[int64]bool, map[int64]bool, map[int64]bool, error) {
	purchased := map[int64]bool{}
	liked := map[int64]bool{}
	collected := map[int64]bool{}
	if userID == 0 || len(postIDs) == 0 {
		return purchased, liked, collected, nil
	}
	ids := uniqueInt64(postIDs)
	if err := runParallel(
		func() error {
			var purchases []domain.UserPurchasedContent
			if err := r.db.WithContext(ctx).Where("user_id = ? AND post_id IN ?", userID, ids).Select("post_id").Find(&purchases).Error; err != nil {
				return err
			}
			for _, row := range purchases {
				purchased[row.PostID] = true
			}
			return nil
		},
		func() error {
			var likes []domain.Like
			if err := r.db.WithContext(ctx).Where("user_id = ? AND target_type = ? AND target_id IN ?", userID, 1, ids).Select("target_id").Find(&likes).Error; err != nil {
				return err
			}
			for _, row := range likes {
				liked[row.TargetID] = true
			}
			return nil
		},
		func() error {
			var collections []domain.Collection
			if err := r.db.WithContext(ctx).Where("user_id = ? AND post_id IN ?", userID, ids).Select("post_id").Find(&collections).Error; err != nil {
				return err
			}
			for _, row := range collections {
				collected[row.PostID] = true
			}
			return nil
		},
	); err != nil {
		return nil, nil, nil, err
	}
	return purchased, liked, collected, nil
}

func (r ContentRepository) CanViewPost(ctx context.Context, authorID int64, visibility string, currentUserID int64) (bool, error) {
	if currentUserID != 0 && authorID == currentUserID {
		return true, nil
	}
	switch normalizeVisibility(visibility) {
	case VisibilityPublic:
		return true, nil
	case VisibilityPrivate:
		return false, nil
	case VisibilityFriendsOnly:
		if currentUserID == 0 {
			return false, nil
		}
		return r.areMutualFollowers(ctx, currentUserID, authorID)
	default:
		return true, nil
	}
}
