<template>
  <div class="mx-auto w-full max-w-6xl px-4 py-8">
    <header class="mb-6">
      <h1 class="text-2xl font-semibold tracking-tight">浏览镜像</h1>
      <p class="mt-1 text-sm text-zinc-500">社区提交的 .boxli 镜像元数据索引</p>
    </header>

    <!-- 搜索 + 排序：手机堆叠，md 起同行 -->
    <div class="mb-6 flex flex-col gap-3 md:flex-row md:items-center">
      <form class="flex-1" role="search" @submit.prevent="applySearch">
        <label class="sr-only" for="explore-search">搜索镜像</label>
        <div class="flex gap-2">
          <input
            id="explore-search"
            v-model="keyword"
            type="search"
            placeholder="搜索名称、描述或命名空间…"
            class="h-11 min-w-0 flex-1 rounded-md border border-zinc-800 bg-zinc-900/60 px-3 text-base text-zinc-100 placeholder:text-zinc-500 focus:border-zinc-600 focus:outline-none"
          >
          <button
            type="submit"
            class="flex min-h-11 shrink-0 items-center rounded-md bg-zinc-100 px-4 text-sm font-medium text-zinc-900 hover:bg-white"
          >
            搜索
          </button>
        </div>
      </form>

      <div class="flex items-center gap-2">
        <label class="sr-only" for="explore-sort">排序</label>
        <select
          id="explore-sort"
          v-model="sort"
          class="h-11 rounded-md border border-zinc-800 bg-zinc-900/60 px-3 text-base text-zinc-300 focus:border-zinc-600 focus:outline-none"
        >
          <option value="stars">按星标</option>
          <option value="pulls">按下载量</option>
          <option value="updated">按更新时间</option>
        </select>
      </div>
    </div>

    <!-- 结果统计 -->
    <p v-if="!pending && !error" class="mb-3 text-sm text-zinc-500">
      共 {{ sorted.length }} 个镜像
      <span v-if="query">· 关键词「{{ query }}」</span>
    </p>

    <div v-if="pending" class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      <CardSkeleton v-for="i in 6" :key="i" />
    </div>

    <EmptyState
      v-else-if="error"
      icon="⚠️"
      title="加载失败"
      description="无法连接到 Hub 后端，请确认服务已启动。"
    >
      <button
        type="button"
        class="flex min-h-11 items-center justify-center rounded-md border border-zinc-800 px-4 text-sm text-zinc-300 hover:bg-zinc-900"
        @click="refresh()"
      >
        重试
      </button>
    </EmptyState>

    <EmptyState
      v-else-if="!sorted.length"
      icon="📭"
      :title="query ? '没有匹配的镜像' : '还没有收录镜像'"
      :description="query ? '换个关键词试试，或浏览全部镜像。' : '成为第一个提交镜像的人。'"
    />

    <div v-else class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      <RepoCard v-for="repo in sorted" :key="repo.id" :repo="repo" />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { RepoSummary } from '~/types/api'

const route = useRoute()
const router = useRouter()

const keyword = ref(typeof route.query.q === 'string' ? route.query.q : '')
const query = ref(keyword.value)
const sort = ref<'stars' | 'pulls' | 'updated'>('stars')

// 统一走搜索接口：关键词为空时 useSearch 不发起请求，
// 此时回退到全量列表，避免条件调用 composable。
const { data: searchData, pending: searchPending, error: searchError, refresh: searchRefresh } = useSearch(query, 60)
const { data: listData, pending: listPending, error: listError, refresh: listRefresh } = await useRepoList({ limit: 60 })

const usingSearch = computed(() => query.value.length > 0)
const data = computed(() => (usingSearch.value ? searchData.value : listData.value))
const pending = computed(() => (usingSearch.value ? searchPending.value : listPending.value))
const error = computed(() => (usingSearch.value ? searchError.value : listError.value))
const refresh = () => (usingSearch.value ? searchRefresh() : listRefresh())

// 监听 q 变化（如从 Header 搜索框跳转过来）
watch(() => route.query.q, (q) => {
  const next = typeof q === 'string' ? q : ''
  keyword.value = next
  query.value = next
})

function applySearch() {
  const q = keyword.value.trim()
  query.value = q
  router.replace({ path: '/explore', query: q ? { q } : {} })
}

const sorted = computed<RepoSummary[]>(() => {
  const items = data.value?.items ?? []
  const list = [...items]
  if (sort.value === 'stars') list.sort((a, b) => b.stars - a.stars)
  else if (sort.value === 'pulls') list.sort((a, b) => b.pulls - a.pulls)
  else list.sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime())
  return list
})

useHead({ title: '浏览镜像 · Boxli Hub' })
</script>
