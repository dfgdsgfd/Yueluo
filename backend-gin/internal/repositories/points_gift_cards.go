package repositories

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yuem-go/backend-gin/internal/domain"
)

func (r PointsRepository) GiftCardProducts(ctx context.Context, activeOnly bool) ([]GiftCardProductStock, error) {
	query := r.db.WithContext(ctx).Model(&domain.GiftCardProduct{})
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}
	var products []domain.GiftCardProduct
	if err := query.Order("sort_order ASC, id DESC").Find(&products).Error; err != nil {
		return nil, err
	}
	counts, err := r.giftCardStockCounts(ctx, productIDs(products))
	if err != nil {
		return nil, err
	}
	out := make([]GiftCardProductStock, 0, len(products))
	for _, product := range products {
		out = append(out, GiftCardProductStock{Product: product, AvailableStock: counts[product.ID][GiftCardCodeStatusAvailable], RedeemedStock: counts[product.ID][GiftCardCodeStatusRedeemed]})
	}
	return out, nil
}

func (r PointsRepository) SaveGiftCardProduct(ctx context.Context, id int64, input domain.GiftCardProduct) (*domain.GiftCardProduct, error) {
	input.PointsRequired = mathRound2(input.PointsRequired)
	now := time.Now()
	if id <= 0 {
		input.CreatedAt = now
		input.UpdatedAt = &now
		if err := r.db.WithContext(ctx).Create(&input).Error; err != nil {
			return nil, err
		}
		return &input, nil
	}
	updates := map[string]any{
		"name":            input.Name,
		"description":     input.Description,
		"face_value":      input.FaceValue,
		"points_required": input.PointsRequired,
		"is_active":       input.IsActive,
		"sort_order":      input.SortOrder,
		"updated_at":      now,
	}
	if err := r.db.WithContext(ctx).Model(&domain.GiftCardProduct{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	input.ID = id
	return &input, nil
}

func (r PointsRepository) DeleteGiftCardProduct(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&domain.GiftCardProduct{}, id).Error
}

func (r PointsRepository) ImportGiftCardCodes(ctx context.Context, productID int64, text string) (*GiftCardImportResult, error) {
	codes := uniqueCodeLines(text)
	if len(codes) == 0 {
		return nil, ErrGiftCardImportEmpty
	}
	var product domain.GiftCardProduct
	if err := r.db.WithContext(ctx).Where("id = ?", productID).First(&product).Error; err != nil {
		return nil, err
	}
	batch := "gift-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	result := GiftCardImportResult{Batch: batch}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, code := range codes {
			var count int64
			if err := tx.Model(&domain.GiftCardCode{}).Where("product_id = ? AND code = ?", productID, code).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				result.Skipped++
				continue
			}
			row := domain.GiftCardCode{ProductID: productID, Code: code, Status: GiftCardCodeStatusAvailable, ImportBatch: &batch}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			result.Imported++
		}
		return nil
	})
	return &result, err
}

func (r PointsRepository) GiftCardCodes(ctx context.Context, productID int64, status string, page, limit int) (int64, []domain.GiftCardCode, error) {
	query := r.db.WithContext(ctx).Model(&domain.GiftCardCode{}).Where("product_id = ?", productID)
	if strings.TrimSpace(status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(status))
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	var rows []domain.GiftCardCode
	err := query.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&rows).Error
	return total, rows, err
}

func (r PointsRepository) RedeemGiftCard(ctx context.Context, userID, productID int64) (*GiftCardRedemptionBundle, error) {
	var bundle GiftCardRedemptionBundle
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var product domain.GiftCardProduct
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", productID).First(&product).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrGiftCardProductNotFound
			}
			return err
		}
		if !product.IsActive {
			return ErrGiftCardProductInactive
		}
		var code domain.GiftCardCode
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("product_id = ? AND status = ?", productID, GiftCardCodeStatusAvailable).
			Order("id ASC").
			First(&code).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrGiftCardOutOfStock
		}
		if err != nil {
			return err
		}
		points, err := getOrCreateUserPointsTx(ctx, tx, userID)
		if err != nil {
			return err
		}
		cost := mathRound2(product.PointsRequired)
		if points.Points < cost {
			return ErrPointsInsufficient
		}
		newBalance := mathRound2(points.Points - cost)
		now := time.Now()
		if err := tx.Model(&domain.UserPoints{}).Where("user_id = ?", userID).Updates(map[string]any{"points": newBalance, "updated_at": now}).Error; err != nil {
			return err
		}
		reason := "兑换礼品卡: " + product.Name
		if err := tx.Create(&domain.PointsLog{UserID: userID, Amount: -cost, BalanceAfter: newBalance, Type: "gift_card_redeem", Reason: &reason}).Error; err != nil {
			return err
		}
		redemption := domain.GiftCardRedemption{UserID: userID, ProductID: productID, CodeID: code.ID, CodeSnapshot: code.Code, PointsSpent: cost, BalanceAfter: newBalance, Status: RedemptionStatusCompleted}
		if err := tx.Create(&redemption).Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.GiftCardCode{}).Where("id = ?", code.ID).Updates(map[string]any{
			"status":        GiftCardCodeStatusRedeemed,
			"redemption_id": redemption.ID,
			"user_id":       userID,
			"redeemed_at":   now,
			"updated_at":    now,
		}).Error; err != nil {
			return err
		}
		bundle = GiftCardRedemptionBundle{Redemption: redemption, Product: &product}
		return nil
	})
	if err == nil {
		_ = r.createGiftCardRedemptionNotification(ctx, userID, bundle.Redemption.ID)
	}
	return &bundle, err
}

