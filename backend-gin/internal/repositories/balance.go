package repositories

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
)

type BalanceRepository struct {
	db *gorm.DB
}

type CreatorRepository struct {
	db  *gorm.DB
	cfg config.CreatorCenterConfig
}

type PurchaseContentInput struct {
	UserID          int64
	PostID          int64
	PaymentMethod   string
	VIPLevel        int
	BalanceAfter    float64
	UserCouponID    *int64
	PlatformFeeRate float64
}

type PurchaseContentResult struct {
	Post           domain.Post
	PaymentSetting domain.PostPaymentSetting
	Already        bool
	Price          float64
	PaidAmount     float64
	DiscountRate   float64
	CouponDiscount float64
	BalanceAfter   float64
	AuthorEarnings float64
	PlatformFee    float64
	PurchaseID     int64
}

type PurchaseIntentReservation struct {
	Intent     domain.ContentPurchaseIntent
	Acquired   bool
	Completed  bool
	PurchaseID int64
}

type PurchaseOrderBundle struct {
	Purchase domain.UserPurchasedContent
	Post     *domain.Post
	Cover    *string
}

type PostPurchaseUserBundle struct {
	Purchase domain.UserPurchasedContent
	Buyer    *domain.User
}

type CreatorOverview struct {
	Earnings      domain.CreatorEarnings
	TodayEarnings float64
	MonthEarnings float64
}

type CreatorTrendData struct {
	Labels    []string
	Views     []int64
	Likes     []int64
	Collects  []int64
	Comments  []int64
	Followers []int64
}

type CreatorStatsData struct {
	Days           int
	GeneratedAt    time.Time
	Fans           map[string]int64
	PostTotals     map[string]int64
	Interactions   map[string]map[string]int64
	LastNDaysLabel string
}

type CreatorEarningsLogBundle struct {
	Log    domain.CreatorEarningsLog
	Buyer  *domain.User
	Source *domain.Post
}

type PaidContentBundle struct {
	Post         domain.Post
	Cover        *string
	Payment      *domain.PostPaymentSetting
	SalesCount   int64
	TotalRevenue float64
}

type ExtendedEarnings struct {
	Enabled   bool
	Rates     config.CreatorEarningsRates
	DailyCap  float64
	Views     CountEarnings
	Likes     CountEarnings
	Collects  CountEarnings
	Comments  CountEarnings
	Followers CountEarnings
	Total     float64
}

type CountEarnings struct {
	Count    int64
	Earnings float64
}

type ClaimExtendedResult struct {
	Success        bool
	Message        string
	AlreadyClaimed bool
	NoEarnings     bool
	Earnings       ExtendedEarnings
	NewBalance     float64
	Details        []string
}

type QualityRewardBundle struct {
	Log  domain.CreatorEarningsLog
	Post *QualityRewardPost
}

type QualityRewardPost struct {
	ID           int64
	Title        string
	Type         int
	QualityLevel string
	Cover        *string
	CreatedAt    time.Time
}

type QualityRewardStats struct {
	QualityLabel string
	Count        int64
	TotalAmount  float64
}

var (
	ErrBalanceDisabled        = errors.New("balance center disabled")
	ErrBalanceUserNotFound    = errors.New("user not found")
	ErrBalanceOAuthMissing    = errors.New("oauth2 missing")
	ErrPurchasePostMissing    = errors.New("post missing")
	ErrPurchaseOwnContent     = errors.New("own content")
	ErrPurchaseNotPaidContent = errors.New("not paid content")
	ErrPurchaseInsufficient   = errors.New("insufficient balance")
	ErrPurchasePaymentMethod  = errors.New("payment method mismatch")
	ErrPurchaseInProgress     = errors.New("purchase in progress")
	ErrCreatorWithdrawClosed  = errors.New("creator withdraw closed")
	ErrCreatorAmountInvalid   = errors.New("creator amount invalid")
	ErrCreatorBelowMinimum    = errors.New("creator below minimum")
	ErrCreatorBalanceLow      = errors.New("creator balance low")
)

func NewBalanceRepository(db *gorm.DB) BalanceRepository {
	return BalanceRepository{db: db}
}

