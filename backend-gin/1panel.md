# 1panel部署.MD

This file tells automation agents how to deploy this project to 1Panel, upload the built artifact, and restart the matching runtime or website. Before deploying, read this file and any nearby project `.md` files for the real panel address, API key, and runtime name.

## Fill These Values

```text
PANEL_BASE_URL=https://192.168.80.101:10010/xnet123
PANEL_API_KEY=pMQz7qrbvmvlsINa8Hv1k8sXCiQTDWZx
RUNTIME_NAME=yuem-go-gin
WEBSITE_NAME=
REMOTE_APP_DIR=/mnt/DISK_20TB/go-gin
REMOTE_DEPLOY_TMP=/tmp/agent-deploy
```

- `PANEL_BASE_URL`: 1Panel URL, for example `https://example.com:8443`. Do not end with `/`.
- `PANEL_API_KEY`: API key from 1Panel settings. Never print or commit it.
- `RUNTIME_NAME`: exact 1Panel runtime name to restart after deployment.
- `WEBSITE_NAME`: optional website primary domain or alias.
- `REMOTE_APP_DIR`: remote project directory. Prefer to infer it from runtime `codeDir`, runtime `workDir`, or website `sitePath`. If it cannot be inferred, stop and ask for it.
- `REMOTE_DEPLOY_TMP`: temporary upload directory on the server.

## 1Panel API Auth

Use the current 1Panel Swagger page to confirm details before calling endpoints:

```text
{PANEL_BASE_URL}/1panel/swagger/index.html
```

For 1Panel v2 API requests:

```text
timestamp = current Unix timestamp in seconds
token = md5("1panel" + PANEL_API_KEY + timestamp)
```

Send these headers:

```text
1Panel-Token: {token}
1Panel-Timestamp: {timestamp}
Content-Type: application/json
```

The panel commonly uses a self-signed certificate. Ignore TLS verification by default when calling the panel API:

- `curl`: use `-k`.
- Node.js: set an HTTPS agent with `rejectUnauthorized: false`.
- Python: use `verify=False`.
- Go: set `InsecureSkipVerify: true`.

Common v2 paths, to be checked against Swagger for the target panel version:

```text
POST /api/v2/runtimes/search
GET  /api/v2/runtimes/{id}
POST /api/v2/runtimes/operate
POST /api/v2/websites/search
GET  /api/v2/websites/{id}
POST /api/v2/websites/operate
POST /api/v2/files
POST /api/v2/files/upload
POST /api/v2/files/decompress
POST /api/v2/files/del
```

## Deploy Rules

1. Build locally first, then upload only the files needed for production.
2. Do not upload `.git`, dependency caches, test output, or local virtual environments.
3. Preserve remote runtime data such as `.env`, `logs/`, `uploads/`, `storage/`, and `data/` unless the user explicitly asks for a full overwrite.
4. Upload to `REMOTE_DEPLOY_TMP`, decompress into a temporary release directory, verify key files, then replace or sync into `REMOTE_APP_DIR`.
5. After upload, restart the runtime matching `RUNTIME_NAME`. If no runtime is configured but `WEBSITE_NAME` is configured, restart the website.
6. Do not change panel SSL settings. Only ignore TLS verification for API calls to the panel.

## Project Detection

Detect the project type in this order. If several match, choose the most specific one.

### Next.js

Detect:

```text
next.config.js
next.config.mjs
next.config.ts
package.json with dependency "next"
```

Install:

```text
pnpm install --frozen-lockfile
yarn install --frozen-lockfile
npm ci
```

Build:

```text
pnpm build
yarn build
npm run build
```

If the project uses static export, upload `out/` contents. Otherwise upload:

```text
.next/
public/
package.json
pnpm-lock.yaml
yarn.lock
package-lock.json
next.config.js
next.config.mjs
next.config.ts
```

Expected runtime start command is usually one of:

```text
pnpm start
yarn start
npm start
```

The `start` script should run `next start` or the project's custom server.

### Node.js Frontend Or Service

Detect:

```text
package.json
```

Package manager:

```text
pnpm-lock.yaml -> pnpm
yarn.lock -> yarn
package-lock.json -> npm
npm-shrinkwrap.json -> npm
otherwise -> npm
```

Install and build:

```text
pnpm install --frozen-lockfile
pnpm build

yarn install --frozen-lockfile
yarn build

npm ci
npm run build
```

If `package.json` has no `build` script but has a `start` script, skip build and upload runtime source. If it has neither `build` nor `start`, stop and report the missing script.

For static frontend projects, upload the first existing directory from:

```text
dist/
build/
out/
```

For Node services, upload:

```text
package.json
pnpm-lock.yaml
yarn.lock
package-lock.json
dist/
build/
public/
server.js
app.js
index.js
```

Do not upload:

```text
node_modules/
.git/
.cache/
coverage/
```

### Go

Detect:

