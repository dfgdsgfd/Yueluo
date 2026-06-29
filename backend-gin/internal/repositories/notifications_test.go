package repositories

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestActivitiesMissingSystemNotificationsTableReturnsEmpty(t *testing.T) {
	db := newNotificationsSchemaTestDB(t, "activities-missing-table")

	total, rows, err := NewNotificationsRepository(db).Activities(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("activities with missing table error = %v", err)
	}
	if total != 0 || len(rows) != 0 {
		t.Fatalf("activities with missing table total=%d len=%d, want empty", total, len(rows))
	}
}

func TestActivitiesMissingOptionalColumnsReturnsEmpty(t *testing.T) {
	db := newNotificationsSchemaTestDB(t, "activities-missing-columns")
	if err := db.Exec(`
        CREATE TABLE system_notifications (
            id INTEGER PRIMARY KEY,
            title TEXT NOT NULL,
            content TEXT NOT NULL,
            type TEXT NOT NULL,
            is_active BOOLEAN NOT NULL,
            start_time DATETIME,
            end_time DATETIME,
            created_at DATETIME NOT NULL
        )
    `).Error; err != nil {
		t.Fatalf("create legacy system_notifications table: %v", err)
	}
	now := time.Now()
	if err := db.Exec(
		"INSERT INTO system_notifications (id, title, content, type, is_active, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		1, "Legacy activity", "Legacy content", "activity", true, now,
	).Error; err != nil {
		t.Fatalf("insert legacy activity: %v", err)
	}

	total, rows, err := NewNotificationsRepository(db).Activities(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("activities with legacy columns error = %v", err)
	}
	if total != 0 || len(rows) != 0 {
		t.Fatalf("activities with legacy columns total=%d len=%d, want empty", total, len(rows))
	}
}

func newNotificationsSchemaTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", name)), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}
