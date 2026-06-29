package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

func (s *QueueService) processBatchNoteCreate(ctx context.Context, task *asynq.Task) error {
	var payload batchNoteTaskPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return err
	}
	post, err := s.createBatchPost(ctx, payload)
	if err != nil {
		return err
	}
	_, _ = s.EnqueueVideoTranscodingForPost(ctx, post.ID)
	if !payload.IsDraft && s.AIPostAutoCommentReady() {
		_, _ = s.EnqueueAIPostAutoComment(ctx, post.ID)
	}
	result, _ := json.Marshal(map[string]any{"postId": post.ID, "noteIndex": payload.NoteIndex})
	_, _ = task.ResultWriter().Write(result)
	return nil
}

func (s *QueueService) createBatchPost(ctx context.Context, payload batchNoteTaskPayload) (domain.Post, error) {
	var post domain.Post
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		post = domain.Post{
			UserID:       payload.UserID,
			Title:        payload.Title,
			Content:      payload.Content,
			Type:         payload.PostType,
			IsDraft:      payload.IsDraft,
			Visibility:   "public",
			QualityLevel: "none",
		}
		if post.Type == 0 {
			post.Type = 1
		}
		if err := tx.Create(&post).Error; err != nil {
			return err
		}
		if post.Type == 1 {
			for idx, file := range payload.Files {
				url := s.cfg.Upload.LocalBase + file.Path
				if err := tx.Create(&domain.PostImage{PostID: post.ID, ImageURL: url, IsFreePreview: true, SortOrder: idx + 1}).Error; err != nil {
					return err
				}
			}
		} else if len(payload.Files) > 0 {
			cover := payload.CoverURL
			url := s.cfg.Upload.LocalBase + payload.Files[0].Path
			if err := tx.Create(&domain.PostVideo{PostID: post.ID, VideoURL: url, CoverURL: &cover}).Error; err != nil {
				return err
			}
		}
		for _, name := range payload.Tags {
			tagID, err := queueGetOrCreateTag(ctx, tx, name)
			if err != nil {
				return err
			}
			if tagID == 0 {
				continue
			}
			if err := tx.Create(&domain.PostTag{PostID: post.ID, TagID: tagID}).Error; err != nil {
				return err
			}
			_ = tx.Model(&domain.Tag{}).Where("id = ?", tagID).Update("use_count", gorm.Expr("use_count + 1")).Error
		}
		return nil
	})
	return post, err
}

func queueGetOrCreateTag(ctx context.Context, tx *gorm.DB, name string) (int, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, nil
	}
	var tag domain.Tag
	err := tx.WithContext(ctx).Where("name = ?", name).First(&tag).Error
	if err == nil {
		return tag.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	tag = domain.Tag{Name: name}
	if err := tx.WithContext(ctx).Create(&tag).Error; err != nil {
		return 0, err
	}
	return tag.ID, nil
}

func taskInfoMap(task *asynq.TaskInfo, stateHint string) map[string]any {
	state := stateHint
	if state == "" && task.State.String() != "" {
		state = task.State.String()
	}
	state = queueStatusLabel(state)
	queuedAt := taskEnqueuedTime(task)
	totalSeconds := durationSeconds(queuedAt, task.CompletedAt)
	detail := jsonAnyBytes(task.Payload)
	return map[string]any{
		"id":            task.ID,
		"name":          task.Type,
		"type":          task.Type,
		"taskType":      task.Type,
		"queue":         task.Queue,
		"queueType":     queueKind(task.Queue),
		"queueLabel":    queueLabel(task.Queue),
		"data":          detail,
		"detail":        detail,
		"timestamp":     timeMillis(queuedAt),
		"queuedAt":      timeMillis(queuedAt),
		"enqueuedAt":    timeMillis(queuedAt),
		"nextProcessAt": nullableTimeMillis(task.NextProcessAt),
		"lastFailedAt":  nullableTimeMillis(task.LastFailedAt),
		"processedOn":   nil,
		"startedAt":     nil,
		"finishedOn":    nullableTimeMillis(task.CompletedAt),
		"completedAt":   nullableTimeMillis(task.CompletedAt),
		"durationMs":    durationMillis(queuedAt, task.CompletedAt),
		"waitMs":        nil,
		"processMs":     nil,
		"attempts":      task.Retried,
		"attemptsMade":  task.Retried,
		"maxRetry":      task.MaxRetry,
		"failedReason":  emptyStringToNil(task.LastErr),
		"returnValue":   jsonAnyBytes(task.Result),
		"headers":       task.Headers,
		"isOrphaned":    task.IsOrphaned,
		"state":         state,
		"status":        state,
		"progress":      nil,
		"stacktrace":    []string{},
		"timing":        map[string]any{"waitTimeSeconds": nil, "processTimeSeconds": nil, "totalTimeSeconds": totalSeconds},
	}
}

func (s *QueueService) attachQueueEvents(ctx context.Context, item map[string]any) {
	if s == nil || item == nil {
		return
	}
	taskID, _ := item["id"].(string)
	if taskID == "" {
		return
	}
	queue, _ := item["queue"].(string)
	events := s.RecentEvents(ctx, queue, 200)
	filtered := make([]map[string]any, 0, 4)
	var enqueuedAt, startedAt, finishedAt int64
	for _, event := range events {
		if fmt.Sprint(event["task_id"]) != taskID {
			continue
		}
		filtered = append(filtered, event)
		at := int64FromAny(event["at"])
		switch fmt.Sprint(event["event"]) {
		case "enqueued", "duplicate":
			if enqueuedAt == 0 || at < enqueuedAt {
				enqueuedAt = at
			}
		case "started":
			if startedAt == 0 || at < startedAt {
				startedAt = at
			}
		case "completed", "failed":
			if at > finishedAt {
				finishedAt = at
			}
		}
	}
	if len(filtered) == 0 {
		return
	}
	item["events"] = filtered
	if enqueuedAt > 0 {
		item["enqueuedAt"] = enqueuedAt
		item["queuedAt"] = enqueuedAt
	}
	if startedAt > 0 {
		item["startedAt"] = startedAt
		item["processedOn"] = startedAt
	}
	if finishedAt > 0 {
		item["completedAt"] = finishedAt
		item["finishedOn"] = finishedAt
	}
	if enqueuedAt > 0 && startedAt > 0 && startedAt >= enqueuedAt {
		item["waitMs"] = startedAt - enqueuedAt
	}
	if startedAt > 0 && finishedAt > 0 && finishedAt >= startedAt {
		item["processMs"] = finishedAt - startedAt
		item["durationMs"] = finishedAt - startedAt
	}
}
