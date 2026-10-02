// 文档站内容与导航结构（阶段 5）
//
// 设计说明：
// - 正文以 Markdown 字符串存放，复用 useMarkdown（markdown-it html:false + DOMPurify），
//   与镜像 README 渲染走同一条安全链路，不引入新的渲染依赖。
// - 内容为**纯静态常量**，不含 Date/随机数，SSR 与客户端输出一致（避免 hydration 不一致）。
// - 命令与 flag 全部照实抄录自本机 `boxli --help` / `<cmd> --help`（boxli version 0.0.0-dev），
//   不做臆造；CLI 与本站 Hub 的对接现状见 CLI_HUB_STATUS 一节。

import { renderMarkdown } from '~/composables/useMarkdown'

/**
 * 渲染文档 Markdown。
 *
 * 复用镜像 README 的同一条安全链路：markdown-it（html:false，禁用原始 HTML）
 * + DOMPurify 清洗，外链自动补 rel="noopener noreferrer nofollow"。
 * 文档内容虽为内置常量，仍统一走该链路，避免日后引入用户内容时出现安全缺口。
 */
export function renderDocMarkdown(body: string): string {
  // anchors: true —— 文档站需要 `#小节标题` 深链（README 渲染不需要，保持原行为）
  return renderMarkdown(body, { anchors: true })
}

export interface DocSection {
  /** 锚点 id，同时用于侧边导航与标题跳转 */
  id: string
  title: string
  /** Markdown 正文 */
  body: string
}

export interface DocPage {
  /** 路由路径（相对 /docs） */
  path: string
  /** 浏览器标题（含站点后缀） */
  title: string
  /** 侧边导航与 H1 用的短标题 */
  navTitle: string
  /** 页面 H1（可与 navTitle 不同，如首页用「Boxli 文档」） */
  heading: string
  description: string
  sections: DocSection[]
}

/**
 * CLI 与本站 Hub 的对接现状。
 *
 * 实测事实（2026-10-02，boxli v0.0.0-dev）：
 * - 本机 `boxli` 对接的是**另一套** Hub（`boxli hub serve`：用户名/密码登录 + blob 存储），
 *   与本站后端（GitHub OAuth + 只存元数据）**并非同一实现**。
 * - `boxli search --hub http://127.0.0.1:3727` → `未登录 …请先 boxli login`
 * - `boxli login --hub http://127.0.0.1:3727` → `hub 404 Not Found`
 * 因此文档中**不提供**「用 boxli CLI 直接拉取本站镜像」的可执行示例，避免给出无法工作的指令。
 */
export const CLI_HUB_STATUS = {
  cliVersion: '0.0.0-dev',
  hubApi: 'http://127.0.0.1:3727/api/v1',
  works: [
    '镜像构建与本地管理：build / pull（本地 .boxli 文件）/ images / tag / save / import',
    '容器生命周期：run / ps / exec / stop / rm / stats',
    '编排与资源：compose / volume / network / dev / lint / doctor',
  ],
  notYet: [
    'boxli search / login / push / pull NAME:VERSION 直连本站 Hub',
  ],
}

const QUICKSTART = `Boxli 是一个用 Go 编写的轻量级容器引擎，自研 \`.boxli\` 镜像格式，
**不兼容 Docker / OCI**。本站（Boxli Hub）只做一件事：**收录社区镜像的元数据**，
不存储镜像文件本体。

## 三步开始

\`\`\`bash
# 1. 检查环境（内核 / namespace / cgroup / systemd / 存储）
boxli doctor

# 2. 从本地 .boxli 文件导入镜像
boxli pull ./myapp.boxli

# 3. 运行容器
boxli run myapp:v1
\`\`\`

> \`boxli pull\` 有**两种**用法：传本地 \`.boxli\` 文件路径时导入文件；
> 传 \`NAME:VERSION\` 时从 Hub 拉取。当前与本站 Hub 的对接状态见
> [「CLI 与 Hub 的对接现状」](/docs/faq#cli-与-hub-的对接现状)。

## 命令总览

| 分类 | 命令 |
|---|---|
| 镜像 | \`build\` \`pull\` \`push\` \`images\` \`tag\` \`save\` \`export\` \`import\` \`load\` \`commit\` |
| 容器 | \`run\` \`ps\` \`exec\` \`stop\` \`rm\` \`stats\` |
| 编排 | \`compose\` \`dev\` \`lint\` \`scaffold\` |
| 资源 | \`volume\` \`network\` \`resource\` |
| 系统 | \`doctor\` \`boot\` \`shutdown\` \`hub\` |
| Hub | \`login\` \`search\` |

## 本站能做什么

- **浏览**：所有收录镜像见 [浏览页](/explore)，支持按名称、描述、命名空间搜索
- **查看**：镜像详情页展示 README、标签、支持架构与**全部下载源**
- **提交**：登录后可在 [提交页](/submit) 收录你自己的镜像（元数据 + 下载地址）
- **管理**：在 [用户中心](/dashboard) 编辑或删除自己提交的镜像`

