# boxli hub

Boxli 官网 + Hub 镜像索引站的 monorepo。

Hub **只收录社区镜像的元数据，不存储镜像本体** —— 镜像文件放在用户自己的地方
（GitHub / Gitee / OSS / S3 / IPFS / BT 等），Hub 只记录「去哪下载」。

## 文档导航

| 文档 | 内容 |
|---|---|
| 本文档 | 项目概览、环境准备、启动步骤、配置字段、API 简表 |
| [`DEVELOPMENT.md`](./DEVELOPMENT.md) | **开发者技术文档**：架构、模块职责、认证实现细节、调试与测试、安全设计 |
| [`DEPLOYMENT.md`](./DEPLOYMENT.md) | **部署文档**：构建、systemd、Nginx、HTTPS、备份与排错 |
| [`plants.md`](./plants.md) | 立项时的分阶段计划 —— **历史存档**，含已被推翻的规格，不作为现行依据 |

---

## 目录

- [快速开始](#快速开始)
- [项目结构](#项目结构)
- [本地开发](#本地开发)
- [配置文件](#配置文件)
- [数据模型](#数据模型)
- [API 简表](#api-简表)
- [前端页面](#前端页面)
- [当前状态](#当前状态)

---

## 快速开始

> 前提：PostgreSQL 已在 `127.0.0.1:5432` 运行、`backend/hub.toml` 已按需配置。
> 完整的环境准备见[本地开发](#本地开发)。

```bash
# 后端（默认读当前目录 hub.toml）
cd backend
cp hub.toml.example hub.toml     # 首次
go run ./cmd/hub                 # 监听 127.0.0.1:3727，启动时自动迁移

# 前端（另开一个终端；不能用 3000，见下）
cd frontend
PORT=3011 npm run dev

# 验证
curl http://localhost:3011/api/v1/health
# {"code":0,"message":"ok","data":{"status":"healthy"}}
```

> **⚠️ 本机前端不能用 3000 端口** —— `:3000` 属 **Forgejo**（系统服务，开机自启），
> 与 Boxli 无关。请改用 **3011** 或 **3077**，并确保 `hub.toml` 的
> `[site] frontend_url` 指向同一端口，否则 OAuth 回跳会落错站点、写请求会被 403 拒绝。
> 详见[配置文件](#配置文件)。

## 技术栈

| 部分 | 选型 |
|---|---|
| 前端 | Nuxt 4 + fuxsto-design + Tailwind CSS v4 |
| 后端 | Go（`net/http`）+ pgx/v5 |
| 数据库 | PostgreSQL 17（免安装二进制，见[本地开发](#本地开发)）|
| 运行环境 | **Ubuntu 22.04.5 LTS（原生 Linux，非 WSL）** |

> **环境迁移记录（2026-10-02）**：开发环境由 Windows + WSL2 迁移到原生 Linux。
> 迁移消除了原有的 L1/L2/L8 三项遗留问题（WSL PG 监听全接口、`pg_hba` 全放行、
> Windows 防火墙规则）。当前机器**无 sudo 权限**（沙箱 `NoNewPrivs: 1`），
> 因此数据库采用免安装二进制方案，仅监听 `127.0.0.1:5432`。

## 项目结构

```
hub/
├── frontend/          # Nuxt 4：官网 + Hub 前端 + 文档站
├── backend/           # Go：Hub API（只存元数据）
│   ├── cmd/hub/       #   服务入口
│   ├── cmd/seed/      #   种子数据入口（幂等）
│   └── internal/
│       ├── auth/      #   JWT + sessions、GitHub OAuth、state 一次性校验
│       │              #    + cookie.go（httpOnly 会话 Cookie）
│       │              #    + origin.go（CSRF 来源白名单校验）
│       ├── config/    #   TOML 配置加载与校验（hub.toml）
│       ├── db/        #   连接池 + 内嵌 SQL 迁移
│       ├── hub/       #   HTTP 路由与处理器（读/写/认证）
│       └── seed/      #   种子数据定义与写入
├── README.md          # 本文档
├── DEVELOPMENT.md     # 开发者技术文档
├── DEPLOYMENT.md      # 部署文档
└── plants.md          # 历史计划存档
```

前端与后端的**逐文件职责说明**见 [`DEVELOPMENT.md`](./DEVELOPMENT.md#二目录与模块职责)。

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

连接串（已写入 `hub.toml.example` 的 `[db] url`）：
`postgres://boxli:boxli@127.0.0.1:5432/boxli_hub?sslmode=disable`

### 2. 配置

```bash
cd backend
cp hub.toml.example hub.toml     # 按需修改，字段说明见下节
```

### 3. 建表 + 种子数据

```bash
cd backend
# Go 缓存需落在工作区内（沙箱不允许写 ~/.cache 与 ~/go）
export GOCACHE=/home/xgp2012/hub/.devtools/gocache \
       GOMODCACHE=/home/xgp2012/hub/.devtools/gomodcache \
       GOPATH=/home/xgp2012/hub/.devtools/gopath
go run ./cmd/seed
```

后端启动时也会自动跑 schema migration（`internal/db/migrations/*.sql`，记录在
`schema_migrations` 表），该命令可**重复执行不报错**（幂等）。

种子内容：5 个示例镜像 / 10 个标签 / 17 个下载源，覆盖 github、gitee、oss、cos、
s3、ipfs、magnet、http 等源类型。

### 4. 后端（127.0.0.1:3727）

```bash
cd backend
export GOCACHE=/home/xgp2012/hub/.devtools/gocache \
       GOMODCACHE=/home/xgp2012/hub/.devtools/gomodcache \
       GOPATH=/home/xgp2012/hub/.devtools/gopath
go run ./cmd/hub                          # 读当前目录的 hub.toml
go run ./cmd/hub --config /path/hub.toml  # 或指定配置文件
```

### 5. 前端

```bash
cd frontend
# npm 缓存也需落在工作区内
export npm_config_cache=/home/xgp2012/hub/.devtools/npmcache
npm install
PORT=3011 npm run dev
```

> 若 `node_modules/.bin/nuxt` 缺少可执行位（从压缩包解压时权限丢失），
> 直接用 node 运行入口：`PORT=3011 node node_modules/nuxt/bin/nuxt.mjs dev`。

前端 dev 通过 `nuxt.config.ts` 的 `routeRules` 把 `/api/**` 代理到
`http://127.0.0.1:3727`，因此**同源、无需 CORS**。

### 6. 登录联调

**方式 A：真实 GitHub OAuth**（推荐，需先配好凭据，见下）

浏览器打开 `http://localhost:3011/login`，点「使用 GitHub 登录」。
会话经 httpOnly Cookie 下发，成功后回跳来源页（无则进 `/dashboard`）。

**方式 B：dev 模拟登录**（临时调试用，默认关闭）

未配 `[github] client_id/secret` **且**显式开启 `[dev] enabled = true` 时，
`POST /auth/login` 直接签发会话：

```bash
curl -X POST http://127.0.0.1:3727/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"github_user":"alice"}' -c cookies.txt
# 响应 data.token 即 JWT；同时下发 httpOnly Cookie，后续写接口可直接用 -b cookies.txt
```

> **⚠️ dev 模拟登录是登录后门**：开启后任何人传一个 `github_user` 即可登录为任意账号，
> **生产环境绝不能开启**。把 `enabled` 设为 `true` 时**程序会直接拒绝启动**，
> 以防误带入生产；未开启且未配 OAuth 时 `/auth/login` 返回 503。
>
> 注意：dev 模拟登录每次生成**随机 `github_id`**，若该 `github_user` 已存在会返回
> **409**（账号接管防护，见 [`DEVELOPMENT.md`](./DEVELOPMENT.md#59-upsertuser-与-l7)），
> 换一个未占用的用户名即可。

**GitHub OAuth App 配置**

在 GitHub → Settings → Developer settings → **OAuth Apps** 创建应用：

- **Homepage URL**：本地填前端实际端口（如 `http://localhost:3011`），
  生产填 `https://boxli.dev`
- **Authorization callback URL**：**必须与 `hub.toml` 的 `[github] redirect` 完全一致**，
  否则 GitHub 拒绝回调

| hub.toml 字段 | 值 | 说明 |
|---|---|---|
| `[github] client_id` | `Ov23lidTkmYmm6aJtJKM` | 公开信息，20 位，`Ov23li` 前缀 |
| `[github] secret` | 见密码管理器 | 40 位十六进制，**不得提交到仓库** |
| `[github] redirect` | 本地 `http://127.0.0.1:3727/api/v1/auth/callback` | 须与 App 登记一致 |

> **✅ 本地回调地址已确认可用（2026-10-02 复测）**：带该 `redirect_uri` 请求
> `access_token` 返回 `bad_verification_code` 而非 `redirect_uri_mismatch`，
> 即**通过了 redirect_uri 校验**（仅 code 无效）。

区分两个凭据：**Client ID** 是 20 位字母数字混排、以 `Ov23li` 开头；
**Client Secret** 是**恰好 40 位纯十六进制** `[0-9a-f]`，用于服务端换 token，绝不能外泄。

验证配对是否正确（用假 code 试探，不会拿到任何真实 token）：

```bash
CID=$(grep -oP '(?<=^client_id = ").*(?=")' backend/hub.toml)
CSEC=$(grep -oP '(?<=^secret = ").*(?=")' backend/hub.toml)
curl -s -X POST https://github.com/login/oauth/access_token \
  -H 'Accept: application/json' \
  -d "client_id=$CID" -d "client_secret=$CSEC" -d "code=invalid_test_code"
# 配对正确 → {"error":"bad_verification_code", ...}   （凭据有效，仅 code 无效）
# 配反对调 → {"error":"Not Found"}                      （GitHub 不认识该 App）
```

> 若要同时验证 `redirect_uri`，加上 `-d "redirect_uri=..."`；返回
> `redirect_uri_mismatch` 说明该地址未在 App 上登记。

> **⚠️ 安全要求**：`[github] secret` 写在 `backend/hub.toml`，该文件已加入 `.gitignore`，
> **禁止提交**。若曾以任何形式外泄（聊天、日志、截图），**必须立即在 GitHub 上
> Regenerate client secret** 并更新部署配置。

## 配置文件

后端**不使用任何环境变量**，全部配置来自 TOML 文件。默认读取当前目录的 `hub.toml`，
可用 `--config` 指定其他路径 —— 这是二进制**唯一**解析的命令行参数。

```bash
./boxli-hub                                     # 读 ./hub.toml
./boxli-hub --config /etc/boxli-hub/hub.toml    # 读指定文件
```

### 字段一览

| TOML 字段 | 默认 | 说明 |
|---|---|---|
| `[server] addr` | `127.0.0.1:3727` | 监听地址。**只允许回环地址**（填 `0.0.0.0` 会拒绝启动），对外由 Nginx 反代 |
| `[db] url` | 本地连接串 | PostgreSQL 连接串 |
| `[session] jwt_secret` | 空 | JWT 签名密钥（**必填**才能登录，未设则认证接口返回 503；≥16 字符）|
| `[session] ttl_hours` | `720` | 会话有效期（小时）|
| `[session] cookie_secure` | `false` | 会话 Cookie 是否带 `Secure`。生产 HTTPS 必须 `true`；本地 `http://` 必须 `false` |
| `[github] client_id` | 空 | GitHub OAuth Client ID |
| `[github] secret` | 空 | GitHub OAuth Client Secret（**禁止提交/外泄**）|
| `[github] redirect` | 空 | GitHub OAuth 回调地址（须与 App 登记一致；配了凭据则必填）|
| `[site] frontend_url` | `http://localhost:3011` | OAuth 回跳落点，**同时是写接口的 CSRF 来源白名单** |
| `[site] extra_origins` | `[]` | 额外允许的跨站来源（数组），仅多域名/CI 场景 |
| `[dev] enabled` | `false` | **⚠️ 模拟登录后门**，设为 `true` 时程序拒绝启动 |

> 完整注释版见 [`backend/hub.toml.example`](./backend/hub.toml.example)。
>
> 旧的 `BOXLI_DATA_DIR` 未迁移：该字段在代码中从未被读取（死配置）。

### 启动期校验

配置写错会在启动时报错退出，而不是带着坏配置跑起来：

| 检查 | 拒绝原因 |
|---|---|
| 无法识别的配置项 | 防止键名拼错后静默不生效 |
| `addr` 非回环 / 缺少端口 | 避免绕开 Nginx 的 HTTPS 与 CSRF 保护 |
| `jwt_secret` 短于 16 字符 | 弱密钥 |
| 只配 `client_id` 或只配 `secret` | 会静默降级为未配置 |
| 配了凭据却没有 `redirect` | 回调必然失败 |
| `frontend_url` 带路径 / 协议非法 | `OriginGuard` 按 `scheme://host:port` 精确比对，带路径永远匹配不上 |
| `cookie_secure=true` 却配 `http://` | 浏览器丢弃 Cookie，表现为「登录后仍显示未登录」 |
| 公网 `https://` 却未开 `cookie_secure` | Cookie 会在明文信道传输 |
| `[dev] enabled = true` | 模拟登录后门 |

> 启动日志会打印一行配置摘要（如 `config{path=... addr=... db=postgres://boxli:***@...}`），
> **数据库密码与全部密钥均已脱敏**。

### 认证模式切换规则

| `client_id` | `secret` | `[dev] enabled` | `POST /auth/login` 行为 |
|---|---|---|---|
| 已设 | 已设 | 任意 | 返回 `authorize_url`，走真实 GitHub OAuth |
| 留空 | 留空 | `true` | dev 模拟登录，按 `{github_user}` 下发会话 |
| 留空 | 留空 | `false` | **503**，明确拒绝（默认行为）|

> 只设 `client_id` 而不设 `secret` 时，配置加载阶段就**直接报错退出**，
> 不会静默降级为模拟登录或 503。

> **⚠️ `cookie_secure` 与访问协议必须匹配**：本地 `http://localhost` 下若设为 `true`，
> 浏览器会丢弃 Cookie，症状是「登录看似成功但始终显示未登录」；生产 HTTPS 下若为 `false`，
> Cookie 会在明文信道上传输。两者不匹配时**启动即报错**。

## 数据模型

```
users (1) ──< repositories (1) ──< tags (1) ──< sources
  │
  └──< sessions

oauth_states（独立，OAuth CSRF 用，无外键）
```

- `repositories` — 镜像仓库（`namespace`/`name` 唯一）
- `tags` — 同一仓库按 `tag + os + arch` 区分
- `sources` — 每个标签的多个下载源（`type`/`url`/`priority`/`region`）
- `oauth_states` — OAuth 的一次性 state（10 分钟 TTL）

完整 DDL 与索引见 [`DEVELOPMENT.md`](./DEVELOPMENT.md#三数据模型)。

## API 简表

所有接口在 `/api/v1/` 下，响应统一 `{code, message, data}`（成功 `code=0`）。
**实现细节、请求体示例与错误码见 [`DEVELOPMENT.md`](./DEVELOPMENT.md#七api-参考)。**

**读接口（公开）**

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/health` | 健康检查 |
| GET | `/search?q=&limit=` | 搜索公开仓库 |
| GET | `/repos?namespace=&limit=&offset=` | 仓库列表 |
| GET | `/repos/{ns}/{repo}` | 仓库详情（含所有 tags 与 sources） |
| GET | `/repos/{ns}/{repo}/tags` | 标签列表 |
| GET | `/repos/{ns}/{repo}/readme` | README 文本 |

**认证**

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/auth/login` | 已配 OAuth 返回 `authorize_url`；否则需 dev 登录开启才可模拟登录 |
| GET | `/auth/callback?code=&state=` | 校验并消费 state → 换 token → **302 回前端**（会话经 Cookie 下发） |
| GET | `/auth/me` | 当前用户（Cookie 或 Bearer） |
| POST | `/auth/logout` | 登出，吊销 session 并清 Cookie |

**写接口（需登录）**

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/repos` | 创建仓库（`namespace` 缺省为当前用户名） |
| PUT | `/repos/{ns}/{repo}` | 更新仓库（仅 owner；**tags/sources 整体替换**） |
| DELETE | `/repos/{ns}/{repo}` | 删除仓库（仅 owner，级联删除 tags/sources） |

> **⚠️ `PUT` 是整体替换语义** —— 前端编辑时必须先拉全量详情回填，
> 否则未列出的标签与下载源会被**静默删除**。

## 前端页面

| 路由 | 页面 | 数据来源 |
|---|---|---|
| `/` | 首页：Hero、快速开始、特性、热门镜像 | `GET /repos?limit=6` |
| `/explore` | 浏览：列表 + 搜索 + 排序 | `GET /repos` 或 `GET /search` |
| `/explore/{ns}/{repo}` | 详情：README、标签切换、多源下载 | `GET /repos/{ns}/{repo}` |
| `/search?q=` | 搜索：关键词 + 推荐词 | `GET /search` |
| `/about` | 关于：项目背景、设计理念、协议 | 静态 |
| `/login` | 登录：GitHub OAuth 入口、错误提示 | 静态 + `POST /auth/login` |
| `/submit` | 提交镜像：动态标签与下载源 | `POST /repos`（需登录） |
| `/dashboard` | 用户中心：我的镜像列表 / 编辑 / 删除 | `GET /repos?namespace=` + `PUT`/`DELETE` |
| `/docs` 及 6 个子页 | 文档站：快速开始、安装、拉取与运行、镜像格式、提交、FAQ | `app/utils/docs.ts` 静态内容 |

`/submit` 与 `/dashboard` 受 `auth` 中间件保护，未登录时跳转 `/login?redirect=<原路径>`，
登录成功后原路返回。

> **⚠️ 登录守卫只在客户端判定**：会话存于 httpOnly Cookie，**SSR 期间不会自动携带**。
> 若在 SSR 就判定，会把**已登录用户也误判为未登录**。因此
> `middleware/auth.ts` 首行即 `if (import.meta.server) return`。

**手机优先实现要点：**

- 所有交互元素 `min-h-11`（44px）且带 `min-w-11`；文字链接加 `px-2` 保证宽度
- 输入框 / 下拉框统一 `text-base`（16px），避免 iOS 聚焦缩放
- 全局 `min-h-[100dvh]`（不用 `100vh`）与 `overflow-x-hidden`
- 页脚 `pb-[calc(2rem+env(safe-area-inset-bottom))]` 防 iPhone 横条遮挡
- 手机汉堡菜单用 fuxsto `Drawer`；导航与信息**不依赖 hover**
- 长 URL 用 `break-all`，标签栏 `overflow-x-auto` 横向滚动，避免撑破窄屏

**README 渲染安全**：`markdown-it`（`html: false`，禁用原始 HTML）+
`isomorphic-dompurify` 清洗后输出，外链自动补
`target="_blank" rel="noopener noreferrer nofollow"`。

## 当前状态

功能已全部实现（官网、Hub 索引、认证与提交、文档站）。**尚缺整机验收与上线**：

| 项 | 状态 |
|---|---|
| 后端 API、数据库、认证、CSRF / 开放重定向 / state 防护 | ✅ 已实现，curl 逐项实测 + 单元测试通过 |
| 前端页面、文档站（7 页） | ✅ 已实现，`eslint` 0 error、`nuxt build` 成功 |
| **真实 GitHub 授权端到端** | 🔄 代码就绪，**尚缺人工点一次授权** |
| **手机端真机 / 浏览器验收** | ⬜ 待做（现有仅为静态断言） |
| **部署上线** | ⬜ 待做，见 [`DEPLOYMENT.md`](./DEPLOYMENT.md) |

**⚠️ 手机端尚未真实验收**：目前仅完成**静态断言**（viewport、`100dvh`、无固定宽度、
输入框 16px、44px 触控类名、`overflow-x-hidden`）。以下**从未验证**，不得视为已验收：

- `CopyButton` 宽度修复（`min-w-11`）**未复测**
- **Lighthouse Mobile ≥ 90** 从未测量
- 真实触摸延迟、iOS 滚动惯性、iOS 地址栏 `100dvh` 表现、iOS 聚焦实测
- `/login`、`/submit`、`/dashboard` 与 7 个文档页均**无真实浏览器测试**

测试用浏览器与 `puppeteer-core` 已移除，需重新准备浏览器环境或用真机。

**上线前必办**：

1. ⚠️ **Regenerate `[github] secret`** —— 当前值曾明文外泄
2. ⚠️ 更新 GitHub App 的 **Authorization callback URL** 为生产地址
3. ⚠️ `[session] cookie_secure = true`、`[site] frontend_url` 改为生产域名、
   重新生成 `[session] jwt_secret`

完整清单见 [`DEPLOYMENT.md`](./DEPLOYMENT.md#七上线前安全清单)。

### 后端测试

```bash
cd backend && go test ./...
```

覆盖：来源白名单校验、OriginGuard、会话 Cookie 属性、state 单次消费/伪造/过期、
开放重定向防护、**TOML 配置加载与校验**（21 个用例，含日志脱敏断言）。

> `internal/auth` 的 state 测试需连接数据库（读 `hub.toml` 的 `[db] url`），
> 连不上会自动 skip，不影响无库环境。
