package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type DesiredIndex struct {
	Table   string
	Name    string
	Columns []string
	Unique  bool
}

type SchemaAuditIssue struct {
	Kind                string               `json:"kind"`
	Table               string               `json:"table"`
	Name                string               `json:"name,omitempty"`
	Columns             []string             `json:"columns,omitempty"`
	Message             string               `json:"message"`
	Repair              string               `json:"repair"`
	Repairable          bool                 `json:"repairable"`
	DuplicateGroupCount int64                `json:"duplicateGroupCount,omitempty"`
	DuplicateRowCount   int64                `json:"duplicateRowCount,omitempty"`
	DuplicateSamples    []DuplicateKeySample `json:"duplicateSamples,omitempty"`
}

type DuplicateKeySample struct {
	Values map[string]any `json:"values"`
	Count  int64          `json:"count"`
}

func DesiredIndexes() []DesiredIndex {
	return []DesiredIndex{
		{Table: "posts", Name: "idx_posts_user_draft_created", Columns: []string{"user_id", "is_draft", "created_at"}},
		{Table: "posts", Name: "idx_posts_user_draft_visibility_created", Columns: []string{"user_id", "is_draft", "visibility", "created_at"}},
		{Table: "posts", Name: "idx_posts_visibility_draft_created", Columns: []string{"visibility", "is_draft", "created_at"}},
		{Table: "posts", Name: "idx_posts_feed_filters_created", Columns: []string{"visibility", "is_draft", "category_id", "type", "created_at"}},
		{Table: "posts", Name: "idx_posts_search_filters_created", Columns: []string{"visibility", "is_draft", "type", "public_access_exempt", "created_at"}},
		{Table: "posts", Name: "idx_posts_public_access_exempt", Columns: []string{"public_access_exempt"}},
		{Table: "posts", Name: "idx_posts_audit_status", Columns: []string{"audit_status"}},
		{Table: "users", Name: "idx_users_user_id", Columns: []string{"user_id"}},
		{Table: "users", Name: "idx_users_nickname", Columns: []string{"nickname"}},
		{Table: "users", Name: "idx_users_created_at", Columns: []string{"created_at"}},
		{Table: "tags", Name: "idx_tags_name", Columns: []string{"name"}},
		{Table: "post_images", Name: "idx_post_images_post_id", Columns: []string{"post_id"}},
		{Table: "post_images", Name: "idx_post_images_post_sort", Columns: []string{"post_id", "sort_order"}},
		{Table: "post_images", Name: "idx_post_images_image_url", Columns: []string{"image_url"}},
		{Table: "post_images", Name: "idx_post_images_watermark_trace", Columns: []string{"watermark_trace_token"}},
		{Table: "post_videos", Name: "idx_post_videos_post_id", Columns: []string{"post_id"}},
		{Table: "post_videos", Name: "idx_post_videos_cover_url", Columns: []string{"cover_url"}},
		{Table: "post_videos", Name: "idx_post_videos_video_url", Columns: []string{"video_url"}},
		{Table: "post_videos", Name: "idx_post_videos_dash_url", Columns: []string{"dash_url"}},
		{Table: "post_videos", Name: "idx_post_videos_preview_video_url", Columns: []string{"preview_video_url"}},
		{Table: "post_attachments", Name: "idx_post_attachments_post_id", Columns: []string{"post_id"}},
		{Table: "upload_assets", Name: "idx_upload_assets_url", Columns: []string{"url"}, Unique: true},
		{Table: "upload_assets", Name: "idx_upload_assets_status_expires", Columns: []string{"status", "expires_at"}},
		{Table: "upload_assets", Name: "idx_upload_assets_user_status_created", Columns: []string{"user_id", "status", "created_at"}},
		{Table: "upload_assets", Name: "idx_upload_assets_bound_post", Columns: []string{"bound_post_id"}},
		{Table: "file_recycle_items", Name: "idx_file_recycle_group", Columns: []string{"group_id"}},
		{Table: "file_recycle_items", Name: "idx_file_recycle_resource", Columns: []string{"resource_type", "resource_id"}},
		{Table: "file_recycle_items", Name: "idx_file_recycle_post", Columns: []string{"post_id"}},
		{Table: "file_recycle_items", Name: "idx_file_recycle_user", Columns: []string{"user_id"}},
		{Table: "file_recycle_items", Name: "idx_file_recycle_kind", Columns: []string{"kind"}},
		{Table: "file_recycle_items", Name: "idx_file_recycle_deleted", Columns: []string{"deleted_at"}},
		{Table: "file_recycle_items", Name: "idx_file_recycle_status_purge", Columns: []string{"status", "purge_after"}},
		{Table: "file_recycle_items", Name: "idx_file_recycle_purge_deleted_id", Columns: []string{"purge_after", "deleted_at", "id"}},
		{Table: "post_payment_settings", Name: "idx_post_payment_settings_post_id", Columns: []string{"post_id"}},
		{Table: "comments", Name: "idx_comments_post_created", Columns: []string{"post_id", "created_at"}},
		{Table: "comments", Name: "idx_comments_parent_created", Columns: []string{"parent_id", "created_at"}},
		{Table: "comments", Name: "idx_comments_post_parent_public_created", Columns: []string{"post_id", "parent_id", "is_public", "created_at"}},
		{Table: "likes", Name: "idx_likes_user_type_target", Columns: []string{"user_id", "target_type", "target_id"}},
		{Table: "likes", Name: "idx_likes_user_type_created", Columns: []string{"user_id", "target_type", "created_at"}},
		{Table: "likes", Name: "idx_likes_target_type_target_created", Columns: []string{"target_type", "target_id", "created_at"}},
		{Table: "collections", Name: "idx_collections_user_post", Columns: []string{"user_id", "post_id"}},
		{Table: "collections", Name: "idx_collections_user_created", Columns: []string{"user_id", "created_at"}},
		{Table: "collections", Name: "idx_collections_post_created", Columns: []string{"post_id", "created_at"}},
		{Table: "follows", Name: "idx_follows_follower_following", Columns: []string{"follower_id", "following_id"}},
		{Table: "follows", Name: "idx_follows_following_follower", Columns: []string{"following_id", "follower_id"}},
		{Table: "follows", Name: "idx_follows_follower_created", Columns: []string{"follower_id", "created_at"}},
		{Table: "follows", Name: "idx_follows_following_created", Columns: []string{"following_id", "created_at"}},
		{Table: "blacklist", Name: "idx_blacklist_blocker_blocked", Columns: []string{"blocker_id", "blocked_id"}},
		{Table: "blacklist", Name: "idx_blacklist_blocked_blocker", Columns: []string{"blocked_id", "blocker_id"}},
		{Table: "post_tags", Name: "idx_post_tags_post_tag", Columns: []string{"post_id", "tag_id"}},
		{Table: "post_tags", Name: "idx_post_tags_tag_post", Columns: []string{"tag_id", "post_id"}},
		{Table: "dislikes", Name: "idx_dislikes_user_post", Columns: []string{"user_id", "post_id"}},
		{Table: "browsing_history", Name: "idx_browsing_history_user_updated", Columns: []string{"user_id", "updated_at"}},
		{Table: "browsing_history", Name: "idx_browsing_history_post_created", Columns: []string{"post_id", "created_at"}},
		{Table: "user_purchased_content", Name: "idx_user_purchased_content_user_post", Columns: []string{"user_id", "post_id"}, Unique: true},
		{Table: "user_purchased_content", Name: "idx_user_purchased_content_post_purchased", Columns: []string{"post_id", "purchased_at"}},
		{Table: "content_purchase_intents", Name: "idx_content_purchase_intents_user_post", Columns: []string{"user_id", "post_id"}, Unique: true},
		{Table: "content_purchase_intents", Name: "idx_content_purchase_intents_status_updated", Columns: []string{"status", "updated_at"}},
		{Table: "external_balance_accounts", Name: "idx_external_balance_accounts_user", Columns: []string{"user_id"}, Unique: true},
		{Table: "external_balance_transactions", Name: "idx_external_balance_transactions_operation", Columns: []string{"operation_key"}, Unique: true},
		{Table: "external_balance_transactions", Name: "idx_external_balance_transactions_user_created", Columns: []string{"user_id", "created_at"}},
		{Table: "external_balance_transactions", Name: "idx_external_balance_transactions_oauth_created", Columns: []string{"oauth2_id", "created_at"}},
		{Table: "external_balance_transactions", Name: "idx_external_balance_transactions_status_updated", Columns: []string{"status", "updated_at"}},
		{Table: "external_balance_transactions", Name: "idx_external_balance_transactions_post_created", Columns: []string{"post_id", "created_at"}},
		{Table: "creator_earnings_log", Name: "idx_creator_earnings_log_user_created", Columns: []string{"user_id", "created_at"}},
		{Table: "creator_earnings_log", Name: "idx_creator_earnings_log_user_type_created", Columns: []string{"user_id", "type", "created_at"}},
		{Table: "points_log", Name: "idx_points_log_created_at", Columns: []string{"created_at"}},
		{Table: "points_log", Name: "idx_points_log_user_created", Columns: []string{"user_id", "created_at"}},
		{Table: "points_log", Name: "idx_points_log_type_created", Columns: []string{"type", "created_at"}},
		{Table: "points_log", Name: "idx_points_log_post_created", Columns: []string{"post_id", "created_at"}},
		{Table: "points_log", Name: "idx_points_log_purchase_role", Columns: []string{"purchase_id", "entry_role"}},
		{Table: "audit", Name: "idx_audit_type_created", Columns: []string{"type", "created_at"}},
		{Table: "audit", Name: "idx_audit_type_status_created", Columns: []string{"type", "status", "created_at"}},
		{Table: "audit", Name: "idx_audit_user_type_created", Columns: []string{"user_id", "type", "created_at"}},
		{Table: "image_protection_jobs", Name: "idx_image_protection_jobs_job_id", Columns: []string{"job_id"}, Unique: true},
		{Table: "image_protection_jobs", Name: "idx_image_protection_jobs_user_status_created", Columns: []string{"user_id", "status", "created_at"}},
		{Table: "image_protection_jobs", Name: "idx_image_protection_jobs_post_user_status", Columns: []string{"post_id", "user_id", "status"}},
		{Table: "image_protection_jobs", Name: "idx_image_protection_jobs_post_kind_status", Columns: []string{"post_id", "package_kind", "status"}},
		{Table: "image_protection_jobs", Name: "idx_image_protection_jobs_expires", Columns: []string{"expires_at"}},
		{Table: "image_watermark_traces", Name: "idx_image_watermark_traces_token", Columns: []string{"token"}, Unique: true},
		{Table: "image_watermark_traces", Name: "idx_image_watermark_traces_short_code", Columns: []string{"short_code"}, Unique: true},
		{Table: "image_watermark_traces", Name: "idx_image_watermark_traces_type_created", Columns: []string{"trace_type", "created_at"}},
		{Table: "image_watermark_traces", Name: "idx_image_watermark_traces_user_created", Columns: []string{"user_id", "created_at"}},
		{Table: "image_watermark_traces", Name: "idx_image_watermark_traces_post", Columns: []string{"post_id"}},
		{Table: "image_watermark_traces", Name: "idx_image_watermark_traces_image", Columns: []string{"image_id"}},
		{Table: "image_watermark_traces", Name: "idx_image_watermark_traces_job_image", Columns: []string{"job_id", "image_id"}, Unique: true},
		{Table: "post_recommend_configs", Name: "idx_post_recommend_configs_post_active_window", Columns: []string{"post_id", "is_active", "start_time", "end_time"}},
		{Table: "recommend_configs", Name: "idx_recommend_configs_user_active", Columns: []string{"user_id", "is_active"}},
		{Table: "notifications", Name: "idx_notifications_user_read_created", Columns: []string{"user_id", "is_read", "created_at"}},
		{Table: "notifications", Name: "idx_notifications_user_created", Columns: []string{"user_id", "created_at"}},
		{Table: "notifications", Name: "idx_notifications_user_type_created", Columns: []string{"user_id", "type", "created_at"}},
		{Table: "system_notifications", Name: "idx_system_notifications_active_popup_created", Columns: []string{"is_active", "show_popup", "created_at"}},
		{Table: "system_notifications", Name: "idx_system_notifications_active_type_created", Columns: []string{"is_active", "type", "created_at"}},
		{Table: "system_notification_confirmations", Name: "idx_system_notification_confirmations_user_notification", Columns: []string{"user_id", "notification_id"}, Unique: true},
		{Table: "user_api_keys", Name: "idx_user_api_keys_api_key", Columns: []string{"api_key"}, Unique: true},
		{Table: "open_apis", Name: "idx_open_apis_api_key", Columns: []string{"api_key"}, Unique: true},
		{Table: "im_conversation_members", Name: "idx_im_conversation_members_user_conversation", Columns: []string{"user_id", "conversation_id"}},
		{Table: "im_conversation_members", Name: "idx_im_conversation_members_conversation_user", Columns: []string{"conversation_id", "user_id"}},
		{Table: "im_conversations", Name: "idx_im_conversations_updated", Columns: []string{"updated_at"}},
		{Table: "im_messages", Name: "idx_im_messages_conversation_id", Columns: []string{"conversation_id", "id"}},
		{Table: "im_messages", Name: "idx_im_messages_conversation_sender_id", Columns: []string{"conversation_id", "sender_id", "id"}},
		{Table: "im_messages", Name: "idx_im_messages_client_msg", Columns: []string{"conversation_id", "client_msg_id"}},
		{Table: "im_message_receipts", Name: "idx_im_message_receipts_message_user", Columns: []string{"message_id", "user_id"}},
		{Table: "im_message_receipts", Name: "idx_im_message_receipts_user_read_message", Columns: []string{"user_id", "read_at", "message_id"}},
		{Table: "announcements", Name: "idx_announcements_published_expires_created", Columns: []string{"is_published", "expires_at", "created_at"}},
		{Table: "system_settings", Name: "idx_system_settings_key", Columns: []string{"setting_key"}, Unique: true},
		{Table: "oauth_app_handoffs", Name: "idx_oauth_app_handoffs_code", Columns: []string{"code_hash"}, Unique: true},
		{Table: "oauth_app_handoffs", Name: "idx_oauth_app_handoffs_expires", Columns: []string{"expires_at"}},
		{Table: "oauth_app_handoffs", Name: "idx_oauth_app_handoffs_user_created", Columns: []string{"user_id", "created_at"}},
		{Table: "access_logs", Name: "idx_access_logs_created_at", Columns: []string{"created_at"}},
		{Table: "access_logs", Name: "idx_access_logs_behavior_created", Columns: []string{"behavior", "created_at"}},
		{Table: "access_logs", Name: "idx_access_logs_user_created", Columns: []string{"user_id", "created_at"}},
		{Table: "access_logs", Name: "idx_access_logs_principal_created", Columns: []string{"principal_type", "created_at"}},
		{Table: "access_logs", Name: "idx_access_logs_target_created", Columns: []string{"target_type", "target_id", "created_at"}},
		{Table: "access_logs", Name: "idx_access_logs_path_created", Columns: []string{"path", "created_at"}},
		{Table: "access_logs", Name: "idx_access_logs_ip_created", Columns: []string{"ip", "created_at"}},
		{Table: "security_audit_logs", Name: "idx_security_audit_created_at", Columns: []string{"created_at"}},
		{Table: "security_audit_logs", Name: "idx_security_audit_category_created", Columns: []string{"category", "created_at"}},
		{Table: "security_audit_logs", Name: "idx_security_audit_action_created", Columns: []string{"action", "created_at"}},
		{Table: "security_audit_logs", Name: "idx_security_audit_actor_created", Columns: []string{"actor_type", "actor_id", "created_at"}},
		{Table: "security_audit_logs", Name: "idx_security_audit_outcome_created", Columns: []string{"outcome", "created_at"}},
		{Table: "security_audit_logs", Name: "idx_security_audit_ip_created", Columns: []string{"ip", "created_at"}},
		{Table: "access_block_import_sources", Name: "idx_access_block_import_sources_url", Columns: []string{"url"}, Unique: true},
		{Table: "access_block_import_sources", Name: "idx_access_block_import_sources_enabled_next_sync", Columns: []string{"enabled", "next_sync_at"}},
		{Table: "access_block_rules", Name: "idx_access_block_rules_source_unique", Columns: []string{"import_source_id", "kind", "match_type", "pattern"}, Unique: true},
		{Table: "access_block_rules", Name: "idx_access_block_rules_import_source", Columns: []string{"import_source_id"}},
		{Table: "access_block_rules", Name: "idx_access_block_rules_enabled_priority_updated", Columns: []string{"enabled", "priority", "updated_at"}},
		{Table: "ai_generation_logs", Name: "idx_ai_generation_logs_job_id", Columns: []string{"job_id"}, Unique: true},
		{Table: "ai_generation_logs", Name: "idx_ai_generation_logs_created", Columns: []string{"created_at"}},
		{Table: "ai_generation_logs", Name: "idx_ai_generation_logs_actor_created", Columns: []string{"actor_type", "actor_id", "created_at"}},
		{Table: "ai_generation_logs", Name: "idx_ai_generation_logs_task_created", Columns: []string{"task_type", "created_at"}},
		{Table: "ai_jobs", Name: "idx_ai_jobs_job_id", Columns: []string{"job_id"}, Unique: true},
		{Table: "ai_jobs", Name: "idx_ai_jobs_created", Columns: []string{"created_at"}},
		{Table: "ai_jobs", Name: "idx_ai_jobs_request_status_created", Columns: []string{"request_hash", "status", "created_at"}},
		{Table: "ai_jobs", Name: "idx_ai_jobs_actor_status_created", Columns: []string{"task_type", "actor_type", "actor_id", "status", "created_at"}},
		{Table: "ai_moderation_logs", Name: "idx_ai_moderation_target", Columns: []string{"target_type", "target_id"}},
		{Table: "ai_moderation_logs", Name: "idx_ai_moderation_user_created", Columns: []string{"user_id", "created_at"}},
		{Table: "ai_moderation_logs", Name: "idx_ai_moderation_status_created", Columns: []string{"status", "created_at"}},
	}
}

