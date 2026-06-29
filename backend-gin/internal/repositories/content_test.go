package repositories

import (
	"context"
	"testing"
	"time"

	"yuem-go/backend-gin/internal/domain"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestPostFromCreateInputDefaultsQualityLevel(t *testing.T) {
	post := postFromCreateInput(CreatePostInput{
		UserID:     2,
		Title:      "ds",
		Content:    "dde",
		Type:       PostTypeImage,
		IsDraft:    false,
		Visibility: VisibilityPublic,
	})

	if post.QualityLevel != PostQualityNone {
		t.Fatalf("QualityLevel = %q, want %q", post.QualityLevel, PostQualityNone)
	}
}

func TestPopularityScoreBoostsOriginalIncentive(t *testing.T) {
	plain := domain.Post{LikeCount: 10, CollectCount: 1, CommentCount: 1, ViewCount: 100}
	reward := 5.0
	rewarded := plain
	rewarded.QualityReward = &reward

	if popularityScore(rewarded) <= popularityScore(plain) {
		t.Fatalf("rewarded popularity score should be higher")
	}
}

func TestNormalizePostImageInputsForcesSortedCoverPolicy(t *testing.T) {
	images := []PostImageInput{
		{URL: "second", IsFreePreview: false, IsProtected: true, SortOrder: 2},
		{URL: "cover", IsFreePreview: false, IsProtected: true, SortOrder: 1},
		{URL: "third", IsFreePreview: false, IsProtected: true, SortOrder: 3},
	}

	normalized := NormalizePostImageInputs(images)
	if len(normalized) != 3 || normalized[0].URL != "cover" {
		t.Fatalf("normalized order = %#v", normalized)
	}
	if !normalized[0].IsFreePreview || normalized[0].IsProtected {
		t.Fatalf("cover flags = %#v", normalized[0])
	}
	if normalized[1].IsFreePreview || !normalized[1].IsProtected {
		t.Fatalf("second flags changed unexpectedly = %#v", normalized[1])
	}
	free, paid := PostImageAccessCounts(normalized)
	if free != 1 || paid != 2 {
		t.Fatalf("access counts = free %d paid %d", free, paid)
	}
}

func TestCreatePostBindsTemporaryUploadAssets(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.Post{}, &domain.PostImage{}, &domain.UploadAsset{}, &domain.ImageWatermarkTrace{}); err != nil {
		t.Fatal(err)
	}
	expires := time.Now().Add(time.Hour)
	asset := domain.UploadAsset{
		UserID:    7,
		Purpose:   "ai_analysis",
		Kind:      "image",
		URL:       "/api/file/images/a.webp",
		Storage:   "local",
		Status:    "temp",
		ExpiresAt: &expires,
		CreatedAt: time.Now(),
	}
	if err := db.Create(&asset).Error; err != nil {
		t.Fatal(err)
	}
	postID, err := NewContentRepository(db).CreatePost(context.Background(), CreatePostInput{
		UserID:          7,
		Title:           "title",
		Content:         "body",
		Type:            PostTypeImage,
		Images:          []PostImageInput{{URL: asset.URL, IsFreePreview: true, SortOrder: 1}},
		Visibility:      VisibilityPublic,
		HoldSideEffects: true,
	})
	if err != nil {
		t.Fatalf("CreatePost() error = %v", err)
	}
	var saved domain.UploadAsset
	if err := db.First(&saved, asset.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Status != "bound" || saved.BoundPostID == nil || *saved.BoundPostID != postID || saved.ExpiresAt != nil {
		t.Fatalf("upload asset after create = status %q post %v expires %v", saved.Status, saved.BoundPostID, saved.ExpiresAt)
	}
}

func TestCreateCommentReplyUsesParentPostWhenRequestPostDiffers(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.User{}, &domain.Post{}, &domain.Comment{}, &domain.Like{}, &domain.Notification{}); err != nil {
		t.Fatal(err)
	}
	author := domain.User{ID: 11, UserID: "author-parent-post", Nickname: "Author", IsActive: true}
	parentAuthor := domain.User{ID: 12, UserID: "parent-author", Nickname: "Parent", IsActive: true}
	replyAuthor := domain.User{ID: 13, UserID: "reply-author", Nickname: "Reply", IsActive: true}
	correctPost := domain.Post{ID: 21, UserID: author.ID, Title: "correct", Type: PostTypeImage, Visibility: VisibilityPublic}
	wrongPost := domain.Post{ID: 22, UserID: author.ID, Title: "wrong", Type: PostTypeImage, Visibility: VisibilityPublic}
	parent := domain.Comment{ID: 31, PostID: correctPost.ID, UserID: parentAuthor.ID, Content: "parent", IsPublic: true, AuditStatus: 1}
	for _, row := range []any{&author, &parentAuthor, &replyAuthor, &correctPost, &wrongPost, &parent} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}

	result, err := NewContentRepository(db).CreateComment(context.Background(), replyAuthor.ID, wrongPost.ID, &parent.ID, "reply")
	if err != nil {
		t.Fatalf("CreateComment() error = %v", err)
	}
	if result.Comment.Comment.PostID != correctPost.ID || result.Comment.Comment.ParentID == nil || *result.Comment.Comment.ParentID != parent.ID {
		t.Fatalf("created reply location = post %d parent %v, want post %d parent %d", result.Comment.Comment.PostID, result.Comment.Comment.ParentID, correctPost.ID, parent.ID)
	}
	var updatedCorrect domain.Post
	if err := db.First(&updatedCorrect, correctPost.ID).Error; err != nil {
		t.Fatal(err)
	}
	var updatedWrong domain.Post
	if err := db.First(&updatedWrong, wrongPost.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updatedCorrect.CommentCount != 1 || updatedWrong.CommentCount != 0 {
		t.Fatalf("comment counts = correct %d wrong %d, want 1/0", updatedCorrect.CommentCount, updatedWrong.CommentCount)
	}
	var notification domain.Notification
	if err := db.First(&notification).Error; err != nil {
		t.Fatal(err)
	}
	if notification.TargetID == nil || *notification.TargetID != correctPost.ID || notification.CommentID == nil || *notification.CommentID != result.Comment.Comment.ID {
		t.Fatalf("notification target/comment = %v/%v, want post %d comment %d", notification.TargetID, notification.CommentID, correctPost.ID, result.Comment.Comment.ID)
	}
}

