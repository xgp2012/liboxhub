// esbuild 插件：把 jsdom 的默认样式表**内联**进产物。
//
// 问题：jsdom 在模块加载时执行
//   fs.readFileSync(path.resolve(__dirname, "../../../browser/default-stylesheet.css"))
// 打包成单文件后 __dirname 指向释放目录，相对层级已不成立，
// 会抛 ENOENT（实测报错路径为 /browser/default-stylesheet.css），
// 导致所有渲染 Markdown 的页面（/docs、镜像详情）500。
//
// 解法：在打包期把该调用替换为**内联的字符串常量**，运行时不再读文件。
// 相比"随包释放一个 11KB 的 css 文件并在启动时按层级摆放"，这样更不容易
// 因目录结构变动而再次失效，也让二进制真正自包含。

import { readFileSync } from 'node:fs'

const STYLESHEET_MARKER = 'default-stylesheet.css'

export function jsdomInlinePlugin() {
  return {
    name: 'jsdom-single-file-fixes',
    setup(build) {
      // ---- 1. 内联默认样式表 ----
      build.onLoad({ filter: /jsdom[\\/]lib[\\/]jsdom[\\/]living[\\/]css[\\/]helpers[\\/]computed-style\.js$/ }, (args) => {
        let src = readFileSync(args.path, 'utf8')

        const cssPath = new URL('../../../browser/default-stylesheet.css', `file://${args.path}`)
        let css
        try {
          css = readFileSync(cssPath, 'utf8')
        } catch (err) {
          throw new Error(
            `jsdomInlinePlugin: 无法读取 ${cssPath.pathname}: ${err.message}`,
          )
        }

        const pattern = /fs\.readFileSync\(\s*path\.resolve\(__dirname,\s*["'][^"']*default-stylesheet\.css["']\s*\)\s*,\s*\{[^}]*\}\s*\)/
        if (!pattern.test(src)) {
          throw new Error(
            'jsdomInlinePlugin: 未匹配到 jsdom 的样式表读取语句，' +
            'jsdom 版本可能已变化，请检查 plugins/jsdom-inline-stylesheet.mjs。',
          )
        }
        src = src.replace(pattern, JSON.stringify(css))

        return { contents: src, loader: 'js' }
      })

      // ---- 2. 处理 require.resolve("./xhr-sync-worker.js") ----
      //
      // jsdom 用 require.resolve 定位一个**同步 XHR 的 worker 脚本**。单文件打包后
      // 该文件既未被打出、也无法解析，导致模块加载即失败。
      //
      // 它只在「同步 XMLHttpRequest」路径上被使用，而 DOMPurify 完全不使用 XHR
      // （已核对 dompurify 源码中没有任何 XMLHttpRequest 引用）。因此替换为一个
      // **惰性抛错**的占位：正常路径永不触发；一旦真的被调用会立刻暴露，
      // 而不是静默返回错误结果。
      build.onLoad({ filter: /jsdom[\\/]lib[\\/]jsdom[\\/]living[\\/]xhr[\\/]XMLHttpRequest-impl\.js$/ }, (args) => {
        let src = readFileSync(args.path, 'utf8')

        const pattern = /const syncWorkerFile = require\.resolve\(["']\.\/xhr-sync-worker\.js["']\);/
        if (!pattern.test(src)) {
          throw new Error(
            'jsdomInlinePlugin: 未匹配到 jsdom 的 xhr-sync-worker 解析语句，' +
            'jsdom 版本可能已变化，请检查 plugins/jsdom-inline-stylesheet.mjs。',
          )
        }
        src = src.replace(
          pattern,
          "const syncWorkerFile = null; /* 单文件打包：同步 XHR 不可用（DOMPurify 不使用 XHR） */",
        )

        return { contents: src, loader: 'js' }
      })
    },
  }
}

export { STYLESHEET_MARKER }
