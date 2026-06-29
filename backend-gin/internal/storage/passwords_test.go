package storage

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/security"
)

func TestResetAdminPasswordUpdatesRequestedAdminOnly(t *testing.T) {
	db := openPasswordSQLite(t)
	if err := db.AutoMigrate(&domain.Admin{}); err != nil {
		t.Fatal(err)
	}
	oldHash, err := security.HashPassword("old")
	if err != nil {
		t.Fatal(err)
	}
	otherHash, err := security.HashPassword("other")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.Admin{Username: "admin", Password: oldHash}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.Admin{Username: "ops", Password: otherHash}).Error; err != nil {
		t.Fatal(err)
	}

	affected, err := ResetAdminPassword(context.Background(), db, "admin", "custom-admin-pass")
	if err != nil {
		t.Fatalf("ResetAdminPassword() error = %v", err)
	}
	if affected != 1 {
		t.Fatalf("affected = %d, want 1", affected)
	}
	var admin domain.Admin
	if err := db.Where("username = ?", "admin").First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if !security.VerifyPassword("custom-admin-pass", admin.Password) {
		t.Fatalf("admin password was not reset to requested value")
	}
	var ops domain.Admin
	if err := db.Where("username = ?", "ops").First(&ops).Error; err != nil {
		t.Fatal(err)
	}
	if !security.VerifyPassword("other", ops.Password) {
		t.Fatalf("unrequested admin password was changed")
	}
}

func TestEnsureInitialAdminCreatesDefaultAdminWhenMissing(t *testing.T) {
	db := openPasswordSQLite(t)
	if err := db.AutoMigrate(&domain.Admin{}); err != nil {
		t.Fatal(err)
	}

	result, err := EnsureInitialAdmin(context.Background(), db, DefaultAdminUsername, DefaultAdminInitialPassword)
	if err != nil {
		t.Fatalf("EnsureInitialAdmin() error = %v", err)
	}
	if !result.Created || result.PasswordFilled != 0 {
		t.Fatalf("result = %+v, want created only", result)
	}
	var admin domain.Admin
	if err := db.Where("username = ?", DefaultAdminUsername).First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if !security.VerifyPassword(DefaultAdminInitialPassword, admin.Password) {
		t.Fatalf("initial admin password does not verify")
	}
}

func TestEnsureInitialAdminFillsEmptyPasswordsWithoutOverwritingExisting(t *testing.T) {
	db := openPasswordSQLite(t)
	if err := db.AutoMigrate(&domain.Admin{}); err != nil {
		t.Fatal(err)
	}
	existingHash, err := security.HashPassword("keep-me")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.Admin{Username: "admin", Password: existingHash}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.Admin{Username: "blank", Password: ""}).Error; err != nil {
		t.Fatal(err)
	}

	result, err := EnsureInitialAdmin(context.Background(), db, DefaultAdminUsername, DefaultAdminInitialPassword)
	if err != nil {
		t.Fatalf("EnsureInitialAdmin() error = %v", err)
	}
	if result.Created || result.PasswordFilled != 1 {
		t.Fatalf("result = %+v, want one filled password", result)
	}
	var admin domain.Admin
	if err := db.Where("username = ?", "admin").First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if admin.Password != existingHash || !security.VerifyPassword("keep-me", admin.Password) {
		t.Fatalf("existing admin password was overwritten")
	}
	var blank domain.Admin
	if err := db.Where("username = ?", "blank").First(&blank).Error; err != nil {
		t.Fatal(err)
	}
	if !security.VerifyPassword(DefaultAdminInitialPassword, blank.Password) {
		t.Fatalf("blank admin password was not initialized")
	}
}

func TestRandomizeLegacyUserPasswordsOnlyUpdatesNonArgon2idPasswords(t *testing.T) {
	db := openPasswordSQLite(t)
	if err := db.AutoMigrate(&domain.User{}); err != nil {
		t.Fatal(err)
	}
	existingHash, err := security.HashPassword("keep")
	if err != nil {
		t.Fatal(err)
	}
	legacyHash := "legacy-sha256-like-password"
	users := []domain.User{
		{ID: 1, UserID: "legacy", Nickname: "Legacy", Password: &legacyHash, IsActive: true},
		{ID: 2, UserID: "argon", Nickname: "Argon", Password: &existingHash, IsActive: true},
		{ID: 3, UserID: "oauth", Nickname: "OAuth", Password: nil, IsActive: true},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}

	affected, err := RandomizeLegacyUserPasswords(context.Background(), db)
	if err != nil {
		t.Fatalf("RandomizeLegacyUserPasswords() error = %v", err)
	}
	if affected != 1 {
		t.Fatalf("affected = %d, want 1", affected)
	}
	var legacy domain.User
	if err := db.Where("user_id = ?", "legacy").First(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	if legacy.Password == nil || !security.IsArgon2idHash(*legacy.Password) {
		t.Fatalf("legacy password = %#v, want Argon2id hash", legacy.Password)
	}
	if security.VerifyPassword("legacy-sha256-like-password", *legacy.Password) {
		t.Fatalf("legacy password should have been replaced by a random value")
	}
	var argon domain.User
	if err := db.Where("user_id = ?", "argon").First(&argon).Error; err != nil {
		t.Fatal(err)
	}
	if argon.Password == nil || *argon.Password != existingHash {
		t.Fatalf("argon2id password changed: %#v", argon.Password)
	}
	var oauth domain.User
	if err := db.Where("user_id = ?", "oauth").First(&oauth).Error; err != nil {
		t.Fatal(err)
	}
	if oauth.Password != nil {
		t.Fatalf("nil OAuth password should remain nil, got %#v", oauth.Password)
	}
}

func openPasswordSQLite(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	return db
}
