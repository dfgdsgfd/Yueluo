# Yueluo

Yueluo 是一个内容社区项目，包含 Go/Gin 后端、Next.js 前端、隐藏水印辅助服务、Android 包装工程。根目录 README 作为开发与部署入口；各子系统更细的说明保留在对应目录。

## 项目结构

| 目录 | 说明 |
| --- | --- |
| `backend-gin/` | Go/Gin 后端，GORM，PostgreSQL/MySQL，Redis/Asynq，上传、审核、IM、钱包、后台管理等 API。 |
| `front-end-nextjs/` | Next.js 16、React 19、TypeScript、next-intl 前端应用。 |
| `blind-watermark-fastapi/` | 内部 FastAPI 隐藏水印服务，供 Go 后端远程嵌入/提取水印 payload。 |
| `App/Android/` | Capacitor Android Release 工程。 |
| `scripts/` | 源码体量检查、联调 readiness 等根级工具。 |

## 环境要求

- Go：`backend-gin/go.mod` 声明 `go 1.25.0`，CI 使用 Go 1.26。
- Node.js：前端建议使用 24+；生产环境按项目约定使用 Node.js 25.x。
- 数据库：推荐 PostgreSQL；后端也保留 MySQL driver。
- Redis：队列、缓存、IM、系统日志和限流相关功能会用到 Redis；开发时可按 `.env.example` 关闭队列。

常用工具：`rg`、`fd`、`git`、`bat`、`delta`、`fzf`、`uv`、`nvm`、`gh`、`yq`。

## 快速启动

### 后端

```bash
cd backend-gin
cp .env.example .env
```

至少修改：

```env
DATABASE_URL=postgresql://user:password@localhost:5432/xiaoshiliu?schema=public
JWT_SECRET=replace-with-a-long-random-secret
DB_AUTO_MIGRATE=true
GIN_PORT=3001
FRONTEND_URL=http://localhost:5173
CORS_ORIGINS=http://localhost:5173,http://localhost:3001
```

启动：

```bash
go mod download
go run ./cmd/api
```

后端默认监听 `http://localhost:3001`。环境变量读取顺序为 `GIN_ENV_FILE` 或 `ENV_FILE` 指定文件、`backend-gin/.env`、根目录 `.env`，进程环境变量优先级最高。

### 前端

```bash
cd front-end-nextjs
cp .env.example .env.local
```

本地直连后端时建议设置：

```env
BACKEND_ORIGIN=http://localhost:3001
NEXT_PUBLIC_BACKEND_ORIGIN=http://localhost:3001
```

安装并启动：

```bash
npm install
npm run dev -- -p 5173
```

浏览器访问 `http://localhost:5173`。默认让浏览器请求保持在 Next.js 同源 `/api/*`，再由 Next rewrite 转发到后端；只有需要浏览器直连后端 origin 时，再启用 `NEXT_PUBLIC_API_BASE_URL`。

### 隐藏水印服务

当后端配置 `HIDDEN_WATERMARK_ENGINE=remote`，或 `auto` 模式需要远程服务时启动：

```bash
cd blind-watermark-fastapi
export BLIND_WATERMARK_API_KEY="replace-with-internal-secret"
./run.sh
```

默认监听 `http://127.0.0.1:8090`。后端侧对应配置：

```env
HIDDEN_WATERMARK_REMOTE_URL=http://127.0.0.1:8090
HIDDEN_WATERMARK_REMOTE_API_KEY=replace-with-internal-secret
```

## 管理员账号与密码

- 后端自动迁移开启时，如果 `admin` 表没有任何管理员，会创建初始账号 `admin / 123456`。
- 已有管理员的非空密码不会被启动覆盖；如果存在空密码管理员，会补齐为初始密码的 Argon2id 哈希。
- 管理员创建、后台直接修改密码、重置密码都会写入 Argon2id 哈希。
- 普通用户旧密码如果不是 Argon2id 格式，会在自动迁移阶段替换为随机 16 位密码的 Argon2id 哈希；OAuth2 登录用户不依赖这些旧密码。

手动重置管理员密码应使用 GitHub Release 产物或本地构建得到的后端二进制执行。命令会重置后退出，不会继续启动 HTTP 服务；执行时需提供与正式服务相同的数据库环境变量或 `.env`：

```bash
chmod +x ./yuem-go-linux-amd64
./yuem-go-linux-amd64 --reset-admin-password --reset-admin-username admin --reset-admin-new-password 'replace-with-new-password'
```

Windows：

```powershell
.\yuem-go-windows-amd64.exe --reset-admin-password --reset-admin-username admin --reset-admin-new-password 'replace-with-new-password'
```

