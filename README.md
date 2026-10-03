# boxli hub

Boxli 官网 + Hub 镜像索引站的 monorepo。

Hub **只收录社区镜像的元数据，不存储镜像本体**——镜像文件放在用户自己的地方（GitHub / Gitee / OSS / S3 / IPFS / BT 等），Hub 只记录「去哪下载」。

- `frontend/` — Nuxt 4 官网与 Hub 前端
- `backend/` — Go 后端（Hub API，只存元数据）
- `plants.md` — 开发说明书（含分阶段计划与进度）
- [`DEVELOPMENT.md`](./DEVELOPMENT.md) — **开发者技术文档**（架构、认证实现、API、调试与安全设计）

## 技术栈

| 部分 | 选型 |
|---|---|
| 前端 | Nuxt 4 + fuxsto-design + Tailwind CSS v4 |
| 后端 | Go（`net/http`）+ pgx/v5 |
| 数据库 | PostgreSQL 17（免安装二进制，见「开发环境」）|
| 运行环境 | **Ubuntu 22.04.5 LTS（原生 Linux，非 WSL）** |

> **环境迁移记录（2026-10-02）**：开发环境由 Windows + WSL2 迁移到原生 Linux。
> 迁移消除了原有的 L1/L2/L8 三项遗留问题（WSL PG 监听全接口、`pg_hba` 全放行、
> Windows 防火墙规则）。当前机器**无 sudo 权限**（沙箱 `NoNewPrivs: 1`），
> 因此数据库采用免安装二进制方案，仅监听 `127.0.0.1:5432`。

## 目录结构（前端）

```
frontend/app/
├── app.vue                    # 根组件（NuxtLayout + NuxtPage）
├── layouts/default.vue        # 全局布局：Header + main + Footer
├── middleware/
│   └── auth.ts                # 登录守卫（仅在客户端判定，见下）
├── components/
│   ├── SiteHeader.vue         # 顶部导航 + 手机汉堡 Drawer + 登录/用户菜单
│   ├── SiteFooter.vue         # 页脚（含 safe-area 处理）
│   ├── RepoCard.vue           # 镜像卡片
│   ├── SourceList.vue         # 多源下载列表（标签切换 + 复制）
│   ├── SourceEditor.vue       # 下载源编辑器（阶段 4，增删改；顺序即优先级）
│   ├── RepoForm.vue           # 建仓/改仓共用表单（阶段 4）
│   ├── CopyButton.vue         # 复制命令按钮
│   ├── EmptyState.vue         # 空状态
│   └── CardSkeleton.vue       # 加载骨架
├── composables/
│   ├── useApi.ts              # 读接口封装（统一解包 {code,message,data}）
│   ├── useAuth.ts             # 登录态 + login/logout + apiWrite（阶段 4）
│   └── useMarkdown.ts         # Markdown 渲染 + DOMPurify 清洗
├── pages/
│   ├── index.vue              # 首页
│   ├── explore/index.vue      # 浏览 + 搜索/排序
│   ├── explore/[ns]/[repo].vue# 镜像详情
│   ├── search.vue             # 搜索
│   ├── login.vue              # 登录（阶段 4）
│   ├── submit.vue             # 提交镜像（阶段 4）
│   ├── dashboard.vue          # 用户中心（阶段 4）
│   ├── docs/                  # 文档站（阶段 5）
│   │   ├── index.vue          #   文档首页
│   │   ├── quickstart.vue     #   快速开始
│   │   ├── install.vue        #   安装与自检
│   │   ├── pull-run.vue       #   拉取与运行
│   │   ├── format.vue         #   镜像格式与下载源
│   │   ├── submit.vue         #   提交镜像到 Hub
│   │   └── faq.vue            #   常见问题
│   └── about.vue              # 关于
├── types/api.ts               # 后端响应类型定义
└── utils/
    ├── format.ts              # 格式化 + 源类型元数据
    └── docs.ts                # 【阶段 5】文档内容与导航结构
```

> **阶段 5 新增组件**：`components/DocsLayout.vue` —— 文档布局
> （桌面 sticky 侧边导航 / 手机折叠目录 / 本页小节锚点 / 上下页翻页）。
> `composables/useMarkdown.ts` 增加标题 `id` 锚点注入（由 `anchors` 选项控制，
> **README 渲染保持原行为**）。

> **⚠️ 为什么登录守卫只在客户端判定**：会话存在 httpOnly Cookie 中，**SSR 期间不会自动携带**
> （服务端内部请求不带浏览器 Cookie）。若在 SSR 就判定，会把**已登录用户也误判为未登录**。
> 因此 `middleware/auth.ts` 首行即 `if (import.meta.server) return`，登录态在 `onMounted` 后拉取。