func NewCreatorRepository(db *gorm.DB, cfg config.CreatorCenterConfig) CreatorRepository {
	return CreatorRepository{db: db, cfg: cfg}
}

func (r BalanceRepository) GetOrCreateUserPoints(ctx context.Context, userID int64) (*domain.UserPoints, error) {
	var points domain.UserPoints
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&points).Error
	if err == nil {
		return &points, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	points = domain.UserPoints{UserID: userID, Points: 0}
	if err := r.db.WithContext(ctx).Create(&points).Error; err != nil {
		return nil, err
	}
	return &points, nil
}

func (r BalanceRepository) UserOAuth2ID(ctx context.Context, userID int64) (*int64, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).Where("id = ?", userID).Select("id", "oauth2_id").First(&user).Error; err != nil {
		return nil, err
	}
	return user.OAuth2ID, nil
}

func (r BalanceRepository) PurchaseQuote(ctx context.Context, input PurchaseContentInput) (*PurchaseContentResult, error) {
	var post domain.Post
	if err := r.db.WithContext(ctx).Where("id = ?", input.PostID).Select("id", "user_id", "title").First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPurchasePostMissing
		}
		return nil, err
	}
	if post.UserID == input.UserID {
		return nil, ErrPurchaseOwnContent
	}
	var payment domain.PostPaymentSetting
	if err := r.db.WithContext(ctx).Where("post_id = ?", input.PostID).First(&payment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPurchaseNotPaidContent
		}
		return nil, err
	}
	if !payment.Enabled {
		return nil, ErrPurchaseNotPaidContent
	}
	method := normalizePurchasePaymentMethod(payment.PaymentMethod)
	if requested := normalizePurchasePaymentMethod(input.PaymentMethod); strings.TrimSpace(input.PaymentMethod) != "" && requested != method {
		return nil, ErrPurchasePaymentMethod
	}
	var existing domain.UserPurchasedContent
	if err := r.db.WithContext(ctx).Where("user_id = ? AND post_id = ?", input.UserID, input.PostID).Select("id").First(&existing).Error; err == nil {
		return &PurchaseContentResult{Post: post, PaymentSetting: payment, Already: true, PurchaseID: existing.ID}, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	price := mathRound2(payment.Price)
	discountRate := 1.0
	couponDiscount := 0.0
	if method == "balance" {
		if input.VIPLevel >= 2 {
			discountRate = 0.95
		} else if input.VIPLevel >= 1 {
			discountRate = 0.98
		}
		actualPayAmount := mathRound2(price * discountRate)
		var err error
		couponDiscount, err = r.couponDiscount(ctx, input.UserID, input.UserCouponID, actualPayAmount)
		if err != nil {
			return nil, err
		}
	}
	actualPayAmount := mathRound2(price * discountRate)
	finalPayAmount := mathRound2(math.Max(0, actualPayAmount-couponDiscount))
	return &PurchaseContentResult{
		Post:           post,
		PaymentSetting: payment,
		Price:          price,
		PaidAmount:     finalPayAmount,
		DiscountRate:   discountRate,
		CouponDiscount: couponDiscount,
	}, nil
}

