# boxli hub

[![Build](https://github.com/xgp2012/liboxhub/actions/workflows/build.yml/badge.svg)](https://github.com/xgp2012/liboxhub/actions/workflows/build.yml)

Boxli 官网 + Hub 镜像索引站的 monorepo。

Hub **只收录社区镜像的元数据，不存储镜像本体** —— 镜像文件放在用户自己的地方
（GitHub / Gitee / OSS / S3 / IPFS / BT 等），Hub 只记录「去哪下载」。

> **部署形态（2026-10-03 起）**：前端产物（SSR bundle + 静态资源）已**内嵌进后端二进制**，
> 生产部署只需「**一个 `boxli-hub` 二进制 + 一个 `hub.toml` + PostgreSQL**」，
> 不再需要单独部署前端目录或第二个 systemd 服务。
> 运行时依赖 **node**（用于服务端渲染，保证 SEO）。详见 [`DEPLOYMENT.md`](./DEPLOYMENT.md)。

## 下载现成二进制

CI（[`.github/workflows/build.yml`](./.github/workflows/build.yml)）会为 **linux/amd64**
构建自包含二进制，两种获取方式：

```bash
# 方式一：从 Release 下载（打 v* tag 时自动创建）
#   https://github.com/xgp2012/liboxhub/releases

# 方式二：从 Actions 产物下载（每次 push 到 main 都会构建）
#   https://github.com/xgp2012/liboxhub/actions/workflows/build.yml
```

校验与运行：

```bash
sha256sum -c boxli-hub.sha256
./boxli-hub --version          # 查看版本与 commit
./boxli-hub --config hub.toml  # 启动
```

> 二进制为**静态链接**（`CGO_ENABLED=0`），可在任意 Linux 发行版运行，
> 且**体积约 27 MB** —— 其中包含内嵌的前端产物。
> 服务器仍需安装 **Node.js**（SSR 依赖）与 PostgreSQL。

## 文档导航

| 文档 | 内容 |
|---|---|
| 本文档 | 项目概览、快速开始、配置字段、API 简表 |
| [`DEVELOPMENT.md`](./DEVELOPMENT.md) | **开发文档**：架构、代码结构、认证实现、调试与测试 |
| [`DEPLOYMENT.md`](./DEPLOYMENT.md) | **部署文档**：单二进制部署、首次引导、Nginx、备份与排障 |

> `plants.md` 是立项时的计划存档，其中的规格**已多次被推翻**，不作为现行依据。

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

### 生产部署（一个二进制）

> 完整步骤见 [`DEPLOYMENT.md`](./DEPLOYMENT.md)。服务器需要 **Node.js**（SSR）。

```bash
make build                       # 产出自包含的 backend/boxli-hub
scp backend/boxli-hub server:    # 传到服务器，配好 hub.toml 后运行
```

### 本地跑起来

前提：PostgreSQL 在 `127.0.0.1:5432` 运行、`backend/hub.toml` 已配置。

```bash
# 单机模式：一个进程跑完整站点
make build && cd backend && cp hub.toml.example hub.toml
./boxli-hub
# 首次启动：日志会打印一次性引导令牌，浏览器打开
#   http://127.0.0.1:3727/setup?token=<日志中的令牌>
# 创建管理员后，引导接口永久关闭

# 或开发模式：前后端分离，前端热更新
make dev-backend                 # 另一个终端
make dev-frontend                # 端口 3011

curl http://localhost:3011/api/v1/health
# {"code":0,"message":"ok","data":{"status":"healthy"}}
```

> **⚠️ 开发模式下前端不能用 3000 端口** —— `:3000` 属 **Forgejo**（系统服务，开机自启），
> 与 Boxli 无关。请改用 **3011** 或 **3077**，并确保 `hub.toml` 的
> `[site] frontend_url` 指向同一端口，否则 OAuth 回跳会落错站点、写请求会被 403 拒绝。
> （单二进制部署模式下不存在此冲突，因为前端没有独立端口。）

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
│   └── scripts/       #   将 SSR 产物打包成单文件，供后端内嵌
├── backend/           # Go：Hub API（只存元数据）+ 内嵌前端
│   ├── cmd/hub/       #   服务入口（唯一进程）
│   ├── cmd/seed/      #   种子数据入口（幂等）
│   └── internal/
│       ├── auth/      #   JWT + sessions、GitHub OAuth、state 一次性校验
│       │              #    + cookie.go（httpOnly 会话 Cookie）
│       │              #    + origin.go（CSRF 来源白名单校验）
│       │              #    + setup_token.go（首次引导令牌）
│       ├── config/    #   TOML 配置加载与校验（hub.toml）
│       ├── db/        #   连接池 + 内嵌 SQL 迁移
│       ├── hub/       #   HTTP 路由与处理器（读/写/认证/引导）
│       ├── localauth/ #   本地用户名+密码认证（bcrypt）
│       ├── seed/      #   种子数据定义与写入
│       └── web/       #   内嵌前端产物 + 托管 node SSR 子进程
├── Makefile           # 一条命令完成「前端打包 + 后端内嵌」构建
├── README.md          # 本文档
├── DEVELOPMENT.md     # 开发者技术文档
├── DEPLOYMENT.md      # 部署文档
└── plants.md          # 历史计划存档
```

前端与后端的**逐文件职责说明**见 [`DEVELOPMENT.md`](./DEVELOPMENT.md#二代码结构)。

## 本地开发

```bash
make dev-backend     # 后端（读 backend/hub.toml）
make dev-frontend    # 前端（3011）
make test / vet / fmt / clean
make help            # 列出全部目标
```

> **端口 3000 属 Forgejo**（系统服务），前端开发用 **3011**，
> 并同步改 `[site] frontend_url`，否则写请求会被 403。

### 数据库

PostgreSQL 在本机 `127.0.0.1:5432` 运行即可（安装方式不限）：

```bash
createdb boxli_hub
# 建表不需要单独执行：后端启动时自动迁移（幂等）
```

### 种子数据（可选）

```bash
cd backend && go run ./cmd/seed
```

5 个示例镜像 / 10 个标签 / 17 个下载源，覆盖 github、gitee、oss、cos、s3、ipfs、
magnet、http 等源类型。

### 登录联调

**方式 A：本地密码** —— 首次启动时用日志里的引导令牌打开 `/setup` 创建管理员，
之后在 `/login` 用用户名密码登录。

**方式 B：真实 GitHub OAuth** —— 在 GitHub → Settings → Developer settings →
**OAuth Apps** 创建应用，然后填入 `hub.toml`：

- **Homepage URL**：本地填前端实际端口（如 `http://localhost:3011`），生产填站点域名
- **Authorization callback URL**：**必须与 `[github] redirect` 完全一致**，
  否则 GitHub 拒绝回调（报 `redirect_uri_mismatch`）

**方式 C：dev 模拟登录**（仅本地调试）—— 未配 OAuth 且 `[dev] enabled = true` 时：

```bash
curl -X POST http://127.0.0.1:3727/api/v1/auth/login \
  -H "Content-Type: application/json" -d '{"github_user":"alice"}' -c cookies.txt
```

> ⚠️ **这是登录后门**：开启后任何人传一个 `github_user` 即可登录为任意账号。
> 生产绝不能开 —— 因此 `enabled = true` 时**程序会直接拒绝启动**。

> 调试技巧：用 curl 验证 OAuth 凭据是否配对，不会拿到任何真实 token：
>
> ```bash
> curl -s -X POST https://github.com/login/oauth/access_token \
>   -H 'Accept: application/json' \
>   -d "client_id=$CID" -d "client_secret=$CSEC" -d "code=invalid_test_code"
> # {"error":"bad_verification_code"} → 凭据有效（仅 code 无效）
> # {"error":"Not Found"}             → 凭据错误
> ```

> ⚠️ **`[github] secret` 禁止提交**。若曾以任何形式外泄（聊天、日志、截图），
> **必须立即在 GitHub 上 Regenerate** 并更新部署配置。

## 配置文件

后端**不使用任何环境变量**，全部配置来自 TOML 文件。默认读取当前目录的 `hub.toml`。

```bash
./boxli-hub                                     # 读 ./hub.toml
./boxli-hub --config /etc/boxli-hub/hub.toml    # 读指定文件
./boxli-hub --version                           # 打印版本与 commit
./boxli-hub --no-ssr                            # 仅排查：不启动 SSR，页面返回 503
```

二进制只解析这三个参数：`--config`、`--version`、`--no-ssr`。

### 字段一览

| TOML 字段 | 默认 | 说明 |
|---|---|---|
| `[server] addr` | `127.0.0.1:3727` | 监听地址。**只允许回环地址**（填 `0.0.0.0` 会拒绝启动），对外由 Nginx 反代 |
| `[db] url` | 本地连接串 | PostgreSQL 连接串 |
| `[session] jwt_secret` | 空 | JWT 签名密钥（**必填**才能登录，未设则认证接口返回 503；≥16 字符）|
| `[session] ttl_hours` | `720` | 会话有效期（小时）|
| `[session] cookie_secure` | `false` | 会话 Cookie 是否带 `Secure`。生产 HTTPS 必须 `true`；本地 `http://` 必须 `false` |
| `[github] client_id` | 空 | GitHub OAuth Client ID（**可整段留空**，改用本地密码登录）|
| `[github] secret` | 空 | GitHub OAuth Client Secret（**禁止提交/外泄**）|
| `[github] redirect` | 空 | GitHub OAuth 回调地址（须与 App 登记一致；配了凭据则必填）|
| `[site] frontend_url` | `http://127.0.0.1:3727` | 对外访问地址，**同时是写接口的 CSRF 来源白名单** |
| `[site] extra_origins` | `[]` | 额外允许的跨站来源（数组），仅多域名/CI 场景 |
| `[frontend] node_path` | 空 | node 可执行文件路径，留空则从 `PATH` 查找 |
| `[frontend] allow_missing_node` | `false` | 找不到 node 时是否降级启动（不推荐，会损失 SEO）|
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

`users` 表在 `0003` 迁移后新增 `password_hash`（本地密码，可为 NULL）与 `is_admin`。

完整 DDL 与索引见 [`DEVELOPMENT.md`](./DEVELOPMENT.md#三数据模型)。

## API 简表

所有接口在 `/api/v1/` 下，响应统一 `{code, message, data}`（成功 `code=0`）。
**实现细节、请求体示例与错误码见 [`DEVELOPMENT.md`](./DEVELOPMENT.md#八api-参考)。**

**读接口（公开）**

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/health` | 健康检查（⚠️ 恒返 healthy，不探数据库） |
| GET | `/search?q=&limit=` | 搜索公开仓库 |
| GET | `/repos?namespace=&limit=&offset=` | 仓库列表 |
| GET | `/repos/{ns}/{repo}` | 仓库详情（含所有 tags 与 sources） |
| GET | `/repos/{ns}/{repo}/tags` | 标签列表 |
| GET | `/repos/{ns}/{repo}/readme` | README 文本 |

**认证**

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/auth/login` | 已配 OAuth 返回 `authorize_url`；否则需 dev 登录开启才可模拟登录 |
| POST | `/auth/password` | **本地用户名 + 密码登录**（与 OAuth 并存） |
| GET | `/auth/callback?code=&state=` | 校验并消费 state → 换 token → **302 回前端**（会话经 Cookie 下发） |
| GET | `/auth/me` | 当前用户（Cookie 或 Bearer） |
| POST | `/auth/logout` | 登出，吊销 session 并清 Cookie |

**首次部署引导**

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/setup?token=` | 引导页面（自包含 HTML，不依赖前端产物） |
| POST | `/setup` | 创建首个管理员。需携带启动日志中的一次性令牌；系统已有用户时永久关闭 |

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
| `/login` | 登录：GitHub OAuth 入口、本地密码登录、错误提示 | 静态 + `POST /auth/login` / `POST /auth/password` |
| `/submit` | 提交镜像：动态标签与下载源 | `POST /repos`（需登录） |
| `/dashboard` | 用户中心：我的镜像列表 / 编辑 / 删除 | `GET /repos?namespace=` + `PUT`/`DELETE` |
| `/docs` 及 6 个子页 | 文档站：快速开始、安装、拉取与运行、镜像格式、提交、FAQ | `app/utils/docs.ts` 静态内容 |

`/submit` 与 `/dashboard` 受 `auth` 中间件保护，未登录时跳转 `/login?redirect=<原路径>`，
登录成功后原路返回。

**手机优先实现要点：**

- 所有交互元素 `min-h-11`（44px）且带 `min-w-11`；文字链接加 `px-2` 保证宽度
- 输入框 / 下拉框统一 `text-base`（16px），避免 iOS 聚焦缩放
- 全局 `min-h-[100dvh]`（不用 `100vh`）与 `overflow-x-hidden`
- 页脚 `pb-[calc(2rem+env(safe-area-inset-bottom))]` 防 iPhone 横条遮挡
- 手机汉堡菜单用 fuxsto `Drawer`；导航与信息**不依赖 hover**
- 长 URL 用 `break-all`，标签栏 `overflow-x-auto` 横向滚动，避免撑破窄屏

实现细节（SSR 取数、登录守卫、Markdown 安全渲染等）见
[`DEVELOPMENT.md`](./DEVELOPMENT.md#六前端实现)。

## 当前状态

功能已全部实现（官网、Hub 索引、认证与提交、文档站、单二进制部署、首次引导）。

| 项 | 状态 |
|---|---|
| 后端 API、数据库、认证、CSRF / 开放重定向 / state 防护 | ✅ 已实现，实测 + 单测通过 |
| 前端页面（15 个，含 7 个文档页） | ✅ 已实现，`nuxt build` 成功 |
| 单二进制内嵌前端（含 SSR） | ✅ 已实测：空目录 + 单二进制跑通全部页面 |
| 首次部署引导（建表 + 创建管理员） | ✅ 已实测：令牌校验、弱口令拒绝、初始化后关闭 |
| 本地用户名/密码登录（bcrypt） | ✅ 已实现，与 GitHub OAuth 并存 |
| CI 构建 linux/amd64 二进制 | ✅ 已实现，Release 自动发布 |
| **真实 GitHub 授权端到端** | 🔄 代码就绪，**尚缺人工点一次授权** |
| **手机端真机 / Lighthouse** | ⬜ 待做（现有仅为静态断言） |
| **生产服务器部署** | ⬜ 待做，见 [`DEPLOYMENT.md`](./DEPLOYMENT.md) |

**未验收的部分**（不得视为已完成）：

- 真实 GitHub OAuth 全链路（需人工点一次授权）
- 手机端真机测试、Lighthouse Mobile 评分
- 生产环境实际部署

测试用浏览器与 `puppeteer-core` 已移除，需重新准备浏览器环境或用真机。

**上线前必办**：见 [`DEPLOYMENT.md`](./DEPLOYMENT.md#四配置) 的「上线前必改」。
（重新生成 GitHub Secret、更新 callback URL、`cookie_secure = true`、
`frontend_url` 改生产域名、重新生成 `jwt_secret`）

### 构建与测试

```bash
make build     # 前端打包 + 后端内嵌（必须用这个顺序）
make test      # go test ./...
```

31 个测试函数覆盖配置校验、CSRF、会话、OAuth state、开放重定向、密码强度、
安装令牌、embed 完整性。详见 [`DEVELOPMENT.md`](./DEVELOPMENT.md#十测试)。