func AuditSchema(ctx context.Context, db *gorm.DB) []SchemaAuditIssue {
	if db == nil {
		return []SchemaAuditIssue{{Kind: "database", Message: "database is not configured", Repair: "configure database"}}
	}
	issues := []SchemaAuditIssue{}
	for _, model := range AutoMigrateModels() {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(model); err != nil {
			continue
		}
		table := stmt.Schema.Table
		if !db.WithContext(ctx).Migrator().HasTable(model) {
			issues = append(issues, SchemaAuditIssue{Kind: "missing_table", Table: table, Message: "table is missing", Repair: "auto_migrate", Repairable: true})
			continue
		}
		for _, field := range stmt.Schema.Fields {
			if field.DBName == "" || db.WithContext(ctx).Migrator().HasColumn(model, field.DBName) {
				continue
			}
			issues = append(issues, SchemaAuditIssue{Kind: "missing_column", Table: table, Name: field.DBName, Message: "column is missing", Repair: "auto_migrate", Repairable: true})
		}
	}
	for _, index := range DesiredIndexes() {
		if !hasDatabaseIndex(ctx, db, index.Table, index.Name) {
			issues = append(issues, SchemaAuditIssue{Kind: "missing_index", Table: index.Table, Name: index.Name, Columns: index.Columns, Message: "index is missing", Repair: "create_index", Repairable: true})
			continue
		}
		if index.Unique && !databaseIndexIsUnique(ctx, db, index.Table, index.Name) {
			diagnostics := inspectDuplicateIndexKeys(ctx, db, index)
			repairable := diagnostics.Err == nil && diagnostics.GroupCount == 0
			repair := "recreate_unique_index"
			message := "index must be unique"
			if diagnostics.Err != nil {
				repair = "inspect_duplicate_rows"
				message = "index must be unique; duplicate-key inspection failed"
			} else if !repairable {
				repair = "resolve_duplicate_rows"
				message = "index must be unique; duplicate business rows block automatic repair"
			}
			issues = append(issues, SchemaAuditIssue{
				Kind:                "invalid_index",
				Table:               index.Table,
				Name:                index.Name,
				Columns:             index.Columns,
				Message:             message,
				Repair:              repair,
				Repairable:          repairable,
				DuplicateGroupCount: diagnostics.GroupCount,
				DuplicateRowCount:   diagnostics.RowCount,
				DuplicateSamples:    diagnostics.Samples,
			})
		}
	}
	return issues
}

