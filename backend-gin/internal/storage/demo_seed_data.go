package storage

import (
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/datatypes"
)

func demoTranslations(value string) datatypes.JSON {
	return jsonValue(map[string]string{
		"en":    value,
		"zh-CN": value,
		"zh-TW": value,
		"vi":    value,
		"ja":    value,
		"ko":    value,
	})
}

func jsonValue(value any) datatypes.JSON {
	raw, err := json.Marshal(value)
	if err != nil {
		return datatypes.JSON([]byte("null"))
	}
	return datatypes.JSON(raw)
}

func isEmptyJSON(value datatypes.JSON) bool {
	trimmed := strings.TrimSpace(string(value))
	return trimmed == "" || trimmed == "null" || trimmed == "{}"
}

func stringPtr(value string) *string {
	return &value
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

var demoCategorySeeds = []demoCategorySeed{
	{Name: "photography", Title: "Photography"},
	{Name: "technology", Title: "Technology"},
	{Name: "lifestyle", Title: "Lifestyle"},
	{Name: "food", Title: "Food"},
	{Name: "travel", Title: "Travel"},
	{Name: "video", Title: "Video"},
}

var demoTagSeeds = []string{
	"demo",
	"campus",
	"photography",
	"technology",
	"lifestyle",
	"food",
	"travel",
	"video",
	"creator",
	"qa",
}

var demoUserSeeds = []demoUserSeed{
	{UserID: "demo_alice", Nickname: "Demo Alice", Email: "demo-alice@example.test", Avatar: unsplash("photo-1494790108377-be9c29b29330", 256, 256), Bio: "Campus photographer and product tester.", Location: "Demo City", Gender: "female", Education: "Undergraduate", Major: "Visual Design", MBTI: "ENFP", Interests: []string{"photography", "travel", "coffee"}, DaysAgo: 64, Points: 680, Wallet: 38.50, Earnings: 92.40},
	{UserID: "demo_ben", Nickname: "Demo Ben", Email: "demo-ben@example.test", Avatar: unsplash("photo-1500648767791-00dcc994a43e", 256, 256), Bio: "Backend engineer exploring creator flows.", Location: "North Campus", Gender: "male", Education: "Graduate", Major: "Computer Science", MBTI: "INTJ", Interests: []string{"technology", "video", "systems"}, DaysAgo: 52, Points: 540, Wallet: 21.00, Earnings: 45.10},
	{UserID: "demo_cora", Nickname: "Demo Cora", Email: "demo-cora@example.test", Avatar: unsplash("photo-1534528741775-53994a69daeb", 256, 256), Bio: "Food notes, field trips, and small daily rituals.", Location: "West Gate", Gender: "female", Education: "Undergraduate", Major: "Media Studies", MBTI: "ISFP", Interests: []string{"food", "lifestyle", "writing"}, DaysAgo: 48, Points: 720, Wallet: 12.75, Earnings: 63.25},
	{UserID: "demo_drew", Nickname: "Demo Drew", Email: "demo-drew@example.test", Avatar: unsplash("photo-1506794778202-cad84cf45f1d", 256, 256), Bio: "Travel planner and weekend route collector.", Location: "East Campus", Gender: "male", Education: "Undergraduate", Major: "Architecture", MBTI: "ENTP", Interests: []string{"travel", "maps", "photography"}, DaysAgo: 36, Points: 430, Wallet: 8.25, Earnings: 18.00},
	{UserID: "demo_elin", Nickname: "Demo Elin", Email: "demo-elin@example.test", Avatar: unsplash("photo-1544005313-94ddf0286df2", 256, 256), Bio: "Music, study rooms, and useful checklists.", Location: "Library Hub", Gender: "female", Education: "Graduate", Major: "Education", MBTI: "INFJ", Interests: []string{"lifestyle", "music", "study"}, DaysAgo: 28, Points: 610, Wallet: 17.90, Earnings: 27.50},
	{UserID: "demo_finn", Nickname: "Demo Finn", Email: "demo-finn@example.test", Avatar: unsplash("photo-1507003211169-0a1dd7228f2d", 256, 256), Bio: "Video clips and campus event coverage.", Location: "Studio B", Gender: "male", Education: "Undergraduate", Major: "Film", MBTI: "ESFP", Interests: []string{"video", "creator", "sports"}, DaysAgo: 21, Points: 390, Wallet: 6.40, Earnings: 34.70},
}

var demoPostSeeds = []demoPostSeed{
	{Key: "library-light", Author: "demo_alice", Title: "Morning light in the library", Category: "photography", Type: demoPostTypeImage, Tags: []string{"demo", "campus", "photography"}, Images: []string{unsplash("photo-1498243691581-b145c3f54a5a", 1200, 900), unsplash("photo-1524995997946-a1c2e315a42f", 1200, 900)}, Content: "A quiet set of reference images for feed layout, image preview, and comment QA.", ViewCount: 420, DaysAgo: 1, HoursOffset: -4, QualityLevel: "featured"},
	{Key: "release-checklist", Author: "demo_ben", Title: "Release checklist for a calm Friday", Category: "technology", Type: demoPostTypeImage, Tags: []string{"demo", "technology", "qa"}, Images: []string{unsplash("photo-1516321318423-f06f85e504b3", 1200, 900)}, Attachment: &demoAttachmentSeed{URL: "https://example.com/demo/release-checklist.pdf", Filename: "release-checklist.pdf", Filesize: 245760}, Content: "A compact launch checklist with logging, rollback, and smoke-test reminders.", ViewCount: 315, DaysAgo: 2, HoursOffset: -2},
	{Key: "coffee-map", Author: "demo_cora", Title: "Three coffee corners near campus", Category: "food", Type: demoPostTypeImage, Tags: []string{"demo", "food", "campus"}, Images: []string{unsplash("photo-1495474472287-4d71bcdd2085", 1200, 900), unsplash("photo-1509042239860-f550ce710b93", 1200, 900)}, Content: "A tasty data point for category tabs, search, and collection flows.", ViewCount: 536, DaysAgo: 3, HoursOffset: -1},
	{Key: "weekend-route", Author: "demo_drew", Title: "A walkable weekend route", Category: "travel", Type: demoPostTypeImage, Tags: []string{"demo", "travel", "photography"}, Images: []string{unsplash("photo-1500530855697-b586d89ba3ee", 1200, 900)}, Content: "A short route with stops that make good card thumbnails and profile history.", ViewCount: 284, DaysAgo: 4, HoursOffset: -3},
	{Key: "study-reset", Author: "demo_elin", Title: "Tiny reset rituals between study blocks", Category: "lifestyle", Type: demoPostTypeImage, Tags: []string{"demo", "lifestyle", "campus"}, Images: []string{unsplash("photo-1517842645767-c639042777db", 1200, 900)}, Content: "Five small habits that make a long study day easier to scan and revisit.", ViewCount: 198, DaysAgo: 5, HoursOffset: -5},
	{Key: "campus-video", Author: "demo_finn", Title: "Campus night market video test", Category: "video", Type: demoPostTypeVideo, Tags: []string{"demo", "video", "creator"}, Images: []string{unsplash("photo-1517457373958-b7bdd4587205", 1200, 900)}, VideoURL: "https://storage.googleapis.com/shaka-demo-assets/angel-one-hls/hls.m3u8", VideoCover: unsplash("photo-1517457373958-b7bdd4587205", 1200, 900), Content: "A video-card sample for player, cover, and guest-access checks.", ViewCount: 642, DaysAgo: 1, HoursOffset: -8},
	{Key: "paid-pack", Author: "demo_alice", Title: "Protected photo pack sample", Category: "photography", Type: demoPostTypeImage, Tags: []string{"demo", "photography", "creator"}, Images: []string{unsplash("photo-1500534314209-a25ddb2bd429", 1200, 900), unsplash("photo-1500530855697-b586d89ba3ee", 1200, 900), unsplash("photo-1506744038136-46273834b3fb", 1200, 900)}, Payment: &demoPaymentSeed{Price: 18, PaymentType: "image_pack", PaymentMethod: "points", FreePreviewCount: 1}, Content: "The first image is free. The remaining images simulate a protected paid pack.", ViewCount: 712, DaysAgo: 6, HoursOffset: -6, QualityLevel: "premium"},
	{Key: "desk-setup", Author: "demo_ben", Title: "Desk setup notes after two weeks", Category: "technology", Type: demoPostTypeImage, Tags: []string{"demo", "technology", "lifestyle"}, Images: []string{unsplash("photo-1497366754035-f200968a6e72", 1200, 900)}, Content: "A practical post for search ranking, recommendations, and tag detail checks.", ViewCount: 259, DaysAgo: 7, HoursOffset: -7},
	{Key: "meal-prep", Author: "demo_cora", Title: "Simple meal prep for exam week", Category: "food", Type: demoPostTypeImage, Tags: []string{"demo", "food", "lifestyle"}, Images: []string{unsplash("photo-1546069901-ba9599a7e63c", 1200, 900)}, Content: "Reusable content for hot feed and food-category smoke tests.", ViewCount: 447, DaysAgo: 8, HoursOffset: -2},
	{Key: "city-rooftop", Author: "demo_drew", Title: "City rooftop at blue hour", Category: "travel", Type: demoPostTypeImage, Tags: []string{"demo", "travel", "photography"}, Images: []string{unsplash("photo-1494526585095-c41746248156", 1200, 900)}, Content: "A calm travel sample with enough engagement to appear in hot sorting.", ViewCount: 388, DaysAgo: 9, HoursOffset: -4},
	{Key: "playlist", Author: "demo_elin", Title: "Playlist for a focused afternoon", Category: "lifestyle", Type: demoPostTypeImage, Tags: []string{"demo", "lifestyle", "creator"}, Images: []string{unsplash("photo-1516280440614-37939bbacd81", 1200, 900)}, Content: "A light lifestyle item for profile, comment, and collection flows.", ViewCount: 223, DaysAgo: 10, HoursOffset: -1},
	{Key: "event-recap", Author: "demo_finn", Title: "Event recap with creator notes", Category: "video", Type: demoPostTypeVideo, Tags: []string{"demo", "video", "campus"}, Images: []string{unsplash("photo-1523580494863-6f3031224c94", 1200, 900)}, VideoURL: "https://storage.googleapis.com/shaka-demo-assets/angel-one-widevine-hls/hls.m3u8", VideoCover: unsplash("photo-1523580494863-6f3031224c94", 1200, 900), Content: "A second video sample for lists, playback fallback, and creator analytics.", ViewCount: 501, DaysAgo: 11, HoursOffset: -9},
}

func unsplash(id string, width int, height int) string {
	return fmt.Sprintf("https://images.unsplash.com/%s?auto=format&fit=crop&w=%d&h=%d&q=75", id, width, height)
}
