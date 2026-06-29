# 前后端 API 对接 TODO

## 环境与文档

- [x] 阅读 `frontend-backend-api-integration-plan.md`。
- [x] 扫描前后端目录、路由矩阵、现有 API client 和页面数据来源。
- [x] 输出进一步执行计划到文件。
- [x] 输出 TODO 到文件。
- [x] 新增 `front-end-nextjs/.env.example`，记录 `BACKEND_ORIGIN`、`NEXT_PUBLIC_BACKEND_ORIGIN`、可选 `NEXT_PUBLIC_API_BASE_URL`、`API_BASE_URL`、fixture 兜底、HTTP timeout/retry 和真实联调测试账号变量。
- [x] 新增 `frontend-backend-api-integration-env.md`，集中记录 readiness/Next.js 实际读取的 `.env` 层级、本地和远端后端地址配置、普通用户 A/B 与管理员 smoke 凭据、可写 smoke 开关、负向验证方式和当前阻塞项。
- [x] 新增 `FEED_FIXTURE_FALLBACK` 配置说明：开发环境默认允许 feed fixture 兜底，生产/真实联调默认禁用，必要演示时可显式开启。
- [x] 新增真实联调就绪检查脚本 `scripts/check-integration-readiness.mjs`，集中检查核心工具、DB/Redis 连通性、后端 health、前端后端 origin、前端配置后端 health 可达性、auth-config 契约、认证公共辅助接口契约、OAuth2 start 302 契约、OAuth2 backend callback 到前端 `/explore` 的回调契约、OAuth2 登录 UI/回调静态契约、OAuth2 start URL 静态契约、前端认证 session/refresh/logout 静态契约、发布/上传静态契约、内容详情/互动静态契约、个人/用户页静态契约、通知/IM 页面静态契约、创作者/钱包/后台页面静态契约、首页 feed 访问模式、搜索访问模式、游客访问登录态/后台只读端点权限边界、创作者/钱包公开配置契约、IM WebSocket 前后端静态契约、后端 route-matrix 关键路由合同、后端无数据库认证引导契约、前端 API 路由契约审计、`.env.example` 配置契约、Next rewrite/API client 后端地址契约、feed fixture 兜底门禁和测试账号环境变量；提供测试账号 token 或账号密码时，会条件执行 authenticated smoke，覆盖个人/用户页只读链路、双普通用户跨账号用户页只读链路、登录态首页内容、从 feed 派生的详情/评论/不喜欢状态/举报状态只读链路、搜索、通知未读数、通知列表、系统通知列表、系统弹窗通知、IM 会话列表、IM sync 和会话消息分页只读链路、创作者/钱包/提现只读端点、后台统计、AI 审核状态、游客访问状态、后台用户/帖子/内容审核/举报列表和提现审核订单列表。
- [x] 新增 readiness 自检脚本 `scripts/check-integration-readiness-selftest.mjs`，自动覆盖默认不跑可写 smoke、fixture fallback 门禁、写 smoke 无凭据、假用户 token 和假管理员 token 等负向场景，并验证报告不输出传入 token。
- [x] 新增登录配置契约门禁：`frontend-auth-config-contract` 会校验前端配置后端的 `/api/auth/auth-config` 响应字段、OAuth2 start URL 可解析性，以及 Geetest 开启时的 `geetestCaptchaId`。
- [x] 新增认证公共辅助接口契约门禁：`frontend-auth-public-aux-contract` 会校验 `/api/auth/email-config` 返回 boolean `emailEnabled` 且与 `auth-config` 一致，`/api/auth/captcha` 返回非空 `captchaId` 和无 `<script>` 的 SVG `captchaSvg`，覆盖注册页邮箱开关和 SVG 验证码兜底链路。
- [x] 新增 OAuth2 start 跳转门禁：`frontend-oauth2-start-redirect` 会在不跟随跳转的情况下校验 `/api/auth/oauth2/login` 返回 302 到 `/oauth2.1/authorize`，并携带 `state`、PKCE S256、`dpop_jkt`、`redirect_uri` 等关键参数。
- [x] 新增 OAuth2 backend callback 回前端门禁：`frontend-oauth2-callback-contract` 会校验真实 OAuth2 start 302 中的 `redirect_uri` 指向当前后端 `/api/auth/oauth2/callback`，后端成功换取本地 token 后回跳 `/explore?oauth2_login=success&access_token=...&refresh_token=...`，前端 `/explore` 页面存在，OAuth success callback 会先渲染以便客户端保存 token，且根布局早期脚本会保存 token、同步 SSR auth cookie 并清理 URL 敏感参数。
- [x] 新增 OAuth2 登录 UI/回调静态契约门禁：`frontend-oauth2-ui-contract` 会校验 `/login` 渲染 `LoginForm`、用户中心按钮指向后端 OAuth2 start endpoint、OAuth2-only 状态隐藏密码表单并显示提示、`NEXT_PUBLIC_API_BASE_URL` 直连拼接、回调早期脚本位于 `<head>` 且会保存 token 并清理 URL 敏感参数。
- [x] 新增 OAuth2 start URL 静态契约门禁：`frontend-oauth2-start-url-contract` 会校验“用户中心一键登录”按钮只解析 `oauth2StartUrl` 或默认 `/api/auth/oauth2/login` 后端启动端点，禁止误用授权中心根地址 `oauth2LoginUrl`，并锁定 `NEXT_PUBLIC_API_BASE_URL` 直连后端拼接逻辑。
- [x] 新增前端认证 session 静态契约门禁：`frontend-auth-session-contract` 会校验普通用户/管理员 localStorage key 隔离、登录/注册/OAuth 回调 token 保存、普通用户 SSR auth cookie 写入/清理、服务端 API 请求从 cookie 还原 token、401 refresh 后单次重试、不可恢复 401 清理并跳登录、logout 始终清理本地普通用户 session。
- [x] 新增发布/上传静态契约门禁：`frontend-publish-upload-contract` 会校验前端图片、视频、附件上传分别走 `/api/upload/single`、`/api/upload/video`、`/api/upload/attachment`，发布走 `/api/posts`，发布工作台的 image/video/podcast 进度上传分支、podcast `audio/*` 限制和 `attachment` 字段映射，以及后端 attachment MIME 放行 `audio/*` 且测试覆盖拒绝 `video/mp4`。
- [x] 新增内容详情/互动静态契约门禁：`frontend-content-interaction-contract` 会校验前端详情、评论、点赞、收藏、关注/取关、不感兴趣和举报 API 封装与后端 route-matrix 的 method/auth/status 对齐，并确认首页 feed 与详情抽屉真实调用这些封装、游客可读详情/评论使用 `auth: false`、登录态状态查询缺 token 时不强制跳转。
- [x] 新增个人/用户页静态契约门禁：`frontend-profile-user-contract` 会校验 `/profile`、`/user/[id]`、当前用户资料、用户资料、本人资料编辑、笔记/收藏/赞过 tab、关注状态、关注/取关、钱包/通知/私信入口和用户链接 helper 均通过前端 helper 调用，并与后端 route-matrix 的 method/auth/status 对齐。
- [x] 新增通知/IM 页面静态契约门禁：`frontend-notifications-im-contract` 会校验通知列表、未读数、根布局全局系统弹窗通知组件、系统弹窗通知 helper、标记已读、全部已读、删除、系统通知确认/移除，以及 IM 会话创建、会话列表、消息分页、发送消息、已读推进、WebSocket 新消息刷新和用户主页私信跳转均与后端 route-matrix 的 method/auth/status 对齐。
- [x] 新增创作者/钱包/后台页面静态契约门禁：`frontend-creator-wallet-admin-contract` 会校验创作者概览/统计/趋势/收益/付费内容/质量奖励、钱包余额/充值/收款码/提现/订单、后台登录/me/统计/开关/列表/提现审核动作均通过前端 helper 调用，并与后端 route-matrix 的 method/auth/status 对齐。
- [x] 新增后端 route-matrix 合同门禁：`backend-route-matrix-contract` 会校验 `backend-gin/internal/http/routes/route-matrix.json` 仍记录 394 条 HTTP API、26 个 mounted module、1 个 WebSocket，所有 route/WebSocket 都是 `native-gin`，并锁定认证、OAuth2、feed/评论/互动、上传、通知、IM、创作者、钱包提现和后台统计/提现审核等 40 条关键集成路由的 method/auth/status。
- [x] 新增后端无数据库认证引导契约门禁：`backend-auth-bootstrap-no-db-contract` 会静态校验 `databaseAvailability` 仅放行 `/api/health`、`/_gin_migration/status`、`/api/auth/auth-config`、`/api/auth/email-config`、`/api/auth/captcha`、`/api/auth/oauth2/login` 等无 DB 启动路径，不误放行登录/注册/内容业务路径，并确认 `AuthMatrix` dispatch 与 `TestDatabaseOptionalAuthBootstrapRoutes` 测试覆盖仍在。
- [x] 新增游客权限边界门禁：`frontend-guest-protected-access` 会以无 token 状态检查 `/api/auth/me`、通知、系统弹窗通知、IM 会话、创作者概览、用户余额、提现钱包、管理员 me/统计/用户列表/提现审核订单等 13 个只读登录态/后台端点，要求均返回 401/403，避免受保护数据误开放或中间件漂移。
- [x] 新增前端 API 路由契约审计门禁：`frontend-api-route-audit` 会执行 `front-end-nextjs/scripts/audit-api-routes.mjs`，要求审计状态为 pass、`unmatchedApiCalls=0` 且 `broadDynamicMatches=0`；当前 81 个前端 API 调用形态均匹配后端 route-matrix。
- [x] 新增 `.env.example` 配置契约门禁：`frontend-env-example-contract` 会校验后端 origin、浏览器直连后端、服务端后端地址、fixture、HTTP timeout/retry 和测试账号变量均有示例，URL 示例可解析，且不会默认启用 fixture fallback 或写入真实 token/password。
- [x] 新增环境速查文档契约门禁：`frontend-env-doc-contract` 会校验 `frontend-backend-api-integration-env.md` 覆盖 Next/readiness 的 `.env` 读取层级、后端地址、用户 A/B/管理员凭据、可写 smoke、假 token 负向验证和禁止写入真实 token 的说明，并确认 `.env.example` 链接到该文档。
- [x] 新增 Next rewrite/API client 后端地址契约门禁：`frontend-backend-address-contract` 会校验 `next.config.ts` 使用 `BACKEND_ORIGIN` / `NEXT_PUBLIC_BACKEND_ORIGIN` 代理 `/api/:path*`，以及 `src/lib/api.ts` 支持浏览器端 `NEXT_PUBLIC_API_BASE_URL` 和服务端 `API_BASE_URL` / `BACKEND_ORIGIN` / `NEXT_PUBLIC_BACKEND_ORIGIN`。
- [x] 新增首页 feed 访问模式门禁：`frontend-initial-feed-access` 会检查推荐 feed 和热门分类首屏端点；当前远端一致返回 401 “请登录后查看内容”，记录为 `guest-restricted`，后续内容链路需要普通用户登录态继续联调。
- [x] 新增搜索访问模式门禁：`frontend-search-access` 会检查 `/api/search?keyword=smoke&type=all&page=1&limit=1`；当前远端返回 401，记录为 `guest-restricted`，真实搜索结果和分页仍需普通用户登录态继续联调。
- [x] 新增创作者/钱包公开配置契约门禁：`frontend-monetization-config-contract` 会检查 `/api/creator-center/config`、`/api/balance/config` 和 `/api/balance/recharge-config`；当前远端三个端点均返回 200，且创作者费率/提现字段、余额开关、充值 URL 与金额选项符合前端契约。
- [x] 扩展钱包充值配置对接：`/wallet` 页面已读取并展示 `/api/balance/recharge-config` 返回的固定充值档位、自定义金额范围和礼品卡档位；`frontend-monetization-config-contract` 现在同时校验 `custom_amount_enable`、`min_amount`、`max_amount`、`gift_card_purchase.options` 等远端字段形状，`frontend-creator-wallet-admin-contract` 静态锁定钱包页实际使用这些字段。
- [x] 新增 IM WebSocket 前后端静态契约门禁：`frontend-im-websocket-contract` 会校验后端 route-matrix 的 `/api/im/ws` WebSocket upgrade、`query-token-and-redis-session` 鉴权模式，以及前端 `getImWebSocketUrl()` 的 `/api/im/ws`、`ws/wss` 协议转换和 `token` query 拼接。
- [x] 新增真实账号 authenticated smoke 条件分支：支持 `INTEGRATION_USER_A_ACCESS_TOKEN` 或 `INTEGRATION_USER_A_ID/PASSWORD`、`INTEGRATION_USER_B_ACCESS_TOKEN` 或 `INTEGRATION_USER_B_ID/PASSWORD`、`INTEGRATION_ADMIN_ACCESS_TOKEN` 或 `INTEGRATION_ADMIN_USERNAME/PASSWORD`；变量可来自进程环境或已加载的 `.env` 文件，脚本不输出 token/password。
- [x] 扩展普通用户 authenticated smoke：拿到有效普通用户 token 后会从 `/api/auth/me` 推导用户 ID，并验证 `/api/users/:id`、`/api/users/:id/posts`、`/api/users/:id/collections`、`/api/users/:id/likes` 和 `/api/users/:id/follow-status` 的状态与基础 payload 结构。
- [x] 扩展用户 A authenticated smoke：拿到有效普通用户 A token 后会额外验证搜索、通知未读数、通知列表、系统通知列表、系统弹窗通知、IM 会话列表、`/api/im/sync`、首个会话的 `/api/im/conversations/:id/messages` 分页、创作者中心概览/统计/趋势/收益/付费内容/质量奖励，以及余额、本地月币、提现钱包、收款码和订单只读端点的状态与基础 payload 结构；推荐 feed 返回帖子 ID 时，还会追加验证 `/api/posts/:id`、`/api/posts/:id/comments`、`/api/dislikes?post_id=...` 和 `/api/reports/check?target_type=post&target_id=...` 的只读状态与基础 payload 结构。
- [x] 新增双普通用户跨账号 authenticated smoke：用户 A/B 凭据同时存在时，会分别用 A 的 token 读取 B 的 `/api/users/:id`、笔记、收藏、赞过和关注状态，再用 B 的 token 读取 A 的同一组只读用户页接口；该 smoke 只执行 GET，不触发关注、私信或其他写动作。
- [x] 新增普通用户草稿发布写入 smoke：设置 `INTEGRATION_ENABLE_WRITE_SMOKE=true` 且提供用户 A 凭据时，脚本会创建一条最小 `is_draft: true` 草稿帖并随后删除；草稿不会触发粉丝新帖通知，但仍会写入帖子相关表，默认关闭。
- [x] 新增普通用户帖子互动写入 smoke：设置 `INTEGRATION_ENABLE_WRITE_SMOKE=true`、提供用户 A 凭据和 `INTEGRATION_WRITE_SMOKE_POST_ID` 时，脚本会读取指定测试帖子的点赞/收藏状态，临时 toggle 点赞和收藏、创建一条唯一烟测评论，再尽量删除评论并恢复点赞/收藏初始状态；缺测试帖子 ID 时显式失败，避免误写真实内容。
- [x] 新增普通用户提现收款码写入 smoke：设置 `INTEGRATION_ENABLE_WRITE_SMOKE=true` 且提供用户 A 凭据时，脚本会先读取 `/api/withdraw/payment-code`；仅当账号已有至少一个收款码 URL 时，才临时保存烟测 URL 并恢复初始值，避免在空账号上创建不可通过现有 API 恢复的首条收款码记录。
- [x] 新增双普通用户 IM 写入 smoke：设置 `INTEGRATION_ENABLE_WRITE_SMOKE=true` 且提供用户 A/B 凭据时，脚本会由 A 创建/复用与 B 的 direct 会话，发送一条带唯一 `client_msg_id` 的烟测消息，再用 B 读取会话消息并标记已读；默认关闭，避免常规 readiness 写入真实数据。
- [x] 扩展管理员 authenticated smoke：拿到有效管理员 token 后会额外验证后台用户、帖子、内容审核、举报列表和提现审核订单列表的状态与基础列表结构；默认 authenticated smoke 仅执行 GET，不触发开关写入或审核动作。
- [x] 新增管理员运行时开关写入 smoke：设置 `INTEGRATION_ENABLE_WRITE_SMOKE=true` 且提供管理员凭据时，脚本会读取 AI 审核和游客访问开关，临时翻转、确认变更，再写回初始值并确认恢复；默认不运行写动作，开启写 smoke 但缺管理员凭据时会显式失败。
- [x] 新增 readiness HTTP 抗抖动配置：只读 GET/manual redirect 默认重试 1 次，可通过 `INTEGRATION_HTTP_RETRY_COUNT` 调整，POST 登录不重试以避免重复提交。
- [ ] 补齐缺失工具：`fd`、`bat`、`delta`、`fzf`、`uv`。当前阻碍：`winget` 不在 PATH。
- [x] 验证本机后端状态：常规 `.env` 启动会因数据库主机不可解析失败，`localhost:3001` 默认不监听。
- [x] 验证后端无数据库模式：进程级空白覆盖数据库变量后，`/api/health` 和 `/_gin_migration/status` 可返回 200，394 条 HTTP 路由和 1 个 WebSocket 入口注册正常。
- [x] 放行无数据库模式下的认证引导接口：`/api/auth/auth-config`、`/api/auth/email-config`、`/api/auth/captcha`、`/api/auth/oauth2/login` handler 本身不依赖数据库，现已从 `databaseAvailability` 中间件放行，可用于本地验证 OAuth2-only/Geetest/captcha 登录页分支和 OAuth2 授权启动跳转。
- [ ] 准备真实联调环境：可访问且已有 schema/data 的数据库、Redis 会话服务、普通用户 A/B、管理员测试账号。当前阻碍：本机无 `docker`、`psql`、`mysql`，仓库未发现可直接建库的 compose/schema SQL/seed 脚本；`scripts/check-integration-readiness.mjs` 当前显示 feed fixture 兜底门禁、前端配置后端 health、auth-config 契约、认证公共辅助接口契约、OAuth2 start 302 契约、OAuth2 backend callback 回前端契约、OAuth2 登录 UI/回调静态契约、OAuth2 start URL 静态契约、前端认证 session 静态契约、发布/上传静态契约、内容详情/互动静态契约、个人/用户页静态契约、通知/IM 页面静态契约、创作者/钱包/后台页面静态契约、首页 feed 访问模式、搜索访问模式、游客权限边界、创作者/钱包公开配置契约、IM WebSocket 静态契约、后端 route-matrix 关键路由合同、后端无数据库认证引导契约、前端 API 路由契约审计、`.env.example` 配置契约和 Next rewrite/API client 后端地址契约已通过，但 DB/Redis 主机不可解析、本地后端 health 不可达、普通用户 A/B 和管理员测试账号环境变量缺失；远端内容访问当前需要登录态，提供测试账号或 access token 后脚本会追加 authenticated smoke。
- [ ] 若进入 Android/Capacitor 构建，切换到 JDK 21 并验证 Android SDK/Gradle。