> **⚠️ 编辑镜像必须整体回填**：`PUT /repos/{ns}/{repo}` 是 **tags/sources 整体替换**语义，
> 所以 `/dashboard` 进编辑时会先单独拉一次详情（列表项不含 tags），用全量数据预填表单；
> 若只提交改动项，其余标签与源会被静默删除。

## 目录结构（后端）

```
backend/
├── cmd/
│   ├── hub/          # 服务入口（监听 127.0.0.1:3727）
│   └── seed/         # 种子数据入口（幂等）
└── internal/
    ├── auth/         # JWT 签发/校验、sessions 表、GitHub OAuth、state 一次性校验
    │                 #  + cookie.go（httpOnly 会话 Cookie）
    │                 #  + origin.go（CSRF 来源白名单校验）
    ├── config/       # 环境变量配置
    ├── db/           # 连接池 + 内嵌 SQL 迁移
    │   └── migrations/
    ├── hub/          # HTTP 路由与处理器（读/写/认证）
    └── seed/         # 种子数据定义与写入
```

## 本地开发

### 1. 数据库（Linux · PostgreSQL 17）

开发机为 Ubuntu 22.04，**无 root/sudo 权限**（沙箱 `no new privileges`），因此 PG 以
**免安装二进制**方式跑在用户目录下，只监听回环：

```bash
# 二进制位置（已下载解压，PG 17.11）—— 注意在**工作区内**的 .devtools/ 下
PGROOT=/home/xgp2012/hub/.devtools/postgresql-17.11.0-x86_64-unknown-linux-gnu
PGDATA=/home/xgp2012/hub/.devtools/pgdata

# 启动（仅 127.0.0.1:5432）
$PGROOT/bin/pg_ctl -D "$PGDATA" \
  -o "-c listen_addresses=127.0.0.1 -p 5432 -c unix_socket_directories=/tmp" \
  -l /home/xgp2012/hub/.devtools/pg.log start

# 停止
$PGROOT/bin/pg_ctl -D "$PGDATA" stop
```

> 若数据目录丢失，重新初始化：
>
> ```bash
> echo 'boxli' > /tmp/pgpw.txt
> $PGROOT/bin/initdb -D "$PGDATA" -U boxli --auth-local=trust \
>   --auth-host=scram-sha-256 --pwfile=/tmp/pgpw.txt -E UTF8 --locale=C
> rm -f /tmp/pgpw.txt
> $PGROOT/bin/psql -h 127.0.0.1 -U boxli -d postgres -c "CREATE DATABASE boxli_hub;"
> ```

连接串：`postgres://boxli:boxli@127.0.0.1:5432/boxli_hub?sslmode=disable`

### 2. 建表 + 种子数据

```bash
cd backend
# Go 缓存需落在工作区内（沙箱不允许写 ~/.cache 与 ~/go）
export GOCACHE=/home/xgp2012/hub/.devtools/gocache GOMODCACHE=/home/xgp2012/hub/.devtools/gomodcache GOPATH=/home/xgp2012/hub/.devtools/gopath
export BOXLI_DB="postgres://boxli:boxli@127.0.0.1:5432/boxli_hub?sslmode=disable"
go run ./cmd/seed
```

后端启动时会自动跑 schema migration（`internal/db/migrations/*.sql`，记录在 `schema_migrations` 表），该命令可**重复执行不报错**（幂等）。

种子内容：5 个示例镜像 / 10 个标签 / 17 个下载源，覆盖 github、gitee、oss、cos、s3、ipfs、magnet、http 等源类型。

### 3. 后端（127.0.0.1:3727）

```bash
cd backend
export GOCACHE=/home/xgp2012/hub/.devtools/gocache GOMODCACHE=/home/xgp2012/hub/.devtools/gomodcache GOPATH=/home/xgp2012/hub/.devtools/gopath
# 设置鉴权密钥（不设则认证接口返回 503，读接口不受影响）
export BOXLI_JWT_SECRET="dev-secret-change-me"
export BOXLI_DB="postgres://boxli:boxli@127.0.0.1:5432/boxli_hub?sslmode=disable"
# ⚠️ 必须与前端实际端口一致（默认值是 3000，而本机 3000 已被 Forgejo 占用）
export BOXLI_FRONTEND_URL="http://localhost:3011"
go run ./cmd/hub
```

### 4. 前端（⚠️ 不能用 3000，见下）

> **⚠️ 本机 3000 端口已被 Forgejo 占用**（`forgejo.service`，uid 115，systemd 开机自启）。
> 此前文档写的「前端跑在 localhost:3000」在本机**不成立**，必须换一个端口，
> 并**同步设置后端 `BOXLI_FRONTEND_URL`**（见下）。当前空闲可用端口：**3011**、**3077**。

