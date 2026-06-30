package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/storage"
)

func main() {
	password := flag.String("password", storage.DefaultDemoPassword, "所有演示账号使用的登录密码")
	userLimit := flag.Int("users", 0, "最多填充多少个演示用户；0 表示全部")
	postLimit := flag.Int("posts", 0, "最多填充多少篇演示帖子；0 表示全部")
	autoMigrate := flag.Bool("auto-migrate", true, "填充前执行数据库自动迁移")
	timeout := flag.Duration("timeout", 30*time.Second, "数据库填充超时时间")
	flag.Parse()

	cfg := config.Load()
	cfg.Database.AutoMigrate = *autoMigrate
	db, err := storage.OpenDatabase(cfg.Database)
	if err != nil {
		exitf("打开数据库失败：%v", err)
	}
	if db == nil {
		exitf("缺少数据库连接，请设置 DATABASE_URL 或 DB_* 环境变量")
	}
	sqlDB, err := db.DB()
	if err == nil {
		defer sqlDB.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	result, err := storage.SeedDemoData(ctx, db, storage.DemoSeedOptions{
		Password:  *password,
		UserLimit: *userLimit,
		PostLimit: *postLimit,
	})
	if err != nil {
		exitf("填充演示数据失败：%v", err)
	}
	printResult(result)
}

func printResult(result storage.DemoSeedResult) {
	fmt.Println("中文演示数据填充完成。")
	fmt.Printf("目标数据：用户=%d 分类=%d 标签=%d 帖子=%d\n", result.UserCount, result.CategoryCount, result.TagCount, result.PostCount)
	fmt.Printf("新增或补齐：用户=%d 帖子=%d 图片=%d 视频=%d 附件=%d 付费设置=%d 评论=%d 点赞=%d 收藏=%d 关注=%d 通知=%d\n",
		result.UsersCreated,
		result.PostsCreated,
		result.PostImagesCreated,
		result.PostVideosCreated,
		result.PostAttachmentsCreated,
		result.PostPaymentSettingsCreated,
		result.CommentsCreated,
		result.LikesCreated,
		result.CollectionsCreated,
		result.FollowsCreated,
		result.NotificationsCreated,
	)
	fmt.Printf("配套数据：钱包=%d 积分=%d 创作者=%d 礼品卡=%d 系统=%d 消息=%d 历史=%d 帖子标签=%d 分类=%d 标签=%d\n",
		result.WalletRowsCreated,
		result.PointRowsCreated,
		result.CreatorRowsCreated,
		result.GiftCardRowsCreated,
		result.SystemRowsCreated,
		result.IMRowsCreated,
		result.HistoryRowsCreated,
		result.PostTagsCreated,
		result.CategoriesCreated,
		result.TagsCreated,
	)
	fmt.Printf("演示密码：%s\n", result.Password)
	fmt.Println("可登录账号：")
	for _, account := range result.LoginAccounts {
		fmt.Printf("- %s 或 %s\n", account.Account, account.Email)
	}
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