## 统一请求层

- [x] 统一 base URL、query、body、FormData 处理。
- [x] Blob / 下载响应处理。
- [x] 支持 `Authorization: Bearer <token>`。
- [x] 支持 401 refresh 与登录跳转。
- [x] 支持业务错误对象和错误消息透传。
- [x] 支持上传接口。
- [x] 加固成功但无 `data` 的响应拆包：关注/取关、已读、后台开关等只返回 `code/message` 时统一返回空对象，避免 envelope 被误当作业务 payload。

## 认证与登录

- [x] 登录页改为真实表单。
- [x] 接入 `/api/auth/login`。
- [x] 接入 `/api/auth/register`。
- [x] 接入 `/api/auth/me`。
- [x] 接入 `/api/auth/refresh`。
- [x] 接入 `/api/auth/logout`。
- [x] 接入 `/api/auth/auth-config`，按后端配置显示 OAuth2 登录入口、OAuth2-only 提示、邮箱验证码注册字段和 Geetest 必需提示。
- [x] 接入 `/api/auth/send-email-code`，兼容后端成功但无 `data` 的响应。
- [x] 登录页新增“用户中心一键登录”按钮：默认使用同源启动入口 `/api/auth/oauth2/login`，兼容 `auth-config` 返回的 `oauth2StartUrl`；配置 `NEXT_PUBLIC_API_BASE_URL` 时会拼接直连后端地址。
- [x] 按 `demo_oauth21` 复核登录入口职责：前端按钮只跳后端启动端点，由后端生成 state、PKCE、DPoP 并 302 到授权中心；`NEXT_PUBLIC_API_BASE_URL` 直连拼接已兼容末尾斜杠或误带 `/api` 路径。
- [x] OAuth2 回调处理：新增 `/explore` 回调目标，首屏早期脚本清理 URL 中的 `access_token`/`refresh_token` 等敏感参数并保存 token，客户端 handler 后续补拉 `/api/auth/me` 写入用户快照。
- [x] OAuth2 回调后仍停留登录页修复：`/explore` 和 `/` 在带 `oauth2_login=success` 且含 token 参数时会先渲染回调壳，避免 SSR feed 401 在 token bootstrap 前重定向 `/login`；回调 token 现在同时写入 `localStorage` 和 `yuem_access_token` cookie，SSR API helper 可从 cookie 还原 `Authorization`，客户端补拉 `/api/auth/me` 后跳回首页。
- [x] OAuth2 回调早期脚本挂载位置修正：`OAuthCallbackBootstrap` 已移入根布局 `<head>`，避免 `<script>` 作为 `<html>` 直接子节点导致 React/Next hydration 结构错误。
- [x] Geetest widget 已接入：注册 tab 会按 `auth-config.geetest.captchaId` 加载 Geetest v4 widget，提交注册时携带 `lot_number`、`captcha_output`、`pass_token`、`gen_time`；真实验证仍需有效 Geetest key、数据库、Redis 和测试账号联调。
- [x] 本地无数据库模式验证用户中心登录分支：临时后端返回 OAuth2-only + Geetest 配置时，`/login` 经 Next rewrite 显示“用户中心一键登录”按钮和“当前环境仅支持用户中心登录。”提示，且控制台无 error。
- [x] 浏览器级验证用户中心登录入口：在现有 `localhost:3000` Next dev server 上通过 CDP 拦截 `/api/auth/auth-config` 返回 OAuth2-only 配置，页面真实渲染 `<a href="/api/auth/oauth2/login">用户中心一键登录</a>`，显示“当前环境仅支持用户中心登录。”，密码表单隐藏，且无 script/hydration 结构类 console error。
- [x] 本地无数据库模式验证 OAuth2 start：`GET /api/auth/oauth2/login` 返回 302 到外部授权地址，并携带 `client_id`、`state`、PKCE、DPoP、`redirect_uri` 等参数。

