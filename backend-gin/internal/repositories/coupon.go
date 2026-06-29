package repositories

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

type CouponRepository struct {
	db *gorm.DB
}

type UserCouponBundle struct {
	UserCoupon domain.UserCoupon
	Coupon     *domain.Coupon
	User       *domain.User
}

type CouponValidation struct {
	Valid       bool
	Message     string
	Discount    float64
	FinalAmount float64
	Coupon      *domain.Coupon
	UserCoupon  *domain.UserCoupon
}

type CouponIssueSkipped struct {
	UserID string `json:"user_id"`
	Reason string `json:"reason"`
}

type CouponStats struct {
	TotalCoupons  int64
	ActiveCoupons int64
	TotalIssued   int64
	TotalUsed     int64
}

func NewCouponRepository(db *gorm.DB) CouponRepository {
	return CouponRepository{db: db}
}

func (r CouponRepository) MyCoupons(ctx context.Context, userID int64, status string) ([]UserCouponBundle, error) {
	now := time.Now()
	sub := r.db.WithContext(ctx).Model(&domain.Coupon{}).Select("id").Where("end_time < ?", now)
	if err := r.db.WithContext(ctx).Model(&domain.UserCoupon{}).
		Where("user_id = ? AND status = ? AND coupon_id IN (?)", userID, "unused", sub).
		Update("status", "expired").Error; err != nil {
		return nil, err
	}

	query := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var userCoupons []domain.UserCoupon
	if err := query.Order("created_at DESC").Find(&userCoupons).Error; err != nil {
		return nil, err
	}
	coupons, err := r.couponsByID(ctx, userCouponCouponIDs(userCoupons))
	if err != nil {
		return nil, err
	}
	out := make([]UserCouponBundle, 0, len(userCoupons))
	for _, userCoupon := range userCoupons {
		out = append(out, UserCouponBundle{UserCoupon: userCoupon, Coupon: coupons[userCoupon.CouponID]})
	}
	return out, nil
}

func (r CouponRepository) Claim(ctx context.Context, userID int64, code string) (*domain.UserCoupon, *domain.Coupon, error) {
	var coupon domain.Coupon
	if err := r.db.WithContext(ctx).Where("code = ?", strings.ToUpper(strings.TrimSpace(code))).First(&coupon).Error; err != nil {
		return nil, nil, err
	}
	if err := validateCouponWindow(coupon, 0, false); err != nil {
		return nil, &coupon, err
	}
	var existing domain.UserCoupon
	err := r.db.WithContext(ctx).Where("user_id = ? AND coupon_id = ?", userID, coupon.ID).First(&existing).Error
	if err == nil {
		return nil, &coupon, ErrCouponAlreadyClaimed
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, err
	}
	if coupon.TotalCount != -1 && coupon.UsedCount >= coupon.TotalCount {
		return nil, &coupon, ErrCouponOutOfStock
	}
	userCoupon := domain.UserCoupon{UserID: userID, CouponID: coupon.ID, Status: "unused"}
	if err := r.db.WithContext(ctx).Create(&userCoupon).Error; err != nil {
		return nil, nil, err
	}
	return &userCoupon, &coupon, nil
}

func (r CouponRepository) Validate(ctx context.Context, userID, userCouponID int64, orderAmount float64) (*CouponValidation, error) {
	bundle, err := r.userCouponBundle(ctx, userID, userCouponID)
	if err != nil {
		return nil, err
	}
	validation := validateUserCoupon(bundle, orderAmount)
	return &validation, nil
}

func (r CouponRepository) Use(ctx context.Context, userID, userCouponID int64, orderAmount float64) (*CouponValidation, error) {
	validation, err := r.Validate(ctx, userID, userCouponID, orderAmount)
	if err != nil {
		return nil, err
	}
	if !validation.Valid {
		return validation, nil
	}
	now := time.Now()
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&domain.UserCoupon{}).Where("id = ?", userCouponID).Updates(map[string]any{"status": "used", "used_at": now}).Error; err != nil {
			return err
		}
		return tx.Model(&domain.Coupon{}).Where("id = ?", validation.Coupon.ID).UpdateColumn("used_count", gorm.Expr("used_count + ?", 1)).Error
	})
	return validation, err
}

