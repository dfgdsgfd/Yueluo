package repositories

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yuem-go/backend-gin/internal/domain"
)

type WithdrawRepository struct {
	db *gorm.DB
}

type WithdrawAdminOrder struct {
	Order   domain.WithdrawOrder
	User    *domain.User
	PayCode *domain.UserPaymentCode
}

type InsufficientBalanceError struct {
	Balance float64
}

func (e InsufficientBalanceError) Error() string {
	return "insufficient balance"
}

var (
	ErrWithdrawInvalidStatus  = errors.New("invalid withdraw order status")
	ErrWithdrawPendingExists  = errors.New("pending withdraw order exists")
	ErrWithdrawMissingPayment = errors.New("missing payment code")
)

func NewWithdrawRepository(db *gorm.DB) WithdrawRepository {
	return WithdrawRepository{db: db}
}

func (r WithdrawRepository) GetOrCreateWallet(ctx context.Context, userID int64) (*domain.UserWallet, error) {
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&domain.UserWallet{UserID: userID}).Error; err != nil {
		return nil, err
	}
	var wallet domain.UserWallet
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (r WithdrawRepository) PaymentCode(ctx context.Context, userID int64) (*domain.UserPaymentCode, error) {
	var code domain.UserPaymentCode
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&code).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &code, nil
}

func (r WithdrawRepository) SavePaymentCode(ctx context.Context, userID int64, updates map[string]any) (*domain.UserPaymentCode, error) {
	var code domain.UserPaymentCode
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&code).Error
	if err == nil {
		if len(updates) > 0 {
			if err := r.db.WithContext(ctx).Model(&domain.UserPaymentCode{}).Where("user_id = ?", userID).Updates(updates).Error; err != nil {
				return nil, err
			}
		}
		return r.PaymentCode(ctx, userID)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	code = domain.UserPaymentCode{UserID: userID}
	if value, ok := updates["wechat_url"]; ok {
		code.WechatURL = stringPtrFromUpdate(value)
	}
	if value, ok := updates["alipay_url"]; ok {
		code.AlipayURL = stringPtrFromUpdate(value)
	}
	if err := r.db.WithContext(ctx).Create(&code).Error; err != nil {
		return nil, err
	}
	return &code, nil
}

func (r WithdrawRepository) Apply(ctx context.Context, userID int64, amount float64, withdrawType string) (*domain.WithdrawOrder, error) {
	if withdrawType == "cash" {
		code, err := r.PaymentCode(ctx, userID)
		if err != nil {
			return nil, err
		}
		if code == nil || (blankStringPtr(code.WechatURL) && blankStringPtr(code.AlipayURL)) {
			return nil, ErrWithdrawMissingPayment
		}
	}

	var order domain.WithdrawOrder
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var pending int64
		if err := tx.Model(&domain.WithdrawOrder{}).
			Where("user_id = ? AND status = ?", userID, "pending").
			Count(&pending).Error; err != nil {
			return err
		}
		if pending > 0 {
			return ErrWithdrawPendingExists
		}

		var earnings domain.CreatorEarnings
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&earnings).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return InsufficientBalanceError{Balance: 0}
		}
		if err != nil {
			return err
		}
		if earnings.Balance < amount {
			return InsufficientBalanceError{Balance: earnings.Balance}
		}

		if err := tx.Model(&domain.CreatorEarnings{}).Where("user_id = ?", userID).Updates(map[string]any{
			"balance":          gorm.Expr("balance - ?", amount),
			"withdrawn_amount": gorm.Expr("withdrawn_amount + ?", amount),
		}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).First(&earnings).Error; err != nil {
			return err
		}
		reason := "\u7533\u8bf7\u63d0\u73b0 " + formatMoney(amount) + " \u81f3" + withdrawTypeLabel(withdrawType)
		log := domain.CreatorEarningsLog{
			UserID:       userID,
			EarningsID:   earnings.ID,
			Amount:       -amount,
			BalanceAfter: earnings.Balance,
			Type:         "withdraw",
			Reason:       &reason,
		}
		if err := tx.Create(&log).Error; err != nil {
			return err
		}

		order = domain.WithdrawOrder{UserID: userID, Amount: amount, Type: withdrawType, Status: "pending"}
		return tx.Create(&order).Error
	})
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r WithdrawRepository) PrepareMoonCoinTransfer(ctx context.Context, userID int64, amount float64) (*domain.WithdrawOrder, error) {
	var order domain.WithdrawOrder
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var earnings domain.CreatorEarnings
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&earnings).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return InsufficientBalanceError{Balance: 0}
		}
		if err != nil {
			return err
		}
		if amount <= 0 || earnings.Balance < amount {
			return InsufficientBalanceError{Balance: earnings.Balance}
		}
		order = domain.WithdrawOrder{UserID: userID, Amount: amount, Type: "moon_coin", Status: "processing"}
		return tx.Create(&order).Error
	})
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r WithdrawRepository) FinalizeMoonCoinTransferTx(ctx context.Context, tx *gorm.DB, orderID int64) error {
	var order domain.WithdrawOrder
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", orderID).First(&order).Error; err != nil {
		return err
	}
	if order.Type != "moon_coin" || order.Status != "processing" {
		return ErrWithdrawInvalidStatus
	}
	var earnings domain.CreatorEarnings
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", order.UserID).First(&earnings).Error; err != nil {
		return err
	}
	if earnings.Balance < order.Amount {
		return InsufficientBalanceError{Balance: earnings.Balance}
	}
	newBalance := mathRound2(earnings.Balance - order.Amount)
	if err := tx.WithContext(ctx).Model(&domain.CreatorEarnings{}).Where("id = ?", earnings.ID).Updates(map[string]any{
		"balance":          newBalance,
		"withdrawn_amount": gorm.Expr("withdrawn_amount + ?", order.Amount),
	}).Error; err != nil {
		return err
	}
	reason := "creator earnings transferred to remote moon coin"
	if err := tx.WithContext(ctx).Create(&domain.CreatorEarningsLog{
		UserID: order.UserID, EarningsID: earnings.ID, Amount: -order.Amount, BalanceAfter: newBalance, Type: "moon_coin_transfer", Reason: &reason,
	}).Error; err != nil {
		return err
	}
	now := time.Now()
	remark := "remote moon coin credited"
	return tx.WithContext(ctx).Model(&domain.WithdrawOrder{}).Where("id = ?", order.ID).Updates(map[string]any{
		"status": "paid", "remark": remark, "updated_at": now,
	}).Error
}

