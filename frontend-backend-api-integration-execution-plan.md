# 前后端 API 对接执行计划

## 当前事实

- 前端目录为 `front-end-nextjs`，使用 Next.js App Router、Next 16.2.7、React 19.2.4。
- 后端目录为 `backend-gin`，本地路由矩阵包含 394 条 API、26 个模块和 1 个 WebSocket 入口 `/api/im/ws`。
- 后端统一响应主体主要为 `{ code, message, data }`，部分上传接口仍返回兼容字段如 `success`。
- 前端已完成统一请求层，并已接入登录、个人页、用户页、feed、详情评论、互动、发布上传、通知页、私信页、发布工作台创作者中心、钱包提现和基础后台管理等核心入口；剩余工作主要是真实后端服务、普通/管理员测试账号和跨账号业务联调。
- 个人页已接入本人资料编辑：`/profile` 的“编辑资料”会打开资料表单，提交 `nickname`、`bio`、`location`、`avatar`、`background` 到后端 `PUT /api/users/:id`，成功后用后端返回的用户 payload 刷新本地资料展示。
- 个人页本人资料编辑已完成浏览器级烟测：通过本地 mock 后端和浏览器直连后端配置模拟登录态用户，`/profile` 可渲染用户资料，点击“编辑资料”会打开弹窗，提交新昵称、简介和地点后会发出 `PUT /api/users/smoke-user`，payload 包含 `nickname`、`bio`、`location`、`avatar`、`background`，页面展示随返回 payload 更新且控制台无 error。
- 登录页已补充读取 `/api/auth/auth-config`：可按后端配置显示 OAuth2 登录入口、OAuth2-only 提示、邮箱验证码注册字段；Geetest 启用时已加载 v4 widget，注册提交会携带 `lot_number`、`captcha_output`、`pass_token`、`gen_time`，真实注册仍需有效 Geetest key、数据库、Redis 和测试账号联调。
- 登录页已新增“用户中心一键登录”按钮：后端 `auth-config` 返回 `oauth2StartUrl: "/api/auth/oauth2/login"` 时，前端优先跳转该同源启动入口，由后端生成 state、PKCE、DPoP 等授权参数后再 302 到外部授权站点；若前端配置了 `NEXT_PUBLIC_API_BASE_URL`，按钮会自动拼接直连后端地址。
- OAuth2 回调目标 `/explore` 已存在；前端会在首屏早期脚本中清理 `oauth2_login/access_token/refresh_token` URL 参数、保存 token，并在客户端 handler 中补拉 `/api/auth/me` 写入用户快照。
- 已修复 OAuth2 真实回调后仍停留登录页的问题：当后端 callback 已回到 `/explore?oauth2_login=success&access_token=...&refresh_token=...` 时，`/explore` 和 `/` 会先允许 OAuth success callback 渲染一个空 feed 壳，避免 SSR feed 401 在 token bootstrap 前重定向 `/login`；回调脚本会同时写入 `localStorage` 和同源 `yuem_access_token` cookie，服务端 API helper 会从该 cookie 还原 `Authorization`，客户端补拉 `/api/auth/me` 后跳回首页。
- 根布局已接入全局系统弹窗通知组件：有本地普通用户 token 时会只读拉取 `/api/notifications/system/popup`，无 token 或 401 时静默跳过，不在公共页触发全局跳登录；弹窗支持确认 `/api/notifications/system/:id/confirm`、移除 `/api/notifications/system/:id/dismiss` 和查看通知链接。
- 前端已新增 `front-end-nextjs/.env.example`，记录本地默认后端 `http://localhost:3001`、Next rewrite 所需 `BACKEND_ORIGIN` / `NEXT_PUBLIC_BACKEND_ORIGIN`、浏览器直连后端时使用的 `NEXT_PUBLIC_API_BASE_URL`、服务端可选 `API_BASE_URL`、fixture 兜底开关、HTTP 超时/重试和真实联调测试账号变量。
- 根目录已新增 `frontend-backend-api-integration-env.md`，集中说明 readiness 和 Next.js 实际读取的 `.env` 层级、本地/远端后端地址配置、用户 A/B/管理员 smoke 凭据、可写 smoke 安全开关和当前环境阻塞项，避免把真实 token/password 写入示例文件。
- 根目录已新增真实联调就绪检查脚本 `scripts/check-integration-readiness.mjs`，用于在不打印密码或 secret 的前提下检查核心工具、DB/Redis 连通性、后端 health、前端后端地址配置、前端配置后端 health 可达性、auth-config 契约、认证公共辅助接口契约、OAuth2 start 302 契约、OAuth2 backend callback 到前端 `/explore` 的回调契约、OAuth2 登录 UI/回调静态契约、OAuth2 start URL 静态契约、前端认证 session/refresh/logout 静态契约、发布/上传静态契约、内容详情/互动静态契约、个人/用户页静态契约、通知/IM 页面静态契约、创作者/钱包/后台页面静态契约、首页 feed 访问模式、搜索访问模式、游客访问登录态/后台只读端点权限边界、创作者/钱包公开配置契约、IM WebSocket 前后端静态契约、后端 route-matrix 关键路由合同、后端无数据库认证引导契约、前端 API 路由契约审计、`.env.example` 配置契约、Next rewrite/API client 后端地址契约、feed fixture 兜底门禁和测试账号环境变量；当提供用户/管理员 access token 或账号密码时，脚本会条件执行 authenticated smoke，验证 `/api/auth/me`、`/api/auth/admin/me`、个人/用户页只读链路、双普通用户跨账号用户页只读链路、登录态首页内容端点、从 feed 派生的详情/评论/不喜欢状态/举报状态只读链路、搜索、通知未读数、通知列表、系统通知列表、系统弹窗通知、IM 会话列表、IM sync、会话消息分页、创作者中心只读端点、余额/提现只读端点、后台统计、AI 审核状态、游客访问状态、后台用户/帖子/内容审核/举报列表和提现审核订单列表；HTTP 只读检查默认会重试 1 次，可通过 `INTEGRATION_HTTP_RETRY_COUNT` 调整。
- 根目录已新增 readiness 自检脚本 `scripts/check-integration-readiness-selftest.mjs`，会隔离敏感环境变量并自动验证默认不跑可写 smoke、fixture fallback 门禁、开启可写 smoke 但缺凭据、假用户 token 和假管理员 token 等负向场景；该脚本不要求真实后端 ready，用于确认 readiness 安全门禁、mutation 前置鉴权和报告脱敏行为没有退化。
- `INTEGRATION_ENABLE_WRITE_SMOKE=true` 时会额外启用可写 smoke：用户 A 临时保存并恢复已有提现收款码、创建并删除最小草稿帖、针对专用测试帖子执行点赞/收藏 toggle 和评论创建/删除恢复、普通用户 A/B 关注状态翻转并恢复、IM direct 会话发唯一烟测消息、管理员 AI 审核和游客访问 runtime toggle 临时翻转并恢复；默认关闭，避免常规 readiness 写入真实业务数据或后台配置；开启写 smoke 但缺少对应用户/管理员凭据或 `INTEGRATION_WRITE_SMOKE_POST_ID` 时会显式失败。
- 首页/探索页 feed fixture 兜底已限制为开发环境默认启用，或通过 `FEED_FIXTURE_FALLBACK=true` 显式开启；生产/真实联调默认不再把后端失败静默渲染成 fixture 数据。
- 环境里 `rg`、`git`、`node v24.10.0`、`npm 11.6.1`、`go 1.25.7` 可用；`fd`、`bat`、`delta`、`fzf`、`uv` 缺失且当前 shell 无 `winget` 可安装。
- 当前机器无 `docker`、`psql`、`mysql`，未发现可直接用于本地建库的 compose、schema SQL 或 seed 脚本；`backend-gin/.env.example` 明确说明后端启动不会自动变更数据库 schema。