## 首页 Feed 与详情

- [x] 首页首屏接入推荐 feed 和热门分类。
- [x] 首页/探索页 fixture feed 兜底改为开发/显式开关行为：生产和真实联调默认暴露后端错误，不再静默显示假数据。
- [x] Feed tab 请求加入错误状态和空状态。
- [x] Feed 分页/加载更多。
- [x] 首页/探索页搜索接入 `/api/search`：桌面搜索框和移动搜索按钮会按后端 `keyword/tag/type/page/limit` 契约请求搜索结果，并复用现有瀑布流、详情、点赞、收藏和加载更多逻辑；真实结果仍需可用数据库后端联调。
- [x] 点赞接入 `/api/likes`。
- [x] 收藏接入 `/api/posts/:id/collect`。
- [x] 详情接入 `/api/posts/:id`。
- [x] 评论列表接入 `/api/posts/:id/comments`。
- [x] 发评论接入 `/api/comments`。
- [x] 移除详情抽屉示例评论。
- [x] 详情抽屉作者关注入口接入：按帖子作者 ID 可选读取 `/api/users/:id/follow-status`，桌面/移动端头部关注按钮接入 `/api/users/:id/follow` 关注/取关；未登录时不阻塞详情打开，真实 mutation 仍待账号联调。
- [x] 详情抽屉更多操作接入：新增不喜欢状态/切换 `/api/dislikes`、举报状态 `/api/reports/check`、举报提交 `/api/reports` 和系统分享/复制链接回退；真实 mutation 和后台举报联动仍待登录账号与后端联调。

