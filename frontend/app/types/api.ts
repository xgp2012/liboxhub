// Boxli Hub API 类型定义
// 与后端 internal/hub/store.go、store_write.go 的响应结构保持一致

/** 统一响应信封：{code, message, data}，成功 code=0 */
export interface ApiEnvelope<T> {
  code: number
  message: string
  data: T
}

export interface Source {
  id?: number
  type: SourceType
  url: string
  priority: number
  region?: string | null
  digest?: string | null
  size_bytes?: number | null
}

export type SourceType =
  | 'github'
  | 'gitlab'
  | 'gitee'
  | 'http'
  | 's3'
  | 'oss'
  | 'cos'
  | 'ipfs'
  | 'magnet'
  | 'direct'

export interface Tag {
  id?: number
  tag: string
  os: string
  arch: string
  digest?: string | null
  size_bytes?: number | null
  sources: Source[]
}

/** 列表项（/repos、/search 返回的精简结构） */
export interface RepoSummary {
  id: number
  namespace: string
  name: string
  description: string | null
  stars: number
  pulls: number
  updated_at: string
}

/** 详情（/repos/{ns}/{repo}） */
export interface RepoDetail extends RepoSummary {
  readme: string | null
  is_public: boolean
  owner_id: number
  author: string
  created_at: string
  tags: Tag[]
}

export interface RepoListData {
  count: number
  items: RepoSummary[]
}

export interface SearchData extends RepoListData {
  query: string
}

export interface TagListData {
  count: number
  items: Tag[]
}

export interface ReadmeData {
  readme: string
}

// ---- 阶段 4：认证 ----

/** 当前登录用户（GET /auth/me 的 data） */
export interface AuthUser {
  id: number
  username: string
  email: string
  avatar_url: string
}

/** GET /auth/login 在已配 OAuth 时返回的授权信息 */
export interface LoginData {
  authorize_url: string
  state: string
  state_expires_at: string
}

/** 写接口提交的下载源 */
export interface SourceInput {
  type: SourceType
  url: string
  priority: number
  region: string
  digest?: string
  size_bytes?: number
}

/** 写接口提交的标签 */
export interface TagInput {
  tag: string
  os: string
  arch: string
  digest?: string
  size_bytes?: number
  sources: SourceInput[]
}

/** POST /repos、PUT /repos/{ns}/{repo} 的请求体 */
export interface RepoInput {
  namespace: string
  name: string
  description: string
  readme: string
  tags: TagInput[]
}