## 本地后端验证结论（2026-06-08）

- 常规启动会读取 `backend-gin/.env`，其中数据库配置指向当前机器不可解析的 PostgreSQL 主机，因此 `go run ./cmd/api` 在创建 router 阶段失败，`localhost:3001` 不会监听。
- 使用进程级空白占位覆盖数据库变量后，后端可以进入无数据库模式并监听 `3001`：`DATABASE_URL`、`DB_HOST`、`DB_USER`、`DB_PASSWORD`、`DB_NAME`、`DB_PORT`、`DB_DRIVER` 设为空白字符串，`QUEUE_ENABLED=false`，`GIN_PORT=3001`。
- 无数据库模式验证结果：`GET /api/health` 返回 200；`GET /_gin_migration/status` 返回 200，显示 `registered_http_routes=394`、`native_http_routes=394`、`proxy_http_routes=0`、`websocket_entries=1`。
- 无数据库模式下业务接口按设计返回 500，例如 `GET /api/posts?page=1&limit=1` 被 `databaseAvailability` 拦截为“数据库未配置”；因此它只能验证后端壳、路由矩阵和前端代理，不能完成真实业务联调。
- 已在 `databaseAvailability` 无数据库放行清单中加入 `/api/auth/auth-config`、`/api/auth/email-config`、`/api/auth/captcha`、`/api/auth/oauth2/login`。这些接口 handler 本身不依赖数据库，现在可用于本地空库模式验证登录页 OAuth2-only / Geetest / captcha 展示和 OAuth2 授权启动跳转；注册、登录、内容等真实业务接口仍会在无数据库模式下返回“数据库未配置”。
- 无数据库模式下已验证 `GET /api/auth/oauth2/login` 返回 302 到外部授权地址，并携带 `client_id`、`state`、`code_challenge`、`code_challenge_method=S256`、`dpop_jkt`、`redirect_uri` 等参数。
- 真实联调的硬前置是可连接且已有 schema/data 的数据库、Redis 会话服务、普通用户测试账号、第二个普通用户测试账号和管理员测试账号。
- 2026-06-09 复跑 `node scripts\check-integration-readiness.mjs` 结果为 `not-ready`：39 项检查中 30 pass、3 warn、6 fail；`frontend-feed-fixture-fallback`、`frontend-backend-health`、`frontend-auth-config-contract`、`frontend-auth-public-aux-contract`、`frontend-oauth2-start-redirect`、`frontend-oauth2-callback-contract`、`frontend-oauth2-ui-contract`、`frontend-oauth2-start-url-contract`、`frontend-auth-session-contract`、`frontend-publish-upload-contract`、`frontend-content-interaction-contract`、`frontend-profile-user-contract`、`frontend-notifications-im-contract`、`frontend-creator-wallet-admin-contract`、`frontend-initial-feed-access`、`frontend-search-access`、`frontend-guest-protected-access`、`frontend-monetization-config-contract`、`frontend-im-websocket-contract`、`backend-route-matrix-contract`、`backend-auth-bootstrap-no-db-contract`、`frontend-api-route-audit`、`frontend-env-example-contract` 和 `frontend-backend-address-contract` 检查已通过，确认真实联调门禁下未开启 feed fixture 兜底，前端当前后端 origin `https://xse.yuelk.com/api/health` 可达，auth-config 支持 OAuth2-only + Geetest 登录配置，`/api/auth/email-config` 与 auth-config 的 `emailEnabled` 一致，`/api/auth/captcha` 返回可用 SVG captcha payload，`/api/auth/oauth2/login` 会 302 到 `https://user.yuelk.com/oauth2.1/authorize` 并携带 `state`、PKCE S256、`dpop_jkt`、`redirect_uri` 等参数，`redirect_uri` 指向当前后端 `/api/auth/oauth2/callback` 且后端 callback 成功后回跳 `/explore`，前端登录页静态契约保证“用户中心一键登录”只进入后端 OAuth2 start endpoint 且回调早期清理 token 参数，前端认证 session 静态契约确认普通用户/管理员 token key 隔离、登录/注册/OAuth 回调保存、401 refresh 单次重试和 logout 本地清理仍对齐，发布/上传静态契约确认前端图片、视频、播客音频上传和 `/api/posts` 发布与后端用户鉴权 route-matrix、音频附件 MIME 放行及测试覆盖仍对齐，内容详情/互动静态契约确认详情、评论、点赞、收藏、关注/取关、不感兴趣和举报封装与后端 route-matrix 以及首页 feed/详情抽屉调用关系仍对齐，个人/用户页静态契约确认 `/profile`、`/user/[id]`、资料 tabs、关注状态、关注/取关、钱包/通知/私信入口和用户链接 helper 与后端用户路由对齐，通知/IM 页面静态契约确认通知列表、未读数、已读/删除、系统确认/移除、会话创建、消息分页、发送、已读推进、WebSocket 新消息刷新和用户主页私信跳转均与后端 route-matrix 对齐，创作者/钱包/后台页面静态契约确认创作者数据面板、钱包提现与后台管理页面动作均通过前端 helper 并与后端 route-matrix 对齐，远端首页 feed/热门分类/搜索当前均处于游客受限模式（返回 401 “请登录后查看内容”或等价未授权响应），12 个登录态/后台只读端点均拒绝游客访问，创作者配置、余额配置和充值配置公开端点均符合前端契约，前端 `getImWebSocketUrl()` 与后端 route-matrix 的 `/api/im/ws` WebSocket upgrade 契约一致，后端 route-matrix 合同确认 394 条 HTTP API、26 个 mounted module、1 个 WebSocket 和 39 条关键集成路由均保持 `native-gin` 且 auth/status 符合前端联调预期，后端无数据库认证引导契约确认 `databaseAvailability` 仍放行登录页所需 auth-config/email-config/captcha/OAuth2 start 且未误放行登录/注册/内容业务路径，前端 API 路由审计确认 80 个前端 API 调用形态均匹配后端 route-matrix（0 未匹配、0 宽动态匹配），`.env.example` 覆盖后端 origin、直连后端、fixture、HTTP 重试和测试账号变量，且 Next rewrite/API client 会按配置读取后端地址；账号检查现在接受 `INTEGRATION_*_ACCESS_TOKEN` 或账号密码组合，缺失时仍失败；警告项为缺少 `docker`、`psql`、`mysql`；失败项为 DB 主机 `1Panel-postgresql-RHin:5432` 不可解析、Redis 主机 `1Panel-redis-Mr0U:6379` 不可解析、`http://localhost:3001/api/health` 不可达，以及普通用户 A/B、管理员测试账号环境变量缺失。前端当前未配置浏览器直连后端的 `NEXT_PUBLIC_API_BASE_URL`。
- 2026-06-09 后续复跑新增系统弹窗通知覆盖：`node scripts\check-integration-readiness.mjs` 仍为 `not-ready`，39 项基础检查中 30 pass、3 warn、6 fail；`frontend-guest-protected-access` 已扩大到 13 个受保护只读端点并确认 `/api/notifications/system/popup` 游客 401，`backend-route-matrix-contract` 关键路由数为 40，`frontend-api-route-audit` 记录 81 个前端 API 调用形态且仍为 0 未匹配、0 宽动态匹配。临时 `INTEGRATION_USER_A_ACCESS_TOKEN=codex-invalid-token` 负向 smoke 会报告 `systemNotificationPopupStatus: 401`，且不输出 token。
- 2026-06-09 继续复跑 readiness 结果保持 `not-ready`：40 项基础检查中 31 pass、3 warn、6 fail；新增 `frontend-env-doc-contract` 并通过，确认 `frontend-backend-api-integration-env.md` 覆盖 `.env` 读取层级、后端地址、用户 A/B/管理员凭据、可写 smoke 和负向验证说明，且 `.env.example` 指向该文档。失败项仍为 DB 主机不可解析、Redis 主机不可解析、本地后端 health 不可达、普通用户 A/B 和管理员测试账号变量缺失。默认配置下未触发 `user-withdraw-payment-code-write-smoke` 等可写 smoke；临时设置 `INTEGRATION_ENABLE_WRITE_SMOKE=true` 但不提供用户 A 凭据时，提现收款码和草稿写 smoke 均在“缺少用户 A 凭据”处失败；提供假 `INTEGRATION_USER_A_ACCESS_TOKEN=codex-invalid-token` 时，提现收款码和草稿写 smoke 均先验证 `/api/auth/me` 并因 401 停止在 mutation 前，报告不泄露 token。
- 2026-06-09 新增并运行 `node scripts\check-integration-readiness-selftest.mjs` 通过：5 个场景均符合预期，baseline 不触发任何 `*-write-smoke`，`FEED_FIXTURE_FALLBACK=true` 会让 `frontend-feed-fixture-fallback` 失败，开启写 smoke 但无凭据会让 6 个写 smoke 均显式失败，假用户 token 会让提现收款码/草稿/帖子互动写 smoke 均停在 `/api/auth/me` 401 且 mutation 前失败，假管理员 token 会让管理员 runtime toggle 停在 `/api/auth/admin/me` 401 且 mutation 前失败，报告不输出传入 token。

