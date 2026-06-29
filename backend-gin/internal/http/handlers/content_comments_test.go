package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

func TestCommentRepliesIncludesNestedReplyCount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.User{}, &domain.Post{}, &domain.Comment{}, &domain.Like{}); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	author := domain.User{ID: 1, UserID: "author", Nickname: "Author", IsActive: true}
	replier := domain.User{ID: 2, UserID: "replier", Nickname: "Replier", IsActive: true}
	post := domain.Post{ID: 10, UserID: author.ID, Title: "post", Type: 1, Visibility: "public", CreatedAt: now}
	root := domain.Comment{ID: 20, PostID: post.ID, UserID: author.ID, Content: "root", IsPublic: true, AuditStatus: 1, CreatedAt: now}
	childParent := root.ID
	child := domain.Comment{ID: 21, PostID: post.ID, UserID: replier.ID, ParentID: &childParent, Content: "child", IsPublic: true, AuditStatus: 1, CreatedAt: now.Add(time.Minute)}
	grandchildParent := child.ID
	grandchild := domain.Comment{ID: 22, PostID: post.ID, UserID: author.ID, ParentID: &grandchildParent, Content: "grandchild", IsPublic: true, AuditStatus: 1, CreatedAt: now.Add(2 * time.Minute)}
	for _, row := range []any{&author, &replier, &post, &root, &child, &grandchild} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Params = gin.Params{{Key: "id", Value: "20"}}
	context.Request = httptest.NewRequest(http.MethodGet, "/api/comments/20/replies", nil)
	NativeHandlers{DB: db}.CommentReplies(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var responseBody struct {
		Data struct {
			Comments []struct {
				ID         int64  `json:"id"`
				ReplyCount int64  `json:"reply_count"`
				ParentID   *int64 `json:"parent_id"`
			} `json:"comments"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(responseBody.Data.Comments) != 1 {
		t.Fatalf("comments len = %d, want 1", len(responseBody.Data.Comments))
	}
	got := responseBody.Data.Comments[0]
	if got.ID != child.ID || got.ParentID == nil || *got.ParentID != root.ID || got.ReplyCount != 1 {
		t.Fatalf("reply payload = %+v, want child %d parent %d reply_count 1", got, child.ID, root.ID)
	}
}