func (r PointsRepository) createGiftCardRedemptionNotification(ctx context.Context, userID, redemptionID int64) error {
	if userID <= 0 || redemptionID <= 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&domain.Notification{
		UserID:   userID,
		SenderID: userID,
		Type:     NotificationTypeGiftCardRedeemed,
		Title:    "notification.giftCardRedeemed.title",
		TargetID: &redemptionID,
	}).Error
}

func (r PointsRepository) Redemptions(ctx context.Context, userID *int64, page, limit int) (int64, []GiftCardRedemptionBundle, error) {
	query := r.db.WithContext(ctx).Model(&domain.GiftCardRedemption{})
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	var rows []domain.GiftCardRedemption
	if err := query.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	products, err := r.productsByID(ctx, redemptionProductIDs(rows))
	if err != nil {
		return 0, nil, err
	}
	out := make([]GiftCardRedemptionBundle, 0, len(rows))
	for _, row := range rows {
		out = append(out, GiftCardRedemptionBundle{Redemption: row, Product: products[row.ProductID]})
	}
	return total, out, nil
}

func (r PointsRepository) Stats(ctx context.Context) (*PointsStats, error) {
	var stats PointsStats
	if err := r.db.WithContext(ctx).Model(&domain.UserPoints{}).Count(&stats.TotalUsers).Error; err != nil {
		return nil, err
	}
	var sum struct{ Total float64 }
	if err := r.db.WithContext(ctx).Model(&domain.UserPoints{}).Select("COALESCE(SUM(points),0) AS total").Scan(&sum).Error; err != nil {
		return nil, err
	}
	stats.TotalPoints = mathRound2(sum.Total)
	todayDate := dateOnly(time.Now())
	var todaySum struct{ Total float64 }
	if err := r.db.WithContext(ctx).Model(&domain.PointsTaskEvent{}).Select("COALESCE(SUM(points),0) AS total").Where("event_date = ?", todayDate).Scan(&todaySum).Error; err != nil {
		return nil, err
	}
	stats.TodayAwarded = mathRound2(todaySum.Total)
	if err := r.db.WithContext(ctx).Model(&domain.GiftCardRedemption{}).Count(&stats.TotalRedeemed).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&domain.GiftCardCode{}).Where("status = ?", GiftCardCodeStatusAvailable).Count(&stats.AvailableCards).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&domain.PointsTaskConfig{}).Where("is_active = ?", true).Count(&stats.ActiveTasks).Error; err != nil {
		return nil, err
	}
	now := time.Now()
	if err := r.db.WithContext(ctx).Model(&domain.UserCreatorBonus{}).Where("is_active = ? AND starts_at <= ? AND (expires_at IS NULL OR expires_at > ?)", true, now, now).Count(&stats.ActiveBonusUsers).Error; err != nil {
		return nil, err
	}
	return &stats, nil
}

func (r PointsRepository) giftCardStockCounts(ctx context.Context, ids []int64) (map[int64]map[string]int64, error) {
	out := map[int64]map[string]int64{}
	for _, id := range ids {
		out[id] = map[string]int64{}
	}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		ProductID int64
		Status    string
		Count     int64
	}
	err := r.db.WithContext(ctx).Model(&domain.GiftCardCode{}).
		Select("product_id, status, COUNT(*) AS count").
		Where("product_id IN ?", ids).
		Group("product_id, status").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if out[row.ProductID] == nil {
			out[row.ProductID] = map[string]int64{}
		}
		out[row.ProductID][row.Status] = row.Count
	}
	return out, nil
}