## 下一步计划

1. 先收口认证真实联调：Geetest widget 已接入并通过本地展示烟测；后续需在真实后端、有效 Geetest key、数据库和 Redis 下验证注册提交、失败重置、登录态保存与 refresh。
2. 真实联调前先运行 `node scripts\check-integration-readiness.mjs`，只在 DB、Redis、后端 health 和测试账号全部通过后进入真实业务动作验证。
3. 补齐联调环境：让 `backend-gin/.env` 或进程环境指向本机可访问数据库；确认 Redis 可用，因为登录会话、refresh token 和用户态鉴权依赖 Redis session；准备普通用户 A/B 与管理员账号。
4. 启动并验证真实后端：`cd backend-gin; go run ./cmd/api` 后确认 `http://localhost:3001/api/health`、`/_gin_migration/status` 正常，且 `GET /api/posts?page=1&limit=1` 不再返回“数据库未配置”。
5. 按依赖顺序联调认证：auth-config、注册/登录、`/api/auth/me`、refresh、logout、OAuth2 start/callback，确认前端 session 保存、401 refresh、OAuth2-only 展示、登录跳转和回调 token 清理行为。
6. 联调公开内容链路：首页 feed、分类、详情、评论列表，确认分页字段、空态、错误态和图片/视频地址映射。
7. 联调登录后互动链路：点赞、收藏、发评论、用户页、个人页、关注/取关，确认 mutation 后局部刷新和计数同步。
8. 联调发布上传：图片、视频、草稿、发布成功后的详情可访问；确认上传进度、失败重试、文件 URL 与后端存储策略一致。
9. 联调通知与 IM：用户 A/B 双账号创建会话、发送/接收、`/api/im/ws` 推送、历史消息翻页、已读状态和断线重连。
10. 联调创作者与钱包：创作者概览/趋势/收益流水、钱包余额、收款码保存、提现申请、充值跳转和订单刷新。
11. 联调后台管理：管理员登录、当前管理员、统计、AI/访客开关、用户/内容/审核/举报列表、提现审核 approve/reject/payout。
12. 每完成一组真实联调后复测 `npm.cmd run lint`、`npm.cmd run build`、关键页面浏览器烟测；若后端代码有改动，补跑 `go test ./...`。

## 对接顺序

1. 建立统一请求层，收口 base URL、token、响应拆包、错误处理、上传和常用 HTTP 方法。
2. 对接认证基础：登录、注册、登出、刷新、当前用户，并让登录页从占位页变为真实表单。
3. 对接现有用户相关页面：`/profile`、`/user/[id]` 的资料、笔记、收藏、赞过、关注状态和关注操作。
4. 对接 feed 详情和互动：帖子详情、评论列表、发评论、点赞、收藏，移除详情抽屉示例评论。
5. 对接发布页基础能力：创建帖子、保存草稿、上传图片/视频，并保留现有布局与工作台结构。
6. 继续扩展通知、IM、创作者中心、钱包和后台管理；当前前端没有页面的模块先记录接口归属和待实现入口。

## 设计约束

- 默认不重做页面布局；只补必要的 loading、empty、error、分页、上传进度和鉴权提示。
- 所有业务请求必须经 `src/lib/api.ts` 或其模块化封装进入。
- 鉴权请求使用 `Authorization: Bearer <access_token>`，refresh token 用于续期；客户端持久化仅保存 token 和必要用户快照。
- 服务端渲染首屏可通过 cookie 透传旧会话，也可在未登录时走公开/可选鉴权接口；客户端互动依赖本地 token。
- fixture 只能作为开发兜底或空数据参考，真实业务路径不得静默假成功。

## 交付与验证

- 更新代码后运行 `npm.cmd run lint` 和 `npm.cmd run build`。
- 对后端变更运行 Go 测试；本轮优先改前端，不主动改后端路由。
- 真实联调前运行 `node scripts\check-integration-readiness.mjs`，确认状态从 `not-ready` 变为 `ready` 后再执行账号和业务链路验证。
- 联调路径：登录、首页 tab、打开详情、评论、点赞、收藏、个人页、用户页、发布图文/视频。
- 若发生布局影响，记录页面、改动、原因、对应接口、影响范围和验证结果。

## 本轮进展