func (r BalanceRepository) PurchaseContent(ctx context.Context, input PurchaseContentInput) (*PurchaseContentResult, error) {
	var result PurchaseContentResult
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var post domain.Post
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", input.PostID).Select("id", "user_id", "title").First(&post).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPurchasePostMissing
			}
			return err
		}
		if post.UserID == input.UserID {
			return ErrPurchaseOwnContent
		}

		var payment domain.PostPaymentSetting
		if err := tx.Where("post_id = ?", input.PostID).First(&payment).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPurchaseNotPaidContent
			}
			return err
		}
		if !payment.Enabled {
			return ErrPurchaseNotPaidContent
		}
		method := normalizePurchasePaymentMethod(payment.PaymentMethod)
		if requested := normalizePurchasePaymentMethod(input.PaymentMethod); strings.TrimSpace(input.PaymentMethod) != "" && requested != method {
			return ErrPurchasePaymentMethod
		}

		var existing domain.UserPurchasedContent
		if err := tx.Where("user_id = ? AND post_id = ?", input.UserID, input.PostID).Select("id").First(&existing).Error; err == nil {
			result = PurchaseContentResult{Post: post, PaymentSetting: payment, Already: true, PurchaseID: existing.ID}
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		price := mathRound2(payment.Price)
		discountRate := 1.0
		couponDiscount := 0.0
		if method == "balance" {
			if input.VIPLevel >= 2 {
				discountRate = 0.95
			} else if input.VIPLevel >= 1 {
				discountRate = 0.98
			}
		}
		actualPayAmount := mathRound2(price * discountRate)
		if method == "balance" {
			var err error
			couponDiscount, err = r.consumeCouponIfAny(ctx, tx, input.UserID, input.UserCouponID, actualPayAmount)
			if err != nil {
				return err
			}
		}
		finalPayAmount := mathRound2(math.Max(0, actualPayAmount-couponDiscount))
		if method == "points" {
			pointsResult, err := r.purchaseContentWithPointsTx(ctx, tx, input, post, payment, price, finalPayAmount, discountRate)
			if err != nil {
				return err
			}
			result = pointsResult
			return nil
		}

		balanceAfter := mathRound2(input.BalanceAfter)

		earnings, err := getOrCreateCreatorEarningsTx(ctx, tx, post.UserID)
		if err != nil {
			return err
		}
		platformFeeRate := input.PlatformFeeRate
		if platformFeeRate < 0 {
			platformFeeRate = 0
		}
		platformFee := mathRound2(price * platformFeeRate)
		netAmount := mathRound2(price - platformFee)
		newBalance := mathRound2(earnings.Balance + netAmount)
		newTotal := mathRound2(earnings.TotalEarnings + netAmount)
		if err := tx.Model(&domain.CreatorEarnings{}).Where("user_id = ?", post.UserID).Updates(map[string]any{
			"balance":        newBalance,
			"total_earnings": newTotal,
		}).Error; err != nil {
			return err
		}
		sourceType := "post"
		reason := "\u4ed8\u8d39\u5185\u5bb9\u9500\u552e: " + post.Title
		log := domain.CreatorEarningsLog{
			UserID:       post.UserID,
			EarningsID:   earnings.ID,
			Amount:       netAmount,
			BalanceAfter: newBalance,
			Type:         "content_sale",
			SourceID:     &post.ID,
			SourceType:   &sourceType,
			BuyerID:      &input.UserID,
			Reason:       &reason,
			PlatformFee:  platformFee,
		}
		if err := tx.Create(&log).Error; err != nil {
			return err
		}

		purchase := domain.UserPurchasedContent{
			UserID:        input.UserID,
			PostID:        input.PostID,
			AuthorID:      post.UserID,
			Price:         price,
			PaidAmount:    finalPayAmount,
			DiscountRate:  discountRate,
			PurchaseType:  payment.PaymentType,
			PaymentMethod: method,
			PurchasedAt:   time.Now(),
		}
		if purchase.PurchaseType == "" {
			purchase.PurchaseType = "single"
		}
		if err := tx.Create(&purchase).Error; err != nil {
			return err
		}
		result = PurchaseContentResult{
			Post:           post,
			PaymentSetting: payment,
			Price:          price,
			PaidAmount:     finalPayAmount,
			DiscountRate:   discountRate,
			CouponDiscount: couponDiscount,
			BalanceAfter:   balanceAfter,
			AuthorEarnings: netAmount,
			PlatformFee:    platformFee,
			PurchaseID:     purchase.ID,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r BalanceRepository) purchaseContentWithPointsTx(ctx context.Context, tx *gorm.DB, input PurchaseContentInput, post domain.Post, payment domain.PostPaymentSetting, price, paidAmount, discountRate float64) (PurchaseContentResult, error) {
	buyerPoints, err := getOrCreateUserPointsTx(ctx, tx, input.UserID)
	if err != nil {
		return PurchaseContentResult{}, err
	}
	if buyerPoints.Points < paidAmount {
		return PurchaseContentResult{}, ErrPointsInsufficient
	}
	authorPoints, err := getOrCreateUserPointsTx(ctx, tx, post.UserID)
	if err != nil {
		return PurchaseContentResult{}, err
	}
	buyerBalance := mathRound2(buyerPoints.Points - paidAmount)
	authorBalance := mathRound2(authorPoints.Points + paidAmount)
	if err := tx.Model(&domain.UserPoints{}).Where("user_id = ?", input.UserID).Update("points", buyerBalance).Error; err != nil {
		return PurchaseContentResult{}, err
	}
	if err := tx.Model(&domain.UserPoints{}).Where("user_id = ?", post.UserID).Update("points", authorBalance).Error; err != nil {
		return PurchaseContentResult{}, err
	}
	purchase := domain.UserPurchasedContent{
		UserID:        input.UserID,
		PostID:        input.PostID,
		AuthorID:      post.UserID,
		Price:         price,
		PaidAmount:    paidAmount,
		DiscountRate:  discountRate,
		PurchaseType:  payment.PaymentType,
		PaymentMethod: "points",
		PurchasedAt:   time.Now(),
	}
	if purchase.PurchaseType == "" {
		purchase.PurchaseType = "single"
	}
	if err := tx.Create(&purchase).Error; err != nil {
		return PurchaseContentResult{}, err
	}
	reason := "付费内容积分购买: " + post.Title
	if err := tx.Create(&domain.PointsLog{
		UserID:             input.UserID,
		Amount:             -paidAmount,
		BalanceAfter:       buyerBalance,
		Type:               "paid_content_purchase",
		Reason:             &reason,
		PostID:             &post.ID,
		PurchaseID:         &purchase.ID,
		EntryRole:          "buyer_debit",
		CounterpartyUserID: &post.UserID,
		PaymentMethod:      "points",
	}).Error; err != nil {
		return PurchaseContentResult{}, err
	}
	saleReason := "付费内容积分收入: " + post.Title
	if err := tx.Create(&domain.PointsLog{
		UserID:             post.UserID,
		Amount:             paidAmount,
		BalanceAfter:       authorBalance,
		Type:               "paid_content_sale",
		Reason:             &saleReason,
		PostID:             &post.ID,
		PurchaseID:         &purchase.ID,
		EntryRole:          "author_credit",
		CounterpartyUserID: &input.UserID,
		PaymentMethod:      "points",
	}).Error; err != nil {
		return PurchaseContentResult{}, err
	}
	return PurchaseContentResult{
		Post:           post,
		PaymentSetting: payment,
		Price:          price,
		PaidAmount:     paidAmount,
		DiscountRate:   discountRate,
		AuthorEarnings: paidAmount,
		PurchaseID:     purchase.ID,
	}, nil
}

func (r BalanceRepository) ReservePurchaseIntent(ctx context.Context, input PurchaseContentInput, price, paidAmount float64) (*PurchaseIntentReservation, error) {
	var reservation PurchaseIntentReservation
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var purchase domain.UserPurchasedContent
		if err := tx.Where("user_id = ? AND post_id = ?", input.UserID, input.PostID).Select("id").First(&purchase).Error; err == nil {
			reservation.Completed = true
			reservation.PurchaseID = purchase.ID
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var intent domain.ContentPurchaseIntent
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND post_id = ?", input.UserID, input.PostID).
			First(&intent).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			now := time.Now()
			intent = domain.ContentPurchaseIntent{
				UserID:        input.UserID,
				PostID:        input.PostID,
				PaymentMethod: normalizePurchasePaymentMethod(input.PaymentMethod),
				Status:        "processing",
				Price:         mathRound2(price),
				PaidAmount:    mathRound2(paidAmount),
				CreatedAt:     now,
				UpdatedAt:     &now,
			}
			if err := tx.Create(&intent).Error; err != nil {
				return ErrPurchaseInProgress
			}
			reservation.Intent = intent
			reservation.Acquired = true
			return nil
		}
		if err != nil {
			return err
		}
		if intent.Status == "completed" {
			reservation.Intent = intent
			reservation.Completed = true
			if err := tx.Where("user_id = ? AND post_id = ?", input.UserID, input.PostID).Select("id").First(&purchase).Error; err == nil {
				reservation.PurchaseID = purchase.ID
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			return nil
		}
		if intent.Status == "processing" {
			return ErrPurchaseInProgress
		}
		now := time.Now()
		if err := tx.Model(&domain.ContentPurchaseIntent{}).Where("id = ?", intent.ID).Updates(map[string]any{
			"payment_method": normalizePurchasePaymentMethod(input.PaymentMethod),
			"status":         "processing",
			"price":          mathRound2(price),
			"paid_amount":    mathRound2(paidAmount),
			"error_code":     nil,
			"updated_at":     now,
		}).Error; err != nil {
			return err
		}
		intent.Status = "processing"
		intent.Price = mathRound2(price)
		intent.PaidAmount = mathRound2(paidAmount)
		intent.UpdatedAt = &now
		reservation.Intent = intent
		reservation.Acquired = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &reservation, nil
}

func (r BalanceRepository) MarkPurchaseIntentDebited(ctx context.Context, intentID int64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&domain.ContentPurchaseIntent{}).Where("id = ?", intentID).Updates(map[string]any{
		"external_debit_applied": true,
		"updated_at":             now,
	}).Error
}

func (r BalanceRepository) CompletePurchaseIntent(ctx context.Context, intentID int64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&domain.ContentPurchaseIntent{}).Where("id = ?", intentID).Updates(map[string]any{
		"status":     "completed",
		"error_code": nil,
		"updated_at": now,
	}).Error
}

