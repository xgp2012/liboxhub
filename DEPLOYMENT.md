# Boxli Hub 部署文档

> 本文档基于**对代码的实际核查**编写，不是照抄规格书。
> 凡与 `plants.md` 第九章至第十一章冲突之处，**以本文档为准**（差异原因逐条标注）。
>
> 核查日期：2026-10-03 · 核查方式：构建实测 + 全栈运行实测 + 源码通读
>
> **2026-10-03 重大变更**：前端产物已内嵌进后端二进制（`go:embed`），
> 部署形态从「两进程 + 两 systemd unit」简化为**一个二进制 + 一个配置文件**。
> 前端 systemd unit 已**不再需要**。详见第一节与第四节。

---

## 目录

- [零、部署前必读：三个坑](#零部署前必读三个坑)
- [一、架构与端口](#一架构与端口)
- [二、构建（实测通过）](#二构建实测通过)
- [三、数据库准备](#三数据库准备)
- [四、systemd（单服务）](#四systemd单服务)
- [五、首次部署引导](#五首次部署引导)
- [六、Nginx 与 HTTPS](#六nginx-与-https)
- [七、上线前安全清单](#七上线前安全清单)
- [八、验收步骤](#八验收步骤)
- [九、备份与恢复](#九备份与恢复)
- [十、本机无 sudo 的备用方案](#十本机无-sudo-的备用方案)
- [十一、排障速查](#十一排障速查)
- [十二、已知缺口](#十二已知缺口)

---

## 零、部署前必读：三个坑

### 坑 1：`plants.md` 里的 systemd `ExecStart` 是错的

`plants.md:566` 写的是：

```ini
ExecStart=/usr/local/bin/boxli hub serve \
    --addr 127.0.0.1:3727 \
    --data-dir /var/lib/boxli/hub \
    --db "postgres://..."
```

**这些写法在代码里全都不成立**（本文档已按 2026-10-03 改造后的实际代码重写）：

| 文档写法 | 实际情况 | 核查证据 |
|---|---|---|
| `boxli hub serve` 子命令 | 不存在。二进制**只解析一个参数 `--config`** | `cmd/hub/main.go` 仅注册 `flag.String("config", ...)`；实测 `--help` 只列出 `-config` |
| `--addr` / `--data-dir` / `--db` | 不存在。配置**全部来自 TOML 文件** | 见 `backend/internal/config/config.go` 的 `Load()`，不读任何环境变量 |
| `--data-dir /var/lib/boxli/hub` | `DataDir` 是**死代码**，且未迁移到 TOML | 该字段全仓库无任何地方读取，故配置里不再暴露 |

**✅ 正确做法**：`ExecStart=/usr/local/bin/boxli-hub --config /etc/boxli-hub/hub.toml`，
配置写在该 TOML 文件里（见[第四节](#四后端-systemd)）。

> ⚠️ `/usr/local/bin/boxli` 是**另一个程序**，不是本项目。`DEVELOPMENT.md:504` 自己承认：
> 本机 `boxli` CLI 对接的是「另一套 Hub（`boxli hub serve`：用户名/密码 + blob 存储）」。
> 实测 `boxli login --hub http://127.0.0.1:3727` 返回 `hub 404 Not Found`。
> **不要把本项目的二进制命名或安装成 `boxli`**，避免与既有 CLI 冲突。

### 坑 2：前端必须用 SSR，不能用 SSG

`plants.md` 第十一章给了「方式二：纯静态生成（SSG）」，**这条路走不通**：

- `frontend/nuxt.config.ts:16` 有 `routeRules: { '/api/**': { proxy: 'http://127.0.0.1:3727/api/**' } }` —— 这是 **Nitro 运行时**代理。SSG 只产出静态文件，没有 Node 服务，这个代理**根本不存在**，所有 API 请求会 404。
- 首页与详情页用了 `await useRepoList()` / `await useRepoDetail()`（`index.vue:105`、`explore/[ns]/[repo].vue:100`），是**服务端取数**。静态生成时若后端不在线，会把错误状态**烙进 HTML**。

**✅ 正确做法**：用 `npm run build`（preset = `node-server`），保留 SSR。

> **2026-10-03 补充**：SSR 现在是**内嵌**在 Go 二进制里执行的（见第一节），
> 但对「必须保留 SSR」这一结论没有影响 —— SEO 依赖服务端渲染的 HTML。
> 曾评估过 `ssr: false`（纯 SPA），**因会丢失 SEO 而否决**。

### 坑 3：`[site] frontend_url` 身兼两职，设错会同时坏两件事

它**既是** OAuth 成功后的 302 回跳落点，**又是**写接口的 CSRF 来源白名单
（`config.go` 的 `AllowedOrigins()`）。生产环境必须是 `https://boxli.dev`。

实测验证该机制确实生效：

```
POST 带 Origin: 白名单内  -> 401（通过 CSRF，仅缺登录）   ✅
POST 带 Origin: evil.com  -> 403（CSRF 拦截）             ✅
```

> **单二进制下要注意**：前后端由**同一个端口**提供，因此 `frontend_url`
> 应填**对外访问地址**（如 `https://boxli.dev`），而不是 `127.0.0.1:3727`。
> 只有在本地直连调试时才填回环地址。

---

## 一、架构与端口

**一个进程 + 一个数据库**，后端与数据库**只绑回环**，外部不可直达：

| 组件 | 形态 | 监听 | 代码位置 |
|---|---|---|---|
| 后端 + 前端 | **单个 Go 二进制**（内嵌前端产物） | `127.0.0.1:3727` | `backend/cmd/hub/main.go` |
| ├─ 静态资源 | `go:embed`，Go 直接伺服 | 同上 | `backend/internal/web/web.go` |
| ├─ API | Go 原生处理 | 同上 | `backend/internal/hub/server.go` |
| └─ SSR 渲染 | node 子进程（内嵌 bundle 释放后执行） | `127.0.0.1:<随机端口>` | `backend/internal/web/web.go` |
| 数据库 | PostgreSQL | `127.0.0.1:5432` | `backend/internal/db/db.go` |

请求链路：

```
浏览器 ──HTTPS──> Nginx :443 ──proxy──> 127.0.0.1:3727  (唯一的 boxli-hub 进程)
                                            ├── /_nuxt/*  静态资源，Go 直接返回
                                            ├── /api/*    Go 原生处理
                                            └── 其他      → 内部反代到 node 子进程（SSR）
```

**与旧架构的差异**：

| | 旧（两进程） | 新（单二进制） |
|---|---|---|
| 进程数 | 2（Go + Node） | 1（Go 托管 Node 子进程） |
| systemd unit | 2 个 | **1 个** |
| 部署文件 | 二进制 + `.output/` 目录（1427 个文件） | **二进制 + hub.toml** |
| Nginx location | 2 条（`/api/` 与 `/`） | **1 条**（全部转 3727） |
| Node 依赖 | 需要（跑前端） | **仍需要**（SSR 必须由 Node 执行） |
| 端口 | 3727 + 3000 | 仅 3727（SSR 用随机内部端口） |

> **为什么仍需要 Node**：Go 无法执行 JavaScript。实测纯 Go 引擎 `goja`
> 会在打包产物的**第一个正则字面量**上 panic（不支持 ES2020+ 正则 `u` 标志），
> 而 Nitro/Vue 产物必然包含这类语法。Node SEA（`--experimental-sea-config`）
> 也不可行——实测其 blob 仅 429 字节，`main` 只是路径引用，不内联依赖。
> 因此 Node 是**运行时依赖**，但前端**产物**完全内嵌，无需任何外部文件。

**端口分配（生产机）**：

| 端口 | 用途 | 对外暴露 |
|---|---|---|
| 80 / 443 | Nginx | ✅ 公开 |
| 3727 | boxli-hub（API + 静态资源 + SSR 入口） | ❌ 仅回环 |
| 随机高位端口 | boxli-hub 内部的 node SSR 子进程 | ❌ 仅回环（程序自动分配） |
| 5432 | PostgreSQL | ❌ 仅回环 |

> **⚠️ 本开发机特例**：本机 `:3000` 已被 **Forgejo**（`forgejo.service`，uid 115）长期占用。
> 单二进制架构下**不再需要给前端单独分配端口**，因此这个冲突自然消失。
> 但**本地开发模式**（`make dev-frontend`，即 `nuxt dev`）仍需要端口，
> 请使用 **3011**，并把 `hub.toml` 的 `[site] frontend_url` 同步改成 `http://localhost:3011`。

---

## 二、构建（实测通过）

### 2.1 一条命令（推荐）

仓库根目录提供了 `Makefile`，**必须按「先前端、后后端」的顺序构建**：

```bash
make build        # = make frontend + make backend
```

**实测结果**：产物 `backend/boxli-hub`，**33 MB** 单二进制（含内嵌前端）。

> ⚠️ **不要跳过前端直接 `go build`**。
> `go:embed` 在**编译期**读取 `backend/internal/web/dist`：
> 该目录缺失时编译会**直接报错**（`pattern all:dist: no matching files found`），
> 这是有意设计 —— 宁可明确失败，也不要产出一个前端全 404 的二进制。

### 2.2 前端（`make frontend` 做的事）

```bash
cd frontend
npm ci
npm run build              # Nuxt 构建 → .output/
npm run build:ssr-bundle   # esbuild 打包 → backend/internal/web/dist/
```

构建分两步，原因：

1. `npm run build` 产出 Nuxt 的 `.output/`，其中 `server/` 依赖同目录
   `node_modules/`（**约 19 MB，含 8.6 MB jsdom**），共 1427 个文件。
2. `build:ssr-bundle` 用 esbuild 把 SSR 服务端**内联成单个 `ssr.mjs`**
   （实测 **15.2 MB / gzip 2.0 MB**），并把客户端静态资源拷到 `dist/public/`。
   打包后不再需要 `node_modules`，可整体 `go:embed`。

**实测（2026-10-03）**：打包产物在**空目录**下可直接 `node ssr.mjs` 运行，
且 SSR 渲染结果与未打包时**完全一致**（首页 12637 字节、`/docs` 16117 字节）。

> **打包脚本处理的两个 jsdom 兼容问题**（见 `frontend/scripts/`）：
> 1. jsdom 是 CommonJS，用动态 `require('node:fs')` 与 `__dirname`。
>    esbuild 打成 ESM 后这些无法解析 → 注入 `createRequire` 兼容层。
> 2. jsdom 在加载时用 `__dirname` 相对路径读 `default-stylesheet.css`，
>    并用 `require.resolve` 定位 `xhr-sync-worker.js`。
>    单文件后这两个路径失效 → 前者**内联为字符串常量**，后者替换为占位
>    （该 worker 仅用于同步 XHR，DOMPurify 不使用 XHR）。

### 2.3 后端（`make backend` 做的事）

```bash
cd backend
go build -o boxli-hub ./cmd/hub
```

- 模块名：`github.com/LiStudioorg/boxli`
- Go 版本要求：`go 1.27`
- 依赖：`pgx/v5 v5.11.0`、`golang-jwt/jwt/v5 v5.3.1`、`golang.org/x/crypto`（bcrypt）

> **服务器需要两个运行时**：拷过去的二进制**不需要 Go**，
> 但**需要 Node.js**（用于 SSR）。实测环境 node v24.21.0。
> 若未安装 node，程序启动即报错并给出提示（不会静默降级）。

### 2.4 其他 make 目标

```bash
make test         # go test ./...
make vet          # go vet ./...
make clean        # 清理二进制与前端产物
make dev-backend  # 本地起后端（读取 backend/hub.toml）
make dev-frontend # 本地起前端（端口 3011，勿用 3000）
```

---

## 三、数据库准备

### 3.1 建库建用户

```bash
sudo -u postgres psql -c "CREATE USER boxli WITH PASSWORD '<强随机密码>';"
sudo -u postgres psql -c "CREATE DATABASE boxli_hub OWNER boxli;"
```

生成随机密码：

```bash
openssl rand -base64 24
```

### 3.2 迁移是自动的

**不需要单独执行迁移命令。** `backend/cmd/hub/main.go` 启动时会调用 `db.Migrate()`，
按文件名顺序执行内嵌的 `migrations/*.sql`，并记录在 `schema_migrations` 表（幂等，可重复启动）。

已有迁移：

| 文件 | 内容 |
|---|---|
| `0001_init.sql` | users / repositories / tags / sources / sessions |
| `0002_oauth_states.sql` | OAuth state（一次性凭证） |
| `0003_local_auth.sql` | `users.password_hash`、`users.is_admin`、用户名大小写不敏感唯一索引 |

> **`0003` 说明**：新增 `password_hash`（可为 NULL）与 `is_admin`（默认 false），
> 对既有数据完全兼容 —— 老的纯 OAuth 用户 `password_hash` 为 NULL，仍只能走 GitHub 登录。

### 3.3 种子数据（可选）

仅供演示。**生产环境通常不需要**：

```bash
cd backend
# 连接串取自 hub.toml 的 [db] url；也可用 --config 指向生产配置
go run ./cmd/seed --config /etc/boxli-hub/hub.toml
```

种子内容：5 个示例镜像 / 10 个标签 / 17 个下载源。

---

## 四、systemd（单服务）

### 4.1 安装二进制

```bash
sudo install -m 0755 boxli-hub /usr/local/bin/boxli-hub
sudo useradd --system --home /var/lib/boxli --shell /usr/sbin/nologin boxli
sudo mkdir -p /var/lib/boxli
sudo chown boxli:boxli /var/lib/boxli
```

> 命名用 **`boxli-hub`** 而非 `boxli`，避免与既有 boxli CLI 冲突（见[坑 1](#坑-1plantsmd-里的-systemd-execstart-是错的)）。

**只需要这一个文件**。前端产物已经内嵌在里面（33 MB），
**不需要**再拷贝 `frontend/.output/` 或 `node_modules/`。

### 4.2 配置文件

`/etc/boxli-hub/hub.toml`：

```toml
# ---- 监听与数据库 ----
[server]
addr = "127.0.0.1:3727"

[db]
url = "postgres://boxli:<强随机密码>@127.0.0.1:5432/boxli_hub?sslmode=disable"

# ---- 会话（jwt_secret 必填，否则认证接口 503）----
[session]
# 生成：openssl rand -hex 32
jwt_secret = "<64位十六进制>"
ttl_hours = 720
# ---- 生产 HTTPS 必须为 true ----
cookie_secure = true

# ---- GitHub OAuth（可选；不用 OAuth 可整段留空）----
[github]
client_id = "<Client ID>"
secret = "<Secret>"
redirect = "https://boxli.dev/api/v1/auth/callback"

# ---- 回跳落点 + CSRF 白名单（两者共用，必须是**对外**访问地址）----
[site]
frontend_url = "https://boxli.dev"

# ---- 内嵌前端（SSR）运行参数 ----
[frontend]
# node_path = "/usr/bin/node"     # 留空则从 PATH 查找
# allow_missing_node = false      # 默认 false：找不到 node 就拒绝启动

# ---- 生产必须保持关闭（设为 true 时程序直接拒绝启动）----
[dev]
enabled = false
```

权限：

```bash
sudo mkdir -p /etc/boxli-hub
sudo chown root:root /etc/boxli-hub/hub.toml
sudo chmod 600 /etc/boxli-hub/hub.toml
```

> 完整字段说明见 `backend/hub.toml.example`（含逐项注释）。

> **为什么 `cookie_secure = true` 是硬要求**：会话 Cookie 承载 JWT（`cookie.go`）。
> 生产 HTTPS 下设 `false`，Cookie 会在明文信道传输；本地 `http://localhost` 下设 `true`，
> 浏览器会**直接丢弃** Cookie，症状是「登录看似成功但始终显示未登录」。
> 两者不匹配时**程序会在启动阶段直接报错退出**，不会带病运行。

> **`[site] frontend_url` 在单二进制下怎么填**：前后端同端口，
> 填**对外访问地址**（`https://boxli.dev`）。它同时是 CSRF 白名单，
> 填错会导致写接口 403。

### 4.3 unit 文件

`/etc/systemd/system/boxli-hub.service`：

```ini
[Unit]
Description=Boxli Hub (single binary: API + SSR frontend)
Documentation=https://boxli.dev/docs
After=network.target postgresql.service
Wants=postgresql.service

[Service]
Type=simple
User=boxli
Group=boxli
# ⚠️ 不要在这里加 --addr/--db/--data-dir，二进制不解析这些参数（见坑 1）
# 支持的参数只有：--config、--no-ssr
ExecStart=/usr/local/bin/boxli-hub --config /etc/boxli-hub/hub.toml
Restart=always
RestartSec=5

# 加固
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/boxli

[Install]
WantedBy=multi-user.target
```

启用：

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now boxli-hub
sudo systemctl status boxli-hub
sudo journalctl -u boxli-hub -n 50 --no-pager
```

**成功日志应包含**（首次部署还会多出引导提示，见第五节）：

```
loaded config{path=/etc/boxli-hub/hub.toml addr=127.0.0.1:3727 db=postgres://boxli:***@... frontend=https://boxli.dev cookie_secure=true ttl=720h oauth=true dev_login=false}
database migrated
已有 N 个用户，跳过首次部署引导
SSR 子进程已启动 (pid=12345, port=45678)
Listening on http://127.0.0.1:45678
SSR 已就绪 (port=45678)
boxli hub listening on 127.0.0.1:3727
```

> **SSR 端口是随机的**：每次启动由内核分配一个空闲回环端口（日志中的 `port=`），
> 仅监听 `127.0.0.1`，外部不可达。无需在配置或防火墙中固定它。

> **不需要单独的前端 unit**。旧版本要求 `boxli-frontend.service` 跑
> `node .output/server/index.mjs`；现在由主进程托管，前端 unit 已废弃。
> 若你从旧版本升级，记得 `sudo systemctl disable --now boxli-frontend` 并删除其 unit 文件。

> **`HOST` 暴露问题已消除**：旧架构中 Nitro 默认监听 `[::]`（所有接口），
> 漏设 `HOST=127.0.0.1` 会让前端**直接暴露在公网**并绕过 Nginx。
> 现在端口由程序内部固定绑 `127.0.0.1`，不存在这个配置失误的可能。

### 4.4 进程管理与清理

程序退出时会：

1. 向 node 子进程所在**进程组**发 SIGTERM（覆盖 node 可能派生的子进程）
2. 若 5 秒内未退出，改发 SIGKILL
3. 删除释放 SSR bundle 的临时目录（`/tmp/boxli-ssr-*`）

此外设置了 `Pdeathsig`：即使主进程被 `SIGKILL` 强杀（来不及执行清理逻辑），
内核也会自动终止 node 子进程，**不会留下孤儿进程占着端口和内存**。

---

## 五、首次部署引导

首次启动时，若数据库中**还没有任何用户**，程序会生成一次性安装令牌并打印到日志：

```
────────────────────────────────────────────────────────────
 首次部署引导已启用：系统中还没有任何用户
 请在浏览器打开： http://<你的域名>/setup?token=3e942af8d100ef...（64 位十六进制）
 该令牌仅在本次启动且系统无用户时有效，初始化后立即作废。
────────────────────────────────────────────────────────────
```

打开该地址即可在网页上完成初始化：

1. **建表** —— 启动时已自动完成（`db.Migrate()`，幂等）
2. **创建管理员** —— 填写用户名与密码，提交即可

### 5.1 为什么需要令牌（安全设计）

引导页若对公网开放，**任何人都能在你之前抢注管理员账号**。因此设置了三重防护：

| 防护 | 说明 |
|---|---|
| 一次性安装令牌 | 32 字节随机（64 位十六进制），**只出现在服务端日志**。公网访问者看不到日志，无法抢先注册 |
| 仅当无用户时可用 | 计数与插入在同一事务内完成，并发请求也无法创建出两个管理员 |
| 恒定时间比较 | 令牌比较用 `subtle.ConstantTimeCompare`，避免时序侧信道推断 |

实测验证（本地全栈）：

```
GET  /setup                     -> 200（页面可取）
POST /api/v1/setup 无令牌        -> 403
POST /api/v1/setup 错误令牌      -> 403
POST /api/v1/setup 弱口令        -> 400（"不能包含常见弱口令"）
POST /api/v1/setup 正确令牌      -> 200（创建成功）
POST 再次使用同一令牌            -> 403（已作废）
重启后                          -> 日志显示"已有 1 个用户，跳过首次部署引导"
```

### 5.2 密码与用户名规则

| 项 | 规则 |
|---|---|
| 用户名 | 3–32 字符，仅限**英文字母、数字、下划线、连字符** |
| 密码 | 至少 8 字符，不超过 72 字节，且不能包含常见弱口令 |

用户名限制字符集是为了避免在 URL 路径（`/repos/{ns}/...`）与日志中引入歧义。
密码上限 72 字节是因为 **bcrypt 只使用前 72 字节**，超出部分会被静默截断 ——
限制它可避免用户误以为超长密码更安全。

密码用 **bcrypt 加盐哈希**存储，不可逆。登录失败时，「用户不存在」与「密码错误」
返回**完全相同**的错误信息，且都会执行一次 bcrypt 比较以拉平耗时，
防止通过错误信息或响应时间**枚举系统内已存在的用户名**。

### 5.3 管理员权限范围

管理员目前**只是一个 `is_admin` 标记**，与普通用户的差别在于：

- 普通用户：可提交、编辑、删除**自己**的镜像
- 管理员：额外带 `is_admin = true`

> **当前没有后台管理界面**（不提供封禁用户、删除他人仓库等操作）。
> 这是有意的范围限定 —— 先保证「首个账号可用」，管理功能后续按需增加。

### 5.4 与 GitHub OAuth 的关系

两种登录方式**并存**，共用同一张 `users` 表：

| 场景 | 行为 |
|---|---|
| 本地密码账号 | `password_hash` 有值，可走 `/api/v1/auth/password` 登录 |
| 纯 OAuth 账号 | `password_hash` 为 NULL，**不能**用密码登录 |
| 首次引导创建的管理员 | 使用本地密码 |

因此**不使用 GitHub OAuth 也完全可用**：把 `[github]` 整段留空即可，
引导创建的管理员通过用户名密码登录。

> 迁移 `0003` 对既有数据完全兼容：老用户的 `password_hash` 为 NULL，
> 登录行为不变。

---

## 六、Nginx 与 HTTPS

`plants.md:482` 的配置**大体可用**，但有三处必须修正：

| 问题 | `plants.md` 写法 | 修正 |
|---|---|---|
| 前端托管方式 | `root /var/www/boxli/frontend` + `try_files ... /index.html`（静态） | SSR 下不能走静态托管，应 `proxy_pass` 给应用 |
| 反代目标 | 一条转后端、一条转前端（两个进程两个端口） | 单二进制下**合并为一条** `location /` → `127.0.0.1:3727` |
| `/docs/` 目录 | `location /docs/ { root /var/www/boxli/docs; }` 指向独立静态目录 | **删除**。文档站就是 Nuxt 的 `/docs` 路由（`frontend/app/pages/docs/`），随 SSR 一起提供，没有独立静态目录 |

`/etc/nginx/sites-available/boxli`：

```nginx
# HTTP -> HTTPS
server {
    listen 80;
    listen [::]:80;
    server_name boxli.dev www.boxli.dev;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name boxli.dev www.boxli.dev;

    ssl_certificate     /etc/letsencrypt/live/boxli.dev/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/boxli.dev/privkey.pem;
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         HIGH:!aNULL:!MD5;

    # 安全响应头
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    client_max_body_size 2m;

    # ===== 全部请求转给 boxli-hub（API + 静态资源 + SSR 都在里面）=====
    #
    # 单二进制架构下只需**一条 location**：
    #   /api/*     → Go 原生处理
    #   /_nuxt/*   → Go 从内嵌资源直接返回
    #   其他       → Go 内部反代给 node 子进程做 SSR
    # 旧版本需要的第二条指向 :3000 的 location 已不再需要。
    location / {
        proxy_pass http://127.0.0.1:3727;
        proxy_http_version 1.1;

        proxy_set_header Host              $host;
        proxy_set_header X-Real-IP         $remote_addr;
        proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        proxy_connect_timeout 10s;
        proxy_read_timeout    60s;
        proxy_send_timeout    60s;

        proxy_hide_header X-Powered-By;
    }

    # ===== 带内容 hash 的静态资源可长缓存 =====
    # 文件名含 hash（如 entry.CTbtbonZ.css），内容变化必然换名，
    # 因此可以安全地长期缓存。
    location /_nuxt/ {
        proxy_pass http://127.0.0.1:3727;
        proxy_set_header Host $host;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

> **⚠️ 不要在 Nginx 层加 `add_header Access-Control-*`**。后端的 CSRF 防线依赖
> `Origin` 头原样传递，且 Cookie 请求**不允许** `Access-Control-Allow-Origin: *`。
> 同源反代后根本不产生跨源请求，CORS 头是多余的。

启用与证书：

```bash
sudo ln -s /etc/nginx/sites-available/boxli /etc/nginx/sites-enabled/
sudo rm -f /etc/nginx/sites-enabled/default    # 避免抢占默认站点
sudo nginx -t
sudo systemctl reload nginx

# Let's Encrypt（首次需先有 80 端口可达 + DNS 已解析）
sudo certbot --nginx -d boxli.dev -d www.boxli.dev
sudo systemctl list-timers | grep certbot      # 确认自动续期定时器
```

### 防火墙

```bash
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
sudo ufw status verbose
# 输出中不应出现 3727 或 5432
```

---

## 七、上线前安全清单

| 优先级 | 项 | 原因 |
|---|---|---|
| 🔴 **必做** | **重新生成 `[github] secret`** | 当前值已在开发过程中明文外泄（进入过对话/文件）。到 GitHub → Settings → Developer settings → OAuth Apps → 该应用 → **Regenerate client secret** |
| 🔴 **必做** | 更新 GitHub App 的 **Authorization callback URL** 为 `https://boxli.dev/api/v1/auth/callback` | `github.go` 的 `Exchange()` 会把配置里的 `[github] redirect` 作为 `redirect_uri` 发给 GitHub，与登记值不一致会返回 `redirect_uri_mismatch` |
| 🔴 **必做** | `[session] cookie_secure = true` | 见[4.2](#42-配置文件)；配 `http://` 时程序会拒绝启动 |
| 🔴 **必做** | `[dev] enabled = false` | 开启后 `POST /auth/login` 传 `{"github_user":"x"}` 即可**登录为任意账号**；设为 true 时程序会拒绝启动 |
| 🔴 **必做** | `[session] jwt_secret` 用 `openssl rand -hex 32` 生成，**不要复用开发值** | 复用开发密钥等于把开发环境的会话迁移到生产 |
| 🟡 建议 | `[site] frontend_url = "https://boxli.dev"` | 回跳落点 + CSRF 白名单双职责 |
| 🟡 建议 | 前端显式 `HOST=127.0.0.1` | 见[5.2](#52-unit-文件) |
| 🟡 建议 | Nginx 传递 `X-Forwarded-Proto` | 便于后续判断协议 |

**关于遗留项 L5（CORS）**：Nginx 同源反代后，浏览器不再产生跨源请求，
`withCORS` 中间件自然不再生效，**无需修改代码**。这印证了 `server.go` 中的注释。

---

## 八、验收步骤

### 8.1 后端就绪

```bash
curl -s https://boxli.dev/api/v1/health
# {"code":0,"message":"ok","data":{"status":"healthy"}}
```

> ⚠️ 该接口**恒返回 healthy**，不探测数据库连通性。数据库挂了它依然返回 200。
> 真正有效的探活是下面这条：

```bash
curl -s "https://boxli.dev/api/v1/repos?limit=1" | head -c 200
# 返回 {"code":0,...,"items":[...]} 才说明「后端 + 数据库」整条链路通
```

### 8.2 端口未外泄

```bash
curl -m 5 http://<服务器公网IP>:3727   # 应超时
curl -m 5 http://<服务器公网IP>:5432   # 应超时
sudo ufw status | grep -E '3727|5432'  # 应无输出
```

### 8.3 前端（含 SSR / SEO 验证）

```bash
curl -s -o /dev/null -w "%{http_code}\n" https://boxli.dev/          # 200
curl -s -o /dev/null -w "%{http_code}\n" https://boxli.dev/explore   # 200
curl -s -o /dev/null -w "%{http_code}\n" https://boxli.dev/docs      # 200
```

**必须验证 SSR 确实生效**（这是 SEO 的根基）：

```bash
# 标题与描述必须出现在**服务端返回的 HTML** 里，而不是靠 JS 注入
curl -s https://boxli.dev/ | grep -o '<title>[^<]*</title>'
# 期望：<title>Boxli Hub · 轻量级容器引擎镜像索引</title>

curl -s https://boxli.dev/ | grep -o '<meta name="description"[^>]*>' | head -1
# 期望：能匹配到 content="..."

# 文档站的 Markdown 也应是服务端渲染的（带锚点 id）
curl -s https://boxli.dev/docs | grep -o 'id="[a-z0-9-]*"' | head -3
```

> 若 `<title>` 为空或 HTML 中看不到正文，说明 SSR 没生效
> （浏览器 F12 看到的标题是 JS 执行后的结果，**不能**作为 SSR 生效的证据）。

**静态资源由应用直接伺服**：

```bash
curl -s -o /dev/null -w "%{http_code} %{content_type}\n" \
  "https://boxli.dev$(curl -s https://boxli.dev/ | grep -o '/_nuxt/[A-Za-z0-9_.-]*\.css' | head -1)"
# 期望：200 text/css
```

### 8.4 首次部署引导（首次上线必做）

```bash
# 1. 启动后查看日志中的引导令牌
sudo journalctl -u boxli-hub -n 30 --no-pager | grep -A3 "首次部署引导"

# 2. 无令牌访问应被拒绝（证明公网无法抢注）
curl -s -X POST https://boxli.dev/api/v1/setup \
  -H 'Content-Type: application/json' \
  -d '{"token":"guess","username":"attacker","password":"Attacker123!"}'
# 期望：{"code":403,...}

# 3. 用日志里的令牌在浏览器完成初始化
#    https://boxli.dev/setup?token=<日志中的令牌>

# 4. 用刚创建的账号登录，确认会话可用
#    https://boxli.dev/login
```

### 8.5 OAuth 全链路（若启用了 GitHub 登录，必须人工点一次）

1. 浏览器打开 `https://boxli.dev/login`
2. 点 GitHub 登录 → 应跳转到 **GitHub 授权页**（若报 `redirect_uri_mismatch`，说明第 7 节的 callback URL 没同步）
3. 授权后应回到 `https://boxli.dev/dashboard`，且**显示已登录**
4. 打开 DevTools → Application → Cookies，确认 `boxli_session` 存在且带 **`HttpOnly` + `Secure`** 标记
5. 在 `/submit` 提交一个测试仓库，确认**写请求不被 403**（403 = `[site] frontend_url` 配错）

> **第 3 步失败但第 2 步成功**，通常是 `[session] cookie_secure` 与协议不匹配，
> 或 `[site] frontend_url` 域名与实际访问域名不一致（含 `www.` 前缀差异）。

> 未启用 GitHub OAuth 时跳过本节，用 8.4 创建的本地账号验证登录与写操作即可。

### 8.5 CSRF 防线仍生效

```bash
curl -s -o /dev/null -w "%{http_code}\n" -X POST \
  -H "Origin: https://evil.com" -H 'Content-Type: application/json' \
  -d '{}' https://boxli.dev/api/v1/repos
# 期望 403
```

---

## 九、备份与恢复

仓库内**没有**备份脚本，需自行创建（上线前检查清单要求但未交付，见[第七节](#七上线前安全清单)）。

### 9.1 备份脚本

`/usr/local/bin/boxli-backup.sh`：

```bash
#!/usr/bin/env bash
set -euo pipefail

BACKUP_DIR=/var/backups/boxli
KEEP_DAYS=14
STAMP=$(date +%Y%m%d-%H%M%S)
DB_URL="postgres://boxli:<密码>@127.0.0.1:5432/boxli_hub?sslmode=disable"

mkdir -p "$BACKUP_DIR"
umask 077

pg_dump --dbname="$DB_URL" --format=custom --file="$BACKUP_DIR/boxli_hub-$STAMP.dump"

# 校验产物非空
[ -s "$BACKUP_DIR/boxli_hub-$STAMP.dump" ] || { echo "backup empty!" >&2; exit 1; }

# 清理过期备份
find "$BACKUP_DIR" -name 'boxli_hub-*.dump' -mtime +"$KEEP_DAYS" -delete

echo "backup ok: $BACKUP_DIR/boxli_hub-$STAMP.dump"
```

```bash
sudo chmod 700 /usr/local/bin/boxli-backup.sh
sudo mkdir -p /var/backups/boxli && sudo chmod 700 /var/backups/boxli
```

### 9.2 定时任务

`/etc/cron.d/boxli-backup`：

```cron
# 每日 03:30 备份 Boxli Hub 数据库
30 3 * * * root /usr/local/bin/boxli-backup.sh >> /var/log/boxli-backup.log 2>&1
```

### 9.3 恢复

```bash
# 1. 停后端，避免写入竞争
sudo systemctl stop boxli-hub

# 2. 恢复（--clean 会先删对象再重建）
pg_restore --dbname="postgres://boxli:<密码>@127.0.0.1:5432/boxli_hub?sslmode=disable" \
  --clean --if-exists /var/backups/boxli/boxli_hub-<时间戳>.dump

# 3. 启动
sudo systemctl start boxli-hub
```

> **恢复到全新库**时，先按[第三节](#三数据库准备)建好空库 `boxli_hub` 再执行 `pg_restore`。
> 备份**不含** PostgreSQL 角色与数据库本身，只含表数据与结构。

---

## 十、本机无 sudo 的备用方案

本开发机实测：**sudo 不可用**（`sudo -n true` 失败），因此第四至九节的
systemd / Nginx / ufw / certbot 路径**在本机无法执行**，只能作为**目标服务器**的蓝图。

在目标服务器上按第四节至第九节操作即可。若你**就是想在本机先把整套跑起来**，
用下面的用户态方案（端口换 3011，避开 Forgejo 占用的 3000）。

### 10.1 前置：PostgreSQL

本机 PG 是工作区内的**免安装二进制**，只监听回环：

```bash
PGROOT=/home/xgp2012/hub/.devtools/postgresql-17.11.0-x86_64-unknown-linux-gnu
PGDATA=/home/xgp2012/hub/.devtools/pgdata

$PGROOT/bin/pg_ctl -D "$PGDATA" \
  -o "-c listen_addresses=127.0.0.1 -p 5432 -c unix_socket_directories=/tmp" \
  -l /home/xgp2012/hub/.devtools/pg.log start

# 确认
$PGROOT/bin/psql "postgres://boxli:boxli@127.0.0.1:5432/boxli_hub?sslmode=disable" \
  -tAc "select 'users='||(select count(*) from users)||' repos='||(select count(*) from repositories);"
```

> 注意：本机连接串密码是 `boxli`（开发默认），**仅限本机回环**。

### 10.2 构建

```bash
cd /home/xgp2012/hub/backend
export GOCACHE=/home/xgp2012/hub/.devtools/gocache \
       GOMODCACHE=/home/xgp2012/hub/.devtools/gomodcache \
       GOPATH=/home/xgp2012/hub/.devtools/gopath
go build -o /home/xgp2012/hub/backend/boxli-hub ./cmd/hub

cd /home/xgp2012/hub/frontend
export npm_config_cache=/home/xgp2012/hub/.devtools/npmcache
npm run build
```

### 10.3 启动后端（用 TOML 配置文件，不要用命令行参数）

```bash
cd /home/xgp2012/hub/backend
# 本机演练：确认 hub.toml 里 frontend_url 与实际前端端口一致
nohup ./boxli-hub --config ./hub.toml > /home/xgp2012/hub/hub.log 2>&1 &
```

`hub.toml` 关键项（本机演练用）：

```toml
[site]
frontend_url = "http://127.0.0.1:3011"   # ⚠️ 必须与前端实际端口一致

[session]
cookie_secure = false                     # 本机是 http
```

> **`frontend_url` 必须显式对上前端端口**：本机 3000 是 **Forgejo**，
> 若该值指向 3000，OAuth 回跳会落到 Forgejo，且所有写请求被 CSRF 白名单 403 拒绝。

### 10.4 启动前端

```bash
cd /home/xgp2012/hub/frontend
PORT=3011 HOST=127.0.0.1 nohup node .output/server/index.mjs > /home/xgp2012/hub/frontend.log 2>&1 &
```

### 10.5 验证

```bash
curl -s http://127.0.0.1:3011/api/v1/health
# {"code":0,"message":"ok","data":{"status":"healthy"}}

curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1:3011/          # 200
curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1:3011/docs      # 200
```

以上**全部命令均经实测通过**（含全栈 `GET /api/v1/repos` 返回真实数据的验证）。

### 10.6 停止

```bash
pkill -f 'boxli-hub'
pkill -f '.output/server/index.mjs'
```

> **⚠️ 不要用 `pkill -f boxli`** —— 会误杀既有 boxli CLI 进程。
> **绝不要杀 `:3000`（Forgejo）与 `:5432`（PostgreSQL）**。

---

## 十一、排障速查

### 11.1 端口归属排查（关键技巧）

```bash
ss -ltnpe | grep -E ':(3000|3011|3727|5432)\b'
```

**必须用 `-e`**。原因：`ss -ltnp` 对**其他用户**持有的 socket **不解析 PID**
（不显示 `users:(...)`），容易被误读成「孤儿 socket / 属主不可见」。
`-e` 会额外输出 `uid:` 与 `cgroup:` 字段，据此可确定归属：

```
*:3000          uid:115   cgroup:/system.slice/forgejo.service   ← 系统服务 Forgejo，与本项目无关
127.0.0.1:3727  uid:1001  users:(("boxli-hub",pid=...))          ← 本项目后端
```

### 11.2 常见症状对照

| 症状 | 原因 | 处理 |
|---|---|---|
| 登录后仍显示未登录 | `[session] cookie_secure = true` 但走的是 http；或域名不一致 | 本地设 `false`，生产设 `true`，协议与域名必须匹配（不匹配时启动即报错） |
| OAuth 回跳跳到别的站点 | `[site] frontend_url` 没改 | 设成实际前端地址（本机 `http://127.0.0.1:3011`） |
| 写请求 403 `cross-site request blocked` | 同上，CSRF 白名单未含实际来源 | 同上；多域名用 `[site] extra_origins` |
| 登录报 `redirect_uri_mismatch` | GitHub App 登记值与 `[github] redirect` 不一致 | 两处改成完全相同 |
| `/auth/login` 返回 503 | `[session] jwt_secret` 未设，或 OAuth 未配且 `[dev] enabled = false` | 正确行为。设密钥 / 配 OAuth |
| dev 登录返回 409 | 该 `github_user` 已被占用（L7 账号接管防护） | 换一个未占用的用户名 |
| 前端页面取不到数据，后端日志无请求 | SSR 取数失败，或 `HOST` 绑错 | 先 `curl 127.0.0.1:3727/api/v1/health` 验证后端 |
| 改了 systemd 的 `--addr` 但端口没变 | 二进制不解析该参数（坑 1） | 改 `hub.toml` 的 `[server] addr` |

### 11.3 日志

```bash
sudo journalctl -u boxli-hub -f
sudo journalctl -u boxli-frontend -f
sudo tail -f /var/log/nginx/error.log
```

---

## 十二、已知缺口

以下是**部署相关但代码/仓库中尚未提供**的部分，上线前需自行补齐：

| 缺口 | 说明 | 影响 |
|---|---|---|
| **无限流** | 全仓库 `grep` 确认：无 rate limit 实现（遗留项 L6） | 登录/写接口可被暴力刷 |
| **无请求 ID** | 日志仅 `log.Printf("%s %s", method, path)`（`server.go` 的 `withLogging`） | 无法串联单次请求的链路 |
| **health 不探数据库** | `handleHealth` 恒返 `healthy` | 数据库挂了监控不会告警，需用 `/api/v1/repos` 替代探活 |
| **无备份脚本** | 见[第九节](#九备份与恢复)自建 | 数据丢失无兜底 |
| **无 CI/CD** | 仓库提供了 `Makefile`，但无 workflow / Dockerfile | 构建已可一条命令，但部署仍手工 |
| **数据目录（原 `BOXLI_DATA_DIR`）** | 该字段为死代码，改造 TOML 时未迁移 | 无需设置；若未来接入 blob 存储需重新实现 |
| **无后台管理界面** | 管理员仅有 `is_admin` 标记，不提供封禁/删除他人仓库的界面 | 见[5.3](#53-管理员权限范围)，属有意范围限定 |
| **无改密码界面** | `localauth.SetPassword()` 已实现，但未暴露接口与页面 | 管理员改密码需暂用 SQL 或后续补接口 |
| **阶段 6 手机端验收未完成** | Lighthouse Mobile、真机触控、iOS `100dvh` 等未实测 | 移动端体验未验收 |

---

## 附：快速对照表

| 事项 | `plants.md` 原写法 | ✅ 正确做法 |
|---|---|---|
| 后端启动 | `boxli hub serve --addr ... --db ...` | `ExecStart=/usr/local/bin/boxli-hub --config /etc/boxli-hub/hub.toml` |
| 前端形态 | 方式一 SSR / 方式二 SSG 二选一 | **只能 SSR**（SEO 依赖；SSG 下 `routeRules` 代理不存在） |
| 前端部署 | 单独跑 `node .output/server/index.mjs` | **内嵌进二进制**，由主进程托管 node 子进程 |
| 前端宿主 | `root` + `try_files` 静态 | `proxy_pass`（单二进制下并入 `location /`） |
| systemd unit | 2 个（后端 + 前端） | **1 个**（`boxli-hub.service`） |
| `/docs/` | 独立静态目录 `/var/www/boxli/docs` | 删除，随 Nuxt `/docs` 路由提供 |
| 前端监听 | 必须 `HOST=127.0.0.1`，否则暴露公网 | 程序内部固定绑回环，**已无此配置失误可能** |
| 构建命令 | 手工两步 | `make build`（内含正确顺序） |
| 首次部署 | 未提 | 日志打印一次性令牌 → `/setup?token=...` 创建管理员 |
| 端口排查 | （原文档误判） | `ss -ltnpe` 读 `uid:` / `cgroup:` |