func (r CouponRepository) AdminList(ctx context.Context, page, limit int, keyword string) (int64, []domain.Coupon, map[int64]int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.Coupon{})
	if strings.TrimSpace(keyword) != "" {
		like := "%" + strings.TrimSpace(keyword) + "%"
		query = query.Where("name LIKE ? OR code LIKE ?", like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, nil, err
	}
	var coupons []domain.Coupon
	if err := query.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&coupons).Error; err != nil {
		return 0, nil, nil, err
	}
	counts, err := r.issuedCounts(ctx, couponIDs(coupons))
	return total, coupons, counts, err
}

func (r CouponRepository) Create(ctx context.Context, coupon domain.Coupon) (*domain.Coupon, error) {
	if coupon.Code != nil {
		exists, err := r.codeExists(ctx, *coupon.Code, nil)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrCouponCodeExists
		}
	}
	if err := r.db.WithContext(ctx).Create(&coupon).Error; err != nil {
		return nil, err
	}
	return &coupon, nil
}

func (r CouponRepository) Update(ctx context.Context, id int64, updates map[string]any, code *string) error {
	var existing domain.Coupon
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&existing).Error; err != nil {
		return err
	}
	if code != nil {
		exists, err := r.codeExists(ctx, *code, &id)
		if err != nil {
			return err
		}
		if exists {
			return ErrCouponCodeExists
		}
	}
	return r.db.WithContext(ctx).Model(&domain.Coupon{}).Where("id = ?", id).Updates(updates).Error
}

func (r CouponRepository) Delete(ctx context.Context, id int64) error {
	var existing domain.Coupon
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&existing).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Delete(&existing).Error
}

func (r CouponRepository) ActiveUsers(ctx context.Context, requireOAuth bool) ([]domain.User, error) {
	query := r.db.WithContext(ctx).Where("is_active = ?", true)
	if requireOAuth {
		query = query.Where("oauth2_id IS NOT NULL")
	}
	var users []domain.User
	err := query.Select("id", "oauth2_id").Find(&users).Error
	return users, err
}

func (r CouponRepository) Issue(ctx context.Context, couponID, adminID int64, targetType string, userIDs []int64) ([]string, []CouponIssueSkipped, error) {
	var coupon domain.Coupon
	if err := r.db.WithContext(ctx).Where("id = ?", couponID).First(&coupon).Error; err != nil {
		return nil, nil, err
	}
	issued := []string{}
	skipped := []CouponIssueSkipped{}
	for _, userID := range uniqueInt64(userIDs) {
		if targetType == "users" {
			var count int64
			if err := r.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", userID).Count(&count).Error; err != nil {
				return nil, nil, err
			}
			if count == 0 {
				skipped = append(skipped, CouponIssueSkipped{UserID: idString(userID), Reason: "\u7528\u6237\u4e0d\u5b58\u5728"})
				continue
			}
		}
		var existing int64
		if err := r.db.WithContext(ctx).Model(&domain.UserCoupon{}).Where("user_id = ? AND coupon_id = ?", userID, couponID).Count(&existing).Error; err != nil {
			return nil, nil, err
		}
		if existing > 0 {
			skipped = append(skipped, CouponIssueSkipped{UserID: idString(userID), Reason: "\u5df2\u53d1\u653e"})
			continue
		}
		userCoupon := domain.UserCoupon{UserID: userID, CouponID: couponID, Status: "unused", IssuedBy: &adminID}
		if err := r.db.WithContext(ctx).Create(&userCoupon).Error; err != nil {
			return nil, nil, err
		}
		issued = append(issued, idString(userID))
	}
	return issued, skipped, nil
}