func (r BalanceRepository) FailPurchaseIntent(ctx context.Context, intentID int64, code string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&domain.ContentPurchaseIntent{}).Where("id = ?", intentID).Updates(map[string]any{
		"status":     "failed",
		"error_code": strings.TrimSpace(code),
		"updated_at": now,
	}).Error
}

func normalizePurchasePaymentMethod(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "points":
		return "points"
	default:
		return "balance"
	}
}

func (r BalanceRepository) couponDiscount(ctx context.Context, userID int64, userCouponID *int64, orderAmount float64) (float64, error) {
	if userCouponID == nil {
		return 0, nil
	}
	var userCoupon domain.UserCoupon
	if err := r.db.WithContext(ctx).Where("id = ?", *userCouponID).First(&userCoupon).Error; err != nil {
		return 0, err
	}
	if userCoupon.UserID != userID {
		return 0, ErrCouponWrongOwner
	}
	if userCoupon.Status == "used" {
		return 0, ErrCouponUsed
	}
	if userCoupon.Status == "expired" {
		return 0, ErrCouponExpired
	}
	var coupon domain.Coupon
	if err := r.db.WithContext(ctx).Where("id = ?", userCoupon.CouponID).First(&coupon).Error; err != nil {
		return 0, err
	}
	if err := validateCouponWindow(coupon, orderAmount, true); err != nil {
		return 0, err
	}
	return calculateDiscount(coupon, orderAmount), nil
}

