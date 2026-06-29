# Route Matrix

Historical compatibility matrix captured before the Express backend was
retired. Legacy source paths are audit metadata only; the Gin backend does not
depend on the old Express runtime.

## Summary

- Generated at: 2026-06-16T00:00:00.000Z
- Mounted modules: 27
- Module routes: 494
- Inline routes: 12
- Total API routes: 506
- WebSocket upgrades: 1

## Method Counts

| Method | Count |
| --- | ---: |
| GET | 230 |
| POST | 140 |
| DELETE | 70 |
| PUT | 64 |
| ALL | 1 |
| PATCH | 1 |

## Module Counts

| Source | Count |
| --- | ---: |
| backend/app.js | 12 |
| backend/routes/admin.js | 267 |
| backend/routes/announcements.js | 2 |
| backend/routes/auth.js | 29 |
| backend/routes/balance.js | 7 |
| backend/routes/categories.js | 3 |
| backend/routes/comments.js | 4 |
| backend/routes/coupon.js | 11 |
| backend/routes/creatorCenter.js | 9 |
| backend/routes/dislikes.js | 2 |
| backend/routes/feedback.js | 5 |
| backend/routes/file.js | 3 |
| backend/routes/im.js | 10 |
| backend/routes/invite.js | 8 |
| backend/routes/license.js | 1 |
| backend/routes/likes.js | 2 |
| backend/routes/notifications.js | 10 |
| backend/routes/openApi.js | 3 |
| backend/routes/points.js | 24 |
| backend/routes/posts.js | 20 |
| backend/routes/pyvideoProxy.js | 1 |
| backend/routes/reports.js | 2 |
| backend/routes/search.js | 1 |
| backend/routes/stats.js | 1 |
| backend/routes/tags.js | 2 |
| backend/routes/upload.js | 11 |
| backend/routes/users.js | 39 |
| backend/routes/withdraw.js | 9 |
| backend/routes/ai.js | 8 |

## HTTP Routes