```bash
cd frontend
# npm 缓存也需落在工作区内
export npm_config_cache=/home/xgp2012/hub/.devtools/npmcache
npm install
# 指定非 3000 端口（示例用 3011）
PORT=3011 npm run dev
```

> 若 `node_modules/.bin/nuxt` 缺少可执行位（从压缩包解压时权限丢失），
> 直接用 node 运行入口：`PORT=3011 node node_modules/nuxt/bin/nuxt.mjs dev`。

前端 dev 通过 `routeRules` 把 `/api/**` 代理到 `http://127.0.0.1:3727`，无需 CORS。

**⚠️ 后端必须知道前端在哪个端口**：`BOXLI_FRONTEND_URL` 既是 OAuth 成功后 302 的落点，
**也是写接口的 CSRF 来源白名单**。若前端跑 3011 而该变量仍是默认的 `http://localhost:3000`，
则回跳会落到 Forgejo，且真实前端的写请求会被 403 拒绝。启动后端时须显式指定：

```bash
export BOXLI_FRONTEND_URL=http://localhost:3011
```

### 验证

```bash
curl http://localhost:3011/api/v1/health
# {"code":0,"message":"ok","data":{"status":"healthy"}}
```

> 端口换成你实际使用的前端端口；`/api/**` 由 Nuxt 同源代理到后端 `127.0.0.1:3727`。

### 5. 登录联调（未配 OAuth 时）

未配置 `BOXLI_GITHUB_CLIENT_ID/SECRET` **且**显式开启 `BOXLI_DEV_LOGIN=1` 时，
`POST /auth/login` 走 **dev 模拟登录**，直接签发会话，便于本地联调：

```bash
curl -X POST http://127.0.0.1:3727/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"github_user":"alice"}' -c cookies.txt
# 响应 data.token 即 JWT；同时下发 httpOnly Cookie，后续写接口可直接用 -b cookies.txt
```

> **⚠️ dev 模拟登录默认关闭**（`BOXLI_DEV_LOGIN=0`）。开启后任何人传一个 `github_user`
> 即可登录为任意账号，**生产环境绝不能开启**。未开启且未配 OAuth 时，`/auth/login` 返回 503。
>
> 注意：dev 模拟登录每次生成**随机 `github_id`**，若该 `github_user` 已存在会返回
> **409**（L7 的账号接管防护），换一个未占用的用户名即可。

配好 OAuth 凭证后，同一接口自动改为返回 `authorize_url`，走真实 GitHub 流程
（此时 `BOXLI_DEV_LOGIN` 不再生效）。


### 6. GitHub OAuth 凭据配置（阶段 4）

在 GitHub → Settings → Developer settings → **OAuth Apps** 创建应用，需要填：

- **Homepage URL**：本地填前端实际端口（如 `http://localhost:3011`；**本机不要用 3000，已被 Forgejo 占用**），生产填 `https://boxli.dev`
- **Authorization callback URL**：**必须与 `BOXLI_GITHUB_REDIRECT` 完全一致**，否则 GitHub 拒绝回调

本项目已申请的应用凭据（Client ID 为公开信息；**Secret 不写入仓库**）：

| 变量 | 值 | 说明 |
|---|---|---|
| `BOXLI_GITHUB_CLIENT_ID` | `Ov23lidTkmYmm6aJtJKM` | 20 位，`Ov23li` 前缀 = 2023 年后 GitHub Client ID 格式 |
| `BOXLI_GITHUB_SECRET` | 见密码管理器 / 环境变量 | 40 位十六进制；**不得提交到仓库或写入文档** |
| `BOXLI_GITHUB_REDIRECT` | 须与 App 登记一致 | 本地建议 `http://127.0.0.1:3727/api/v1/auth/callback` |

> **✅ 回调地址已确认可用（2026-10-02 复测）**：App 上登记的 Authorization callback URL 即
> `http://127.0.0.1:3727/api/v1/auth/callback`，与 `BOXLI_GITHUB_REDIRECT` 一致。
>
> 实测证据：
> - `access_token` 换 token 请求带该 `redirect_uri` → 返回 `bad_verification_code`
>   （即**通过 redirect_uri 校验**，仅 code 无效）
> - `/auth/login` 返回的 `authorize_url` 经浏览器跳转 → **HTTP 200 进入 GitHub 登录页**，
>   无 `redirect_uri_mismatch`；且 `state` 在跳转中完整保留
>
> > **⚠️ 测量方法说明（供后续参考）**：早期一次探测曾把该地址误判为「未登记」，
> > 原因是**未等 GitHub 侧保存生效就立即探测**。判定 `redirect_uri` 是否登记，
> > 必须看 `access_token` 接口是否返回 `redirect_uri_mismatch`；
> > 单次结果不可信，应在 GitHub 保存后复测，并以「带 redirect_uri 能过」为准。