func (r WithdrawRepository) FailMoonCoinTransfer(ctx context.Context, orderID int64, reason string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&domain.WithdrawOrder{}).Where("id = ? AND type = ? AND status = ?", orderID, "moon_coin", "processing").Updates(map[string]any{
		"status": "rejected", "remark": strings.TrimSpace(reason), "updated_at": now,
	}).Error
}

func (r WithdrawRepository) UserOrders(ctx context.Context, userID int64, status string, page, limit int) (int64, []domain.WithdrawOrder, error) {
	query := r.db.WithContext(ctx).Model(&domain.WithdrawOrder{}).Where("user_id = ?", userID)
	if strings.TrimSpace(status) != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	var orders []domain.WithdrawOrder
	if err := query.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&orders).Error; err != nil {
		return 0, nil, err
	}
	return total, orders, nil
}

func (r WithdrawRepository) AdminOrders(ctx context.Context, status, keyword string, page, limit int) (int64, []WithdrawAdminOrder, error) {
	query := r.db.WithContext(ctx).Model(&domain.WithdrawOrder{}).
		Joins("LEFT JOIN users ON users.id = withdraw_orders.user_id")
	if strings.TrimSpace(status) != "" {
		query = query.Where("withdraw_orders.status = ?", status)
	}
	if strings.TrimSpace(keyword) != "" {
		like := "%" + strings.TrimSpace(keyword) + "%"
		query = query.Where("users.nickname LIKE ? OR users.user_id LIKE ?", like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	var orders []domain.WithdrawOrder
	if err := query.Select("withdraw_orders.*").Order("withdraw_orders.created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&orders).Error; err != nil {
		return 0, nil, err
	}
	users, err := r.usersByID(ctx, withdrawOrderUserIDs(orders))
	if err != nil {
		return 0, nil, err
	}
	payCodes, err := r.paymentCodesByUserID(ctx, withdrawOrderUserIDs(orders))
	if err != nil {
		return 0, nil, err
	}
	out := make([]WithdrawAdminOrder, 0, len(orders))
	for _, order := range orders {
		out = append(out, WithdrawAdminOrder{Order: order, User: users[order.UserID], PayCode: payCodes[order.UserID]})
	}
	return total, out, nil
}

func (r WithdrawRepository) Approve(ctx context.Context, orderID int64, remark *string) (*domain.WithdrawOrder, error) {
	var order domain.WithdrawOrder
	if err := r.db.WithContext(ctx).Where("id = ?", orderID).First(&order).Error; err != nil {
		return nil, err
	}
	if order.Status != "pending" {
		return nil, ErrWithdrawInvalidStatus
	}
	updates := map[string]any{"status": "approved", "remark": remark}
	if err := r.db.WithContext(ctx).Model(&domain.WithdrawOrder{}).Where("id = ?", orderID).Updates(updates).Error; err != nil {
		return nil, err
	}
	order.Status = "approved"
	order.Remark = remark
	return &order, nil
}

func (r WithdrawRepository) Reject(ctx context.Context, orderID int64, remark *string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order domain.WithdrawOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", orderID).First(&order).Error; err != nil {
			return err
		}
		if order.Status != "pending" && order.Status != "approved" {
			return ErrWithdrawInvalidStatus
		}
		rejectRemark := "\u5ba1\u6838\u672a\u901a\u8fc7"
		if remark != nil && strings.TrimSpace(*remark) != "" {
			rejectRemark = *remark
		}
		if err := tx.Model(&domain.WithdrawOrder{}).Where("id = ?", orderID).Updates(map[string]any{
			"status": "rejected",
			"remark": rejectRemark,
		}).Error; err != nil {
			return err
		}

		var earnings domain.CreatorEarnings
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", order.UserID).First(&earnings).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			earnings = domain.CreatorEarnings{UserID: order.UserID, Balance: order.Amount}
			if err := tx.Create(&earnings).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			if err := tx.Model(&domain.CreatorEarnings{}).Where("user_id = ?", order.UserID).Updates(map[string]any{
				"balance":          gorm.Expr("balance + ?", order.Amount),
				"withdrawn_amount": gorm.Expr("withdrawn_amount - ?", order.Amount),
			}).Error; err != nil {
				return err
			}
			if err := tx.Where("user_id = ?", order.UserID).First(&earnings).Error; err != nil {
				return err
			}
		}
		reason := "\u63d0\u73b0\u7533\u8bf7\u88ab\u9a73\u56de\uff0c\u4f59\u989d\u9000\u8fd8: " + rejectRemark
		log := domain.CreatorEarningsLog{
			UserID:       order.UserID,
			EarningsID:   earnings.ID,
			Amount:       order.Amount,
			BalanceAfter: earnings.Balance,
			Type:         "withdraw_rejected",
			Reason:       &reason,
		}
		return tx.Create(&log).Error
	})
}

