# Boxli Hub 开发文档

面向**改这个仓库的人**：讲清楚代码长什么样、为什么这么写、以及容易踩的地方。

- 想跑起来、想部署 → 见 [`README.md`](./README.md)
- 想上线 → 见 [`DEPLOYMENT.md`](./DEPLOYMENT.md)

---

## 目录

- [一、技术栈与架构](#一技术栈与架构)
- [二、代码结构](#二代码结构)
- [三、数据模型](#三数据模型)
- [四、后端实现](#四后端实现)
- [五、认证与会话](#五认证与会话)
- [六、前端实现](#六前端实现)
- [七、前端内嵌与 SSR 托管](#七前端内嵌与-ssr-托管)
- [八、API 参考](#八api-参考)
- [九、本地开发](#九本地开发)
- [十、测试](#十测试)
- [十一、安全设计](#十一安全设计)
- [十二、已知限制](#十二已知限制)

---

## 一、技术栈与架构

| 部分 | 选型 |
|---|---|
| 前端 | Nuxt 4 + fuxsto-design + Tailwind CSS v4 |
| 后端 | Go（标准库 `net/http`）+ pgx/v5 |
| 数据库 | PostgreSQL 17 |
| 认证 | GitHub OAuth **与** 本地密码（bcrypt）并存 |
| 部署 | **单个二进制**（前端产物 `go:embed` 内嵌） |

### 运行时拓扑（生产）

```
浏览器 ──HTTPS──> Nginx
                    │ 全部请求
                    ▼
          127.0.0.1:3727  boxli-hub（唯一进程）
                    ├── /api/*     Go 原生处理
                    ├── /_nuxt/*   go:embed 的静态资源
                    └── 其他        反代 → node 子进程（SSR，随机回环端口）
                                        │
                                        ▼
                                   PostgreSQL 127.0.0.1:5432
```

**一个进程，两种执行环境。** 前端产物内嵌在 Go 二进制里，但 SSR 仍需由 Node 执行
（Go 无法跑 JavaScript）。所以主进程会拉起一个 node 子进程并在内部反代给它。
详见[第七节](#七前端内嵌与-ssr-托管)。

### 为什么保留 SSR

`index.vue`、`explore/index.vue`、`explore/[ns]/[repo].vue`、`search.vue` 四页
在**服务端**取数（`await useRepoList()` 等），HTML 里就有真实内容和 `<title>`、
`<meta name="description">` —— 这是 SEO 的基础。

纯 SPA（`ssr: false`）会让首屏变成空壳、SEO 归零，因此**否决**。
SSG 也不可行：`nuxt.config.ts` 的 `routeRules` 代理是 Nitro **运行时**行为，
静态导出后不存在。**结论：必须 SSR。**

### 关键约束

| 约束 | 原因 |
|---|---|
| 只存元数据 | `sources.url` 只记录「去哪下载」，不存镜像文件 |
| 只绑回环 | 后端、SSR 子进程、数据库都只听 `127.0.0.1`，对外由 Nginx 暴露 |
| 无环境变量 | 配置只来自 TOML（`--config` 指定路径） |
| 手机优先 | 先写 <640px，再 `md:`/`lg:` |

---

## 二、代码结构

```
backend/
├── cmd/hub/                    # 服务入口（唯一进程）
├── cmd/seed/                   # 种子数据（幂等）
└── internal/
    ├── auth/                   # 会话与 CSRF
    │   ├── auth.go             #   JWT 签发/校验 + sessions 表（存 sid 的 SHA-256）
    │   ├── github.go           #   OAuth：authorize URL / code 换 token / 拉用户
    │   ├── middleware.go       #   鉴权中间件：解析会话 → 注入 context
    │   ├── cookie.go           #   httpOnly 会话 Cookie
    │   ├── origin.go           #   CSRF 来源白名单
    │   ├── state.go            #   OAuth state 一次性校验
    │   └── setup_token.go      #   首次引导令牌（随机 + 恒定时间比较）
    ├── config/config.go        # TOML 加载 + 启动期校验
    ├── db/
    │   ├── db.go               # pgxpool + go:embed 迁移执行器
    │   └── migrations/         # 0001_init / 0002_oauth_states / 0003_local_auth
    ├── hub/                    # HTTP 层
    │   ├── server.go           #   路由 + 中间件链 + 响应封装 + 前端挂载
    │   ├── store.go            #   读查询
    │   ├── store_write.go      #   写事务（整体替换语义）
    │   ├── handlers_read.go    #   读接口
    │   ├── handlers_write.go   #   写接口
    │   ├── handlers_auth.go    #   登录/回调/me/登出
    │   ├── handlers_password.go#   本地密码登录
    │   └── handlers_setup.go   #   首次部署引导
    ├── localauth/localauth.go  # bcrypt 校验 + 首个管理员创建
    ├── seed/seed.go            # 种子数据定义
    └── web/                    # 前端内嵌与 node 子进程托管
        ├── web.go              #   go:embed all:dist + 启动/守护/清理 node
        └── dist/               #   构建产物（gitignore，make frontend 生成）

frontend/
├── app/
│   ├── pages/                  # 15 个页面（含 7 个文档页）
│   ├── components/             # 10 个组件（含 DocsLayout）
│   ├── composables/            # useApi / useAuth / useMarkdown
│   ├── middleware/auth.ts      # 登录守卫（仅客户端）
│   └── layouts/default.vue
├── scripts/
│   ├── build-ssr-bundle.mjs    # esbuild 打包成单个 ssr.mjs
│   └── plugins/                # jsdom 兼容补丁
└── nuxt.config.ts

Makefile                        # 固化「先前端后后端」的构建顺序
.github/workflows/build.yml     # CI：测试 + 构建 linux/amd64 + 发布
```

---

## 三、数据模型

**7 张表**，Hub 只存元数据：

| 表 | 说明 |
|---|---|
| `users` | 用户。`password_hash`（本地密码，可为 NULL）与 `github_id`（OAuth）互不冲突 |
| `repositories` | 镜像元数据，`(namespace, name)` 唯一 |
| `tags` | 单个仓库按 `tag + os + arch` 区分 |
| `sources` | 每个 tag 的多个下载源（`type`/`url`/`priority`/`region`） |
| `sessions` | 会话，存 `sid` 的 **SHA-256**（不存明文，可吊销） |
| `oauth_states` | OAuth 一次性 state（10 分钟 TTL） |
| `schema_migrations` | 迁移记录 |

迁移在启动时**自动执行**（`db.Migrate()`），按文件名排序、记录在 `schema_migrations`、
可重复启动（幂等）。新增迁移就加一个 `0004_xxx.sql`。

`0003_local_auth.sql` 的改动对老数据完全兼容：既有 OAuth 用户
`password_hash` 为 NULL，登录行为不变。

---

## 四、后端实现

### 4.1 中间件链

```go
// server.go
handler := withLogging(s.withCORS(auth.OriginGuard(s.cfg.AllowedOrigins())(mux)))
```

| 顺序 | 中间件 | 作用 |
|---|---|---|
| 1 | `withLogging` | 记录 `method path` |
| 2 | `withCORS` | 处理跨源头 |
| 3 | `OriginGuard` | 写请求校验 `Origin` 白名单（CSRF 防线） |
| 4 | `mux` | 路由分发 |

> **`OriginGuard` 在白名单为空时放行**（本地/CLI 调试方便）。
> 生产由 `[site] frontend_url` 提供白名单，因此不会空。

### 4.2 统一响应格式

```json
{ "code": 0, "message": "ok", "data": { } }
```

`code = 0` 表示成功；非 0 时 HTTP 状态码与 `code` 一致。

### 4.3 路由分发

Go 1.22+ 的 `ServeMux` 按**模式具体程度**优先匹配，所以 `/api/v1/*` 不会被 `/` 吞掉：

```go
mux.HandleFunc("/api/v1/health", ...)   // 具体路径优先
mux.Handle("/_nuxt/", fileServer)       // 静态资源
mux.HandleFunc("/", ssrOrFallback)      // 兜底 → SSR
```

### 4.4 写接口：整体替换语义

`PUT /api/v1/repos/{ns}/{repo}` 对 **tags 和 sources 是整体替换**，在单个事务内完成
（删旧 + 插新）。

> ⚠️ **前端编辑时必须先拉全量详情回填**，否则未提交的 tag/source 会被**静默删除**。

---

## 五、认证与会话

### 5.1 两种登录方式并存

| 方式 | 凭据 | 适用 |
|---|---|---|
| GitHub OAuth | `github_id` | 普通用户 |
| 本地密码 | `password_hash`（bcrypt） | 首次引导创建的管理员；不想用 GitHub 的部署 |

共用 `users` 表。`password_hash` 为 NULL 的账号不能用密码登录，反之亦然。

### 5.2 会话存储：JWT + sessions 表双写

- JWT（HS256）承载 `uid` / `sid` / 过期时间，**无状态校验签名**
- `sessions` 表存 `sid` 的 **SHA-256**，**有状态校验是否被吊销**

两者都要通过。这样既有 JWT 的效率，又能真正登出（删表行即失效）。

### 5.3 会话交付：httpOnly Cookie

| 属性 | 值 | 原因 |
|---|---|---|
| `HttpOnly` | 是 | 防 XSS 读取 |
| `SameSite` | `Lax` | 防 CSRF（配合 OriginGuard） |
| `Secure` | 由 `cookie_secure` 决定 | HTTPS 下必须 true；http 下必须 false，否则浏览器直接丢弃 |

`SessionTokenFrom` **优先读 Cookie，其次读 `Authorization: Bearer`** —— 后者供 CLI/curl 调试。

### 5.4 CSRF 防护

Cookie 方案的必要配套，两层：

1. `SameSite=Lax` —— 跨站写请求不携带 Cookie
2. `OriginGuard` —— 校验 `Origin` 与白名单**精确相等**（scheme + host + port，用
   `url.Parse` 解析后比对，防 `evil.com` 伪造成 `boxli.dev.evil.com`）

> CLI/curl 不带 `Origin` 时放行（不是浏览器发起的请求，不存在 CSRF 前提）。

### 5.5 开放重定向防护

OAuth 回跳落点经 `safeRedirectPath` 校验：只允许站内相对路径，
`//evil.com`、`http://evil.com` 一律拒绝，回落到 `/dashboard`。

### 5.6 OAuth 流程

```
/login 点登录 → POST /auth/login 返回 authorize_url
   → 跳 GitHub 授权 → 回 /api/v1/auth/callback?code=&state=
   → 校验并**原子消费** state（DELETE ... RETURNING，防重放）
   → code 换 token → 拉用户 → upsertUser → 下发 Cookie → 302 回前端
```

失败一律 302 到 `/login?error=<code>`，不把内部错误暴露给用户。

### 5.7 `upsertUser` 的账号接管防护

```sql
INSERT INTO users (username, email, avatar_url, github_id) VALUES (...)
ON CONFLICT (github_id) DO UPDATE SET ...
```

username/email 被**另一个** `github_id` 占用时返回 409。

> ⚠️ **绝不能**在冲突时「按 username 退化查找并登录」—— 那会让新 GitHub 账号
> 接管同名老账号。这是已修复的账号接管漏洞。

### 5.8 本地密码（`internal/localauth`）

| 关注点 | 做法 |
|---|---|
| 存储 | bcrypt 加盐哈希 |
| 用户枚举防护 | 「用户不存在」与「密码错误」返回**同一错误**，且账号不存在时**仍执行一次 bcrypt 比较**拉平耗时 |
| 用户名 | 3–32 字符，仅 ASCII 字母/数字/`_`/`-`（避免 URL 路径与日志歧义） |
| 密码 | ≥8 字符、≤**72 字节**、拒绝常见弱口令 |

> **为什么限制 72 字节**：bcrypt 静默忽略第 72 字节之后的内容。不限制的话，
> 用户会以为超长密码更安全，实际强度等同截断后的前缀。

### 5.9 首次部署引导（`/setup`）

启动时若 `users` 表为空，生成一次性令牌并打印到**服务端日志**：

```go
setupToken, _ := auth.NewSetupToken()        // 32 字节随机 → 64 位十六进制
if n, _ := localauth.NewStore(pool).UserCount(ctx); n == 0 {
    srv.EnableSetup(setupToken)
    log.Printf("请在浏览器打开： http://<域名>/setup?token=%s", setupToken)
}
```

**三重防护**：

| 防护 | 防的是什么 |
|---|---|
| 令牌只出现在服务端日志 | 公网访问者看不到日志，无法抢先注册管理员 |
| 计数 + 插入在**同一事务** | 并发请求创建出多个管理员；初始化后被重放 |
| `subtle.ConstantTimeCompare` | 时序侧信道推断令牌 |

**引导页是内嵌的自包含 HTML，不走 Nuxt。** 因为引导发生时 `frontend_url` 可能还是
默认值 —— 依赖 SPA 会让「首次部署」这个最需要可靠的环节变脆弱。

初始化成功后立即 `invalidateSetupToken()`，并因 `users` 非空而永久关闭。

### 5.10 dev 模拟登录

`[dev] enabled = true` 时，`POST /auth/login` 传 `{"github_user":"x"}` 即可登录为任意账号。

> **置为 true 时程序直接拒绝启动**，避免误带入生产。仅用于本地调试。

---

## 六、前端实现

### 6.1 数据获取（`useApi.ts`）

`useFetch` + `transform` 统一解包 `{code,message,data}`，页面直接消费 `data`。

**四页在服务端取数**（SEO 关键）：`index.vue`、`explore/index.vue`、
`explore/[ns]/[repo].vue`、`search.vue`。

### 6.2 登录态（`useAuth.ts`）

`ensure()` 有缓存，避免每个页面重复请求 `/auth/me`。`apiWrite()` 统一封装写请求
（自动带 Cookie、处理 CSRF 与 401）。

### 6.3 登录守卫（`middleware/auth.ts`）

```ts
if (import.meta.server) return   // ← 关键
```

**会话在 httpOnly Cookie 里，SSR 阶段拿不到**（服务端内部请求不会自动携带 Cookie）。
若在 SSR 就跳转，会把已登录用户也误判为未登录。因此登录态**只能在客户端判定**。

### 6.4 手机优先要点

- 交互元素 `min-h-11`（44px）且带 `min-w-11`
- 输入框 `text-base`（16px），避免 iOS 聚焦缩放
- 全局 `min-h-[100dvh]`（不用 `100vh`）与 `overflow-x-hidden`
- 页脚 `pb-[calc(2rem+env(safe-area-inset-bottom))]` 防 iPhone 横条遮挡
- **不依赖 hover**（手机没有 hover）；汉堡菜单用 fuxsto `Drawer`
- 长 URL `break-all`，标签栏 `overflow-x-auto`

### 6.5 README 渲染安全

`markdown-it`（**`html: false`**，禁用原始 HTML）+ `isomorphic-dompurify` 清洗后输出。
外链自动补 `target="_blank" rel="noopener noreferrer nofollow"`。

---

## 七、前端内嵌与 SSR 托管

这是「单二进制部署」的实现所在。

### 构建管线

```
frontend/app/**                      源码
      │ npm run build                （Nuxt）
      ▼
frontend/.output/
      ├── public/                    客户端资源（_nuxt/*.js|css）
      └── server/                    Nitro SSR + node_modules（约 19MB）
                │ npm run build:ssr-bundle   （esbuild --bundle）
                ▼
backend/internal/web/dist/
      ├── public/                    直接拷贝
      └── ssr.mjs                    单文件 15.2MB（gzip 2.0MB，已内联全部依赖）
                │ go:embed all:dist
                ▼
backend/boxli-hub                    33MB 单二进制
```

打包后不再需要 `node_modules`，因此可以整体 embed。

### 必须用 `go:embed all:`

Go 的 `go:embed` **默认忽略**以 `.` 或 `_` 开头的目录，而 Nuxt 的资源目录正是 **`_nuxt/`**：

```go
//go:embed all:dist      // ✅ 包含 _nuxt/
//go:embed dist          // ❌ 静态资源全部丢失
```

`embed_check_test.go` 专门守住这点：断言嵌入文件数非零且存在 `_nuxt`。

> 若 `dist/` **整体缺失**，`go build` 会报 `pattern all:dist: no matching files found`。
> 这是有意保留的失败方式 —— 比静默产出前端不可用的二进制要好。

### SSR 子进程生命周期

`web.StartSSR()`：

1. 读取内嵌的 `ssr.mjs`
2. `os.MkdirTemp`（0600）写入
3. `net.Listen("tcp","127.0.0.1:0")` 取一个空闲端口后关闭，交给 node
4. 启动 node，注入 `NITRO_PORT` / `NITRO_HOST=127.0.0.1` / `NODE_ENV=production`
5. 轮询端口直到可连接（最多 30 秒）

**`SysProcAttr` 的两个关键设置**：

| 设置 | 作用 |
|---|---|
| `Setpgid: true` | 独立进程组，退出时 `kill(-pid, SIGTERM)` 连带清理 node 派生的子进程 |
| `Pdeathsig: SIGTERM` | **内核级兜底**：主进程被 `SIGKILL` 或崩溃时，node 自动终止 |

> **`Pdeathsig` 是实测补上的**：只靠 `defer ssr.Stop()` 时，主进程被 `SIGKILL`
> 会来不及执行 defer，node 变成 `PPID=1` 的孤儿并持续占用端口与内存。
> 加 `Pdeathsig` 后实测 `kill -9` 主进程，node 随之消失。

`Stop()`：SIGTERM 整个进程组 → 等 5 秒 → SIGKILL → 删除临时目录。可安全重复调用。

> ⚠️ **Nitro 读的是 `NITRO_PORT`/`NITRO_HOST`，不是 `PORT`/`HOST`。**
> 用后者设置端口是无效的。

### 静态资源挂载的坑

```go
// Assets() 返回的文件系统根对应 dist/public，其下**就包含 _nuxt/**
mux.Handle(web.PublicPrefix, http.FileServer(http.FS(pub)))
```

**不能加 `http.StripPrefix`** —— URL 路径可直接映射到该 fs；加了会让它去找根下的
`entry.css` 而 404。

### 打包解决的三个 jsdom 兼容问题

`isomorphic-dompurify` 在 Node 下 `import { JSDOM } from 'jsdom'`，而 jsdom 是
CommonJS 且有运行时文件依赖。打成 ESM 后会连续出错：

| 报错 | 原因 | 解法 |
|---|---|---|
| `Dynamic require of "node:fs" is not supported` | jsdom 用动态 require | 注入 `createRequire` 兼容层 |
| `ReferenceError: __dirname is not defined` | esbuild 不提供 CJS 全局量 | banner 补 `__filename`/`__dirname` |
| `ENOENT ... /default-stylesheet.css` | jsdom 用 `__dirname` 相对路径读文件 | 插件**内联为字符串常量** |
| `Cannot find module './xhr-sync-worker.js'` | `require.resolve` 的文件未被打出 | 替换为占位（仅同步 XHR 用，DOMPurify 不用） |

兼容层用**精确正则匹配源码**：jsdom 升级导致语句形式变化时会**直接报错**，而非静默失效。

### 开发 vs 生产

| | 开发 | 生产（单二进制） |
|---|---|---|
| 前端 | `nuxt dev`（热更新） | 内嵌 bundle，主进程托管 |
| API | Nuxt `routeRules` 代理 → 3727 | Go 原生处理 |
| 端口 | 3011 | 仅 3727 |

---

## 八、API 参考

所有接口在 `/api/v1/` 下，响应统一 `{code, message, data}`。

### 读接口（公开）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/health` | 健康检查 |
| GET | `/search?q=&limit=` | 搜索 |
| GET | `/repos?namespace=&limit=&offset=` | 列表 |
| GET | `/repos/{ns}/{repo}` | 详情（含全部 tags 与 sources） |
| GET | `/repos/{ns}/{repo}/tags` | 标签列表 |
| GET | `/repos/{ns}/{repo}/readme` | README 文本 |

> ⚠️ `/health` **恒返回 healthy**，不探测数据库。真正的探活用 `/repos?limit=1`。

### 认证

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/auth/login` | 返回 `authorize_url` |
| POST | `/auth/password` | 本地用户名 + 密码登录 |
| GET | `/auth/callback?code=&state=` | 校验并消费 state → 302 回前端 |
| GET | `/auth/me` | 当前用户（Cookie 或 Bearer） |
| POST | `/auth/logout` | 吊销 session 并清 Cookie |

### 首次部署引导

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/setup?token=` | 引导页面（自包含 HTML） |
| POST | `/setup` | 创建首个管理员（需一次性令牌） |

### 写接口（需登录）

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/repos` | 创建（`namespace` 缺省为当前用户名） |
| PUT | `/repos/{ns}/{repo}` | 更新（仅 owner，**整体替换**） |
| DELETE | `/repos/{ns}/{repo}` | 删除（仅 owner，级联） |

### 请求体示例

```jsonc
// POST /repos
{
  "namespace": "alice",
  "name": "my-app",
  "description": "示例镜像",
  "readme": "# 标题\n正文",
  "tags": [
    {
      "tag": "v1.0.0", "os": "linux", "arch": "amd64",
      "sources": [
        { "type": "github", "url": "https://...", "priority": 1, "region": "cn" }
      ]
    }
  ]
}
```

### 错误码

| 码 | 含义 |
|---|---|
| 400 | 参数错误 |
| 401 | 未登录 / 凭据错误 |
| 403 | CSRF 拒绝 / 无权限 / 引导令牌无效 |
| 404 | 资源不存在 |
| 409 | 冲突（用户名被占用、系统已初始化） |
| 500 | 服务端错误 |
| 503 | 未配 `jwt_secret`，或未配 OAuth 且未开 dev 登录 |

---

## 九、本地开发

### 9.1 常用命令

```bash
make build          # 前端打包 + 后端内嵌（生产形态）
make dev-backend    # 起后端（读 backend/hub.toml）
make dev-frontend   # 起前端（3011）
make test / vet / fmt / clean
make help           # 列出全部目标
```

> **必须遵守「先前端后后端」的顺序**，`make build` 已经固化。
> 直接 `cd backend && go build` 时若 `internal/web/dist/` 不存在会编译失败。

### 9.2 数据库

```bash
createdb boxli_hub          # 或用 psql
# 建表不需要单独执行：启动时自动迁移
```

Go 缓存路径（本机沙箱不允许写 `~/.cache` 与 `~/go`）：

```bash
export GOCACHE=$PWD/.devtools/gocache \
       GOMODCACHE=$PWD/.devtools/gomodcache \
       GOPATH=$PWD/.devtools/gopath
```

### 9.3 用 curl 调试

**Bearer 方式**（无需处理 Cookie）：

```bash
# 1. 登录拿 token（需 [dev] enabled = true 且未配 OAuth）
curl -s -X POST localhost:3727/api/v1/auth/login \
  -H 'Content-Type: application/json' -d '{"github_user":"alice"}'

# 2. 带 Bearer 访问
curl -s localhost:3727/api/v1/auth/me -H "Authorization: Bearer <token>"

# 3. 写请求（CLI 不带 Origin 即放行 CSRF）
curl -s -X POST localhost:3727/api/v1/repos \
  -H "Authorization: Bearer <token>" -H 'Content-Type: application/json' \
  -d '{"name":"demo","tags":[]}'
```

**Cookie 方式**（模拟浏览器，`Origin` 必须是白名单里的值）：

```bash
curl -s -c jar.txt -X POST localhost:3011/api/v1/auth/login \
  -H 'Content-Type: application/json' -H 'Origin: http://localhost:3011' \
  -d '{"github_user":"alice"}'
curl -s -b jar.txt localhost:3011/api/v1/auth/me
```

### 9.4 常见问题

| 症状 | 原因 |
|---|---|
| `pattern all:dist: no matching files found` | 没先构建前端。跑 `make frontend` |
| 页面 200 但静态资源 404 | `go:embed` 少了 `all:` 前缀（见第七节） |
| 登录成功但仍显示未登录 | `cookie_secure` 与协议不匹配（http 下必须 false） |
| 写请求 403 | `Origin` 不在 `[site] frontend_url` 白名单里 |
| `/docs` 返回 500 | jsdom 补丁失效（jsdom 升级了），看第七节的表格 |
| 前端起不来 | 端口冲突，换一个（见下） |

> **端口 3000 被 Forgejo 占用**（`forgejo.service`，系统服务）。前端开发用 **3011**，
> 并同步改 `[site] frontend_url`。排查占用：
>
> ```bash
> ss -ltnpe | grep :3000     # 读 uid:/cgroup: 判断归属
> ```

---

## 十、测试

```bash
make test                              # 全部
cd backend && go test ./internal/web/ -v   # 单个包
```

**31 个测试函数，5 个包**：

| 包 | 数量 | 覆盖 |
|---|---|---|
| `config` | 10 | 默认值、未知键拒绝、校验、日志脱敏、`frontend_url` 规范化 |
| `auth` | 9 | 来源白名单、OriginGuard、Cookie 属性、state 单次消费/伪造/过期、开放重定向、令牌随机性与恒定时间比较 |
| `hub` | 7 | 引导页自包含性、令牌校验（含前后缀不匹配）、作废、方法限制 |
| `localauth` | 3 | 用户名校验（含中文/emoji/路径字符拒绝）、密码强度、错误类型 |
| `web` | 2 | **embed 完整性**（含 `_nuxt`）、SSR bundle 体积 |

> `internal/auth` 的 state 测试需要数据库（读 `hub.toml` 的 `[db] url`），
> 连不上会自动 **skip**，不影响无库环境。
>
> `web` 包的两个测试断言的是**真实构建产物**。CI 的 test job 不做前端构建，
> 因此会 `-skip` 它们、改在 build job（前端构建完成后）执行。

### 尚未验证（不得视为已验收）

- **真实 GitHub OAuth 端到端** —— 需人工点一次授权
- **手机端真机 / Lighthouse** —— 现有仅为静态断言（viewport、`100dvh`、44px 触控类名）
- **生产服务器实际部署**

---

## 十一、安全设计

| 威胁 | 防护 |
|---|---|
| XSS 读取会话 | Cookie `HttpOnly` |
| CSRF | `SameSite=Lax` + `OriginGuard` 精确比对 |
| 开放重定向 | `safeRedirectPath` 只允许站内相对路径 |
| OAuth state 重放 | `DELETE ... RETURNING` 原子消费 |
| 账号接管 | `upsertUser` 冲突时返回 409，**不退化查找** |
| README XSS | `markdown-it(html:false)` + DOMPurify |
| 用户名枚举 | 统一错误信息 + bcrypt 耗时拉平 |
| 管理员抢注 | 一次性令牌（仅日志可见）+ 事务内判定 |
| 请求体过大 | `http.MaxBytesReader`：写接口 1 MiB，登录接口 64 KiB |
| 密钥泄漏 | 配置摘要日志脱敏（`StringRedactsSecrets` 测试守住） |
| 孤儿进程 | `Setpgid` + `Pdeathsig` |

---

## 十二、已知限制

| 限制 | 说明 |
|---|---|
| **无限流** | 登录/写接口可被暴力刷，尚未实现 rate limit |
| **无请求 ID** | 日志只有 `method path`，无法串联单次请求 |
| **`/health` 不探数据库** | 恒返 healthy，数据库挂了监控不会告警 |
| **管理员无后台** | `is_admin` 仅作标记，没有管理界面 |
| **无改密码 UI** | `SetPassword()` 已实现，但未暴露接口与页面 |
| **无备份脚本** | 需自行配置（见 `DEPLOYMENT.md`） |
| **`BOXLI_DATA_DIR` 为死代码** | 从未被读取；若将来接入 blob 存储需重新实现 |

> 立项时的分阶段计划存档在 `plants.md`。其中的规格**已多次被推翻**，
> 不作为现行依据 —— 本仓库的三份文档才是。