func RepairSchema(ctx context.Context, db *gorm.DB) ([]SchemaAuditIssue, error) {
	if db == nil {
		return AuditSchema(ctx, db), nil
	}
	if err := AutoMigrate(db.WithContext(ctx)); err != nil {
		return nil, err
	}
	for _, index := range DesiredIndexes() {
		if hasDatabaseIndex(ctx, db, index.Table, index.Name) {
			if index.Unique && !databaseIndexIsUnique(ctx, db, index.Table, index.Name) {
				diagnostics := inspectDuplicateIndexKeys(ctx, db, index)
				if diagnostics.Err != nil || diagnostics.GroupCount > 0 {
					continue
				}
				if err := replaceDatabaseIndexWithUnique(ctx, db, index); err != nil {
					if retryDiagnostics := inspectDuplicateIndexKeys(ctx, db, index); retryDiagnostics.GroupCount > 0 {
						continue
					}
					return AuditSchema(ctx, db), err
				}
			}
			continue
		}
		if err := createDatabaseIndex(ctx, db, index); err != nil {
			return AuditSchema(ctx, db), err
		}
	}
	return AuditSchema(ctx, db), nil
}

type duplicateIndexDiagnostics struct {
	GroupCount int64
	RowCount   int64
	Samples    []DuplicateKeySample
	Err        error
}