| Method | Path | Auth | Source | Middleware | Status |
| --- | --- | --- | --- | --- | --- |
| GET | `/api/${JWT_TEST_TOKEN_PATH}` | public | backend/app.js:193 |  | native-gin |
| POST | `/api/${JWT_TEST_TOKEN_PATH}` | public | backend/app.js:148 |  | native-gin |
| GET | `/api/${SWAGGER_DOCS_PATH}.json` | public | backend/app.js:142 |  | native-gin |
| DELETE | `/api/admin/admins` | admin | backend/routes/admin.js:2189 | adminAuth | native-gin |
| GET | `/api/admin/admins` | admin | backend/routes/admin.js:2078 | adminAuth | native-gin |
| POST | `/api/admin/admins` | admin | backend/routes/admin.js:2130 | adminAuth | native-gin |
| DELETE | `/api/admin/admins/:id` | admin | backend/routes/admin.js:2178 | adminAuth | native-gin |
| GET | `/api/admin/admins/:id` | admin | backend/routes/admin.js:2111 | adminAuth | native-gin |
| PUT | `/api/admin/admins/:id` | admin | backend/routes/admin.js:2156 | adminAuth | native-gin |
| GET | `/api/admin/ai-moderation-logs` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/ai-review-status` | admin | backend/routes/admin.js:22 | adminAuth | native-gin |
| POST | `/api/admin/ai-review-toggle` | admin | backend/routes/admin.js:37 | adminAuth | native-gin |
| GET | `/api/admin/access-block/import-sources` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| POST | `/api/admin/access-block/import-sources` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| PUT | `/api/admin/access-block/import-sources/:id` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| DELETE | `/api/admin/access-block/import-sources/:id` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| POST | `/api/admin/access-block/import-sources/:id/sync` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/access-block/rules` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| POST | `/api/admin/access-block/rules` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| PUT | `/api/admin/access-block/rules/:id` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| DELETE | `/api/admin/access-block/rules/:id` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| POST | `/api/admin/access-block/rules/batch` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| POST | `/api/admin/ai/generate/stream` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/ai/jobs` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| POST | `/api/admin/ai/jobs` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/ai/jobs/:jobId` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/ai/jobs/:jobId/stream` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| POST | `/api/admin/ai/jobs/:jobId/cancel` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/ai/logs` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| POST | `/api/admin/ai/moderation/debug` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/ai/settings` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| PUT | `/api/admin/ai/settings` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/announcements` | admin | backend/routes/admin.js:7133 | adminAuth | native-gin |
| POST | `/api/admin/announcements` | admin | backend/routes/admin.js:7208 | adminAuth | native-gin |
| DELETE | `/api/admin/announcements/:id` | admin | backend/routes/admin.js:7289 | adminAuth | native-gin |
| GET | `/api/admin/announcements/:id` | admin | backend/routes/admin.js:7183 | adminAuth | native-gin |
| PUT | `/api/admin/announcements/:id` | admin | backend/routes/admin.js:7247 | adminAuth | native-gin |
| GET | `/api/admin/apk-files` | admin | backend/routes/admin.js:3319 | adminAuth | native-gin |
| DELETE | `/api/admin/app-versions` | admin | backend/routes/admin.js:5905 | adminAuth | native-gin |
| GET | `/api/admin/app-versions` | admin | backend/routes/admin.js:5587 | adminAuth | native-gin |
| POST | `/api/admin/app-versions` | admin | backend/routes/admin.js:5775 | adminAuth | native-gin |
| DELETE | `/api/admin/app-versions/:id` | admin | backend/routes/admin.js:5883 | adminAuth | native-gin |
| GET | `/api/admin/app-versions/:id` | admin | backend/routes/admin.js:5754 | adminAuth | native-gin |
| PUT | `/api/admin/app-versions/:id` | admin | backend/routes/admin.js:5830 | adminAuth | native-gin |
| GET | `/api/admin/app-versions/last-form-data` | admin | backend/routes/admin.js:5720 | adminAuth | native-gin |
| POST | `/api/admin/app-versions/last-form-data` | admin | backend/routes/admin.js:5739 | adminAuth | native-gin |
| GET | `/api/admin/app-versions/stats` | admin | backend/routes/admin.js:5628 | adminAuth | native-gin |
| DELETE | `/api/admin/audit` | admin | backend/routes/admin.js:5549 | adminAuth | native-gin |
| GET | `/api/admin/audit` | admin | backend/routes/admin.js:5309 | adminAuth | native-gin |
| POST | `/api/admin/audit` | admin | backend/routes/admin.js:5413 | adminAuth | native-gin |
| DELETE | `/api/admin/audit/:id` | admin | backend/routes/admin.js:5524 | adminAuth | native-gin |
| GET | `/api/admin/audit/:id` | admin | backend/routes/admin.js:5375 | adminAuth | native-gin |
| PUT | `/api/admin/audit/:id` | admin | backend/routes/admin.js:5438 | adminAuth | native-gin |
| PUT | `/api/admin/audit/:id/approve` | admin | backend/routes/admin.js:5465 | adminAuth | native-gin |
| PUT | `/api/admin/audit/:id/reject` | admin | backend/routes/admin.js:5501 | adminAuth | native-gin |
| GET | `/api/admin/balance-transactions` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| POST | `/api/admin/balance-transactions/:id/compensate` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/banned-word-categories` | admin | backend/routes/admin.js:3191 | adminAuth | native-gin |
| POST | `/api/admin/banned-word-categories` | admin | backend/routes/admin.js:3221 | adminAuth | native-gin |
| DELETE | `/api/admin/banned-word-categories/:id` | admin | backend/routes/admin.js:3286 | adminAuth | native-gin |
| PUT | `/api/admin/banned-word-categories/:id` | admin | backend/routes/admin.js:3250 | adminAuth | native-gin |
| DELETE | `/api/admin/banned-words` | admin | backend/routes/admin.js:3105 | adminAuth | native-gin |
| GET | `/api/admin/banned-words` | admin | backend/routes/admin.js:2967 | adminAuth | native-gin |
| POST | `/api/admin/banned-words` | admin | backend/routes/admin.js:3032 | adminAuth | native-gin |
| DELETE | `/api/admin/banned-words/:id` | admin | backend/routes/admin.js:3089 | adminAuth | native-gin |
| GET | `/api/admin/banned-words/:id` | admin | backend/routes/admin.js:3008 | adminAuth | native-gin |
| PUT | `/api/admin/banned-words/:id` | admin | backend/routes/admin.js:3060 | adminAuth | native-gin |
| GET | `/api/admin/banned-words/export` | admin | backend/routes/admin.js:3158 | adminAuth | native-gin |
| POST | `/api/admin/banned-words/import` | admin | backend/routes/admin.js:3125 | adminAuth | native-gin |
| POST | `/api/admin/batch-upload/async-create` | admin | backend/routes/admin.js:3583 | adminAuth | native-gin |
| POST | `/api/admin/batch-upload/create` | admin | backend/routes/admin.js:3422 | adminAuth | native-gin |
| DELETE | `/api/admin/batch-upload/files` | admin | backend/routes/admin.js:3550 | adminAuth | native-gin |
| GET | `/api/admin/batch-upload/files` | admin | backend/routes/admin.js:3366 | adminAuth | native-gin |
| GET | `/api/admin/batch-upload/status/:batchId` | admin | backend/routes/admin.js:3743 | adminAuth | native-gin |
| DELETE | `/api/admin/categories` | admin | backend/routes/admin.js:2457 | adminAuth | native-gin |
| GET | `/api/admin/categories` | admin | backend/routes/admin.js:2303 | adminAuth | native-gin |
| POST | `/api/admin/categories` | admin | backend/routes/admin.js:2365 | adminAuth | native-gin |
| DELETE | `/api/admin/categories/:id` | admin | backend/routes/admin.js:2440 | adminAuth | native-gin |
| GET | `/api/admin/categories/:id` | admin | backend/routes/admin.js:2346 | adminAuth | native-gin |
| PUT | `/api/admin/categories/:id` | admin | backend/routes/admin.js:2398 | adminAuth | native-gin |
| DELETE | `/api/admin/collections` | admin | backend/routes/admin.js:1531 | adminAuth | native-gin |
| GET | `/api/admin/collections` | admin | backend/routes/admin.js:1391 | adminAuth | native-gin |
| POST | `/api/admin/collections` | admin | backend/routes/admin.js:1459 | adminAuth | native-gin |
| DELETE | `/api/admin/collections/:id` | admin | backend/routes/admin.js:1520 | adminAuth | native-gin |
| GET | `/api/admin/collections/:id` | admin | backend/routes/admin.js:1437 | adminAuth | native-gin |
| PUT | `/api/admin/collections/:id` | admin | backend/routes/admin.js:1495 | adminAuth | native-gin |
| DELETE | `/api/admin/comments` | admin | backend/routes/admin.js:1099 | adminAuth | native-gin |
| GET | `/api/admin/comments` | admin | backend/routes/admin.js:945 | adminAuth | native-gin |
| POST | `/api/admin/comments` | admin | backend/routes/admin.js:1020 | adminAuth | native-gin |
| DELETE | `/api/admin/comments/:id` | admin | backend/routes/admin.js:1081 | adminAuth | native-gin |
| GET | `/api/admin/comments/:id` | admin | backend/routes/admin.js:998 | adminAuth | native-gin |
| PUT | `/api/admin/comments/:id` | admin | backend/routes/admin.js:1063 | adminAuth | native-gin |
| DELETE | `/api/admin/content-review` | admin | backend/routes/admin.js:2693 | adminAuth | native-gin |
| GET | `/api/admin/content-review` | admin | backend/routes/admin.js:2485 | adminAuth | native-gin |
| POST | `/api/admin/content-review` | admin | backend/routes/admin.js:2620 | adminAuth | native-gin |
| DELETE | `/api/admin/content-review/:id` | admin | backend/routes/admin.js:2669 | adminAuth | native-gin |
| GET | `/api/admin/content-review/:id` | admin | backend/routes/admin.js:2601 | adminAuth | native-gin |
| PUT | `/api/admin/content-review/:id` | admin | backend/routes/admin.js:2645 | adminAuth | native-gin |
| PUT | `/api/admin/content-review/:id/approve` | admin | backend/routes/admin.js:2710 | adminAuth | native-gin |
| PUT | `/api/admin/content-review/:id/reject` | admin | backend/routes/admin.js:2734 | adminAuth | native-gin |
| PUT | `/api/admin/content-review/:id/retry` | admin | backend/routes/admin.js:2758 | adminAuth | native-gin |
| GET | `/api/admin/content-review/settings` | admin | backend/routes/admin.js:2538 | adminAuth | native-gin |
| PUT | `/api/admin/content-review/settings` | admin | backend/routes/admin.js:2553 | adminAuth | native-gin |
| GET | `/api/admin/dashboard/hot-content` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/dashboard/overview` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/dashboard/trends` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/database/index-audit` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/database/overview` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| POST | `/api/admin/database/repair` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/database/tables` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/database/tables/:table/columns` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| POST | `/api/admin/database/vacuum-analyze` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/database/vacuum-config` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| PUT | `/api/admin/database/vacuum-config` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/feedback` | admin | backend/routes/admin.js:6743 | adminAuth | native-gin |
| DELETE | `/api/admin/feedback/:id` | admin | backend/routes/admin.js:6846 | adminAuth | native-gin |
| GET | `/api/admin/feedback/:id` | admin | backend/routes/admin.js:6792 | adminAuth | native-gin |
| PUT | `/api/admin/feedback/:id` | admin | backend/routes/admin.js:6818 | adminAuth | native-gin |
| DELETE | `/api/admin/follows` | admin | backend/routes/admin.js:1693 | adminAuth | native-gin |
| GET | `/api/admin/follows` | admin | backend/routes/admin.js:1549 | adminAuth | native-gin |
| POST | `/api/admin/follows` | admin | backend/routes/admin.js:1618 | adminAuth | native-gin |
| DELETE | `/api/admin/follows/:id` | admin | backend/routes/admin.js:1682 | adminAuth | native-gin |
| GET | `/api/admin/follows/:id` | admin | backend/routes/admin.js:1596 | adminAuth | native-gin |
| PUT | `/api/admin/follows/:id` | admin | backend/routes/admin.js:1658 | adminAuth | native-gin |
| GET | `/api/admin/guest-access-status` | admin | backend/routes/admin.js:54 | adminAuth | native-gin |
| POST | `/api/admin/guest-access-toggle` | admin | backend/routes/admin.js:69 | adminAuth | native-gin |
| POST | `/api/admin/image-watermark/extract` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| DELETE | `/api/admin/licenses` | admin | backend/routes/admin.js:6573 | adminAuth | native-gin |
| GET | `/api/admin/licenses` | admin | backend/routes/admin.js:6451 | adminAuth | native-gin |
| POST | `/api/admin/licenses` | admin | backend/routes/admin.js:6501 | adminAuth | native-gin |
| DELETE | `/api/admin/licenses/:id` | admin | backend/routes/admin.js:6561 | adminAuth | native-gin |
| GET | `/api/admin/licenses/:id` | admin | backend/routes/admin.js:6484 | adminAuth | native-gin |
| PUT | `/api/admin/licenses/:id` | admin | backend/routes/admin.js:6534 | adminAuth | native-gin |
| DELETE | `/api/admin/likes` | admin | backend/routes/admin.js:1373 | adminAuth | native-gin |
| GET | `/api/admin/likes` | admin | backend/routes/admin.js:1234 | adminAuth | native-gin |
| POST | `/api/admin/likes` | admin | backend/routes/admin.js:1299 | adminAuth | native-gin |
| DELETE | `/api/admin/likes/:id` | admin | backend/routes/admin.js:1362 | adminAuth | native-gin |
| GET | `/api/admin/likes/:id` | admin | backend/routes/admin.js:1280 | adminAuth | native-gin |
| PUT | `/api/admin/likes/:id` | admin | backend/routes/admin.js:1340 | adminAuth | native-gin |
| GET | `/api/admin/logs/access` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/logs/access/analytics` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/logs/balance` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/logs/points` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/logs/security` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/maintenance` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| PUT | `/api/admin/maintenance` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| POST | `/api/admin/maintenance/rotate-entry` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| DELETE | `/api/admin/media-library` | admin | backend/routes/admin.js:6723 | adminAuth | native-gin |
| GET | `/api/admin/media-library` | admin | backend/routes/admin.js:6631 | adminAuth | native-gin |
| POST | `/api/admin/media-library` | admin | backend/routes/admin.js:6650 | adminAuth, (req, res, next) => { mediaUploadParser.single('file')(req, res, (err) => { if (err) { return res.status(HTTP_STATUS.BAD_REQUEST).json({ code: RESPONSE_CODES.VALIDATION_ERROR, message: err.message || '文件上传失败' }) } next() }) } | native-gin |
| DELETE | `/api/admin/media-library/:id` | admin | backend/routes/admin.js:6708 | adminAuth | native-gin |
| GET | `/api/admin/media-library/public` | public | backend/routes/admin.js:6612 |  | native-gin |
| DELETE | `/api/admin/file-recycle-bin` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/file-recycle-bin` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/file-recycle-bin/:id/inspect` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/file-recycle-bin/:id/preview` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/file-recycle-bin/:id/download` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| DELETE | `/api/admin/file-recycle-bin/:id` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| POST | `/api/admin/file-recycle-bin/run-cleanup` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/monitor/activities` | admin | backend/routes/admin.js:2207 | adminAuth | native-gin |
| DELETE | `/api/admin/notification-templates` | admin | backend/routes/admin.js:5145 | adminAuth | native-gin |
| GET | `/api/admin/notification-templates` | admin | backend/routes/admin.js:4948 | adminAuth | native-gin |
| POST | `/api/admin/notification-templates` | admin | backend/routes/admin.js:5021 | adminAuth | native-gin |
| DELETE | `/api/admin/notification-templates/:id` | admin | backend/routes/admin.js:5123 | adminAuth | native-gin |
| GET | `/api/admin/notification-templates/:id` | admin | backend/routes/admin.js:5000 | adminAuth | native-gin |
| PUT | `/api/admin/notification-templates/:id` | admin | backend/routes/admin.js:5079 | adminAuth | native-gin |
| POST | `/api/admin/notification-templates/:id/test-discord` | admin | backend/routes/admin.js:5252 | adminAuth | native-gin |
| POST | `/api/admin/notification-templates/:id/test-email` | admin | backend/routes/admin.js:5208 | adminAuth | native-gin |
| GET | `/api/admin/notification-templates/defaults` | admin | backend/routes/admin.js:4985 | adminAuth | native-gin |
| POST | `/api/admin/notification-templates/preview` | admin | backend/routes/admin.js:5181 | adminAuth | native-gin |
| GET | `/api/admin/observability/access-log` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/observability/events` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| DELETE | `/api/admin/open-apis` | admin | backend/routes/admin.js:6431 | adminAuth | native-gin |
| GET | `/api/admin/open-apis` | admin | backend/routes/admin.js:6302 | adminAuth | native-gin |
| POST | `/api/admin/open-apis` | admin | backend/routes/admin.js:6354 | adminAuth | native-gin |
| DELETE | `/api/admin/open-apis/:id` | admin | backend/routes/admin.js:6419 | adminAuth | native-gin |
| GET | `/api/admin/open-apis/:id` | admin | backend/routes/admin.js:6334 | adminAuth | native-gin |
| PUT | `/api/admin/open-apis/:id` | admin | backend/routes/admin.js:6394 | adminAuth | native-gin |
| GET | `/api/admin/performance` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| DELETE | `/api/admin/posts` | admin | backend/routes/admin.js:868 | adminAuth | native-gin |
| GET | `/api/admin/posts` | admin | backend/routes/admin.js:392 | adminAuth | native-gin |
| POST | `/api/admin/posts` | admin | backend/routes/admin.js:517 | adminAuth | native-gin |
| GET | `/api/admin/posts-quality` | admin | backend/routes/admin.js:4117 | adminAuth | native-gin |
| PUT | `/api/admin/posts-quality/batch` | admin | backend/routes/admin.js:4318 | adminAuth | native-gin |
| DELETE | `/api/admin/posts/:id` | admin | backend/routes/admin.js:829 | adminAuth | native-gin |
| GET | `/api/admin/posts/:id` | admin | backend/routes/admin.js:466 | adminAuth | native-gin |
| PUT | `/api/admin/posts/:id` | admin | backend/routes/admin.js:710 | adminAuth | native-gin |
| PUT | `/api/admin/posts/:id/quality` | admin | backend/routes/admin.js:4181 | adminAuth | native-gin |
| PUT | `/api/admin/posts/set-category` | admin | backend/routes/admin.js:668 | adminAuth | native-gin |
| PUT | `/api/admin/posts/set-private` | admin | backend/routes/admin.js:608 | adminAuth | native-gin |
| PUT | `/api/admin/posts/set-public` | admin | backend/routes/admin.js:638 | adminAuth | native-gin |
| POST | `/api/admin/posts/transfer` | admin | backend/routes/admin.js:903 | adminAuth | native-gin |
| GET | `/api/admin/quality-reward-settings` | admin | backend/routes/admin.js:4032 | adminAuth | native-gin |
| PUT | `/api/admin/quality-reward-settings/:id` | admin | backend/routes/admin.js:4085 | adminAuth | native-gin |
| GET | `/api/admin/queue-names` | admin | backend/routes/admin.js:2952 | adminAuth | native-gin |
| GET | `/api/admin/queues` | admin | backend/routes/admin.js:2864 | adminAuth | native-gin |
| DELETE | `/api/admin/queues/:name` | admin | backend/routes/admin.js:2935 | adminAuth | native-gin |
| GET | `/api/admin/queues/:name/jobs` | admin | backend/routes/admin.js:2879 | adminAuth | native-gin |
| GET | `/api/admin/queues/:name/jobs/:jobId` | admin | backend/routes/admin.js:2897 | adminAuth | native-gin |
| POST | `/api/admin/queues/:name/jobs/:jobId/retry` | admin | backend/routes/admin.js:2918 | adminAuth | native-gin |
| GET | `/api/admin/recommendation/config` | admin | backend/routes/admin.js:5927 | adminAuth | native-gin |
| PUT | `/api/admin/recommendation/config` | admin | backend/routes/admin.js:5938 | adminAuth | native-gin |
| GET | `/api/admin/recommendation/post-configs` | admin | backend/routes/admin.js:5950 | adminAuth | native-gin |
| POST | `/api/admin/recommendation/post-configs` | admin | backend/routes/admin.js:6045 | adminAuth | native-gin |
| DELETE | `/api/admin/recommendation/post-configs/:id` | admin | backend/routes/admin.js:6119 | adminAuth | native-gin |
| PUT | `/api/admin/recommendation/post-configs/:id` | admin | backend/routes/admin.js:6088 | adminAuth | native-gin |
| POST | `/api/admin/recommendation/post-configs/batch` | admin | backend/routes/admin.js:5994 | adminAuth | native-gin |
| POST | `/api/admin/recommendation/push` | admin | backend/routes/admin.js:6258 | adminAuth | native-gin |
| GET | `/api/admin/recommendation/user-configs` | admin | backend/routes/admin.js:6130 | adminAuth | native-gin |
| POST | `/api/admin/recommendation/user-configs` | admin | backend/routes/admin.js:6167 | adminAuth | native-gin |
| DELETE | `/api/admin/recommendation/user-configs/:id` | admin | backend/routes/admin.js:6247 | adminAuth | native-gin |
| PUT | `/api/admin/recommendation/user-configs/:id` | admin | backend/routes/admin.js:6209 | adminAuth | native-gin |
| GET | `/api/admin/redis-maintenance` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| PUT | `/api/admin/redis-maintenance` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| POST | `/api/admin/redis-maintenance/run` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/reports` | admin | backend/routes/admin.js:6988 | adminAuth | native-gin |
| DELETE | `/api/admin/reports/:id` | admin | backend/routes/admin.js:7116 | adminAuth | native-gin |
| GET | `/api/admin/reports/:id` | admin | backend/routes/admin.js:7040 | adminAuth | native-gin |
| PUT | `/api/admin/reports/:id` | admin | backend/routes/admin.js:7073 | adminAuth | native-gin |
| POST | `/api/admin/reset-all-onboarding` | admin | backend/routes/admin.js:373 | adminAuth | native-gin |
| DELETE | `/api/admin/sessions` | admin | backend/routes/admin.js:1885 | adminAuth | native-gin |
| GET | `/api/admin/sessions` | admin | backend/routes/admin.js:1711 | adminAuth | native-gin |
| POST | `/api/admin/sessions` | admin | backend/routes/admin.js:1819 | adminAuth | native-gin |
| DELETE | `/api/admin/sessions/:id` | admin | backend/routes/admin.js:1874 | adminAuth | native-gin |
| GET | `/api/admin/sessions/:id` | admin | backend/routes/admin.js:1787 | adminAuth | native-gin |
| PUT | `/api/admin/sessions/:id` | admin | backend/routes/admin.js:1852 | adminAuth | native-gin |
| GET | `/api/admin/settings` | admin | backend/routes/admin.js:79 | adminAuth | native-gin |
| GET | `/api/admin/stats/overview` | admin | backend/routes/admin.js:2841 | adminAuth | native-gin |
| GET | `/api/admin/system-logs` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| DELETE | `/api/admin/system-notifications` | admin | backend/routes/admin.js:4893 | adminAuth | native-gin |
| GET | `/api/admin/system-notifications` | admin | backend/routes/admin.js:4703 | adminAuth | native-gin |
| POST | `/api/admin/system-notifications` | admin | backend/routes/admin.js:4782 | adminAuth | native-gin |
| DELETE | `/api/admin/system-notifications/:id` | admin | backend/routes/admin.js:4872 | adminAuth | native-gin |
| GET | `/api/admin/system-notifications/:id` | admin | backend/routes/admin.js:4758 | adminAuth | native-gin |
| PUT | `/api/admin/system-notifications/:id` | admin | backend/routes/admin.js:4837 | adminAuth | native-gin |
| POST | `/api/admin/system-notifications/:id/resend` | admin | backend/routes/admin.js:4916 | adminAuth | native-gin |
| GET | `/api/admin/system-settings` | admin | backend/routes/admin.js:94 | adminAuth | native-gin |
| PUT | `/api/admin/system-settings` | admin | backend/routes/admin.js:231 | adminAuth | native-gin |
| POST | `/api/admin/system-update/check` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| PUT | `/api/admin/system-update/config` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/system-update/releases` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| POST | `/api/admin/system-update/run` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| GET | `/api/admin/system-update/status` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| DELETE | `/api/admin/tags` | admin | backend/routes/admin.js:1216 | adminAuth | native-gin |
| GET | `/api/admin/tags` | admin | backend/routes/admin.js:1117 | adminAuth | native-gin |
| POST | `/api/admin/tags` | admin | backend/routes/admin.js:1163 | adminAuth | native-gin |
| DELETE | `/api/admin/tags/:id` | admin | backend/routes/admin.js:1204 | adminAuth | native-gin |
| GET | `/api/admin/tags/:id` | admin | backend/routes/admin.js:1147 | adminAuth | native-gin |
| PUT | `/api/admin/tags/:id` | admin | backend/routes/admin.js:1183 | adminAuth | native-gin |
| GET | `/api/admin/test-users` | admin | backend/routes/admin.js:2287 | adminAuth | native-gin |
| DELETE | `/api/admin/user-toolbar` | admin | backend/routes/admin.js:3960 | adminAuth | native-gin |
| GET | `/api/admin/user-toolbar` | admin | backend/routes/admin.js:3782 | adminAuth | native-gin |
| POST | `/api/admin/user-toolbar` | admin | backend/routes/admin.js:3846 | adminAuth | native-gin |
| DELETE | `/api/admin/user-toolbar/:id` | admin | backend/routes/admin.js:3935 | adminAuth | native-gin |
| GET | `/api/admin/user-toolbar/:id` | admin | backend/routes/admin.js:3822 | adminAuth | native-gin |
| PUT | `/api/admin/user-toolbar/:id` | admin | backend/routes/admin.js:3887 | adminAuth | native-gin |
| PUT | `/api/admin/user-toolbar/:id/toggle-active` | admin | backend/routes/admin.js:3984 | adminAuth | native-gin |
| DELETE | `/api/admin/users` | admin | backend/routes/admin.js:2060 | adminAuth | native-gin |
| GET | `/api/admin/users` | admin | backend/routes/admin.js:1902 | adminAuth | native-gin |
| POST | `/api/admin/users` | admin | backend/routes/admin.js:1967 | adminAuth | native-gin |
| DELETE | `/api/admin/users/:id` | admin | backend/routes/admin.js:2044 | adminAuth | native-gin |
| GET | `/api/admin/users/:id` | admin | backend/routes/admin.js:1943 | adminAuth | native-gin |
| PUT | `/api/admin/users/:id` | admin | backend/routes/admin.js:2011 | adminAuth | native-gin |
| POST | `/api/admin/users/:id/points` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| POST | `/api/admin/users/:id/reset-onboarding` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| POST | `/api/admin/users/:userId/add-earnings` | admin | backend/routes/admin.js:4575 | adminAuth | native-gin |
| POST | `/api/admin/users/:userId/deduct-earnings` | admin | backend/routes/admin.js:4638 | adminAuth | native-gin |
| GET | `/api/admin/users/:userId/earnings-info` | admin | backend/routes/admin.js:4456 | adminAuth | native-gin |
| POST | `/api/admin/users/:userId/transfer-to-earnings` | admin | backend/routes/admin.js:4487 | adminAuth | native-gin |
| POST | `/api/admin/users/batch-generate` | admin | backend/routes/admin.js:0 | adminAuth | native-gin |
| POST | `/api/admin/videos/generate-missing-covers` | admin | backend/routes/admin.js:6916 | adminAuth | native-gin |
| GET | `/api/admin/videos/missing-covers/stats` | admin | backend/routes/admin.js:6888 | adminAuth | native-gin |
| POST | `/api/ai/format-markdown/stream` | user | backend/routes/ai.js:0 | authenticateToken | native-gin |
| POST | `/api/ai/jobs` | user | backend/routes/ai.js:0 | authenticateToken | native-gin |
| GET | `/api/ai/jobs/:jobId` | user | backend/routes/ai.js:0 | authenticateToken | native-gin |
| GET | `/api/ai/jobs/:jobId/stream` | user | backend/routes/ai.js:0 | authenticateToken | native-gin |
| POST | `/api/ai/jobs/:jobId/cancel` | user | backend/routes/ai.js:0 | authenticateToken | native-gin |
| GET | `/api/ai/jobs/active` | user | backend/routes/ai.js:0 | authenticateToken | native-gin |
| GET | `/api/ai/publish-generation/settings` | user | backend/routes/ai.js:0 | authenticateToken | native-gin |
| POST | `/api/ai/publish-generation` | user | backend/routes/ai.js:0 | authenticateToken | native-gin |
| GET | `/api/announcements` | public | backend/routes/announcements.js:16 |  | native-gin |
| GET | `/api/announcements/:id` | public | backend/routes/announcements.js:75 |  | native-gin |
| GET | `/api/app/check-update` | public | backend/app.js:391 |  | native-gin |
| GET | `/api/app/download-config` | public | backend/app.js:390 |  | native-gin |
| POST | `/api/app/report-event` | public | backend/app.js:477 |  | native-gin |
| GET | `/api/auth/admin/admins` | user | backend/routes/auth.js:1181 | authenticateToken | native-gin |
| POST | `/api/auth/admin/admins` | user | backend/routes/auth.js:1235 | authenticateToken | native-gin |
| DELETE | `/api/auth/admin/admins/:id` | user | backend/routes/auth.js:1325 | authenticateToken | native-gin |
| PUT | `/api/auth/admin/admins/:id` | user | backend/routes/auth.js:1282 | authenticateToken | native-gin |
| PUT | `/api/auth/admin/admins/:id/password` | user | backend/routes/auth.js:1358 | authenticateToken | native-gin |
| POST | `/api/auth/admin/login` | public | backend/routes/auth.js:1096 |  | native-gin |
| GET | `/api/auth/admin/me` | user | backend/routes/auth.js:1151 | authenticateToken | native-gin |
| GET | `/api/auth/auth-config` | public | backend/routes/auth.js:96 |  | native-gin |
| POST | `/api/auth/bind-email` | user | backend/routes/auth.js:270 | authenticateToken | native-gin |
| GET | `/api/auth/captcha` | public | backend/routes/auth.js:127 |  | native-gin |
| GET | `/api/auth/check-user-id` | public | backend/routes/auth.js:185 |  | native-gin |
| GET | `/api/auth/email-config` | public | backend/routes/auth.js:116 |  | native-gin |
| POST | `/api/auth/file-token` | user | backend/routes/auth.js:2135 | authenticateToken | native-gin |
| POST | `/api/auth/login` | public | backend/routes/auth.js:803 |  | native-gin |
| POST | `/api/auth/logout` | user | backend/routes/auth.js:1036 | authenticateToken | native-gin |
| GET | `/api/auth/me` | user | backend/routes/auth.js:1057 | authenticateToken | native-gin |
| POST | `/api/auth/oauth2/app-token` | public | backend/routes/auth.js:0 |  | native-gin |
| GET | `/api/auth/oauth2/callback` | public | backend/routes/auth.js:1612 |  | native-gin |
| GET | `/api/auth/oauth2/login` | public | backend/routes/auth.js:1556 |  | native-gin |
| POST | `/api/auth/oauth2/mobile-session` | public | backend/routes/auth.js:0 |  | native-gin |
| POST | `/api/auth/oauth2/mobile-token` | public | backend/routes/auth.js:1927 |  | native-gin |
| POST | `/api/auth/refresh` | public | backend/routes/auth.js:902 |  | native-gin |
| POST | `/api/auth/register` | public | backend/routes/auth.js:545 |  | native-gin |
| POST | `/api/auth/reset-password` | public | backend/routes/auth.js:444 |  | native-gin |
| POST | `/api/auth/send-email-code` | public | backend/routes/auth.js:209 |  | native-gin |
| POST | `/api/auth/send-reset-code` | public | backend/routes/auth.js:339 |  | native-gin |
| POST | `/api/auth/token` | public | backend/routes/auth.js:966 |  | native-gin |
| DELETE | `/api/auth/unbind-email` | user | backend/routes/auth.js:501 | authenticateToken | native-gin |
| POST | `/api/auth/verify-reset-code` | public | backend/routes/auth.js:404 |  | native-gin |
| GET | `/api/balance/check-purchase/:postId` | user | backend/routes/balance.js:557 | authenticateToken | native-gin |
| GET | `/api/balance/config` | public | backend/routes/balance.js:66 |  | native-gin |
| GET | `/api/balance/local-points` | user | backend/routes/balance.js:114 | authenticateToken | native-gin |
| GET | `/api/balance/orders` | user | backend/routes/balance.js:484 | authenticateToken | native-gin |
| POST | `/api/balance/purchase-content` | user | backend/routes/balance.js:205 | authenticateToken | native-gin |
| GET | `/api/balance/recharge-config` | public | backend/routes/balance.js:77 |  | native-gin |
| GET | `/api/balance/user-balance` | user | backend/routes/balance.js:136 | authenticateToken | native-gin |
| GET | `/api/categories` | optional-note-guest-restricted | backend/routes/categories.js:9 | optionalAuthWithNoteGuestRestriction | native-gin |
| POST | `/api/categories` | user | backend/routes/categories.js:59 | authenticateToken | native-gin |
| GET | `/api/categories/hot` | optional-note-guest-restricted | backend/routes/categories.js:32 | optionalAuthWithNoteGuestRestriction | native-gin |
| GET | `/api/comments` | optional-note-guest-restricted | backend/routes/comments.js:47 | optionalAuthWithNoteGuestRestriction | native-gin |
| POST | `/api/comments` | user | backend/routes/comments.js:182 | authenticateToken | native-gin |
| DELETE | `/api/comments/:id` | user | backend/routes/comments.js:609 | authenticateToken | native-gin |
| GET | `/api/comments/:id/replies` | optional-note-guest-restricted | backend/routes/comments.js:512 | optionalAuthWithNoteGuestRestriction | native-gin |
| DELETE | `/api/coupon/admin/:id` | admin | backend/routes/coupon.js:529 | adminAuth | native-gin |
| PUT | `/api/coupon/admin/:id` | admin | backend/routes/coupon.js:473 | adminAuth | native-gin |
| POST | `/api/coupon/admin/:id/issue` | admin | backend/routes/coupon.js:566 | adminAuth | native-gin |
| GET | `/api/coupon/admin/:id/usages` | admin | backend/routes/coupon.js:685 | adminAuth | native-gin |
| POST | `/api/coupon/admin/create` | admin | backend/routes/coupon.js:404 | adminAuth | native-gin |
| GET | `/api/coupon/admin/list` | admin | backend/routes/coupon.js:340 | adminAuth | native-gin |
| GET | `/api/coupon/admin/stats` | admin | backend/routes/coupon.js:743 | adminAuth | native-gin |
| POST | `/api/coupon/claim` | user | backend/routes/coupon.js:98 | authenticateToken | native-gin |
| GET | `/api/coupon/my` | user | backend/routes/coupon.js:19 | authenticateToken | native-gin |
| POST | `/api/coupon/use` | user | backend/routes/coupon.js:239 | authenticateToken | native-gin |
| POST | `/api/coupon/validate` | user | backend/routes/coupon.js:196 | authenticateToken | native-gin |
| POST | `/api/creator-center/claim-incentive` | user | backend/routes/creatorCenter.js:1005 | authenticateToken | native-gin |
| GET | `/api/creator-center/config` | public | backend/routes/creatorCenter.js:294 |  | native-gin |
| GET | `/api/creator-center/earnings-log` | user | backend/routes/creatorCenter.js:708 | authenticateToken | native-gin |
| GET | `/api/creator-center/overview` | user | backend/routes/creatorCenter.js:328 | authenticateToken | native-gin |
| GET | `/api/creator-center/paid-content` | user | backend/routes/creatorCenter.js:795 | authenticateToken | native-gin |
| GET | `/api/creator-center/quality-rewards` | user | backend/routes/creatorCenter.js:1041 | authenticateToken | native-gin |
| GET | `/api/creator-center/stats` | user | backend/routes/creatorCenter.js:544 | authenticateToken | native-gin |
| GET | `/api/creator-center/trends` | user | backend/routes/creatorCenter.js:392 | authenticateToken | native-gin |
| POST | `/api/creator-center/withdraw` | user | backend/routes/creatorCenter.js:881 | authenticateToken | native-gin |
| GET | `/api/diagnostics/network` | public | backend/app.js:0 |  | native-gin |
| GET | `/api/dislikes` | user | backend/routes/dislikes.js:45 | authenticateToken | native-gin |
| POST | `/api/dislikes` | user | backend/routes/dislikes.js:8 | authenticateToken | native-gin |
| POST | `/api/feedback` | user | backend/routes/feedback.js:95 | authenticateToken | native-gin |
| GET | `/api/feedback/:id` | user | backend/routes/feedback.js:190 | authenticateToken | native-gin |
| GET | `/api/feedback/mine` | user | backend/routes/feedback.js:143 | authenticateToken | native-gin |
| POST | `/api/feedback/upload-image` | user | backend/routes/feedback.js:50 | authenticateToken, imageUpload.single('file') | native-gin |
| POST | `/api/feedback/upload-video` | user | backend/routes/feedback.js:75 | authenticateToken, videoUpload.single('file') | native-gin |
| GET | `/api/file/:type/:filename` | file-access | backend/routes/file.js:252 | authenticateFileAccess | native-gin |
| GET | `/api/file/:type/*` | file-access | backend/routes/file.js:387 | authenticateFileAccess | native-gin |
| GET | `/api/file/public/:filename` | public | backend/routes/file.js:179 |  | native-gin |
| GET | `/api/health` | public | backend/app.js:335 |  | native-gin |
| GET | `/api/im/conversations` | user | backend/routes/im.js:318 | authenticateToken, imRateLimit | native-gin |
| POST | `/api/im/conversations` | user | backend/routes/im.js:402 | authenticateToken, imRateLimit | native-gin |
| GET | `/api/im/conversations/:id/messages` | user | backend/routes/im.js:510 | authenticateToken, imRateLimit | native-gin |
| POST | `/api/im/conversations/:id/messages` | user | backend/routes/im.js:567 | authenticateToken, imRateLimit | native-gin |
| POST | `/api/im/messages/:id/delivered` | user | backend/routes/im.js:663 | authenticateToken, imRateLimit | native-gin |
| POST | `/api/im/messages/:id/read` | user | backend/routes/im.js:703 | authenticateToken, imRateLimit | native-gin |
| ALL | `/api/im/proxy/*` | user | backend/routes/im.js:92 | authenticateToken | native-gin |
| POST | `/api/im/session` | user | backend/routes/im.js:141 | authenticateToken | native-gin |
| GET | `/api/im/sync` | user | backend/routes/im.js:780 | authenticateToken, imRateLimit | native-gin |
| GET | `/api/im/users` | user | backend/routes/im.js:834 | authenticateToken, imRateLimit | native-gin |
| GET | `/api/image-archives/:jobId/download` | optional-note-guest-restricted | backend/routes/posts.js:0 | optionalAuthWithNoteGuestRestriction | native-gin |
| POST | `/api/image-watermark/extract` | user | backend/app.js:0 | authenticateToken, upload.single('file') | native-gin |
| PATCH | `/api/invite/admin/:id/toggle` | admin | backend/routes/invite.js:288 | adminAuth | native-gin |
| GET | `/api/invite/admin/list` | admin | backend/routes/invite.js:227 | adminAuth | native-gin |
| GET | `/api/invite/admin/overview` | admin | backend/routes/invite.js:346 | adminAuth | native-gin |
| POST | `/api/invite/admin/reward` | admin | backend/routes/invite.js:308 | adminAuth | native-gin |
| POST | `/api/invite/click/:code` | optional | backend/routes/invite.js:93 | optionalAuth | native-gin |
| GET | `/api/invite/info/:code` | public | backend/routes/invite.js:200 |  | native-gin |
| GET | `/api/invite/my-code` | user | backend/routes/invite.js:67 | authenticateToken | native-gin |
| GET | `/api/invite/stats` | user | backend/routes/invite.js:132 | authenticateToken | native-gin |
| POST | `/api/license/verify` | public | backend/routes/license.js:21 |  | native-gin |
| DELETE | `/api/likes` | user | backend/routes/likes.js:140 | authenticateToken | native-gin |
| POST | `/api/likes` | user | backend/routes/likes.js:9 | authenticateToken | native-gin |
| POST | `/api/maintenance/enter` | public | backend/app.js:0 |  | native-gin |
| GET | `/api/maintenance/status` | public | backend/app.js:0 |  | native-gin |
| GET | `/api/notifications` | user | backend/routes/notifications.js:66 | authenticateToken | native-gin |
| DELETE | `/api/notifications/:id` | user | backend/routes/notifications.js:245 | authenticateToken | native-gin |
| PUT | `/api/notifications/:id/read` | user | backend/routes/notifications.js:219 | authenticateToken | native-gin |
| GET | `/api/notifications/activities` | optional | backend/routes/notifications.js:10 | optionalAuth | native-gin |
| PUT | `/api/notifications/read-all` | user | backend/routes/notifications.js:202 | authenticateToken | native-gin |
| GET | `/api/notifications/system` | user | backend/routes/notifications.js:270 | authenticateToken | native-gin |
| POST | `/api/notifications/system/:id/confirm` | user | backend/routes/notifications.js:358 | authenticateToken | native-gin |
| DELETE | `/api/notifications/system/:id/dismiss` | user | backend/routes/notifications.js:394 | authenticateToken | native-gin |
| GET | `/api/notifications/system/popup` | user | backend/routes/notifications.js:331 | authenticateToken | native-gin |
| GET | `/api/notifications/unread-count` | user | backend/routes/notifications.js:167 | authenticateToken | native-gin |
| GET | `/api/open/posts` | open-api-key | backend/routes/openApi.js:56 | openApiAuth | native-gin |
| GET | `/api/open/posts/:id` | open-api-key | backend/routes/openApi.js:116 | openApiAuth | native-gin |
| GET | `/api/open/posts/:id/images` | open-api-key | backend/routes/openApi.js:162 | openApiAuth | native-gin |
| GET | `/api/points/admin/achievement-rules` | admin | backend/routes/points.js:0 | adminAuth | native-gin |
| POST | `/api/points/admin/achievement-rules` | admin | backend/routes/points.js:0 | adminAuth | native-gin |
| DELETE | `/api/points/admin/achievement-rules/:id` | admin | backend/routes/points.js:0 | adminAuth | native-gin |
| PUT | `/api/points/admin/achievement-rules/:id` | admin | backend/routes/points.js:0 | adminAuth | native-gin |
| POST | `/api/points/admin/clear-balances` | admin | backend/routes/points.js:0 | adminAuth | native-gin |
| GET | `/api/points/admin/gift-card-products` | admin | backend/routes/points.js:0 | adminAuth | native-gin |
| POST | `/api/points/admin/gift-card-products` | admin | backend/routes/points.js:0 | adminAuth | native-gin |
| DELETE | `/api/points/admin/gift-card-products/:id` | admin | backend/routes/points.js:0 | adminAuth | native-gin |
| PUT | `/api/points/admin/gift-card-products/:id` | admin | backend/routes/points.js:0 | adminAuth | native-gin |
| GET | `/api/points/admin/gift-card-products/:id/codes` | admin | backend/routes/points.js:0 | adminAuth | native-gin |
| POST | `/api/points/admin/gift-card-products/:id/import-codes` | admin | backend/routes/points.js:0 | adminAuth | native-gin |
| GET | `/api/points/admin/redemptions` | admin | backend/routes/points.js:0 | adminAuth | native-gin |
| POST | `/api/points/admin/reset-task-progress` | admin | backend/routes/points.js:0 | adminAuth | native-gin |
| GET | `/api/points/admin/settings` | admin | backend/routes/points.js:0 | adminAuth | native-gin |
| PUT | `/api/points/admin/settings` | admin | backend/routes/points.js:0 | adminAuth | native-gin |
| GET | `/api/points/admin/stats` | admin | backend/routes/points.js:0 | adminAuth | native-gin |
| GET | `/api/points/admin/tasks` | admin | backend/routes/points.js:0 | adminAuth | native-gin |
| POST | `/api/points/admin/tasks` | admin | backend/routes/points.js:0 | adminAuth | native-gin |
| DELETE | `/api/points/admin/tasks/:id` | admin | backend/routes/points.js:0 | adminAuth | native-gin |
| PUT | `/api/points/admin/tasks/:id` | admin | backend/routes/points.js:0 | adminAuth | native-gin |
| POST | `/api/points/gift-cards/:productId/redeem` | user | backend/routes/points.js:0 | authenticateToken | native-gin |
| GET | `/api/points/logs` | user | backend/routes/points.js:0 | authenticateToken | native-gin |
| GET | `/api/points/overview` | user | backend/routes/points.js:0 | authenticateToken | native-gin |
| GET | `/api/points/redemptions` | user | backend/routes/points.js:0 | authenticateToken | native-gin |
| GET | `/api/posts` | optional-note-guest-restricted | backend/routes/posts.js:872 | optionalAuthWithNoteGuestRestriction | native-gin |
| POST | `/api/posts` | user | backend/routes/posts.js:1333 | authenticateToken | native-gin |
| DELETE | `/api/posts/:id` | user | backend/routes/posts.js:1711 | authenticateToken | native-gin |
| GET | `/api/posts/:id` | optional-note-guest-restricted | backend/routes/posts.js:1200 | optionalAuthWithNoteGuestRestriction | native-gin |
| PUT | `/api/posts/:id` | user | backend/routes/posts.js:1549 | authenticateToken | native-gin |
| POST | `/api/posts/:id/collect` | user | backend/routes/posts.js:1764 | authenticateToken | native-gin |
| GET | `/api/posts/:id/comments` | optional-note-guest-restricted | backend/routes/posts.js:1084 | optionalAuthWithNoteGuestRestriction | native-gin |
| GET | `/api/posts/:id/image-archive` | optional-note-guest-restricted | backend/routes/posts.js:0 | optionalAuthWithNoteGuestRestriction | native-gin |
| POST | `/api/posts/:id/protected-package` | user | backend/routes/posts.js:1826 | authenticateToken | native-gin |
| GET | `/api/posts/:id/purchases` | user | backend/routes/posts.js:0 | authenticateToken | native-gin |
| GET | `/api/posts/following` | user | backend/routes/posts.js:680 | authenticateToken | native-gin |
| GET | `/api/posts/friends` | user | backend/routes/posts.js:486 | authenticateToken | native-gin |
| GET | `/api/posts/hot` | optional-note-guest-restricted | backend/routes/posts.js:315 | optionalAuthWithNoteGuestRestriction | native-gin |
| GET | `/api/posts/protection-config` | public | backend/routes/posts.js:1810 |  | native-gin |
| GET | `/api/posts/recommended` | optional-note-guest-restricted | backend/routes/posts.js:201 | optionalAuthWithNoteGuestRestriction | native-gin |
| GET | `/api/posts/video-center` | optional-video-guest-restricted | backend/routes/posts.js:416 | optionalAuthWithVideoGuestRestriction | native-gin |
| GET | `/api/protected-packages/:jobId` | user | backend/routes/posts.js:1864 | authenticateToken | native-gin |
| GET | `/api/protected-packages/:jobId/download` | user | backend/routes/posts.js:1885 | authenticateToken | native-gin |
| GET | `/api/protected-packages/:jobId/events` | user | backend/routes/posts.js:1885 | authenticateToken | native-gin |
| GET | `/api/pyvideo-api-proxy/*` | proxy-api-key | backend/routes/pyvideoProxy.js:59 | authenticateProxyAccess | native-gin |
| POST | `/api/reports` | user | backend/routes/reports.js:28 | authenticateToken | native-gin |
| GET | `/api/reports/check` | user | backend/routes/reports.js:99 | authenticateToken | native-gin |
| GET | `/api/search` | optional-note-guest-restricted | backend/routes/search.js:9 | optionalAuthWithNoteGuestRestriction | native-gin |
| GET | `/api/stats` | public | backend/routes/stats.js:7 |  | native-gin |
| GET | `/api/tags` | optional-note-guest-restricted | backend/routes/tags.js:9 | optionalAuthWithNoteGuestRestriction | native-gin |
| GET | `/api/tags/hot` | optional-note-guest-restricted | backend/routes/tags.js:29 | optionalAuthWithNoteGuestRestriction | native-gin |
| GET | `/api/ua-block/check` | public | backend/app.js:125 |  | native-gin |
| POST | `/api/upload/apk` | user | backend/routes/upload.js:899 | authenticateToken, apkUpload.single('file') | native-gin |
| POST | `/api/upload/attachment` | user | backend/routes/upload.js:819 | authenticateToken, attachmentUpload.single('file') | native-gin |
| POST | `/api/upload/chunk` | user | backend/routes/upload.js:443 | authenticateToken, chunkUpload.single('file') | native-gin |
| GET | `/api/upload/chunk/config` | user | backend/routes/upload.js:398 | authenticateToken | native-gin |
| POST | `/api/upload/chunk/merge` | user | backend/routes/upload.js:498 | authenticateToken | native-gin |
| POST | `/api/upload/chunk/merge/apk` | user | backend/routes/upload.js:689 | authenticateToken | native-gin |
| POST | `/api/upload/chunk/merge/image` | user | backend/routes/upload.js:599 | authenticateToken | native-gin |
| GET | `/api/upload/chunk/verify` | user | backend/routes/upload.js:412 | authenticateToken | native-gin |
| POST | `/api/upload/multiple` | user | backend/routes/upload.js:158 | authenticateToken, upload.array('files', 100) | native-gin |
| POST | `/api/upload/single` | user | backend/routes/upload.js:90 | authenticateToken, upload.single('file') | native-gin |
| POST | `/api/upload/video` | user | backend/routes/upload.js:248 | authenticateToken, videoUpload.fields([ { name: 'file', maxCount: 1 }, { name: 'thumbnail', maxCount: 1 } ]) | native-gin |
| GET | `/api/users` | user | backend/routes/users.js:790 | authenticateToken | native-gin |
| DELETE | `/api/users/:id` | user | backend/routes/users.js:1082 | authenticateToken | native-gin |
| GET | `/api/users/:id` | user | backend/routes/users.js:691 | authenticateToken | native-gin |
| PUT | `/api/users/:id` | user | backend/routes/users.js:828 | authenticateToken | native-gin |
| DELETE | `/api/users/:id/block` | user | backend/routes/users.js:2203 | authenticateToken | native-gin |
| POST | `/api/users/:id/block` | user | backend/routes/users.js:2137 | authenticateToken | native-gin |
| GET | `/api/users/:id/block-status` | user | backend/routes/users.js:2238 | authenticateToken | native-gin |
| GET | `/api/users/:id/collections` | user | backend/routes/users.js:1794 | authenticateToken | native-gin |
| DELETE | `/api/users/:id/follow` | user | backend/routes/users.js:1196 | authenticateToken | native-gin |
| POST | `/api/users/:id/follow` | user | backend/routes/users.js:1127 | authenticateToken | native-gin |
| GET | `/api/users/:id/follow-status` | user | backend/routes/users.js:1242 | authenticateToken | native-gin |
| GET | `/api/users/:id/followers` | user | backend/routes/users.js:1406 | authenticateToken | native-gin |
| GET | `/api/users/:id/following` | user | backend/routes/users.js:1305 | authenticateToken | native-gin |
| GET | `/api/users/:id/likes` | user | backend/routes/users.js:1931 | authenticateToken | native-gin |
| GET | `/api/users/:id/mutual-follows` | user | backend/routes/users.js:1507 | authenticateToken | native-gin |
| PUT | `/api/users/:id/password` | user | backend/routes/users.js:1026 | authenticateToken | native-gin |
| GET | `/api/users/:id/personality-tags` | user | backend/routes/users.js:637 | authenticateToken | native-gin |
| GET | `/api/users/:id/posts` | user | backend/routes/users.js:1621 | authenticateToken | native-gin |
| GET | `/api/users/:id/stats` | user | backend/routes/users.js:2046 | authenticateToken | native-gin |
| GET | `/api/users/api-keys` | user | backend/routes/users.js:2410 | authenticateToken | native-gin |
| POST | `/api/users/api-keys` | user | backend/routes/users.js:2460 | authenticateToken | native-gin |
| DELETE | `/api/users/api-keys/:id` | user | backend/routes/users.js:2529 | authenticateToken | native-gin |
| DELETE | `/api/users/history` | user | backend/routes/users.js:387 | authenticateToken | native-gin |
| GET | `/api/users/history` | user | backend/routes/users.js:229 | authenticateToken | native-gin |
| POST | `/api/users/history` | user | backend/routes/users.js:179 | authenticateToken | native-gin |
| DELETE | `/api/users/history/:postId` | user | backend/routes/users.js:347 | authenticateToken | native-gin |
| POST | `/api/users/me/avatar` | user | backend/routes/users.js:0 | authenticateToken, upload.single('image') | native-gin |
| POST | `/api/users/me/banner` | user | backend/routes/users.js:0 | authenticateToken, upload.single('image') | native-gin |
| POST | `/api/users/onboarding` | user | backend/routes/users.js:465 | authenticateToken | native-gin |
| GET | `/api/users/onboarding-config` | public | backend/routes/users.js:410 |  | native-gin |
| GET | `/api/users/onboarding-draft` | user | backend/routes/users.js:432 | authenticateToken | native-gin |
| PUT | `/api/users/onboarding-draft` | user | backend/routes/users.js:448 | authenticateToken | native-gin |
| GET | `/api/users/privacy-settings` | user | backend/routes/users.js:561 | authenticateToken | native-gin |
| PUT | `/api/users/privacy-settings` | user | backend/routes/users.js:593 | authenticateToken | native-gin |
| GET | `/api/users/search` | user | backend/routes/users.js:79 | authenticateToken | native-gin |
| GET | `/api/users/toolbar/items` | user | backend/routes/users.js:2094 | authenticateToken | native-gin |
| POST | `/api/users/verification` | user | backend/routes/users.js:2316 | authenticateToken | native-gin |
| DELETE | `/api/users/verification/revoke` | user | backend/routes/users.js:2361 | authenticateToken | native-gin |
| GET | `/api/users/verification/status` | user | backend/routes/users.js:2284 | authenticateToken | native-gin |
| GET | `/api/withdraw/admin/orders` | admin | backend/routes/withdraw.js:289 | adminAuth | native-gin |
| PUT | `/api/withdraw/admin/orders/:id/approve` | admin | backend/routes/withdraw.js:370 | adminAuth | native-gin |
| PUT | `/api/withdraw/admin/orders/:id/payout` | admin | backend/routes/withdraw.js:476 | adminAuth | native-gin |
| PUT | `/api/withdraw/admin/orders/:id/reject` | admin | backend/routes/withdraw.js:409 | adminAuth | native-gin |
| POST | `/api/withdraw/apply` | user | backend/routes/withdraw.js:112 | authenticateToken | native-gin |
| GET | `/api/withdraw/orders` | user | backend/routes/withdraw.js:232 | authenticateToken | native-gin |
| GET | `/api/withdraw/payment-code` | user | backend/routes/withdraw.js:51 | authenticateToken | native-gin |
| POST | `/api/withdraw/payment-code` | user | backend/routes/withdraw.js:75 | authenticateToken | native-gin |
| GET | `/api/withdraw/wallet` | user | backend/routes/withdraw.js:37 | authenticateToken | native-gin |

## WebSocket Upgrades

| Path | Auth | Source | Status | Notes |
| --- | --- | --- | --- | --- |
| `/api/im/ws` | query-token-and-redis-session | backend/app.js:96 | native-gin | HTTP upgrade is registered as GET /api/im/ws and handled by NativeHandlers.IMWebSocket |
