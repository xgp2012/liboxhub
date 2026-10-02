// Boxli Hub 认证状态管理（阶段 4）
//
// 会话由后端以 **httpOnly Cookie** 下发，JS 读不到 token —— 这正是选该方案的原因
// （XSS 无法窃取会话）。因此前端不保存 token，只保存「当前用户是谁」这一衍生状态，
// 真相来源永远是 GET /api/v1/auth/me。
//
// 注意：Cookie 只在浏览器端自动携带，SSR 期间不会带上，所以用户状态**只在客户端拉取**。

import type { ApiEnvelope, AuthUser } from '~/types/api'

export function useAuth() {
  // 跨页面共享同一份用户状态
  const user = useState<AuthUser | null>('auth-user', () => null)
  // 是否已向服务端确认过登录态（避免闪烁与重复请求）
  const resolved = useState<boolean>('auth-resolved', () => false)
  const loading = useState<boolean>('auth-loading', () => false)

  const isLoggedIn = computed(() => user.value !== null)

  /** 向服务端确认当前登录态。失败（未登录/网络错误）一律视为未登录。 */
  async function refresh(): Promise<AuthUser | null> {
    loading.value = true
    try {
      const res = await $fetch<ApiEnvelope<AuthUser>>('/api/v1/auth/me', {
        credentials: 'include',
        // 401 是「未登录」的正常表达，不应抛错中断流程
        ignoreResponseError: true,
      })
      user.value = res.code === 0 && res.data ? res.data : null
    } catch {
      user.value = null
    } finally {
      resolved.value = true
      loading.value = false
    }
    return user.value
  }

  /** 首次需要登录态时调用；已确认过则直接复用。 */
  async function ensure(): Promise<AuthUser | null> {
    if (resolved.value) return user.value
    return refresh()
  }

  /**
   * 发起 GitHub OAuth 登录。
   * redirect 为登录成功后的站内回跳路径（后端限制为相对路径，防开放重定向）。
   */
  async function login(redirect = '/dashboard'): Promise<void> {
    const res = await $fetch<ApiEnvelope<{ authorize_url: string }>>('/api/v1/auth/login', {
      method: 'POST',
      credentials: 'include',
      body: { redirect },
    })
    if (res.code !== 0 || !res.data?.authorize_url) {
      throw new Error(res.message || '无法获取授权地址')
    }
    // 整页跳转到 GitHub（站外），客户端跳转即可
    window.location.href = res.data.authorize_url
  }

  /** 登出：吊销服务端会话并清除 Cookie。 */
  async function logout(): Promise<void> {
    try {
      await $fetch('/api/v1/auth/logout', { method: 'POST', credentials: 'include' })
    } finally {
      user.value = null
      resolved.value = true
    }
  }

  return { user, resolved, loading, isLoggedIn, refresh, ensure, login, logout }
}

/**
 * 带 Cookie 的写请求封装。
 *
 * 所有写接口都需要 credentials: 'include'，否则浏览器不会附带 httpOnly Cookie，
 * 请求会被后端以 401 拒绝。统一走这里，避免漏写。
 *
 * 抛出的错误消息优先取后端 {code,message,data} 里的 message。
 */
export async function apiWrite<T>(
  url: string,
  opts: { method?: 'POST' | 'PUT' | 'DELETE'; body?: unknown } = {},
): Promise<T> {
  const res = await $fetch<ApiEnvelope<T>>(url, {
    method: opts.method ?? 'POST',
    credentials: 'include',
    body: opts.body as Record<string, unknown> | undefined,
    ignoreResponseError: true,
  })
  if (res.code !== 0) {
    throw new Error(res.message || `请求失败（${res.code}）`)
  }
  return res.data
}
