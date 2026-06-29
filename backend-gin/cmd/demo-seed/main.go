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
	password := flag.String("password", storage.DefaultDemoPassword, "password for all demo login accounts")
	userLimit := flag.Int("users", 0, "maximum demo users to seed; 0 seeds all")
	postLimit := flag.Int("posts", 0, "maximum demo posts to seed; 0 seeds all")
	autoMigrate := flag.Bool("auto-migrate", true, "run schema auto migration before seeding")
	timeout := flag.Duration("timeout", 30*time.Second, "database seeding timeout")
	flag.Parse()

	cfg := config.Load()
	cfg.Database.AutoMigrate = *autoMigrate
	db, err := storage.OpenDatabase(cfg.Database)
	if err != nil {
		exitf("open database: %v", err)
	}
	if db == nil {
		exitf("database URL is required; set DATABASE_URL or DB_* environment values")
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
		exitf("seed demo data: %v", err)
	}
	printResult(result)
}

func printResult(result storage.DemoSeedResult) {
	fmt.Println("Demo data seed complete.")
	fmt.Printf("Targets: users=%d categories=%d tags=%d posts=%d\n", result.UserCount, result.CategoryCount, result.TagCount, result.PostCount)
	fmt.Printf("Created or completed rows: users=%d posts=%d images=%d videos=%d attachments=%d payments=%d comments=%d likes=%d collections=%d follows=%d notifications=%d\n",
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
	fmt.Printf("Support rows: wallets=%d points=%d creators=%d gift_cards=%d system=%d im=%d history=%d post_tags=%d categories=%d tags=%d\n",
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
	fmt.Printf("Demo password: %s\n", result.Password)
	fmt.Println("Login accounts:")
	for _, account := range result.LoginAccounts {
		fmt.Printf("- %s or %s\n", account.Account, account.Email)
	}
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
