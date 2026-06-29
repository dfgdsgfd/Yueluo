package storage

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestAuditSchemaWithoutDatabaseReportsUnavailable(t *testing.T) {
	issues := AuditSchema(context.Background(), nil)
	if len(issues) != 1 || issues[0].Kind != "database" {
		t.Fatalf("AuditSchema(nil) issues = %#v, want database issue", issues)
	}
}

func TestDesiredIndexesRegistryIsPopulated(t *testing.T) {
	indexes := DesiredIndexes()
	if len(indexes) == 0 {
		t.Fatalf("DesiredIndexes() is empty")
	}
	hasAnnouncementExpiryIndex := false
	hasAccessBehaviorIndex := false
	hasSecurityActorIndex := false
	hasLikeInteractionIndex := false
	hasIMMessageIndex := false
	hasIMUnreadMessageIndex := false
	hasIMReceiptReadIndex := false
	hasSystemNotificationConfirmationIndex := false
	hasPostFeedFilterIndex := false
	hasPostPaymentIndex := false
	hasNotificationTypeIndex := false
	hasSystemNotificationTypeIndex := false
	hasPostSearchFilterIndex := false
	hasUserIDIndex := false
	hasUserNicknameIndex := false
	hasUserCreatedIndex := false
	hasTagNameIndex := false
	hasUserAPIKeyIndex := false
	hasOpenAPIKeyIndex := false
	hasImageProtectionJobIndex := false
	hasImageProtectionPostKindIndex := false
	hasImageWatermarkTraceTokenIndex := false
	hasImageWatermarkTraceShortCodeIndex := false
	hasImageWatermarkTraceJobImageIndex := false
	hasPostImageSortIndex := false
	hasPostImageURLIndex := false
	hasPostVideoCoverURLIndex := false
	hasPostVideoURLIndex := false
	hasPostVideoDASHURLIndex := false
	hasPostVideoPreviewURLIndex := false
	hasUploadAssetURLIndex := false
	hasUploadAssetExpiryIndex := false
	hasUploadAssetUserIndex := false
	hasUploadAssetPostIndex := false
	hasFileRecycleGroupIndex := false
	hasFileRecycleResourceIndex := false
	hasFileRecyclePostIndex := false
	hasFileRecycleStatusPurgeIndex := false
	hasPurchasedContentUniqueIndex := false
	hasPurchaseIntentUniqueIndex := false
	hasPurchaseIntentStatusIndex := false
	hasExternalOperationIndex := false
	hasExternalStatusIndex := false
	hasPointsCreatedIndex := false
	hasPointsUserIndex := false
	hasPointsTypeIndex := false
	hasAuditTypeIndex := false
	hasAuditStatusIndex := false
	hasAuditUserIndex := false
	hasAIJobIndex := false
	hasAICreatedIndex := false
	hasAIActorIndex := false
	hasAITaskIndex := false
	hasAccessBlockImportURLIndex := false
	hasAccessBlockImportDueIndex := false
	hasAccessBlockRuleSourceUniqueIndex := false
	hasAccessBlockRuleImportSourceIndex := false
	for _, index := range indexes {
		if index.Table == "" || index.Name == "" || len(index.Columns) == 0 {
			t.Fatalf("invalid desired index entry: %#v", index)
		}
		if index.Table == "announcements" && index.Name == "idx_announcements_published_expires_created" {
			hasAnnouncementExpiryIndex = true
		}
		if index.Table == "access_logs" && index.Name == "idx_access_logs_behavior_created" {
			hasAccessBehaviorIndex = true
		}
		if index.Table == "access_block_import_sources" && index.Name == "idx_access_block_import_sources_url" && index.Unique {
			hasAccessBlockImportURLIndex = true
		}
		if index.Table == "access_block_import_sources" && index.Name == "idx_access_block_import_sources_enabled_next_sync" {
			hasAccessBlockImportDueIndex = true
		}
		if index.Table == "access_block_rules" && index.Name == "idx_access_block_rules_source_unique" && index.Unique {
			hasAccessBlockRuleSourceUniqueIndex = true
		}
		if index.Table == "access_block_rules" && index.Name == "idx_access_block_rules_import_source" {
			hasAccessBlockRuleImportSourceIndex = true
		}
		if index.Table == "security_audit_logs" && index.Name == "idx_security_audit_actor_created" {
			hasSecurityActorIndex = true
		}
		if index.Table == "likes" && index.Name == "idx_likes_user_type_target" {
			hasLikeInteractionIndex = true
		}
		if index.Table == "im_messages" && index.Name == "idx_im_messages_conversation_id" {
			hasIMMessageIndex = true
		}
		if index.Table == "im_messages" && index.Name == "idx_im_messages_conversation_sender_id" {
			hasIMUnreadMessageIndex = true
		}
		if index.Table == "im_message_receipts" && index.Name == "idx_im_message_receipts_user_read_message" {
			hasIMReceiptReadIndex = true
		}
		if index.Table == "system_notification_confirmations" && index.Name == "idx_system_notification_confirmations_user_notification" && index.Unique {
			hasSystemNotificationConfirmationIndex = true
		}
		if index.Table == "posts" && index.Name == "idx_posts_feed_filters_created" {
			hasPostFeedFilterIndex = true
		}
		if index.Table == "posts" && index.Name == "idx_posts_search_filters_created" {
			hasPostSearchFilterIndex = true
		}
		if index.Table == "users" && index.Name == "idx_users_user_id" {
			hasUserIDIndex = true
		}
		if index.Table == "users" && index.Name == "idx_users_nickname" {
			hasUserNicknameIndex = true
		}
		if index.Table == "users" && index.Name == "idx_users_created_at" {
			hasUserCreatedIndex = true
		}
		if index.Table == "tags" && index.Name == "idx_tags_name" {
			hasTagNameIndex = true
		}
		if index.Table == "post_payment_settings" && index.Name == "idx_post_payment_settings_post_id" {
			hasPostPaymentIndex = true
		}
		if index.Table == "notifications" && index.Name == "idx_notifications_user_type_created" {
			hasNotificationTypeIndex = true
		}
		if index.Table == "system_notifications" && index.Name == "idx_system_notifications_active_type_created" {
			hasSystemNotificationTypeIndex = true
		}
		if index.Table == "user_api_keys" && index.Name == "idx_user_api_keys_api_key" && index.Unique {
			hasUserAPIKeyIndex = true
		}
		if index.Table == "open_apis" && index.Name == "idx_open_apis_api_key" && index.Unique {
			hasOpenAPIKeyIndex = true
		}
		if index.Table == "image_protection_jobs" && index.Name == "idx_image_protection_jobs_job_id" && index.Unique {
			hasImageProtectionJobIndex = true
		}
		if index.Table == "image_protection_jobs" && index.Name == "idx_image_protection_jobs_post_kind_status" {
			hasImageProtectionPostKindIndex = true
		}
		if index.Table == "image_watermark_traces" && index.Name == "idx_image_watermark_traces_token" && index.Unique {
			hasImageWatermarkTraceTokenIndex = true
		}
		if index.Table == "image_watermark_traces" && index.Name == "idx_image_watermark_traces_short_code" && index.Unique {
			hasImageWatermarkTraceShortCodeIndex = true
		}
		if index.Table == "image_watermark_traces" && index.Name == "idx_image_watermark_traces_job_image" && index.Unique {
			hasImageWatermarkTraceJobImageIndex = true
		}
		if index.Table == "post_images" && index.Name == "idx_post_images_post_sort" {
			hasPostImageSortIndex = true
		}
		if index.Table == "post_images" && index.Name == "idx_post_images_image_url" {
			hasPostImageURLIndex = true
		}
		if index.Table == "post_videos" && index.Name == "idx_post_videos_cover_url" {
			hasPostVideoCoverURLIndex = true
		}
		if index.Table == "post_videos" && index.Name == "idx_post_videos_video_url" {
			hasPostVideoURLIndex = true
		}
		if index.Table == "post_videos" && index.Name == "idx_post_videos_dash_url" {
			hasPostVideoDASHURLIndex = true
		}
		if index.Table == "post_videos" && index.Name == "idx_post_videos_preview_video_url" {
			hasPostVideoPreviewURLIndex = true
		}
		if index.Table == "upload_assets" && index.Name == "idx_upload_assets_url" && index.Unique {
			hasUploadAssetURLIndex = true
		}
		if index.Table == "upload_assets" && index.Name == "idx_upload_assets_status_expires" {
			hasUploadAssetExpiryIndex = true
		}
		if index.Table == "upload_assets" && index.Name == "idx_upload_assets_user_status_created" {
			hasUploadAssetUserIndex = true
		}
		if index.Table == "upload_assets" && index.Name == "idx_upload_assets_bound_post" {
			hasUploadAssetPostIndex = true
		}
		if index.Table == "file_recycle_items" && index.Name == "idx_file_recycle_group" {
			hasFileRecycleGroupIndex = true
		}
		if index.Table == "file_recycle_items" && index.Name == "idx_file_recycle_resource" {
			hasFileRecycleResourceIndex = true
		}
		if index.Table == "file_recycle_items" && index.Name == "idx_file_recycle_post" {
			hasFileRecyclePostIndex = true
		}
		if index.Table == "file_recycle_items" && index.Name == "idx_file_recycle_status_purge" {
			hasFileRecycleStatusPurgeIndex = true
		}
		if index.Table == "user_purchased_content" && index.Name == "idx_user_purchased_content_user_post" && index.Unique {
			hasPurchasedContentUniqueIndex = true
		}
		if index.Table == "content_purchase_intents" && index.Name == "idx_content_purchase_intents_user_post" && index.Unique {
			hasPurchaseIntentUniqueIndex = true
		}
		if index.Table == "content_purchase_intents" && index.Name == "idx_content_purchase_intents_status_updated" {
			hasPurchaseIntentStatusIndex = true
		}
		if index.Table == "external_balance_transactions" && index.Name == "idx_external_balance_transactions_operation" && index.Unique {
			hasExternalOperationIndex = true
		}
		if index.Table == "external_balance_transactions" && index.Name == "idx_external_balance_transactions_status_updated" {
			hasExternalStatusIndex = true
		}
		if index.Table == "points_log" && index.Name == "idx_points_log_created_at" {
			hasPointsCreatedIndex = true
		}
		if index.Table == "points_log" && index.Name == "idx_points_log_user_created" {
			hasPointsUserIndex = true
		}
		if index.Table == "points_log" && index.Name == "idx_points_log_type_created" {
			hasPointsTypeIndex = true
		}
		if index.Table == "audit" && index.Name == "idx_audit_type_created" {
			hasAuditTypeIndex = true
		}
		if index.Table == "audit" && index.Name == "idx_audit_type_status_created" {
			hasAuditStatusIndex = true
		}
		if index.Table == "audit" && index.Name == "idx_audit_user_type_created" {
			hasAuditUserIndex = true
		}
		if index.Table == "ai_generation_logs" && index.Name == "idx_ai_generation_logs_job_id" && index.Unique {
			hasAIJobIndex = true
		}
		if index.Table == "ai_generation_logs" && index.Name == "idx_ai_generation_logs_created" {
			hasAICreatedIndex = true
		}
		if index.Table == "ai_generation_logs" && index.Name == "idx_ai_generation_logs_actor_created" {
			hasAIActorIndex = true
		}
		if index.Table == "ai_generation_logs" && index.Name == "idx_ai_generation_logs_task_created" {
			hasAITaskIndex = true
		}
	}
	requiredNewIndexes := []string{
		"posts.idx_posts_user_draft_visibility_created",
		"likes.idx_likes_user_type_created",
		"likes.idx_likes_target_type_target_created",
		"collections.idx_collections_user_created",
		"collections.idx_collections_post_created",
		"follows.idx_follows_follower_created",
		"follows.idx_follows_following_created",
		"browsing_history.idx_browsing_history_post_created",
		"creator_earnings_log.idx_creator_earnings_log_user_created",
		"creator_earnings_log.idx_creator_earnings_log_user_type_created",
		"comments.idx_comments_post_parent_public_created",
		"file_recycle_items.idx_file_recycle_purge_deleted_id",
	}
	for _, required := range requiredNewIndexes {
		found := false
		for _, index := range indexes {
			if index.Table+"."+index.Name == required {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("new performance index %s missing from DesiredIndexes(): %#v", required, indexes)
		}
	}
	if !hasAnnouncementExpiryIndex {
		t.Fatalf("announcement expiry index missing from DesiredIndexes(): %#v", indexes)
	}
	if !hasAccessBehaviorIndex || !hasSecurityActorIndex {
		t.Fatalf("log indexes missing from DesiredIndexes(): access=%v security=%v indexes=%#v", hasAccessBehaviorIndex, hasSecurityActorIndex, indexes)
	}
	if !hasAccessBlockImportURLIndex || !hasAccessBlockImportDueIndex || !hasAccessBlockRuleSourceUniqueIndex || !hasAccessBlockRuleImportSourceIndex {
		t.Fatalf("access block import indexes missing: url=%v due=%v sourceUnique=%v source=%v", hasAccessBlockImportURLIndex, hasAccessBlockImportDueIndex, hasAccessBlockRuleSourceUniqueIndex, hasAccessBlockRuleImportSourceIndex)
	}
	if !hasLikeInteractionIndex || !hasIMMessageIndex || !hasSystemNotificationConfirmationIndex {
		t.Fatalf("hot-path indexes missing from DesiredIndexes(): likes=%v im=%v confirmations=%v indexes=%#v", hasLikeInteractionIndex, hasIMMessageIndex, hasSystemNotificationConfirmationIndex, indexes)
	}
	if !hasIMUnreadMessageIndex || !hasIMReceiptReadIndex || !hasPostFeedFilterIndex || !hasPostPaymentIndex || !hasNotificationTypeIndex || !hasSystemNotificationTypeIndex {
		t.Fatalf(
			"performance indexes missing from DesiredIndexes(): imUnread=%v imReceiptRead=%v postFeed=%v postPayment=%v notificationType=%v systemNotificationType=%v",
			hasIMUnreadMessageIndex,
			hasIMReceiptReadIndex,
			hasPostFeedFilterIndex,
			hasPostPaymentIndex,
			hasNotificationTypeIndex,
			hasSystemNotificationTypeIndex,
		)
	}
	if !hasPostSearchFilterIndex || !hasUserIDIndex || !hasUserNicknameIndex || !hasUserCreatedIndex || !hasTagNameIndex {
		t.Fatalf(
			"search indexes missing: postSearch=%v userID=%v nickname=%v userCreated=%v tagName=%v",
			hasPostSearchFilterIndex,
			hasUserIDIndex,
			hasUserNicknameIndex,
			hasUserCreatedIndex,
			hasTagNameIndex,
		)
	}
	if !hasUserAPIKeyIndex || !hasOpenAPIKeyIndex {
		t.Fatalf("API key indexes missing from DesiredIndexes(): user=%v open=%v indexes=%#v", hasUserAPIKeyIndex, hasOpenAPIKeyIndex, indexes)
	}
	if !hasImageProtectionJobIndex || !hasImageProtectionPostKindIndex || !hasPostImageSortIndex {
		t.Fatalf("image protection indexes missing from DesiredIndexes(): job=%v post_kind=%v sort=%v indexes=%#v", hasImageProtectionJobIndex, hasImageProtectionPostKindIndex, hasPostImageSortIndex, indexes)
	}
	if !hasPostImageURLIndex || !hasPostVideoCoverURLIndex || !hasPostVideoURLIndex || !hasPostVideoDASHURLIndex || !hasPostVideoPreviewURLIndex {
		t.Fatalf(
			"file access URL indexes missing: image=%v cover=%v video=%v dash=%v preview=%v",
			hasPostImageURLIndex,
			hasPostVideoCoverURLIndex,
			hasPostVideoURLIndex,
			hasPostVideoDASHURLIndex,
			hasPostVideoPreviewURLIndex,
		)
	}
	if !hasUploadAssetURLIndex || !hasUploadAssetExpiryIndex || !hasUploadAssetUserIndex || !hasUploadAssetPostIndex {
		t.Fatalf(
			"upload asset indexes missing: url=%v expiry=%v user=%v post=%v",
			hasUploadAssetURLIndex,
			hasUploadAssetExpiryIndex,
			hasUploadAssetUserIndex,
			hasUploadAssetPostIndex,
		)
	}
	if !hasFileRecycleGroupIndex || !hasFileRecycleResourceIndex || !hasFileRecyclePostIndex || !hasFileRecycleStatusPurgeIndex {
		t.Fatalf(
			"file recycle indexes missing: group=%v resource=%v post=%v statusPurge=%v",
			hasFileRecycleGroupIndex,
			hasFileRecycleResourceIndex,
			hasFileRecyclePostIndex,
			hasFileRecycleStatusPurgeIndex,
		)
	}
	if !hasImageWatermarkTraceTokenIndex || !hasImageWatermarkTraceShortCodeIndex || !hasImageWatermarkTraceJobImageIndex {
		t.Fatalf("image watermark trace indexes missing: token=%v short_code=%v job_image=%v indexes=%#v", hasImageWatermarkTraceTokenIndex, hasImageWatermarkTraceShortCodeIndex, hasImageWatermarkTraceJobImageIndex, indexes)
	}
	if !hasPurchasedContentUniqueIndex || !hasPurchaseIntentUniqueIndex || !hasPurchaseIntentStatusIndex {
		t.Fatalf(
			"purchase indexes missing from DesiredIndexes(): purchased=%v intent_unique=%v intent_status=%v indexes=%#v",
			hasPurchasedContentUniqueIndex,
			hasPurchaseIntentUniqueIndex,
			hasPurchaseIntentStatusIndex,
			indexes,
		)
	}
	if !hasExternalOperationIndex || !hasExternalStatusIndex {
		t.Fatalf("external balance indexes missing: operation=%v status=%v", hasExternalOperationIndex, hasExternalStatusIndex)
	}
	if !hasPointsCreatedIndex || !hasPointsUserIndex || !hasPointsTypeIndex {
		t.Fatalf("points audit indexes missing: created=%v user=%v type=%v", hasPointsCreatedIndex, hasPointsUserIndex, hasPointsTypeIndex)
	}
	if !hasAuditTypeIndex || !hasAuditStatusIndex || !hasAuditUserIndex {
		t.Fatalf("audit indexes missing: type=%v status=%v user=%v", hasAuditTypeIndex, hasAuditStatusIndex, hasAuditUserIndex)
	}
	if !hasAIJobIndex || !hasAICreatedIndex || !hasAIActorIndex || !hasAITaskIndex {
		t.Fatalf("AI generation log indexes missing: job=%v created=%v actor=%v task=%v", hasAIJobIndex, hasAICreatedIndex, hasAIActorIndex, hasAITaskIndex)
	}
}

func TestRepairInvalidUniqueIndexReplacesNonUniqueIndexWithoutDeletingRows(t *testing.T) {
	db := openSchemaAuditSQLite(t)
	if err := db.Exec(`CREATE TABLE user_purchased_content (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, post_id INTEGER NOT NULL)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE INDEX idx_user_purchased_content_user_post ON user_purchased_content (user_id, post_id)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO user_purchased_content (id, user_id, post_id) VALUES (1, 10, 20), (2, 10, 21)`).Error; err != nil {
		t.Fatal(err)
	}
	index := DesiredIndex{
		Table:   "user_purchased_content",
		Name:    "idx_user_purchased_content_user_post",
		Columns: []string{"user_id", "post_id"},
		Unique:  true,
	}
	if err := replaceDatabaseIndexWithUnique(context.Background(), db, index); err != nil {
		t.Fatalf("replaceDatabaseIndexWithUnique() error = %v", err)
	}
	if !databaseIndexIsUnique(context.Background(), db, index.Table, index.Name) {
		t.Fatalf("index was not replaced with a unique index")
	}
	var count int64
	if err := db.Table(index.Table).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("row count = %d, want 2", count)
	}
}