区分两个凭据的方法（容易搞混）：

- **Client ID**：20 位、字母数字混合、以 `Ov23li` 开头，可公开出现在授权 URL 中
- **Client Secret**：**恰好 40 位纯十六进制** `[0-9a-f]`，用于服务端换 token，绝不能外泄

验证配对是否正确（用假 code 试探，不会拿到任何真实 token）：

```bash
curl -s -X POST https://github.com/login/oauth/access_token \
  -H 'Accept: application/json' \
  -d "client_id=$BOXLI_GITHUB_CLIENT_ID" \
  -d "client_secret=$BOXLI_GITHUB_SECRET" \
  -d "code=invalid_test_code"
# 配对正确 → {"error":"bad_verification_code", ...}   （凭据有效，仅 code 无效）
# 配反对调 → {"error":"Not Found"}                      （GitHub 不认识该 App）
```

> **注意**：若要同时验证 `redirect_uri`，加上 `-d "redirect_uri=..."`；若返回
> `redirect_uri_mismatch` 则说明该地址未在 App 上登记（见上方说明）。

> **⚠️ 安全要求**：`BOXLI_GITHUB_SECRET` 只应通过环境变量注入，`.env` 文件需加入
> `.gitignore`（已配置）；若曾以任何形式外泄（聊天、日志、截图），**必须立即在 GitHub 上
> Regenerate client secret** 并更新部署环境。

## 开发环境遗留项（⚠️ 上线前需还原）

开发环境已从 Windows + WSL2 **迁移到原生 Linux（Ubuntu 22.04）**，原先的 L1/L2/L8
（WSL PG 监听全接口、`pg_hba` 全放行、Windows 防火墙规则）**已随迁移一并消除**：
当前 PG 为工作区内的免安装实例，只监听 `127.0.0.1:5432`，无额外防火墙规则。

剩余遗留项（详见 `plants.md` 阶段 2 末尾）：

| # | 改动 | 处理时机 |
|---|---|---|
| ~~L3~~ | ~~认证仅有 dev 模拟登录~~ → **已接入真实 GitHub OAuth**（配好凭证即自动切换；dev 模拟登录改为需显式 `BOXLI_DEV_LOGIN=1` 才启用的调试开关） | ✅ 阶段 4 已解决 |
| ~~L4~~ | ~~OAuth `state` 未做服务端校验~~ → **已实现一次性 state 校验**（`oauth_states` 表，10 分钟 TTL + 单次消费，防 CSRF/重放） | ✅ 阶段 4 已解决 |
| ~~L7~~ | ~~`upsertUser` 的 username 冲突回退逻辑~~ → **已移除危险的「按 username 退化查找」**（原逻辑会把新 GitHub 账号登录成同名老用户，属账号接管）；现返回 409 | ✅ 阶段 4 已解决 |
| L5 | 后端 CORS 曾为 `Access-Control-Allow-Origin: *` | 🔄 **阶段 4 已部分推进**：`*` 与 Cookie 凭证请求不兼容，已改为白名单回显 Origin + `Allow-Credentials` + `Vary: Origin`；生产由 Nginx 同源反代后可彻底移除 |
| L6 | 后端无速率限制 / 无请求 ID 日志 | 阶段 6/7 视需要 |

## 环境变量

后端从环境变量读取，均带默认值（见 `backend/.env.example`）：

| 变量 | 默认 | 说明 |
|---|---|---|
| `BOXLI_ADDR` | `127.0.0.1:3727` | 后端监听地址 |
| `BOXLI_DB` | 上表连接串 | PostgreSQL 连接串 |
| `BOXLI_DATA_DIR` | `./data` | 数据目录 |
| `BOXLI_JWT_SECRET` | 空 | JWT 签名密钥（**必填**才能登录，未设则认证接口返回 503）|
| `BOXLI_GITHUB_CLIENT_ID` | 空 | GitHub OAuth Client ID（未设则启用 dev 模拟登录）|
| `BOXLI_GITHUB_SECRET` | 空 | GitHub OAuth Client Secret（40 位 hex，**禁止提交/外泄**）|
| `BOXLI_GITHUB_REDIRECT` | 空 | GitHub OAuth 回调地址（须与 App 登记一致）|
| `BOXLI_SESSION_TTL_HOURS` | `720` | 会话有效期（小时）|
| `BOXLI_FRONTEND_URL` | `http://localhost:3000` | OAuth 成功后 302 回跳的前端站点；**同时是写接口的 CSRF 来源白名单**。⚠️ **本机必须显式覆盖**：3000 已被 Forgejo 占用，前端须换端口（如 3011），此变量需同步改，否则回跳落错站点且写请求被 403 |
| `BOXLI_COOKIE_SECURE` | `false` | 会话 Cookie 是否带 `Secure`。生产 HTTPS 必须 `true`；本地 `http://localhost` 必须 `false`，否则浏览器丢弃 Cookie |
| `BOXLI_EXTRA_ORIGINS` | 空 | 额外允许的跨站来源（逗号分隔），仅多域名/CI 场景 |
| `BOXLI_DEV_LOGIN` | `false` | **⚠️ 模拟登录后门**，仅本地调试；生产必须保持关闭 |