func inspectDuplicateIndexKeys(ctx context.Context, db *gorm.DB, index DesiredIndex) duplicateIndexDiagnostics {
	if db == nil || index.Table == "" || len(index.Columns) == 0 {
		return duplicateIndexDiagnostics{}
	}
	quotedColumns := make([]string, 0, len(index.Columns))
	for _, column := range index.Columns {
		quotedColumns = append(quotedColumns, quoteIdentifier(db, column))
	}
	groupColumns := strings.Join(quotedColumns, ", ")
	base := fmt.Sprintf(
		"SELECT %s, COUNT(*) AS duplicate_count FROM %s GROUP BY %s HAVING COUNT(*) > 1",
		groupColumns,
		quoteIdentifier(db, index.Table),
		groupColumns,
	)
	type countRow struct {
		GroupCount int64 `gorm:"column:group_count"`
		RowCount   int64 `gorm:"column:row_count"`
	}
	var counts countRow
	countSQL := fmt.Sprintf(
		"SELECT COUNT(*) AS group_count, COALESCE(SUM(duplicate_count - 1), 0) AS row_count FROM (%s) AS duplicate_groups",
		base,
	)
	if err := db.WithContext(ctx).Raw(countSQL).Scan(&counts).Error; err != nil {
		return duplicateIndexDiagnostics{Err: err}
	}
	diagnostics := duplicateIndexDiagnostics{GroupCount: counts.GroupCount, RowCount: counts.RowCount}
	if diagnostics.GroupCount == 0 {
		return diagnostics
	}
	rows, err := db.WithContext(ctx).Raw(base + " ORDER BY duplicate_count DESC LIMIT 5").Rows()
	if err != nil {
		return diagnostics
	}
	defer rows.Close()
	columnNames, err := rows.Columns()
	if err != nil {
		return diagnostics
	}
	for rows.Next() {
		values := make([]any, len(columnNames))
		destinations := make([]any, len(columnNames))
		for i := range values {
			destinations[i] = &values[i]
		}
		if err := rows.Scan(destinations...); err != nil {
			break
		}
		sample := DuplicateKeySample{Values: map[string]any{}}
		for i, column := range columnNames {
			if column == "duplicate_count" {
				sample.Count = int64Value(values[i])
				continue
			}
			sample.Values[column] = values[i]
		}
		diagnostics.Samples = append(diagnostics.Samples, sample)
	}
	return diagnostics
}

