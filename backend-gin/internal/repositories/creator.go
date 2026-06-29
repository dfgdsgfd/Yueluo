package repositories

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
)

func (r CreatorRepository) Config() config.CreatorCenterConfig {
	return r.cfg
}

func (r CreatorRepository) GetOrCreateEarnings(ctx context.Context, userID int64) (*domain.CreatorEarnings, error) {
	return getOrCreateCreatorEarningsTx(ctx, r.db.WithContext(ctx), userID)
}

func (r CreatorRepository) AddCreatorEarnings(ctx context.Context, userID int64, grossAmount float64, logType string, sourceID *int64, sourceType *string, buyerID *int64, reason *string) (*PurchaseContentResult, error) {
	var res PurchaseContentResult
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		earnings, err := getOrCreateCreatorEarningsTx(ctx, tx, userID)
		if err != nil {
			return err
		}
		platformFee := mathRound2(grossAmount * r.cfg.PlatformFeeRate)
		netAmount := mathRound2(grossAmount - platformFee)
		newBalance := mathRound2(earnings.Balance + netAmount)
		newTotal := mathRound2(earnings.TotalEarnings + netAmount)
		if err := tx.Model(&domain.CreatorEarnings{}).Where("user_id = ?", userID).Updates(map[string]any{
			"balance":        newBalance,
			"total_earnings": newTotal,
		}).Error; err != nil {
			return err
		}
		log := domain.CreatorEarningsLog{UserID: userID, EarningsID: earnings.ID, Amount: netAmount, BalanceAfter: newBalance, Type: logType, SourceID: sourceID, SourceType: sourceType, BuyerID: buyerID, Reason: reason, PlatformFee: platformFee}
		if err := tx.Create(&log).Error; err != nil {
			return err
		}
		res = PurchaseContentResult{Price: grossAmount, PlatformFee: platformFee, AuthorEarnings: netAmount}
		return nil
	})
	return &res, err
}

