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

type demoTagSeed struct {
	Name        string
	LegacyNames []string
}

func (seed demoPostSeed) postTitles() []string {
	titles := []string{seed.Title}
	if strings.TrimSpace(seed.LegacyTitle) != "" {
		titles = append(titles, seed.LegacyTitle)
	}
	return titles
}

var demoCategorySeeds = []demoCategorySeed{
	{Name: "photography", Title: "摄影"},
	{Name: "technology", Title: "科技"},
	{Name: "lifestyle", Title: "生活"},
	{Name: "food", Title: "美食"},
	{Name: "travel", Title: "旅行"},
	{Name: "video", Title: "视频"},
}

var demoTagSeeds = []demoTagSeed{
	{Name: "演示", LegacyNames: []string{"demo"}},
	{Name: "校园", LegacyNames: []string{"campus"}},
	{Name: "摄影", LegacyNames: []string{"photography"}},
	{Name: "科技", LegacyNames: []string{"technology"}},
	{Name: "生活", LegacyNames: []string{"lifestyle"}},
	{Name: "美食", LegacyNames: []string{"food"}},
	{Name: "旅行", LegacyNames: []string{"travel"}},
	{Name: "视频", LegacyNames: []string{"video"}},
	{Name: "创作者", LegacyNames: []string{"creator"}},
	{Name: "测试", LegacyNames: []string{"qa"}},
}

var demoUserSeeds = []demoUserSeed{
	{UserID: "demo_alice", Nickname: "林晓雨", Email: "demo-alice@example.test", Avatar: unsplash("photo-1494790108377-be9c29b29330", 256, 256), Bio: "上海高校摄影社成员，喜欢记录城市清晨和校园角落。", Location: "中国 上海 杨浦", Gender: "女", Education: "本科", Major: "视觉传达设计", MBTI: "ENFP", Interests: []string{"摄影", "旅行", "咖啡"}, DaysAgo: 64, Points: 680, Wallet: 38.50, Earnings: 92.40},
	{UserID: "demo_ben", Nickname: "陈柏言", Email: "demo-ben@example.test", Avatar: unsplash("photo-1500648767791-00dcc994a43e", 256, 256), Bio: "北京的后端工程师，常写发布检查清单和校园产品笔记。", Location: "中国 北京 海淀", Gender: "男", Education: "研究生", Major: "计算机科学", MBTI: "INTJ", Interests: []string{"科技", "视频", "系统设计"}, DaysAgo: 52, Points: 540, Wallet: 21.00, Earnings: 45.10},
	{UserID: "demo_cora", Nickname: "周可然", Email: "demo-cora@example.test", Avatar: unsplash("photo-1534528741775-53994a69daeb", 256, 256), Bio: "成都生活记录者，喜欢小店、美食和慢节奏的周末。", Location: "中国 四川 成都", Gender: "女", Education: "本科", Major: "新闻传播", MBTI: "ISFP", Interests: []string{"美食", "生活", "写作"}, DaysAgo: 48, Points: 720, Wallet: 12.75, Earnings: 63.25},
	{UserID: "demo_drew", Nickname: "何远舟", Email: "demo-drew@example.test", Avatar: unsplash("photo-1506794778202-cad84cf45f1d", 256, 256), Bio: "杭州路线收集者，周末用脚步丈量西湖和街巷。", Location: "中国 浙江 杭州", Gender: "男", Education: "本科", Major: "建筑学", MBTI: "ENTP", Interests: []string{"旅行", "地图", "摄影"}, DaysAgo: 36, Points: 430, Wallet: 8.25, Earnings: 18.00},
	{UserID: "demo_elin", Nickname: "许安宁", Email: "demo-elin@example.test", Avatar: unsplash("photo-1544005313-94ddf0286df2", 256, 256), Bio: "南京图书馆常驻用户，整理学习方法和中文歌单。", Location: "中国 江苏 南京", Gender: "女", Education: "研究生", Major: "教育学", MBTI: "INFJ", Interests: []string{"生活", "音乐", "学习"}, DaysAgo: 28, Points: 610, Wallet: 17.90, Earnings: 27.50},
	{UserID: "demo_finn", Nickname: "沈亦凡", Email: "demo-finn@example.test", Avatar: unsplash("photo-1507003211169-0a1dd7228f2d", 256, 256), Bio: "深圳校园活动拍摄者，负责短视频、夜市和社团记录。", Location: "中国 广东 深圳", Gender: "男", Education: "本科", Major: "影视制作", MBTI: "ESFP", Interests: []string{"视频", "创作者", "运动"}, DaysAgo: 21, Points: 390, Wallet: 6.40, Earnings: 34.70},
}