func (r BalanceRepository) consumeCouponIfAny(ctx context.Context, tx *gorm.DB, userID int64, userCouponID *int64, orderAmount float64) (float64, error) {
	if userCouponID == nil {
		return 0, nil
	}
	var userCoupon domain.UserCoupon
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", *userCouponID).First(&userCoupon).Error; err != nil {
		return 0, err
	}
	if userCoupon.UserID != userID {
		return 0, ErrCouponWrongOwner
	}
	if userCoupon.Status == "used" {
		return 0, ErrCouponUsed
	}
	if userCoupon.Status == "expired" {
		return 0, ErrCouponExpired
	}
	var coupon domain.Coupon
	if err := tx.Where("id = ?", userCoupon.CouponID).First(&coupon).Error; err != nil {
		return 0, err
	}
	if err := validateCouponWindow(coupon, orderAmount, true); err != nil {
		return 0, err
	}
	discount := calculateDiscount(coupon, orderAmount)
	now := time.Now()
	if err := tx.Model(&domain.UserCoupon{}).Where("id = ?", userCoupon.ID).Updates(map[string]any{"status": "used", "used_at": now}).Error; err != nil {
		return 0, err
	}
	if err := tx.Model(&domain.Coupon{}).Where("id = ?", coupon.ID).UpdateColumn("used_count", gorm.Expr("used_count + ?", 1)).Error; err != nil {
		return 0, err
	}
	return discount, nil
}

