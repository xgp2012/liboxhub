import type { SourceType } from '~/types/api'

/** 字节数格式化为人类可读，如 5242880 → "5.0 MB" */
export function formatBytes(bytes?: number | null): string {
  if (bytes === null || bytes === undefined || Number.isNaN(bytes)) return '—'
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  const idx = Math.min(i, units.length - 1)
  const value = bytes / 1024 ** idx
  return `${value.toFixed(idx === 0 ? 0 : 1)} ${units[idx]}`
}

/** 下载量紧凑格式，如 120300 → "120.3k" */
export function formatCount(n?: number | null): string {
  if (n === null || n === undefined || Number.isNaN(n)) return '0'
  if (n < 1000) return String(n)
  if (n < 1_000_000) return `${(n / 1000).toFixed(1).replace(/\.0$/, '')}k`
  return `${(n / 1_000_000).toFixed(1).replace(/\.0$/, '')}M`
}

/**
 * 相对时间（中文）。为避免 SSR / 客户端时间不一致导致 hydration 报错，
 * 传入固定基准时间或只在客户端调用。
 */
export function formatRelativeTime(iso: string, now: number = Date.now()): string {
  const then = new Date(iso).getTime()
  if (Number.isNaN(then)) return '—'
  const diff = Math.max(0, now - then)
  const min = 60_000
  const hour = 60 * min
  const day = 24 * hour

  if (diff < min) return '刚刚'
  if (diff < hour) return `${Math.floor(diff / min)} 分钟前`
  if (diff < day) return `${Math.floor(diff / hour)} 小时前`
  if (diff < 30 * day) return `${Math.floor(diff / day)} 天前`
  if (diff < 365 * day) return `${Math.floor(diff / (30 * day))} 个月前`
  return `${Math.floor(diff / (365 * day))} 年前`
}

/** 源类型展示元数据：图标符号 + 中文名 + 区域建议标签 */
export interface SourceMeta {
  label: string
  icon: string
  hint?: string
}

export const SOURCE_META: Record<SourceType, SourceMeta> = {
  github: { label: 'GitHub Releases', icon: '🌍', hint: '推荐' },
  gitlab: { label: 'GitLab Releases', icon: '🦊' },
  gitee: { label: 'Gitee', icon: '🇨🇳', hint: '国内快' },
  http: { label: 'HTTP', icon: '🔗' },
  s3: { label: 'AWS S3', icon: '☁️' },
  oss: { label: '阿里云 OSS', icon: '☁️', hint: '国内快' },
  cos: { label: '腾讯云 COS', icon: '☁️', hint: '国内快' },
  ipfs: { label: 'IPFS', icon: '🧩' },
  magnet: { label: 'BT 磁力', icon: '🧲' },
  direct: { label: '直链', icon: '🔗' },
}

export function sourceMeta(type: string): SourceMeta {
  return SOURCE_META[type as SourceType] ?? { label: type, icon: '🔗' }
}

/** 区域展示 */
export function regionLabel(region?: string | null): string | null {
  if (!region) return null
  const map: Record<string, string> = {
    global: '全球',
    cn: '中国大陆',
    us: '美国',
    eu: '欧洲',
  }
  return map[region] ?? region
}

/** 生成 pull 命令 */
export function pullCommand(ns: string, name: string, tag?: string): string {
  return `boxli pull ${ns}/${name}${tag ? `:${tag}` : ''}`
}

/** 复制到剪贴板，带降级方案（http 非安全上下文下 navigator.clipboard 不可用） */
export async function copyText(text: string): Promise<boolean> {
  try {
    if (import.meta.client && navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text)
      return true
    }
  }
  catch {
    // 落到下面的降级方案
  }
  if (!import.meta.client) return false
  try {
    const ta = document.createElement('textarea')
    ta.value = text
    ta.setAttribute('readonly', '')
    ta.style.position = 'fixed'
    ta.style.top = '-9999px'
    document.body.appendChild(ta)
    ta.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(ta)
    return ok
  }
  catch {
    return false
  }
}
