package markdownmigration

import (
	"context"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestRunDryRunApplyBackupAndIdempotent(t *testing.T) {
	db := openTestDB(t)
	seedMigrationTables(t, db)

	ctx := context.Background()
	dry, err := Run(ctx, db, Options{Sample: 2, Tables: []string{"posts.content", "comments.content"}})
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if dry.Apply {
		t.Fatalf("dry run should not apply")
	}
	if dry.Targets[0].Changed == 0 || dry.Targets[0].Updated != 0 {
		t.Fatalf("dry run target = %#v", dry.Targets[0])
	}
	var content string
	if err := db.Raw(`SELECT content FROM posts WHERE id = 1`).Scan(&content).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "style=") {
		t.Fatalf("dry run unexpectedly changed row: %s", content)
	}

	applied, err := Run(ctx, db, Options{Apply: true, Backup: true, Sample: 2, Tables: []string{"posts.content", "comments.content"}})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if applied.Targets[0].Updated == 0 || applied.Targets[0].BackedUp == 0 {
		t.Fatalf("apply target = %#v", applied.Targets[0])
	}
	if err := db.Raw(`SELECT content FROM posts WHERE id = 1`).Scan(&content).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "## 标题") || strings.Contains(content, "<h2") || strings.Contains(content, "style=") {
		t.Fatalf("post content not migrated to markdown: %s", content)
	}
	var backupCount int64
	if err := db.Table("markdown_migration_backups").Count(&backupCount).Error; err != nil {
		t.Fatal(err)
	}
	if backupCount == 0 {
		t.Fatalf("expected backup rows")
	}

	again, err := Run(ctx, db, Options{Apply: true, Backup: true, Tables: []string{"posts.content", "comments.content"}})
	if err != nil {
		t.Fatalf("idempotent apply: %v", err)
	}
	for _, target := range again.Targets {
		if target.Changed != 0 || target.Updated != 0 {
			t.Fatalf("expected idempotent target, got %#v", target)
		}
	}
}

func TestRunDefaultTargets(t *testing.T) {
	db := openTestDB(t)
	seedMigrationTables(t, db)

	result, err := Run(context.Background(), db, Options{Apply: true, Backup: true})
	if err != nil {
		t.Fatalf("run default targets: %v", err)
	}
	if len(result.Targets) != len(DefaultTargets()) {
		t.Fatalf("targets = %d, want %d", len(result.Targets), len(DefaultTargets()))
	}
	var bio string
	if err := db.Raw(`SELECT bio FROM users WHERE id = 1`).Scan(&bio).Error; err != nil {
		t.Fatal(err)
	}
	if strings.Contains(bio, "<") || !strings.Contains(bio, "**加粗**") {
		t.Fatalf("bio not migrated: %s", bio)
	}
}

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func seedMigrationTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	statements := []string{
		`CREATE TABLE posts (id INTEGER PRIMARY KEY, content TEXT NOT NULL)`,
		`CREATE TABLE comments (id INTEGER PRIMARY KEY, content TEXT NOT NULL)`,
		`CREATE TABLE im_messages (id INTEGER PRIMARY KEY, content TEXT NOT NULL)`,
		`CREATE TABLE users (id INTEGER PRIMARY KEY, bio TEXT)`,
		`CREATE TABLE feedback (id INTEGER PRIMARY KEY, content TEXT NOT NULL)`,
		`CREATE TABLE reports (id INTEGER PRIMARY KEY, description TEXT)`,
		`CREATE TABLE audit (id INTEGER PRIMARY KEY, content TEXT NOT NULL)`,
		`INSERT INTO posts (id, content) VALUES (1, '<h2 style="color:red">标题</h2><p onclick="x()">正文 <strong>加粗</strong><script>alert(1)</script></p>')`,
		`INSERT INTO comments (id, content) VALUES (1, '<p>评论 <a href="javascript:alert(1)">bad</a></p>')`,
		`INSERT INTO im_messages (id, content) VALUES (1, 'plain message')`,
		`INSERT INTO users (id, bio) VALUES (1, '<p>简介 <strong>加粗</strong></p>')`,
		`INSERT INTO feedback (id, content) VALUES (1, '<p>反馈 <em>斜体</em></p>')`,
		`INSERT INTO reports (id, description) VALUES (1, '<p>举报 <img src="data:text/html;base64,xx"></p>')`,
		`INSERT INTO audit (id, content) VALUES (1, '<p>认证 <script>x()</script></p>')`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("exec %q: %v", statement, err)
		}
	}
}
