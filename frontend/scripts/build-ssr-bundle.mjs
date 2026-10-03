#!/usr/bin/env node
// 把 Nuxt 的 SSR 产物（.output/server/）打包成**单个自包含的 .mjs 文件**，
// 供 Go 二进制以 go:embed 内嵌，运行时释放到临时目录交给 node 执行。
//
// 为什么需要这一步：
//   .output/server/ 依赖同目录下的 node_modules（约 19MB，含 8.6MB jsdom）。
//   若直接内嵌整个目录，体积大且存在大量小文件；esbuild 把它们内联进单个文件后，
//   产物不再需要 node_modules，可整体嵌入二进制。
//
// 实测（2026-10-03）：打包后 15.2MB，gzip 约 2.0MB，在**空目录**下可直接运行。
//
// ⚠️ 只对 SSR bundle 做 Node 侧打包；客户端静态资源（.output/public）保持原样，
//    由 Go 直接以 http.FileServer 伺服，不经过 node。

import { build } from 'esbuild'
import {
  cpSync,
  existsSync,
  mkdirSync,
  readFileSync,
  readdirSync,
  rmSync,
  statSync,
} from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { gzipSync } from 'node:zlib'

import { jsdomInlinePlugin } from './plugins/jsdom-inline-stylesheet.mjs'

const here = dirname(fileURLToPath(import.meta.url))
const frontendDir = resolve(here, '..')                 // frontend/
const serverDir = join(frontendDir, '.output', 'server')
const entry = join(serverDir, 'index.mjs')

// 输出到 backend 内的可提交目录，供 go:embed 使用。
// 注意：不能指向 frontend/.output（该目录被 gitignore，且 embed 需要仓库内稳定路径）。
const outDir = resolve(frontendDir, '..', 'backend', 'internal', 'web', 'dist')
const outFile = join(outDir, 'ssr.mjs')

if (!existsSync(entry)) {
  console.error(`✗ 未找到 ${entry}`)
  console.error('  请先在 frontend/ 执行：npm run build')
  process.exit(1)
}

mkdirSync(outDir, { recursive: true })

// jsdom（isomorphic-dompurify 的 Node 依赖）是 CommonJS，内部使用**动态**
// require('node:fs') 以及 __dirname / __filename 等 CJS 专有全局量。
// esbuild 打成 ESM 后这些都无法解析，运行时会连续抛错：
//   "Dynamic require of node:fs is not supported"
//   → "ReferenceError: __dirname is not defined"
// 导致任何渲染 Markdown 的页面（/docs、镜像详情）500。
//
// 解法：在 ESM 产物顶部注入 CJS 兼容层。这是 esbuild 在 ESM 输出下
// 打包 CJS 依赖的标准做法。
const cjsShim = [
  "import { createRequire as __boxliCreateRequire } from 'node:module';",
  "import { fileURLToPath as __boxliFileURLToPath } from 'node:url';",
  "import { dirname as __boxliDirname } from 'node:path';",
  'const require = __boxliCreateRequire(import.meta.url);',
  'const __filename = __boxliFileURLToPath(import.meta.url);',
  'const __dirname = __boxliDirname(__filename);',
].join('\n')

console.log('▸ 打包 SSR bundle …')
try {
  await build({
    entryPoints: [entry],
    bundle: true,
    platform: 'node',
    target: 'node20',
    format: 'esm',
    outfile: outFile,
    // 保留 node 内置模块为外部依赖（由 node 运行时提供）
    external: ['node:*'],
    banner: { js: cjsShim },
    // 把 jsdom 的默认样式表内联，避免运行时按 __dirname 相对路径找不到文件
    plugins: [jsdomInlinePlugin()],
    logLevel: 'warning',
  })
} catch (err) {
  console.error('✗ esbuild 打包失败：', err.message)
  process.exit(1)
}

const mb = n => (n / 1048576).toFixed(1)
const raw = statSync(outFile).size
const gz = gzipSync(readFileSync(outFile)).length
console.log(`✓ SSR bundle: ${outFile}`)
console.log(`  原始 ${mb(raw)} MB / gzip ${mb(gz)} MB`)

// ---------------------------------------------------------------------------
// 客户端静态资源：整体拷进 dist/public，由 Go 直接伺服（不经过 node）。
// 同样不能用 frontend/.output（被 gitignore），必须落在仓库内可 embed 的位置。
// ---------------------------------------------------------------------------
const publicSrc = join(frontendDir, '.output', 'public')
const publicDst = join(outDir, 'public')

if (!existsSync(publicSrc)) {
  console.error(`✗ 未找到 ${publicSrc}`)
  process.exit(1)
}

console.log('▸ 拷贝客户端静态资源 …')
rmSync(publicDst, { recursive: true, force: true })
cpSync(publicSrc, publicDst, { recursive: true })

function dirSize(p) {
  let total = 0
  for (const e of readdirSync(p, { withFileTypes: true })) {
    const full = join(p, e.name)
    total += e.isDirectory() ? dirSize(full) : statSync(full).size
  }
  return total
}

console.log(`✓ 静态资源: ${publicDst}  (${mb(dirSize(publicDst))} MB)`)
console.log('')
console.log('全部产物已就绪，可执行：cd backend && go build -o boxli-hub ./cmd/hub')