func TestAuditInvalidUniqueIndexReportsDuplicateBlockerAndPreservesIndex(t *testing.T) {
	db := openSchemaAuditSQLite(t)
	if err := db.Exec(`CREATE TABLE user_purchased_content (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, post_id INTEGER NOT NULL)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE INDEX idx_user_purchased_content_user_post ON user_purchased_content (user_id, post_id)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO user_purchased_content (id, user_id, post_id) VALUES (1, 10, 20), (2, 10, 20), (3, 10, 20), (4, 11, 20), (5, 11, 20)`).Error; err != nil {
		t.Fatal(err)
	}
	index := DesiredIndex{
		Table:   "user_purchased_content",
		Name:    "idx_user_purchased_content_user_post",
		Columns: []string{"user_id", "post_id"},
		Unique:  true,
	}
	diagnostics := inspectDuplicateIndexKeys(context.Background(), db, index)
	if diagnostics.GroupCount != 2 || diagnostics.RowCount != 3 {
		t.Fatalf("duplicate diagnostics = %#v, want groups=2 rows=3", diagnostics)
	}
	if len(diagnostics.Samples) == 0 {
		t.Fatalf("duplicate samples are empty")
	}
	if diagnostics.Samples[0].Count < 2 {
		t.Fatalf("duplicate sample = %#v", diagnostics.Samples[0])
	}
	if err := replaceDatabaseIndexWithUnique(context.Background(), db, index); err == nil {
		t.Fatalf("replacement should fail while duplicate rows exist")
	}
	if databaseIndexIsUnique(context.Background(), db, index.Table, index.Name) {
		t.Fatalf("original non-unique index should remain unchanged")
	}
	var count int64
	if err := db.Table(index.Table).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 5 {
		t.Fatalf("row count = %d, want 5", count)
	}
}

func openSchemaAuditSQLite(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	return db
}
