package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
)

const (
	ExternalBalancePrepared       = "prepared"
	ExternalBalanceRequesting     = "requesting"
	ExternalBalanceApplied        = "applied"
	ExternalBalanceLocalCommitted = "local_committed"
	ExternalBalanceCompensating   = "compensating"
	ExternalBalanceCompensated    = "compensated"
	ExternalBalanceUnknown        = "unknown"
	ExternalBalanceFailed         = "failed"
)

var (
	ErrBalanceCenterConfig     = errors.New("balance center configuration is incomplete")
	ErrBalanceOAuth2Missing    = errors.New("oauth2 id is missing")
	ErrBalanceOperationBusy    = errors.New("another balance operation is in progress")
	ErrBalanceOperationUnknown = errors.New("balance operation result is unknown")
	ErrBalanceRemoteRejected   = errors.New("balance center rejected operation")
	ErrBalanceMutationInvalid  = errors.New("balance mutation is invalid")
)

type BalanceCenterService struct {
	db     *gorm.DB
	cfg    config.BalanceCenterConfig
	client *http.Client
	locks  [128]sync.Mutex
}

type BalanceCenterUser struct {
	ID          int64
	Username    string
	Email       string
	Balance     float64
	VIPLevel    int
	VIPExpireAt *time.Time
	IsActive    bool
	CreatedAt   *time.Time
}

type BalanceMutationInput struct {
	OperationKey       string
	UserID             int64
	OAuth2ID           int64
	Amount             float64
	Reason             string
	PostID             *int64
	PurchaseID         *int64
	CounterpartyUserID *int64
	EntryRole          string
	PaymentMethod      string
	PlatformFee        float64
	LocalCommit        func(*gorm.DB, *domain.ExternalBalanceTransaction) error
}

type BalanceMutationResult struct {
	Transaction  domain.ExternalBalanceTransaction
	BalanceAfter *float64
	AlreadyDone  bool
}

type balanceCenterEnvelope struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

type balanceRemoteError struct {
	err     error
	unknown bool
}

func (e *balanceRemoteError) Error() string { return e.err.Error() }
func (e *balanceRemoteError) Unwrap() error { return e.err }

func NewBalanceCenterService(db *gorm.DB, cfg config.BalanceCenterConfig) *BalanceCenterService {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &BalanceCenterService{db: db, cfg: cfg, client: &http.Client{Timeout: timeout}}
}

func (s *BalanceCenterService) UserByInternalID(ctx context.Context, userID int64) (*BalanceCenterUser, int64, error) {
	if s == nil || s.db == nil {
		return nil, 0, gorm.ErrInvalidDB
	}
	var user domain.User
	if err := s.db.WithContext(ctx).Select("id", "oauth2_id").Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, 0, err
	}
	if user.OAuth2ID == nil || *user.OAuth2ID <= 0 {
		return nil, 0, ErrBalanceOAuth2Missing
	}
	remote, err := s.User(ctx, *user.OAuth2ID)
	return remote, *user.OAuth2ID, err
}

func (s *BalanceCenterService) User(ctx context.Context, oauth2ID int64) (*BalanceCenterUser, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if oauth2ID <= 0 {
		return nil, ErrBalanceOAuth2Missing
	}
	endpoint := strings.TrimRight(s.cfg.APIURL, "/") + "/api/external/user?user_id=" + url.QueryEscape(strconv.FormatInt(oauth2ID, 10))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", s.cfg.APIKey)
	envelope, err := s.do(req, false)
	if err != nil {
		return nil, err
	}
	if !envelope.Success {
		return nil, fmt.Errorf("%w: %s", ErrBalanceRemoteRejected, strings.TrimSpace(envelope.Message))
	}
	id, ok := int64FromBalanceValue(envelope.Data["id"])
	if !ok || id != oauth2ID {
		return nil, errors.New("balance center user response id does not match requested oauth2 id")
	}
	balance, ok := float64FromBalanceValue(envelope.Data["balance"])
	if !ok {
		return nil, errors.New("balance center user response is missing balance")
	}
	vipLevel, _ := int64FromBalanceValue(envelope.Data["vip_level"])
	return &BalanceCenterUser{
		ID:          id,
		Username:    stringFromBalanceValue(envelope.Data["username"]),
		Email:       stringFromBalanceValue(envelope.Data["email"]),
		Balance:     roundBalance(balance),
		VIPLevel:    int(vipLevel),
		VIPExpireAt: timeFromBalanceValue(envelope.Data["vip_expire_at"]),
		IsActive:    boolFromBalanceValue(envelope.Data["is_active"]),
		CreatedAt:   timeFromBalanceValue(envelope.Data["created_at"]),
	}, nil
}