func (r CouponRepository) Usages(ctx context.Context, couponID int64, page, limit int, status string) (int64, []UserCouponBundle, error) {
	query := r.db.WithContext(ctx).Model(&domain.UserCoupon{}).Where("coupon_id = ?", couponID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	var usages []domain.UserCoupon
	if err := query.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&usages).Error; err != nil {
		return 0, nil, err
	}
	users, err := r.usersByID(ctx, userCouponUserIDs(usages))
	if err != nil {
		return 0, nil, err
	}
	bundles := make([]UserCouponBundle, 0, len(usages))
	for _, usage := range usages {
		bundles = append(bundles, UserCouponBundle{UserCoupon: usage, User: users[usage.UserID]})
	}
	return total, bundles, nil
}

func (r CouponRepository) Stats(ctx context.Context) (*CouponStats, error) {
	var stats CouponStats
	if err := r.db.WithContext(ctx).Model(&domain.Coupon{}).Count(&stats.TotalCoupons).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&domain.Coupon{}).Where("is_active = ?", true).Count(&stats.ActiveCoupons).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&domain.UserCoupon{}).Count(&stats.TotalIssued).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&domain.UserCoupon{}).Where("status = ?", "used").Count(&stats.TotalUsed).Error; err != nil {
		return nil, err
	}
	return &stats, nil
}

var (
	ErrCouponInactive       = errors.New("coupon inactive")
	ErrCouponNotStarted     = errors.New("coupon not started")
	ErrCouponExpired        = errors.New("coupon expired")
	ErrCouponAlreadyClaimed = errors.New("coupon already claimed")
	ErrCouponOutOfStock     = errors.New("coupon out of stock")
	ErrCouponWrongOwner     = errors.New("coupon wrong owner")
	ErrCouponUsed           = errors.New("coupon used")
	ErrCouponMinOrder       = errors.New("coupon min order")
	ErrCouponCodeExists     = errors.New("coupon code exists")
)

func (r CouponRepository) userCouponBundle(ctx context.Context, userID, userCouponID int64) (UserCouponBundle, error) {
	var userCoupon domain.UserCoupon
	if err := r.db.WithContext(ctx).Where("id = ?", userCouponID).First(&userCoupon).Error; err != nil {
		return UserCouponBundle{}, err
	}
	if userCoupon.UserID != userID {
		return UserCouponBundle{}, ErrCouponWrongOwner
	}
	var coupon domain.Coupon
	if err := r.db.WithContext(ctx).Where("id = ?", userCoupon.CouponID).First(&coupon).Error; err != nil {
		return UserCouponBundle{}, err
	}
	return UserCouponBundle{UserCoupon: userCoupon, Coupon: &coupon}, nil
}

func validateUserCoupon(bundle UserCouponBundle, orderAmount float64) CouponValidation {
	if bundle.Coupon == nil {
		return CouponValidation{Valid: false, Message: "\u4f18\u60e0\u5238\u4e0d\u5b58\u5728"}
	}
	if bundle.UserCoupon.Status == "used" {
		return CouponValidation{Valid: false, Message: "\u8be5\u4f18\u60e0\u5238\u5df2\u4f7f\u7528"}
	}
	if bundle.UserCoupon.Status == "expired" {
		return CouponValidation{Valid: false, Message: "\u8be5\u4f18\u60e0\u5238\u5df2\u8fc7\u671f"}
	}
	if err := validateCouponWindow(*bundle.Coupon, orderAmount, true); err != nil {
		return CouponValidation{Valid: false, Message: couponErrorMessage(err, *bundle.Coupon)}
	}
	discount := calculateDiscount(*bundle.Coupon, orderAmount)
	return CouponValidation{
		Valid:       true,
		Discount:    discount,
		FinalAmount: maxFloat(0, orderAmount-discount),
		Coupon:      bundle.Coupon,
		UserCoupon:  &bundle.UserCoupon,
	}
}

func validateCouponWindow(coupon domain.Coupon, orderAmount float64, checkMin bool) error {
	now := time.Now()
	if !coupon.IsActive {
		return ErrCouponInactive
	}
	if now.Before(coupon.StartTime) {
		return ErrCouponNotStarted
	}
	if now.After(coupon.EndTime) {
		return ErrCouponExpired
	}
	if checkMin && coupon.MinOrder > 0 && orderAmount < coupon.MinOrder {
		return ErrCouponMinOrder
	}
	return nil
}