func TestDeleteCommentAfterAIModerationDeletesThreadAndUpdatesCounters(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.Post{}, &domain.Comment{}, &domain.Like{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	post := domain.Post{ID: 10, UserID: 1, Title: "post", Type: PostTypeImage, Visibility: VisibilityPublic, CommentCount: 3, CreatedAt: now}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	parent := domain.Comment{ID: 20, PostID: post.ID, UserID: 2, Content: "bad", IsPublic: true, AuditStatus: 1, CreatedAt: now}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	replyParent := parent.ID
	publicReply := domain.Comment{ID: 21, PostID: post.ID, UserID: 3, ParentID: &replyParent, Content: "reply", IsPublic: true, AuditStatus: 1, CreatedAt: now}
	hiddenReply := domain.Comment{ID: 22, PostID: post.ID, UserID: 4, ParentID: &replyParent, Content: "hidden", IsPublic: false, AuditStatus: 2, CreatedAt: now}
	if err := db.Create(&publicReply).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&hiddenReply).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.Like{UserID: 9, TargetType: 2, TargetID: parent.ID, CreatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.Like{UserID: 9, TargetType: 2, TargetID: publicReply.ID, CreatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}

	if err := NewContentRepository(db).DeleteCommentAfterAIModeration(context.Background(), parent.ID); err != nil {
		t.Fatalf("DeleteCommentAfterAIModeration() error = %v", err)
	}
	var remainingComments int64
	if err := db.Model(&domain.Comment{}).Count(&remainingComments).Error; err != nil {
		t.Fatal(err)
	}
	if remainingComments != 0 {
		t.Fatalf("remaining comments = %d, want 0", remainingComments)
	}
	var remainingLikes int64
	if err := db.Model(&domain.Like{}).Count(&remainingLikes).Error; err != nil {
		t.Fatal(err)
	}
	if remainingLikes != 0 {
		t.Fatalf("remaining likes = %d, want 0", remainingLikes)
	}
	var updated domain.Post
	if err := db.First(&updated, post.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.CommentCount != 1 {
		t.Fatalf("comment_count = %d, want 1", updated.CommentCount)
	}
}