var demoPostSeeds = []demoPostSeed{
	{Key: "library-light", Author: "demo_alice", Title: "图书馆清晨的第一束光", LegacyTitle: "Morning light in the library", Category: "photography", Type: demoPostTypeImage, Tags: []string{"演示", "校园", "摄影"}, Images: []string{unsplash("photo-1498243691581-b145c3f54a5a", 1200, 900), unsplash("photo-1524995997946-a1c2e315a42f", 1200, 900)}, Content: "上海高校图书馆的清晨照片，用来检查信息流、图片预览和评论区的中文排版。", ViewCount: 420, DaysAgo: 1, HoursOffset: -4, QualityLevel: "featured"},
	{Key: "release-checklist", Author: "demo_ben", Title: "周五发布前的检查清单", LegacyTitle: "Release checklist for a calm Friday", Category: "technology", Type: demoPostTypeImage, Tags: []string{"演示", "科技", "测试"}, Images: []string{unsplash("photo-1516321318423-f06f85e504b3", 1200, 900)}, Attachment: &demoAttachmentSeed{URL: "https://example.com/demo/release-checklist.pdf", Filename: "发布检查清单.pdf", Filesize: 245760}, Content: "一份给国内校园项目使用的发布清单，包含日志、回滚、冒烟测试和上线后观察。", ViewCount: 315, DaysAgo: 2, HoursOffset: -2},
	{Key: "coffee-map", Author: "demo_cora", Title: "校园附近的三家咖啡角", LegacyTitle: "Three coffee corners near campus", Category: "food", Type: demoPostTypeImage, Tags: []string{"演示", "美食", "校园"}, Images: []string{unsplash("photo-1495474472287-4d71bcdd2085", 1200, 900), unsplash("photo-1509042239860-f550ce710b93", 1200, 900)}, Content: "成都校园周边三家适合自习和聊天的小店，用来测试分类、搜索和收藏流程。", ViewCount: 536, DaysAgo: 3, HoursOffset: -1},
	{Key: "weekend-route", Author: "demo_drew", Title: "杭州周末可步行路线", LegacyTitle: "A walkable weekend route", Category: "travel", Type: demoPostTypeImage, Tags: []string{"演示", "旅行", "摄影"}, Images: []string{unsplash("photo-1500530855697-b586d89ba3ee", 1200, 900)}, Content: "从西湖边到老街巷的一条慢走路线，适合测试个人主页浏览历史和卡片缩略图。", ViewCount: 284, DaysAgo: 4, HoursOffset: -3},
	{Key: "study-reset", Author: "demo_elin", Title: "自习间隙的五分钟重启", LegacyTitle: "Tiny reset rituals between study blocks", Category: "lifestyle", Type: demoPostTypeImage, Tags: []string{"演示", "生活", "校园"}, Images: []string{unsplash("photo-1517842645767-c639042777db", 1200, 900)}, Content: "南京同学常用的五个短休息方法，让长自习日更容易坚持，也方便测试中文长文换行。", ViewCount: 198, DaysAgo: 5, HoursOffset: -5},
	{Key: "campus-video", Author: "demo_finn", Title: "深圳夜市短视频测试", LegacyTitle: "Campus night market video test", Category: "video", Type: demoPostTypeVideo, Tags: []string{"演示", "视频", "创作者"}, Images: []string{unsplash("photo-1517457373958-b7bdd4587205", 1200, 900)}, VideoURL: "https://storage.googleapis.com/shaka-demo-assets/angel-one-hls/hls.m3u8", VideoCover: unsplash("photo-1517457373958-b7bdd4587205", 1200, 900), Content: "深圳校园夜市的短视频样例，用于检查播放器、封面和游客访问权限。", ViewCount: 642, DaysAgo: 1, HoursOffset: -8},
	{Key: "paid-pack", Author: "demo_alice", Title: "西湖风景付费图包样例", LegacyTitle: "Protected photo pack sample", Category: "photography", Type: demoPostTypeImage, Tags: []string{"演示", "摄影", "创作者"}, Images: []string{unsplash("photo-1500534314209-a25ddb2bd429", 1200, 900), unsplash("photo-1500530855697-b586d89ba3ee", 1200, 900), unsplash("photo-1506744038136-46273834b3fb", 1200, 900)}, Payment: &demoPaymentSeed{Price: 18, PaymentType: "image_pack", PaymentMethod: "points", FreePreviewCount: 1}, Content: "第一张图免费预览，其余图片模拟受保护的付费图包，适合测试积分购买流程。", ViewCount: 712, DaysAgo: 6, HoursOffset: -6, QualityLevel: "premium"},
	{Key: "desk-setup", Author: "demo_ben", Title: "用了两周的桌面整理心得", LegacyTitle: "Desk setup notes after two weeks", Category: "technology", Type: demoPostTypeImage, Tags: []string{"演示", "科技", "生活"}, Images: []string{unsplash("photo-1497366754035-f200968a6e72", 1200, 900)}, Content: "一篇偏实用的中文科技笔记，用来测试搜索排序、推荐和标签详情页。", ViewCount: 259, DaysAgo: 7, HoursOffset: -7},
	{Key: "meal-prep", Author: "demo_cora", Title: "考试周简单备餐记录", LegacyTitle: "Simple meal prep for exam week", Category: "food", Type: demoPostTypeImage, Tags: []string{"演示", "美食", "生活"}, Images: []string{unsplash("photo-1546069901-ba9599a7e63c", 1200, 900)}, Content: "适合国内校园考试周的轻量备餐记录，可用于热门信息流和美食分类测试。", ViewCount: 447, DaysAgo: 8, HoursOffset: -2},
	{Key: "city-rooftop", Author: "demo_drew", Title: "城市天台的蓝调时刻", LegacyTitle: "City rooftop at blue hour", Category: "travel", Type: demoPostTypeImage, Tags: []string{"演示", "旅行", "摄影"}, Images: []string{unsplash("photo-1494526585095-c41746248156", 1200, 900)}, Content: "傍晚的城市天台照片，有足够互动量，便于在热门排序里出现。", ViewCount: 388, DaysAgo: 9, HoursOffset: -4},
	{Key: "playlist", Author: "demo_elin", Title: "专注下午的中文歌单", LegacyTitle: "Playlist for a focused afternoon", Category: "lifestyle", Type: demoPostTypeImage, Tags: []string{"演示", "生活", "创作者"}, Images: []string{unsplash("photo-1516280440614-37939bbacd81", 1200, 900)}, Content: "一条轻量生活内容，用来检查个人主页、评论和收藏流程。", ViewCount: 223, DaysAgo: 10, HoursOffset: -1},
	{Key: "event-recap", Author: "demo_finn", Title: "校园活动视频复盘", LegacyTitle: "Event recap with creator notes", Category: "video", Type: demoPostTypeVideo, Tags: []string{"演示", "视频", "校园"}, Images: []string{unsplash("photo-1523580494863-6f3031224c94", 1200, 900)}, VideoURL: "https://storage.googleapis.com/shaka-demo-assets/angel-one-widevine-hls/hls.m3u8", VideoCover: unsplash("photo-1523580494863-6f3031224c94", 1200, 900), Content: "第二条中文视频样例，用于列表、播放回退和创作者数据测试。", ViewCount: 501, DaysAgo: 11, HoursOffset: -9},
}

func unsplash(id string, width int, height int) string {
	return fmt.Sprintf("https://images.unsplash.com/%s?auto=format&fit=crop&w=%d&h=%d&q=75", id, width, height)
}
