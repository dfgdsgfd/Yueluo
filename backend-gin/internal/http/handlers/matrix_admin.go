package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/http/response"
)

type adminResource struct {
	Name         string
	Table        string
	Alias        string
	Select       string
	Joins        []string
	SearchFields []string
	Filters      map[string]adminFilter
	SortFields   map[string]string
	HiddenFields []string
	DefaultOrder string
	ListShape    string
	ListMessage  string
	PageSizeKey  string
	BaseWhere    []adminWhere
}

type adminFilter struct {
	Column string
	Mode   string
}

type adminWhere struct {
	Clause string
	Args   []any
}

const (
	adminListNested   = "nested"
	adminListTopLevel = "top-level"
	adminListItems    = "items"
	adminListAudit    = "audit"
	adminListReports  = "reports"
)

var adminResources = map[string]adminResource{
	"admins": {
		Name: "admins", Table: "admin", Alias: "a",
		Select:       "a.id, a.username, a.created_at",
		SearchFields: []string{"a.username"},
		Filters:      adminFilters("username:like:a.username"),
		SortFields:   adminSortFields("id:a.id", "username:a.username", "created_at:a.created_at"),
		HiddenFields: []string{"password"},
		DefaultOrder: "a.created_at DESC", ListShape: adminListNested,
	},
	"announcements": {
		Name: "announcements", Table: "announcements", Alias: "a",
		SearchFields: []string{"a.title", "a.content"},
		Filters:      adminFilters("title:like:a.title", "type:eq:a.type", "is_published:bool:a.is_published"),
		SortFields:   adminSortFields("id:a.id", "title:a.title", "type:a.type", "is_published:a.is_published", "published_at:a.published_at", "expires_at:a.expires_at", "created_at:a.created_at", "updated_at:a.updated_at"),
		DefaultOrder: "a.created_at DESC", ListShape: adminListNested,
	},
	"app-versions": {Name: "app-versions", Table: "app_versions", SearchFields: []string{"app_name", "version_name", "platform"}, DefaultOrder: "created_at DESC"},
	"audit": {
		Name: "audit", Table: "audit", Alias: "a",
		Select:       "a.id, a.user_id, COALESCE(u.xise_id, u.user_id) AS user_display_id, u.nickname, u.avatar, a.type, a.content, a.audit_result, a.status, a.reason, a.created_at, a.audit_time",
		Joins:        []string{"LEFT JOIN users u ON u.id = a.user_id"},
		SearchFields: []string{"a.content", "a.reason", "u.user_id", "u.nickname"},
		Filters:      adminFilters("user_id:int64:a.user_id", "user_display_id:like:u.user_id", "type:int:a.type", "status:int:a.status"),
		SortFields:   adminSortFields("id:a.id", "type:a.type", "status:a.status", "created_at:a.created_at", "audit_time:a.audit_time"),
		BaseWhere:    []adminWhere{{Clause: "a.type IN ?", Args: []any{[]int{1, 2}}}},
		DefaultOrder: "a.created_at DESC", ListShape: adminListAudit, ListMessage: "获取认证申请列表成功",
	},
	"banned-word-categories": {
		Name: "banned-word-categories", Table: "banned_word_categories", Alias: "bwc",
		Select:       "bwc.*, (SELECT COUNT(*) FROM banned_words bw WHERE bw.category_id = bwc.id) AS word_count",
		SearchFields: []string{"bwc.name", "bwc.description"},
		Filters:      adminFilters("name:like:bwc.name"),
		SortFields:   adminSortFields("id:bwc.id", "name:bwc.name", "created_at:bwc.created_at", "updated_at:bwc.updated_at"),
		DefaultOrder: "bwc.name ASC", ListShape: adminListTopLevel, ListMessage: matrixMsgOK,
	},
	"banned-words": {
		Name: "banned-words", Table: "banned_words", Alias: "bw",
		Select:       "bw.*, bwc.id AS category_id_value, bwc.name AS category_name",
		Joins:        []string{"LEFT JOIN banned_word_categories bwc ON bwc.id = bw.category_id"},
		SearchFields: []string{"bw.word"},
		Filters:      adminFilters("word:like:bw.word", "category_id:nullint:bw.category_id", "enabled:bool:bw.enabled", "is_regex:bool:bw.is_regex"),
		SortFields:   adminSortFields("id:bw.id", "word:bw.word", "category_id:bw.category_id", "enabled:bw.enabled", "created_at:bw.created_at", "updated_at:bw.updated_at"),
		DefaultOrder: "bw.created_at DESC", ListShape: adminListNested,
	},
	"categories": {
		Name: "categories", Table: "categories", Alias: "c",
		Select:       "c.id, c.name, c.category_title, c.translations, c.use_count, c.created_at, (SELECT COUNT(*) FROM posts p WHERE p.category_id = c.id) AS post_count",
		SearchFields: []string{"c.name", "c.category_title"},
		Filters:      adminFilters("name:like:c.name", "category_title:like:c.category_title"),
		SortFields:   adminSortFields("id:c.id", "name:c.name", "category_title:c.category_title", "use_count:c.use_count", "created_at:c.created_at"),
		DefaultOrder: "c.id ASC", ListShape: adminListTopLevel, ListMessage: "获取成功",
	},
	"collections": {
		Name: "collections", Table: "collections", Alias: "c",
		Select:       "c.id, c.user_id, c.post_id, c.created_at, u.user_id AS user_display_id, u.nickname, p.title AS post_title",
		Joins:        []string{"LEFT JOIN users u ON u.id = c.user_id", "LEFT JOIN posts p ON p.id = c.post_id"},
		Filters:      adminFilters("user_display_id:like:u.user_id", "post_id:int64:c.post_id"),
		SortFields:   adminSortFields("id:c.id", "user_id:c.user_id", "post_id:c.post_id", "created_at:c.created_at"),
		DefaultOrder: "c.created_at DESC", ListShape: adminListNested,
	},
	"comments": {
		Name: "comments", Table: "comments", Alias: "c",
		Select:       "c.id, c.post_id, c.user_id, c.parent_id, c.content, c.like_count, c.audit_status, c.is_public, c.created_at, u.user_id AS user_display_id, u.nickname, p.title AS post_title",
		Joins:        []string{"LEFT JOIN users u ON u.id = c.user_id", "LEFT JOIN posts p ON p.id = c.post_id"},
		SearchFields: []string{"c.content"},
		Filters:      adminFilters("user_display_id:like:u.user_id", "post_id:int64:c.post_id", "content:like:c.content", "audit_status:int:c.audit_status"),
		SortFields:   adminSortFields("id:c.id", "post_id:c.post_id", "user_id:c.user_id", "audit_status:c.audit_status", "created_at:c.created_at"),
		DefaultOrder: "c.created_at DESC", ListShape: adminListNested,
	},
	"content-review": {
		Name: "content-review", Table: "audit", Alias: "a",
		Select:       "a.id, a.user_id, a.type, a.target_id, a.content, a.risk_level, a.categories, a.reason, a.status, a.created_at, a.audit_time, u.user_id AS user_display_id, u.nickname, u.avatar",
		Joins:        []string{"LEFT JOIN users u ON u.id = a.user_id"},
		SearchFields: []string{"a.content", "a.reason", "u.user_id", "u.nickname"},
		Filters:      adminFilters("user_id:int64:a.user_id", "user_display_id:like:u.user_id", "type:int:a.type", "status:int:a.status"),
		SortFields:   adminSortFields("id:a.id", "user_id:a.user_id", "type:a.type", "status:a.status", "created_at:a.created_at", "audit_time:a.audit_time"),
		BaseWhere:    []adminWhere{{Clause: "a.type IN ?", Args: []any{[]int{3, 4}}}},
		DefaultOrder: "a.created_at DESC", ListShape: adminListAudit, ListMessage: "获取审核列表成功",
	},
	"ai-moderation-logs": {
		Name: "ai-moderation-logs", Table: "ai_moderation_logs", Alias: "aml",
		Select:       "aml.*, COALESCE(aml.metadata->>'contentSnapshot', '') AS content_snapshot, u.user_id AS user_display_id, u.nickname, u.avatar",
		Joins:        []string{"LEFT JOIN users u ON u.id = aml.user_id"},
		SearchFields: []string{"aml.trigger_reason", "aml.error_message", "aml.metadata->>'contentSnapshot'", "u.user_id", "u.nickname"},
		Filters:      adminFilters("target_type:eq:aml.target_type", "target_id:int64:aml.target_id", "user_id:int64:aml.user_id", "status:eq:aml.status", "action:eq:aml.action"),
		SortFields:   adminSortFields("id:aml.id", "user_id:aml.user_id", "target_id:aml.target_id", "status:aml.status", "action:aml.action", "created_at:aml.created_at"),
		DefaultOrder: "aml.created_at DESC", ListShape: adminListNested,
	},
	"feedback": {
		Name: "feedback", Table: "feedback", Alias: "f",
		Select:       "f.*, u.id AS user_id_value, u.user_id AS user_display_id, u.nickname AS user_nickname, u.avatar AS user_avatar",
		Joins:        []string{"LEFT JOIN users u ON u.id = f.user_id"},
		SearchFields: []string{"f.content", "f.admin_reply", "u.nickname", "u.user_id"},
		Filters:      adminFilters("status:eq:f.status", "keyword:like:f.content"),
		SortFields:   adminSortFields("id:f.id", "status:f.status", "created_at:f.created_at", "updated_at:f.updated_at"),
		DefaultOrder: "f.created_at DESC", ListShape: adminListNested,
	},
	"file-recycle-bin": {
		Name: "file-recycle-bin", Table: "file_recycle_items", Alias: "fr",
		Select:       "fr.id, fr.group_id, fr.post_id, fr.user_id, fr.kind, fr.storage, fr.original_url, fr.original_path, fr.recycled_path, fr.is_dir, fr.file_count, fr.size_bytes, fr.status, fr.deleted_at, fr.purge_after, fr.purged_at, fr.error, fr.created_at, u.user_id AS user_display_id, u.nickname",
		Joins:        []string{"LEFT JOIN users u ON u.id = fr.user_id"},
		SearchFields: []string{"fr.original_url", "fr.original_path", "fr.recycled_path", "fr.group_id", "u.user_id", "u.nickname"},
		Filters:      adminFilters("post_id:int64:fr.post_id", "user_id:int64:fr.user_id", "kind:eq:fr.kind", "storage:eq:fr.storage", "status:eq:fr.status"),
		SortFields:   adminSortFields("id:fr.id", "post_id:fr.post_id", "user_id:fr.user_id", "kind:fr.kind", "size_bytes:fr.size_bytes", "status:fr.status", "deleted_at:fr.deleted_at", "purge_after:fr.purge_after", "created_at:fr.created_at"),
		DefaultOrder: "fr.purge_after ASC, fr.deleted_at ASC, fr.id ASC", ListShape: adminListNested,
	},
	"follows": {
		Name: "follows", Table: "follows", Alias: "f",
		Select:       "f.id, f.follower_id, f.following_id, f.created_at, fu.user_id AS follower_display_id, fu.nickname AS follower_nickname, tu.user_id AS following_display_id, tu.nickname AS following_nickname",
		Joins:        []string{"LEFT JOIN users fu ON fu.id = f.follower_id", "LEFT JOIN users tu ON tu.id = f.following_id"},
		Filters:      adminFilters("follower_display_id:eq:fu.user_id", "following_display_id:eq:tu.user_id"),
		SortFields:   adminSortFields("id:f.id", "follower_id:f.follower_id", "following_id:f.following_id", "created_at:f.created_at"),
		DefaultOrder: "f.created_at DESC", ListShape: adminListNested,
	},
	"licenses": {
		Name: "licenses", Table: "licenses", Alias: "l",
		SearchFields: []string{"l.license_key", "l.remark", "l.machine_id", "l.machine_model"},
		Filters:      adminFilters("machine_model:like:l.machine_model", "license_key:like:l.license_key", "machine_id:like:l.machine_id", "is_active:bool:l.is_active"),
		SortFields:   adminSortFields("id:l.id", "license_key:l.license_key", "machine_model:l.machine_model", "is_active:l.is_active", "expires_at:l.expires_at", "created_at:l.created_at"),
		DefaultOrder: "l.created_at DESC", ListShape: adminListNested,
	},
	"likes": {
		Name: "likes", Table: "likes", Alias: "l",
		Select:       "l.id, l.user_id, l.target_type, l.target_id, l.created_at, u.user_id AS user_display_id, u.nickname",
		Joins:        []string{"LEFT JOIN users u ON u.id = l.user_id"},
		Filters:      adminFilters("user_display_id:like:u.user_id", "target_type:int:l.target_type", "target_id:int64:l.target_id"),
		SortFields:   adminSortFields("id:l.id", "user_id:l.user_id", "target_type:l.target_type", "target_id:l.target_id", "created_at:l.created_at"),
		DefaultOrder: "l.created_at DESC", ListShape: adminListNested,
	},
	"media-library": {
		Name: "media-library", Table: "media_library", Alias: "m",
		SearchFields: []string{"m.title", "m.filename", "m.url"},
		Filters:      adminFilters("type:eq:m.type", "title:like:m.title"),
		SortFields:   adminSortFields("id:m.id", "title:m.title", "type:m.type", "created_at:m.created_at", "updated_at:m.updated_at"),
		DefaultOrder: "m.created_at DESC", ListShape: adminListItems, PageSizeKey: "pageSize",
	},
	"notification-templates": {
		Name: "notification-templates", Table: "notification_templates", Alias: "nt",
		SearchFields: []string{"nt.template_key", "nt.name", "nt.content"},
		Filters:      adminFilters("template_key:eq:nt.template_key", "name:like:nt.name", "type:eq:nt.type", "is_active:bool:nt.is_active"),
		SortFields:   adminSortFields("id:nt.id", "template_key:nt.template_key", "name:nt.name", "type:nt.type", "is_active:nt.is_active", "created_at:nt.created_at", "updated_at:nt.updated_at"),
		DefaultOrder: "nt.created_at DESC", ListShape: adminListNested,
	},
	"open-apis": {
		Name: "open-apis", Table: "open_apis", Alias: "oa",
		Select:       "oa.id, oa.name, oa.api_key_prefix, oa.permissions, oa.is_active, oa.last_used_at, oa.created_at",
		SearchFields: []string{"oa.name"},
		Filters:      adminFilters("name:like:oa.name", "is_active:bool:oa.is_active"),
		SortFields:   adminSortFields("id:oa.id", "name:oa.name", "is_active:oa.is_active", "last_used_at:oa.last_used_at", "created_at:oa.created_at"),
		HiddenFields: []string{"api_key"},
		DefaultOrder: "oa.created_at DESC", ListShape: adminListNested,
	},
	"posts": {
		Name: "posts", Table: "posts", Alias: "p",
		Select:       "p.id, p.user_id, p.title, p.content, p.category_id, c.name AS category, p.type, p.view_count, p.like_count, p.collect_count, p.comment_count, p.created_at, p.is_draft, p.visibility, p.public_access_exempt, u.user_id AS user_display_id, u.nickname, (SELECT image_url FROM post_images pi WHERE pi.post_id = p.id ORDER BY pi.id ASC LIMIT 1) AS first_image_url, (SELECT video_url FROM post_videos pv WHERE pv.post_id = p.id ORDER BY pv.id ASC LIMIT 1) AS video_url, (SELECT cover_url FROM post_videos pv WHERE pv.post_id = p.id ORDER BY pv.id ASC LIMIT 1) AS cover_url",
		Joins:        []string{"LEFT JOIN users u ON u.id = p.user_id", "LEFT JOIN categories c ON c.id = p.category_id"},
		SearchFields: []string{"CONCAT('', p.id)", "p.title", "p.content", "u.user_id", "u.nickname"},
		Filters:      adminFilters("title:like:p.title", "user_display_id:like:u.user_id", "category_id:nullint:p.category_id", "type:int:p.type", "is_draft:bool:p.is_draft", "visibility:eq:p.visibility", "public_access_exempt:bool:p.public_access_exempt"),
		SortFields:   adminSortFields("id:p.id", "title:p.title", "user_id:p.user_id", "category_id:p.category_id", "type:p.type", "view_count:p.view_count", "like_count:p.like_count", "collect_count:p.collect_count", "comment_count:p.comment_count", "created_at:p.created_at", "public_access_exempt:p.public_access_exempt"),
		DefaultOrder: "p.created_at DESC", ListShape: adminListNested,
	},
	"post-configs": {
		Name: "post-configs", Table: "post_recommend_configs", Alias: "prc",
		Select:       "prc.id, prc.post_id, prc.target_user_id, prc.boost_score, prc.is_pinned, prc.is_suppressed, prc.is_active, prc.reason, prc.created_at, prc.updated_at, p.title, p.content, p.type, u.user_id AS user_display_id, u.nickname, (SELECT image_url FROM post_images pi WHERE pi.post_id = p.id ORDER BY pi.id ASC LIMIT 1) AS first_image_url, (SELECT cover_url FROM post_videos pv WHERE pv.post_id = p.id ORDER BY pv.id ASC LIMIT 1) AS cover_url",
		Joins:        []string{"LEFT JOIN posts p ON p.id = prc.post_id", "LEFT JOIN users u ON u.id = p.user_id"},
		SearchFields: []string{"CONCAT('', prc.post_id)", "p.title", "p.content", "u.user_id", "u.nickname", "prc.reason"},
		Filters:      adminFilters("post_id:int64:prc.post_id", "is_active:bool:prc.is_active", "is_pinned:bool:prc.is_pinned", "is_suppressed:bool:prc.is_suppressed"),
		SortFields:   adminSortFields("id:prc.id", "post_id:prc.post_id", "boost_score:prc.boost_score", "created_at:prc.created_at", "updated_at:prc.updated_at"),
		DefaultOrder: "prc.created_at DESC",
	},
	"quality-reward-settings": {Name: "quality-reward-settings", Table: "post_quality_reward_settings", SearchFields: []string{"quality_level", "description"}, DefaultOrder: "created_at DESC"},
	"reports": {
		Name: "reports", Table: "reports", Alias: "r",
		Select:       "r.*, u.id AS reporter_id_value, u.user_id AS reporter_user_id, u.nickname AS reporter_nickname, u.avatar AS reporter_avatar",
		Joins:        []string{"LEFT JOIN users u ON u.id = r.reporter_id"},
		SearchFields: []string{"r.reason", "r.description", "r.admin_note"},
		Filters:      adminFilters("status:eq:r.status", "target_type:eq:r.target_type", "keyword:like:r.reason"),
		SortFields:   adminSortFields("id:r.id", "status:r.status", "target_type:r.target_type", "created_at:r.created_at", "updated_at:r.updated_at", "reviewed_at:r.reviewed_at"),
		DefaultOrder: "r.created_at DESC", ListShape: adminListReports, PageSizeKey: "pageSize",
	},
	"system-notifications": {
		Name: "system-notifications", Table: "system_notifications", Alias: "sn",
		Select:       "sn.*, (SELECT COUNT(*) FROM system_notification_confirmations sc WHERE sc.notification_id = sn.id) AS confirmed_count, ((SELECT COUNT(*) FROM users) - (SELECT COUNT(*) FROM system_notification_confirmations sc WHERE sc.notification_id = sn.id)) AS unread_count",
		SearchFields: []string{"sn.title", "sn.content"},
		Filters:      adminFilters("title:like:sn.title", "type:eq:sn.type", "is_active:bool:sn.is_active"),
		SortFields:   adminSortFields("id:sn.id", "title:sn.title", "type:sn.type", "is_active:sn.is_active", "start_time:sn.start_time", "end_time:sn.end_time", "created_at:sn.created_at", "updated_at:sn.updated_at"),
		DefaultOrder: "sn.created_at DESC", ListShape: adminListNested,
	},
	"tags": {
		Name: "tags", Table: "tags", Alias: "t",
		SearchFields: []string{"t.name"},
		Filters:      adminFilters("name:like:t.name"),
		SortFields:   adminSortFields("id:t.id", "name:t.name", "use_count:t.use_count", "created_at:t.created_at"),
		DefaultOrder: "t.use_count DESC", ListShape: adminListNested,
	},
	"user-configs": {Name: "user-configs", Table: "recommend_configs", DefaultOrder: "created_at DESC"},
	"user-toolbar": {
		Name: "user-toolbar", Table: "user_toolbar", Alias: "ut",
		SearchFields: []string{"ut.name", "ut.url"},
		Filters:      adminFilters("name:like:ut.name", "is_active:bool:ut.is_active"),
		SortFields:   adminSortFields("id:ut.id", "name:ut.name", "sort_order:ut.sort_order", "is_active:ut.is_active", "created_at:ut.created_at", "updated_at:ut.updated_at"),
		DefaultOrder: "ut.sort_order ASC, ut.created_at DESC", ListShape: adminListNested,
	},
	"users": {
		Name: "users", Table: "users", Alias: "u",
		Select:       "u.id, u.id AS uid, u.user_id, u.oauth2_id, COALESCE(up.points, 0) AS points, u.nickname, u.avatar, u.background, u.bio, u.location, u.email, u.follow_count, u.fans_count, u.like_count, u.is_active, u.created_at, u.verified, u.gender, u.zodiac_sign, u.mbti, u.education, u.major, u.interests",
		Joins:        []string{"LEFT JOIN user_points up ON up.user_id = u.id"},
		SearchFields: []string{"CONCAT('', u.id)", "u.user_id", "CONCAT('', u.oauth2_id)", "u.nickname", "u.email"},
		Filters:      adminFilters("uid:int64:u.id", "oauth2_id:int64:u.oauth2_id", "user_id:like:u.user_id", "nickname:like:u.nickname", "location:like:u.location", "is_active:bool:u.is_active"),
		SortFields:   adminSortFields("id:u.id", "uid:u.id", "user_id:u.user_id", "oauth2_id:u.oauth2_id", "points:up.points", "nickname:u.nickname", "is_active:u.is_active", "created_at:u.created_at", "verified:u.verified"),
		HiddenFields: []string{"password"},
		DefaultOrder: "u.created_at DESC", ListShape: adminListNested,
	},
}