func calculateDiscount(coupon domain.Coupon, orderAmount float64) float64 {
	discount := 0.0
	if coupon.Type == "amount" {
		discount = coupon.Value
	} else if coupon.Type == "percent" {
		discount = orderAmount * (coupon.Value / 100)
		if coupon.MaxDiscount != nil && discount > *coupon.MaxDiscount {
			discount = *coupon.MaxDiscount
		}
	}
	return mathRound2(discount)
}

func couponErrorMessage(err error, coupon domain.Coupon) string {
	switch err {
	case ErrCouponInactive:
		return "\u8be5\u4f18\u60e0\u5238\u5df2\u505c\u7528"
	case ErrCouponNotStarted:
		return "\u8be5\u4f18\u60e0\u5238\u5c1a\u672a\u5f00\u59cb"
	case ErrCouponExpired:
		return "\u8be5\u4f18\u60e0\u5238\u5df2\u8fc7\u671f"
	case ErrCouponMinOrder:
		return "\u8ba2\u5355\u91d1\u989d\u9700\u6ee1 " + formatMoney(coupon.MinOrder) + " \u5143\u624d\u53ef\u4f7f\u7528"
	default:
		return "\u4f18\u60e0\u5238\u4e0d\u53ef\u7528"
	}
}

func (r CouponRepository) couponsByID(ctx context.Context, ids []int64) (map[int64]*domain.Coupon, error) {
	out := map[int64]*domain.Coupon{}
	if len(ids) == 0 {
		return out, nil
	}
	var coupons []domain.Coupon
	if err := r.db.WithContext(ctx).Where("id IN ?", uniqueInt64(ids)).Find(&coupons).Error; err != nil {
		return nil, err
	}
	for i := range coupons {
		out[coupons[i].ID] = &coupons[i]
	}
	return out, nil
}

func (r CouponRepository) usersByID(ctx context.Context, ids []int64) (map[int64]*domain.User, error) {
	out := map[int64]*domain.User{}
	if len(ids) == 0 {
		return out, nil
	}
	var users []domain.User
	if err := r.db.WithContext(ctx).Where("id IN ?", uniqueInt64(ids)).Select("id", "user_id", "nickname", "avatar").Find(&users).Error; err != nil {
		return nil, err
	}
	for i := range users {
		out[users[i].ID] = &users[i]
	}
	return out, nil
}

func (r CouponRepository) issuedCounts(ctx context.Context, ids []int64) (map[int64]int64, error) {
	out := map[int64]int64{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		CouponID int64
		Count    int64
	}
	err := r.db.WithContext(ctx).Model(&domain.UserCoupon{}).Select("coupon_id, COUNT(*) AS count").Where("coupon_id IN ?", ids).Group("coupon_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.CouponID] = row.Count
	}
	return out, nil
}

func (r CouponRepository) codeExists(ctx context.Context, code string, excludeID *int64) (bool, error) {
	query := r.db.WithContext(ctx).Model(&domain.Coupon{}).Where("code = ?", code)
	if excludeID != nil {
		query = query.Where("id <> ?", *excludeID)
	}
	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}

func couponIDs(coupons []domain.Coupon) []int64 {
	out := make([]int64, 0, len(coupons))
	for _, coupon := range coupons {
		out = append(out, coupon.ID)
	}
	return out
}

func userCouponCouponIDs(rows []domain.UserCoupon) []int64 {
	out := make([]int64, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.CouponID)
	}
	return out
}

func userCouponUserIDs(rows []domain.UserCoupon) []int64 {
	out := make([]int64, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.UserID)
	}
	return out
}

func idString(id int64) string {
	return strconv.FormatInt(id, 10)
}

func formatMoney(value float64) string {
	return fmt.Sprintf("%.2f", value)
}

func mathRound2(value float64) float64 {
	return math.Round(value*100) / 100
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