func (s *BalanceCenterService) ChangeBalance(ctx context.Context, input BalanceMutationInput) (*BalanceMutationResult, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	input.OperationKey = strings.TrimSpace(input.OperationKey)
	input.Reason = strings.TrimSpace(input.Reason)
	input.Amount = roundBalance(input.Amount)
	if input.OperationKey == "" || input.UserID <= 0 || input.OAuth2ID <= 0 || input.Amount == 0 || math.IsNaN(input.Amount) || math.IsInf(input.Amount, 0) {
		return nil, ErrBalanceMutationInvalid
	}
	lock := &s.locks[uint64(input.OAuth2ID)%uint64(len(s.locks))]
	lock.Lock()
	defer lock.Unlock()

	record, alreadyDone, err := s.prepareMutation(ctx, input)
	if err != nil || alreadyDone {
		if err != nil {
			return nil, err
		}
		return &BalanceMutationResult{Transaction: *record, BalanceAfter: record.RemoteBalanceAfter, AlreadyDone: true}, nil
	}

	balanceAfter, err := s.changeRemote(ctx, input.OAuth2ID, input.Amount, input.Reason)
	if err != nil {
		status := ExternalBalanceFailed
		var remoteErr *balanceRemoteError
		if errors.As(err, &remoteErr) && remoteErr.unknown {
			status = ExternalBalanceUnknown
		}
		_ = s.finishMutation(ctx, record.ID, input.OAuth2ID, status, nil, err)
		if status == ExternalBalanceUnknown {
			return nil, errors.Join(ErrBalanceOperationUnknown, err)
		}
		return nil, err
	}
	if err := s.markApplied(ctx, record.ID, balanceAfter); err != nil {
		_ = s.finishMutation(ctx, record.ID, input.OAuth2ID, ExternalBalanceUnknown, balanceAfter, err)
		return nil, errors.Join(ErrBalanceOperationUnknown, err)
	}

	commitErr := s.commitLocal(ctx, record.ID, input.OAuth2ID, input.LocalCommit)
	if commitErr != nil {
		compensation := roundBalance(-input.Amount)
		_ = s.setCompensating(ctx, record.ID, compensation, commitErr)
		compensatedBalance, compensateErr := s.changeRemote(ctx, input.OAuth2ID, compensation, "compensation: "+input.Reason)
		if compensateErr != nil {
			_ = s.finishMutation(ctx, record.ID, input.OAuth2ID, ExternalBalanceUnknown, nil, errors.Join(commitErr, compensateErr))
			return nil, errors.Join(commitErr, ErrBalanceOperationUnknown, compensateErr)
		}
		_ = s.finishMutation(ctx, record.ID, input.OAuth2ID, ExternalBalanceCompensated, compensatedBalance, commitErr)
		return nil, commitErr
	}

	if err := s.db.WithContext(ctx).Where("id = ?", record.ID).First(record).Error; err != nil {
		return nil, err
	}
	return &BalanceMutationResult{Transaction: *record, BalanceAfter: record.RemoteBalanceAfter}, nil
}