func (h NativeHandlers) adminDispatch(c *gin.Context) {
	path := c.Request.URL.Path
	if path == "/api/admin/media-library/public" {
		h.adminMediaPublic(c)
		return
	}
	if _, ok := h.requireMatrixAdmin(c); !ok {
		return
	}
	if !h.requireDB(c) {
		return
	}

	if h.adminSpecial(c) {
		return
	}
	segments := adminSegments(c)
	resource, id := adminResourceFromSegments(segments)
	if resource.Table == "" {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "admin route not found", nil)
		return
	}
	h.adminGenericResource(c, resource, id)
}

func (h NativeHandlers) adminSpecial(c *gin.Context) bool {
	path := c.Request.URL.Path
	method := matrixMethod(c)
	switch {
	case path == "/api/admin/media-library" && method == http.MethodPost && c.ContentType() == "multipart/form-data":
		h.adminMediaUpload(c)
	case path == "/api/admin/ai-review-status" && method == http.MethodGet:
		writeSuccess(c, matrixMsgOK, gin.H{
			"enabled":          h.Settings != nil && (h.Settings.Bool("ai_username_review_enabled") || h.Settings.Bool("ai_content_review_enabled")),
			"username_enabled": h.Settings != nil && h.Settings.Bool("ai_username_review_enabled"),
			"content_enabled":  h.Settings != nil && h.Settings.Bool("ai_content_review_enabled"),
		})
	case path == "/api/admin/ai-review-toggle" && method == http.MethodPost:
		enabled, _ := boolFromAny(readBodyMap(c)["enabled"])
		if h.Settings != nil {
			if !h.Settings.Set(c.Request.Context(), "ai_username_review_enabled", enabled) ||
				!h.Settings.Set(c.Request.Context(), "ai_content_review_enabled", enabled) {
				response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
				return true
			}
		}
		writeSimpleSuccess(c, "AI自动审核状态已更新")
	case path == "/api/admin/guest-access-status" && method == http.MethodGet:
		writeSuccess(c, matrixMsgOK, gin.H{
			"restricted":       h.Settings != nil && h.Settings.Bool("guest_access_restricted"),
			"note_restricted":  h.Settings != nil && h.Settings.Bool("guest_access_note_restricted"),
			"video_restricted": h.Settings != nil && h.Settings.Bool("guest_access_video_restricted"),
			"admin_restricted": true,
		})
	case path == "/api/admin/guest-access-toggle" && method == http.MethodPost:
		restricted, _ := boolFromAny(readBodyMap(c)["restricted"])
		if h.Settings != nil {
			if !h.Settings.Set(c.Request.Context(), "guest_access_restricted", restricted) ||
				!h.Settings.Set(c.Request.Context(), "guest_access_note_restricted", restricted) ||
				!h.Settings.Set(c.Request.Context(), "guest_access_video_restricted", restricted) {
				response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
				return true
			}
		}
		writeSimpleSuccess(c, "游客访问限制状态已更新")
	case path == "/api/admin/settings" && method == http.MethodGet:
		h.adminSettings(c)
	case path == "/api/admin/system-settings" && method == http.MethodGet:
		h.adminSystemSettings(c)
	case path == "/api/admin/system-settings" && method == http.MethodPut:
		h.adminUpdateSettings(c)
	case path == "/api/admin/image-watermark/extract" && method == http.MethodPost:
		h.adminExtractImageWatermark(c)
	case path == "/api/admin/reset-all-onboarding" && method == http.MethodPost:
		h.adminResetAllOnboarding(c)
	case strings.HasPrefix(path, "/api/admin/users/") && strings.HasSuffix(path, "/reset-onboarding") && method == http.MethodPost:
		h.adminResetUserOnboarding(c, adminResetOnboardingUserID(path))
	case path == "/api/admin/stats/overview" && method == http.MethodGet:
		h.adminStatsOverview(c)
	case path == "/api/admin/queues" && method == http.MethodGet:
		h.adminQueueStats(c)
	case strings.HasPrefix(path, "/api/admin/queues/") && method == http.MethodGet:
		h.adminQueueDispatch(c)
	case strings.HasPrefix(path, "/api/admin/queues/") && method == http.MethodPost:
		h.adminQueueDispatch(c)
	case strings.HasPrefix(path, "/api/admin/queues/") && method == http.MethodDelete:
		h.adminQueueDispatch(c)
	case path == "/api/admin/queue-names" && method == http.MethodGet:
		h.adminQueueNames(c)
	case path == "/api/admin/redis-maintenance" && method == http.MethodGet:
		h.AdminRedisMaintenance(c)
	case path == "/api/admin/redis-maintenance" && method == http.MethodPut:
		h.AdminRedisMaintenanceUpdate(c)
	case path == "/api/admin/redis-maintenance/run" && method == http.MethodPost:
		h.AdminRedisMaintenanceRun(c)
	case strings.HasPrefix(path, "/api/admin/file-recycle-bin"):
		h.AdminFileRecycleBin(c)
	case path == "/api/admin/system-logs" && method == http.MethodGet:
		h.adminSystemLogs(c)
	case path == "/api/admin/performance" && method == http.MethodGet:
		h.adminPerformance(c)
	case path == "/api/admin/logs/access" && method == http.MethodGet:
		h.AdminAccessLogs(c)
	case path == "/api/admin/logs/security" && method == http.MethodGet:
		h.AdminSecurityAuditLogs(c)
	case path == "/api/admin/logs/access/analytics" && method == http.MethodGet:
		h.AdminAccessLogAnalytics(c)
	case path == "/api/admin/monitor/activities" && method == http.MethodGet:
		h.adminMonitorActivities(c)
	case path == "/api/admin/test-users" && method == http.MethodGet:
		h.adminTestUsers(c)
	case path == "/api/admin/apk-files" && method == http.MethodGet:
		h.adminAPKFiles(c)
	case strings.HasPrefix(path, "/api/admin/batch-upload/"):
		h.adminBatchUpload(c)
	case strings.HasPrefix(path, "/api/admin/app-versions"):
		h.adminAppVersions(c)
	case path == "/api/admin/app-versions/stats" && method == http.MethodGet:
		h.adminAppVersionStats(c)
	case path == "/api/admin/app-versions/last-form-data" && method == http.MethodGet:
		h.adminLastAppForm(c)
	case path == "/api/admin/app-versions/last-form-data" && method == http.MethodPost:
		h.adminSaveLastAppForm(c)
	case path == "/api/admin/recommendation/config" && method == http.MethodGet:
		h.adminRecommendationConfig(c)
	case path == "/api/admin/recommendation/config" && method == http.MethodPut:
		h.adminSaveRecommendationConfig(c)
	case path == "/api/admin/recommendation/post-configs/batch" && method == http.MethodPost:
		h.adminRecommendationPostBatchCompat(c)
	case path == "/api/admin/recommendation/push" && method == http.MethodPost:
		h.adminRecommendationPushCompat(c)
	case strings.HasPrefix(path, "/api/admin/recommendation/post-configs"):
		h.adminRecommendationPostConfigs(c)
	case strings.HasPrefix(path, "/api/admin/recommendation/user-configs"):
		h.adminRecommendationUserConfigs(c)
	case path == "/api/admin/posts" && method == http.MethodPost:
		h.adminPostCreateCompat(c)
	case path == "/api/admin/posts/set-private" && method == http.MethodPut:
		h.adminPostsVisibility(c, "private")
	case path == "/api/admin/posts/set-public" && method == http.MethodPut:
		h.adminPostsVisibility(c, "public")
	case path == "/api/admin/posts/set-category" && method == http.MethodPut:
		h.adminPostsSetCategory(c)
	case path == "/api/admin/posts/transfer" && method == http.MethodPost:
		h.adminPostsTransfer(c)
	case path == "/api/admin/posts-quality" && method == http.MethodGet:
		h.adminPostsQualityList(c)
	case path == "/api/admin/posts-quality/batch" && method == http.MethodPut:
		h.adminPostsQualityBatchCompat(c)
	case strings.HasSuffix(path, "/quality") && strings.HasPrefix(path, "/api/admin/posts/") && method == http.MethodPut:
		h.adminPostQualityCompat(c)
	case strings.HasPrefix(path, "/api/admin/posts/") && method == http.MethodPut:
		h.adminPostUpdateCompat(c)
	case strings.HasPrefix(path, "/api/admin/posts/") && method == http.MethodDelete:
		h.adminPostDeleteCompat(c)
	case path == "/api/admin/posts" && method == http.MethodDelete:
		h.adminPostsBulkDeleteCompat(c)
	case path == "/api/admin/content-review/settings" && method == http.MethodGet:
		h.adminContentReviewSettings(c)
	case path == "/api/admin/content-review/settings" && method == http.MethodPut:
		h.adminUpdateContentReviewSettings(c)
	case strings.Contains(path, "/content-review/") && strings.HasSuffix(path, "/approve") && method == http.MethodPut:
		h.adminContentReviewAction(c, auditStatusOK)
	case strings.Contains(path, "/content-review/") && strings.HasSuffix(path, "/reject") && method == http.MethodPut:
		h.adminContentReviewAction(c, 2)
	case strings.Contains(path, "/content-review/") && strings.HasSuffix(path, "/retry") && method == http.MethodPut:
		h.adminContentReviewRetry(c)
	case strings.Contains(path, "/audit/") && strings.HasSuffix(path, "/approve") && method == http.MethodPut:
		h.adminAuditAction(c, auditStatusOK)
	case strings.Contains(path, "/audit/") && strings.HasSuffix(path, "/reject") && method == http.MethodPut:
		h.adminAuditAction(c, 2)
	case path == "/api/admin/banned-words/import" && method == http.MethodPost:
		h.adminBannedWordsImport(c)
	case path == "/api/admin/banned-words/export" && method == http.MethodGet:
		h.adminBannedWordsExport(c)
	case path == "/api/admin/notification-templates/defaults" && method == http.MethodGet:
		writeSuccess(c, matrixMsgOK, defaultNotificationTemplates())
	case path == "/api/admin/notification-templates/preview" && method == http.MethodPost:
		body := readBodyMap(c)
		writeSuccess(c, matrixMsgOK, gin.H{"subject": toString(body["subject"]), "content": toString(body["content"])})
	case strings.Contains(path, "/notification-templates/") && (strings.HasSuffix(path, "/test-email") || strings.HasSuffix(path, "/test-discord")):
		h.adminNotificationTemplateTest(c)
	case strings.Contains(path, "/system-notifications/") && strings.HasSuffix(path, "/resend") && method == http.MethodPost:
		writeSimpleSuccess(c, "系统通知已重新发送")
	case strings.Contains(path, "/user-toolbar/") && strings.HasSuffix(path, "/toggle-active") && method == http.MethodPut:
		h.adminToggleActive(c, "user_toolbar")
	case strings.Contains(path, "/users/") && strings.HasSuffix(path, "/earnings-info") && method == http.MethodGet:
		h.adminUserEarnings(c)
	case strings.Contains(path, "/users/") && strings.HasSuffix(path, "/transfer-to-earnings") && method == http.MethodPost:
		h.adminUserTransferEarnings(c)
	case strings.Contains(path, "/users/") && strings.HasSuffix(path, "/add-earnings") && method == http.MethodPost:
		h.adminUserAdjustEarnings(c, true)
	case strings.Contains(path, "/users/") && strings.HasSuffix(path, "/deduct-earnings") && method == http.MethodPost:
		h.adminUserAdjustEarnings(c, false)
	case strings.HasPrefix(path, "/api/admin/sessions"):
		h.adminSessions(c)
	case path == "/api/admin/quality-reward-settings" && method == http.MethodGet:
		h.adminQualityRewardSettings(c)
	case strings.HasPrefix(path, "/api/admin/quality-reward-settings/") && method == http.MethodPut:
		h.adminUpdateQualityRewardSetting(c)
	case path == "/api/admin/videos/missing-covers/stats" && method == http.MethodGet:
		h.adminMissingCoverStats(c)
	case path == "/api/admin/videos/generate-missing-covers" && method == http.MethodPost:
		h.adminGenerateMissingCovers(c)
	default:
		return false
	}
	return true
}
