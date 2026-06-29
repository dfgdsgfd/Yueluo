# 前后端 API 联调环境变量速查

本文档记录真实联调时需要配置的 `.env` 变量、读取位置和推荐组合。不要把真实 token、密码或 secret 写入示例文件；需要临时验证时优先使用当前 PowerShell 进程环境变量。

## 读取位置

- 前端 Next.js 运行时会读取 `front-end-nextjs/.env`、`front-end-nextjs/.env.local` 和当前进程环境。
- `scripts/check-integration-readiness.mjs` 会合并 `backend-gin/.env`、根目录 `.env`、`front-end-nextjs/.env`、`front-end-nextjs/.env.local` 和当前进程环境。
- 后端地址以 `BACKEND_ORIGIN`、`NEXT_PUBLIC_BACKEND_ORIGIN`、`NEXT_PUBLIC_API_BASE_URL`、`API_BASE_URL` 为主；数据库和 Redis 以 `backend-gin/.env` 或进程环境为主。

## 本地后端开发

用于前端通过 Next rewrite 访问本机后端：

```env
BACKEND_ORIGIN=http://localhost:3001
NEXT_PUBLIC_BACKEND_ORIGIN=http://localhost:3001
# NEXT_PUBLIC_API_BASE_URL=http://localhost:3001
# API_BASE_URL=http://localhost:3001
# NEXT_ALLOWED_DEV_ORIGINS=cs.yuelk.com,xse.yuelk.com
```

默认建议不启用 `NEXT_PUBLIC_API_BASE_URL`，让浏览器请求保持在 Next.js 同源 `/api/*`，由 `next.config.ts` 转发到后端。只有需要浏览器直连后端 origin，或需要整页跳转到直连后端 OAuth2 start endpoint 时，再启用它。

## 远端后端联调

用于前端和 readiness 直接对远端后端做契约检查：

```env
BACKEND_ORIGIN=https://xse.yuelk.com
NEXT_PUBLIC_BACKEND_ORIGIN=https://xse.yuelk.com
# NEXT_PUBLIC_API_BASE_URL=https://xse.yuelk.com
# API_BASE_URL=https://xse.yuelk.com
```

`BACKEND_ORIGIN` 和 `NEXT_PUBLIC_BACKEND_ORIGIN` 会驱动 Next rewrite 与 readiness 的前端后端 health、auth-config、OAuth2 start/callback、feed/search 和公开配置契约检查。

## 私有页面入口

以下变量配置在 `front-end-nextjs/.env` 或 `front-end-nextjs/.env.local`，只在 Next.js 服务端读取：

```env
ADMIN_ENTRY_PATH=/ops-7f3a
BACKEND_API_ENTRY_PATH=/api-docs-91c2
```

- 两个值必须以 `/` 开头、彼此不同，且不能使用 `/api`、`/_next` 前缀。
- 配置自定义入口后，默认 `/admin`、`/backend-api` 会返回 `404`。
- 后台导航中的“后端 API”链接会自动使用 `BACKEND_API_ENTRY_PATH`。

## 真实账号 smoke

提供短期 token 时：

```env
INTEGRATION_USER_A_ACCESS_TOKEN=
INTEGRATION_USER_B_ACCESS_TOKEN=
INTEGRATION_ADMIN_ACCESS_TOKEN=
```

没有 token、但允许脚本登录时：

```env
INTEGRATION_USER_A_ID=
INTEGRATION_USER_A_PASSWORD=
INTEGRATION_USER_B_ID=
INTEGRATION_USER_B_PASSWORD=
INTEGRATION_ADMIN_USERNAME=
INTEGRATION_ADMIN_PASSWORD=
```

用户 A 用于登录态首页、搜索、通知、IM、创作者、钱包和提现只读 smoke；用户 B 用于跨账号用户页只读 smoke；管理员用于后台统计、开关状态、资源列表和提现审核订单只读 smoke。

## 可写 smoke

默认不要开启可写 smoke。只有在准备好专用测试账号、测试帖子和可恢复数据后再设置：

```env
INTEGRATION_ENABLE_WRITE_SMOKE=true
INTEGRATION_WRITE_SMOKE_POST_ID=
```

开启后会追加以下写入检查：

- `user-withdraw-payment-code-write-smoke`：读取用户 A 收款码；只有已有至少一个收款码 URL 时才临时写入烟测 URL 并恢复。
- `user-draft-post-write-smoke`：用户 A 创建最小草稿帖并删除。
- `user-post-interaction-write-smoke`：用户 A 对专用测试帖执行点赞、收藏、评论创建和清理恢复。
- `user-cross-account-follow-write-smoke`：用户 A/B 临时关注状态翻转并恢复。
- `user-cross-account-im-write-smoke`：用户 A/B 创建或复用 direct 会话，发送唯一 smoke 消息并标记已读。
- `admin-runtime-toggle-write-smoke`：管理员临时翻转 AI 审核和游客访问开关并恢复。

如果缺少凭据，写 smoke 会在写入前显式失败；如果提供无效 token，会先验证 `/api/auth/me` 或 `/api/auth/admin/me`，401 时停止在 mutation 之前。

## 常用命令

```powershell
node scripts\check-integration-readiness.mjs
node scripts\check-integration-readiness-selftest.mjs
npm.cmd run audit:api
npm.cmd run lint
npm.cmd run build
```

`check-integration-readiness-selftest.mjs` 会自动跑一组负向场景：默认不开可写 smoke、fixture fallback 门禁、开启可写 smoke 但无凭据、假用户 token、假管理员 token。它不要求真实后端 ready，只验证 readiness 的安全门禁和报告脱敏行为。

临时假 token 负向验证示例：

```powershell
$env:INTEGRATION_ENABLE_WRITE_SMOKE='true'
$env:INTEGRATION_USER_A_ACCESS_TOKEN='codex-invalid-token'
node scripts\check-integration-readiness.mjs
Remove-Item Env:\INTEGRATION_ENABLE_WRITE_SMOKE -ErrorAction SilentlyContinue
Remove-Item Env:\INTEGRATION_USER_A_ACCESS_TOKEN -ErrorAction SilentlyContinue
```

## 当前已知阻塞

截至 2026-06-09，当前机器 readiness 仍为 `not-ready`：本机 `backend-gin/.env` 中的 PostgreSQL 主机 `1Panel-postgresql-RHin` 和 Redis 主机 `1Panel-redis-Mr0U` 不可解析，`http://localhost:3001/api/health` 不可达，并且未提供普通用户 A/B 与管理员测试账号变量。远端 `https://xse.yuelk.com` 的公开契约检查可达且通过，但真实内容、互动、IM、钱包和后台流程仍需要有效登录态继续联调。
## OAuth2 redirect_uri

Backend OAuth2 start uses the following precedence:

```env
# Full callback URL override. Use this in production when the exact callback is fixed.
OAUTH2_REDIRECT_URI=https://xse.yuelk.com/api/auth/oauth2/callback

# Used when OAUTH2_REDIRECT_URI is empty. Local default is http://localhost:3000.
OAUTH2_REDIRECT_BASE_URL=http://localhost:3000
OAUTH2_CALLBACK_PATH=/api/auth/oauth2/callback
```

For local Next.js debugging, keep `OAUTH2_REDIRECT_URI` empty and use
`OAUTH2_REDIRECT_BASE_URL=http://localhost:3000`. The authorization server will
callback to the Next.js origin first; `next.config.ts` rewrites
`/api/auth/oauth2/callback` to the backend, and the backend then redirects the
browser to `/explore?oauth2_login=success&access_token=...&refresh_token=...`.