func (s *BalanceCenterService) CompensateApplied(ctx context.Context, transactionID int64) (*domain.ExternalBalanceTransaction, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	var target domain.ExternalBalanceTransaction
	if err := s.db.WithContext(ctx).Select("id", "oauth2_id").Where("id = ?", transactionID).First(&target).Error; err != nil {
		return nil, err
	}
	lock := &s.locks[uint64(target.OAuth2ID)%uint64(len(s.locks))]
	lock.Lock()
	defer lock.Unlock()

	var record domain.ExternalBalanceTransaction
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var account domain.ExternalBalanceAccount
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("oauth2_id = ?", target.OAuth2ID).First(&account).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", transactionID).First(&record).Error; err != nil {
			return err
		}
		if record.OAuth2ID != target.OAuth2ID || record.Status != ExternalBalanceApplied {
			return ErrBalanceOperationUnknown
		}
		if account.ActiveOperationID != nil && *account.ActiveOperationID != record.ID {
			return ErrBalanceOperationBusy
		}
		now := time.Now()
		compensation := roundBalance(-record.Amount)
		if err := tx.Model(&domain.ExternalBalanceAccount{}).Where("oauth2_id = ?", record.OAuth2ID).Updates(map[string]any{
			"active_operation_id": record.ID, "updated_at": now,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&domain.ExternalBalanceTransaction{}).Where("id = ?", record.ID).Updates(map[string]any{
			"status": ExternalBalanceCompensating, "compensation_amount": compensation, "updated_at": now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	compensatedBalance, err := s.changeRemote(ctx, record.OAuth2ID, -record.Amount, "manual compensation: "+record.Reason)
	if err != nil {
		_ = s.finishMutation(ctx, record.ID, record.OAuth2ID, ExternalBalanceUnknown, nil, err)
		return nil, err
	}
	if err := s.finishMutation(ctx, record.ID, record.OAuth2ID, ExternalBalanceCompensated, compensatedBalance, nil); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("id = ?", record.ID).First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func (s *BalanceCenterService) prepareMutation(ctx context.Context, input BalanceMutationInput) (*domain.ExternalBalanceTransaction, bool, error) {
	var record domain.ExternalBalanceTransaction
	alreadyDone := false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&domain.ExternalBalanceAccount{
			OAuth2ID: input.OAuth2ID, UserID: input.UserID, CreatedAt: now, UpdatedAt: &now,
		}).Error; err != nil {
			return err
		}
		var account domain.ExternalBalanceAccount
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("oauth2_id = ?", input.OAuth2ID).First(&account).Error; err != nil {
			return err
		}
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("operation_key = ?", input.OperationKey).First(&record).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			record = domain.ExternalBalanceTransaction{
				OperationKey:       input.OperationKey,
				UserID:             input.UserID,
				OAuth2ID:           input.OAuth2ID,
				Amount:             input.Amount,
				Reason:             input.Reason,
				PostID:             input.PostID,
				PurchaseID:         input.PurchaseID,
				CounterpartyUserID: input.CounterpartyUserID,
				EntryRole:          strings.TrimSpace(input.EntryRole),
				PaymentMethod:      strings.TrimSpace(input.PaymentMethod),
				PlatformFee:        roundBalance(input.PlatformFee),
				Status:             ExternalBalancePrepared,
				CreatedAt:          now,
				UpdatedAt:          &now,
			}
			if err := tx.Create(&record).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		if record.UserID != input.UserID || record.OAuth2ID != input.OAuth2ID || roundBalance(record.Amount) != input.Amount {
			return ErrBalanceMutationInvalid
		}
		switch record.Status {
		case ExternalBalanceLocalCommitted:
			alreadyDone = true
			return nil
		case ExternalBalanceRequesting, ExternalBalanceApplied, ExternalBalanceCompensating, ExternalBalanceUnknown:
			return ErrBalanceOperationUnknown
		case ExternalBalanceCompensated:
			return ErrBalanceRemoteRejected
		}
		if account.ActiveOperationID != nil && *account.ActiveOperationID != record.ID {
			return ErrBalanceOperationBusy
		}
		if err := tx.Model(&domain.ExternalBalanceAccount{}).Where("oauth2_id = ?", input.OAuth2ID).Updates(map[string]any{
			"active_operation_id": record.ID,
			"updated_at":          now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.ExternalBalanceTransaction{}).Where("id = ?", record.ID).Updates(map[string]any{
			"status":     ExternalBalanceRequesting,
			"attempts":   gorm.Expr("attempts + 1"),
			"last_error": nil,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
		record.Status = ExternalBalanceRequesting
		return nil
	})
	return &record, alreadyDone, err
}

func (s *BalanceCenterService) markApplied(ctx context.Context, transactionID int64, balanceAfter *float64) error {
	now := time.Now()
	result := s.db.WithContext(ctx).Model(&domain.ExternalBalanceTransaction{}).Where("id = ? AND status = ?", transactionID, ExternalBalanceRequesting).Updates(map[string]any{
		"status":               ExternalBalanceApplied,
		"remote_balance_after": balanceAfter,
		"applied_at":           now,
		"updated_at":           now,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrBalanceOperationUnknown
	}
	return nil
}

func (s *BalanceCenterService) commitLocal(ctx context.Context, transactionID, oauth2ID int64, commit func(*gorm.DB, *domain.ExternalBalanceTransaction) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var account domain.ExternalBalanceAccount
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("oauth2_id = ?", oauth2ID).First(&account).Error; err != nil {
			return err
		}
		var record domain.ExternalBalanceTransaction
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", transactionID).First(&record).Error; err != nil {
			return err
		}
		if record.Status != ExternalBalanceApplied {
			return ErrBalanceOperationUnknown
		}
		if commit != nil {
			if err := commit(tx, &record); err != nil {
				return err
			}
		}
		now := time.Now()
		if err := tx.Model(&domain.ExternalBalanceTransaction{}).Where("id = ?", transactionID).Updates(map[string]any{
			"status":       ExternalBalanceLocalCommitted,
			"completed_at": now,
			"updated_at":   now,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&domain.ExternalBalanceAccount{}).Where("oauth2_id = ? AND active_operation_id = ?", oauth2ID, transactionID).Updates(map[string]any{
			"active_operation_id": nil,
			"updated_at":          now,
		}).Error
	})
}

func (s *BalanceCenterService) setCompensating(ctx context.Context, transactionID int64, amount float64, cause error) error {
	now := time.Now()
	message := errorText(cause)
	return s.db.WithContext(ctx).Model(&domain.ExternalBalanceTransaction{}).Where("id = ?", transactionID).Updates(map[string]any{
		"status":              ExternalBalanceCompensating,
		"compensation_amount": amount,
		"last_error":          message,
		"updated_at":          now,
	}).Error
}

func (s *BalanceCenterService) finishMutation(ctx context.Context, transactionID, oauth2ID int64, status string, balanceAfter *float64, cause error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		updates := map[string]any{
			"status":               status,
			"remote_balance_after": balanceAfter,
			"last_error":           errorText(cause),
			"updated_at":           now,
		}
		if status == ExternalBalanceCompensated || status == ExternalBalanceFailed {
			updates["completed_at"] = now
		}
		if err := tx.Model(&domain.ExternalBalanceTransaction{}).Where("id = ?", transactionID).Updates(updates).Error; err != nil {
			return err
		}
		if status == ExternalBalanceUnknown {
			return nil
		}
		return tx.Model(&domain.ExternalBalanceAccount{}).Where("oauth2_id = ? AND active_operation_id = ?", oauth2ID, transactionID).Updates(map[string]any{
			"active_operation_id": nil,
			"updated_at":          now,
		}).Error
	})
}