func (r WithdrawRepository) Payout(ctx context.Context, orderID int64, remark *string) (*domain.WithdrawOrder, error) {
	var order domain.WithdrawOrder
	if err := r.db.WithContext(ctx).Where("id = ?", orderID).First(&order).Error; err != nil {
		return nil, err
	}
	if order.Status != "approved" {
		return nil, ErrWithdrawInvalidStatus
	}
	payoutRemark := "\u5df2\u5b8c\u6210\u6253\u6b3e"
	if remark != nil && strings.TrimSpace(*remark) != "" {
		payoutRemark = *remark
	}
	if err := r.db.WithContext(ctx).Model(&domain.WithdrawOrder{}).Where("id = ?", orderID).Updates(map[string]any{
		"status": "paid",
		"remark": payoutRemark,
	}).Error; err != nil {
		return nil, err
	}
	order.Status = "paid"
	order.Remark = &payoutRemark
	return &order, nil
}

func (r WithdrawRepository) usersByID(ctx context.Context, ids []int64) (map[int64]*domain.User, error) {
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

func (r WithdrawRepository) paymentCodesByUserID(ctx context.Context, ids []int64) (map[int64]*domain.UserPaymentCode, error) {
	out := map[int64]*domain.UserPaymentCode{}
	if len(ids) == 0 {
		return out, nil
	}
	var codes []domain.UserPaymentCode
	if err := r.db.WithContext(ctx).Where("user_id IN ?", uniqueInt64(ids)).Find(&codes).Error; err != nil {
		return nil, err
	}
	for i := range codes {
		out[codes[i].UserID] = &codes[i]
	}
	return out, nil
}

func withdrawOrderUserIDs(orders []domain.WithdrawOrder) []int64 {
	out := make([]int64, 0, len(orders))
	for _, order := range orders {
		out = append(out, order.UserID)
	}
	return out
}

func stringPtrFromUpdate(value any) *string {
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return nil
	}
	return &text
}

func blankStringPtr(value *string) bool {
	return value == nil || strings.TrimSpace(*value) == ""
}

func withdrawTypeLabel(value string) string {
	if value == "cash" {
		return "\u73b0\u91d1"
	}
	return "\u6708\u5e01"
}