const INSTALL = `Boxli 是单个 Go 二进制，没有运行时依赖。当前版本为开发版 \`0.0.0-dev\`。

## 从源码构建

\`\`\`bash
git clone https://github.com/LiStudioorg/boxli
cd boxli
go build -o boxli ./cmd/boxli
\`\`\`

安装到 PATH：

\`\`\`bash
install -m 0755 boxli /usr/local/bin/boxli
boxli version
\`\`\`

## 安装后自检

\`\`\`bash
boxli doctor
\`\`\`

\`doctor\` 会依次检查**内核 / namespace / cgroup / systemd / 存储**是否满足运行要求：

| Flag | 说明 |
|---|---|
| \`--skip\` | 跳过指定检查项（逗号分隔的检查 ID） |
| \`--test-run\` | 额外执行一次**真实容器冒烟测试** |
| \`--version\` | 指定用于输出的版本号（默认由 CLI 注入） |

> 容器能跑起来的前提是内核支持 namespace 与 cgroup。\`doctor\` 报错时请先解决对应项，
> 否则 \`boxli run\` 会在创建阶段失败。

## 数据目录

Boxli 的所有状态（镜像、容器、卷、网络）都在一个数据目录下，默认 \`$BOXLI_HOME\` 或 \`~/.boxli\`：

\`\`\`bash
# 覆盖数据目录
boxli --data-dir /data/boxli images

# 或用环境变量
export BOXLI_HOME=/data/boxli
boxli images
\`\`\`

\`images\` 等子命令也各自支持 \`--data-dir\`。

## 常用环境变量

| 变量 | 作用 |
|---|---|
| \`BOXLI_HOME\` | 数据目录，替代 \`~/.boxli\` |
| \`BOXLI_HUB\` | Hub 地址，供 \`login\` / \`search\` / \`push\` / \`pull\` 使用 |`

