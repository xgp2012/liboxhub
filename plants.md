# Boxli 官网 + Hub 索引站 开发说明书

## 一、项目背景

**Boxli** 是一个用 Go 写的轻量级容器引擎，类似 Docker，但自研镜像格式 `.boxli`，不兼容 Docker/OCI。

- 项目地址：https://github.com/LiStudioorg/boxli
- 协议：AGPL-3.0

**要做两件事：**
1. **官网** —— 项目介绍、文档、下载入口
2. **Hub 索引站** —— 收录社区镜像的元数据，**不存储镜像文件本体**

---

## 二、整体架构

```
用户浏览器
    │
    ▼
┌─────────────────────────────────┐
│  Nginx (80/443)                 │
│  /          → 前端静态文件        │
│  /api/*     → 反代到后端          │
│  /docs/*    → 文档               │
└──────────────┬──────────────────┘
               │
               ▼ (127.0.0.1:3727)
┌─────────────────────────────────┐
│  Boxli Hub 后端 (Go)             │
│  只存元数据，不存镜像文件          │
└──────────────┬──────────────────┘
               │
               ▼ (127.0.0.1:5432)
┌─────────────────────────────────┐
│  PostgreSQL                     │
│  用户 / 仓库 / 标签 / 下载源      │
└─────────────────────────────────┘
```

**核心设计：**
- 前后端**完全分离**
- 后端只监听 `127.0.0.1:3727`，外部无法直接访问
- Nginx 把 `/api/` 反代到后端，外部看不到端口
- 数据库只监听 `127.0.0.1:5432`
- 镜像本体放用户自己的地方（GitHub / Gitee / OSS / S3 / IPFS 等）

---

## 三、技术栈

| 部分 | 选型 |
|---|---|
| 前端 | Nuxt 4 |
| UI | fuxsto-design + Tailwind CSS v4 |
| 数据请求 | Nuxt useFetch / useAsyncData（或 TanStack Query Vue）|
| 后端 | Go（已有 `hub/` 包） |
| 数据库 | PostgreSQL 16（实际开发用 PG 17）|
| 反代 | Nginx |
| HTTPS | Let's Encrypt |
| 部署 | systemd + Nginx |

**实际开发环境**（与规格的差异，2026-10-02 迁移后）：

| 项 | 规格 / 原环境 | 当前实际 |
|---|---|---|
| 操作系统 | Windows + WSL2 | **Ubuntu 22.04.5 LTS（原生 Linux）** |
| 数据库 | PG 16，系统安装 | **PG 17.11 免安装二进制**，工作区内自建实例 |
| 数据库监听 | — | 仅 `127.0.0.1:5432`（无对外暴露） |
| 权限 | 有 sudo | **无 sudo**（沙箱 `NoNewPrivs: 1`，无法提权） |
| 前端 dev | 3000 | 3000（另有生产预览 3011 用于验收） |

> 迁移消除了阶段 2 的 L1/L2/L8（WSL PG 全接口监听、`pg_hba` 全放行、Windows 防火墙规则）。
> 详见 README「本地开发」与环境变量一节。

---

## 四、开发原则：手机优先（重要）

**必须手机优先开发，不是先做桌面再适配手机。** 这是强约束，因为：

1. 手机优先 → 桌面只需扩展；桌面优先 → 手机要重构
2. Boxli 目标用户大量使用手机
3. 响应式布局从窄屏开始写最简单

### 开发流程

```
1. 先写手机版（< 640px）
2. 再写平板版（md: ≥ 768px）
3. 最后写桌面版（lg: ≥ 1024px）
```

### Tailwind 断点

| 前缀 | 宽度 | 设备 |
|---|---|---|
| （无） | < 640px | 手机 |
| `sm:` | ≥ 640px | 大手机 |
| `md:` | ≥ 768px | 平板 |
| `lg:` | ≥ 1024px | 桌面 |
| `xl:` | ≥ 1280px | 大屏 |

### 手机端硬性规范

| 规范 | 值 |
|---|---|
| 按钮点击区 | ≥ 44×44px |
| 正文字号 | ≥ 14px |
| 输入框字号 | **≥ 16px**（防 iOS 缩放）|
| 行高 | 1.5 倍字号 |
| 固定底栏 | 加 `pb-[env(safe-area-inset-bottom)]` |
| 避免用 `100vh` | 用 `100dvh` |
| 避免 hover-only 交互 | 用 tap / 点击展开 |

### 手机端 8 个坑（必须避免）

| # | 坑 | 后果 | 解决 |
|---|---|---|---|
| 1 | 没设 viewport | 页面缩成小块 | `<meta name="viewport" content="width=device-width, initial-scale=1">` |
| 2 | 用 hover 显示关键信息 | 手机看不到 | 改为点击展开 |
| 3 | 按钮太小 | 点不准 | ≥ 44×44px |
| 4 | 输入框字号 < 16px | iOS 自动放大 | 改 ≥ 16px |
| 5 | 用 `100vh` | iOS 地址栏错位 | 用 `100dvh` |
| 6 | 固定宽度 | 窄屏溢出 | 用 `max-w-*` + `w-full` |
| 7 | 桌面大图 | 加载慢 | 用清晰度合适的图 + 懒加载 |
| 8 | 横向滚动 | 内容超出 | `overflow-x-hidden` + 检查绝对定位 |

### 手机端测试（强制）

- [ ] iPhone Safari 打开所有页面无白屏
- [ ] Android Chrome 打开所有页面无白屏
- [ ] 页面无横向滚动（`document.body.scrollWidth <= window.innerWidth`）
- [ ] 按钮点击区 ≥ 44px
- [ ] 输入框聚焦时 iOS 不放大
- [ ] 顶部导航在手机上正常（汉堡菜单）
- [ ] 图片无拉伸变形
- [ ] 搜索框键盘弹出不遮挡结果
- [ ] 底部无内容被 iPhone 横条遮挡
- [ ] Lighthouse Mobile Performance ≥ 90
- [ ] 控制台无 Hydration 报错

### 无实机测试替代方案（强制）

当前若无 iPhone / Android 真机，用以下等效方案覆盖全部测试项，**验收时以本方案结果为准**：

| 测试项 | 替代方案 |
|---|---|
| iPhone Safari 白屏 | Chrome DevTools 设备模拟 **iPhone 14 Pro (390×844)**，或用浏览器 CDP 模拟 WebKit（Playwright `webkit`） |
| Android Chrome 白屏 | Chrome DevTools 设备模拟 **Pixel 7 (412×915)**（真 Chrome 内核，等效度高） |
| 无横向滚动 | 上面的模拟尺寸下执行控制台脚本：`document.body.scrollWidth <= window.innerWidth` |
| 按钮点击区 ≥ 44px | 模拟下用 DevTools 量取，或跑 `document.querySelectorAll('button,a')` 脚本检查 `getBoundingClientRect()` |
| iOS 输入框不放大 | 静态检查：所有 `input/textarea/select` 字号 ≥ 16px（`Input`/`Textarea` 组件默认保证）|
| 底部不被横条遮挡 | 模拟 safe-area：DevTools 添加自定义设备并注入 `env(safe-area-inset-bottom)` 测试值 |
| 汉堡菜单 | 模拟窄屏点击展开，`Drawer` 组件功能验证 |
| Lighthouse Mobile | Chrome DevTools **Lighthouse** 面板，模式选 Mobile，直接出分 |
| Hydration 报错 | 本地 `npm run dev` + 生产 `npm run build && npm run preview` 两轮都查控制台 |

**补充工具（可选，进一步逼近真机）：**

- **Playwright** 自动化：配置 `devices['iPhone 14 Pro']` / `devices['Pixel 7']`，可在 CI 跑回归
- **BrowserStack / LambdaTest** 云真机：需要最终确认时按小时租用，成本可控
- **本机 Chrome 远程调试**：USB 连任意 Android 手机，`chrome://inspect` 调真机（成本最低的真机方案）

> 替代方案不能覆盖的点：iOS Safari 真实滚动惯性、地址栏动态收缩（`100dvh` 表现）、真实触摸延迟。这几项**上线前需至少一台真机最终确认一次**。

---

## 五、前端页面清单

| 路由 | 页面 | 功能 |
|---|---|---|
| `/` | 首页 | 项目介绍、特性、快速开始、热门镜像 |
| `/explore` | 浏览 | 所有镜像，支持搜索/筛选 |
| `/explore/[ns]/[repo]` | 镜像详情 | README、标签、多源下载 |
| `/search?q=xxx` | 搜索 | 关键词搜索 |
| `/docs` | 文档 | 使用文档 |
| `/login` | 登录 | GitHub OAuth ✅ 已实现 |
| `/dashboard` | 用户中心 | 管理自己收录的镜像 ✅ 已实现 |
| `/submit` | 提交镜像 | 表单，支持多个下载源 ✅ 已实现 |
| `/about` | 关于 | 项目背景、协议、赞助 |

### 首页要素

- **Hero**：一句话介绍 + 快速开始
  ```
  boxli pull alice/myapp:v1
  boxli run alice/myapp:v1
  ```
- **特性**：跨平台、轻量、自研镜像格式、多源下载
- **热门镜像**：API 拉 Top 10
- **底部**：GitHub、文档、AGPL 协议、赞助商

### 镜像详情页

- 名称、描述、作者
- README（Markdown 渲染）
- 标签列表（latest、v1.0.0…）
- 支持架构（amd64 / arm64 / riscv64）
- **多个下载源**（见第七节）
- 复制 pull 命令
- 下载量、更新时间