**认证模式切换规则**（见 `backend/internal/hub/handlers_auth.go`）：

| `CLIENT_ID` | `SECRET` | `BOXLI_DEV_LOGIN` | `POST /auth/login` 行为 |
|---|---|---|---|
| 已设 | 已设 | 任意 | 返回 `authorize_url`，走真实 GitHub OAuth（阶段 4 目标状态）|
| 任一为空 | 任一为空 | `1`/`true` | **dev 模拟登录**，按 `{github_user}` 下发会话 |
| 任一为空 | 任一为空 | 未设/`0` | **503**，明确拒绝（默认行为）|

> 即：只设 Client ID 而不设 Secret 且未开启 dev 登录时会返回 503，不会静默降级为模拟登录。

> **⚠️ `BOXLI_COOKIE_SECURE` 与访问协议必须匹配**：本地 `http://localhost` 下若设为 `true`，
> 浏览器会丢弃 Cookie，症状是「登录看似成功但始终显示未登录」。生产 HTTPS 下若设为 `false`，
> Cookie 会在明文信道上传输。


## 数据模型

```
users (1) ──< repositories (1) ──< tags (1) ──< sources
  │
  └──< sessions

oauth_states（独立，OAuth CSRF 用，无外键）
```

- `repositories`：镜像仓库（`namespace`/`name` 唯一）
- `tags`：同一仓库按 `tag + os + arch` 区分
- `sources`：每个标签的多个下载源（`type`/`url`/`priority`/`region`）
- `oauth_states`：OAuth 授权链接的一次性 state（`state` 主键、10 分钟 TTL）

## API

所有接口在 `/api/v1/` 下，响应统一 `{code, message, data}`（成功 `code=0`）。

### 读接口（公开）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/health` | 健康检查 |
| GET | `/search?q=&limit=` | 搜索公开仓库（`name`/`description`/`namespace` ILIKE） |
| GET | `/repos?namespace=&limit=&offset=` | 仓库列表 |
| GET | `/repos/{ns}/{repo}` | 仓库详情（含所有 tags 与 sources） |
| GET | `/repos/{ns}/{repo}/tags` | 标签列表 |
| GET | `/repos/{ns}/{repo}/readme` | README 文本 |

### 认证

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/auth/login` | 已配 OAuth：返回 `authorize_url` + `state` + `state_expires_at`（可传 `redirect` 指定登录后回跳的站内路径）；未配：dev 模拟登录 `{github_user}` 直接下发会话 |
| GET | `/auth/callback?code=&state=` | GitHub OAuth 回调：**先校验并消费 `state`** → 换 token → 拉用户 → **302 重定向回前端**（会话经 httpOnly Cookie 下发） |
| GET | `/auth/me` | 当前用户（`Cookie: boxli_session` 或 `Authorization: Bearer <token>`） |
| POST | `/auth/logout` | 登出，吊销该 session 并清除 Cookie |

认证采用 **JWT + sessions 表双写**：JWT 内含 `sid`，服务端在 `sessions` 表存 `sid` 的 SHA-256。校验时同时验证签名与 session 有效性，登出即删除 session。

**会话交付：httpOnly Cookie（阶段 4 定案）**

`/auth/callback` 不再返回裸 JSON，而是 **302 回前端**，会话以 **httpOnly Cookie**（`boxli_session`）下发：

- **token 绝不放进重定向 URL 的 query** —— 那会进入浏览器历史、Referer 头与服务器访问日志
- **httpOnly** 使 JS 读不到会话，XSS 无法窃取（这是选 Cookie 而非 localStorage 的核心原因）
- 同时仍返回 `token` 字段，curl / CLI 可用 `Authorization: Bearer` 联调（两条路径都支持）
- 登录成功回跳**来源页**（如从 `/submit` 发起则回到 `/submit`），无则进 `/dashboard`

> **⚠️ CORS 必须与 Cookie 配套**：`Access-Control-Allow-Origin: *` 与带凭证的请求**不兼容**，
> 浏览器会直接拒绝。现改为「Origin 在白名单内则回显该 Origin + `Access-Control-Allow-Credentials: true`」，
> 并加 `Vary: Origin` 防缓存串站。

**CSRF 防护（Cookie 方案的必要配套）**

改用 Cookie 后浏览器会自动携带凭证，「跨站发起的写请求」因此自带认证 —— 这正是 CSRF。采用**双层**防护：

| 层 | 机制 | 说明 |
|---|---|---|
| 浏览器层 | Cookie `SameSite=Lax` | 跨站 POST 不携带 Cookie；用 Lax 而非 Strict，是为了让「从 GitHub 跳回本站」的顶层导航能带上 Cookie |
| 应用层 | `OriginGuard` 校验 `Origin`/`Referer` | 必须与白名单**精确同源**（scheme+host+port）；不依赖浏览器实现 |

判定细节：请求缺 `Origin` 时回退 `Referer`；两者都没有则放行（curl/CLI 不携带 ambient 凭证，不构成 CSRF）；
`GET`/`HEAD`/`OPTIONS` 不校验（不改状态，且预检必须放行）。

> **⚠️ 不能用 `strings.HasPrefix` 判来源** —— 否则（前端在 3011 时）`http://localhost:3011.evil.com`
> 会被误判为可信。实现用 `url.Parse` 比较 scheme+host。