func (s *BalanceCenterService) changeRemote(ctx context.Context, oauth2ID int64, amount float64, reason string) (*float64, error) {
	payload, err := json.Marshal(map[string]any{"user_id": oauth2ID, "amount": amount, "reason": reason})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(s.cfg.APIURL, "/")+"/api/external/balance", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", s.cfg.APIKey)
	envelope, err := s.do(req, true)
	if err != nil {
		return nil, err
	}
	if !envelope.Success {
		return nil, &balanceRemoteError{err: fmt.Errorf("%w: %s", ErrBalanceRemoteRejected, strings.TrimSpace(envelope.Message))}
	}
	if balance, ok := float64FromBalanceValue(envelope.Data["balance"]); ok {
		balance = roundBalance(balance)
		return &balance, nil
	}
	return nil, nil
}

func (s *BalanceCenterService) do(req *http.Request, mutation bool) (*balanceCenterEnvelope, error) {
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, &balanceRemoteError{err: err, unknown: mutation}
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if readErr != nil {
		return nil, &balanceRemoteError{err: readErr, unknown: mutation}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		unknown := mutation && resp.StatusCode >= 500
		return nil, &balanceRemoteError{err: fmt.Errorf("balance center returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body))), unknown: unknown}
	}
	var envelope balanceCenterEnvelope
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&envelope); err != nil {
		return nil, &balanceRemoteError{err: err, unknown: mutation}
	}
	if envelope.Data == nil {
		envelope.Data = map[string]any{}
	}
	return &envelope, nil
}

func (s *BalanceCenterService) ready() error {
	if s == nil || strings.TrimSpace(s.cfg.APIURL) == "" || strings.TrimSpace(s.cfg.APIKey) == "" || s.client == nil {
		return ErrBalanceCenterConfig
	}
	return nil
}

func roundBalance(value float64) float64 { return math.Round(value*100) / 100 }

func float64FromBalanceValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func int64FromBalanceValue(value any) (int64, bool) {
	parsed, ok := float64FromBalanceValue(value)
	return int64(parsed), ok
}

func stringFromBalanceValue(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func boolFromBalanceValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		parsed, _ := strconv.ParseBool(strings.TrimSpace(typed))
		return parsed
	default:
		return false
	}
}

func timeFromBalanceValue(value any) *time.Time {
	text := stringFromBalanceValue(value)
	if text == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, text)
	if err != nil {
		return nil
	}
	return &parsed
}

func errorText(err error) any {
	if err == nil {
		return nil
	}
	text := strings.TrimSpace(err.Error())
	if len(text) > 1000 {
		text = text[:1000]
	}
	return text
}