func (r CreatorRepository) Overview(ctx context.Context, userID int64) (*CreatorOverview, error) {
	earnings, err := r.GetOrCreateEarnings(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	today := dayStart(now)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	todaySum, err := r.earningsSum(ctx, userID, today)
	if err != nil {
		return nil, err
	}
	monthSum, err := r.earningsSum(ctx, userID, monthStart)
	if err != nil {
		return nil, err
	}
	return &CreatorOverview{Earnings: *earnings, TodayEarnings: todaySum, MonthEarnings: monthSum}, nil
}

func (r CreatorRepository) Trends(ctx context.Context, userID int64, days int) (*CreatorTrendData, error) {
	if days < 1 {
		days = 7
	}
	if days > 90 {
		days = 90
	}
	today := dayStart(time.Now())
	dates := make([]time.Time, 0, days)
	for i := days - 1; i >= 0; i-- {
		dates = append(dates, today.AddDate(0, 0, -i))
	}
	postIDs, err := r.userPostIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	data := CreatorTrendData{
		Labels:    make([]string, 0, days),
		Views:     make([]int64, 0, days),
		Likes:     make([]int64, 0, days),
		Collects:  make([]int64, 0, days),
		Comments:  make([]int64, 0, days),
		Followers: make([]int64, 0, days),
	}
	for _, date := range dates {
		next := date.AddDate(0, 0, 1)
		data.Labels = append(data.Labels, fmt.Sprintf("%d/%d", int(date.Month()), date.Day()))
		views, likes, collects, comments, followers, err := r.countDailyInteractions(ctx, userID, postIDs, date, next)
		if err != nil {
			return nil, err
		}
		data.Views = append(data.Views, views)
		data.Likes = append(data.Likes, likes)
		data.Collects = append(data.Collects, collects)
		data.Comments = append(data.Comments, comments)
		data.Followers = append(data.Followers, followers)
	}
	return &data, nil
}

func (r CreatorRepository) Stats(ctx context.Context, userID int64, days int) (*CreatorStatsData, error) {
	if days < 1 {
		days = 30
	}
	if days > 365 {
		days = 365
	}
	now := time.Now()
	today := dayStart(now)
	tomorrow := today.AddDate(0, 0, 1)
	yesterday := today.AddDate(0, 0, -1)
	last7 := today.AddDate(0, 0, -6)
	lastN := today.AddDate(0, 0, -(days - 1))
	lastNKey := fmt.Sprintf("last_%d_days", days)

	fansTotal, err := r.countFollows(ctx, userID, nil, nil)
	if err != nil {
		return nil, err
	}
	fansToday, err := r.countFollows(ctx, userID, &today, &tomorrow)
	if err != nil {
		return nil, err
	}
	fansYesterday, err := r.countFollows(ctx, userID, &yesterday, &today)
	if err != nil {
		return nil, err
	}
	fansLast7, err := r.countFollows(ctx, userID, &last7, nil)
	if err != nil {
		return nil, err
	}
	fansLastN, err := r.countFollows(ctx, userID, &lastN, nil)
	if err != nil {
		return nil, err
	}

	postTotals, postIDs, err := r.postTotals(ctx, userID)
	if err != nil {
		return nil, err
	}
	interactions, err := r.rangeInteractions(ctx, userID, postIDs, []rangeSpec{
		{Key: "today", Start: today, End: &tomorrow},
		{Key: "yesterday", Start: yesterday, End: &today},
		{Key: "last_7_days", Start: last7},
		{Key: lastNKey, Start: lastN},
	})
	if err != nil {
		return nil, err
	}
	return &CreatorStatsData{
		Days:           days,
		GeneratedAt:    now,
		LastNDaysLabel: lastNKey,
		Fans: map[string]int64{
			"total": fansTotal, "today": fansToday, "yesterday": fansYesterday, "last_7_days": fansLast7, lastNKey: fansLastN,
		},
		PostTotals:   postTotals,
		Interactions: interactions,
	}, nil
}

func (r CreatorRepository) EarningsLog(ctx context.Context, userID int64, page, limit int, logType string) (int64, []CreatorEarningsLogBundle, error) {
	query := r.db.WithContext(ctx).Model(&domain.CreatorEarningsLog{}).Where("user_id = ?", userID)
	if strings.TrimSpace(logType) != "" {
		query = query.Where("type = ?", logType)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	var logs []domain.CreatorEarningsLog
	if err := query.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&logs).Error; err != nil {
		return 0, nil, err
	}
	users, err := r.usersByID(ctx, earningBuyerIDs(logs))
	if err != nil {
		return 0, nil, err
	}
	posts, err := r.postsByID(ctx, earningPostSourceIDs(logs))
	if err != nil {
		return 0, nil, err
	}
	out := make([]CreatorEarningsLogBundle, 0, len(logs))
	for _, log := range logs {
		var buyer *domain.User
		if log.BuyerID != nil {
			buyer = users[*log.BuyerID]
		}
		var source *domain.Post
		if log.SourceID != nil && log.SourceType != nil && *log.SourceType == "post" {
			source = posts[*log.SourceID]
		}
		out = append(out, CreatorEarningsLogBundle{Log: log, Buyer: buyer, Source: source})
	}
	return total, out, nil
}

func (r CreatorRepository) PaidContent(ctx context.Context, userID int64, page, limit int) (int64, []PaidContentBundle, error) {
	query := r.db.WithContext(ctx).Model(&domain.Post{}).
		Joins("JOIN post_payment_settings ON post_payment_settings.post_id = posts.id AND post_payment_settings.enabled = ?", true).
		Where("posts.user_id = ? AND posts.is_draft = ?", userID, false)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	var posts []domain.Post
	if err := query.Select("posts.*").Order("posts.created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&posts).Error; err != nil {
		return 0, nil, err
	}
	postIDs := paidPostIDs(posts)
	payments, err := r.paymentSettingsByPostID(ctx, postIDs)
	if err != nil {
		return 0, nil, err
	}
	covers, err := r.postCovers(ctx, postIDs)
	if err != nil {
		return 0, nil, err
	}
	sales, err := r.salesStats(ctx, postIDs)
	if err != nil {
		return 0, nil, err
	}
	out := make([]PaidContentBundle, 0, len(posts))
	for _, post := range posts {
		stat := sales[post.ID]
		out = append(out, PaidContentBundle{Post: post, Payment: payments[post.ID], Cover: covers[post.ID], SalesCount: stat.Count, TotalRevenue: stat.Total})
	}
	return total, out, nil
}

func (r CreatorRepository) WithdrawToPoints(ctx context.Context, userID int64, amount float64, bonusAmount float64) (float64, float64, error) {
	if !r.cfg.WithdrawEnabled {
		return 0, 0, ErrCreatorWithdrawClosed
	}
	if amount <= 0 || math.IsNaN(amount) {
		return 0, 0, ErrCreatorAmountInvalid
	}
	if bonusAmount < 0 || math.IsNaN(bonusAmount) {
		bonusAmount = 0
	}
	bonusAmount = mathRound2(bonusAmount)
	if amount < r.cfg.MinWithdrawAmount {
		return 0, 0, ErrCreatorBelowMinimum
	}
	var newEarningsBalance float64
	var newPointsBalance float64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		earnings, err := getOrCreateCreatorEarningsTx(ctx, tx, userID)
		if err != nil {
			return err
		}
		if earnings.Balance < amount {
			return ErrCreatorBalanceLow
		}
		newEarningsBalance = mathRound2(earnings.Balance - amount)
		newWithdrawn := mathRound2(earnings.WithdrawnAmount + amount)
		if err := tx.Model(&domain.CreatorEarnings{}).Where("user_id = ?", userID).Updates(map[string]any{
			"balance":          newEarningsBalance,
			"withdrawn_amount": newWithdrawn,
		}).Error; err != nil {
			return err
		}
		reason := fmt.Sprintf("\u63d0\u73b0 %.2f \u6708\u5e01\u5230\u4f59\u989d", amount)
		if err := tx.Create(&domain.CreatorEarningsLog{UserID: userID, EarningsID: earnings.ID, Amount: -amount, BalanceAfter: newEarningsBalance, Type: "withdraw", Reason: &reason}).Error; err != nil {
			return err
		}
		points, err := getOrCreateUserPointsTx(ctx, tx, userID)
		if err != nil {
			return err
		}
		creditAmount := mathRound2(amount + bonusAmount)
		newPointsBalance = mathRound2(points.Points + creditAmount)
		if err := tx.Model(&domain.UserPoints{}).Where("user_id = ?", userID).Update("points", newPointsBalance).Error; err != nil {
			return err
		}
		pointsReason := "\u4ece\u521b\u4f5c\u8005\u6536\u76ca\u63d0\u73b0"
		if bonusAmount > 0 {
			pointsReason = fmt.Sprintf("\u4ece\u521b\u4f5c\u8005\u6536\u76ca\u63d0\u73b0\uff0c\u542b\u6210\u5c31\u52a0\u6210 %.2f", bonusAmount)
		}
		return tx.Create(&domain.PointsLog{UserID: userID, Amount: creditAmount, BalanceAfter: newPointsBalance, Type: PointsLogTypeWithdrawFromEarnings, Reason: &pointsReason}).Error
	})
	return newEarningsBalance, newPointsBalance, err
}