**开放重定向防护**：登录回跳地址来自前端查询参数，`safeRedirectPath` 只放行**站内相对路径**，
拒绝 `//evil.com`、`https://evil.com`、`javascript:` 及 CRLF 注入。

**OAuth CSRF 防护（state）**：`/auth/login` 生成随机 state 并写入 `oauth_states` 表（10 分钟 TTL），
`/auth/callback` 用 `DELETE ... RETURNING` **原子校验并消费** state —— 因此 state **一次性**，
重放、伪造、过期均返回 302 到 `/login?error=invalid_state`。state 的校验在回调处理中**最先执行**，
即使后续失败或被用户拒绝也已被消费，不留可重放的凭证。过期行在每次 `login` 时顺带清理。

**回调错误码**（302 到 `/login?error=<code>`，前端映射为中文提示）：

| code | 含义 |
|---|---|
| `invalid_state` | state 缺失 / 伪造 / 已使用 / 过期 |
| `state_error` | state 校验时数据库出错 |
| `access_denied` | 用户在 GitHub 上取消了授权 |
| `missing_code` | GitHub 未返回授权码 |
| `exchange_failed` | 换取 access token 失败 |
| `github_failed` | 拉取 GitHub 用户信息失败 |
| `identity_taken` | 用户名或邮箱已被其他账号占用（对应 L7 的 409 语义） |
| `user_failed` | 创建/更新用户失败 |
| `issue_failed` | 签发会话失败 |


### 写接口（需登录）

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/repos` | 创建仓库（`namespace` 缺省为当前用户名） |
| PUT | `/repos/{ns}/{repo}` | 更新仓库（仅 owner；tags/sources 整体替换） |
| DELETE | `/repos/{ns}/{repo}` | 删除仓库（仅 owner，级联删除 tags/sources） |

请求体示例（`POST /repos`）：

```json
{
  "name": "myapp",
  "description": "My awesome app",
  "readme": "# myapp",
  "tags": [
    {
      "tag": "v1.0.0", "os": "linux", "arch": "amd64",
      "sources": [
        { "type": "github", "url": "https://github.com/...", "priority": 1, "region": "global" },
        { "type": "gitee",  "url": "https://gitee.com/...",  "priority": 2, "region": "cn" }
      ]
    }
  ]
}
```

> dev 模拟登录（未配 OAuth）见「本地开发 · 5. 登录联调」。

## 前端页面（阶段 3 + 阶段 5）

| 路由 | 页面 | 数据来源 |
|---|---|---|
| `/` | 首页：Hero、快速开始、特性、热门镜像 | `GET /repos?limit=6` |
| `/explore` | 浏览：列表 + 搜索 + 排序（星标/下载量/更新） | `GET /repos` 或 `GET /search` |
| `/explore/{ns}/{repo}` | 详情：README、标签切换、多源下载、复制 pull | `GET /repos/{ns}/{repo}` |
| `/search?q=` | 搜索：关键词 + 推荐词 | `GET /search` |
| `/about` | 关于：项目背景、设计理念、协议 | 静态 |
| `/login` | 登录：GitHub OAuth 入口、错误提示、已登录引导 | 静态 + `POST /auth/login` |
| `/submit` | 提交镜像：动态标签与下载源 | `POST /repos`（需登录） |
| `/dashboard` | 用户中心：我的镜像列表 / 编辑 / 删除 | `GET /repos?namespace=` + `PUT`/`DELETE` |

### 文档站（阶段 5）

| 路由 | 内容 |
|---|---|
| `/docs` | 文档首页：总览 + 阅读顺序 + 快速索引 |
| `/docs/quickstart` | 快速开始：环境自检 → 导入镜像 → 运行容器 |
| `/docs/install` | 安装与自检：源码构建、`doctor` 检查项、数据目录与环境变量 |
| `/docs/pull-run` | 拉取与运行：`pull` / `run` 全部 flag 表、容器管理、停止语义 |
| `/docs/format` | 镜像格式与下载源：`.boxli` 与 Docker 差异、10 种源类型、多架构、Boxfile |
| `/docs/submit` | 提交镜像到 Hub：准备、表单字段、校验规则、整体替换语义 |
| `/docs/faq` | 常见问题：**CLI 与 Hub 对接现状**、容器行为、登录与完整性校验 |

文档内容为 `app/utils/docs.ts` 中的静态常量，经 `markdown-it(html:false)` + DOMPurify
渲染（与镜像 README 同一条安全链路），标题自动注入 `id` 锚点以支持 `#小节` 深链。