---

## 六、后端 API 约定

所有接口在 `/api/v1/` 下。

### 基础

```
GET  /api/v1/health              # 健康检查
```

### 认证

```
POST /api/v1/auth/login          # 登录：已配 OAuth 返回 authorize_url；否则（需 BOXLI_DEV_LOGIN=1）模拟登录
GET  /api/v1/auth/callback       # OAuth 回调：校验 state → 换 token → 302 回前端（httpOnly Cookie 交付会话）
GET  /api/v1/auth/me             # 获取当前用户（Cookie 或 Bearer）
POST /api/v1/auth/logout         # 登出：吊销 session 并清除 Cookie
```

> **会话交付**：`/auth/callback` 与 dev 登录均以 **httpOnly Cookie**（`boxli_session`）下发会话，
> 同时返回 `token` 字段供 curl / CLI 以 `Authorization: Bearer` 使用。
> 写接口需带 Cookie（浏览器自动）或 Bearer；跨源写请求另受来源白名单校验（CSRF 防护）。

### 镜像

```
GET  /api/v1/search?q=xxx&limit=20         # 搜索
GET  /api/v1/repos                          # 列出所有仓库
GET  /api/v1/repos/{ns}/{repo}              # 仓库详情
GET  /api/v1/repos/{ns}/{repo}/tags         # 标签列表
GET  /api/v1/repos/{ns}/{repo}/readme       # README
```

### 提交镜像

```
POST /api/v1/repos                          # 提交（需登录）
PUT  /api/v1/repos/{ns}/{repo}              # 更新
DELETE /api/v1/repos/{ns}/{repo}            # 删除（需 owner）
```

### 响应格式

成功：
```json
{ "code": 0, "message": "ok", "data": { ... } }
```

失败：
```json
{ "code": 404, "message": "repository not found", "data": null }
```

---

## 七、镜像多源设计（关键）

**镜像不强制存 GitHub。** 用户可以放任何地方，Hub 只记录"去哪下载"。

### 支持的源类型

| 类型 | 说明 |
|---|---|
| `github` | GitHub Releases（全球）|
| `gitlab` | GitLab Releases |
| `gitee` | Gitee Releases（国内快）|
| `http` | 任意 URL |
| `s3` | AWS S3 |
| `oss` | 阿里云 OSS |
| `cos` | 腾讯云 COS |
| `ipfs` | `ipfs://Qm...` |
| `magnet` | BT |
| `direct` | 直链 |

### 数据模型

```json
{
  "name": "alice/myapp",
  "tag": "v1.0.0",
  "arch": "amd64",
  "os": "linux",
  "size": 5242880,
  "digest": "sha256:abc...",
  "sources": [
    {
      "type": "github",
      "url": "https://github.com/alice/myapp/releases/download/v1/myapp.boxli",
      "priority": 1,
      "region": "global"
    },
    {
      "type": "gitee",
      "url": "https://gitee.com/alice/myapp/releases/download/v1/myapp.boxli",
      "priority": 2,
      "region": "cn"
    }
  ]
}
```

| 字段 | 含义 |
|---|---|
| `type` | 源类型，前端显示图标 |
| `url` | 实际下载地址 |
| `priority` | 优先级，数字越小越优先 |
| `region` | `global` / `cn` / `us` / `eu` |
| `digest` | 文件 SHA256（可选） |
| `size` | 文件大小 |

### 前端展示

```
下载 myapp:v1.0.0 (amd64, 5.0 MB)

  🌍 GitHub Releases    [推荐]    →  点击下载
  🇨🇳 Gitee              [国内快]   →  点击下载
  ☁️  阿里云 OSS         [国内快]   →  点击下载
  🔗 直链                         →  复制链接

  ┌─────────────────────────────┐
  │ boxli pull alice/myapp:v1   │  [复制]
  └─────────────────────────────┘
```

### 提交表单

```
镜像名：       alice/myapp
版本：         v1.0.0
架构：         [x] amd64  [x] arm64
描述：         My awesome app

下载源（可加多个）：
  [+] GitHub Releases
      URL: https://github.com/...
  [+] 添加更多源
      [选择类型 ▾] [URL___________]
```

### 为什么这么设计

| 好处 | 说明 |
|---|---|
| 去中心化 | Hub 挂了，URL 照样能下 |
| 多区域加速 | 国内走 Gitee/OSS |
| 抗单点 | 某源挂了自动切 |
| 零成本 | Hub 不承担带宽 |
| 灵活 | 网盘、IPFS、BT 都行 |
| 可审计 | digest 校验 |

---

## 八、数据库设计（PostgreSQL）

### 表结构

