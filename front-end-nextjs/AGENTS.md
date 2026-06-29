<!-- BEGIN:nextjs-agent-rules -->
# This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in `node_modules/next/dist/docs/` before writing any code. Heed deprecation notices.
<!-- END:nextjs-agent-rules -->

## Frontend maintainability rules

- Hand-written production TS/TSX should normally stay below about `30KB / 600 lines`; `40KB / 800 lines` is the hard limit. Split before adding more feature logic when either hard limit is reached.
- Responsibility is the primary boundary: split earlier when a file mixes controller state, network orchestration, formatters, desktop/mobile views, dialogs, or unrelated leaf components. Do not mechanically divide by line number merely to satisfy the checker.
- Preserve public compatibility: route URLs, public entry files, exported names, component props, API behavior, i18n keys, storage/cookie keys, environment variables, request timing, and optimistic-update semantics must remain unchanged during a structural split.
- Keep public entries as thin compatibility facades. Move controller hooks, pure state/formatting logic, views and leaf UI into colocated feature modules; internal modules import from the concrete owner rather than from `@/lib/api`, `@/lib/types`, or another public barrel when a concrete module exists.
- Avoid circular feature imports. Shared types must use type-only imports and live in the smallest common ownership module; do not solve cycles by moving unrelated code back into a large root file.
- A large Client Component should normally become `entry component → controller hook → view/leaf components`. Preserve the original `"use client"` boundary unless the task explicitly includes a reviewed Server/Client boundary change.
- Before changing a Client/Server Component boundary, read the relevant local Next 16 docs in `node_modules/next/dist/docs/` and preserve required directives and serialization constraints.
- Generated assets, lockfiles and test fixtures are excluded from production-source budgets. Any hand-written exception must be explicit and documented; never evade the limit through minification, renamed extensions, string-loaded source, or a higher threshold.
- After a split, run `node ../scripts/check-source-size-budgets.mjs`, `npm run check:contracts`, `npx tsc --noEmit`, `npm run lint`, and relevant tests. Run `npm run build` when CSS, i18n, route entries, providers, or component boundaries changed.

# Next.js App Router 性能避坑指南

> **适用版本：** Next.js 16.2.9  
> **资料依据：** Next.js 官网文档、Next.js 16.2 官方发布说明、Vercel/Next.js 官方仓库发布页  
> **核心原则：** 此处仅作为建议,无需完全遵循, 服务端渲染优先，客户端交互精简；追求“点击秒反馈”，而不是所有内容都必须“秒完成”。

---

## 官网版本与性能特性速览