> **⚠️ 文档内容以真实 CLI 为准**：命令与 flag 抄录自本机 `boxli --help`
> （`boxli version 0.0.0-dev`，实际 **28 个子命令**），未按规格臆造。
> 其中**镜像引用为 `NAME:VERSION`**（无命名空间段），`pull` 为**双语义**
> （传 `.boxli` 文件则导入，传 `NAME:VERSION` 才走 Hub）。
>
> **⚠️ CLI 与本站 Hub 尚未对接**：本机 `boxli` 对接的是另一套 Hub
> （`boxli hub serve`：用户名/密码 + blob 存储），实测
> `boxli search --hub http://127.0.0.1:3727` → `未登录`、
> `boxli login --hub http://127.0.0.1:3727` → `hub 404 Not Found`。
> 因此文档未提供 CLI 直连本站 Hub 的示例，现阶段请经网页端获取下载地址。
> 详见 `/docs/faq` 首节。

`/submit` 与 `/dashboard` 受 `auth` 中间件保护，未登录时跳转 `/login?redirect=<原路径>`，
登录成功后原路返回。

**手机优先实现要点：**

- 所有交互元素 `min-h-11`（44px）且带 `min-w-11`；文字链接加 `px-2` 保证宽度
- 输入框 / 下拉框统一 `text-base`（16px），避免 iOS 聚焦缩放
- 全局 `min-h-[100dvh]`（不用 `100vh`）与 `overflow-x-hidden`
- 页脚 `pb-[calc(2rem+env(safe-area-inset-bottom))]` 防 iPhone 横条遮挡
- 手机汉堡菜单用 fuxsto `Drawer`；导航与信息**不依赖 hover**（坑 2）
- 长 URL 用 `break-all`，标签栏 `overflow-x-auto` 横向滚动，避免撑破窄屏

**README 渲染安全**：`markdown-it`（`html: false`，禁用原始 HTML）+ `isomorphic-dompurify`
清洗后输出，外链自动补 `target="_blank" rel="noopener noreferrer nofollow"`。

**⚠️ 手机端验收状态**：当前仅完成**静态断言**（viewport、`100dvh`、无固定宽度、
输入框 16px、44px 触控类名、`overflow-x-hidden`）。一轮真实浏览器测试曾发现
`CopyButton` 宽度缺陷（34px）并修复，但**该修复未复测**，且以下项**从未验证**：
Lighthouse Mobile ≥ 90、真实触摸/滚动惯性、iOS 地址栏 `100dvh` 表现、iOS 聚焦实测。
**阶段 4 新增的 `/login`、`/submit`、`/dashboard` 同样只有静态断言。**
**阶段 5 新增的 7 个文档页（`/docs` 及子页面）亦为静态断言。**
测试用浏览器与 `puppeteer-core` 已移除，需在**阶段 6** 补齐（详见 `plants.md`）。

## 进度

| 阶段 | 内容 | 状态 |
|---|---|---|
| 0 | 环境与脚手架 | ✅ 2026-10-02 |
| 1 | 数据库与数据层 | ✅ 2026-10-02 |
| 2 | 后端 API | ✅ 2026-10-02 |
| 3 | 前端核心页面 | ✅ 2026-10-02（手机端验收部分待补，见下） |
| 4 | 认证与提交 | 🔄 **代码已全部完成**；仅剩「人工点一次真实授权」（原端口占用阻塞已于 2026-10-03 清理） |
| 5 | 文档站 | ✅ 2026-10-02（7 个页面；手机端为静态断言，真实浏览器验收属阶段 6） |
| 6 | 手机端全面验收 | 待做 |
| 7 | 部署上线 | 待做 |

### 阶段 3 未完成的验收项（阶段 6 必须补齐）

阶段 3 的页面功能与静态规范已通过，但**手机端真实测试不完整**，以下**不得视为已验收**：

