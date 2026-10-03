# Boxli Hub 部署文档

**目标**：在一台干净的 Linux 服务器上，用**一个二进制**把站点跑起来。

- 想了解代码 → 见 [`DEVELOPMENT.md`](./DEVELOPMENT.md)
- 想本地开发 → 见 [`README.md`](./README.md)

**部署形态**：`boxli-hub`（二进制，内含前端）+ `hub.toml`（配置）+ PostgreSQL。
运行时依赖 **Node.js**（SSR 需要）。

---

## 目录

- [一、准备](#一准备)
- [二、拿到二进制](#二拿到二进制)
- [三、数据库](#三数据库)
- [四、配置](#四配置)
- [五、启动](#五启动)
- [六、首次引导](#六首次引导)
- [七、Nginx 与 HTTPS](#七nginx-与-https)
- [八、systemd 守护](#八systemd-守护)
- [九、上线检查](#九上线检查)
- [十、备份](#十备份)
- [十一、排障](#十一排障)

---

## 一、准备

| 依赖 | 版本 | 说明 |
|---|---|---|
| Linux x86_64 | — | 二进制为静态链接，任意发行版可用 |
| **Node.js** | ≥ 20 | **必需**。SSR 由 Node 执行，缺失会导致启动失败 |
| PostgreSQL | ≥ 14 | 数据库 |
| Nginx | — | 反向代理 + HTTPS |

```bash
# 以 Debian/Ubuntu 为例
sudo apt install -y nodejs postgresql nginx
node --version      # 确认 ≥ 20
```

> **为什么需要 Node**：Go 无法执行 JavaScript。前端产物已内嵌在二进制里，
> 但渲染仍需 Node 进程 —— 主进程会自己在内部拉起并管理它，你不需要单独启动前端服务。

**数据库只绑回环**（见[第三节](#三数据库)），公网只需开 80/443。

---

## 二、拿到二进制

两种方式任选：

```bash
# 方式一：从 Release 下载
wget https://github.com/xgp2012/liboxhub/releases/latest/download/boxli-hub \
     https://github.com/xgp2012/liboxhub/releases/latest/download/boxli-hub.sha256
sha256sum -c boxli-hub.sha256

# 方式二：自己构建（需 Go 1.27+ 与 Node 20+）
git clone https://github.com/xgp2012/liboxhub.git && cd liboxhub
make build          # 产物：backend/boxli-hub
```

安装：

```bash
sudo install -m 0755 boxli-hub /usr/local/bin/boxli-hub
boxli-hub --version     # 确认可执行
```

> 二进制**不需要 Go 环境**，但**需要 Node**（见第一节）。

---

## 三、数据库

```bash
# 生成一个强密码
openssl rand -base64 24
```

```sql
-- 用 postgres 用户执行
CREATE USER boxli WITH PASSWORD '<上面生成的密码>';
CREATE DATABASE boxli_hub OWNER boxli;
```

**建表不需要手动做。** 启动时会自动执行内嵌的迁移（幂等，可重复启动）。

> 迁移只创建**表与索引**，不会 `CREATE DATABASE` / `CREATE ROLE` ——
> 这两件事需要更高权限，由你按上面的命令完成。

---

## 四、配置

```bash
sudo mkdir -p /etc/boxli-hub
sudo cp hub.toml.example /etc/boxli-hub/hub.toml    # 从 Release 或仓库获取
sudo chmod 600 /etc/boxli-hub/hub.toml
sudo chown root:root /etc/boxli-hub/hub.toml
```

编辑 `/etc/boxli-hub/hub.toml`，**至少改这四项**：

```toml
[server]
addr = "127.0.0.1:3727"          # 必须回环，对外交给 Nginx

[db]
url = "postgres://boxli:<密码>@127.0.0.1:5432/boxli_hub?sslmode=disable"

[session]
# openssl rand -hex 32
jwt_secret = "<64 位十六进制>"
cookie_secure = true             # 生产 HTTPS 必须 true

[site]
frontend_url = "https://boxli.dev"   # 对外访问地址，必须与实际一致
```

其余字段按需：

| 字段 | 何时需要 |
|---|---|
| `[github] client_id` / `secret` / `redirect` | 想启用 GitHub 登录。**不用就整段留空**，靠本地密码登录 |
| `[frontend] node_path` | node 不在 `PATH` 里时指定绝对路径 |
| `[site] extra_origins` | 有多个域名时 |
| `[dev] enabled` | **生产必须 `false`**（设为 `true` 程序拒绝启动） |

### 上线前必改（若启用 GitHub 登录）

- ⚠️ **重新生成 GitHub Client Secret** —— 仓库历史里曾有过明文
- ⚠️ 把 GitHub App 的 **Authorization callback URL** 改为
  `https://boxli.dev/api/v1/auth/callback`（必须与 `[github] redirect` 完全一致）
- ⚠️ `[github] redirect` 也要同步改成上面的地址

### 配置写错会怎样

程序**启动即报错退出**，不会带病运行：

| 检查 | 拒绝原因 |
|---|---|
| 无法识别的字段 | 防拼错后静默不生效 |
| `addr` 非回环 | 会绕开 Nginx 的 HTTPS 与 CSRF 保护 |
| `jwt_secret` < 16 字符 | 弱密钥 |
| 只配 `client_id` 或只配 `secret` | 会静默降级为未配置 |
| 配了凭据却没配 `redirect` | 回调必然失败 |
| `frontend_url` 带路径 | `OriginGuard` 按 `scheme://host:port` 精确比对，带路径永远匹配不上 |
| `cookie_secure=true` 却配 `http://` | 浏览器丢弃 Cookie，表现为「登录后仍显示未登录」 |
| 公网 `https://` 却未开 `cookie_secure` | Cookie 会在明文信道传输 |

启动日志会打印配置摘要，**密码与密钥已脱敏**：

```
loaded config{path=/etc/boxli-hub/hub.toml addr=127.0.0.1:3727
  db=postgres://boxli:***@... frontend=https://boxli.dev cookie_secure=true ...}
```

---

## 五、启动

```bash
/srv/boxli          # 或任意目录
sudo -u boxli /usr/local/bin/boxli-hub --config /etc/boxli-hub/hub.toml
```

成功的日志：

```
boxli-hub v0.1.0 (44aef5b3)
loaded config{...}
database migrated
首次部署引导已启用：系统中还没有任何用户      ← 首次才有，见第六节
SSR 子进程已启动 (pid=12345, port=45678)
SSR 已就绪 (port=45678)
boxli hub listening on 127.0.0.1:3727
```

> **SSR 端口是随机的**（日志里的 `port=`），由内核分配、只绑 `127.0.0.1`，
> 外部不可达。无需在配置或防火墙里固定它。

停止：`Ctrl-C` 或 `SIGTERM`。程序会先停 SSR 子进程、清理临时文件再退出。

### 参数

| 参数 | 用途 |
|---|---|
| `--config <路径>` | 指定 TOML 配置（默认 `./hub.toml`） |
| `--no-ssr` | **仅排查用**：不启动 SSR，`/_nuxt/*` 等静态资源仍可访问，但页面返回 503 |
| `--version` | 打印版本与 commit |

> `--no-ssr` **不是**「降级运行模式」—— 页面会返回 503。
> 它的用途是在「怀疑 SSR 有问题」时确认 Go 侧是否正常。

---

## 六、首次引导

**数据库里没有任何用户时**，启动日志会打印一次性安装令牌：

```
────────────────────────────────────────────────────────────
 首次部署引导已启用：系统中还没有任何用户
 请在浏览器打开： https://boxli.dev/setup?token=<64 位十六进制>
 该令牌仅在本次启动且系统无用户时有效，初始化后立即作废。
────────────────────────────────────────────────────────────
```

浏览器打开该地址，填写**用户名 + 密码**即可完成初始化（建表已在启动时完成）。

### 为什么要有令牌

引导页对公网开放 —— 若不设防，**任何人都能在你之前抢注管理员**。因此：

| 防护 | 说明 |
|---|---|
| 令牌只出现在**服务端日志** | 公网访问者看不到日志，无法抢先注册 |
| 只在无用户时可用 | 计数与插入在同一事务内，并发也创建不出第二个 |
| 恒定时间比较 | 防时序侧信道推断令牌 |

初始化后接口**永久关闭**，令牌立即作废。

### 规则

| 项 | 规则 |
|---|---|
| 用户名 | 3–32 字符，仅英文字母、数字、`_`、`-` |
| 密码 | ≥8 字符，≤72 字节，不能是常见弱口令 |

> 密码上限 72 **字节**是因为 bcrypt 只使用前 72 字节，超出会被静默截断。

### 管理员权限

目前 `is_admin` **只是一个标记**，**没有后台管理界面** ——
管理员与普通用户一样只能管理**自己**的仓库。封禁用户、删除他人仓库等功能尚未实现。

### 不用 GitHub 也能跑

把 `[github]` 整段留空即可，引导创建的管理员用用户名密码登录。

---

## 七、Nginx 与 HTTPS

`/etc/nginx/sites-available/boxli`：

```nginx
# HTTP → HTTPS
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

    # 证书由 certbot 自动填充
    # ssl_certificate     /etc/letsencrypt/live/boxli.dev/fullchain.pem;
    # ssl_certificate_key /etc/letsencrypt/live/boxli.dev/privkey.pem;

    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    client_max_body_size 2m;

    # ===== 全部请求转给 boxli-hub =====
    # /api/*    → Go 原生处理
    # /_nuxt/*  → Go 从内嵌资源返回
    # 其他       → Go 内部反代给 node 子进程做 SSR
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

    # 文件名含内容 hash，内容变化必然换名，可长期缓存
    location /_nuxt/ {
        proxy_pass http://127.0.0.1:3727;
        proxy_set_header Host $host;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

**一条 `location /` 就够了** —— 前后端在同一个进程里，不需要分别代理。

> ⚠️ **不要在 Nginx 层加 `Access-Control-*`**。CSRF 防线依赖 `Origin` 原样传递，
> 且 Cookie 请求不允许 `Access-Control-Allow-Origin: *`。同源反代根本不产生跨源请求。

启用与证书：

```bash
sudo ln -s /etc/nginx/sites-available/boxli /etc/nginx/sites-enabled/
sudo rm -f /etc/nginx/sites-enabled/default      # 避免抢占默认站点
sudo nginx -t && sudo systemctl reload nginx

sudo certbot --nginx -d boxli.dev -d www.boxli.dev
sudo systemctl list-timers | grep certbot        # 确认自动续期
```

防火墙：

```bash
sudo ufw allow 80/tcp && sudo ufw allow 443/tcp && sudo ufw enable
sudo ufw status verbose      # 不应出现 3727 或 5432
```

---

## 八、systemd 守护

`/etc/systemd/system/boxli-hub.service`：

```ini
[Unit]
Description=Boxli Hub
Documentation=https://github.com/xgp2012/liboxhub
After=network.target postgresql.service
Wants=postgresql.service

[Service]
Type=simple
User=boxli
Group=boxli
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

```bash
sudo useradd --system --home /var/lib/boxli --shell /usr/sbin/nologin boxli
sudo mkdir -p /var/lib/boxli && sudo chown boxli:boxli /var/lib/boxli

sudo systemctl daemon-reload
sudo systemctl enable --now boxli-hub
sudo systemctl status boxli-hub
sudo journalctl -u boxli-hub -n 50 --no-pager      # 首次引导令牌在这里
```

> **只有这一个 unit。** 前端由主进程托管，不需要单独的服务。
>
> 程序退出时会向 node 子进程的**整个进程组**发 SIGTERM（5 秒后 SIGKILL），
> 并删除临时目录。另设了 `Pdeathsig` —— 即使主进程被 `SIGKILL` 强杀，
> 内核也会终止 node，**不会留下孤儿进程**。

---

## 九、上线检查

```bash
# 1) 端口未外泄
curl -m 5 http://<公网IP>:3727    # 应超时
curl -m 5 http://<公网IP>:5432    # 应超时

# 2) API 与数据库连通
curl -s https://boxli.dev/api/v1/health
# {"code":0,"message":"ok","data":{"status":"healthy"}}
curl -s "https://boxli.dev/api/v1/repos?limit=1"   # 返回 items 才算真通

# 3) SSR 与 SEO —— 标题必须在**服务端返回的 HTML** 里
curl -s https://boxli.dev/ | grep -o '<title>[^<]*</title>'
curl -s https://boxli.dev/ | grep -o '<meta name="description"[^>]*>' | head -1

# 4) 页面可达
for p in / /explore /docs /search; do
  curl -s -o /dev/null -w "$p → %{http_code}\n" "https://boxli.dev$p"
done
```

> ⚠️ **`<title>` 必须在 curl 的原始 HTML 里能看到**。用浏览器 F12 看到的标题是
> JS 执行后的结果，**不能**作为 SSR 生效的证据。

> `/health` **恒返回 healthy**，不探测数据库 —— 数据库挂了它依然 200。
> 真正有效的探活是第 2 条。

**若启用了 GitHub 登录**，还需人工走一遍：点登录 → 跳 GitHub → 授权 →
回到 `/dashboard` 且显示已登录 → 提交一个测试仓库（不被 403）。

---

## 十、备份

```bash
#!/usr/bin/env bash
# /usr/local/bin/boxli-backup.sh
set -euo pipefail
DIR=/var/backups/boxli
mkdir -p "$DIR"
STAMP=$(date +%F_%H%M)
pg_dump "postgres://boxli:<密码>@127.0.0.1:5432/boxli_hub" \
  | gzip > "$DIR/boxli_$STAMP.sql.gz"
# 只保留 14 天
find "$DIR" -name 'boxli_*.sql.gz' -mtime +14 -delete
```

```bash
sudo chmod +x /usr/local/bin/boxli-backup.sh

# 每日 03:30
echo '30 3 * * * root /usr/local/bin/boxli-backup.sh' | sudo tee /etc/cron.d/boxli-backup
```

恢复：

```bash
sudo systemctl stop boxli-hub                       # 避免写入竞争
gunzip -c /var/backups/boxli/boxli_<时间>.sql.gz | psql "postgres://..."
sudo systemctl start boxli-hub
```

> 建议定期**实际演练一次恢复** —— 没验证过的备份不算备份。

---

## 十一、排障

### 启动失败

| 报错 | 原因 |
|---|---|
| `未找到 node 可执行文件` | 没装 Node，或 `[frontend] node_path` 写错 |
| `pattern all:dist: no matching files found` | 用 `go build` 直接构建了，没先生成前端产物 → 改用 `make build` |
| `addr 必须是回环地址` | `[server] addr` 填了 `0.0.0.0`，应交给 Nginx |
| `cookie_secure 与 frontend_url 协议不匹配` | 见[第四节](#四配置)的校验表 |

### 页面能开但不对

| 症状 | 原因 |
|---|---|
| 页面 503 | SSR 没起来。查日志里的 SSR 报错；`--no-ssr` 时页面必然是 503 |
| 页面 200 但样式/JS 全丢 | `go:embed` 前缀问题（见 `DEVELOPMENT.md` 第七节） |
| 登录成功但仍显示未登录 | `cookie_secure` 与协议不匹配 |
| 写请求 403 | `Origin` 不在 `[site] frontend_url` 白名单里 |
| OAuth 报 `redirect_uri_mismatch` | `[github] redirect` 与 GitHub App 登记的不一致 |

### 端口归属排查

```bash
ss -ltnpe | grep :3727
```

输出里的 `uid:` 与 `cgroup:` 能告诉你进程属于谁（普通用户看不到别人的 `pid=`）。

> ⚠️ 排查时不要用 `pkill -f <模式>` —— 它可能匹配到你自己所在的 shell。
> 用 `ss -ltnp` 拿到 PID 再精确 `kill`。

### 日志

```bash
sudo journalctl -u boxli-hub -f              # 实时
sudo journalctl -u boxli-hub --since "1 hour ago"
```

---

## 附：与旧版部署的差异

早期版本是「两个进程 + 两个 systemd unit」（前端单独跑 `node .output/server/index.mjs`）。
现已合并为单二进制。**若从旧版升级**：

```bash
sudo systemctl disable --now boxli-frontend
sudo rm /etc/systemd/system/boxli-frontend.service
sudo systemctl daemon-reload
```

Nginx 侧原本指向 `:3000` 的 location 也可以删掉，只保留一条 `location /` → `:3727`。