根据 Vercel/Next.js 官方仓库的 [`v16.2.9` 发布页](https://github.com/vercel/next.js/releases/tag/v16.2.9)，该版本是为了确保 `next@latest` 指向稳定版本而发布的空发布。因此，性能能力应主要参考：
- [Next.js 16.2 发布说明](https://nextjs.org/blog/next-16-2)

本地校验：

```bash
npm view next version
# 16.2.9

npm view create-next-app version
# 16.2.9
```

### Next.js 16/16.2 重点性能能力

官方资料中和性能最相关的能力可以归纳为：

- **Turbopack 默认启用：** 从 Next.js 16 开始，`next dev` 和 `next build` 默认使用稳定版 Turbopack。
- **开发启动更快：** Next.js 16.2 官方发布说明中提到，默认应用的 `next dev` 启动到本地 URL 可用的时间，相比 16.1 有明显提升。
- **服务端渲染更快：** 16.2 优化了 Server Components payload 反序列化，官方说明中给出的真实应用渲染到 HTML 提升区间约为 25% 到 60%。
- **Turbopack Server Fast Refresh：** 16.2 改进了服务端热刷新，只重新加载真正变化的模块，减少开发时无效刷新。
- **动态导入 Tree Shaking：** Turbopack 16.2 可以对解构形式的 `dynamic import()` 移除未使用导出，减少客户端包体积。
- **App Router 流式渲染：** Server Component、`loading.tsx`、`<Suspense>` 和路由预取缓存共同改善首屏和导航体验。
- **路由预取更细粒度：** Next.js 16 对预取做了布局去重和增量预取，多个页面共享布局时不会重复下载同一份布局数据。
- **Cache Components / `use cache`：** Next.js 16 引入新的缓存模型，启用 `cacheComponents: true` 后，可以把可缓存的数据和 UI 纳入静态 shell。
- **图片 API 变化：** Next.js 16 起，`next/image` 的 `priority` 已被 `preload` 取代；确定的 LCP / hero 图可以使用 `preload`，多数场景优先考虑 `loading="eager"` 或 `fetchPriority="high"`。
- **React Compiler 支持稳定：** Next.js 16 支持稳定版 React Compiler，可自动减少不必要的重新渲染，但需要评估构建耗时后再启用。
- **性能指标口径变化：** Next.js 16 的 `next build` 输出移除了 `size` 和 `First Load JS` 指标，官方建议用 Lighthouse 或 Vercel Analytics 评估真实路由性能。

---

## 0. 当前架构建议：Next.js + Go-Gin

如果后端已经是 Go-Gin，推荐把系统边界划清楚：

```text
Browser
  -> Next.js App Router
      -> Server Component / Server Action / Route Handler
          -> Go-Gin API
              -> DB / Redis / MQ / Object Storage
```

### 推荐分工

- **Next.js：** 页面路由、SSR、RSC、首屏数据聚合、SEO、图片优化、用户交互壳。
- **Go-Gin：** 业务 API、鉴权校验、数据库事务、领域逻辑、任务队列、文件处理、权限判断。
- **Route Handler：** 只做 BFF 适配，例如聚合多个 Go 接口、隐藏内部 token、处理上传、适配第三方回调。
- **Proxy：** 只做轻量网络边界逻辑，不要在 Proxy 中请求 Go-Gin 做复杂业务判断。

### 不推荐的做法

- 不要把 Go-Gin 已经实现的业务逻辑再复制一份到 Next.js。
- 不要让所有客户端请求都先经过 Next.js Route Handler 再原样转发到 Go-Gin，这会增加一层无意义延迟。
- 不要为了“前端秒响应”把所有数据都搬到客户端 `useEffect` 里请求。
- 不要把 Go-Gin 内网地址写成 `NEXT_PUBLIC_*`，否则会暴露给浏览器。


只有确实需要暴露给浏览器的变量才使用 `NEXT_PUBLIC_*`。

---

## 1. 组件拆分：Client vs Server

### 默认先写 Server Component

在 App Router 中，组件默认是 Server Component。只有确实需要浏览器端能力时，才添加 `'use client'`。

常见需要 Client Component 的场景：

- `useState`
- `useEffect`
- `useRef`
- `onClick`、`onChange` 等浏览器交互事件
- 依赖 `window`、`document`、`localStorage`
- 第三方纯客户端组件库

### `'use client'` 要尽量下沉

不要把 `'use client'` 放在整个页面或大组件顶部。应尽量下沉到具体的交互组件中，例如按钮、输入框、筛选器、弹窗、Tab 切换等。

```tsx
// 推荐：页面仍然是 Server Component
import LikeButton from './LikeButton'

export default async function Page() {
  const data = await getData()

  return (
    <main>
      <h1>{data.title}</h1>
      <LikeButton />
    </main>
  )
}
```

```tsx
// LikeButton.tsx
'use client'

import { useState } from 'react'

export default function LikeButton() {
  const [liked, setLiked] = useState(false)

  return (
    <button onClick={() => setLiked(!liked)}>
      {liked ? '已喜欢' : '喜欢'}
    </button>
  )
}
```

### 不要把 `layout.tsx` 变成 Client Component

一般情况下，不要在 `layout.tsx` 顶部使用 `'use client'`。这会让整个布局树进入客户端渲染边界，增加客户端 JS 体积，也会削弱 Server Component 的收益。

如果布局中只有某个导航、主题切换、用户菜单需要交互，应把这些交互部分单独拆成 Client Component。

---

## 2. 导航体验：秒反馈，不假死

### 关键目标

Next.js 应用不一定要让所有数据都在 1 秒内完成加载，但用户点击后应该马上看到反馈。

更准确的性能目标是：

- 点击后立即出现 loading、skeleton 或 pending 状态
- 首屏内容尽快可见
- 慢数据局部加载，不阻塞整个页面
- 页面跳转过程中不要让用户误以为“卡死了”

### 合理使用 `loading.tsx`

有异步数据、动态路由、耗时渲染的路由，建议添加 `loading.tsx`，用于提供即时导航反馈。

```tsx
// app/dashboard/loading.tsx
export default function Loading() {
  return <div>加载中...</div>
}
```

注意：不是所有目录都必须机械地添加 `loading.tsx`。纯静态、极轻量、几乎瞬开的页面可以按需省略。

### 利用 `<Link>` 预取改善跳转

App Router 中的 `<Link>` 会在合适的情况下预取路由资源。静态路由通常可以完整预取，动态路由会结合 `loading.tsx` 预取到最近的加载边界，让用户点击后更快看到页面反馈。

```tsx
import Link from 'next/link'

export default function Nav() {
  return <Link href="/dashboard">Dashboard</Link>
}
```

如果某些链接数量巨大、变化频繁，或者预取会带来不必要请求，可以按场景关闭预取：

```tsx
<Link href="/reports" prefetch={false}>
  Reports
</Link>
```

### 使用 `<Suspense>` 做局部流式加载

对于耗时的数据组件，不要让它阻塞整个页面。可以使用 `<Suspense>` 包裹慢组件，让页面主体先展示，慢内容随后补上。

```tsx
import { Suspense } from 'react'
import SlowChart from './SlowChart'

export default function Page() {
  return (
    <main>
      <h1>数据看板</h1>

      <Suspense fallback={<div>图表加载中...</div>}>
        <SlowChart />
      </Suspense>
    </main>
  )
}
```

### 静态页面声明：区分新旧缓存模型

如果项目未启用 Cache Components，且页面确定不依赖请求态、不依赖动态数据，可以显式声明为静态页面：

```ts
export const dynamic = 'force-static'
```

如果项目启用了 Next.js 16 的 `cacheComponents: true`，`dynamic`、`revalidate`、`fetchCache` 等旧 Route Segment Config 不再是主路径，应改用 `use cache`、`cacheLife`、`cacheTag` 和 `<Suspense>` 组合表达缓存与流式加载。

### Next.js 16 的 Cache Components

Next.js 16 引入了 Cache Components 和 `use cache`。它的思路是：默认把页面按请求动态渲染，但允许你把可缓存的数据或组件显式纳入静态 shell。

先在 `next.config.ts` 中启用：

```ts
// next.config.ts
import type { NextConfig } from 'next'

const nextConfig: NextConfig = {
  cacheComponents: true,
}

export default nextConfig
```

适合缓存的内容：

- CMS 文章
- 商品分类
- 导航配置
- 不随单个用户请求变化的数据

不适合缓存的内容：

- 当前登录用户信息
- 购物车实时状态
- 请求头、Cookie、权限强相关内容

示例：

```tsx
import { cacheLife } from 'next/cache'

async function ProductList() {
  'use cache'
  cacheLife('hours')

  const products = await getProducts()
  return <ProductGrid products={products} />
}
```

---

## 3. 数据获取：Next.js 调 Go-Gin

### 推荐：Server Component 直连内网 Go-Gin

首屏数据优先在 Server Component 中获取，再由服务端通过内网调用 Go-Gin。这样可以减少浏览器端 JavaScript，避免暴露后端内网地址，也能把 Cookie、Header、Token 处理留在服务端。

```tsx
// app/products/page.tsx
export default async function Page() {
  const products = await getProducts()

  return <ProductGrid products={products} />
}
```

不推荐为了取首屏数据而滥用客户端 `useEffect`：

```tsx
// 不推荐：首屏数据依赖客户端 useEffect
'use client'

import { useEffect, useState } from 'react'

export default function Page() {
  const [data, setData] = useState(null)

  useEffect(() => {
    fetch('/api/products')
      .then((res) => res.json())
      .then(setData)
  }, [])

  return <div>{data ? data.title : '加载中...'}</div>
}
```

### 封装统一的 Go-Gin 请求函数

建议在 Next.js 服务端封装一个内部请求函数，统一处理 base URL、超时、错误、缓存和鉴权头。

```ts
// lib/go-api.ts
const GO_API_INTERNAL_URL = process.env.GO_API_INTERNAL_URL

if (!GO_API_INTERNAL_URL) {
  throw new Error('Missing GO_API_INTERNAL_URL')
}

type GoFetchOptions = RequestInit & {
  timeoutMs?: number
}

export async function goFetch<T>(
  path: string,
  options: GoFetchOptions = {},
): Promise<T> {
  const { timeoutMs = 5000, headers, ...init } = options
  const controller = new AbortController()
  const timeout = setTimeout(() => controller.abort(), timeoutMs)

  try {
    const res = await fetch(`${GO_API_INTERNAL_URL}${path}`, {
      ...init,
      headers: {
        Accept: 'application/json',
        ...headers,
      },
      signal: controller.signal,
    })

    if (!res.ok) {
      throw new Error(`Go API ${path} failed: ${res.status}`)
    }

    return (await res.json()) as T
  } finally {
    clearTimeout(timeout)
  }
}
```

业务函数只关心接口语义：

```ts
type Product = {
  id: string
  name: string
  price: number
}

export function getProducts() {
  return goFetch<Product[]>('/v1/products', {
    next: { revalidate: 60 },
  })
}
```

### 不要连续串行 `await`

如果多个 Go-Gin 接口之间没有依赖关系，不要连续写多个 `await`，否则总耗时会叠加。

```ts
// 不推荐：请求串行执行
const profile = await getProfile()
const orders = await getOrders()
const notices = await getNotices()
```

```ts
// 推荐：请求并行执行
const [profile, orders, notices] = await Promise.all([
  getProfile(),
  getOrders(),
  getNotices(),
])
```

### Route Handler 只做必要 BFF

如果浏览器必须发起请求，可以用 Next.js Route Handler 作为 BFF 层。但它应该只做必要适配，不要把所有 Go-Gin 接口都机械代理一遍。

适合放在 Route Handler 的场景：

- 隐藏 Go-Gin 内网地址
- 组合多个 Go-Gin 接口
- 把 Cookie / Session 转成后端需要的 Authorization
- 处理文件上传、回调、Webhook
- 给客户端提供更窄的响应字段

```ts
// app/api/me/route.ts
import { cookies } from 'next/headers'
import { NextResponse } from 'next/server'
import { goFetch } from '@/lib/go-api'

export async function GET() {
  const token = (await cookies()).get('token')?.value

  const user = await goFetch('/v1/me', {
    headers: {
      Authorization: `Bearer ${token}`,
    },
    cache: 'no-store',
  })

  return NextResponse.json(user)
}
```

### 缓存责任要分层

Next.js 和 Go-Gin 都可以做缓存，但不要让缓存责任混乱。

建议：

- **Next.js 缓存 UI 和页面数据：** CMS、商品列表、配置、公开内容。
- **Go-Gin 缓存业务查询结果：** Redis、内存缓存、数据库查询结果、权限无关的基础数据。
- **浏览器缓存静态资源：** 图片、字体、脚本、公开 CDN 资源。
- **用户态数据默认不缓存：** 个人信息、订单、账户余额、后台权限、购物车。

在服务端请求 HTTP 数据时，优先使用 Next.js 原生增强的 `fetch`。Next.js 可以对相同 URL 和选项的 GET 请求做渲染期间 memoization，避免同一次服务端渲染里重复请求。

注意：Next.js 16 中，`fetch` 请求默认不持久缓存。需要持久缓存时，应显式使用 `use cache`，或按旧缓存模型使用 `cache: 'force-cache'` / `next.revalidate`。

```ts
const res = await fetch('https://example.com/api/posts', {
  next: { revalidate: 60 },
})
```

如果数据必须每次请求都获取最新值：

```ts
const res = await fetch('https://example.com/api/account', {
  cache: 'no-store',
})
```

### Go-Gin 接口要配合 SSR

Go-Gin 后端最好为 Next.js SSR 准备稳定、粗粒度的页面接口，避免一个页面首屏要拼十几个细碎接口。

建议：

- 为页面首屏提供聚合 API，例如 `/v1/page/dashboard`
- 接口响应字段按页面需要裁剪，不要把大对象完整返回
- 慢接口设置超时、降级和默认值
- 静态或半静态数据返回合理的 `Cache-Control`
- 列表接口必须分页，避免 SSR 一次拉取过多数据
- 错误响应使用统一 JSON 结构，方便 Next.js 做兜底 UI

---

## 4. 资源与包体积优化

### 图片使用 `next/image`

页面图片优先使用 `next/image`。如果图片是响应式布局，应提供合适的 `sizes`，避免浏览器下载过大的图片。

```tsx
import Image from 'next/image'

export default function Hero() {
  return (
    <Image
      src="/hero.jpg"
      alt="产品首页截图"
      width={1200}
      height={720}
      sizes="(max-width: 768px) 100vw, 1200px"
      preload
    />
  )
}
```

建议：

- Next.js 16.2.9 中，确定的 LCP / hero 图片可以使用 `preload`
- 多个首屏候选图、响应式主题图或不确定 LCP 时，优先考虑 `loading="eager"` 或 `fetchPriority="high"`
- 响应式图片提供 `sizes`
- 装饰性图片不要影响核心内容加载
- 不要用普通 `<img>` 替代可优化的业务图片

### 第三方库按需引入

避免全量引入大型工具库。

```ts
// 不推荐
import { map } from 'lodash'
```

```ts
// 推荐
import map from 'lodash/map'
```

更好的做法是：如果只需要少量能力，优先考虑原生 JavaScript API。

### 使用 `optimizePackageImports`

Next.js 支持通过 `optimizePackageImports` 优化部分包的导入方式，减少因为 barrel file 或全量入口导致的额外编译和打包成本。

注意：官网将该能力标记为 experimental，不建议在生产项目中未经测试直接启用。

```ts
// next.config.ts
import type { NextConfig } from 'next'

const nextConfig: NextConfig = {
  experimental: {
    optimizePackageImports: ['lodash', 'lucide-react'],
  },
}

export default nextConfig
```

是否启用要结合项目依赖测试，不建议盲目把所有包都塞进去。

另外，Next.js 已经默认优化了一批常见库，例如 `lucide-react`、`date-fns`、`lodash-es`、`antd`、`@mui/material`、`recharts`、`react-icons/*` 等。对于这些库，一般不需要重复配置。

### 大型客户端库动态加载

图表、富文本编辑器、地图、代码编辑器等大型客户端库，不要默认打进首屏包。可以用 `next/dynamic` 懒加载。

```tsx
import dynamic from 'next/dynamic'

const Chart = dynamic(() => import('./Chart'), {
  loading: () => <div>图表加载中...</div>,
  ssr: false,
})

export default function Page() {
  return <Chart />
}
```

注意：`ssr: false` 只适合强依赖浏览器环境的组件，不要为了省事滥用。

---

## 5. Proxy 使用克制

Next.js 16 中，`middleware.ts` 被重命名为 `proxy.ts`。官方仍然兼容旧文件名，但新项目建议使用 `proxy.ts`。

Proxy 会运行在请求链路中，逻辑过重会直接影响响应速度。在 Next.js + Go-Gin 架构里，不建议每次页面访问都在 Proxy 中请求 Go-Gin 做完整鉴权。

建议 Proxy 只做轻量工作：

- 判断 Cookie / Session 是否存在
- 简单重定向
- A/B 标识注入
- 国际化路径处理

不建议在 Proxy 中做：

- 大量数据库查询
- 复杂权限计算
- 大体积依赖加载
- 慢接口请求

完整权限校验应尽量放到 Go-Gin API、Server Component 数据请求、Route Handler 或 Server Action 中处理。

简单示例：

```ts
// proxy.ts
import { NextResponse, type NextRequest } from 'next/server'

export function proxy(request: NextRequest) {
  const token = request.cookies.get('token')?.value

  if (!token && request.nextUrl.pathname.startsWith('/dashboard')) {
    return NextResponse.redirect(new URL('/login', request.url))
  }

  return NextResponse.next()
}
```

---

## 6. Go-Gin 后端配合清单

### API 形态配合页面

Go-Gin 不只是“提供接口”，还要配合 Next.js SSR 的页面形态。

建议：

- 页面首屏尽量有聚合接口，减少 Next.js 同时请求过多 Go 接口。
- 列表接口必须分页，并支持 `limit` 上限。
- 响应字段按页面裁剪，不要返回大而全的对象。
- 慢数据拆成独立接口，让 Next.js 用 `<Suspense>` 局部加载。
- 统一错误格式，例如 `{ "code": "...", "message": "...", "requestId": "..." }`。

### HTTP 与缓存

Go-Gin 应该给 Next.js 明确的缓存信号。

建议：

- 公开且短期稳定的数据返回 `Cache-Control`。
- 用户态数据返回 `Cache-Control: private, no-store`。
- 图片、导出文件、静态资源放对象存储或 CDN，不要都从 Gin 进程直出。
- 大响应启用 gzip / brotli，或在网关层处理压缩。
- 给每个响应带 `X-Request-Id`，方便前后端串日志。

### 超时和连接复用

Next.js 服务端调用 Go-Gin 时，慢接口会直接拖慢 SSR。

建议：

- Go-Gin 服务端设置 read / write / idle timeout。
- Next.js 调 Go-Gin 设置 `AbortController` 超时。
- Go-Gin 到数据库、Redis、第三方服务也要设置超时。
- 使用连接池，不要每个请求新建数据库连接。
- 慢查询要打日志并进入监控。

### CORS 与内网调用

如果浏览器不直接调用 Go-Gin，Go-Gin 可以只暴露内网地址给 Next.js，减少 CORS 和安全复杂度。

推荐：

- 浏览器访问 Next.js 域名。
- Next.js 服务端通过 `GO_API_INTERNAL_URL` 调 Go-Gin。
- Go-Gin 不直接暴露给公网，或只通过 API 网关暴露必要接口。
- 不把 Go-Gin 内网地址放进 `NEXT_PUBLIC_*`。

---

## 7. 快速检查清单

- [ ] 点击后是否能立即给出反馈？例如 `loading.tsx`、`Suspense fallback`、按钮 pending 状态。
- [ ] `'use client'` 是否只存在于必要的叶子交互组件中？
- [ ] `layout.tsx` 是否保持为 Server Component？
- [ ] Next.js 服务端是否通过内网地址调用 Go-Gin？
- [ ] 多个无依赖的 Go-Gin 请求是否使用 `Promise.all` 并行触发？
- [ ] 首屏数据是否优先在 Server Component 中获取，而不是客户端 `useEffect`？
- [ ] Go-Gin 是否为首屏提供必要的聚合接口？
- [ ] Next.js 调 Go-Gin 是否设置了超时和错误兜底？
- [ ] 是否避免了不必要的 `useEffect` 首屏取数？
- [ ] 响应式图片是否使用 `next/image` 并配置合适的 `sizes`？
- [ ] 确定的 LCP / hero 图片是否使用了合适的 `preload`、`loading` 或 `fetchPriority` 策略？
- [ ] 大型第三方库是否按需引入或动态加载？
- [ ] Proxy / Middleware 是否保持轻量？
- [ ] Proxy 是否避免每次请求都调用 Go-Gin 做复杂鉴权？
- [ ] 可以缓存的数据或组件是否考虑使用 `use cache`？
- [ ] Go-Gin 是否对公开数据返回合理 `Cache-Control`？
- [ ] 未启用 Cache Components 的旧缓存模型页面，是否考虑声明 `export const dynamic = 'force-static'`？
- [ ] 启用 Cache Components 后，是否避免继续依赖 `dynamic`、`revalidate`、`fetchCache` 等旧配置？

---

好的 Next.js 应用不要求所有内容都秒完成，但点击后应该秒反馈；首屏优先可见，慢数据局部加载。

---

## 总结

好的 Next.js App Router 应用应该是：

- Next.js 负责页面、SSR、RSC 和交互体验
- Go-Gin 负责业务 API、事务、权限和核心数据
- 客户端只承担必要交互
- 点击后马上有反馈
- 慢内容局部加载
- 图片、依赖、Proxy 和跨服务调用都保持克制

一句话记住：

> **Next.js 管体验，Go-Gin 管业务；秒反馈，不假死。**

---

## 官方参考链接

- [Next.js 16.2 官方发布说明](https://nextjs.org/blog/next-16-2)
- [Turbopack: What's New in Next.js 16.2](https://nextjs.org/blog/next-16-2-turbopack)
- [Next.js 16 升级指南](https://nextjs.org/docs/app/guides/upgrading/version-16)
- [Next.js Fetching Data 文档](https://nextjs.org/docs/app/getting-started/fetching-data)
- [Next.js Caching / Cache Components 文档](https://nextjs.org/docs/app/getting-started/caching)
- [Next.js Image Component 文档](https://nextjs.org/docs/app/api-reference/components/image)
- [Next.js Lazy Loading 文档](https://nextjs.org/docs/app/guides/lazy-loading)
- [Next.js `optimizePackageImports` 文档](https://nextjs.org/docs/app/api-reference/config/next-config-js/optimizePackageImports)
- [Vercel/Next.js `v16.2.9` 官方仓库发布页](https://github.com/vercel/next.js/releases/tag/v16.2.9)
- [npm registry: `next@latest`](https://registry.npmjs.org/next/latest)
- [Gin 官方文档](https://gin-gonic.com/en/docs/)