const PULL_RUN = `本节对应 \`boxli pull\` 与 \`boxli run\` 两条最常用命令。
下面所有 flag 均照实抄录自 \`boxli pull --help\` / \`boxli run --help\`。

## 导入镜像

\`\`\`
Usage:
  boxli pull <file.boxli|NAME:VERSION> [flags]
\`\`\`

| Flag | 说明 |
|---|---|
| \`--force\` | 同 name/version 已存在时覆盖（用于本地文件） |
| \`--hub\` | Hub 地址（默认 \`$BOXLI_HUB\` 或 \`http://127.0.0.1:3727\`） |
| \`--data-dir\` | 数据目录（默认 \`$BOXLI_HOME\` 或 \`~/.boxli\`） |

\`\`\`bash
# 从本地文件导入
boxli pull ./myapp.boxli

# 已存在同名同版本时覆盖
boxli pull ./myapp.boxli --force

# 查看结果
boxli images
\`\`\`

> \`boxli import\` / \`boxli load\` 也能把 \`.boxli\` 文件导入为本地镜像，作用与
> \`pull <file>\` 相同。

## 运行容器

\`\`\`
Usage:
  boxli run [flags] <image> [command...]
\`\`\`

默认**前台运行**，stdio 直连容器，\`Ctrl+C\` 停止容器；加 \`-d\` 后台运行并打印容器 ID。

> **注意 flag 位置**：\`run\` 自身的 flag 必须写在镜像引用**之前**，镜像之后的内容一律作为
> 容器命令原样传入（与 Docker 一致）。

\`\`\`bash
# 前台运行
boxli run myapp:v1

# 后台运行
boxli run -d --name web -p 8080:80 myapp:v1

# 镜像之后的内容是容器内命令，foo=bar 不会被 boxli 解析
boxli run -e FOO=bar myapp:v1 sh -c 'echo $FOO'
\`\`\`

### 常用 flag

| Flag | 说明 |
|---|---|
| \`-d, --detach\` | 后台运行，打印容器 ID |
| \`--name\` | 容器名（默认自动生成） |
| \`-e, --env\` | 环境变量 \`KEY=VALUE\`，可重复 |
| \`-p, --publish\` | 端口映射 \`HOST:CONTAINER[:PROTO]\` |
| \`-v, --volume\` | 卷挂载 \`SRC:TARGET[:ro]\`，SRC 可为宿主路径或命名卷 |
| \`--network\` | 接入网络：\`boxli0\`(bridge) \\| \`host\` \\| \`none\` \\| 自定义（默认 \`boxli0\`） |
| \`--restart\` | 重启策略：\`no\` \\| \`always\` \\| \`unless-stopped\` \\| \`on-failure\` |
| \`--memory\` | 内存上限（MiB，0=不限制） |
| \`--cpus\` | CPU 配额（核数，0=不限制） |
| \`--cpuset-cpus\` | 允许使用的 CPU 列表（如 \`0-3,7\`） |
| \`--workdir\` | 工作目录 |
| \`--user\` | 运行用户 \`uid:gid\` |
| \`--entrypoint\` | 覆盖镜像 entrypoint |
| \`--hostname\` | 容器主机名（默认取容器名） |
| \`--pids-limit\` | 进程数上限（0=不限制） |
| \`--storage\` | 可写层存储配额（MiB，0=不限制） |
| \`--blkio-weight\` | 块设备相对权重 [10,1000]，0=不设置 |
| \`--gpu\` / \`--npu\` | 直通 GPU / NPU 设备数（0=不直通） |
| \`--network-bandwidth\` | 出向带宽上限（如 \`10mbps\`） |

## 管理容器

\`\`\`bash
boxli ps              # 仅运行中
boxli ps -a           # 含已停止
boxli ps -q           # 只输出容器 ID
boxli stats           # CPU / 内存 / 进程用量
boxli exec -it web sh # 进入运行中的容器
boxli stop web        # 停止（宽限默认 15s，超时 SIGKILL）
boxli rm web          # 删除已停止的容器
boxli rm -f web       # 运行中也删除（先停止）
\`\`\`

> \`boxli stop\` 会先写 \`stopped-by-user\` 标记，再向 shim 发 SIGTERM（转发给容器 init）。
> 若容器 1 号进程未自行处理 SIGTERM，内核会忽略该信号（PID-1 保护），
> 此时会在宽限到期后被强杀（退出码 137）——**这是预期行为，不是缺陷**。`