## 用户页与个人页

- [x] `/profile` 接入当前用户。
- [x] `/profile` 接入本人资料编辑：编辑资料弹窗提交 `nickname`、`bio`、`location`、`avatar`、`background` 到 `PUT /api/users/:id`，成功后用后端返回 payload 更新本地 profile 展示。
- [x] `/profile` 本人资料编辑浏览器烟测：本地 mock 登录态下可打开编辑资料弹窗，填写昵称、简介和地点后提交，前端发出 `PUT /api/users/:id` 且页面按返回 payload 更新。
- [x] `/user/[id]` 接入 `/api/users/:id`。
- [x] 用户笔记接入 `/api/users/:id/posts`。
- [x] 用户收藏接入 `/api/users/:id/collections`。
- [x] 用户赞过接入 `/api/users/:id/likes`。
- [x] 关注状态接入 `/api/users/:id/follow-status`。
- [x] 关注/取消关注接入 `/api/users/:id/follow`。
- [x] `/user/[id]` 浏览器烟测：本地 mock 登录态下可渲染目标用户资料，关注按钮调用 `POST /api/users/:id/follow` 并更新粉丝数，已关注按钮调用 `DELETE /api/users/:id/follow` 并恢复状态，私信按钮调用 `/api/im/conversations` 后跳转 `/messages?conversation=...`。
- [x] 清理旧用户 fixture 资料入口：`src/lib/users.ts` 已瘦身为作者 ID/链接工具，不再提供本地假用户资料、收藏和点赞 tab 数据。
- [x] 静态复核用户页接口契约：用户详情、用户 tab 和关注相关接口均按后端 handler 对齐；注意这些用户接口按后端设计要求登录，真实联调时需确认未登录访问用户页的产品预期。