- 已完成统一请求层、Blob/下载响应入口、认证表单、用户页/个人页、关注状态、Feed 详情、Feed 分页/加载更多、点赞、收藏、评论、图片/视频上传和发布/草稿基础对接。
- 已接入认证配置读取 `/api/auth/auth-config`、邮箱验证码发送 `/api/auth/send-email-code`，并让登录页根据 OAuth2-only、邮箱注册和 Geetest 开关调整可见入口与注册提交状态。
- 已完成后端无数据库认证引导接口放行：`/api/auth/auth-config`、`/api/auth/email-config`、`/api/auth/captcha`、`/api/auth/oauth2/login` 在空数据库配置下可返回或跳转；新增后端测试锁定 auth-config/email-config/captcha 200、OAuth2 start 302，同时确认 `/api/posts?page=1&limit=1` 仍返回 500 “数据库未配置”。
- 已将登录页 OAuth2 文案和入口收口为“用户中心一键登录”按钮：默认使用 `/api/auth/oauth2/login` 同源启动入口，并在 `auth-config` 中兼容读取 `oauth2StartUrl`；配置 `NEXT_PUBLIC_API_BASE_URL` 时会拼到后端 origin，避免整页跳转仍落在前端域名。
- 已按 `demo_oauth21` 的服务端 OAuth2.1 启动模式复核前端入口：按钮只进入后端启动端点，PKCE、state、DPoP 和授权站点 302 均由后端完成；前端不会直跳授权中心基础 `oauth2LoginUrl`。`NEXT_PUBLIC_API_BASE_URL` 直连拼接已改用 URL API，兼容末尾斜杠或误带 `/api` 路径的配置。
- 已新增全局 OAuth2 回调处理：早期内联脚本会在首屏清理 URL 中的 `access_token`/`refresh_token` 等敏感参数并保存 token，客户端 handler 随后用 token 拉取 `/api/auth/me` 并刷新页面用户态。
- 已修正 OAuth2 回调早期脚本挂载位置：`OAuthCallbackBootstrap` 现在位于根布局 `<head>` 内，避免 `<script>` 作为 `<html>` 直接子节点导致 React/Next hydration 结构错误。
- 已新增 `/explore` 页面作为后端 OAuth2 callback redirect 的可渲染目标，复用现有 feed 页面。
- 已接入 Geetest v4 注册 widget：登录页在后端 `auth-config` 返回 `geetest.enabled` 和 `captchaId` 时加载 `https://static.geetest.com/v4/gt4.js`，将 widget 挂载到注册表单，并在注册提交时向 `/api/auth/register` 传递 Geetest 验证 payload；验证失败或注册失败后会重置 widget。
- 已接入首页/探索页搜索入口：桌面搜索框和移动搜索按钮现在调用后端 `/api/search`，按后端 `keyword/tag/type/page/limit` 契约归一化 `type=all` 与 `type=posts/videos` 的帖子结果，并复用现有瀑布流、详情、点赞、收藏和加载更多逻辑。
- 已加固统一请求层对“成功但无 `data`”响应的拆包行为：后端如关注/取关、已读、后台开关等只返回 `code/message` 时，前端现在统一得到空对象，避免把 envelope 误当业务 payload。
- 已完成上传进度百分比；上传层在存在进度回调时使用 XHR 分支，保留原 `apiUpload` 调用兼容。
- 已完成播客音频发布基础对接：前端 podcast 模式上传走 `/api/upload/attachment`，后端附件 MIME 放行 `audio/*`，发布 `/api/posts` 时复用现有 `attachment: { url, filename, filesize }` 字段，不新增专用 podcast schema 或 handler。
- 已完成详情抽屉附件展示：`post.attachment` 会显示文件名、大小和外链入口；识别常见音频扩展名时内嵌原生 audio 播放器，用于发布后回看播客音频。
- 已接入现有入口的后续模块：个人页通知未读数 `/api/notifications/unread-count`、用户页私信会话创建 `/api/im/conversations`、发布工作台创作者概览/统计 `/api/creator-center/overview` 和 `/api/creator-center/stats`。
- 已清理旧用户 fixture 工具：`src/lib/users.ts` 不再提供本地假用户资料、收藏和点赞 tab 数据，仅保留从真实/fixture 帖子字段解析作者 ID 与 `/user/[id]` 链接的工具；个人页和用户页数据继续统一走 `src/lib/api.ts` 的真实用户接口。
- 已新增 `/notifications` 页面，接入通知列表、系统通知列表、未读数、标记已读、全部已读、删除通知、系统通知确认和移除。
- 已新增 `/messages` 页面，接入 IM 会话列表、会话搜索、消息列表、历史消息向上翻页、发送消息、已读推进、`/api/im/ws` 新消息刷新和断线重连；用户主页私信成功后会跳转到对应会话。
- 已按后端真实响应修正 IM 历史消息分页：`GET /api/im/conversations/:id/messages` 的消息数组在 `data`，分页在顶层 `pagination`，前端现在保留 envelope 并使用 `has_more`、`next_before` 推进“加载更早消息”。
- 已扩展发布工作台首页创作者中心，新增创作经营区块并接入 `/api/creator-center/trends`、`/api/creator-center/earnings-log`、`/api/creator-center/paid-content`、`/api/creator-center/quality-rewards`；收益流水、付费内容和质量奖励已支持加载更多分页；接口失败或未登录时保留空态，不阻塞发布工作台。
- 已新增 `/wallet` 页面和个人页入口，接入余额配置、充值配置、本地月币、外部月币、提现钱包、收款码保存、提现申请、提现订单、购买订单和创作者收益转月币；接口失败时按模块记录错误，不阻塞其余钱包数据展示。
- 已扩展 `/wallet` 充值配置展示：页面会从 `/api/balance/recharge-config` 渲染月币充值档位、礼品卡档位和自定义充值金额范围；有 `recharge_url` 时充值入口统一跳外部用户中心充值页。
- 已新增 `/admin` 页面，接入管理员登录 `/api/auth/admin/login`、当前管理员 `/api/auth/admin/me`、后台统计 `/api/admin/stats/overview`、AI 审核状态/开关、访客访问状态/开关、用户/内容/审核/举报分页列表和提现审核订单/动作；管理端 token 独立存储，不覆盖普通用户会话。
- 已保留现有页面结构；新增 UI 仅限登录表单、上传文件状态/进度与失败重试、通知未读徽标、系统弹窗通知、评论输入/loading/empty/error、后台管理表格/筛选/分页/开关等接口必要状态。
- 已运行并通过 `npm.cmd run lint`。
- 已运行并通过 `npm.cmd run build`。
- 已新增可复跑的前端 API 路由契约审计：`npm.cmd run audit:api` 会用 TypeScript AST 收集前端 `/api/...` 调用，并与后端 `route-matrix.json` 做方法和动态路径匹配；脚本会读取 `AdminListResource` 白名单展开 `/api/admin/${resource}`，当前结果为 81 个前端 API 调用形态、0 个未匹配、0 个宽动态匹配。
- 已把后端 route-matrix 关键路由合同纳入真实联调 readiness 门禁：`backend-route-matrix-contract` 会读取 `backend-gin/internal/http/routes/route-matrix.json`，校验 394 条 HTTP API、26 个 mounted module、1 个 WebSocket 的摘要未漂移，所有 route/WebSocket 都是 `native-gin`，并锁定认证、OAuth2、feed/评论/互动、上传、通知、IM、创作者、钱包提现和后台统计/提现审核等 40 条关键集成路由的 method/auth/status；当前检查通过。
- 已把后端无数据库认证引导契约纳入真实联调 readiness 门禁：`backend-auth-bootstrap-no-db-contract` 会静态检查 `databaseAvailability` 仍只放行 `/api/health`、`/_gin_migration/status`、`/api/auth/auth-config`、`/api/auth/email-config`、`/api/auth/captcha`、`/api/auth/oauth2/login` 等无 DB 启动路径，不误放行 `/api/auth/login`、`/api/auth/register`、`/api/auth/me`、`/api/posts` 等真实业务路径，并确认 `AuthMatrix` dispatch 与 `TestDatabaseOptionalAuthBootstrapRoutes` 测试覆盖仍在；当前检查通过。
- 已把游客权限边界纳入真实联调 readiness 门禁：`frontend-guest-protected-access` 会以无 token 状态检查 `/api/auth/me`、通知、系统弹窗通知、IM 会话、创作者概览、用户余额、提现钱包、管理员 me/统计/用户列表/提现审核订单等 13 个只读登录态/后台端点，要求均返回 401/403；当前远端均返回 401 “访问令牌缺失”，说明这些受保护端点未向游客开放。
- 已把前端 API 路由契约审计纳入真实联调 readiness 门禁：`frontend-api-route-audit` 会直接执行 `front-end-nextjs/scripts/audit-api-routes.mjs`，要求审计状态为 pass、`unmatchedApiCalls=0` 且 `broadDynamicMatches=0`；当前检查通过，记录 81 个前端 API 调用形态、394 条后端 HTTP 路由和 1 个 WebSocket。
- 已把 `.env.example` 配置覆盖纳入真实联调 readiness 门禁：`frontend-env-example-contract` 会检查 `BACKEND_ORIGIN`、`NEXT_PUBLIC_BACKEND_ORIGIN`、`NEXT_PUBLIC_API_BASE_URL`、`API_BASE_URL`、fixture 开关、HTTP timeout/retry 和用户 A/B/管理员测试账号变量均有示例，后端 URL 示例可解析，且示例文件没有默认启用 fixture fallback 或写入真实 token/password。
- 已把环境速查文档纳入真实联调 readiness 门禁：`frontend-env-doc-contract` 会检查 `frontend-backend-api-integration-env.md` 存在并覆盖 Next/readiness 的 `.env` 读取层级、后端地址、用户 A/B/管理员凭据、可写 smoke 检查、假 token 负向验证和禁止写入真实 token 的说明，同时确认 `.env.example` 指向该文档。
- 已新增 readiness 自检脚本：`check-integration-readiness-selftest.mjs` 复用真实 readiness 输出进行负向断言，防止可写 smoke 默认运行、fixture 兜底误开、缺凭据写入、无效 token 进入 mutation 或报告泄露 token。
- 已把 Next rewrite/API client 后端地址接线纳入真实联调 readiness 门禁：`frontend-backend-address-contract` 会检查 `next.config.ts` 使用 `BACKEND_ORIGIN` / `NEXT_PUBLIC_BACKEND_ORIGIN` 代理 `/api/:path*`，并确认 `src/lib/api.ts` 在浏览器端支持 `NEXT_PUBLIC_API_BASE_URL` 直连、服务端按 `API_BASE_URL`、`BACKEND_ORIGIN`、`NEXT_PUBLIC_BACKEND_ORIGIN` 顺序解析后端 origin。
- 已完成浏览器烟测：`/`、`/publish`、`/login` 可渲染且无控制台 error；`/profile` 在未登录/后端不可用时显示资料加载失败。
- 已复测 `/login`：后端不可用时登录表单与注册 tab 均可渲染，注册 tab 会尝试加载 SVG captcha 但不会产生控制台 error；临时无数据库后端返回 OAuth2-only + Geetest 配置时，前端经 Next rewrite 能显示“用户中心一键登录”按钮和“当前环境仅支持用户中心登录。”提示，页面无控制台 error。
- 已完成浏览器级 OAuth2-only 登录入口烟测：在现有 `localhost:3000` Next dev server 上通过 CDP 拦截 `/api/auth/auth-config` 返回 OAuth2-only 配置，页面真实渲染 `<a href="/api/auth/oauth2/login">用户中心一键登录</a>`，显示“当前环境仅支持用户中心登录。”，密码表单隐藏，且无 script/hydration 结构类 console error。
- 已完成 OAuth2 回调浏览器烟测：访问 `/explore?oauth2_login=success&access_token=smoke-access&refresh_token=smoke-refresh&keep=1` 后，地址栏会清理为 `/explore?keep=1` 并写入本地 token；使用假 token 时后续首页刷新因 `/api/auth/me`/feed 401 回到 `/login`，符合负向预期。真实 token 场景依赖后端返回有效本地 token、Redis session 和生产部署后的新前端代码。
- 已完成 Geetest 注册页浏览器烟测：临时无数据库后端返回 OAuth2 + Geetest 配置时，`/login` 注册 tab 能经 Next rewrite 拉取 `auth-config`，加载 Geetest 外部脚本并挂载出“点击按钮开始验证”控件；未解 CAPTCHA、未提交注册；页面无新增控制台 error，仅保留既有 Next 图片 LCP 和 Tiptap duplicate warning。
- 已完成搜索接入静态验证：`/api/search` 封装、首页/探索页搜索表单和 6 个语言包文案已通过 `npm.cmd run lint` 与 `npm.cmd run build`；真实搜索结果仍需可用数据库后端联调。
- 已收紧 feed fixture 兜底策略：`getInitialFeedData` 仅在开发环境或显式设置 `FEED_FIXTURE_FALLBACK=true` 时返回 `fixtureInitialFeedData`；生产默认会暴露后端错误，避免真实联调时把 API 不可用误判为页面正常。
- 已完成生产运行时验证：`next start` 指向不可达后端时，默认 `/` 返回 500 且不含 fixture feed；设置 `FEED_FIXTURE_FALLBACK=true` 后 `/` 返回 200 且渲染 fixture feed。验证后已清理临时 `3101/3102` 监听进程。
- 已把 feed fixture 兜底纳入真实联调 readiness 门禁：默认检查项 `frontend-feed-fixture-fallback` 通过；临时设置 `FEED_FIXTURE_FALLBACK=true` 复跑时会失败并提示该配置会掩盖后端错误。
- 已把前端后端地址可达性纳入真实联调 readiness 门禁：`frontend-backend-health` 会请求前端配置的 backend origin `/api/health`；当前 `https://xse.yuelk.com/api/health` 返回 200，临时覆盖 `BACKEND_ORIGIN=http://127.0.0.1:39999` 时该检查按预期失败。
- 已把登录配置契约纳入真实联调 readiness 门禁：`frontend-auth-config-contract` 会请求前端配置的 backend origin `/api/auth/auth-config`，校验 `emailEnabled`、`oauth2Enabled`、`oauth2OnlyLogin`、`geetestEnabled` 布尔字段、OAuth2 start URL 可解析性，以及 Geetest 开启时必须存在 `geetestCaptchaId`；当前 `https://xse.yuelk.com/api/auth/auth-config` 通过，远端未返回 `oauth2StartUrl` 时前端 fallback 到 `/api/auth/oauth2/login`。
- 已把认证公共辅助接口纳入真实联调 readiness 门禁：`frontend-auth-public-aux-contract` 会只读请求 `/api/auth/email-config` 和 `/api/auth/captcha`，校验 `emailEnabled` 与 auth-config 一致、captcha 返回非空 `captchaId` 和无 `<script>` 的 SVG `captchaSvg`；当前远端 `emailEnabled=false` 且 captcha SVG payload 符合前端注册页契约。
- 已把 OAuth2 start 跳转契约纳入真实联调 readiness 门禁：`frontend-oauth2-start-redirect` 会在不跟随跳转的情况下请求 `/api/auth/oauth2/login`，校验 302 `Location` 指向 `/oauth2.1/authorize`，并携带 `client_id`、`state`、`code_challenge`、`code_challenge_method=S256`、`redirect_uri` 和 `dpop_jkt`；当前远端检查通过，说明“用户中心一键登录”按钮对应的后端启动端点符合 `demo_oauth21` 的服务端授权启动模式。
- 已把 OAuth2 backend callback 回前端契约纳入真实联调 readiness 门禁：`frontend-oauth2-callback-contract` 会复用真实 start redirect 中的 `redirect_uri`，校验其 origin 为当前后端、path 为 `/api/auth/oauth2/callback`，并静态确认后端成功换取本地 token 后回跳 `/explore?oauth2_login=success&access_token=...&refresh_token=...`，前端 `/explore` 页面可渲染且会放行 OAuth success callback 的客户端 token bootstrap，根布局回调脚本会早期保存 token、同步 SSR auth cookie 并清理 URL 敏感参数；当前检查通过。
- 已把 OAuth2 登录 UI/回调静态契约纳入真实联调 readiness 门禁：`frontend-oauth2-ui-contract` 会检查 `/login` 渲染 `LoginForm`，登录表单读取 `auth-config`、优先使用 `oauth2StartUrl`、fallback 到 `/api/auth/oauth2/login`、支持 `NEXT_PUBLIC_API_BASE_URL` 直连拼接、展示“用户中心一键登录”和 OAuth2-only 提示，并确认 `OAuthCallbackBootstrap` 位于根布局 `<head>` 且会早期保存 token、清理 URL 敏感参数，客户端 handler 会用 token 补拉 `/api/auth/me`。
- 已把 OAuth2 start URL 静态契约纳入真实联调 readiness 门禁：`frontend-oauth2-start-url-contract` 会检查登录按钮只解析 `oauth2StartUrl` 或默认 `/api/auth/oauth2/login` 后端启动端点，禁止误用授权中心根地址 `oauth2LoginUrl`，并确认 `NEXT_PUBLIC_API_BASE_URL` 直连后端拼接逻辑仍在；当前检查通过。
- 已把前端认证 session 静态契约纳入真实联调 readiness 门禁：`frontend-auth-session-contract` 会检查 `src/lib/api.ts` 中普通用户/管理员 localStorage key 隔离、登录/注册/OAuth 回调 token 保存、普通用户 SSR auth cookie 写入/清理、SSR API 请求从 cookie 还原 token、`/api/auth/refresh` 的 refresh_token payload、401 后单次 refresh 重试、不可恢复 401 清理并跳 `/login`、`/api/auth/logout` finally 清理本地普通用户 session，以及管理员 401 仅清理管理员 session；当前检查通过。
- 已把发布/上传静态契约纳入真实联调 readiness 门禁：`frontend-publish-upload-contract` 会检查前端 `uploadImage`、`uploadVideo`、`uploadAttachment` 分别调用 `/api/upload/single`、`/api/upload/video`、`/api/upload/attachment`，`createPost` 调用 `/api/posts`，发布工作台 image/video/podcast 分支复用这些 helper 且保留进度回调，podcast 仅接受 `audio/*` 并映射到 `attachment: { url, filename, filesize }`，后端 route-matrix 中这些上传/发布路由均为用户鉴权 `native-gin`，且后端附件 MIME 放行 `audio/*` 并有测试覆盖拒绝 `video/mp4`；当前检查通过。
- 已把内容详情/互动静态契约纳入真实联调 readiness 门禁：`frontend-content-interaction-contract` 会检查前端详情、评论、点赞、收藏、关注/取关、不感兴趣、举报与状态查询 API 封装，确认游客可读详情/评论使用 `auth: false`，登录态状态查询在无 token/401 时返回 null 而不是强制跳转，首页 feed 调用详情/点赞/收藏封装，详情抽屉调用评论、关注、不感兴趣和举报封装，并校验这些路由在后端 route-matrix 中 method/auth/status 符合预期；当前检查通过。
- 已把个人/用户页静态契约纳入真实联调 readiness 门禁：`frontend-profile-user-contract` 会检查 `/profile`、`/user/[id]`、当前用户资料、用户资料、笔记/收藏/赞过 tab、关注状态、关注/取关、钱包/通知/私信入口和用户链接 helper，并校验这些用户路由在后端 route-matrix 中 method/auth/status 符合预期；当前检查通过。
- 已扩展个人/用户页静态契约：`frontend-profile-user-contract` 现在还会校验 `PUT /api/users/:id` 后端路由、前端 `updateUserProfile()` helper、个人页编辑资料弹窗、昵称必填校验、提交后本地资料状态刷新和 profile 编辑多语言文案；当前检查通过。
- 已完成 `/profile` 本人资料编辑浏览器烟测：使用本地 mock 后端模拟 `/api/auth/me`、用户 tab、通知未读数和 `PUT /api/users/:id`，并通过 OAuth 回调路径写入本地 token；页面渲染模拟用户 `Smoke Viewer`，编辑弹窗显示昵称、简介、地点、头像 URL 和封面 URL 字段，提交 `Updated Smoke`、`Updated profile bio from browser smoke`、`Hangzhou` 后 mock 收到正确 `PUT` payload，弹窗关闭，页面资料更新，控制台无 error。真实资料保存仍需可用数据库、Redis 和普通用户测试账号联调。
- 已完成 `/user/[id]` 用户页浏览器烟测：使用本地 mock 后端模拟 `/api/users/target-user`、三个用户 tab、`/api/users/target-user/follow-status`、关注/取关和 `/api/im/conversations`；页面渲染 `Target Creator` 资料、认证标记、关注/私信入口和空 tab，点击关注会调用 `POST /api/users/target-user/follow` 并把粉丝数从 20 更新为 21，点击已关注会调用 `DELETE /api/users/target-user/follow` 并恢复粉丝数，点击私信会提交 `member_ids: [9202]` 到 `/api/im/conversations` 并跳转 `/messages?conversation=conv-target-smoke`，控制台无 error。真实关注状态持久化、会话列表数据和 WebSocket 消息仍需可用数据库、Redis 与用户 A/B 测试账号联调。
- 已把通知/IM 页面静态契约纳入真实联调 readiness 门禁：`frontend-notifications-im-contract` 会检查前端通知未读数、通知列表、系统通知、根布局全局系统弹窗通知组件、系统弹窗通知 optional helper、标记已读、全部已读、删除、确认、移除、IM 会话创建、会话列表、消息分页、发送、已读推进、WebSocket 新消息刷新和用户主页私信跳转，并校验这些路由在后端 route-matrix 中 method/auth/status 符合预期；当前检查通过。
- 已完成 `/notifications` 通知动作浏览器烟测：使用本地 mock 后端模拟普通用户 token 下的 `/api/notifications`、`/api/notifications/system`、`/api/notifications/unread-count`、`/api/notifications/system/popup`，以及标记、删除、确认和移除动作；页面初始显示 2 条未读，点击“全部已读”命中 `PUT /api/notifications/read-all` 并禁用用户通知已读按钮，删除首条通知命中 `DELETE /api/notifications/user-n-1` 并从列表移除，切到系统通知后确认命中 `POST /api/notifications/system/system-n-1/confirm`，移除命中 `DELETE /api/notifications/system/system-n-1/dismiss`，页面状态随 mock payload 更新且控制台无 error。真实通知持久化、系统弹窗通知和跨用户消息联动仍需可用数据库、Redis 与用户 A/B 测试账号联调。
- 已新增 opt-in 提现收款码写入 smoke：`INTEGRATION_ENABLE_WRITE_SMOKE=true` 且用户 A 凭据可用时，`user-withdraw-payment-code-write-smoke` 会读取 `/api/withdraw/payment-code`；仅当账号已有至少一个收款码 URL 时，才临时写入烟测 URL 并恢复初始值，避免在空账号上创建现有 API 无法清回全空状态的首条收款码记录。
- 已新增 opt-in 草稿发布写入 smoke：`INTEGRATION_ENABLE_WRITE_SMOKE=true` 且用户 A 凭据可用时，`user-draft-post-write-smoke` 会通过 `/api/posts` 创建一条最小 `is_draft: true` 草稿帖，再通过 `/api/posts/:id` 删除清理；草稿不会触发粉丝新帖通知，但仍会写入帖子相关表，默认关闭。
- 已新增 opt-in 帖子互动写入 smoke：`INTEGRATION_ENABLE_WRITE_SMOKE=true`、用户 A 凭据和 `INTEGRATION_WRITE_SMOKE_POST_ID` 可用时，`user-post-interaction-write-smoke` 会读取指定测试帖子的点赞/收藏状态、临时 toggle 点赞和收藏、创建一条唯一烟测评论，再尽量删除评论并恢复点赞/收藏初始状态；缺测试帖子 ID 时显式失败，避免误把真实内容当测试靶子。
- 已新增 opt-in 双账号 IM 写入 smoke：`INTEGRATION_ENABLE_WRITE_SMOKE=true` 且用户 A/B 凭据可用时，`user-cross-account-im-write-smoke` 会由 A 创建/复用与 B 的 direct 会话、发送带唯一 `client_msg_id` 的烟测消息、由 B 读取最近消息并标记已读；默认关闭，避免常规 readiness 写入真实 IM 数据。
- 已新增 opt-in 管理员 runtime toggle 写入 smoke：`INTEGRATION_ENABLE_WRITE_SMOKE=true` 且管理员凭据可用时，`admin-runtime-toggle-write-smoke` 会读取 AI 审核和游客访问状态、临时翻转开关、确认变更，再写回初始值并确认恢复；默认关闭，避免常规 readiness 修改后台运行时配置；若开启写 smoke 但缺少管理员 token 或账号密码，会显式失败，避免误以为后台写链路已覆盖。
- 已把创作者/钱包/后台页面静态契约纳入真实联调 readiness 门禁：`frontend-creator-wallet-admin-contract` 会检查创作者概览/统计/趋势/收益/付费内容/质量奖励分页、钱包配置/余额/收款码/提现/订单、后台登录/me/统计/AI 审核/访客访问/资源列表/提现审核动作，并校验这些路由在后端 route-matrix 中 method/auth/status 符合预期；当前检查通过。
- 已扩展创作者/钱包/后台页面静态契约：`frontend-creator-wallet-admin-contract` 现在还会确认钱包页实际使用充值档位、礼品卡档位和自定义金额范围，避免 `/api/balance/recharge-config` 只被请求但未呈现在钱包页面。
- 已完成 `/wallet` 钱包动作浏览器烟测：使用本地 mock 后端模拟普通用户 token 下的钱包公开配置、充值配置、本地月币、外部月币、创作者概览、提现钱包、收款码、提现订单和购买订单；页面显示充值中心外链、月币充值档位、礼品卡档位和提现表单，保存收款码命中 `POST /api/withdraw/payment-code` 且 payload 包含更新后的微信/支付宝 URL，提交现金提现命中 `POST /api/withdraw/apply` 的 `{ amount: 88, type: "cash" }` 并在刷新后新增 `withdraw-2` 订单，创作者收益转入命中 `POST /api/creator-center/withdraw` 的 `{ amount: 50 }`，页面本地月币从 1200 更新到 1250、创作者收益从 430 更新到 380，控制台无 error。真实提现申请审核、充值跳转后的订单同步和收款码持久化仍需可用数据库、Redis 与普通用户测试账号联调。
- 已把首页初始内容访问模式纳入真实联调 readiness 门禁：`frontend-initial-feed-access` 会检查 `/api/posts/recommended?page=1&limit=1` 和 `/api/categories/hot?limit=12`；当前远端两个端点均返回 401 “请登录后查看内容”，脚本记录为 `guest-restricted`，说明首页/探索页内容链路下一步必须使用有效普通用户会话继续联调。
- 已把创作者/钱包公开配置纳入真实联调 readiness 门禁：`frontend-monetization-config-contract` 会检查 `/api/creator-center/config`、`/api/balance/config` 和 `/api/balance/recharge-config`；当前远端三个端点均返回 200，且创作者费率/提现字段、余额开关、充值 URL 与金额选项符合前端契约。
- 已扩展钱包公开充值配置 payload 校验：`frontend-monetization-config-contract` 现在会验证 `custom_amount_enable`、`min_amount`、`max_amount`、`gift_card_purchase.options`、礼品卡价格/折扣、过期天数和用户折扣列表等字段形状；当前远端响应通过。
- 已把 IM WebSocket 前后端静态契约纳入真实联调 readiness 门禁：`frontend-im-websocket-contract` 会读取后端 route-matrix 中的 `/api/im/ws` WebSocket upgrade，确认 auth 模式为 `query-token-and-redis-session`，并检查前端 `getImWebSocketUrl()` 使用 `/api/im/ws`、按 `http/https` 转换为 `ws/wss`、通过 `token` query 传参，且 `/messages` 页面用该 helper 建立 WebSocket；当前检查通过，示例 URL 为 `wss://xse.yuelk.com/api/im/ws?token=<redacted>`。
- 已把真实账号 authenticated smoke 纳入 readiness 条件分支：提供 `INTEGRATION_USER_A_ACCESS_TOKEN` 或 `INTEGRATION_USER_A_ID/PASSWORD` 时，会验证用户 A `/api/auth/me`、由当前用户 ID 推导出的 `/api/users/:id`、用户笔记/收藏/赞过/关注状态、登录态推荐 feed/热门分类/搜索，并在 feed 返回帖子 ID 时追加验证 `/api/posts/:id`、`/api/posts/:id/comments`、`/api/dislikes?post_id=...`、`/api/reports/check?target_type=post&target_id=...` 只读详情链路；还会验证 `/api/notifications/unread-count`、`/api/notifications`、`/api/notifications/system`、`/api/notifications/system/popup`、`/api/im/conversations`、`/api/im/sync`、首个会话的 `/api/im/conversations/:id/messages` 分页、创作者中心概览/统计/趋势/收益/付费内容/质量奖励，以及余额、本地月币、提现钱包、收款码和订单只读端点；提供用户 B token 或账号密码时，会验证用户 B `/api/auth/me`、用户资料、用户笔记/收藏/赞过和关注状态；同时提供用户 A/B 凭据时，会追加 `user-cross-account-read-smoke`，分别验证 A 读取 B、B 读取 A 的 `/api/users/:id`、笔记、收藏、赞过和关注状态，只执行 GET；提供 `INTEGRATION_ADMIN_ACCESS_TOKEN` 或管理员账号密码时，会验证 `/api/auth/admin/me`、`/api/admin/stats/overview`、`/api/admin/ai-review-status`、`/api/admin/guest-access-status`、`/api/admin/users`、`/api/admin/posts`、`/api/admin/content-review`、`/api/admin/reports` 和 `/api/withdraw/admin/orders`。脚本只报告 token 来源和接口状态，不输出 token/password；临时假用户 A/B token 负向验证会新增 `user-a-authenticated-smoke`、`user-b-authenticated-smoke` 和 `user-cross-account-read-smoke`，按 `/api/auth/me` 及相关只读端点 401 失败；临时假管理员 token 会新增 `admin-authenticated-smoke` 并按 admin me/stats/AI/guest-access、后台只读列表和提现审核订单列表 401 失败，证明这些分支可执行。
- 已完成搜索入口浏览器烟测：`/` 桌面搜索框可输入并提交关键词，页面进入 “Results for travel” 状态且无新增控制台 error；当前本机 `3001` 后端未监听，真实搜索结果和分页仍需数据库后端联调。
- 已完成详情抽屉作者关注入口接入：详情抽屉会从帖子作者解析用户 ID，在存在本地 token 时可选读取 `/api/users/:id/follow-status`，并将桌面/移动端头部关注按钮接入 `/api/users/:id/follow` 的关注/取关动作；未登录时不主动跳转登录，也不阻塞详情打开。
- 已完成详情抽屉关注入口浏览器烟测：`/` fixture feed 可打开第一张帖子详情，抽屉内可见作者区 `Follow` 按钮、标题和评论输入，控制台无 error；当前本机 `3001` 后端未监听，真实关注/取关 mutation 和计数同步仍需可用数据库、Redis 与普通用户测试账号联调。
- 已完成详情抽屉更多操作入口接入：新增 `/api/dislikes`、`/api/reports`、`/api/reports/check` 前端封装；详情抽屉会在存在本地 token 时可选读取不喜欢/举报状态，更多操作面板提供“不感兴趣”切换和帖子举报表单，分享按钮使用系统分享并在不支持时复制当前链接；未登录或后端不可用时不影响详情渲染。
- 已完成详情更多操作浏览器烟测：修正桌面端 Vaul 详情抽屉 open-state transform 后，`/` fixture feed 可打开第一张帖子详情，抽屉真实落在视口内；“更多操作”可展开，面板内可见 `Not interested`、举报原因下拉、补充说明、`Report` 按钮和 `Share` 按钮，控制台无 error。真实不喜欢/举报状态、提交成功、重复举报拦截和后台举报列表联动仍需可用后端、Redis 与普通用户/管理员测试账号联调。
- 已运行并通过 `go test ./internal/http/server` 和 `go test ./...`。
- 已完成新增页面浏览器烟测：`/notifications` 显示通知空态且无控制台 error；`/messages` 显示会话搜索/消息页面未登录态且无控制台 error；`/publish` 创作经营区块和创作者明细分页入口可渲染；`/wallet` 未登录态可渲染。当前本机 `3001` 后端未监听，真实通知/IM/创作者/钱包数据和 WebSocket 收发仍需后端服务与测试账号联调。
- `npm.cmd ci` 因 `package.json` 与 `package-lock.json` 不同步失败；随后使用 `npm.cmd install` 安装依赖并更新 lockfile 后完成验证。
- 已新增并复跑真实联调就绪检查：`node scripts\check-integration-readiness.mjs` 当前返回 `not-ready`，40 项基础检查中 31 pass、3 warn、6 fail；feed fixture 兜底门禁、前端配置后端 health、auth-config 契约、认证公共辅助接口契约、OAuth2 start 302 契约、OAuth2 backend callback 回前端契约、OAuth2 登录 UI/回调静态契约、OAuth2 start URL 静态契约、前端认证 session 静态契约、发布/上传静态契约、内容详情/互动静态契约、个人/用户页静态契约、通知/IM 页面静态契约、创作者/钱包/后台页面静态契约、首页 feed 访问模式、搜索访问模式、游客权限边界、创作者/钱包公开配置契约、IM WebSocket 静态契约、后端 route-matrix 关键路由合同、后端无数据库认证引导契约、前端 API 路由契约审计、`.env.example` 配置契约、环境速查文档契约和 Next rewrite/API client 后端地址契约均通过；`frontend-api-route-audit` 当前记录 82 个前端 API 调用形态且 0 未匹配、0 宽动态匹配，新增个人资料更新 `PUT /api/users/:id` 已覆盖；若提供测试账号或 access token，还会追加 authenticated smoke 检查。当前阻塞点集中在 DB/Redis 主机不可解析、本地后端 health 不可达，以及普通用户 A/B、管理员测试账号环境变量缺失；远端内容访问当前需要登录态。
- 待真实后端和测试账号联调的流程：登录/注册、首页详情、评论、点赞、收藏、用户页、发布图文/视频/播客音频。
- 当前源码静态契约复核结果：认证、上传、用户页、feed/互动、通知、IM、创作者中心、钱包提现和后台基础列表/动作的路径、方法和主要字段已对齐后端 handler；用户详情与用户 tab 接口按后端设计需要登录，未登录访问 `/user/[id]` 会进入前端错误态，后续真实联调时需确认这是否符合产品预期。
- 当前未完成项：登录/注册/OAuth2 真实账号联调、IM 页面级双账号收发与 WebSocket 推送联调、创作者真实数据联调、钱包提现真实账号联调、后台管理真实管理员账号联调；真实后端联调仍需可用数据库、Redis、后端服务与测试账号。