- `CopyButton` 宽度修复（`min-w-11`）**未经复测**
- **Lighthouse Mobile ≥ 90** 从未测量
- 真实触摸延迟、iOS 滚动惯性、iOS 地址栏 `100dvh` 表现、iOS 聚焦实测

测试用浏览器与 `puppeteer-core` 已移除；阶段 6 需重新准备浏览器环境或用真机。

### 阶段 4 状态

**代码已全部完成**（2026-10-02）；**唯一未完成项是「人工点一次真实授权」**。
原记录的「端口被孤儿进程占用」**阻塞已于 2026-10-03 清理，且该诊断经实测有误、已更正**（见下）。

后端：

- ✅ 配置凭证后 `/auth/login` 自动切换到真实 OAuth（返回 `authorize_url`）
- ✅ **L4 已修复**：`oauth_states` 表 + 一次性 state 校验（`DELETE ... RETURNING` 原子消费，10 分钟 TTL）
- ✅ **L7 已修复**：移除「username 冲突时按用户名退化查找」的账号接管路径，改为 409
- ✅ **回调改为 302 回前端**，会话经 **httpOnly Cookie** 交付（token 不进 URL）
- ✅ **CSRF 双层防护**：`SameSite=Lax` + `OriginGuard` 精确同源校验
- ✅ **开放重定向防护**：`safeRedirectPath` 仅放行站内相对路径
- ✅ **CORS 修正**：`Allow-Origin: *` 与 Cookie 凭证不兼容，改为白名单回显 + `Allow-Credentials`
- ✅ **dev 模拟登录改为显式开关**（`BOXLI_DEV_LOGIN`，默认关闭）
- ✅ 授权页实测可达，显示 App 名 **boxli**（Client ID 有效）
- ✅ 单元测试 **8 个用例组（另 13 个子例）** 全通过（`go test ./...`）

前端：

- ✅ `/login`（OAuth 入口 + 9 种错误码中文映射 + 已登录态引导）
- ✅ `/submit`（建仓表单，支持动态添加多个下载源）
- ✅ `/dashboard`（我的镜像：列表 / 编辑 / 删除）
- ✅ 鉴权状态管理（`useAuth`）与未登录跳转（`auth` 中间件，仅在客户端判定）
- ✅ `eslint` 0 error、`nuxt build` 成功

**实测通过**（curl 逐项）：Cookie 会话下发与校验、无凭证 401、跨站 Origin 403、
302 回跳全链路（含错误码落地为中文提示）、state 伪造/重放/拒绝均一次性消费、
完整 CRUD（含 3 源入库、整体替换、级联删除）、非 owner 403、空 tags 400。

**尚需处理：**

1. ✅ `BOXLI_GITHUB_REDIRECT` 已与 App 登记值一致（复测通过）
2. ⚠️ `BOXLI_GITHUB_SECRET` **曾明文外泄，上线前必须 Regenerate**（阶段 7）
3. ✅ **陈旧进程已清理（2026-10-03）** —— 原先记录的「`127.0.0.1:3727` 与 `localhost:3000`
   被**孤儿进程**占用且跑陈旧构建」**诊断有误，已更正**：

   - `127.0.0.1:3727` / `:3740` / `:3750` 确曾堆积 **5 组重复的 `go run ./cmd/hub`**
     （父进程 + 子进程共 10 个，**属主可见、并非孤儿**），已全部 SIGTERM 清理，端口释放；
   - **`localhost:3000` 与 Boxli 前端无关** —— 该端口属 **Forgejo**
     （`forgejo.service`，uid 115，systemd `enabled` + `active`，长期占用）。
     此前观察到的 `/` 200、`/explore` 303、`/login` `/docs` 404
     **全部是 Forgejo 的响应**，并非「陈旧的前端构建」。

   ⚠️ **由此得出一个必须遵守的约束**：本机**前端不能再用 3000**（与 Forgejo 冲突），
   须改用其他端口（如 **3011**），并**同步设置 `BOXLI_FRONTEND_URL`** 指向该端口。
   否则该变量会落到默认值 `http://localhost:3000`（即 Forgejo），导致
   **OAuth 回跳落错站点**、且**写接口 CSRF 白名单拒绝真实前端（403）**。
   详见 `plants.md` 阶段 4「阻塞项」一节。

4. ⚠️ **真实授权联调仍未完成** —— 但**已不再被端口占用阻塞**（进程已清理）。
   现只需：设定 `BOXLI_FRONTEND_URL` → 启动后端 :3727 → 启动前端（非 3000 端口）
   → 人工点一次授权。
5. ⚠️ 阶段 4 新增页面与阶段 3 一样**只有静态断言**，真实浏览器验收属**阶段 6**

详见 `plants.md` 阶段 4。