## 发布与上传

- [x] 发布页接入 `/api/posts`。
- [x] 草稿保存使用 `is_draft: true`。
- [x] 图片上传接入 `/api/upload/single`。
- [x] 视频上传接入 `/api/upload/video`。
- [x] 上传前校验格式和大小。
- [x] 显示上传中和失败状态。
- [x] 上传进度百分比。
- [x] 上传失败重试和移除状态。
- [x] 播客音频上传接入 `/api/upload/attachment`：后端附件 MIME 已放行 `audio/*`，前端 podcast 模式只接受音频文件并复用上传进度、失败重试和移除状态。
- [x] 播客音频发布复用 `/api/posts` 的 `attachment` 字段：提交 `{ url, filename, filesize }`，不新增后端 schema 或专用 podcast handler。
- [x] 详情抽屉已渲染 `post.attachment`：显示附件文件名、大小和外链入口；识别常见音频扩展名时内嵌原生 audio 播放器。

## 后续模块

- [x] 通知：个人页通知按钮已接未读数 `/api/notifications/unread-count`。
- [x] 通知：新增 `/notifications` 页面，接入完整列表、标记已读、全部已读、删除通知、系统通知确认/移除。
- [x] 通知：新增 `getPopupSystemNotifications()` helper，对接 `/api/notifications/system/popup`，并纳入游客权限边界和用户 A authenticated smoke 只读校验。
- [x] 通知：新增全局 `SystemNotificationPopup`，挂载到根布局；有本地 token 时只读拉取系统弹窗通知，无 token/401 静默跳过，弹窗可确认、移除或打开通知链接。
- [x] 通知：`/notifications` 页面浏览器烟测已覆盖本地 mock 登录态下的通知列表、系统通知列表、未读数、全部已读、单条删除、系统确认和系统移除；mock 收到 `PUT /api/notifications/read-all`、`DELETE /api/notifications/:id`、`POST /api/notifications/system/:id/confirm`、`DELETE /api/notifications/system/:id/dismiss`，页面状态同步更新且控制台无 error。
- [x] IM：用户页私信按钮已接会话创建 `/api/im/conversations`。
- [x] IM：新增 `/messages` 页面，接入会话列表、消息列表、发送消息、已读推进、WebSocket 新消息刷新与断线重连。
- [x] IM：新增会话搜索和历史消息向上翻页，使用 `/api/im/conversations/:id/messages?before&limit`。
- [x] IM：按后端真实结构对齐消息历史分页，读取顶层 `pagination.has_more` 和 `pagination.next_before`，避免只按数组长度猜测下一页。
- [ ] IM：登录态只读 sync 与会话消息分页已纳入 authenticated smoke；双账号创建/发送/读取/已读已纳入 opt-in 写入 smoke；真实 WebSocket 推送和页面级双账号收发仍需可用数据库、Redis 和用户 A/B 测试账号联调。
- [x] 创作者中心：发布工作台首页已接概览和统计 `/api/creator-center/overview`、`/api/creator-center/stats`。
- [x] 创作者中心：发布工作台首页新增创作经营区块，接入趋势图、收益流水、付费内容、质量奖励 `/api/creator-center/trends`、`/api/creator-center/earnings-log`、`/api/creator-center/paid-content`、`/api/creator-center/quality-rewards`。
- [x] 创作者中心：收益流水、付费内容、质量奖励已增加加载更多分页入口。
- [ ] 创作者中心：真实账号数据联调仍需可用数据库、Redis 和创作者测试账号验证。
- [x] 钱包提现：新增 `/wallet` 页面和个人页入口，接入余额配置、充值配置、本地月币、外部月币、提现钱包、收款码、提现申请、提现订单、购买订单、创作者收益转月币。
- [x] 钱包充值：`/wallet` 已展示后端充值配置中的月币充值档位、礼品卡档位和自定义充值范围；有 `recharge_url` 时所有充值入口指向外部用户中心充值页。
- [x] 钱包提现：`/wallet` 页面浏览器烟测已覆盖本地 mock 登录态下的钱包数据渲染、充值中心外链、收款码保存、现金提现申请、提现订单刷新和创作者收益转入月币；mock 收到 `POST /api/withdraw/payment-code`、`POST /api/withdraw/apply`、`POST /api/creator-center/withdraw`，页面余额与订单随刷新更新且控制台无 error。
- [ ] 钱包提现：真实账号数据、收款码保存、提现申请、充值跳转和订单刷新仍需可用数据库、Redis、钱包/提现相关测试数据联调。
- [x] 后台管理：新增 `/admin` 页面，接入管理员登录、当前管理员、统计总览、AI 审核开关、访客访问开关、用户/内容/审核/举报分页列表和提现审核订单/动作。
- [ ] 后台管理：真实管理员账号登录、列表数据、开关写入和提现审核动作仍需可用数据库、Redis 和管理员测试账号联调。

