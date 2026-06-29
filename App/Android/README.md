# 月梦-快速版 Android

月梦-快速版的 Android Release 工程。应用包名为 `com.yuelk.xsewebfast`，使用 Capacitor 加载远程站点。OAuth/Google 登录在系统浏览器中完成，再通过 `xsewebfast://auth-return` 携带一次性 ticket 返回应用；Android 原生层不保存登录状态。

## 环境变量

复制 `.env.example` 为 `.env.local` 后按部署环境修改。服务地址优先级为 `CAP_SERVER_URL` → `NEXT_PUBLIC_YUEM_MOBILE_SERVER_URL` → `NEXT_PUBLIC_API_BASE_URL` → `https://xse.yuelk.com`。生产至少配置：

```env
CAP_SERVER_URL=https://xse.yuelk.com
NEXT_PUBLIC_YUEM_MOBILE_SERVER_URL=https://xse.yuelk.com
NEXT_PUBLIC_API_BASE_URL=https://xse.yuelk.com
NEXT_PUBLIC_YUEM_MOBILE_CALLBACK_SCHEME=xsewebfast
NEXT_PUBLIC_YUEM_MOBILE_APP_VERSION=1.0.0
NEXT_PUBLIC_YUEM_IN_APP_HOSTS=xse.yuelk.com,cs.yuelk.com
NEXT_PUBLIC_YUEM_AUTH_BROWSER_HOSTS=user.yuelk.com
```

## 构建 Release

需要 Node.js 25、Docker 和可用的 Docker Engine。在项目目录执行：

```bash
npm ci
npm run build:release
```

构建会同步 Capacitor、运行 Android Release Lint、启用 R8/资源压缩、默认只打包 Android arm64-v8a（ARMv8）原生库，生成已签名 APK/AAB，并用 `apksigner`、Bundletool 和 `jarsigner` 校验。产物位于 `dist/`：

- `月梦-快速版-1.0.0-release.apk`：直接安装测试
- `月梦-快速版-1.0.0-release.aab`：应用商店上传包
- `SHA256SUMS`：产物校验值

只重新校验现有产物可执行：

```bash
npm run verify:release
```

## 发布签名

本机发布密钥保存在 `.release-secrets/yuem-release.p12`，Gradle 配置保存在 `.release-secrets/keystore.properties`。整个目录已被 Git 忽略，必须另行做加密备份；以后更新同一个 Android 应用必须继续使用这把密钥。

如确实要创建一套新签名，先安全备份旧密钥，再执行：

```bash
npm run signing:generate
```

签名证书 SHA-256 指纹为：

```text
F5:D0:6B:43:DD:37:18:AA:25:A5:A2:8F:F1:1B:BE:C9:0D:10:49:9D:41:99:9B:D4:8D:52:91:AD:18:E7:28:63
```

## 上线前置项

1. 部署本仓库的 `/api/auth/oauth2/mobile-session` 接口和 Next.js ticket bridge 改动。
2. 后端配置 `OAUTH2_APP_CALLBACK_URL=xsewebfast://auth-return`；如需多 App，优先在后台“系统设置 → 登录 / OAuth”维护 `oauth2_app_callback_urls`（一行一个完整 callback，例如 `xsewebfast://auth-return`、`yuempro://auth-return`），也可用 `OAUTH2_APP_CALLBACK_URLS` 做初始默认值；用户中心允许的 Web OAuth 回调仍是 `https://xse.yuelk.com/api/auth/oauth2/callback`。
3. 自定义 scheme 不需要 `assetlinks.json`；仅在另行启用 HTTPS Android App Links 时才需要配置它。
4. Google/OAuth 服务端允许用户中心 `https://user.yuelk.com` 当前使用的 Web 回调地址；App 本身不需要 `google-services.json`，因为登录走系统浏览器和服务端交换。
5. 上架新版本时同时递增 `android/app/build.gradle` 中的 `versionCode`，并更新 `versionName`、`package.json` 版本及发布脚本中的产物版本名。

OAuth 回调只把两分钟有效、一次性消费的 ticket 带回应用；访问令牌不会出现在回调 URL。WebView 使用 ticket 向后端换取正常 Cookie 与兼容 token payload，重复兑换会返回 401。