func int64Value(value any) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case []byte:
		var parsed int64
		_, _ = fmt.Sscan(string(typed), &parsed)
		return parsed
	case string:
		var parsed int64
		_, _ = fmt.Sscan(typed, &parsed)
		return parsed
	default:
		var parsed int64
		_, _ = fmt.Sscan(fmt.Sprint(value), &parsed)
		return parsed
	}
}

func replaceDatabaseIndexWithUnique(ctx context.Context, db *gorm.DB, index DesiredIndex) error {
	if db == nil || !index.Unique || index.Table == "" || index.Name == "" || len(index.Columns) == 0 {
		return nil
	}
	tempName := temporaryRepairIndexName(index.Name)
	tempIndex := index
	tempIndex.Name = tempName
	dialect := ""
	if db.Dialector != nil {
		dialect = db.Dialector.Name()
	}
	switch dialect {
	case "mysql":
		if err := createDatabaseIndex(ctx, db, tempIndex); err != nil {
			return err
		}
		sql := fmt.Sprintf(
			"ALTER TABLE %s DROP INDEX %s, RENAME INDEX %s TO %s",
			quoteIdentifier(db, index.Table),
			quoteIdentifier(db, index.Name),
			quoteIdentifier(db, tempName),
			quoteIdentifier(db, index.Name),
		)
		if err := db.WithContext(ctx).Exec(sql).Error; err != nil {
			_ = db.WithContext(ctx).Exec(fmt.Sprintf(
				"DROP INDEX %s ON %s",
				quoteIdentifier(db, tempName),
				quoteIdentifier(db, index.Table),
			)).Error
			return err
		}
		return nil
	case "postgres":
		return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := createDatabaseIndex(ctx, tx, tempIndex); err != nil {
				return err
			}
			if err := tx.Exec("DROP INDEX " + quoteIdentifier(tx, index.Name)).Error; err != nil {
				return err
			}
			return tx.Exec(fmt.Sprintf(
				"ALTER INDEX %s RENAME TO %s",
				quoteIdentifier(tx, tempName),
				quoteIdentifier(tx, index.Name),
			)).Error
		})
	default:
		return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := createDatabaseIndex(ctx, tx, tempIndex); err != nil {
				return err
			}
			if err := tx.Exec("DROP INDEX " + quoteIdentifier(tx, index.Name)).Error; err != nil {
				return err
			}
			if err := createDatabaseIndex(ctx, tx, index); err != nil {
				return err
			}
			return tx.Exec("DROP INDEX " + quoteIdentifier(tx, tempName)).Error
		})
	}
}