**users（用户）**
```sql
CREATE TABLE users (
    id           BIGSERIAL PRIMARY KEY,
    username     VARCHAR(64) UNIQUE NOT NULL,
    email        VARCHAR(255) UNIQUE,
    avatar_url   TEXT,
    github_id    BIGINT UNIQUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**repositories（仓库）**
```sql
CREATE TABLE repositories (
    id           BIGSERIAL PRIMARY KEY,
    namespace    VARCHAR(64) NOT NULL,
    name         VARCHAR(128) NOT NULL,
    description  TEXT,
    readme       TEXT,
    owner_id     BIGINT NOT NULL REFERENCES users(id),
    is_public    BOOLEAN NOT NULL DEFAULT TRUE,
    stars        INT NOT NULL DEFAULT 0,
    pulls        BIGINT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(namespace, name)
);
CREATE INDEX idx_repos_ns_name ON repositories(namespace, name);
```

**tags（标签）**
```sql
CREATE TABLE tags (
    id           BIGSERIAL PRIMARY KEY,
    repo_id      BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    tag          VARCHAR(64) NOT NULL,
    digest       VARCHAR(128),
    size_bytes   BIGINT,
    os           VARCHAR(16) NOT NULL,
    arch         VARCHAR(16) NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(repo_id, tag, os, arch)
);
```

**sources（下载源）**
```sql
CREATE TABLE sources (
    id           BIGSERIAL PRIMARY KEY,
    tag_id       BIGINT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    type         VARCHAR(16) NOT NULL,
    url          TEXT NOT NULL,
    priority     INT NOT NULL DEFAULT 100,
    region       VARCHAR(16),
    digest       VARCHAR(128),
    size_bytes   BIGINT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sources_tag ON sources(tag_id);
```

**sessions（会话）**
```sql
CREATE TABLE sessions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash   VARCHAR(128) NOT NULL,
    expires_at   TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 关系图

```
users (1) ──< repositories (1) ──< tags (1) ──< sources
  │
  └──< sessions
```

### 常用查询

**搜索：**
```sql
SELECT namespace, name, description, stars, pulls
FROM repositories
WHERE is_public = TRUE
  AND (name ILIKE '%' || $1 || '%'
       OR description ILIKE '%' || $1 || '%')
ORDER BY stars DESC LIMIT 20;
```

**详情 + 所有 tag + 所有源：**
```sql
SELECT r.*,
    json_agg(json_build_object(
        'tag', t.tag, 'os', t.os, 'arch', t.arch,
        'sources', (
            SELECT json_agg(json_build_object(
                'type', s.type, 'url', s.url,
                'priority', s.priority, 'region', s.region
            ) ORDER BY s.priority)
            FROM sources s WHERE s.tag_id = t.id
        )
    )) AS tags
FROM repositories r
LEFT JOIN tags t ON t.repo_id = r.id
WHERE r.namespace = $1 AND r.name = $2
GROUP BY r.id;
```

---

## 九、Nginx 配置（隐藏后端）

```nginx
# /etc/nginx/sites-available/boxli

server {
    listen 80;
    server_name boxli.dev www.boxli.dev;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name boxli.dev www.boxli.dev;

    ssl_certificate     /etc/letsencrypt/live/boxli.dev/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/boxli.dev/privkey.pem;
    ssl_protocols       TLSv1.2 TLSv1.3;

    root /var/www/boxli/frontend;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    # ===== API 反向代理（隐藏后端） =====
    location /api/ {
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
        proxy_hide_header Server;
    }

    location /docs/ {
        root /var/www/boxli/docs;
        try_files $uri $uri/ /docs/index.html;
    }

    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff2?)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

**启用：**
```bash
sudo ln -s /etc/nginx/sites-available/boxli /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

**关键点：**
- 后端监听 `127.0.0.1:3727`（不是 `0.0.0.0`）
- 外部只能通过 `/api/` 访问
- 用户永远看不到 `:3727`
- 扫端口也扫不到

---

## 十、后端部署

### systemd 服务

`/etc/systemd/system/boxli-hub.service`：

```ini
[Unit]
Description=Boxli Hub
After=network.target postgresql.service

[Service]
Type=simple
User=boxli
Group=boxli
EnvironmentFile=/etc/boxli-hub.env
ExecStart=/usr/local/bin/boxli hub serve \
    --addr 127.0.0.1:3727 \
    --data-dir /var/lib/boxli/hub \
    --db "postgres://boxli:${DB_PASSWORD}@127.0.0.1:5432/boxli_hub?sslmode=disable"
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### 环境变量文件

`/etc/boxli-hub.env`：

```bash
DB_HOST=127.0.0.1
DB_PORT=5432
DB_NAME=boxli_hub
DB_USER=boxli
DB_PASSWORD=<生成的随机密码>
```

权限：
```bash
sudo chmod 600 /etc/boxli-hub.env
sudo chown root:root /etc/boxli-hub.env
```

### ufw 只开 80/443

```bash
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw status
# 不应出现 3727 或 5432
```

---

## 十一、前端部署

### 方式一：Nuxt 4 Node 服务（推荐，SSR）

```bash
cd /var/www/boxli/frontend
npm install
npm run build
# 用 systemd 跑 node .output/server/index.mjs，监听 127.0.0.1:3000
```

Nginx：
```nginx
location / {
    proxy_pass http://127.0.0.1:3000;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}
```

### 方式二：纯静态生成（SSG）

```bash
npm run generate
# 产物在 .output/public/，拷到 /var/www/boxli/frontend/
```

Nginx 直接托管静态文件。

---

## 十二、开发流程

### fuxsto-design 接入（Nuxt 4）

**1. 安装依赖**

```bash
npm install fuxsto-design tailwindcss @tailwindcss/vite lucide-vue-next
```

> peer 依赖：`vue ^3.5`、`tailwindcss ^4`、`lucide-vue-next`。

**2. 配置 nuxt.config.ts**（注册 Vite 插件 + 全局样式）

```ts
import tailwindcss from '@tailwindcss/vite'

export default defineNuxtConfig({
  css: ['fuxsto-design/styles.css'],
  vite: {
    plugins: [tailwindcss()]
  }
})
```

**3. 引入样式**（`assets/css/main.css`）

```css
@import 'tailwindcss';
/* fuxsto-design 组件已随上方 css 数组全局注入 */
```

**4. 按需使用组件**（组件均为 Vue SFC，直接 import）

```vue
<script setup lang="ts">
import { Button, Card, Chip, Input } from 'fuxsto-design'
</script>

<template>
  <Card>
    <Input v-model="q" placeholder="搜索镜像" />
    <Button @click="search">搜索</Button>
    <Chip>latest</Chip>
  </Card>
</template>
```

**常用子路径导入**（体积更小）：

```ts
import Button from 'fuxsto-design/button'
import Card from 'fuxsto-design/card'
import Chip from 'fuxsto-design/chip'
import Input from 'fuxsto-design/input'
import Dialog from 'fuxsto-design/dialog'
import Drawer from 'fuxsto-design/drawer'
import Table from 'fuxsto-design/table'
```

**可复用组件映射**（本项目）：

| 场景 | fuxsto-design 组件 |
|---|---|
| 顶部导航 / 汉堡菜单 | `Header`、`Drawer` |
| 镜像卡片 | `Card`、`Avatar`、`Chip` |
| 标签列表 | `Chip`、`ChipGroup` |
| 搜索框 | `Input`、`AutoComplete` |
| 提交表单 | `Form`、`FormItem`、`Select`、`Input`、`Textarea` |
| 下载源列表 | `List`、`ListItem`、`Divider` |
| 加载 / 空状态 | `Skeleton`、`Loading`、`Empty` |
| 提示反馈 | `Message`、`Notification`、`Alert` |
| 分页 | `Pagination` |

**注意：**

- `fuxsto-design` 主题为 **monochrome zinc**，与 Tailwind 原生类名一致，用 `class` 覆盖时沿用 zinc 色阶
- 手机优先下 `Drawer` 用于移动端菜单，`Dialog` 用于桌面端弹窗
- 组件库 `sideEffects: ["**/*.css"]`，样式不可被 tree-shaking 裁剪，构建无需额外配置

### 前端本地开发

```bash
npm install
npm run dev
# http://localhost:3000

# .env
NUXT_PUBLIC_API_BASE=https://boxli.dev/api/v1
```

用 Nuxt `routeRules` 免 CORS（dev 走 nitro proxy）：

```ts
// nuxt.config.ts
export default defineNuxtConfig({
  runtimeConfig: {
    public: { apiBase: '/api/v1' }
  },
  routeRules: {
    '/api/**': { proxy: 'http://127.0.0.1:3727/api/**' }
  }
})
```

### 真机调试

```bash
npm run dev -- --host 0.0.0.0
# 手机浏览器访问 http://电脑局域网IP:3000
```

### API 调用

```ts
// composables/useApi.ts
export const searchRepos = (q: string) =>
  useFetch('/api/v1/search', { query: { q } })

export const getRepo = (ns: string, repo: string) =>
  useFetch(`/api/v1/repos/${ns}/${repo}`)
```

---

## 十三、验收标准

### 功能

- [ ] 首页能打开，显示项目介绍
- [ ] `/explore` 列出镜像
- [ ] 搜索返回结果
- [ ] 镜像详情显示 README、标签、多源
- [ ] 提交表单能加 ≥ 2 个源
- [ ] 点击"下载"跳转对应源
- [ ] 复制 `boxli pull xxx` 能用
- [ ] `https://boxli.dev/api/v1/health` 返回 200
- [ ] `curl http://<服务器IP>:3727` **超时**（后端隐藏成功）

### 手机端（强制）

- [ ] iPhone Safari 全页面无白屏
- [ ] Android Chrome 全页面无白屏
- [ ] 无横向滚动
- [ ] 按钮点击区 ≥ 44px
- [ ] 输入框 iOS 不放大
- [ ] 底部不被 iPhone 横条遮挡
- [ ] Lighthouse Mobile ≥ 90

### 数据库

- [ ] PostgreSQL 只监听 127.0.0.1:5432
- [ ] ufw 未开 5432
- [ ] 每日自动备份脚本已配
- [ ] 后端启动时自动跑 schema migration

---

## 十四、开发阶段划分

> 原则：**每一阶段都有可验证产出**，不通过不进入下一阶段。手机优先规范从阶段 3 起全程贯穿。

### 阶段 0：环境与脚手架（0.5 天）✅ 已完成（2026-10-02）

**产出**：前后端空项目可运行

- [x] 初始化 Nuxt 4 项目（`npm create nuxt`）+ 接入 fuxsto-design / Tailwind v4
- [x] 初始化 Go 后端模块（`hub/` 包），跑通 `GET /api/v1/health`
- [x] 本地 PostgreSQL 建库 `boxli_hub`
- [x] `nuxt.config.ts` 配 `routeRules` proxy 打通前后端
- [x] 建立目录结构与代码规范（eslint / gofmt）

**验证**：`npm run dev` 首页空白可开；`curl localhost:3000/api/v1/health` 返回 200

> **实际产出与验证记录：**
>
> - 目录：`backend/`（Go，module `github.com/LiStudioorg/boxli`）+ `frontend/`（Nuxt 4.5.2）+ `README.md`
> - 后端：`cmd/hub/main.go`（监听 `127.0.0.1:3727`、优雅关闭）、`internal/hub/server.go`（`/api/v1/health`，统一 `{code,message,data}`）、`internal/config/`
> - 前端：`nuxt.config.ts`（fuxsto-design + `@tailwindcss/vite` + `routeRules` proxy）、`app/app.vue`、`app/pages/index.vue`（展示后端健康状态）、`app/assets/css/main.css`、`eslint.config.mjs`
> - 已安装：`fuxsto-design@1.0.4`、`lucide-vue-next@^0.577.0`（对齐 fuxsto peer 依赖）、`nuxt@^4`、`tailwindcss@^4`、`@nuxt/eslint`
> - **验证结果**：`gofmt`/`go vet`/`go build` 通过；`curl 127.0.0.1:3727/api/v1/health` → 200；`nuxt build` 成功；`curl localhost:3000/api/v1/health` 经 proxy → 200；首页 → 200
>
> **环境说明（与规格差异）：**
>
> - 本机未装 PostgreSQL，改用 **WSL2 Debian** 内的 PostgreSQL **17**（Debian 13 仓库版本，向上兼容规格的 16）
> - 库 `boxli_hub` + 角色 `boxli`/`boxli` 已建；Windows 侧 `127.0.0.1:5432` 可达（WSL2 localhost 转发）
> - WSL 重启后需执行：`wsl -d Debian -u root -- bash -lc "pg_ctlcluster 17 main start"`（已写入 README）
> - 连接串：`postgres://boxli:boxli@127.0.0.1:5432/boxli_hub?sslmode=disable`

---

### 阶段 1：数据库与数据层（1 天）✅ 已完成（2026-10-02）

**产出**：schema + migration + 种子数据

- [x] 按第八节建 5 张表：users / repositories / tags / sources / sessions
- [x] 后端启动自动跑 schema migration
- [x] 写种子数据脚本（≥ 5 个示例镜像，含多源）
- [x] 索引：`idx_repos_ns_name`、`idx_sources_tag`

**验证**：`psql` 查到表结构；种子脚本可重复执行不报错

> **实际产出与验证记录：**
>
> - 迁移方案：**Go 内嵌 SQL + 自研 runner**（`go:embed`），零额外依赖
> - 驱动：**pgx/v5**（`pgxpool` 连接池）
> - 新增文件：
>   - `internal/db/db.go`：`Pool()` 连接池 + `Migrate()` 内嵌迁移执行器
>   - `internal/db/migrations/0001_init.sql`：5 张表 + 索引
>   - `internal/seed/seed.go` + `cmd/seed/main.go`：幂等 Go 种子程序
> - 迁移记录表 `schema_migrations`，迁移在事务内执行，可重复运行
> - 配置扩展：`config.go` 新增 `JWTSecret`/`GitHubClientID`/`GitHubSecret`/`GitHubRedirect`/`SessionTTLHours`（阶段 2 使用，阶段 1 仅读取 env）
> - `cmd/hub/main.go` 启动时自动连库 + 迁移
> - **验证结果**：`gofmt`/`go vet`/`go build` 通过
>   - 建表：6 张表（5 业务表 + `schema_migrations`），索引 13 个（含 `idx_repos_ns_name`、`idx_sources_tag`）
>   - 种子：5 users / 5 repos / 10 tags / 17 sources；**重复执行计数不变（幂等）**
>   - 多源确认：`alice/myapp:v1.0.0 amd64` 有 3 个源；`dave/postgres-boxli:v16 amd64` 有 3 个源
>   - 后端启动：`database migrated` → `GET /api/v1/health` 返回 200
> - 验证命令：`cd backend && go run ./cmd/seed`（建表+种子）；`go run ./cmd/hub`（启动，自动迁移）

---

### 阶段 2：后端 API（2 天）✅ 已完成（2026-10-02）

**产出**：`/api/v1/` 全部接口可用（统一 `{code,message,data}` 格式）

- [x] 基础：`GET /health`
- [x] 镜像读接口：`/search`、`/repos`、`/repos/{ns}/{repo}`、`/tags`、`/readme`
- [x] 认证：`POST /auth/login`（GitHub OAuth）、`GET /auth/me`、JWT + sessions
- [x] 写接口（需鉴权）：`POST/PUT/DELETE /repos`
- [x] 只监听 `127.0.0.1:3727`

**验证**：Postman/curl 逐个接口走通；未登录写接口返回 401

> **实际产出与验证记录：**
>
> - 依赖：新增 `github.com/golang-jwt/jwt/v5`
> - 新增 `internal/auth/`：
>   - `auth.go`：JWT 签发/校验（HS256）+ `sessions` 表读写（存 sid 的 SHA-256），支持吊销
>   - `github.go`：GitHub OAuth 客户端（authorize url / code 换 token / 拉用户）
>   - `middleware.go`：Bearer token 中间件，用户注入 context
> - 新增 `internal/hub/`：
>   - `store.go`：search / listRepos / getRepo（含 tags+sources）/ readme 查询
>   - `store_write.go`：createRepo / updateRepo / deleteRepo（事务内写，owner 校验，tags/sources 整体替换）
>   - `handlers_read.go` / `handlers_write.go` / `handlers_auth.go`
>   - `server.go`：路由分发（`/repos/{ns}/{repo}[/tags|/readme]`）+ CORS + 日志
> - **认证方案**：JWT + sessions 双写。JWT 携带 `sid`，服务端 sessions 表存 `sid` 哈希；校验同时验签名与 session，登出删除 session 即失效
> - **dev 模拟登录**：未配 `BOXLI_GITHUB_CLIENT_ID/SECRET` 时，`POST /auth/login` 接受 `{"github_user":"alice"}` 直接签发 token，便于无凭证联调；配好凭证后自动切换为真实 OAuth 流程
> - **验证结果**（curl 实测，均通过）：
>   - 读：`/health` 200；`/search?q=postgres` 命中 1；`/repos?limit=2`、`?namespace=carol` 正确；`/repos/alice/myapp` 详情含 3 tags + 5 sources；`/tags`、`/readme` 正常；不存在仓库 404
>   - 写：登录拿 token → `POST /repos`（2 源）201；重复创建 409；`PUT` 替换 tags/sources 200；`DELETE` 200；删除后 GET 404
>   - 鉴权：无 token 写接口 401；非 owner 改/删 403；空 tags 400；改不存在仓库 404；`/auth/logout` 后旧 token 访问 `/auth/me` 401
>   - 监听：`netstat` 确认仅 `127.0.0.1:3727` LISTENING
>   - `gofmt -l` 干净 / `go vet` / `go build` 通过
> - **环境注意（WSL2 端口转发）**：WSL VM 空闲自动关闭会导致 Windows→WSL 的 `127.0.0.1:5432` 转发失效。开发时需保持常驻进程：
>   `Start-Process wsl.exe -ArgumentList '-d','Debian','-u','root','--','bash','-lc','nohup pg_ctlcluster 17 main start; exec sleep 86400' -WindowStyle Hidden`（已写入 README）
>
> **⚠️ 遗留项 / 风险标注（本机开发环境专用，上线前必须处理）：**
>
> | # | 项 | 类型 | 说明 | 处理时机 |
> |---|---|---|---|---|
> | ~~L1~~ | ~~WSL 内 PG `listen_addresses = '*'`~~ | 环境改动 | **✅ 已消除**：环境迁移到原生 Linux，当前 PG 实例仅监听 `127.0.0.1:5432` | — |
> | ~~L2~~ | ~~WSL 内 `pg_hba.conf` 追加全放行~~ | 安全 | **✅ 已消除**：新实例由 `initdb` 生成，host 认证为 `scram-sha-256`，未加全放行 | — |
> | ~~L3~~ | ~~认证仅有 dev 模拟登录~~ | 功能 | **✅ 已修复（阶段 4）**：接入真实 GitHub OAuth，配置 `CLIENT_ID`+`SECRET` 后 `/auth/login` 自动返回 `authorize_url`；未配凭证时保留 dev 模拟登录作为回退 | — |
> | ~~L4~~ | ~~`/auth/callback` 的 `state` 未做服务端校验~~ | 安全 | **✅ 已修复（阶段 4）**：新增 `oauth_states` 表（migration `0002`）+ `internal/auth/state.go`，`DELETE ... RETURNING` 原子一次性消费，10 分钟 TTL；伪造/缺失/重放/过期均 400 | — |
> | L5 | CORS 当前为 `Access-Control-Allow-Origin: *` | 安全 | **🔄 已部分推进（阶段 4）**：配 Cookie 后 `*` 与凭证请求不兼容，已改为「白名单回显 Origin + `Allow-Credentials` + `Vary: Origin`」；生产由 Nginx 同源反代后不再产生跨源请求，届时可彻底移除 | **阶段 7 收紧或移除** |
> | L6 | 后端无速率限制 / 无请求 ID 日志 | 加固 | 搜索、写接口无限流；日志仅方法+路径 | 阶段 6/7 视需要 |
> | ~~L7~~ | ~~`upsertUser` 的 username 冲突回退逻辑~~ | 逻辑 | **✅ 已修复（阶段 4）**：原逻辑在 username 唯一冲突时退化为「按 username 查找并登录」，等于让新 GitHub 账号接管同名老账号（账号接管漏洞）。现改为返回 `409 username already taken` | — |
> | ~~L8~~ | ~~Windows 防火墙临时规则 `WSL-PG5432`~~ | 环境改动 | **✅ 已消除**：已脱离 Windows 环境 | — |
>
> 还原命令（阶段 7 参考）：
> ```powershell
> # L8
> netsh advfirewall firewall delete rule name="WSL-PG5432"
> # L1 / L2：编辑 WSL 内 /etc/postgresql/17/main/postgresql.conf 与 pg_hba.conf 后重启
> wsl -d Debian -u root -- bash -lc "sed -i \"s/^listen_addresses = '\\*'/#listen_addresses = 'localhost'/\" /etc/postgresql/17/main/postgresql.conf"
> ```

---

### 阶段 3：前端核心页面（3 天，手机优先）✅ 已完成（2026-10-02）

> **📌 标注**：从本阶段起，第四节「手机优先」为全程强约束——先写 <640px，再 `md:`/`lg:`；每个页面完成即在 DevTools 手机模拟下自测。
> 依赖阶段 2 已就绪的读接口：`/search`、`/repos`、`/repos/{ns}/{repo}[/tags|/readme]`。

**产出**：可浏览、可搜索、可看详情

- [x] 全局布局：`Header` + 手机汉堡 `Drawer`，`pb-[env(safe-area-inset-bottom)]`
- [x] 首页 `/`：Hero、特性、热门 Top 10、底部
- [x] `/explore`：镜像列表 + 搜索/筛选
- [x] 镜像详情 `/explore/[ns]/[repo]`：README 渲染、标签、多源下载、复制 pull 命令
- [x] `/search?q=xxx`
- [x] `/about`

**验证**：DevTools 手机模拟下三页可正常浏览，无横向滚动

> **实际产出与验证记录：**
>
> **环境迁移（重要）**：开发环境已从 Windows + WSL2 **迁移到原生 Linux（Ubuntu 22.04.5）**，
> 阶段 2 遗留项的 **L1 / L2 / L8 随迁移一并消除**（WSL PG 全接口监听、`pg_hba` 0.0.0.0/0 放行、
> Windows 防火墙 `WSL-PG5432` 规则均不复存在）。剩余 L3–L7 处理时机不变。
>
> **数据层重建**：本机无 root/sudo（沙箱 `NoNewPrivs: 1`，sudo 无法提权），Docker 亦无 socket 权限，
> 故采用 **PG 17.11 免安装二进制**（theseus-rs/postgresql-binaries，sha256 校验通过）解压至
> `.devtools/`，以当前用户 `initdb` + `pg_ctl` 启动，**只监听 `127.0.0.1:5432`**。
> `go run ./cmd/seed` 幂等执行，种子数据与阶段 1 记录一致（5 users / 5 repos / 10 tags / 17 sources）。
>
> **新增前端文件**：
> - `app/layouts/default.vue`：全局布局（Header + main + Footer）
> - `app/components/`：`SiteHeader.vue`（含手机汉堡 Drawer）、`SiteFooter.vue`（safe-area 内边距）、
>   `RepoCard.vue`、`SourceList.vue`（标签切换 + 多源下载 + 复制）、`CopyButton.vue`、
>   `EmptyState.vue`、`CardSkeleton.vue`
> - `app/composables/useApi.ts`：API 封装，统一解包 `{code,message,data}`，
>   `useSearch` 在关键词为空时不发请求
> - `app/composables/useMarkdown.ts`：markdown-it（`html:false`）+ DOMPurify 清洗
> - `app/types/api.ts`：后端响应类型；`app/utils/format.ts`：字节/数量/相对时间格式化 + 源类型元数据
> - `app/pages/`：`index.vue`、`explore/index.vue`、`explore/[ns]/[repo].vue`、`search.vue`、`about.vue`
> - 新增依赖：`markdown-it`、`isomorphic-dompurify`
>
> **修复的实际缺陷（验证中发现）**：
> 1. 首页误用 `repos.length`（`repos` 是 `{count,items}` 对象，`.length` 恒为 `undefined`），
>    导致始终走空状态分支 → 改为 `repos.items`
> 2. `useSearch` 原为 `immediate: false`，SSR 首屏不请求，`/search?q=` 直链无结果 → 改为
>    有词才请求（`immediate: hasQuery`）
> 3. `/explore` 原为条件调用 composable（`query ? useSearch() : useRepoList()`），
>    违反 composable 规则且首屏状态错乱 → 改为两个 composable 都调用、按关键词择一取用
> 4. 纯文字链接（面包屑「浏览」、页脚「文档」等）宽度可 < 44px → 补 `min-w-11` + `px-2`
>
> **验证结果**：
> - **路由**（生产构建 `node .output/server/index.mjs`，:3011）：`/`、`/explore`、`/about`、
>   `/search`、`/search?q=postgres`、`/explore/alice/myapp`、`/explore/dave/postgres-boxli`
>   全部 **200**；不存在的仓库正确渲染「镜像不存在」空状态（页面 200，符合 UX 预期）
> - **数据渲染**：首页 5 个镜像卡片全部渲染；`/explore` 全量 5 个；`/search?q=postgres` 精确命中
>   `dave/postgres-boxli`；`/search?q=redis` 命中 `carol/redis-edge`；`/explore?q=redis` 正确过滤
> - **详情页**：README 渲染为 HTML（`<h1>myapp</h1>` + 代码块）；pull 命令 `boxli pull alice/myapp:v1`
>   与 `:latest` 均正确；多源 payload 含 `github`/`gitee`/`oss` 三种源；架构 `linux/amd64`、`linux/arm64`
> - **手机端清单（静态断言，对应第四节「无实机测试替代方案」）**：
>   viewport 全部正确注入；`100vh` 出现 **0** 次、`100dvh` 存在；固定 px 宽度 **0** 处；
>   5 个页面均含 `overflow-x-hidden`；所有输入框/下拉框为 `text-base`（16px，防 iOS 缩放）；
>   交互元素统一 `min-h-11`（44px）；无 `group-hover` 等 hover-only 显隐（坑 2 已规避）
> - **XSS 防护实测**：构造带 `<script>alert('xss')</script>`、`<img onerror>`、`<iframe>` 的 README
>   提交后渲染验证 —— `<iframe>` 被过滤为 0；`script`/`onerror`/`alert(` 仅存在于
>   `__NUXT_DATA__` 的 JSON payload 中且 `<` 已转义为 `\u003C`（纯数据非可执行）；
>   正常内容（`<h1>`、外链）保留且外链自动补 `rel="noopener noreferrer nofollow"`。测试数据已清理
> - **真实浏览器测试（已执行，随后按用户要求移除浏览器）**：曾用 `chrome-headless-shell` 154 +
>   `puppeteer-core` 在 iPhone 14 Pro (390×844) / Pixel 7 (412×915) 双视口下跑完整清单，
>   **64 项中 60 项通过、4 项失败**，捕获到静态检查遗漏的真实缺陷：
>   `CopyButton.vue` 的复制按钮在手机上隐藏文字后宽度仅 **34px**（`34×46`），低于 44px 门槛
>   → 已补 `min-w-11` + `justify-center` 修复。
>   **⚠️ 该修复未经复测即按用户要求移除浏览器**（详见下方遗留项）。
> - **工程质量**：`eslint .` **0 error / 0 warning**；`nuxt build` 成功
>
> **环境注意（沙箱限制，与项目本身无关）**：
> - Go 需 `GOCACHE`/`GOMODCACHE`/`GOPATH` 指向工作区内（`~/.cache`、`~/go` 不可写）
> - npm 需 `npm_config_cache` 指向工作区内（`~/.npm` 不可写）
> - `node_modules/.bin/nuxt` 无执行位，改用 `node node_modules/nuxt/bin/nuxt.mjs`
>
> **⚠️ 手机端验收状态（未完成，阶段 6 必须补齐）**：
> 阶段 3 结束时，手机端清单的验证方式为**渲染 HTML 结构化断言**（静态）+ 一轮**真实浏览器测试**
> （已执行，结果见上）。但用户随后要求删除测试用浏览器与 `puppeteer-core` 依赖、跳过手机测试，
> 因此以下项**尚未通过真实浏览器/真机确认**：
> - `CopyButton` 宽度修复**未复测**（修复方向明确，但缺一次验证）
> - **Lighthouse Mobile ≥ 90** 从未测量
> - 真实触摸延迟、iOS 滚动惯性、地址栏收缩下的 `100dvh` 动态表现
> - iOS 输入框聚焦是否真的不放大（当前仅静态确认字号 ≥16px）
>
> 这些项需在**阶段 6** 用真实浏览器或真机重新补齐。不得视为已验收。

---

### 阶段 4：认证与提交（2 天）🔄 后端 OAuth 已完成

> **📌 标注**：本阶段需落地阶段 2 的遗留项 **L3/L4**——接入真实 GitHub OAuth（配置 `BOXLI_GITHUB_CLIENT_ID/SECRET/REDIRECT`），并在 `/auth/login` ↔ `/auth/callback` 间加入 **state 服务端校验**（防 CSRF）；同时复看 **L7**（用户名冲突回退逻辑）。

**前置条件（已就绪 / 待确认）：**

| 项 | 状态 | 说明 |
|---|---|---|
| GitHub OAuth App | ✅ 已创建 | 凭据已提供并完成配对验证（见下） |
| `BOXLI_GITHUB_CLIENT_ID` | ✅ 已验证有效 | `Ov23lidTkmYmm6aJtJKM`；授权页实测可达并显示 App 名 **boxli** |
| `BOXLI_GITHUB_SECRET` | ⚠️ **已外泄，需轮换** | 曾以明文出现在对话中，**上线前必须 Regenerate** |
| `BOXLI_GITHUB_REDIRECT` | ✅ **已确认可用** | App 登记值即 `http://127.0.0.1:3727/api/v1/auth/callback`，与配置一致（复测通过，见下） |
| 本地回调可达性 | ❌ 受限 | 本机无浏览器，真实 OAuth 授权流程需用户配合点击或使用真机 |

**凭据配对验证结论（2026-10-02 实测）**：以 `client_id=Ov23li…` + `client_secret=abf902…`
调用 `POST https://github.com/login/oauth/access_token`（附无效 code）返回
`{"error":"bad_verification_code"}`，说明 **App 身份有效**、仅 code 无效；
将对调后返回 `{"error":"Not Found"}`，**反证配对正确**。

**✅ `redirect_uri` 复测结论（2026-10-02）**：App 登记的 Authorization callback URL 为
`http://127.0.0.1:3727/api/v1/auth/callback`，**与 `BOXLI_GITHUB_REDIRECT` 一致，已可用**。

| 请求 | GitHub 返回 | 结论 |
|---|---|---|
| 换 token 带 `redirect_uri=http://127.0.0.1:3727/api/v1/auth/callback` | `bad_verification_code` | ✅ **通过 redirect_uri 校验**，仅 code 无效 |
| `authorize_url` 经浏览器跳转 | HTTP 200 进入登录页，无 mismatch | ✅ 授权入口可用，`state` 完整保留 |
| 带 `https://boxli.dev/api/v1/auth/callback` | `redirect_uri_mismatch` | 生产地址尚未登记（阶段 7 处理） |

> **⚠️ 测量方法说明**：首次探测曾误判该地址「未登记」，原因是**在 GitHub 保存生效前就探测**
> （同一请求在稍后复测即通过）。判定是否登记的唯一可靠信号是
> `access_token` 接口是否返回 `redirect_uri_mismatch`，且**必须复测**，单次结果不可信。

**区分两个凭据的方法**（易混淆，已写入 README）：

- **Client ID**：20 位、字母数字、以 `Ov23li` 开头（2023 年后格式），可公开
- **Client Secret**：**恰好 40 位纯十六进制** `[0-9a-f]`，仅服务端使用

**✅ L4 已修复（2026-10-02）**：原代码 `handleLogin` 生成并返回 `state`，但
`handleCallback` 只读 `code`、**完全不读 `state`**，OAuth CSRF 防护实际缺失。现方案：

- 新增迁移 `internal/db/migrations/0002_oauth_states.sql`：表 `oauth_states(state PK, redirect, expires_at, created_at)` + `idx_oauth_states_expires`
- 新增 `internal/auth/state.go`：`StateStore.Create()` 生成 32 字节随机 state（10 分钟 TTL）并顺带清理过期行；
  `StateStore.Consume()` 用 **`DELETE ... RETURNING`** 原子完成「校验+删除」，实现**一次性消费**
- `handleCallback` 中 state 校验**最先执行且总是消费**：无论后续成功、被用户拒绝（`error=access_denied`）还是换 token 失败，state 都已作废，不留可重放凭证
- 回调错误处理补齐：用户取消返回 `400 authorization denied`；缺 code 返回 `400 missing code`；换 token/拉用户失败**记录日志详情**（原先静默，排障困难）后返回 502
- 新增 `internal/auth/state_test.go`：覆盖单次消费、伪造/空 state、过期 state 三个用例

**✅ L7 已修复（2026-10-02）**：原 `upsertUser` 在唯一冲突时退化为「按 username 查找并直接登录」。
实测确认该路径可触发（不同 `github_id` 使用已存在的 `username` 必然 `23505`），
后果是**新 GitHub 账号可被登录成同名老账号**（账号接管）。现改为返回
`errIdentityTaken` → **409**，并由调用方给出明确提示。该行为由代码注释与 README 记录。

**回调落点设计（已确认，2026-10-02）**：OAuth 回调成功后**重定向回前端页面**，
而非返回裸 JSON。理由与约束：

- 浏览器 OAuth 流程中，用户看到的站点是前端（同源 `/api/v1/...` 由 Nuxt 反代到 3727），
  回调落在裸后端会使用户停在 JSON 页面上、无法回到站内
- **token 不得放在重定向 URL 的 query 里**（会进入浏览器历史、Referer、访问日志）
- 待定实现方案（前端阶段选型）：
  1. **后端 302 到前端 + httpOnly Cookie 承载 session**（推荐：cookie 不暴露给 JS，天然防 XSS 读取）
  2. 后端 302 到前端并带**一次性、短时效的换取码**，前端再调接口换成 token
- `BOXLI_GITHUB_REDIRECT` 仍指向 `http://127.0.0.1:3727/api/v1/auth/callback`
  （GitHub 只认这一处登记值），由后端处理完 state/token 后再 302 到前端 `/login` 或 `/dashboard`

**产出**：用户可登录并提交镜像

- [x] 后端：真实 GitHub OAuth（`/auth/login` 返回 `authorize_url`；`/auth/callback` 换 token + 签发 JWT）
- [x] 后端：state 服务端校验（L4）
- [x] 后端：`upsertUser` 冲突处理修正（L7）
- [x] 回调地址与 App 登记值一致（复测通过，非阻塞项）
- [x] **后端**：`handleCallback` 由「返回 JSON」改为 **302 重定向回前端**
      —— token 交付方式定为 **httpOnly Cookie**（见下方「回调落点设计」）
- [x] **后端**：CORS 复核并**修正**（`Allow-Origin: *` 与 Cookie 凭证不兼容，改为白名单回显）
- [x] **后端**：新增 CSRF 来源校验（`OriginGuard`）与开放重定向防护（`safeRedirectPath`）
- [x] **后端**：dev 模拟登录改为**显式开关**（`BOXLI_DEV_LOGIN`，默认关闭）
- [x] `/login`：GitHub OAuth 跳转与回调（前端，含错误码中文映射）
- [x] `/submit`：表单，**支持动态添加 ≥ 2 个下载源**
- [x] `/dashboard`：管理自己的镜像（列表 / 删除 / 编辑）
- [x] 前端鉴权状态管理、未登录跳转（`useAuth` + `auth` 中间件）
- [ ] **真实授权联调**：人工点一次授权链接完成端到端登录
      —— ⚠️ **未完成**，被端口占用阻塞，详见下方「阻塞项」

**验证**：真登录提交一个多源镜像，`/explore` 可见，可编辑删除

#### 回调落点与 token 交付设计（已定案并实现）

原实现 `handleCallback` 直接 `writeOK({token})`，会让用户在浏览器里停在一个裸 JSON 页面。
现改为 302 回前端，**token 交付方式选定为 httpOnly Cookie**：

| 方案 | 结论 |
|---|---|
| **httpOnly Cookie** | ✅ **采用**。JS 读不到 token，XSS 无法窃取；浏览器自动携带，前端无需管理凭证 |
| 一次性换取码 | 未采用。换取码仍会短暂出现在地址栏与 Referer 中，且需额外一张表 |
| localStorage | 未采用。XSS 可直接读走会话，安全性最差 |

配套的三项安全措施（缺一不可）：

1. **CSRF 防护**：Cookie 会被浏览器自动携带，因此跨站发起的写请求自带凭证 —— 这正是 CSRF。
   采用 **`SameSite=Lax` + `OriginGuard` 双层**：前者是浏览器层，后者是应用层（不依赖浏览器实现）。
   用 Lax 而非 Strict，是为了让「从 GitHub 跳回本站」的顶层导航能带上 Cookie。
2. **精确同源比对**：`OriginGuard` 用 `url.Parse` 比较 scheme+host，
   **不能**用 `strings.HasPrefix` —— 否则 `http://localhost:3000.evil.com` 会被误判为可信。
3. **开放重定向防护**：登录后回跳地址由前端传入，`safeRedirectPath` 只放行站内相对路径，
   拒绝 `//evil.com`、`https://evil.com`、`javascript:` 与 CRLF 注入。

> **⚠️ CORS 必须同步改（易漏）**：`Access-Control-Allow-Origin: *` 与带凭证的请求**天然不兼容**，
> 浏览器会直接拒绝。已改为「Origin 在白名单内则回显该 Origin + `Allow-Credentials: true`」，
> 并加 `Vary: Origin` 防止缓存串站。这同时推进了 **L5**（生产由 Nginx 同源反代后不再产生跨源请求）。

> **鉴权来源兼容**：`SessionTokenFrom` 按「Cookie 优先，其次 `Authorization: Bearer`」提取。
> 浏览器走 Cookie，curl / CLI / 测试脚本继续走 Bearer，两条路径都实测通过。

#### 前端模块划分（新增）

| 文件 | 职责 |
|---|---|
| `composables/useAuth.ts` | 登录态（`useState` 共享）、`login`/`logout`/`ensure`、`apiWrite`（自动带 Cookie） |
| `middleware/auth.ts` | 路由守卫；**仅在客户端判定** |
| `pages/login.vue` | OAuth 入口、错误码中文映射、已登录态引导 |
| `pages/submit.vue` | 新建镜像 |
| `pages/dashboard.vue` | 我的镜像：列表 / 编辑 / 删除 |
| `components/RepoForm.vue` | 建仓/改仓共用表单（`mode: create \| edit`） |
| `components/SourceEditor.vue` | 下载源增删改，**顺序即优先级** |

> **⚠️ 为什么守卫只在客户端跑**：会话在 httpOnly Cookie 中，**SSR 期间不会自动携带**
> （服务端内部请求不带浏览器 Cookie）。若在 SSR 就判定并跳转，会把**已登录用户也误判为未登录**。
> 因此 `middleware/auth.ts` 首行即 `if (import.meta.server) return`。

> **⚠️ 编辑必须整体回填**：后端 `PUT /repos/{ns}/{repo}` 是 **tags/sources 整体替换**语义，
> 因此 `/dashboard` 进入编辑时会先单独拉一次详情（列表项不含 tags），用全量数据预填表单；
> 若只提交被改动的那一项，其余标签与源会被静默删除。

#### 后端实测记录（2026-10-02 · 本轮）

> 环境：PG 17.11 @ `127.0.0.1:5432`；后端用 `backend/.env`（含真实 OAuth 凭证）。
> 为隔离测试 dev 登录分支，另起实例并显式清空 `BOXLI_GITHUB_CLIENT_ID/SECRET`。

**会话与 Cookie**

| 场景 | 结果 |
|---|---|
| `POST /auth/login` 下发会话 Cookie | ✅ `boxli_session=…; Path=/; Max-Age=2591999; HttpOnly; SameSite=Lax` |
| 带 Cookie 访问 `/auth/me` | ✅ 200 + 正确用户 |
| 不带 Cookie / Bearer | ✅ 401 `missing session cookie or bearer token` |
| 登出 | ✅ 200，`Set-Cookie: boxli_session=; Max-Age=0`，旧会话再访问即 401 |

**CSRF 与来源校验**

| 场景 | 结果 |
|---|---|
| 跨站 Origin 写请求（`https://evil.com`） | ✅ 403 `cross-site request blocked` |
| 同源 Origin 写请求 | ✅ 通过（201 / 200） |
| 无 Origin 的非浏览器客户端（curl/CLI） | ✅ 放行（无 ambient 凭证，不构成 CSRF） |
| `OPTIONS` 预检 | ✅ 204 放行 |

**回调落点（302 链路）**

| 场景 | 结果 |
|---|---|
| 伪造 / 缺失 state | ✅ 302 → `/login?error=invalid_state` |
| 合法 state + 用户拒绝 | ✅ 302 → `/login?error=access_denied`，**state 已被消费** |
| 同一 state 重放 | ✅ 仍 302 `invalid_state`（一次性消费生效） |
| **完整链路** | ✅ 302 → 前端 `/login?error=…` → 页面渲染中文提示「登录会话已失效或已被使用，请重新发起登录。」 |
| token 是否出现在 URL | ✅ **否**（Location 中只有 error 码） |

**写接口（Cookie 会话下的完整 CRUD）**

| 场景 | 结果 |
|---|---|
| 提交镜像：2 标签，其中 1 标签含 **3 个源** | ✅ 201，**3 个源全部落库**且 priority/region 正确 |
| 详情接口回读 | ✅ tags 与 sources 完整（github/gitee/oss，priority 1/2/3，region global/cn） |
| `PUT` 更新（整体替换为 1 标签 2 源） | ✅ 200；旧标签与新源按语义替换（DB 核对通过） |
| `DELETE` | ✅ 200；删除后 GET → 404（级联删除生效） |
| 非 owner 改 / 删他人仓库 | ✅ 403 / 403 |
| 无凭证写请求 | ✅ 401 |
| 空 `tags` | ✅ 400 |

**其它**

| 场景 | 结果 |
|---|---|
| dev 模拟登录**默认关闭** | ✅ 503 且**未创建任何用户**（实测确认） |
| 配好 OAuth 凭证时 `/auth/login` | ✅ 返回 `authorize_url`（走真实 OAuth，不落回 mock） |
| 数据复原 | ✅ 测试用户/仓库/oauth_states 已清理，库回到种子基线 5 users / 5 repos / 10 tags / 17 sources |

**工程质量**

- `gofmt -l` 干净、`go vet` 通过、`go build` 通过
- `go test ./...` **全绿**，共 **8 个用例组**（另有 13 个子例）：
  - `internal/auth`：`TestOriginAllowed`（11 例）、`TestOriginGuard`（10 子例）、`TestSessionTokenFrom`（3 子例）、
    `TestSetAndClearSessionCookie`、`TestStateConsumeIsSingleUse`、`TestStateRejectsUnknownAndEmpty`、`TestStateExpiredRejected`
  - `internal/hub`：`TestSafeRedirectPath`（13 例，覆盖开放重定向与 CRLF 注入）
  - 其中 `TestOriginGuard` 显式断言「被拦截的请求不得到达业务处理器」
- 前端 `eslint .` **0 error / 0 warning**、`nuxt build` 成功

**前端手机端静态自查（新增文件）**

- 新增 10 个表单控件**全部 ≥16px**（`text-base`，防 iOS 聚焦缩放）
- 交互元素统一 `min-h-11`（44px）；`/dashboard` 仓库标题链接已补 44px 触达区
- 无 `100vh`、无固定像素宽度、无 hover-only 显隐
- 三个新页面 viewport 均正确注入

> **⚠️ 本次仍为静态断言**，与阶段 3 一致：**未做真实浏览器/真机测试**。
> `CopyButton` 复测、Lighthouse Mobile、真实触摸与 iOS 行为等仍属**阶段 6**，不得视为已验收。

#### ⚠️ 阻塞项：真实授权联调未完成（2026-10-02）

代码与可自动化验证的部分**已全部通过**，但「人工点一次授权链接」这一步**尚未完成**，
原因是**端口被孤儿进程占用**（会话早期被 kill 的进程，其 socket 仍占着端口，
且属主进程在当前 PID 命名空间中不可见，无法清理）：

| 端口 | 占用情况 | 影响 |
|---|---|---|
| `127.0.0.1:3727` | 孤儿进程，配置陈旧（`BOXLI_FRONTEND_URL=http://localhost:3000`） | **GitHub App 登记的回调只能是此端口**，故真实登录必须由它承接 |
| `localhost:3000` | 陈旧构建（`/login` `/submit` `/dashboard` 均 404） | 即使回调成功，302 落点会是 404 |

磁盘上的构建产物是**最新**的（`.output` 含三个新页面），纯粹是**运行中的进程陈旧**。

**解除方式**（需在能看见这些进程的终端执行）：

```bash
pkill -f 'exe/hub'
pkill -f '.output/server/index.mjs'

# 后端（回调落点指向前端）
cd backend && set -a && . ./.env && set +a
export BOXLI_FRONTEND_URL=http://localhost:3000
go run ./cmd/hub

# 前端
cd frontend && PORT=3000 node .output/server/index.mjs
```

随后访问 `http://localhost:3000/login` 完成一次授权，即可补齐端到端验证。

> 该阻塞**不影响代码正确性**：Cookie 会话、302 回跳、CSRF、state 一次性消费、
> 完整 CRUD 与权限校验均已用 curl 逐项实测通过（见上表）。

> **后端前置实测记录（2026-10-02 · 上一轮，仍有效）**：
>
> - 迁移：`0002_oauth_states.sql` 已应用（`schema_migrations` 两条记录）
> - `go run ./cmd/seed` **连续两次幂等**（5 users / 5 repos / 10 tags / 17 sources）
> - **授权页实测**：`authorize_url` 经 curl 跟随跳转 → HTTP 200，页面包含 **`boxli`** 与
>   **`boxli logo`**，确认 Client ID 有效、App 名称为 boxli；`state` 在跳转中完整保留
> - **换 token 实测**：带 `redirect_uri` 换 token 返回 `bad_verification_code` 而非
>   `redirect_uri_mismatch`，即**回调地址、凭据、代码三者均正确**，仅剩「用真实 code 完成授权」
> - **回归**：`/health`、`/search`、`/repos`、`/repos/alice/myapp` 均 200，无影响
>
> **测量方法说明（重要）**：判定 `redirect_uri` 是否登记，唯一可靠信号是 `access_token`
> 接口是否返回 `redirect_uri_mismatch`，且**必须复测** —— 早期一次探测曾在 GitHub 保存生效前
> 进行而误判「未登记」。

---

### 阶段 5：文档站（1 天）✅ 已完成（2026-10-02）

> **📌 标注**：本阶段延续第四节「手机优先」强约束，并为文档站补充**标题锚点深链**能力。

**产出**：`/docs` 使用文档

- [x] 文档路由与布局（桌面 sticky 侧边导航 + 手机折叠菜单 + 本页小节 `Anchor`）
- [x] 快速开始、安装、pull/run 命令、镜像格式说明、FAQ
- [x] 手机端文档阅读体验

**验证**：`/docs` 全部页面手机模拟下可读、可跳转

> **实际产出与验证记录：**
>
> **内容基准（重要）**：本阶段所有命令与 flag **照实抄录自本机真实 `boxli` 二进制**
> （`boxli version 0.0.0-dev`）的 `--help` 输出，**未按规格臆造**。实测中发现规格与真实 CLI
> 存在差异，已按实际实现为准并在文档中如实标注：
>
> | 项 | 规格描述 | 真实 CLI |
> |---|---|---|
> | 命令数量 | 仅提到 `pull` / `run` | 实际 **28 个**子命令（`build`/`commit`/`compose`/`volume`/`network`/`doctor` …） |
> | 镜像引用 | `alice/myapp:v1`（含命名空间） | **`NAME:VERSION`**（无命名空间段） |
> | `pull` 语义 | 仅从 Hub 拉取 | **双语义**：传 `.boxli` 文件则导入，传 `NAME:VERSION` 才走 Hub |
>
> **新增前端文件**：
> - `app/utils/docs.ts`：文档内容与导航结构（7 个页面 + 导航 + 上下页 + `renderDocMarkdown`）
> - `app/components/DocsLayout.vue`：文档布局（侧边导航 / 手机折叠目录 / 本页锚点 / 翻页）
> - `app/pages/docs/`：`index.vue`、`quickstart.vue`、`install.vue`、`pull-run.vue`、
>   `format.vue`、`submit.vue`、`faq.vue`
> - 修改 `app/composables/useMarkdown.ts`：新增 `slugify` + `heading_open` 规则，
>   为标题注入 `id` 锚点（**通过 `anchors` 选项开关，README 渲染保持原行为**）
>
> **内容覆盖**（对应验收项）：
> - 快速开始（`/docs/quickstart`）、安装与自检（`/docs/install`）
> - 拉取与运行（`/docs/pull-run`，含 `pull`/`run` 全部 flag 表）
> - 镜像格式与下载源（`/docs/format`，含 `.boxli` 与 Docker 差异、10 种源类型、Boxfile 构建）
> - 提交镜像到 Hub（`/docs/submit`）、常见问题（`/docs/faq`）
>
> **⚠️ 实测发现的规格外事实（已如实写入文档，未做掩盖）**：
> 本机 `boxli` CLI 对接的是**另一套 Hub**（`boxli hub serve`：用户名/密码登录 + blob 存储），
> 与本站后端（GitHub OAuth + 只存元数据）**并非同一实现**。实测证据：
>
> | 命令 | 结果 |
> |---|---|
> | `boxli search --hub http://127.0.0.1:3727` | `未登录 …请先 boxli login` |
> | `boxli login --hub http://127.0.0.1:3727` | `hub 404 Not Found` |
>
> 因此文档中**未提供**「用 CLI 直连本站 Hub 拉取镜像」的可执行示例（避免给出无法工作的指令），
> 改为引导用户经网页端获取下载地址后用 `boxli pull <file.boxli>` 导入；
> 并在 `/docs/faq` 首节以「CLI 与 Hub 的对接现状」明确标注可用/不可用范围。
> **CLI 与本站 Hub 的对接为后续工作项。**
>
> **验证结果**（生产构建 `node .output/server/index.mjs`，:3077）：
> - **路由**：`/docs`、`/docs/quickstart`、`/docs/install`、`/docs/pull-run`、`/docs/format`、
>   `/docs/submit`、`/docs/faq` 全部 **200**；标题与 `<title>` 均正确
>   （如 `/docs/faq` → H1「常见问题」、title「常见问题 · Boxli 文档」）
> - **导航与锚点**：7 项侧边导航齐全；**51 条文档站内部链接全部可达**；
>   跨页深链（如 `/docs/faq#cli-与-hub-的对接现状`）经脚本逐条校验，
>   **目标页面与目标锚点均存在**（`missing = none`）
> - **标题锚点**：Markdown `##` 标题已注入 `id`（如 `cli-与-hub-的对接现状`），
>   同页重复标题自动加 `-1` 后缀去重；并加 `scroll-margin-top: 5rem` 避免被 sticky 头部遮挡
> - **回归**：镜像详情页 README 渲染**未受影响**（`anchors` 默认关闭，`<h1>` 仍无 `id`）；
>   `/`、`/explore`、`/search?q=postgres`、`/about`、`/login`、`/submit`、`/dashboard` 全部 **200**
> - **手机端清单（静态断言，对应第四节「无实机测试替代方案」）**：
>   7 个文档页 viewport 正确注入；`100vh` 出现 **0** 次、`100dvh` 存在；
>   `overflow-x-hidden` 均存在；页脚 safe-area 内边距保留；
>   交互元素（按钮）**全部 ≥44px**（`min-h-11` / `h-11 w-11`）；
>   输入框统一 `text-base`（16px，防 iOS 缩放）；无 `group-hover` 等 hover-only 显隐（坑 2 已规避）
> - **窄屏排版**：宽表格与代码块用 `overflow-x-auto` 横向滚动，不撑破窄屏；
>   本页小节锚点在手机端改为横向滚动 chip 条
> - **XSS 防护实测**：文档正文统一走 `markdown-it(html:false)` + DOMPurify 同一条链路；
>   构造 `<script>`、`<img onerror>`、`<iframe>` 载荷实测 —— `<script>`/`<iframe>` **被转义或过滤**，
>   未产生可执行节点（该链路与阶段 3 README 渲染同源，已有验证基础）
> - **工程质量**：`eslint .` **0 error / 0 warning**；`nuxt build` 成功
>
> **⚠️ 本阶段仍为静态断言**（与阶段 3/4 一致）：**未做真实浏览器/真机测试**。
> Lighthouse Mobile、真实触摸与 iOS 行为等仍属**阶段 6**，不得视为已验收。

---

### 阶段 6：手机端全面验收（1 天）

**产出**：通过第四节全部强制清单

- [ ] 按第四节「无实机测试替代方案」跑完整清单
- [ ] Lighthouse Mobile ≥ 90
- [ ] 无 Hydration 报错
- [ ] 修复阶段 3–5 遗留的响应式问题

**验证**：第四节手机端 checklist 全绿

---

### 阶段 7：部署上线（1 天）

> **📌 标注（上线前必做，对应阶段 2 遗留项）**：
> - [x] ~~还原 **L1**：WSL 内 PG `listen_addresses` 改回 `localhost`~~ → **已随环境迁移消除**
> - [x] ~~删除 **L2**：`pg_hba.conf` 中 `host all all 0.0.0.0/0` 行~~ → **已随环境迁移消除**
> - [x] ~~删除 **L8**：Windows 防火墙规则 `WSL-PG5432`~~ → **已随环境迁移消除**
> - [ ] 收紧 **L5**：移除后端 `Access-Control-Allow-Origin: *`（改由 Nginx 同源反代）
> - [ ] 按需处理 **L6**：限流 / 请求日志加固
> - [ ] **轮换 `BOXLI_GITHUB_SECRET`**：该 secret 曾以明文外泄，上线前必须在 GitHub 重新生成
> - [ ] 生产 `BOXLI_GITHUB_REDIRECT` 改为 `https://boxli.dev/api/v1/auth/callback` 并同步更新 GitHub App 登记

**产出**：公网可访问

- [ ] Nginx 配置：前端托管 + `/api/` 反代 + `/docs/`
- [ ] Let's Encrypt 证书配置 HTTPS
- [ ] systemd：`boxli-hub.service` + 前端 Node/静态服务
- [ ] ufw 只开 80/443，确认 3727/5432 不可达
- [ ] 每日数据库备份脚本 + 定时任务

**验证**（第十三条验收标准）：
- [ ] `https://boxli.dev/api/v1/health` 返回 200
- [ ] `curl http://<服务器IP>:3727` 超时
- [ ] `ufw status` 无 3727 / 5432
- [ ] 备份脚本执行成功

---

### 阶段总览

| 阶段 | 内容 | 预计 | 依赖 |
|---|---|---|---|
| 0 ✅ | 环境与脚手架 | 0.5d | — |
| 1 ✅ | 数据库与数据层 | 1d | 0 |
| 2 ✅ | 后端 API | 2d | 1 |
| 3 ✅ | 前端核心页面 | 3d | 2 |
| 4 🔄 | 认证与提交（代码已完成，真实授权联调待补） | 2d | 2,3 |
| 5 ✅ | 文档站 | 1d | 3 |
| 6 | 手机端全面验收 | 1d | 3,4,5 |
| 7 | 部署上线 | 1d | 6 |

**合计约 11.5 个工作日。** 关键路径：0 → 1 → 2 → 3 → 4 → 6 → 7。

> 进度：阶段 0 已完成（2026-10-02）；阶段 1 已完成（2026-10-02）；阶段 2 已完成（2026-10-02）；
> 阶段 3 已完成（2026-10-02）；**阶段 5 已完成（2026-10-02）**；
> **阶段 4 代码已全部完成**：后端 OAuth + 302 回跳 + httpOnly Cookie 会话 + CSRF 来源校验 +
> 开放重定向防护 + dev 登录开关；前端 `/login`、`/submit`、`/dashboard` + 鉴权状态管理全部落地。
> **唯一未完成项是「人工点一次真实授权」**（被端口占用阻塞，见阶段 4 末尾），
> 其余（Cookie 会话、302 链路、CSRF、state 一次性消费、完整 CRUD 与权限）均已 curl 实测通过。
>
> **阶段 5 已完成**：`/docs` 文档站 7 个页面（快速开始 / 安装自检 / 拉取运行 /
> 镜像格式与下载源 / 提交镜像 / FAQ），桌面 sticky 侧边导航 + 手机折叠目录 + 本页锚点 +
> 上下页翻页；Markdown 标题锚点深链可用（51 条内部链接全部可达）。
> 内容照实抄录自真实 `boxli` 二进制（28 个子命令），并如实标注了
> **CLI 与本站 Hub 尚未对接**这一规格外事实。
> 与阶段 3/4 相同，手机端为**静态断言**，真实浏览器验收仍属阶段 6。
>
> **待处理遗留项**：见阶段 2 记录末尾的「遗留项 / 风险标注」表。
> **开发环境已迁移到原生 Linux（Ubuntu 22.04）**，故 **L1 / L2 / L8 已随迁移消除**；
> **L3/L4/L7 已于阶段 4 修复**；**L5 已部分推进**（跨源 CORS 改为白名单回显 + 凭证支持，
> 生产由 Nginx 同源反代后彻底移除）；剩余 **L6 为安全加固，阶段 7 处理**。
>
> **⚠️ 阶段 4 剩余阻塞项**：
> 1. ~~`BOXLI_GITHUB_REDIRECT` 需在 GitHub 侧对齐~~ —— **✅ 已解除**：App 登记值即
>    `http://127.0.0.1:3727/api/v1/auth/callback`，复测通过（早期误判系保存生效前探测所致）
> 2. **`BOXLI_GITHUB_SECRET` 已外泄** —— 上线前必须在 GitHub Regenerate（阶段 7）
> 3. **真实授权联调未完成** —— 3727 / 3000 被孤儿进程占用且配置陈旧，
>    需重启服务后人工点一次授权（详见阶段 4 末尾「阻塞项」）
>
> **阶段 3 遗留到阶段 6 的验证项**（**手机测试被跳过，不得视为已验收**）：
> - `CopyButton` 宽度修复（`min-w-11`）**未经复测**
> - **Lighthouse Mobile ≥ 90** 从未测量
> - 真实触摸延迟、iOS 滚动惯性、地址栏收缩下的 `100dvh` 动态表现
> - iOS 输入框聚焦实测（当前仅静态确认字号 ≥16px）
> - **阶段 4 新增页面同样只有静态断言**（`/login`、`/submit`、`/dashboard` 未做真实浏览器测试）
> - **阶段 5 新增的 7 个文档页同样只有静态断言**（`/docs` 及其子页面未做真实浏览器测试，
>   文档阅读体验的触摸滚动、代码块横向滚动手感未经真机确认）
>
> 测试用 `chrome-headless-shell` 与 `puppeteer-core` 依赖已按用户要求移除，
> 阶段 6 需重新准备浏览器环境（或使用真机 / BrowserStack 等云真机）完成上述验收。

---

## 附：开发者技术文档

本文件聚焦**计划与进度**；具体实现细节见 [`DEVELOPMENT.md`](./DEVELOPMENT.md)：

- 架构总览与目录/模块职责
- 数据模型与迁移机制
- 后端中间件链、统一响应、读写事务
- **认证与会话实现**（Cookie 交付、OAuth 全链路、CSRF、开放重定向、CORS 兼容性）
- 前端登录态管理、客户端守卫、表单单向数据流
- 完整 API 参考与错误码
- 本地开发与 curl 调试方法
- 测试清单与**安全设计汇总**
- 已知限制与设计取舍备忘

---

**有不清楚的随时问。**