const IMAGE_FORMAT = `\`.boxli\` 是 Boxli 自研的镜像格式，**不兼容 Docker / OCI**，
也不使用 Docker Registry 协议。

## 与 Docker 镜像的关键差异

| 维度 | Docker / OCI | Boxli |
|---|---|---|
| 格式 | OCI Image Spec | 自研 \`.boxli\` 单文件格式 |
| 分发 | Registry API（\`/v2/\`） | 本地文件 + Hub **元数据索引** |
| 与 Hub 的关系 | 推送整个 blob | Hub **只记录下载地址**，不存镜像本体 |
| 依赖 | containerd / runc 等 | 单个 Go 二进制 |

## Hub 只收录元数据

这是本站最核心的设计。一个镜像条目由以下字段构成：

\`\`\`json
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
    }
  ]
}
\`\`\`

| 字段 | 含义 |
|---|---|
| \`type\` | 源类型，前端显示对应图标 |
| \`url\` | 实际下载地址 |
| \`priority\` | 优先级，**数字越小越优先** |
| \`region\` | \`global\` / \`cn\` / \`us\` / \`eu\` |
| \`digest\` | 文件 SHA256（可选，用于校验完整性） |
| \`size\` | 文件大小（字节） |

## 下载源类型

| 类型 | 说明 |
|---|---|
| \`github\` | GitHub Releases（全球） |
| \`gitlab\` | GitLab Releases |
| \`gitee\` | Gitee Releases（国内快） |
| \`http\` | 任意 URL |
| \`s3\` | AWS S3 |
| \`oss\` | 阿里云 OSS |
| \`cos\` | 腾讯云 COS |
| \`ipfs\` | \`ipfs://Qm...\` |
| \`magnet\` | BT 磁力链接 |
| \`direct\` | 直链 |

## 为什么这样设计

| 好处 | 说明 |
|---|---|
| 去中心化 | Hub 挂了，下载地址依然有效 |
| 多区域加速 | 国内走 Gitee / OSS / COS，海外走 GitHub / S3 |
| 抗单点 | 一个源不可用时切换同标签的其它源 |
| 零带宽成本 | Hub 不承担镜像分发带宽 |
| 格式灵活 | 网盘直链、IPFS、BT 都可以作为下载源 |
| 可审计 | 下载源可附 SHA256 digest 校验 |

## 多架构镜像

同一仓库按 \`tag + os + arch\` 三个维度区分标签，因此一个 \`v1.0.0\` 可以同时存在
\`linux/amd64\`、\`linux/arm64\`、\`linux/riscv64\` 等多个条目，各自拥有独立的下载源列表。

## 用 Boxfile 构建

\`\`\`
Usage:
  boxli build [--file Boxfile] [--tag NAME:VERSION] [context] [flags]
\`\`\`

\`build\` 解析 Boxfile 指令（\`FROM\` / \`COPY\` / \`ENV\` / \`WORKDIR\` / \`ENTRYPOINT\` /
\`CMD\` / \`EXPOSE\` / \`VOLUME\` / \`LABEL\` / \`USER\` / \`ARG\`），对基础镜像或
\`scratch\` 执行指令，构造 \`.boxli\` 镜像并自动导入本地 store。

| Flag | 说明 |
|---|---|
| \`-f, --file\` | Boxfile 路径（默认 \`<context>/Boxfile\` 或 \`./Boxfile\`） |
| \`-t, --tag\` | 镜像引用 \`NAME:VERSION\`（默认由 Boxfile \`FROM\` 或文件名派生） |
| \`--context\` | 构建上下文目录（\`COPY\` 源相对它解析，默认 \`.\`） |
| \`--no-cache\` | 禁用构建缓存 |
| \`--slim\` | 构建精简镜像（当前与常规构建同） |

\`\`\`bash
# 生成脚手架
boxli scaffold init

# 构建并打标签
boxli build -t myapp:v1 .

# 导出为 .boxli 文件，便于上传到 GitHub Releases / OSS 等
boxli save myapp:v1 -o myapp.boxli
\`\`\`

> \`boxli save\` 与 \`boxli export\` 都把本地镜像写出为 \`.boxli\` 文件；
> 用 \`-o\` 指定输出路径（必填）。导出后即可自行上传到任意支持直链的地方。`