## 真实联调执行顺序

- [ ] 运行 `node scripts\check-integration-readiness.mjs` 并确认状态为 `ready`。
- [ ] 启动真实后端并确认 `GET /api/health`、`GET /_gin_migration/status` 返回 200，且 `GET /api/posts?page=1&limit=1` 不再返回“数据库未配置”。
- [ ] 认证链路：auth-config、注册/登录、`/api/auth/me`、refresh、logout、OAuth2 真实账号 start/callback。
- [ ] 公开内容链路：首页 feed、分类、详情、评论列表、分页、图片/视频地址。
- [ ] 用户互动链路：点赞、收藏、发评论、用户页、个人页、关注/取关。
- [ ] 发布上传链路：图片上传、视频上传、播客音频附件上传、草稿保存、发布成功后详情可访问。
- [ ] 通知与 IM 链路：通知列表/未读/标记已读，用户 A/B 会话、发送/接收、WebSocket 推送、已读和历史消息翻页。
- [ ] 创作者与钱包链路：创作者概览/趋势/收益，余额、收款码、提现申请、充值跳转、订单刷新。
- [ ] 后台管理链路：管理员登录、统计、开关、列表分页、提现审核 approve/reject/payout。

## 验证

- [x] `npm.cmd run lint`。
- [x] `npm.cmd run build`。
- [x] 真实联调就绪检查脚本：`node scripts\check-integration-readiness.mjs` 当前返回 `not-ready`，40 项基础检查中 31 pass、3 warn、6 fail；`frontend-feed-fixture-fallback`、`frontend-backend-health`、`frontend-auth-config-contract`、`frontend-auth-public-aux-contract`、`frontend-oauth2-start-redirect`、`frontend-oauth2-callback-contract`、`frontend-oauth2-ui-contract`、`frontend-oauth2-start-url-contract`、`frontend-auth-session-contract`、`frontend-publish-upload-contract`、`frontend-content-interaction-contract`、`frontend-profile-user-contract`、`frontend-notifications-im-contract`、`frontend-creator-wallet-admin-contract`、`frontend-initial-feed-access`、`frontend-search-access`、`frontend-guest-protected-access`、`frontend-monetization-config-contract`、`frontend-im-websocket-contract`、`backend-route-matrix-contract`、`backend-auth-bootstrap-no-db-contract`、`frontend-api-route-audit`、`frontend-env-example-contract`、`frontend-env-doc-contract` 和 `frontend-backend-address-contract` 通过，失败项为 DB/Redis 主机不可解析、本地后端 health 不可达、普通用户 A/B 和管理员测试账号变量缺失；远端首屏内容和搜索端点当前为游客受限模式，登录态/后台只读端点均拒绝游客访问。
- [x] 真实联调就绪检查脚本系统弹窗通知覆盖：`frontend-guest-protected-access` 已确认 `/api/notifications/system/popup` 游客 401；`backend-route-matrix-contract` 关键路由数为 40；`frontend-api-route-audit` 当前 81 个前端 API 调用形态、0 个未匹配、0 个宽动态匹配。
- [x] 真实联调就绪检查脚本钱包充值覆盖：`frontend-monetization-config-contract` 已验证远端充值配置中的自定义金额、充值档位和礼品卡配置字段；`frontend-creator-wallet-admin-contract` 已确认钱包页渲染充值档位、礼品卡档位和金额范围。
- [x] 真实联调就绪检查脚本草稿发布写入覆盖：`INTEGRATION_ENABLE_WRITE_SMOKE=true` 且用户 A 凭据可用时会追加 `user-draft-post-write-smoke`，验证 `/api/posts` 创建草稿和 `/api/posts/:id` 删除清理；默认不运行写动作。
- [x] 真实联调就绪检查脚本帖子互动写入覆盖：`INTEGRATION_ENABLE_WRITE_SMOKE=true` 且用户 A 凭据和 `INTEGRATION_WRITE_SMOKE_POST_ID` 可用时会追加 `user-post-interaction-write-smoke`，验证指定测试帖子的点赞 toggle、收藏 toggle、评论创建和评论删除恢复；默认不运行写动作，缺测试帖子 ID 时显式失败。
- [x] 真实联调就绪检查脚本提现收款码写入覆盖：`INTEGRATION_ENABLE_WRITE_SMOKE=true` 且用户 A 凭据可用时会追加 `user-withdraw-payment-code-write-smoke`，验证 `/api/withdraw/payment-code` 的读取、临时保存和恢复；账号没有初始收款码时会失败在写入前。
- [x] 真实联调就绪检查脚本提现收款码写入负向验证：默认 readiness 不触发该可写 smoke；临时开启 `INTEGRATION_ENABLE_WRITE_SMOKE=true` 但无用户 A 凭据时会在缺凭据处失败；提供假用户 A token 时会先请求 `/api/auth/me` 并因 401 停止在 mutation 前，报告不输出 token。
- [x] 真实联调就绪检查脚本自检：`node scripts\check-integration-readiness-selftest.mjs` 通过 5 个负向场景，确认 baseline 不触发写 smoke、fixture fallback 会失败、6 个写 smoke 在缺凭据时失败、假用户 token 停在用户鉴权前置、假管理员 token 停在管理员鉴权前置，且报告不泄露传入 token。
- [x] 真实联调就绪检查脚本 IM 写入覆盖：`INTEGRATION_ENABLE_WRITE_SMOKE=true` 时会追加 `user-cross-account-im-write-smoke`，验证双普通用户 direct 会话创建/复用、发送唯一消息、另一用户读取消息和标记已读；默认不运行写动作。
- [x] 真实联调就绪检查脚本管理员运行时开关写入覆盖：`INTEGRATION_ENABLE_WRITE_SMOKE=true` 且管理员凭据可用时会追加 `admin-runtime-toggle-write-smoke`，验证 AI 审核和游客访问 runtime toggle 的读写确认与恢复；默认不运行写动作，开启写 smoke 但缺管理员凭据时会显式失败。
- [x] 真实联调就绪检查脚本 authenticated smoke 负向验证：临时设置 `INTEGRATION_USER_A_ACCESS_TOKEN=codex-invalid-token` 时，脚本追加 `user-a-authenticated-smoke` 并按 `/api/auth/me`、推荐 feed、热门分类、搜索、通知未读数、通知列表、系统通知列表、系统弹窗通知、IM 会话列表、创作者/钱包/提现只读端点 401 失败，且报告不输出 token。
- [x] 浏览器烟测：清空 `yuem_access_token`、`yuem_refresh_token`、`yuem_user` 后访问 `http://localhost:3000/login`，全局系统通知弹窗不显示，登录表单正常渲染且控制台无 error。
- [x] 真实联调就绪检查脚本跨账号 authenticated smoke 负向验证：临时同时设置 `INTEGRATION_USER_A_ACCESS_TOKEN` 和 `INTEGRATION_USER_B_ACCESS_TOKEN` 为无效 token 时，脚本追加 `user-cross-account-read-smoke`，按双方 `/api/auth/me` 401 失败并停止在只读用户页派生前，且报告不输出 token。
- [x] 真实联调就绪检查脚本管理员 authenticated smoke 负向验证：临时设置 `INTEGRATION_ADMIN_ACCESS_TOKEN=codex-invalid-admin-token` 时，脚本追加 `admin-authenticated-smoke` 并按 `/api/auth/admin/me`、后台统计、AI 审核状态、游客访问状态、后台只读列表和提现审核订单列表 401 失败，且报告不输出 token。
- [x] 真实联调就绪检查脚本门禁验证：临时设置 `FEED_FIXTURE_FALLBACK=true` 时，`frontend-feed-fixture-fallback` 会失败并提示该配置会掩盖后端错误。
- [x] 真实联调就绪检查脚本门禁验证：临时设置 `BACKEND_ORIGIN=http://127.0.0.1:39999` 时，`frontend-backend-health` 会失败并提示前端配置后端 health 不可达。
- [x] 前端 API 路由契约审计脚本：`npm.cmd run audit:api` 用 AST 收集前端 `/api/...` 调用并匹配后端 `route-matrix.json`；脚本会读取 `AdminListResource` 白名单展开 `/api/admin/${resource}`，当前 82 个前端 API 调用形态、0 个未匹配、0 个宽动态匹配，新增 `PUT /api/users/:id` 已覆盖。
- [x] 浏览器烟测：`/`、`/publish`、`/login` 可渲染且无控制台 error；`/profile` 在未登录/后端不可用时显示资料加载失败。
- [x] 浏览器烟测：最新 `/login` 认证配置改动后，后端不可用时登录表单与注册 tab 均可渲染，控制台无 error；真实 OAuth2-only/Geetest 分支仍需可访问后端 auth-config。
- [x] 浏览器烟测：OAuth2-only auth-config 下，`/login` 显示 `<a href="/api/auth/oauth2/login">用户中心一键登录</a>` 和仅支持用户中心登录提示，隐藏密码表单，且回调 bootstrap script 移入 `<head>` 后无结构类 hydration console error。
- [x] 浏览器烟测：OAuth2 回调 URL `/explore?oauth2_login=success&access_token=smoke-access&refresh_token=smoke-refresh&keep=1` 会清理为 `/explore?keep=1` 并写入本地 token；使用假 token 时后续首页刷新因登录态校验 401 回到 `/login`，符合负向预期。
- [x] 浏览器烟测：临时无数据库后端返回 OAuth2 + Geetest 配置时，`/login` 注册 tab 能经 Next rewrite 拉取 `auth-config`，加载 Geetest 外部脚本并挂载出“点击按钮开始验证”控件；未解 CAPTCHA、未提交注册；页面无新增控制台 error，仅保留既有 Next 图片 LCP 和 Tiptap duplicate warning。
- [x] 静态验证：`/api/search` 封装、首页/探索页搜索表单和 6 个语言包文案已通过 `npm.cmd run lint` 与 `npm.cmd run build`；真实搜索结果仍需可用数据库后端联调。
- [x] 生产 runtime 验证：`next start` 指向不可达后端时，默认 `/` 返回 500 且不渲染 fixture；显式 `FEED_FIXTURE_FALLBACK=true` 后 `/` 返回 200 且渲染 fixture feed。
- [x] 浏览器烟测：`/` 桌面搜索框可输入并提交关键词，页面进入 “Results for travel” 状态且无新增控制台 error；当前本机 `3001` 后端未监听，真实搜索结果和分页仍需数据库后端联调。
- [x] 浏览器烟测：`/` fixture feed 可打开第一张帖子详情，详情抽屉内可见作者 `Follow` 按钮、标题和评论输入，控制台无 error；当前本机 `3001` 后端未监听，真实关注/取关动作仍需数据库后端、Redis 和普通用户测试账号联调。
- [x] 浏览器烟测：详情抽屉桌面端 Vaul open-state transform 修正后，`/` fixture feed 可打开第一张帖子详情并展开“更多操作”，可见 `Not interested`、举报原因下拉、补充说明、`Report` 和 `Share`，控制台无 error；真实不喜欢/举报提交仍需数据库后端、Redis 和登录账号联调。
- [x] 浏览器烟测：新增 `/notifications`、`/messages` 页面可渲染且无控制台 error；本机 `3001` 后端未监听，真实数据联调待后端服务与测试账号。
- [x] 浏览器烟测：`/publish` 创作经营区块可渲染且无控制台 error；本机 `3001` 后端未监听，真实创作者数据联调待后端服务与测试账号。
- [x] 浏览器烟测：`/publish` 创作者明细分页入口改动后仍可渲染且无控制台 error。
- [x] 浏览器烟测：`/wallet` 未登录态可渲染且无控制台 error；真实钱包/提现数据联调待后端服务与测试账号。
- [x] 浏览器烟测：`/wallet` 在本地 mock 后端下可渲染登录态钱包数据；保存收款码提交更新后的微信/支付宝 URL，提现申请提交 `{ amount: 88, type: "cash" }` 并新增提现订单，创作者收益转入提交 `{ amount: 50 }` 后本地月币和创作者收益数值更新，控制台无 error。
- [x] 浏览器烟测：`/messages` 搜索入口和未登录态可渲染且无控制台 error；真实 IM 数据联调待后端服务与测试账号。
- [x] 浏览器烟测：`/messages` 在 IM 分页响应对齐后仍可渲染未登录态，控制台无 error。
- [x] 浏览器烟测：`/admin` 管理员登录/未登录态可渲染且无控制台 error；真实后台数据联调待后端服务与管理员测试账号。
- [x] 浏览器烟测：`/profile` 本人资料编辑在本地 mock 后端下可渲染模拟用户，编辑弹窗字段完整，提交后 mock 收到 `nickname`、`bio`、`location`、`avatar`、`background` payload，页面更新为新昵称/简介/地点且控制台无 error；真实保存仍待数据库、Redis 和普通用户测试账号。
- [x] 浏览器烟测：`/user/target-user` 在本地 mock 后端下可渲染他人主页；关注/取关分别命中 `POST`/`DELETE /api/users/target-user/follow` 并同步按钮与粉丝数；私信命中 `/api/im/conversations`，提交 `member_ids: [9202]` 并跳转 `/messages?conversation=conv-target-smoke`，控制台无 error。
- [x] 浏览器烟测：`/notifications` 在本地 mock 后端下可渲染登录态通知数据；全部已读、删除通知、系统通知确认和系统通知移除分别命中对应后端路由，页面已读状态和列表内容随返回结果更新，控制台无 error。
- [x] 后端无数据库模式烟测：`/api/health` 返回 200；`/_gin_migration/status` 返回 200 且显示 394 条 HTTP 路由、1 个 WebSocket、0 个代理路由；`/api/posts?page=1&limit=1` 返回 500，证明真实业务联调必须先接入数据库。
- [x] 后端无数据库认证引导接口烟测：`/api/auth/auth-config` 返回 OAuth2-only + Geetest 配置；`/api/auth/captcha` 返回 SVG captcha；`/api/auth/oauth2/login` 返回 OAuth2 302 授权跳转；前端 `/api/auth/auth-config` 代理返回同样配置。
- [x] 后端测试：`go test ./internal/http/server`。
- [x] 后端测试：`go test ./...`。
- [x] 后端测试：`go test ./internal/http/handlers` 覆盖附件上传 `audio/*` MIME 放行。
- [ ] 登录联调。
- [ ] 首页和详情联调。
- [x] 用户页联调前端路径烟测。
- [ ] 发布/上传联调。
- [x] 布局影响记录。
- [x] 未完成/阻塞项记录。