func (r CreatorRepository) CalculateExtendedEarnings(ctx context.Context, userID int64, start, end time.Time) (*ExtendedEarnings, error) {
	out := &ExtendedEarnings{Enabled: r.cfg.ExtendedEarningsEnabled, Rates: r.cfg.EarningsRates, DailyCap: r.cfg.DailyExtendedCap}
	if !r.cfg.ExtendedEarningsEnabled {
		return out, nil
	}
	postIDs, err := r.userPostIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	var viewCount, likeCount, collectCount, commentCount int64
	if len(postIDs) > 0 {
		viewCount, err = r.countBrowsingByPosts(ctx, userID, start, &end)
		if err != nil {
			return nil, err
		}
		likeCount, err = r.countLikesByPosts(ctx, postIDs, start, &end)
		if err != nil {
			return nil, err
		}
		collectCount, err = r.countCollectionsByPosts(ctx, postIDs, start, &end)
		if err != nil {
			return nil, err
		}
		commentCount, err = r.countCommentsByPosts(ctx, postIDs, start, &end)
		if err != nil {
			return nil, err
		}
	}
	followerCount, err := r.countFollows(ctx, userID, &start, &end)
	if err != nil {
		return nil, err
	}
	out.Views = CountEarnings{Count: viewCount, Earnings: mathRound2(float64(viewCount) * r.cfg.EarningsRates.PerView)}
	out.Likes = CountEarnings{Count: likeCount, Earnings: mathRound2(float64(likeCount) * r.cfg.EarningsRates.PerLike)}
	out.Collects = CountEarnings{Count: collectCount, Earnings: mathRound2(float64(collectCount) * r.cfg.EarningsRates.PerCollect)}
	out.Comments = CountEarnings{Count: commentCount, Earnings: mathRound2(float64(commentCount) * r.cfg.EarningsRates.PerComment)}
	out.Followers = CountEarnings{Count: followerCount, Earnings: mathRound2(float64(followerCount) * r.cfg.EarningsRates.PerFollower)}
	total := mathRound2(out.Views.Earnings + out.Likes.Earnings + out.Collects.Earnings + out.Comments.Earnings + out.Followers.Earnings)
	if r.cfg.DailyExtendedCap > 0 && total > r.cfg.DailyExtendedCap {
		total = r.cfg.DailyExtendedCap
	}
	out.Total = total
	return out, nil
}