func temporaryRepairIndexName(name string) string {
	const maxIdentifierLength = 63
	suffix := fmt.Sprintf("__repair_%d", time.Now().UnixNano())
	if len(name)+len(suffix) > maxIdentifierLength {
		name = name[:maxIdentifierLength-len(suffix)]
	}
	return name + suffix
}

func hasDatabaseIndex(ctx context.Context, db *gorm.DB, table, name string) bool {
	if db == nil || table == "" || name == "" {
		return false
	}
	if db.Dialector != nil && db.Dialector.Name() == "postgres" {
		var exists bool
		err := db.WithContext(ctx).Raw(`SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = current_schema() AND tablename = ? AND indexname = ?)`, table, name).Scan(&exists).Error
		return err == nil && exists
	}
	if db.Dialector != nil && db.Dialector.Name() == "mysql" {
		var count int64
		err := db.WithContext(ctx).Raw(`SELECT COUNT(1) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?`, table, name).Scan(&count).Error
		return err == nil && count > 0
	}
	return db.WithContext(ctx).Migrator().HasIndex(table, name)
}

func databaseIndexIsUnique(ctx context.Context, db *gorm.DB, table, name string) bool {
	if db == nil || table == "" || name == "" {
		return false
	}
	if db.Dialector != nil && db.Dialector.Name() == "postgres" {
		var unique bool
		err := db.WithContext(ctx).Raw(`
			SELECT EXISTS (
				SELECT 1
				FROM pg_indexes
				WHERE schemaname = current_schema()
					AND tablename = ?
					AND indexname = ?
					AND indexdef ILIKE 'CREATE UNIQUE INDEX%'
			)
		`, table, name).Scan(&unique).Error
		return err == nil && unique
	}
	if db.Dialector != nil && db.Dialector.Name() == "mysql" {
		var nonUnique int
		err := db.WithContext(ctx).Raw(`
			SELECT COALESCE(MIN(non_unique), 1)
			FROM information_schema.statistics
			WHERE table_schema = DATABASE()
				AND table_name = ?
				AND index_name = ?
		`, table, name).Scan(&nonUnique).Error
		return err == nil && nonUnique == 0
	}
	indexes, err := db.WithContext(ctx).Migrator().GetIndexes(table)
	if err != nil {
		return false
	}
	for _, index := range indexes {
		if index.Name() != name {
			continue
		}
		unique, ok := index.Unique()
		return ok && unique
	}
	return false
}