```text
go.mod
```

Build for Linux:

```powershell
go mod download
$env:GOOS="linux"
$env:GOARCH="amd64"
go build -trimpath -ldflags="-s -w" -o deploy/app .
```

If the main package is not at the repository root, use `go list ./...` to find the `package main` directory and build that package.

Upload:

```text
deploy/app
public/
static/
templates/
config/
```

Typical runtime start command:

```text
./app
```

### Python

Detect:

```text
requirements.txt
pyproject.toml
Pipfile
main.py
app.py
manage.py
```

Upload:

```text
*.py
requirements.txt
pyproject.toml
poetry.lock
Pipfile
Pipfile.lock
app/
src/
static/
templates/
alembic/
migrations/
```

Do not upload:

```text
.venv/
venv/
__pycache__/
.pytest_cache/
```

Common start commands:

```text
gunicorn app:app -b 0.0.0.0:$PORT
uvicorn app:app --host 0.0.0.0 --port $PORT
python main.py
python manage.py runserver 0.0.0.0:$PORT
```

If the 1Panel runtime already has a start command, do not change it. Upload and restart only.

## Automation Flow

1. Read config values from this file, environment variables, or nearby project `.md` files.
2. Generate the 1Panel auth headers for every request.
3. Search runtime with `POST /api/v2/runtimes/search` using `RUNTIME_NAME`.
4. Exact-match the runtime by `name`; save `id`, `type`, `codeDir`, `workDir`, and `status`.
5. If no runtime is found and `WEBSITE_NAME` is set, search websites with `POST /api/v2/websites/search`.
6. Exact-match website by `primaryDomain` or `alias`; save `id` and `sitePath`.
7. Infer `REMOTE_APP_DIR` from runtime `codeDir`, runtime `workDir`, or website `sitePath`.
8. Detect project type and build locally.
9. Create a fresh archive such as `deploy-package.tar.gz` or `deploy-package.zip`.
10. Ensure `REMOTE_DEPLOY_TMP` exists via the files API.
11. Upload the archive to `REMOTE_DEPLOY_TMP`.
12. Decompress the archive into a new temporary release directory.
13. Verify required files exist in the release directory.
14. Sync release contents into `REMOTE_APP_DIR` while preserving runtime data.
15. Restart runtime with `POST /api/v2/runtimes/operate` and body `{"ID": runtime_id, "operate": "restart"}`.
16. If restarting a website instead, call `POST /api/v2/websites/operate` with body `{"id": website_id, "operate": "restart"}`.
17. Query status and perform an HTTP health check if a domain is available.

If the target 1Panel version does not expose a directory sync or overwrite API, stop after upload/decompress and report that Swagger must be used to choose the correct file move/copy endpoint. Do not guess a destructive file operation.

## PowerShell Token Example

```powershell
$ts = [int][double]::Parse((Get-Date -UFormat %s))
$raw = "1panel$env:PANEL_API_KEY$ts"
$md5 = [System.Security.Cryptography.MD5]::Create()
$token = -join ($md5.ComputeHash([Text.Encoding]::UTF8.GetBytes($raw)) | ForEach-Object { $_.ToString("x2") })
```

Search runtime:

```powershell
curl.exe -k -X POST "$env:PANEL_BASE_URL/api/v2/runtimes/search" `
  -H "Content-Type: application/json" `
  -H "1Panel-Token: $token" `
  -H "1Panel-Timestamp: $ts" `
  -d "{\"page\":1,\"pageSize\":100,\"name\":\"$env:RUNTIME_NAME\"}"
```

Restart runtime:

```powershell
curl.exe -k -X POST "$env:PANEL_BASE_URL/api/v2/runtimes/operate" `
  -H "Content-Type: application/json" `
  -H "1Panel-Token: $token" `
  -H "1Panel-Timestamp: $ts" `
  -d "{\"ID\":123,\"operate\":\"restart\"}"
```

## Verification

After deployment, do at least one verification step:

```text
GET /api/v2/runtimes/{id}
GET /api/v2/websites/{id}
curl -k https://target-domain/
curl -k http://target-domain/
```

If verification fails, read runtime or website logs and report the cause. Do not restart more than twice.

## Failure Handling

- `401`: check API key, timestamp, server clock, and panel API whitelist.
- `404`: check whether the panel uses `/api/v2` or `/api/v1`; confirm in Swagger.
- Upload failure: check directory permissions, archive size, and panel upload limits.
- Decompress failure: check archive format and API `type` value.
- Restart failure: confirm exact runtime or website name.
- Bad site after deployment: roll back to the previous release or restore backup, then inspect logs.

## Security

- Never print the full API key.
- Never commit API keys, `.env` files, panel logs, or deployment archives.
- Never delete remote runtime data directories.
- Never overwrite remote `.env` unless explicitly requested.
- Ignore TLS verification for panel API calls, but do not modify panel SSL configuration.