const SUBMIT_GUIDE = `Hub 收录的是**元数据**：镜像名称、描述、标签、架构，以及**去哪下载**。
镜像文件本体请自行上传到 GitHub Releases、Gitee、OSS、S3 等任意可直链访问的地方。

## 提交前准备

1. 用 Boxfile 构建镜像：\`boxli build -t myapp:v1 .\`
2. 导出为文件：\`boxli save myapp:v1 -o myapp.boxli\`
3. 上传到至少一个能直链下载的位置，拿到 URL
4. 建议记录文件大小与 SHA256，便于用户校验

## 在网站提交

登录后进入 [提交镜像](/submit)，填写：

- **镜像名**：\`命名空间/名称\`，例如 \`alice/myapp\`
- **描述**：一句话说明用途
- **README**：Markdown，会渲染在详情页
- **标签**：版本号 + 操作系统 + 架构，可添加多个
- **下载源**：每个标签**至少一个**源，可添加多个

> **源在列表中的顺序就是优先级**（第 1 个最优先），不需要手填 \`priority\`。
> 建议把最快的源放最前：国内用户优先 Gitee / OSS / COS，海外优先 GitHub / S3。

## 校验规则

| 规则 | 说明 |
|---|---|
| 名称必填 | \`namespace\` + \`name\` |
| 至少 1 个标签 | 空 \`tags\` 会返回 400 |
| 每标签至少 1 个源 | 无源的标签无法提交 |
| 源必填项 | \`type\` 与 \`url\` 都不能为空 |

## 管理已提交的镜像

在 [用户中心](/dashboard) 可以查看、编辑、删除自己提交的镜像。

> **⚠️ 编辑是「整体替换」语义**：保存时会用表单内容**完整替换**该镜像的标签与下载源，
> 表单里没有的标签和源会被删除。前端进入编辑时会自动拉取全量详情预填，
> 请确认内容完整后再保存。

## 权限说明

- 只有**提交者本人**能修改或删除自己的镜像，他人操作会返回 403
- 删除镜像会**级联删除**其下所有标签与下载源，且不可恢复`

const FAQ = `## CLI 与 Hub 的对接现状

**这是当前最重要的一条说明，请先阅读。**

本站后端是一套**自研的元数据索引 API**（\`/api/v1/*\`，GitHub OAuth 登录，只存元数据）。
而本机安装的 \`boxli\` CLI（\`0.0.0-dev\`）对接的是**另一套 Hub 实现**（\`boxli hub serve\`：
用户名/密码登录 + blob 存储）。两者**并非同一实现**，因此：

实测结果（2026-10-02）：

| 命令 | 结果 |
|---|---|
| \`boxli search --hub http://127.0.0.1:3727\` | \`未登录 http://127.0.0.1:3727，请先 boxli login\` |
| \`boxli login --hub http://127.0.0.1:3727\` | \`hub 404 Not Found\` |

**已经可用**：

- 镜像构建与本地管理：\`build\`、\`pull <file.boxli>\`、\`images\`、\`tag\`、\`save\`、\`import\`
- 容器生命周期：\`run\`、\`ps\`、\`exec\`、\`stop\`、\`rm\`、\`stats\`
- 编排与资源：\`compose\`、\`volume\`、\`network\`、\`dev\`、\`lint\`、\`doctor\`

**尚不可用**：

- \`boxli search\` / \`login\` / \`push\` / \`pull NAME:VERSION\` **直连本站 Hub**

因此现阶段请通过**网页端**浏览与获取镜像：在[浏览页](/explore)或镜像详情页找到下载源，
直接点击即可从对应的 GitHub / Gitee / OSS 等地址下载 \`.boxli\` 文件，
再用 \`boxli pull <file.boxli>\` 导入。

> 本节内容基于实测，避免给出无法执行的命令。CLI 与本站 Hub 的对接为后续工作项。

## 为什么要先 \`boxli doctor\`

容器依赖内核的 namespace 与 cgroup 能力。\`doctor\` 会检查**内核 / namespace /
cgroup / systemd / 存储**五项，提前暴露环境问题，避免 \`run\` 时才失败。
加 \`--test-run\` 可以跑一次真实容器冒烟测试。

## \`boxli run\` 的 flag 为什么必须写在镜像前面

因为镜像之后的内容要**原样传给容器**作为命令。若把 \`-e\` 写在镜像后面，
它会被当成容器里的命令而不是 Boxli 的参数：

\`\`\`bash
# 正确：-e 被 boxli 解析，容器执行 sh -c 'echo $FOO'
boxli run -e FOO=bar myapp:v1 sh -c 'echo $FOO'

# 错误：-e 会被当作容器命令
boxli run myapp:v1 -e FOO=bar
\`\`\`

这与 Docker 的行为一致。

## 停止容器后退出码是 137

\`boxli stop\` 先发 SIGTERM 并等待宽限（默认 15s）。若容器 1 号进程没有自行处理 SIGTERM，
内核会忽略该信号（PID-1 保护），于是在宽限到期后被 SIGKILL —— 退出码 137。
**这是预期行为**，不是缺陷。要让容器优雅退出，请在镜像里正确处理 SIGTERM。

## 删除容器会删掉镜像吗

不会。\`boxli rm\` 只移除该容器目录下的配置、运行状态、日志与**该容器独占**的 rootfs；
共享的镜像层缓存（\`layers/sha256/<hex>\`）会保留，其它容器仍可复用。
运行中的容器会拒绝删除，需先 \`boxli stop\`，或用 \`-f\` 强制删除。

## \`pull\` / \`import\` / \`load\` 有什么区别

三者在「把 \`.boxli\` 文件变成本地镜像」这件事上作用相同。
\`pull\` 是主入口（同时支持传 Hub 的 \`NAME:VERSION\`），\`import\` 与 \`load\` 是等价的别名式命令。

## Hub 会存储我的镜像文件吗

**不会。** Hub 只保存元数据 —— 名称、描述、README、标签、架构，以及一组下载地址。
镜像文件始终在你自己选择的地方（GitHub Releases / Gitee / OSS / S3 / IPFS / BT 等）。
这也是为什么镜像可以添加多个下载源：某个源失效时用户仍可从其它源下载。

## 支持哪些架构

后端按 \`tag + os + arch\` 三个维度存储标签，因此没有硬性架构限制。
目前社区收录的示例覆盖 \`linux/amd64\`、\`linux/arm64\` 与 \`linux/riscv64\`。

## 登录不了 / 一直显示未登录

1. 确认浏览器已允许本站 Cookie —— 会话存在 **httpOnly Cookie** 中，禁用 Cookie 会导致登录态无法保持
2. 若从 GitHub 授权跳回后仍显示未登录，重新发起一次登录（授权 state 为**一次性**，刷新或后退重放会失效）
3. 本地以 \`http\` 访问时，会话 Cookie 必须允许非 Secure 传输；生产环境为 HTTPS

## 提交的镜像没出现在浏览页

确认提交时**每个标签至少有一个下载源**，且必填的名称、标签、源地址都已填写完整。
若仍不显示，请在 [用户中心](/dashboard) 检查该镜像是否创建成功。

## 如何校验下载的镜像完整性

提交者可以在下载源上附 SHA256 \`digest\`。下载后自行计算比对：

\`\`\`bash
sha256sum myapp.boxli
\`\`\``

