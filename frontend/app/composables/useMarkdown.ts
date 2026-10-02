// README Markdown 渲染（SSR 安全）
// markdown-it 解析 + DOMPurify 清洗，防止仓库 README 注入脚本

import MarkdownIt from 'markdown-it'
import DOMPurify from 'isomorphic-dompurify'

const md = new MarkdownIt({
  html: false, // 禁用原始 HTML，从源头避免注入
  linkify: true,
  breaks: false,
  typographer: false,
})

// 外链加 rel/target，避免 tabnabbing
const defaultLinkOpen
  = md.renderer.rules.link_open
    ?? ((tokens, idx, options, _env, self) => self.renderToken(tokens, idx, options))

md.renderer.rules.link_open = (tokens, idx, options, env, self) => {
  const token = tokens[idx]
  const href = token.attrGet('href') ?? ''
  if (/^https?:\/\//i.test(href)) {
    token.attrSet('target', '_blank')
    token.attrSet('rel', 'noopener noreferrer nofollow')
  }
  return defaultLinkOpen(tokens, idx, options, env, self)
}

/**
 * 生成标题锚点 slug。
 *
 * 规则与常见 Markdown 站点（GitHub 风格）一致，便于手写深链：
 * 小写 → 去掉行内代码反引号与标点 → 空白转 `-`；保留中日韩文字。
 * 例："CLI 与 Hub 的对接现状" → "cli-与-hub-的对接现状"
 */
function slugify(text: string): string {
  return text
    .trim()
    .toLowerCase()
    .replace(/`/g, '')
    .replace(/[\s]+/g, '-')
    // 保留：字母、数字、下划线、连字符、中日韩文字
    .replace(/[^\p{L}\p{N}_-]/gu, '')
    .replace(/-{2,}/g, '-')
    .replace(/^-+|-+$/g, '')
}

/**
 * 给标题加 `id`，使 `#anchor` 深链可用。
 * 仅在 env.headingAnchors !== false 时写入（文档站启用，README 关闭）。
 * 同一页重复标题追加 `-1`、`-2` 后缀，避免 id 冲突。
 */
md.renderer.rules.heading_open = (tokens, idx, options, env, self) => {
  const token = tokens[idx]
  const inline = tokens[idx + 1]
  const raw = inline?.type === 'inline' ? inline.content : ''
  const slug = slugify(raw)
  if (slug && env.headingAnchors !== false) {
    const seen = (env.headingSlugs ??= new Map<string, number>())
    const n = seen.get(slug) ?? 0
    seen.set(slug, n + 1)
    token.attrSet('id', n === 0 ? slug : `${slug}-${n}`)
  }
  return self.renderToken(tokens, idx, options)
}

export interface RenderMarkdownOptions {
  /** 为标题注入 id 锚点（文档站需要，README 不需要） */
  anchors?: boolean
}

/**
 * 把 Markdown 渲染为安全的 HTML 字符串。
 * 在服务端与客户端产出相同结果（不依赖 Date/随机），避免 hydration 不一致。
 */
export function renderMarkdown(
  source?: string | null,
  options: RenderMarkdownOptions = {},
): string {
  if (!source) return ''
  // 锚点规则常驻在 renderer 上；通过 env 开关控制是否真正写入 id
  const env = options.anchors ? {} : { headingAnchors: false }
  const raw = md.render(source, env)
  return DOMPurify.sanitize(raw, {
    // id 供锚点跳转使用；target/rel 供外链安全
    ADD_ATTR: ['target', 'rel', 'id'],
    FORBID_TAGS: ['style', 'form', 'input', 'iframe', 'script'],
  })
}

/** 供模板直接使用的 composable */
export function useMarkdown(source: MaybeRefOrGetter<string | null | undefined>) {
  return computed(() => renderMarkdown(toValue(source)))
}
