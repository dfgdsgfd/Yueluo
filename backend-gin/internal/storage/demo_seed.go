package storage

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/security"
)

const (
	DefaultDemoPassword = "Demo123456!"

	demoPostTypeImage = 1
	demoPostTypeVideo = 2

	demoVisibilityPublic = "public"

	demoGiftCardCodeStatusAvailable = "available"
)

type DemoSeedOptions struct {
	Password  string
	UserLimit int
	PostLimit int
	Now       time.Time
}

type DemoSeedResult struct {
	UsersCreated               int64
	CategoriesCreated          int64
	TagsCreated                int64
	PostsCreated               int64
	PostImagesCreated          int64
	PostVideosCreated          int64
	PostAttachmentsCreated     int64
	PostPaymentSettingsCreated int64
	PostTagsCreated            int64
	CommentsCreated            int64
	LikesCreated               int64
	CollectionsCreated         int64
	FollowsCreated             int64
	NotificationsCreated       int64
	WalletRowsCreated          int64
	PointRowsCreated           int64
	CreatorRowsCreated         int64
	GiftCardRowsCreated        int64
	SystemRowsCreated          int64
	IMRowsCreated              int64
	HistoryRowsCreated         int64
	UserCount                  int
	CategoryCount              int
	TagCount                   int
	PostCount                  int
	LoginAccounts              []DemoLoginAccount
	Password                   string
}

type DemoLoginAccount struct {
	Account string
	Email   string
}

type demoSeedState struct {
	passwordHash string
	result       DemoSeedResult
	users        map[string]domain.User
	categories   map[string]domain.Category
	tags         map[string]domain.Tag
	posts        map[string]domain.Post
}

type demoUserSeed struct {
	UserID    string
	Nickname  string
	Email     string
	Avatar    string
	Bio       string
	Location  string
	Gender    string
	Education string
	Major     string
	MBTI      string
	Interests []string
	DaysAgo   int
	Points    float64
	Wallet    float64
	Earnings  float64
}

type demoCategorySeed struct {
	Name  string
	Title string
}

type demoPostSeed struct {
	Key          string
	Author       string
	Title        string
	LegacyTitle  string
	Content      string
	Category     string
	Type         int
	Tags         []string
	Images       []string
	VideoURL     string
	VideoCover   string
	Attachment   *demoAttachmentSeed
	Payment      *demoPaymentSeed
	ViewCount    int64
	DaysAgo      int
	HoursOffset  int
	QualityLevel string
}

type demoAttachmentSeed struct {
	URL      string
	Filename string
	Filesize int64
}

type demoPaymentSeed struct {
	Price            float64
	PaymentType      string
	PaymentMethod    string
	FreePreviewCount int
	PreviewDuration  int
	HideAll          bool
}

func SeedDemoData(ctx context.Context, db *gorm.DB, options DemoSeedOptions) (DemoSeedResult, error) {
	if db == nil {
		return DemoSeedResult{}, errors.New("database is required")
	}
	options = normalizeDemoSeedOptions(options)
	passwordHash, err := security.HashPassword(options.Password)
	if err != nil {
		return DemoSeedResult{}, err
	}
	state := &demoSeedState{
		passwordHash: passwordHash,
		result: DemoSeedResult{
			Password:      options.Password,
			UserCount:     len(selectedDemoUsers(options.UserLimit)),
			CategoryCount: len(demoCategorySeeds),
			TagCount:      len(demoTagSeeds),
			PostCount:     len(selectedDemoPosts(options.PostLimit)),
		},
		users:      map[string]domain.User{},
		categories: map[string]domain.Category{},
		tags:       map[string]domain.Tag{},
		posts:      map[string]domain.Post{},
	}
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := state.seedCategories(ctx, tx, options.Now); err != nil {
			return err
		}
		if err := state.seedTags(ctx, tx, options.Now); err != nil {
			return err
		}
		if err := state.seedUsers(ctx, tx, selectedDemoUsers(options.UserLimit), options.Now); err != nil {
			return err
		}
		if err := state.seedRelationships(ctx, tx, options.Now); err != nil {
			return err
		}
		posts := selectedDemoPosts(options.PostLimit)
		if err := state.seedPosts(ctx, tx, posts, options.Now); err != nil {
			return err
		}
		if err := state.seedContentActivity(ctx, tx, posts, options.Now); err != nil {
			return err
		}
		if err := state.seedCommerce(ctx, tx, options.Now); err != nil {
			return err
		}
		if err := state.seedSystemRows(ctx, tx, options.Now); err != nil {
			return err
		}
		if err := state.seedIM(ctx, tx, options.Now); err != nil {
			return err
		}
		if err := state.recountAffectedRows(ctx, tx, options.Now); err != nil {
			return err
		}
		return nil
	})
	return state.result, err
}

func normalizeDemoSeedOptions(options DemoSeedOptions) DemoSeedOptions {
	if strings.TrimSpace(options.Password) == "" {
		options.Password = DefaultDemoPassword
	}
	if options.UserLimit <= 0 || options.UserLimit > len(demoUserSeeds) {
		options.UserLimit = len(demoUserSeeds)
	}
	if options.PostLimit <= 0 || options.PostLimit > len(demoPostSeeds) {
		options.PostLimit = len(demoPostSeeds)
	}
	if options.Now.IsZero() {
		options.Now = time.Now()
	}
	return options
}

func selectedDemoUsers(limit int) []demoUserSeed {
	if limit <= 0 || limit > len(demoUserSeeds) {
		limit = len(demoUserSeeds)
	}
	return demoUserSeeds[:limit]
}

func selectedDemoPosts(limit int) []demoPostSeed {
	if limit <= 0 || limit > len(demoPostSeeds) {
		limit = len(demoPostSeeds)
	}
	return demoPostSeeds[:limit]
}