func (r CreatorRepository) ClaimExtendedEarnings(ctx context.Context, userID int64) (*ClaimExtendedResult, error) {
	if !r.cfg.ExtendedEarningsEnabled {
		return &ClaimExtendedResult{Success: false, Message: "\u6269\u5c55\u6536\u76ca\u529f\u80fd\u672a\u542f\u7528"}, nil
	}
	today := dayStart(time.Now())
	tomorrow := today.AddDate(0, 0, 1)
	var existing int64
	if err := r.db.WithContext(ctx).Model(&domain.CreatorEarningsLog{}).Where("user_id = ? AND type = ? AND created_at >= ? AND created_at < ?", userID, "extended_daily", today, tomorrow).Count(&existing).Error; err != nil {
		return nil, err
	}
	if existing > 0 {
		return &ClaimExtendedResult{Success: false, Message: "\u4eca\u65e5\u6fc0\u52b1\u5956\u52b1\u5df2\u9886\u53d6", AlreadyClaimed: true}, nil
	}
	extended, err := r.CalculateExtendedEarnings(ctx, userID, today, tomorrow)
	if err != nil {
		return nil, err
	}
	if extended.Total <= 0 {
		return &ClaimExtendedResult{Success: false, Message: "\u4eca\u65e5\u6682\u65e0\u53ef\u9886\u53d6\u7684\u6fc0\u52b1\u5956\u52b1", NoEarnings: true, Earnings: *extended}, nil
	}
	details := extendedDetails(*extended)
	var newBalance float64
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		earnings, err := getOrCreateCreatorEarningsTx(ctx, tx, userID)
		if err != nil {
			return err
		}
		newBalance = mathRound2(earnings.Balance + extended.Total)
		newTotal := mathRound2(earnings.TotalEarnings + extended.Total)
		if err := tx.Model(&domain.CreatorEarnings{}).Where("user_id = ?", userID).Updates(map[string]any{"balance": newBalance, "total_earnings": newTotal}).Error; err != nil {
			return err
		}
		sourceType := "incentive"
		reason := "\u4eca\u65e5\u6fc0\u52b1\u5956\u52b1: " + strings.Join(details, "\u3001")
		return tx.Create(&domain.CreatorEarningsLog{UserID: userID, EarningsID: earnings.ID, Amount: extended.Total, BalanceAfter: newBalance, Type: "extended_daily", SourceType: &sourceType, Reason: &reason}).Error
	})
	if err != nil {
		return nil, err
	}
	return &ClaimExtendedResult{Success: true, Message: "\u6fc0\u52b1\u5956\u52b1\u9886\u53d6\u6210\u529f", Earnings: *extended, NewBalance: newBalance, Details: details}, nil
}

func (r CreatorRepository) QualityRewards(ctx context.Context, userID int64, page, limit int) (int64, float64, []QualityRewardBundle, []QualityRewardStats, error) {
	query := r.db.WithContext(ctx).Model(&domain.CreatorEarningsLog{}).Where("user_id = ? AND type = ?", userID, "quality_reward")
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, 0, nil, nil, err
	}
	var sum struct{ Total float64 }
	if err := r.db.WithContext(ctx).Model(&domain.CreatorEarningsLog{}).Select("COALESCE(SUM(amount),0) AS total").Where("user_id = ? AND type = ?", userID, "quality_reward").Scan(&sum).Error; err != nil {
		return 0, 0, nil, nil, err
	}
	var logs []domain.CreatorEarningsLog
	if err := query.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&logs).Error; err != nil {
		return 0, 0, nil, nil, err
	}
	posts, err := r.qualityPostsByID(ctx, earningPostSourceIDs(logs))
	if err != nil {
		return 0, 0, nil, nil, err
	}
	out := make([]QualityRewardBundle, 0, len(logs))
	for _, log := range logs {
		var post *QualityRewardPost
		if log.SourceID != nil {
			post = posts[*log.SourceID]
		}
		out = append(out, QualityRewardBundle{Log: log, Post: post})
	}
	stats, err := r.qualityRewardStats(ctx, userID)
	return total, sum.Total, out, stats, err
}