不传 `--reset-admin-new-password` 时默认使用 `123456`。生产环境首次登录后应立即在后台修改管理员密码。

## 构建与发布

### 后端二进制

Linux amd64：

```bash
cd backend-gin
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o yuem-go-linux-amd64 ./cmd/api
```

Windows amd64：

```bash
cd backend-gin
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o yuem-go-windows-amd64.exe ./cmd/api
```

仓库也保留了 `backend-gin/build.sh` 和 `backend-gin/build.bat`，用于快速生成 Linux amd64 后端二进制。

### 前端生产构建

```bash
cd front-end-nextjs
npm install
npm run build
npm run start
```

生产启动前请确保 `.env` 或进程环境里的后端地址、私有入口、OAuth2 回调、公开资源域名等配置已经固定。

### GitHub Actions

- `.github/workflows/build.yml`：push/PR 默认构建 Linux amd64；手动 `workflow_dispatch` 会额外构建 Windows amd64 `.exe`。非 PR 运行会创建自动 Release。
- `.github/workflows/release.yml`：tag `v*` 触发 Release 构建；手动 `workflow_dispatch` 会同时构建 Linux 和 Windows。
- 两个 workflow 的后端产物都由 `go build ... ./cmd/api` 编译，`--reset-admin-password` 等启动参数会打包进最终二进制。
- Release 产物包含后端二进制、前端 zip、`swagger.json` 和 `SHA256SUMS`。

## 验证命令

后端：

```bash
cd backend-gin
go test ./...
```

前端：

```bash
cd front-end-nextjs
npx tsc --noEmit
npm run lint
npm run check:contracts
npm run check:messages
npm run test:api-core
```

根目录：

```bash
node scripts/check-source-size-budgets.mjs
node scripts/check-integration-readiness.mjs
```

新增或修改 API 时，需要同步维护后端 route matrix、Swagger 文档和前端公共契约检查。

## 核心能力摘要

- 认证：普通用户登录/注册、管理员登录、访问/刷新 token。
- 内容：帖子、评论、点赞、收藏、关注、举报、不感兴趣、搜索、分类和推荐。
- 上传：图片、视频、附件、分片合并、本地存储、图床和 R2/S3 兼容存储。
- 水印：WebP 转换、图片尺寸限制、隐藏水印、受保护内容下载包动态嵌入追踪 payload。
- IM 与通知：会话、消息分页、WebSocket 刷新、系统通知、弹窗通知和未读数。
- 商业化：创作者中心、钱包、提现、礼品卡、充值配置和后台审核。
- 运维：后台资源管理、系统设置、Redis 内存治理、系统更新、Swagger 和 readiness 检查。

## 关键配置

完整配置请以 `backend-gin/.env.example` 和 `front-end-nextjs/.env.example` 为准。常见生产配置包括：

| 配置 | 说明 |
| --- | --- |
| `DATABASE_URL` | 后端数据库连接，设置后优先于拆分的 `DB_*` 变量。 |
| `JWT_SECRET` | 用户和管理员 token 签名密钥，生产必须使用长随机值。 |
| `DB_AUTO_MIGRATE` | 是否启动时执行 GORM AutoMigrate 和默认数据补齐。 |
| `REDIS_ADDR` / `REDIS_HOST` / `REDIS_PORT` | Redis 连接。 |
| `QUEUE_ENABLED` | 是否启用 Asynq 队列。 |
| `BACKEND_ORIGIN` / `NEXT_PUBLIC_BACKEND_ORIGIN` | Next.js 服务端和浏览器侧 API rewrite 的后端 origin。 |
| `HIDDEN_WATERMARK_SECRET` | 隐藏水印签名密钥；为空时回退到 `FILE_SIGNING_SECRET`。 |
| `HIDDEN_WATERMARK_ENGINE` | 隐藏水印引擎，支持 `auto`、`local`、`remote`。 |
| `ADMIN_ENTRY_PATH` / `BACKEND_API_ENTRY_PATH` | 可选私有后台和 API 文档入口。 |

不要把真实 token、密码、密钥、私有 Release 地址或仍有效的签名 URL 写入仓库。

## 更多文档

- `backend-gin/.env.example`：后端环境变量全集。
- `front-end-nextjs/.env.example`：前端环境变量和联调开关。
- `blind-watermark-fastapi/README.md`：远程隐藏水印服务。
- `App/Android/README.md`：Android Release 构建与签名。
- `demo_oauth21/README.md`：OAuth2.1 + DPoP 示例。
- `frontend-backend-api-integration-env.md`：前后端联调环境变量速查。
