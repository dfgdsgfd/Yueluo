package storage

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/security"
)

const (
	DefaultAdminUsername        = "admin"
	DefaultAdminInitialPassword = "123456"
	generatedUserPasswordLength = 16
)

type InitialAdminResult struct {
	Created        bool
	PasswordFilled int64
}

func EnsureInitialAdmin(ctx context.Context, db *gorm.DB, username string, password string) (InitialAdminResult, error) {
	if db == nil {
		return InitialAdminResult{}, nil
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return InitialAdminResult{}, fmt.Errorf("admin username is required")
	}
	if strings.TrimSpace(password) == "" {
		return InitialAdminResult{}, fmt.Errorf("admin password is required")
	}
	passwordHash, err := security.HashPassword(password)
	if err != nil {
		return InitialAdminResult{}, err
	}
	var count int64
	if err := db.WithContext(ctx).Model(&domain.Admin{}).Count(&count).Error; err != nil {
		return InitialAdminResult{}, err
	}
	if count == 0 {
		row := domain.Admin{Username: username, Password: passwordHash}
		if err := db.WithContext(ctx).Create(&row).Error; err != nil {
			return InitialAdminResult{}, err
		}
		return InitialAdminResult{Created: true}, nil
	}
	result := db.WithContext(ctx).
		Model(&domain.Admin{}).
		Where("password = ''").
		Update("password", passwordHash)
	return InitialAdminResult{PasswordFilled: result.RowsAffected}, result.Error
}

func ResetAdminPassword(ctx context.Context, db *gorm.DB, username string, password string) (int64, error) {
	if db == nil {
		return 0, nil
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return 0, fmt.Errorf("admin username is required")
	}
	if strings.TrimSpace(password) == "" {
		return 0, fmt.Errorf("admin password is required")
	}
	passwordHash, err := security.HashPassword(password)
	if err != nil {
		return 0, err
	}
	result := db.WithContext(ctx).
		Model(&domain.Admin{}).
		Where("username = ?", username).
		Update("password", passwordHash)
	return result.RowsAffected, result.Error
}

func RandomizeLegacyUserPasswords(ctx context.Context, db *gorm.DB) (int64, error) {
	if db == nil {
		return 0, nil
	}
	var users []domain.User
	if err := db.WithContext(ctx).
		Where("password IS NOT NULL AND password <> '' AND password NOT LIKE ?", "$argon2id$%").
		Select("id").
		Find(&users).Error; err != nil {
		return 0, err
	}
	var updated int64
	for _, user := range users {
		password, err := security.RandomPassword(generatedUserPasswordLength)
		if err != nil {
			return updated, err
		}
		passwordHash, err := security.HashPassword(password)
		if err != nil {
			return updated, err
		}
		result := db.WithContext(ctx).
			Model(&domain.User{}).
			Where("id = ? AND password IS NOT NULL AND password <> '' AND password NOT LIKE ?", user.ID, "$argon2id$%").
			Update("password", passwordHash)
		if result.Error != nil {
			return updated, result.Error
		}
		updated += result.RowsAffected
	}
	return updated, nil
}
