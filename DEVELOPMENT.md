# Boxli Hub 开发文档

> 面向开发者的**技术实现说明**：架构、模块职责、认证实现、API 细节、调试与安全设计。
>
> 本文档描述**当前代码的实际实现**，并明确标注哪些行为已被实测验证、哪些仅做过静态检查。

## 文档分工

| 文档 | 回答什么问题 | 受众 |
|---|---|---|
| [`README.md`](./README.md) | **怎么把它跑起来** —— 环境准备、启动步骤、配置字段、API 简表 | 初次接触项目的人 |
| 本文档 | **它是怎么实现的** —— 架构、设计理由、实现细节、调试与测试 | 改代码的人 |
| [`DEPLOYMENT.md`](./DEPLOYMENT.md) | **怎么上线** —— 构建、systemd、Nginx、HTTPS、备份 | 运维 / 部署者 |
| [`plants.md`](./plants.md) | 立项时的分阶段计划 | **历史存档，仅作考据** |

> **⚠️ `plants.md` 含已被推翻的规格**（错误的 systemd `ExecStart`、SSG 方案、`:3000` 端口
> 误判等），**不作为现行依据**。本文档与 `DEPLOYMENT.md` 描述的是实际实现，
> 冲突时以那两者为准。

---

## 目录

1. [架构总览](#一架构总览)
2. [目录与模块职责](#二目录与模块职责)
3. [数据模型](#三数据模型)
4. [后端实现](#四后端实现)
5. [认证与会话](#五认证与会话)
6. [前端实现](#六前端实现)
7. [API 参考](#七api-参考)
8. [本地开发与调试](#八本地开发与调试)
9. [测试与验证](#九测试与验证)
10. [安全设计汇总](#十安全设计汇总)
11. [已知限制与后续工作](#十一已知限制与后续工作)

---

## 一、架构总览

```
用户浏览器
    │
    ▼
┌─────────────────────────────────┐
│  Nginx (80/443)                 │  ← 生产环境
│  /api/*  → 反代到后端            │
│  其余路径 → 前端 (Nuxt SSR)      │
│           （/docs 也是 Nuxt 路由）│
└──────────────┬──────────────────┘
               │ 127.0.0.1:3727（仅回环，外部不可达）
               ▼
┌─────────────────────────────────┐
│  Boxli Hub 后端 (Go)             │  只存元数据，不存镜像文件
└──────────────┬──────────────────┘
               │ 127.0.0.1:5432（仅回环）
               ▼
┌─────────────────────────────────┐
│  PostgreSQL 17                  │  users / repositories / tags / sources / sessions / oauth_states
└─────────────────────────────────┘

镜像文件本体：用户自己的 GitHub / Gitee / OSS / S3 / IPFS / BT …
Hub 只记录「去哪下载」
```

**核心约束：**

| 约束 | 说明 |
|---|---|
| 前后端完全分离 | 开发期前端经 Nuxt `routeRules` 同源代理 `/api/**`；生产经 Nginx 反代 |
| 后端仅监听回环 | `127.0.0.1:3727`，端口扫描不可见 |
| 数据库仅监听回环 | `127.0.0.1:5432` |
| 不存储镜像本体 | `sources.url` 只记录外链地址 |
| 手机优先 | 先写 <640px，再 `md:`/`lg:`（要点见 [6.6](#66-手机优先实现要点)） |

---

## 二、目录与模块职责

### 后端 `backend/`

```
cmd/
├── hub/main.go              # 服务入口：加载配置 → 连库 → 迁移 → 监听 → 优雅关闭
└── seed/main.go             # 种子数据入口（幂等，可重复执行）

internal/
├── auth/
│   ├── auth.go              # JWT 签发/校验 + sessions 表读写（存 sid 的 SHA-256）
│   ├── github.go            # GitHub OAuth：authorize URL / code 换 token / 拉用户
│   ├── middleware.go        # 鉴权中间件：解析会话 → 注入 context
│   ├── cookie.go            # httpOnly 会话 Cookie 下发/清除/提取
│   ├── origin.go            # CSRF 来源白名单校验
│   ├── state.go             # OAuth state 一次性校验（DELETE ... RETURNING）
│   ├── state_test.go        # state 单次消费 / 伪造 / 过期
│   └── origin_test.go       # 来源校验 + Cookie 属性测试
├── config/config.go         # TOML 配置加载 + 启动期校验（hub.toml）
├── config/config_test.go    # 默认值/未知键/校验/脱敏 断言
├── db/
│   ├── db.go                # pgxpool 连接池 + go:embed 迁移执行器
│   └── migrations/
│       ├── 0001_init.sql    # 5 张业务表 + 索引
│       └── 0002_oauth_states.sql
├── hub/
│   ├── server.go            # 路由分发 + 中间件链 + 统一响应封装 + CORS
│   ├── store.go             # 读查询：search / listRepos / getRepo / readme
│   ├── store_write.go       # 写事务：create / update / delete（整体替换语义）
│   ├── handlers_read.go     # 读接口处理器
│   ├── handlers_write.go    # 写接口处理器
│   ├── handlers_auth.go     # 登录/回调/me/登出 + upsertUser + 回跳与 Cookie
│   └── redirect_test.go     # 开放重定向防护测试
└── seed/seed.go             # 种子数据定义
```

### 前端 `frontend/`

```
app/
├── app.vue                  # 根组件（NuxtLayout + NuxtPage）+ 全局 meta
├── layouts/default.vue      # Header + main + Footer，min-h-[100dvh] + overflow-x-hidden
├── middleware/auth.ts       # 登录守卫（仅客户端判定）
├── components/
│   ├── SiteHeader.vue       # 导航 + 手机 Drawer + 登录态用户菜单
│   ├── SiteFooter.vue       # 页脚（safe-area 内边距）
│   ├── RepoCard.vue         # 镜像卡片
│   ├── SourceList.vue       # 详情页多源下载列表
│   ├── SourceEditor.vue     # 下载源增删改（顺序即优先级）
│   ├── RepoForm.vue         # 建仓/改仓共用表单
│   ├── CopyButton.vue       # 复制按钮（含降级方案）
│   ├── EmptyState.vue       # 空状态
│   └── CardSkeleton.vue     # 加载骨架
├── composables/
│   ├── useApi.ts            # 读接口：useFetch + 统一解包 {code,message,data}
│   ├── useAuth.ts           # 登录态 + login/logout + apiWrite
│   └── useMarkdown.ts       # markdown-it(html:false) + DOMPurify
├── pages/
│   ├── index.vue            # 首页
│   ├── explore/index.vue    # 浏览 + 搜索/排序
│   ├── explore/[ns]/[repo].vue  # 详情
│   ├── search.vue           # 搜索
│   ├── login.vue            # 登录
│   ├── submit.vue           # 提交镜像
│   ├── dashboard.vue        # 用户中心
│   ├── docs/                # 文档站（7 个页面）
│   │   ├── index.vue        #   文档首页
│   │   ├── quickstart.vue   #   快速开始
│   │   ├── install.vue      #   安装与自检
│   │   ├── pull-run.vue     #   拉取与运行
│   │   ├── format.vue       #   镜像格式与下载源
│   │   ├── submit.vue       #   提交镜像到 Hub
│   │   └── faq.vue          #   常见问题
│   └── about.vue            # 关于
├── types/api.ts             # 后端响应类型（AuthUser/RepoInput 等）
└── utils/
    ├── format.ts            # 字节/数量/相对时间格式化 + 源类型元数据 + copyText
    └── docs.ts              # 文档内容、导航与标题锚点渲染
```

**文档站布局**：`components/DocsLayout.vue`（桌面 sticky 侧边导航 /
手机折叠目录 / 本页小节锚点 / 上下页翻页）。

---

## 三、数据模型

```
users (1) ──< repositories (1) ──< tags (1) ──< sources
  │
  └──< sessions

oauth_states（独立，OAuth CSRF 用，无外键）
```

| 表 | 关键约束 | 说明 |
|---|---|---|
| `users` | `username` UNIQUE、`email` UNIQUE、`github_id` UNIQUE | 唯一约束是 L7 账号接管防护的基础 |
| `repositories` | `UNIQUE(namespace, name)`、`owner_id` → users | 镜像仓库 |
| `tags` | `UNIQUE(repo_id, tag, os, arch)` | 同一仓库按 tag+os+arch 区分 |
| `sources` | `tag_id` → tags `ON DELETE CASCADE` | 每标签多个下载源 |
| `sessions` | `id` UUID PK、`user_id` CASCADE、`token_hash` | 会话；登出即删行 |
| `oauth_states` | `state` PK、`expires_at` | 一次性 state，10 分钟 TTL |

迁移由 `internal/db/db.go` 的 `go:embed` 执行器在**后端启动时自动运行**，
记录在 `schema_migrations` 表，**可重复执行**。

- `tags` 与 `sources` 均随 `repositories` 级联删除（`ON DELETE CASCADE`）
- `sources` 随 `tags` 级联删除

---

## 四、后端实现

### 4.1 中间件链

```go
// server.go
handler := withLogging(s.withCORS(auth.OriginGuard(s.cfg.AllowedOrigins())(mux)))
```

执行顺序（由外到内）：

| 顺序 | 中间件 | 职责 |
|---|---|---|
| 1 | `withLogging` | 记录 `METHOD /path` |
| 2 | `s.withCORS` | 白名单内的 Origin 才回显 CORS 头 + `Allow-Credentials` |
| 3 | `auth.OriginGuard` | **写方法**的 Origin/Referer 同源校验（CSRF） |
| 4 | `mux` | 路由；写接口再叠 `auth.Middleware`（会话鉴权） |

> `OriginGuard` 放在路由**之前**，确保覆盖所有写方法（含未来新增的写接口），
> 不需要每个 handler 单独记得加。

### 4.2 统一响应格式

```go
type response struct {
    Code    int         `json:"code"`     // 0 = 成功；失败时为 HTTP 状态码
    Message string      `json:"message"`  // "ok" 或错误描述
    Data    interface{} `json:"data"`
}
```

- 成功：`writeOK(w, data)` → `{code:0, message:"ok", data:…}`
- 失败：`writeErr(w, status, msg)` → `{code:<status>, message:…, data:null}`
- 请求体解析：`decodeJSON` 限制 **1 MiB** 并 `DisallowUnknownFields()`

### 4.3 读接口

`store.go` 中，`getRepo` 用 `json_agg` 一次查出仓库 + 所有 tags + 每个 tag 的 sources
（sources 按 `priority` 排序），避免 N+1 查询。

### 4.4 写接口（事务 + 整体替换）

`createRepo` / `updateRepo` 均在**单个事务**内完成：

1. 校验 payload（`RepoInput.validate()`）
2. 写入/定位 repository
3. `insertTagsAndSources` 写入全部 tags 与 sources
4. 提交；失败则 `defer tx.Rollback`

**权限**：`updateRepo` / `deleteRepo` 先查 `owner_id`，非 owner 返回 `ErrForbidden` → 403。

> **⚠️ `PUT` 是 tags/sources 整体替换语义**：请求体中未包含的标签与源会被**删除**。
> 前端 `/dashboard` 编辑时因此先拉全量详情再预填（见 [6.4](#64-dashboard-编辑必须整体回填)）。

---

## 五、认证与会话

### 5.1 会话存储：JWT + sessions 表双写

`internal/auth/auth.go`：

- `Issue()`：生成随机 `sid`（16 字节 hex）→ 写入 `sessions`（存 **sid 的 SHA-256**，不存原文）
  → 签发 HS256 JWT（claims 含 `uid` 与 `sid`）
- `Verify()`：验签名 + 查 `sessions` 表确认 `sid` 未过期未吊销 + 比对哈希 + 载入用户
- `Revoke()`：删除 session 行，旧 token 立即失效

即 **JWT 无状态签名 + 服务端可吊销会话** 的组合：既能水平扩展，又能即时登出。

### 5.2 会话交付：httpOnly Cookie

回调不返回裸 JSON，改为 **302 回前端**，会话以 Cookie 下发。

```go
// internal/auth/cookie.go
const SessionCookieName = "boxli_session"

http.SetCookie(w, &http.Cookie{
    Name:     SessionCookieName,
    Value:    token,
    Path:     "/",
    HttpOnly: true,                  // JS 读不到 → XSS 无法窃取
    Secure:   opts.Secure,           // 生产 HTTPS 必须 true
    SameSite: http.SameSiteLaxMode,  // 跨站 POST 不带 Cookie
    MaxAge:   opts.MaxAge,
})
```

**为什么不用 localStorage**：XSS 可直接读走会话。
**为什么不把 token 放重定向 URL**：会进入浏览器历史、`Referer` 头与服务器访问日志。

**token 提取顺序**（`SessionTokenFrom`）：Cookie 优先 → 回退 `Authorization: Bearer`。
两条路径都支持，浏览器走 Cookie，**curl / CLI / 测试脚本走 Bearer**，无需处理 Cookie。

> **⚠️ `SameSite` 为什么用 `Lax` 而非 `Strict`**：`Strict` 会导致「从 GitHub 跳回本站」
> 这一顶层导航**不携带 Cookie**，表现为登录后仍未登录。

### 5.3 OAuth 流程全链路

```
① 前端 POST /auth/login {redirect:"/submit"}
   └─ 后端生成随机 state（32 字节）写入 oauth_states，同时记下 redirect
   └─ 返回 authorize_url（含 state）
② 浏览器整页跳转 → GitHub 授权页
③ 用户点「授权」
④ GitHub 回调 GET /auth/callback?code=…&state=…（回调到 127.0.0.1:3727，App 登记值）
   └─ ① 校验并消费 state（DELETE ... RETURNING，原子）
   └─ ② 检查 error 参数（用户取消）
   └─ ③ code 换 access token
   └─ ④ 拉取 GitHub 用户信息
   └─ ⑤ upsert 用户（冲突 → 409 语义）
   └─ ⑥ 签发会话 → Set-Cookie
   └─ ⑦ 302 到 [site] frontend_url + redirect（站内路径）
⑤ 前端页面 → GET /auth/me（自动带 Cookie）→ 显示已登录
```

**state 校验必须最先执行且总是消费**：无论后续成功、被用户拒绝还是出错，
state 都已作废，不留可重放的凭证。实现用 `DELETE ... RETURNING` 保证
「校验+删除」原子性，并发重放只有一个能成功。

### 5.4 错误码（302 到 `/login?error=<code>`）

| code | 触发条件 |
|---|---|
| `invalid_state` | state 缺失 / 伪造 / 已使用 / 过期 |
| `state_error` | state 校验时数据库错误（500 级，但为 UX 仍回跳） |
| `access_denied` | 用户在 GitHub 取消授权 |
| `missing_code` | GitHub 未返回 code |
| `exchange_failed` | 换 access token 失败（502 级） |
| `github_failed` | 拉取 GitHub 用户失败 |
| `identity_taken` | username/email 被另一 `github_id` 占用（L7） |
| `user_failed` | 创建/更新用户失败 |
| `issue_failed` | 签发会话失败 |

前端 `login.vue` 把每个 code 映射为中文可读提示。

### 5.5 CSRF 防护（Cookie 方案的必要配套）

改用 Cookie 后，浏览器会自动携带凭证，「跨站发起的写请求」因此自带认证 —— 这正是 CSRF。

| 层 | 机制 | 说明 |
|---|---|---|
| 浏览器层 | Cookie `SameSite=Lax` | 跨站 POST 不携带 Cookie |
| 应用层 | `OriginGuard` | 校验 `Origin`（缺失则回退 `Referer`）必须与白名单**精确同源** |

判定规则：

- 只作用于**非安全方法**（`POST`/`PUT`/`DELETE`/`PATCH`）；`GET`/`HEAD`/`OPTIONS` 放行
  （`OPTIONS` 是 CORS 预检，拦掉会让正常请求也失败）
- `Origin` 与 `Referer` **都没有** → 放行（curl/CLI 不携带 ambient 凭证，不构成 CSRF）
- `Origin: null` → **拒绝**（沙箱 iframe、`file://` 等不可信场景）

> **⚠️ 必须用 URL 解析而非前缀匹配**：`strings.HasPrefix(origin, "http://localhost:3011")`
> 会把 `http://localhost:3011.evil.com` 判为可信。实现用 `url.Parse` 比较 scheme+host+port。

### 5.6 开放重定向防护

登录回跳地址由前端查询参数传入，属于不可信输入：

```go
func safeRedirectPath(p string) string {
    if p == "" { return "" }
    if !strings.HasPrefix(p, "/") || strings.HasPrefix(p, "//") { return "" }  // 拒绝协议相对
    if strings.ContainsAny(p, "\\\r\n") { return "" }                          // 拒绝 CRLF 注入
    if u, err := url.Parse(p); err != nil || u.IsAbs() || u.Host != "" { return "" }
    return p
}
```

放行 `/submit`、`/explore/a/b`、`/submit?x=1`；拒绝 `https://evil.com`、`//evil.com`、
`javascript:…`、`/path\r\nLocation:…`。不合法时回退 `/dashboard`。

### 5.7 CORS 与 Cookie 的兼容性（易漏点）

`Access-Control-Allow-Origin: *` 与带凭证的请求**天然不兼容**，浏览器会直接拒绝。
因此在改用 Cookie 的同时必须修改 CORS：

```go
if origin != "" && auth.OriginAllowed(origin, allowed) {
    w.Header().Set("Access-Control-Allow-Origin", origin)   // 回显具体 Origin，不能用 *
    w.Header().Set("Access-Control-Allow-Credentials", "true")
    w.Header().Add("Vary", "Origin")                        // 防中间缓存串站
}
```

> 开发期前端经 Nuxt 同源代理，**根本不产生跨源请求**；
> 生产由 Nginx 同源反代后同样不产生。此改动主要为兼容直连后端的场景，并推进 L5。

### 5.8 dev 模拟登录（默认关闭）

`[dev] enabled = true` 且未配 OAuth 凭证时，`POST /auth/login {"github_user":"x"}` 直接下发会话。
**默认关闭**——开启后任何人可登录为任意账号。

> 注意：dev 登录每次生成**随机 `github_id`**，若用户名已存在会触发 L7 的 409 防护。
> 这是**正确行为**，测试时换个未占用的用户名即可。

### 5.9 `upsertUser` 与 L7

```sql
INSERT INTO users (username, email, avatar_url, github_id)
VALUES ($1, NULLIF($2,''), NULLIF($3,''), $4)
ON CONFLICT (github_id) DO UPDATE SET username = EXCLUDED.username, …
RETURNING id, username
```

唯一冲突（username/email 被**另一个** `github_id` 占用）时返回 `errIdentityTaken` → 409。

> **⚠️ 绝不能在冲突时「按 username 退化查找并登录」** —— 那会让新 GitHub 账号
> 接管同名老账号（账号接管漏洞）。这是已修复的 L7。

---

## 六、前端实现

### 6.1 读接口封装（`useApi.ts`）

`useFetch` + `transform` 统一解包 `{code,message,data}`，页面直接消费 `data`。
`useSearch` 在关键词为空时不发请求（`immediate: hasQuery`）。

### 6.2 登录态管理（`useAuth.ts`）

```ts
const user = useState<AuthUser | null>('auth-user', () => null)
const resolved = useState<boolean>('auth-resolved', () => false)
```

- **不保存 token**：会话在 httpOnly Cookie 里，JS 读不到（这正是选该方案的目的）。
  前端只保存「当前用户是谁」这一衍生状态，真相来源是 `GET /auth/me`。
- `refresh()`：调 `/auth/me`，`ignoreResponseError: true` —— 401 是「未登录」的正常表达，
  不该抛错中断流程。
- `ensure()`：已确认过则复用，避免重复请求与 UI 闪烁。
- `apiWrite()`：所有写请求统一加 `credentials: 'include'`，
  否则浏览器不附带 Cookie，会被后端 401。

### 6.3 登录守卫（`middleware/auth.ts`）

```ts
export default defineNuxtRouteMiddleware(async (to) => {
  if (import.meta.server) return          // ← 关键
  const { ensure } = useAuth()
  if (await ensure()) return
  return navigateTo({ path: '/login', query: { redirect: to.fullPath } })
})
```

> **⚠️ 为什么首行必须 `if (import.meta.server) return`**：
> 会话在 httpOnly Cookie 中，**SSR 期间不会自动携带**（服务端内部请求不带浏览器 Cookie）。
> 若在 SSR 就判定并跳转，会把**已登录用户也误判为未登录**，且产生 hydration 不一致。

### 6.4 `/dashboard` 编辑必须整体回填

后端 `PUT` 是 tags/sources **整体替换**语义，而列表接口 `GET /repos` 返回的
`RepoSummary` **不含 tags**。因此 `startEdit()` 会先单独拉一次
`GET /repos/{ns}/{repo}` 拿全量数据，再交给 `RepoForm` 预填。

> 若直接用列表项预填并提交，未在表单中出现的标签与源会被**静默删除**。

### 6.5 `SourceEditor` 的单向数据流

初版实现直接修改传入的 `sources` prop 数组（依赖「数组按引用传递」这一脆弱假设），
被 `eslint` 的 `vue/no-mutating-props` 抓出。现改为：

```ts
const emit = defineEmits<{ (e: 'change', sources: SourceRow[]): void }>()
function commit(next: SourceRow[]) {
  // 复制为新数组并重排 priority，保证「展示顺序 == 优先级」
  emit('change', next.map((s, i) => ({ ...s, priority: i + 1 })))
}
```

**优先级由顺序决定**，用户无需手填 `priority`，避免顺序与优先级不一致。

### 6.6 手机优先实现要点

| 规范 | 实现 |
|---|---|
| 点击区 ≥44px | 统一 `min-h-11` / `min-h-12`；图标按钮补 `min-w-11` |
| 输入框 ≥16px | 所有 `input/textarea/select` 用 `text-base`（防 iOS 聚焦缩放） |
| 不用 `100vh` | 布局用 `min-h-[100dvh]` |
| 无横向滚动 | 布局 `overflow-x-hidden`；输入框 `w-full min-w-0`；长文本 `truncate`/`line-clamp` |
| 不依赖 hover | 菜单、操作按钮全部可点，无 `group-hover` 显隐 |
| safe-area | 页脚 `pb-[calc(2rem+env(safe-area-inset-bottom))]` |

### 6.7 SSR / hydration 注意点

- 相对时间（`formatRelativeTime`）传入固定基准 `now`，避免服务端与客户端时间差导致
  hydration 不一致
- 页脚年份用 `useState` 固定
- 登录态相关的 DOM 差异**一律在客户端渲染后出现**（SSR 阶段统一按未登录渲染）

### 6.8 文档站

文档站不引入任何新依赖，复用既有渲染链路与组件风格：

| 文件 | 职责 |
|---|---|
| `utils/docs.ts` | 文档内容（Markdown 常量）+ `DOC_PAGES` 结构 + `docNav` / `docNeighbors` / `renderDocMarkdown` |
| `components/DocsLayout.vue` | 文档布局：侧边导航、手机折叠目录、本页锚点、上下页翻页、正文排版 |
| `pages/docs/*.vue` | 7 个页面，每个仅一行 `<DocsLayout path="…" />`，标题与描述由数据驱动 |

**内容与导航同源**：页面标题、`<title>`、描述、侧边导航标签、上下页全部从
`DOC_PAGES` 派生，避免多处维护产生不一致。`pages/docs/*.vue` 因此只是薄壳，
不重复声明 `useHead`。

**标题锚点深链**：`useMarkdown.ts` 新增 `slugify` 与 `heading_open` 渲染规则，
为 `##`/`###` 标题注入 `id`：

- slug 规则：小写 → 去反引号与标点 → 空白转 `-`，**保留中日韩文字**
  （如 `CLI 与 Hub 的对接现状` → `cli-与-hub-的对接现状`）
- 同页重复标题自动加 `-1`、`-2` 后缀，避免 `id` 冲突
- **通过 `anchors` 选项开关**：文档站开启，**镜像 README 保持关闭**（行为不变）
- DOMPurify 的 `ADD_ATTR` 增加 `id`，否则清洗时会被剥掉导致锚点失效
- 标题加 `scroll-margin-top: 5rem`，避免锚点跳转后被 sticky 头部遮挡

> **⚠️ 踩坑记录**：首版未给标题注入 `id`，导致文档中手写的跨页深链
> （如 `/docs/faq#cli-与-hub-的对接现状`）**全部是死链**。已用脚本逐条校验
> 「目标页面存在 + 目标锚点存在」修复并回归。

**手机端阅读体验**：

- 侧边导航在 `lg` 以下收起为**点击展开**的折叠目录（不用 hover，规避坑 2）
- 本页小节在手机端改为**横向滚动 chip 条**（`overflow-x-auto` + `w-max`）
- 宽表格 `display: block` + `overflow-x-auto` 横向滚动；代码块 `pre` 同样可横向滚动，
  避免撑破窄屏
- 正文 `text-base`、行高 1.75；行内代码 15px 且 `word-break: break-all` 防长串溢出

**内容准确性约束**：文档中的命令与 flag 照实抄录自真实 `boxli` 二进制
（`boxli version 0.0.0-dev`），不臆造。实测发现 CLI 对接的是**另一套 Hub**
（`boxli hub serve`），与本站 `/api/v1/*` 并非同一实现，已在 `/docs/faq` 首节
如实标注可用范围，且**不提供**无法执行的示例。

---

## 七、API 参考

所有接口在 `/api/v1/` 下，响应统一 `{code, message, data}`。

### 读接口（公开）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/health` | 健康检查 |
| GET | `/search?q=&limit=` | 搜索（`name`/`description`/`namespace` ILIKE） |
| GET | `/repos?namespace=&limit=&offset=` | 仓库列表 |
| GET | `/repos/{ns}/{repo}` | 详情（含所有 tags 与 sources） |
| GET | `/repos/{ns}/{repo}/tags` | 标签列表 |
| GET | `/repos/{ns}/{repo}/readme` | README 文本 |

### 认证

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/auth/login` | 有 OAuth 凭证返回 `authorize_url`；否则需 `[dev] enabled = true` 才模拟登录 |
| GET | `/auth/callback?code=&state=` | 校验并消费 state → 换 token → **302 回前端** + Set-Cookie |
| GET | `/auth/me` | 当前用户（Cookie 或 Bearer） |
| POST | `/auth/logout` | 吊销 session + 清 Cookie |

### 写接口（需登录）

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/repos` | 创建（`namespace` 缺省为当前用户名）→ 201 |
| PUT | `/repos/{ns}/{repo}` | 更新（仅 owner；tags/sources **整体替换**） |
| DELETE | `/repos/{ns}/{repo}` | 删除（仅 owner；级联删除 tags/sources） |

### 请求体示例

```json
{
  "namespace": "alice",
  "name": "myapp",
  "description": "My awesome app",
  "readme": "# myapp",
  "tags": [
    {
      "tag": "v1.0.0", "os": "linux", "arch": "amd64",
      "sources": [
        { "type": "github", "url": "https://github.com/…", "priority": 1, "region": "global" },
        { "type": "gitee",  "url": "https://gitee.com/…",  "priority": 2, "region": "cn" }
      ]
    }
  ]
}
```

**支持的源类型**：`github` `gitlab` `gitee` `http` `s3` `oss` `cos` `ipfs` `magnet` `direct`

**校验规则**：`namespace`+`name` 必填、至少 1 个 tag、每 tag 至少 1 个 source、
每 source 的 `type` 与 `url` 必填。

### 错误码

| HTTP | 场景 |
|---|---|
| 400 | 请求体非法 / 缺必填字段 / 空 tags |
| 401 | 无会话凭证或会话失效 |
| 403 | 非 owner / **跨站来源被 CSRF 拦截** |
| 404 | 仓库不存在 |
| 409 | 仓库已存在 / 用户名被占用 |
| 503 | 未配 `[session] jwt_secret`；或未配 OAuth 且未开 dev 登录 |

---

## 八、本地开发与调试

### 8.1 后端

```bash
cd backend
export GOCACHE=/home/xgp2012/hub/.devtools/gocache \
       GOMODCACHE=/home/xgp2012/hub/.devtools/gomodcache \
       GOPATH=/home/xgp2012/hub/.devtools/gopath
# 首次：复制 hub.toml.example 为 hub.toml 并按需修改
cp hub.toml.example hub.toml
go run ./cmd/hub     # 启动时自动迁移；监听 127.0.0.1:3727（可加 --config 指定文件）
go run ./cmd/seed    # 种子数据（幂等）
```

> 沙箱不允许写 `~/.cache` 与 `~/go`，故 Go 缓存必须指向工作区内。

### 8.2 前端

> **⚠️ 本机不能用 3000 端口** —— 3000 属 **Forgejo**（`forgejo.service`，uid 115，开机自启），
> 与 Boxli 前端无关。必须换端口（如 3011），并**同步修改 `hub.toml` 的 `[site] frontend_url`**，
> 否则 OAuth 回跳落错站点、写请求被 CSRF 拦截（403）。见 [11.1](#111-本机端口约束务必遵守)。

```bash
cd frontend
export npm_config_cache=/home/xgp2012/hub/.devtools/npmcache
npm install
PORT=3011 node node_modules/nuxt/bin/nuxt.mjs dev   # 注意：.bin/nuxt 可能缺可执行位
node node_modules/nuxt/bin/nuxt.mjs build
PORT=3011 node .output/server/index.mjs             # 生产预览（端口任选非 3000）
```

前端 dev 通过 `routeRules` 把 `/api/**` 代理到 `http://127.0.0.1:3727`，**无需 CORS**。

> **如何判断某个端口归谁**（本机排查要点）：
> `ss -ltnp` 对**其他用户**持有的 socket **不显示 PID**，容易被误读成「孤儿 socket」。
> 应改用 `ss -ltnpe`，读取 `uid:` 与 `cgroup:` 字段来判定归属：
>
> ```bash
> ss -ltnpe | grep -E ':(3000|3727)\b'
> # *:3000  uid:115  cgroup:/system.slice/forgejo.service   ← 系统服务 Forgejo，不要动
> # 127.0.0.1:3727  users:(("hub",pid=…)) uid:1001          ← 本项目后端
> ```

### 8.3 用 curl 调试认证（走 Bearer，无需处理 Cookie）

```bash
# 1. 登录拿 token（dev 模式，需 [dev] enabled = true 且未配 OAuth）
TOKEN=$(curl -s -X POST http://127.0.0.1:3727/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"github_user":"localdev"}' | python3 -c 'import sys,json;print(json.load(sys.stdin)["data"]["token"])')

# 2. 带 Bearer 访问
curl -s http://127.0.0.1:3727/api/v1/auth/me -H "Authorization: Bearer $TOKEN"

# 3. 写请求（注意：带 Origin 时会被 CSRF 校验，CLI 不带 Origin 即放行）
curl -s -X POST http://127.0.0.1:3727/api/v1/repos \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"demo","tags":[{"tag":"v1","os":"linux","arch":"amd64",
       "sources":[{"type":"github","url":"https://github.com/a/b","priority":1}]}]}'
```

### 8.4 用 curl 调试 Cookie 会话（模拟浏览器）

```bash
# Origin 必须是后端白名单里的前端站点（即 [site] frontend_url，本机示例 3011）
curl -s -c cookies.txt -X POST http://127.0.0.1:3727/api/v1/auth/login \
  -H 'Origin: http://localhost:3011' -H 'Content-Type: application/json' \
  -d '{"github_user":"localdev"}'

curl -s -b cookies.txt http://127.0.0.1:3727/api/v1/auth/me
```

### 8.5 常见问题

| 症状 | 原因 / 处理 |
|---|---|
| 登录后仍显示未登录 | `[session] cookie_secure = true` 却在 `http://localhost` 下访问 → 浏览器丢弃 Cookie；本地应设 `false`（配置不匹配时启动即报错） |
| 写请求 401 | 前端漏了 `credentials: 'include'` → 统一走 `apiWrite()` |
| 写请求 403 `cross-site request blocked` | 请求 `Origin` 不在 `[site] frontend_url` / `extra_origins` 白名单 |
| `/auth/login` 返回 503 | 未配 OAuth 凭证且 `[dev] enabled` 未开 → 补凭证或临时开启 dev |
| dev 登录返回 409 | 用户名已被占用（L7 防护）；换一个用户名 |
| 端口 `EADDRINUSE` | 旧进程未退出。先 `ss -ltnpe` 看**归属**再动手：`uid:`/`cgroup:` 能说明是谁占用（如 `forgejo.service`），`users:(("hub",pid=…))` 则是本项目后端，可用 `pkill -f 'go run ./cmd/hub'` 清理 |
| 前端起不来 / 打开 `localhost:3000` 不是本站 | **3000 是 Forgejo**，不是 Boxli 前端；换端口（如 `PORT=3011`）并同步 `[site] frontend_url` |
| 登录后回跳到 Forgejo 页面 | `[site] frontend_url` 仍指向 3000（Forgejo）→ 改为前端实际端口 |
| `.bin/nuxt` 权限拒绝 | 用 `node node_modules/nuxt/bin/nuxt.mjs` 直接运行 |

---

## 九、测试与验证

### 9.1 自动化测试

```bash
cd backend && go test ./... -count=1 -v
```

**8 个用例组**（另有 **13 个子例**），全部通过：

| 用例 | 覆盖 |
|---|---|
| `TestOriginAllowed`（11 例） | 同源匹配、端口/scheme 差异、前缀绕过（`boxli.dev.evil.com`）、`null`、空值 |
| `TestOriginGuard`（10 子例） | 同源放行、跨站拦截、仿前缀拦截、`null` 拦截、无 Origin 放行、Referer 回退（同源/跨站）、DELETE 拦截、GET 放行、OPTIONS 放行；并断言**被拦截请求不得到达处理器** |
| `TestSessionTokenFrom`（3 子例） | Cookie 优先、回退 Bearer、两者皆空 |
| `TestSetAndClearSessionCookie` | `HttpOnly` / `Secure` / `SameSite=Lax` / `Path=/`；登出 `MaxAge<0` 且值为空 |
| `TestSafeRedirectPath`（13 例） | 站内路径放行；`//evil.com`、绝对 URL、反斜杠、CRLF、`javascript:`、无前导斜杠全部拒绝 |
| `TestStateConsumeIsSingleUse` | state 单次消费 |
| `TestStateRejectsUnknownAndEmpty` | 伪造 / 空 state |
| `TestStateExpiredRejected` | 过期 state |

前端：`eslint .` **0 error / 0 warning**；`nuxt build` 成功。

### 9.1.1 文档站验证

生产构建（`node .output/server/index.mjs`，:3077）下的脚本化校验：

| 项 | 方法 | 结果 |
|---|---|---|
| 路由可用 | 逐个请求 7 个 `/docs*` 路径 | ✅ 全部 200 |
| 标题正确 | 比对渲染出的 `<h1>` 与 `<title>` | ✅ 与 `DOC_PAGES` 一致 |
| 内部链接 | 提取全部 `href="/docs…"` 并逐条解析 | ✅ **51 条全部可达** |
| 跨页深链 | 校验「目标页面存在 **且** 目标锚点存在」 | ✅ `missing = none` |
| 标题锚点 | 检查 `##` 是否带 `id`、重复标题是否去重 | ✅ 如 `cli-与-hub-的对接现状` |
| README 回归 | 确认详情页 README 标题**不带** `id` | ✅ 行为未变 |
| 既有页面回归 | `/`、`/explore`、`/search`、`/about`、`/login`、`/submit`、`/dashboard` | ✅ 全部 200 |
| 手机端静态断言 | viewport / `100dvh` / 无 `100vh` / `overflow-x-hidden` / safe-area | ✅ 7 页全通过 |
| 触控与字号 | 按钮 ≥44px、输入框 `text-base`、无 `group-hover` | ✅ 全通过 |
| XSS 防护 | 构造 `<script>` / `<img onerror>` / `<iframe>` 载荷 | ✅ 无可执行节点 |

### 9.2 已实测的行为（curl，2026-10-02）

| 类别 | 场景 | 结果 |
|---|---|---|
| Cookie | 登录下发 `HttpOnly; SameSite=Lax` Cookie | ✅ |
| | Cookie 访问 `/auth/me` | ✅ 200 |
| | 无凭证 | ✅ 401 |
| | 登出清 Cookie + 旧会话失效 | ✅ |
| CSRF | 跨站 Origin 写请求 | ✅ 403 |
| | 同源 Origin 写请求 | ✅ 通过 |
| | `OPTIONS` 预检 | ✅ 204 |
| 回调 | 伪造 / 缺失 state | ✅ 302 `invalid_state` |
| | 用户拒绝 | ✅ 302 `access_denied`，state 已消费 |
| | state 重放 | ✅ 仍 302 `invalid_state` |
| | **完整链路**：302 → `/login?error=…` → 中文提示 | ✅ |
| | token 是否出现在 URL | ✅ **否** |
| 写接口 | 提交 2 标签、其中 1 标签含 **3 源** | ✅ 201，**3 源全部落库**，priority/region 正确 |
| | `PUT` 整体替换 | ✅ 200，旧标签与新源按语义替换 |
| | `DELETE` 级联 | ✅ 200，删除后 404 |
| | 非 owner 改 / 删 | ✅ 403 / 403 |
| | 空 tags | ✅ 400 |
| 其它 | dev 登录默认关闭 | ✅ 503 且**未创建用户** |
| | 数据复原 | ✅ 回到种子基线（5 users / 5 repos / 10 tags / 17 sources） |

### 9.3 ⚠️ 尚未验证的部分（不得视为已验收）

| 项 | 状态 |
|---|---|
| **真实 GitHub 授权端到端** | ❌ **未完成** —— 与端口无关（见 [11.1](#111-本机端口约束务必遵守)），但仍需人工点一次授权 |
| 真实浏览器 / 真机测试 | ❌ 未做。全部页面均为**静态断言** |
| `CopyButton` 宽度修复复测 | ❌ 未复测 |
| Lighthouse Mobile ≥ 90 | ❌ 从未测量 |
| 真实触摸延迟、iOS 滚动惯性、`100dvh` 动态表现、iOS 聚焦 | ❌ 未验证 |
| 文档站 7 个页面的真实浏览器阅读体验 | ❌ 未做（同样只有静态断言） |

以上需用真实浏览器或真机补齐（见 [11.3](#113-手机端验收待做)）。

---

## 十、安全设计汇总

| 威胁 | 防护 | 位置 |
|---|---|---|
| XSS 窃取会话 | httpOnly Cookie（JS 不可读） | `auth/cookie.go` |
| CSRF | `SameSite=Lax` + `OriginGuard` 精确同源校验 | `auth/origin.go` |
| 来源伪造（前缀绕过） | `url.Parse` 比较 scheme+host，非前缀匹配 | `auth/origin.go` |
| 开放重定向 | `safeRedirectPath` 仅放行站内相对路径 | `hub/handlers_auth.go` |
| CRLF / 响应头注入 | 回跳路径拒绝 `\r` `\n` `\` | 同上 |
| OAuth CSRF / 授权码重放 | state 一次性原子消费（`DELETE ... RETURNING`） | `auth/state.go` |
| 账号接管 | username 冲突返回 409，**不退化查找** | `hub/handlers_auth.go` |
| 会话伪造 / 盗用 | JWT HS256 签名 + `sessions` 表哈希校验 + 可吊销 | `auth/auth.go` |
| 本地调试后门流入生产 | `[dev] enabled` **默认关闭**，且设为 `true` 时程序拒绝启动 | `config/config.go` |
| Markdown XSS | `markdown-it(html:false)` + DOMPurify 清洗 | `composables/useMarkdown.ts` |
| 外链劫持 | 自动补 `rel="noopener noreferrer nofollow"` | 同上 |
| 请求体过大 | `http.MaxBytesReader` 限制 1 MiB | `hub/server.go` |
| 后端端口暴露 | 仅监听 `127.0.0.1:3727` | `config/config.go` |
| 数据库端口暴露 | 仅监听 `127.0.0.1:5432` | PG 启动参数 |
| 越权修改 | owner 校验 → 403 | `hub/store_write.go` |

**待处理（上线前）**：

- **L6**：无速率限制、无请求 ID 日志
- **L5 收尾**：生产由 Nginx 同源反代后移除 CORS 头
- **轮换 `[github] secret`**：该 secret 曾以明文外泄，**上线前必须在 GitHub Regenerate**

---

## 十一、已知限制与后续工作

### 11.1 本机端口约束（务必遵守）

本机 `:3000` 属 **Forgejo**（`forgejo.service`，uid 115，systemd 开机自启），
**与 Boxli 前端无关**。因此：

- **前端不能用 3000**，改用 **3011** 或 **3077**（`:3001` 属 docker-proxy）
- `hub.toml` 的 **`[site] frontend_url` 必须与前端实际端口一致**

该字段身兼两职，写错会同时坏两件事：

| 故障 | 原因 |
|---|---|
| OAuth 成功后 **302 落到 Forgejo** 而非本站 | `[site] frontend_url` 即 302 回跳目标 |
| 真实前端的**写请求被 403 拒绝** | 同一字段也是 `OriginGuard` 的 CSRF 来源白名单 |

```toml
[site]
frontend_url = "http://localhost:3011"
```

**排查端口归属时用 `ss -ltnpe`**，读 `uid:` 与 `cgroup:` 字段：

```bash
ss -ltnpe | grep -E ':(3000|3727)\b'
# *:3000           uid:115   cgroup:/system.slice/forgejo.service  ← Forgejo，不得清理
# 127.0.0.1:3727   uid:1001  users:(("hub",pid=937975,fd=6))       ← 本项目后端
```

> **⚠️ `ss -ltnp` 会误导**：它对**其他用户**持有的 socket 不解析 PID，输出里没有
> `users:(...)`。曾据此误判 Forgejo 占用的 3000 为「本项目的孤儿 socket」——
> 实际是「属主是别的用户，当前用户无权解析」。**判定归属必须看 `uid:`/`cgroup:`**。

> 该约束**不影响代码正确性**：Cookie 会话、302 回跳、CSRF、state 一次性消费、
> 完整 CRUD 与权限校验均已 curl 逐项实测通过（见 [9.2](#92-已实测的行为curl2026-10-02)）。

### 11.2 文档站 `/docs`（已完成）

- [x] 文档路由与侧边导航（桌面 sticky + 手机折叠目录 + 本页小节锚点 + 上下页翻页）
- [x] 快速开始、安装、`pull`/`run` 命令、镜像格式说明、FAQ（共 7 个页面）
- [x] 手机端阅读体验（折叠目录、横向滚动 chip/表格/代码块、16px 正文）
- [x] 标题锚点深链（`slugify` + `heading_open`，`anchors` 选项控制，README 行为不变）

实现细节见 [6.8 文档站](#68-文档站)。

**验证**：7 个路由全部 200；51 条内部链接与跨页深链逐条校验通过；
`eslint` 0 error / 0 warning；`nuxt build` 成功。

> **⚠️ 手机端仍为静态断言**（viewport、`100dvh`、`overflow-x-hidden`、44px 触控、
> 16px 输入框、无 hover-only），**未做真实浏览器/真机测试**，
> 真实验收属 [11.3](#113-手机端验收待做)。

> **⚠️ 内容准确性**：命令与 flag 抄录自真实 `boxli` 二进制（`0.0.0-dev`，实际 28 个子命令），
> 镜像引用为 `NAME:VERSION`（无命名空间段），`pull` 为双语义。
> 实测确认本机 `boxli` 对接的是**另一套 Hub**（`boxli hub serve`），
> 与本站 `/api/v1/*` 并非同一实现 —— 已在 `/docs/faq` 首节如实标注，
> 且不提供无法执行的示例。**CLI 与本站 Hub 的对接为后续工作项。**

### 11.3 手机端验收（待做）

需重新准备浏览器环境（或真机 / BrowserStack），补齐 [9.3](#93-尚未验证的部分不得视为已验收) 全部项。

### 11.4 部署上线（待做）

> 完整步骤见 [`DEPLOYMENT.md`](./DEPLOYMENT.md)，此处仅列要点。

- Nginx：前端 SSR 反代 + `/api/` 反代
  （**注意：`/docs` 就是 Nuxt 路由，没有独立的静态目录**）
- Let's Encrypt HTTPS
- systemd：`boxli-hub.service` + 前端服务
- **`[session] cookie_secure = true`**（生产 HTTPS 必须；配 `http://` 会拒绝启动）
- **`[site] frontend_url = "https://boxli.dev"`**，并同步更新 GitHub App 登记的 production 回调地址
- **Regenerate `[github] secret`**
- ufw 只开 80/443；确认 3727/5432 不可达
- 每日数据库备份 + 定时任务
- 移除 CORS 头（同源反代后不再需要）
- L6：按需加限流与请求 ID 日志

### 11.5 设计取舍备忘

| 取舍 | 选择与理由 |
|---|---|
| 会话载体 | **httpOnly Cookie** 而非 localStorage（防 XSS）或换取码（仍经地址栏） |
| state 存储 | **数据库**而非内存：多实例可校验、进程重启不丢失 |
| 会话模型 | JWT + sessions 表双写：无状态签名 + 可即时吊销 |
| 鉴权来源 | Cookie **与** Bearer 并存：浏览器安全、CLI 便利 |
| 登录守卫时机 | **仅客户端**：Cookie 不参与 SSR，服务端判定会误判已登录用户 |
| 更新语义 | tags/sources **整体替换**：实现简单、语义明确；代价是前端必须全量回填 |
| 优先级表达 | 由**数组顺序**决定：避免用户手填 `priority` 与顺序矛盾 |
| 来源校验 | `url.Parse` 精确比较，而非前缀匹配（防前缀绕过） |
| 文档内容载体 | **Markdown 常量 + 数据驱动导航**：复用 README 的安全渲染链路，页面仅为薄壳，标题/导航/翻页单点维护 |
| 标题锚点开关 | `anchors` **选项控制**：文档站开启、README 关闭，避免改变既有渲染行为 |
| 文档内容口径 | **以真实 CLI 为准**（抄录 `--help`），不按规格臆造；CLI 与本站 Hub 的差异**如实标注**而非掩盖 |

---

**有不清楚的随时问。**
