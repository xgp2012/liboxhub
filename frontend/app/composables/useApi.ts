// Boxli Hub API 调用封装
// 走 nuxt.config.ts 的 routeRules proxy（/api/** → 127.0.0.1:3727），无需 CORS

import type {
  ApiEnvelope,
  ReadmeData,
  RepoDetail,
  RepoListData,
  SearchData,
  TagListData,
} from '~/types/api'

/** 从统一信封中取出 data，失败抛错 */
function unwrap<T>(res: ApiEnvelope<T>): T {
  if (res.code !== 0) {
    throw new Error(res.message || `API error ${res.code}`)
  }
  return res.data
}

/**
 * 带统一解包的 useFetch。
 * 后端所有响应都是 {code,message,data}，这里直接返回 data 便于页面消费。
 */
function useApiData<T>(url: string, opts: Record<string, unknown> = {}) {
  return useFetch(url, {
    ...opts,
    transform: (res: ApiEnvelope<T>) => unwrap(res),
  // eslint-disable-next-line @typescript-eslint/no-explicit-any -- useFetch 重载对 options 推断过窄
  } as any)
}

/** 搜索仓库：GET /api/v1/search?q=&limit=
 *  仅当关键词非空时才发起请求（空关键词返回空结果，避免无意义请求）。 */
export function useSearch(q: MaybeRefOrGetter<string>, limit = 20) {
  const query = computed(() => toValue(q).trim())
  const hasQuery = computed(() => query.value.length > 0)
  return useApiData<SearchData>('/api/v1/search', {
    query: { q: query, limit },
    immediate: hasQuery.value,
    watch: [query],
  })
}

/** 仓库列表：GET /api/v1/repos?namespace=&limit=&offset= */
export function useRepoList(params: {
  namespace?: MaybeRefOrGetter<string | undefined>
  limit?: number
  offset?: MaybeRefOrGetter<number>
}) {
  const namespace = computed(() => toValue(params.namespace) || undefined)
  const offset = computed(() => toValue(params.offset) || 0)
  return useApiData<RepoListData>('/api/v1/repos', {
    query: { namespace, limit: params.limit ?? 20, offset },
    watch: [namespace, offset],
  })
}

/** 仓库详情：GET /api/v1/repos/{ns}/{repo} */
export function useRepoDetail(ns: MaybeRefOrGetter<string>, repo: MaybeRefOrGetter<string>) {
  const path = computed(() => `/api/v1/repos/${toValue(ns)}/${toValue(repo)}`)
  return useApiData<RepoDetail>(path, { watch: [path] })
}

/** 标签列表：GET /api/v1/repos/{ns}/{repo}/tags */
export function useRepoTags(ns: MaybeRefOrGetter<string>, repo: MaybeRefOrGetter<string>) {
  const path = computed(() => `/api/v1/repos/${toValue(ns)}/${toValue(repo)}/tags`)
  return useApiData<TagListData>(path, { watch: [path] })
}

/** README：GET /api/v1/repos/{ns}/{repo}/readme */
export function useRepoReadme(ns: MaybeRefOrGetter<string>, repo: MaybeRefOrGetter<string>) {
  const path = computed(() => `/api/v1/repos/${toValue(ns)}/${toValue(repo)}/readme`)
  return useApiData<ReadmeData>(path, { watch: [path] })
}

/** 健康检查 */
export function useHealth() {
  return useApiData<{ status: string }>('/api/v1/health')
}
