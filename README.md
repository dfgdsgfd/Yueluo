# Yueluo 月梦社区

Yueluo 是一个面向图文、视频、创作者和运营后台的内容社区项目。项目包含 Next.js 前端、Go/Gin 后端、PostgreSQL/MySQL 数据层、Redis/Asynq 队列、图片/视频上传、隐藏水印、内容审核、消息通知、创作者中心和运营管理能力。

## 在线演示

- 演示站：[https://yueluo.yuemoxi.cn/](https://yueluo.yuemoxi.cn/)
- 后台入口：[https://yueluo.yuemoxi.cn/admin](https://yueluo.yuemoxi.cn/admin)
- 后台账号：`admin`
- 后台密码：`123456`

> 以上账号仅用于演示环境。生产部署必须更换管理员密码、JWT 密钥、数据库密码和所有第三方服务密钥。

## 界面预览

### 运营后台

![运营后台看板](docs/readme/admin-dashboard.png)

### Web 首页信息流

![Web 首页信息流](docs/readme/web-feed.png)

### 创作中心

![创作中心](docs/readme/creator-center.png)

### 帖子详情与视频互动

![帖子详情与视频互动](docs/readme/post-preview.png)

### AI 发布详情生成

![AI 发布详情生成](docs/readme/ai-publish.png)

### 移动端首页

<img src="docs/readme/mobile-feed.png" alt="移动端首页" width="360">

### 移动端消息

<img src="docs/readme/mobile-messages.png" alt="移动端消息" width="360">

## 核心能力

- 内容社区：图文笔记、视频笔记、分类、标签、搜索、评论、点赞、收藏、关注、举报和推荐排序。
- 创作工具：创作者中心、发布入口、草稿、附件、视频上传、AI 标题与详情生成。
- 移动适配：移动端首页、发布、消息、通知、帖子详情和底部导航。
- 商业化：积分、钱包、创作者收益、提现、礼品卡、付费图包和付费内容预览。
- 消息通知：站内信、IM 会话、未读角标、系统通知和弹窗通知。
- 内容安全：AI 审核、人工审核、举报处理、敏感词、访客访问限制和维护模式。
- 媒体处理：图片/视频/附件上传、本地存储、R2/S3 兼容存储、WebP 转换、隐藏水印和受保护图片包。
- 运营后台：用户管理、内容管理、评论管理、分类标签、媒体库、日志中心、队列状态、数据库管理和系统观测。

## 技术栈

| 模块 | 技术 |
| --- | --- |
| 前端 | Next.js 16、React 19、TypeScript、Tailwind CSS、next-intl |
| 后端 | Go、Gin、GORM |
| 数据库 | PostgreSQL / MySQL |
| 缓存与队列 | Redis、Asynq |
| 媒体 | 本地文件、R2/S3 兼容存储、FFmpeg、WebP、隐藏水印服务 |
| 移动端 | Capacitor Android |
| 辅助服务 | FastAPI 隐藏水印服务 |

## 项目结构

| 目录 | 说明 |
| --- | --- |
| `backend-gin/` | Go/Gin 后端，包含 API、GORM 模型、迁移、队列任务、上传、审核、IM、钱包和后台管理。 |
| `front-end-nextjs/` | Next.js 前端应用，包含 Web、移动端、后台、创作者中心和多语言资源。 |
| `blind-watermark-fastapi/` | FastAPI 隐藏水印服务，供后端远程嵌入和提取水印 payload。 |
| `App/Android/` | Capacitor Android Release 工程。 |
| `docs/` | 项目文档与 README 截图资源。 |
| `scripts/` | 源码体量检查、联调 readiness 和工程自动化脚本。 |

## 本地启动

### 1. 启动后端

```bash
cd backend-gin
cp .env.example .env
go mod download
go run ./cmd/api
```

至少需要配置：

```env
DATABASE_URL=postgresql://user:password@localhost:5432/yueluo?schema=public
JWT_SECRET=replace-with-a-long-random-secret
DB_AUTO_MIGRATE=true
GIN_PORT=3001
FRONTEND_URL=http://localhost:5173
CORS_ORIGINS=http://localhost:5173,http://localhost:3001
```

后端默认监听 `http://localhost:3001`。环境变量读取顺序为 `GIN_ENV_FILE` 或 `ENV_FILE` 指定文件、`backend-gin/.env`、根目录 `.env`，进程环境变量优先级最高。

### 2. 填充中文演示数据

```bash
cd backend-gin
go run ./cmd/demo-seed
```

该命令会根据现有 GORM 数据表自动迁移并幂等填充中国校园与城市生活场景的中文演示数据，包括用户、分类、标签、图文/视频帖子、评论、点赞、收藏、关注、通知、钱包/积分、礼品卡、系统公告和 IM 会话。

默认演示用户示例：`demo_alice` 或 `demo-alice@example.test`，密码 `Demo123456!`。

可限制数量或修改密码：

```bash
go run ./cmd/demo-seed --password 'LocalDemo123!' --users 6 --posts 12
```

### 3. 启动前端

```bash
cd front-end-nextjs
cp .env.example .env.local
npm install
npm run dev -- -p 5173
```

本地直连后端时建议设置：

```env
BACKEND_ORIGIN=http://localhost:3001
NEXT_PUBLIC_BACKEND_ORIGIN=http://localhost:3001
```

浏览器访问 `http://localhost:5173`。

### 4. 隐藏水印服务（可选）

当后端配置 `HIDDEN_WATERMARK_ENGINE=remote`，或 `auto` 模式需要远程服务时启动：

```bash
cd blind-watermark-fastapi
export BLIND_WATERMARK_API_KEY="replace-with-internal-secret"
./run.sh
```

后端侧对应配置：

```env
HIDDEN_WATERMARK_REMOTE_URL=http://127.0.0.1:8090
HIDDEN_WATERMARK_REMOTE_API_KEY=replace-with-internal-secret
```

## 管理员账号

后端自动迁移开启时，如果 `admin` 表没有任何管理员，会创建初始账号：

```text
username: admin
password: 123456
```

手动重置管理员密码：

```bash
cd backend-gin
go run ./cmd/api --reset-admin-password --reset-admin-username admin --reset-admin-new-password 'replace-with-new-password'
```

生产环境首次登录后应立即修改管理员密码。

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

## 构建发布

后端 Linux amd64：

```bash
cd backend-gin
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o yuem-go-linux-amd64 ./cmd/api
```

后端 Windows amd64：

```bash
cd backend-gin
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o yuem-go-windows-amd64.exe ./cmd/api
```

前端生产构建：

```bash
cd front-end-nextjs
npm install
npm run build
npm run start
```

## 更多文档

- `backend-gin/.env.example`：后端环境变量全集。
- `front-end-nextjs/.env.example`：前端环境变量和联调开关。
- `blind-watermark-fastapi/README.md`：远程隐藏水印服务。
- `App/Android/README.md`：Android Release 构建与签名。
- `demo_oauth21/README.md`：OAuth2.1 + DPoP 示例。
- `frontend-backend-api-integration-env.md`：前后端联调环境变量速查。