export const DOC_PAGES: DocPage[] = [
  {
    path: '/docs',
    title: 'Boxli 文档',
    navTitle: '文档首页',
    heading: 'Boxli 文档',
    description: 'Boxli 轻量级容器引擎与 Hub 镜像索引站的使用文档。',
    sections: [
      {
        id: 'overview',
        title: '总览',
        body: `Boxli 是一个用 Go 编写的**轻量级容器引擎**，自研 \`.boxli\` 镜像格式，
不兼容 Docker / OCI。本文档站覆盖从安装、构建、运行到通过 Hub 分发镜像的完整流程。

## 阅读顺序建议

| 顺序 | 章节 | 适合 |
|---|---|---|
| 1 | [快速开始](/docs/quickstart) | 想先把容器跑起来 |
| 2 | [安装与自检](/docs/install) | 需要确认环境是否支持 |
| 3 | [拉取与运行](/docs/pull-run) | 日常最常用的两条命令 |
| 4 | [镜像格式与下载源](/docs/format) | 想理解 \`.boxli\` 与多源设计 |
| 5 | [提交镜像到 Hub](/docs/submit) | 想把自己的镜像收录进来 |
| 6 | [常见问题](/docs/faq) | 遇到问题先查这里 |

## 快速索引

**想立刻跑一个容器？**

\`\`\`bash
boxli doctor
boxli pull ./myapp.boxli
boxli run myapp:v1
\`\`\`

**想找镜像？** 去[浏览页](/explore)，或在镜像详情页直接点下载源。

**想发布镜像？** 见[提交镜像到 Hub](/docs/submit)。

> **⚠️ 关于 CLI 与本站 Hub 的对接现状**：\`boxli search\` / \`login\` / \`push\` 目前
> **还不能**对接本站 Hub（实测 \`hub 404 Not Found\`）。现阶段请通过网页端获取下载地址。
> 详见[常见问题 · CLI 与 Hub 的对接现状](/docs/faq#cli-与-hub-的对接现状)。`,
      },
    ],
  },
  {
    path: '/docs/quickstart',
    title: '快速开始 · Boxli 文档',
    navTitle: '快速开始',
    heading: '快速开始',
    description: '三步把 Boxli 容器跑起来：环境自检、导入镜像、运行容器。',
    sections: [
      { id: 'quickstart', title: '快速开始', body: QUICKSTART },
    ],
  },
  {
    path: '/docs/install',
    title: '安装与自检 · Boxli 文档',
    navTitle: '安装与自检',
    heading: '安装与自检',
    description: '从源码构建安装 Boxli，并用 doctor 完成环境自检。',
    sections: [
      { id: 'install', title: '安装与自检', body: INSTALL },
    ],
  },
  {
    path: '/docs/pull-run',
    title: '拉取与运行 · Boxli 文档',
    navTitle: '拉取与运行',
    heading: '拉取与运行',
    description: 'boxli pull 与 boxli run 的完整用法与常用参数。',
    sections: [
      { id: 'pull', title: '导入镜像', body: PULL_RUN.split('## 运行容器')[0]!.trimEnd() },
      { id: 'run', title: '运行容器', body: `## 运行容器${PULL_RUN.split('## 运行容器')[1]}` },
    ],
  },
  {
    path: '/docs/format',
    title: '镜像格式与下载源 · Boxli 文档',
    navTitle: '镜像格式与下载源',
    heading: '镜像格式与下载源',
    description: '.boxli 镜像格式、Hub 元数据模型、多下载源设计与 Boxfile 构建。',
    sections: [
      { id: 'format', title: '镜像格式', body: IMAGE_FORMAT.split('## 多架构镜像')[0]!.trimEnd() },
      { id: 'arch', title: '多架构镜像', body: `## 多架构镜像${IMAGE_FORMAT.split('## 多架构镜像')[1]!.split('## 用 Boxfile 构建')[0]}` },
      { id: 'boxfile', title: '用 Boxfile 构建', body: `## 用 Boxfile 构建${IMAGE_FORMAT.split('## 用 Boxfile 构建')[1]}` },
    ],
  },
  {
    path: '/docs/submit',
    title: '提交镜像到 Hub · Boxli 文档',
    navTitle: '提交镜像到 Hub',
    heading: '提交镜像到 Hub',
    description: '把镜像元数据与多下载源提交到 Boxli Hub 的完整流程。',
    sections: [
      { id: 'submit', title: '提交镜像到 Hub', body: SUBMIT_GUIDE },
    ],
  },
  {
    path: '/docs/faq',
    title: '常见问题 · Boxli 文档',
    navTitle: '常见问题',
    heading: '常见问题',
    description: 'Boxli 与 Hub 的常见问题：CLI 对接现状、容器行为、镜像分发与登录问题。',
    sections: [
      { id: 'faq', title: '常见问题', body: FAQ },
    ],
  },
]

/** 侧边栏顺序即数组顺序 */
export function docNav() {
  return DOC_PAGES.map(p => ({ path: p.path, label: p.navTitle }))
}

export function findDocPage(path: string): DocPage | undefined {
  return DOC_PAGES.find(p => p.path === path)
}

/** 上一页 / 下一页，用于页尾翻页 */
export function docNeighbors(path: string) {
  const i = DOC_PAGES.findIndex(p => p.path === path)
  if (i < 0) return { prev: undefined, next: undefined }
  return {
    prev: i > 0 ? DOC_PAGES[i - 1] : undefined,
    next: i < DOC_PAGES.length - 1 ? DOC_PAGES[i + 1] : undefined,
  }
}