func (r BalanceRepository) Orders(ctx context.Context, userID int64, page, limit int) (int64, []PurchaseOrderBundle, error) {
	query := r.db.WithContext(ctx).Model(&domain.UserPurchasedContent{}).Where("user_id = ?", userID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	var purchases []domain.UserPurchasedContent
	if err := query.Order("purchased_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&purchases).Error; err != nil {
		return 0, nil, err
	}
	posts, err := r.postsByID(ctx, purchasePostIDs(purchases))
	if err != nil {
		return 0, nil, err
	}
	covers, err := r.freePreviewCovers(ctx, purchasePostIDs(purchases))
	if err != nil {
		return 0, nil, err
	}
	out := make([]PurchaseOrderBundle, 0, len(purchases))
	for _, purchase := range purchases {
		out = append(out, PurchaseOrderBundle{Purchase: purchase, Post: posts[purchase.PostID], Cover: covers[purchase.PostID]})
	}
	return total, out, nil
}

func (r BalanceRepository) PostPurchaseUsers(ctx context.Context, postID int64, page, limit int) (int64, []PostPurchaseUserBundle, error) {
	query := r.db.WithContext(ctx).Model(&domain.UserPurchasedContent{}).Where("post_id = ?", postID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	var purchases []domain.UserPurchasedContent
	if err := query.Order("purchased_at DESC, id DESC").Offset((page - 1) * limit).Limit(limit).Find(&purchases).Error; err != nil {
		return 0, nil, err
	}
	users, err := r.purchaseUsersByID(ctx, purchaseUserIDs(purchases))
	if err != nil {
		return 0, nil, err
	}
	out := make([]PostPurchaseUserBundle, 0, len(purchases))
	for _, purchase := range purchases {
		out = append(out, PostPurchaseUserBundle{Purchase: purchase, Buyer: users[purchase.UserID]})
	}
	return total, out, nil
}

func (r BalanceRepository) CheckPurchase(ctx context.Context, userID, postID int64) (*domain.UserPurchasedContent, error) {
	var purchase domain.UserPurchasedContent
	err := r.db.WithContext(ctx).Where("user_id = ? AND post_id = ?", userID, postID).Select("id", "purchased_at").First(&purchase).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &purchase, nil
}

func purchaseUserIDs(purchases []domain.UserPurchasedContent) []int64 {
	ids := make([]int64, 0, len(purchases))
	for _, purchase := range purchases {
		ids = append(ids, purchase.UserID)
	}
	return ids
}

func (r BalanceRepository) purchaseUsersByID(ctx context.Context, ids []int64) (map[int64]*domain.User, error) {
	out := map[int64]*domain.User{}
	if len(ids) == 0 {
		return out, nil
	}
	var users []domain.User
	if err := r.db.WithContext(ctx).
		Where("id IN ?", uniqueInt64(ids)).
		Select("id", "user_id", "nickname", "avatar", "verified").
		Find(&users).Error; err != nil {
		return nil, err
	}
	for i := range users {
		out[users[i].ID] = &users[i]
	}
	return out, nil
}

func (r BalanceRepository) postsByID(ctx context.Context, ids []int64) (map[int64]*domain.Post, error) {
	out := map[int64]*domain.Post{}
	if len(ids) == 0 {
		return out, nil
	}
	var posts []domain.Post
	if err := r.db.WithContext(ctx).Where("id IN ?", uniqueInt64(ids)).Select("id", "title").Find(&posts).Error; err != nil {
		return nil, err
	}
	for i := range posts {
		out[posts[i].ID] = &posts[i]
	}
	return out, nil
}

func (r BalanceRepository) freePreviewCovers(ctx context.Context, ids []int64) (map[int64]*string, error) {
	out := map[int64]*string{}
	if len(ids) == 0 {
		return out, nil
	}
	var images []domain.PostImage
	if err := r.db.WithContext(ctx).
		Where("post_id IN ? AND is_protected = ?", uniqueInt64(ids), false).
		Order("post_id ASC, sort_order ASC, id ASC").
		Find(&images).Error; err != nil {
		return nil, err
	}
	for i := range images {
		if _, exists := out[images[i].PostID]; !exists {
			out[images[i].PostID] = &images[i].ImageURL
		}
	}
	return out, nil
}