func createDatabaseIndex(ctx context.Context, db *gorm.DB, index DesiredIndex) error {
	columns := make([]string, 0, len(index.Columns))
	for _, column := range index.Columns {
		columns = append(columns, quoteIdentifier(db, column))
	}
	if len(columns) == 0 {
		return nil
	}
	unique := ""
	if index.Unique {
		unique = "UNIQUE "
	}
	sql := fmt.Sprintf("CREATE %sINDEX IF NOT EXISTS %s ON %s (%s)", unique, quoteIdentifier(db, index.Name), quoteIdentifier(db, index.Table), strings.Join(columns, ", "))
	if db.Dialector != nil && db.Dialector.Name() == "postgres" && !index.Unique {
		sql = fmt.Sprintf("CREATE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s (%s)", quoteIdentifier(db, index.Name), quoteIdentifier(db, index.Table), strings.Join(columns, ", "))
	}
	if db.Dialector != nil && db.Dialector.Name() == "mysql" {
		if hasDatabaseIndex(ctx, db, index.Table, index.Name) {
			return nil
		}
		sql = fmt.Sprintf("CREATE %sINDEX %s ON %s (%s)", unique, quoteIdentifier(db, index.Name), quoteIdentifier(db, index.Table), strings.Join(columns, ", "))
	}
	return db.WithContext(ctx).Exec(sql).Error
}

func quoteIdentifier(db *gorm.DB, value string) string {
	value = strings.ReplaceAll(value, `"`, `""`)
	if db != nil && db.Dialector != nil && db.Dialector.Name() == "mysql" {
		return "`" + strings.ReplaceAll(value, "`", "``") + "`"
	}
	return `"` + value + `"`
}
