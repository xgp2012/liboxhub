<template>
  <div class="mx-auto w-full max-w-4xl px-4 py-8">
    <header class="mb-6 flex flex-wrap items-start justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">用户中心</h1>
        <p class="mt-1 text-sm text-zinc-500">
          管理你收录的镜像元数据
          <span v-if="user">· {{ user.username }}</span>
        </p>
      </div>
      <NuxtLink
        to="/submit"
        class="flex min-h-11 items-center justify-center rounded-md bg-zinc-100 px-4 text-sm font-medium text-zinc-900 hover:bg-white"
      >
        + 提交镜像
      </NuxtLink>
    </header>

    <!-- 操作反馈 -->
    <div
      v-if="notice"
      class="mb-4 rounded-md border px-4 py-3 text-sm"
      :class="notice.kind === 'error'
        ? 'border-red-900/60 bg-red-950/40 text-red-300'
        : 'border-emerald-900/60 bg-emerald-950/30 text-emerald-300'"
      role="status"
    >
      {{ notice.text }}
    </div>

    <!-- 编辑态：复用提交表单（后端 PUT 为 tags/sources 整体替换） -->
    <section v-if="editing" class="mb-8">
      <div class="mb-4 flex items-center justify-between gap-3">
        <h2 class="text-lg font-medium">
          编辑 {{ editing.namespace }}/{{ editing.name }}
        </h2>
        <button
          type="button"
          class="flex min-h-11 items-center rounded-md border border-zinc-800 px-3 text-sm text-zinc-400 hover:bg-zinc-900"
          @click="cancelEdit"
        >
          退出编辑
        </button>
      </div>
      <RepoForm mode="edit" :repo="editing" @success="onUpdated" />
    </section>

    <!-- 列表 -->
    <section>
      <h2 v-if="!editing" class="mb-3 text-sm font-medium text-zinc-400">
        我的镜像（{{ repos.length }}）
      </h2>

      <div v-if="loading" class="flex flex-col gap-3">
        <CardSkeleton v-for="i in 3" :key="i" />
      </div>

      <EmptyState
        v-else-if="error"
        icon="⚠️"
        title="加载失败"
        :description="error"
      >
        <button
          type="button"
          class="flex min-h-11 items-center justify-center rounded-md border border-zinc-800 px-4 text-sm text-zinc-300 hover:bg-zinc-900"
          @click="load()"
        >
          重试
        </button>
      </EmptyState>

      <EmptyState
        v-else-if="!repos.length"
        icon="📦"
        title="你还没有收录任何镜像"
        description="提交第一个镜像，让它出现在浏览页与搜索里。"
      >
        <NuxtLink
          to="/submit"
          class="flex min-h-11 items-center justify-center rounded-md bg-zinc-100 px-4 text-sm font-medium text-zinc-900 hover:bg-white"
        >
          提交镜像
        </NuxtLink>
      </EmptyState>

      <ul v-else class="flex flex-col gap-3">
        <li
          v-for="repo in repos"
          :key="repo.id"
          class="rounded-lg border border-zinc-800 bg-zinc-900/30 p-4"
        >
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div class="min-w-0 flex-1">
              <NuxtLink
                :to="`/explore/${repo.namespace}/${repo.name}`"
                class="-mx-2 block min-h-11 truncate px-2 py-2 text-base font-medium leading-7 text-zinc-100 hover:text-white"
              >
                {{ repo.namespace }}/{{ repo.name }}
              </NuxtLink>
              <p class="mt-1 line-clamp-2 text-sm leading-relaxed text-zinc-500">
                {{ repo.description || '（无简介）' }}
              </p>
              <p class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-zinc-600">
                <span>★ {{ formatCount(repo.stars) }}</span>
                <span>⤓ {{ formatCount(repo.pulls) }}</span>
                <span>更新于 {{ formatRelativeTime(repo.updated_at, now) }}</span>
              </p>
            </div>

            <div class="flex shrink-0 items-center gap-2">
              <button
                type="button"
                class="flex min-h-11 items-center rounded-md border border-zinc-800 px-3 text-sm text-zinc-300 hover:bg-zinc-900"
                @click="startEdit(repo)"
              >
                编辑
              </button>
              <button
                type="button"
                class="flex min-h-11 items-center rounded-md border border-red-900/60 px-3 text-sm text-red-400 hover:bg-red-950/40 disabled:opacity-50"
                :disabled="deleting === repo.id"
                @click="confirmDelete(repo)"
              >
                {{ deleting === repo.id ? '删除中…' : '删除' }}
              </button>
            </div>
          </div>
        </li>
      </ul>
    </section>
  </div>
</template>

<script setup lang="ts">
import type { ApiEnvelope, RepoDetail, RepoListData, RepoSummary } from '~/types/api'
import { apiWrite } from '~/composables/useAuth'

// 登录态存在 httpOnly Cookie 中，SSR 无法读取，守卫必须在客户端执行
definePageMeta({ middleware: 'auth' })

const { user, ensure } = useAuth()

const repos = ref<RepoSummary[]>([])
const loading = ref(true)
const error = ref('')
const deleting = ref<number | null>(null)
const editing = ref<RepoDetail | null>(null)
const notice = ref<{ kind: 'ok' | 'error'; text: string } | null>(null)

// 用固定时间基准渲染相对时间，避免 SSR/客户端时间差导致 hydration 不一致
const now = ref(0)

function flash(kind: 'ok' | 'error', text: string) {
  notice.value = { kind, text }
  if (kind === 'ok') {
    setTimeout(() => {
      if (notice.value?.text === text) notice.value = null
    }, 4000)
  }
}

/** 拉取当前用户的镜像列表（按 namespace 过滤） */
async function load() {
  loading.value = true
  error.value = ''
  try {
    const me = user.value ?? await ensure()
    if (!me) {
      error.value = '未登录。'
      return
    }
    const res = await $fetch<ApiEnvelope<RepoListData>>('/api/v1/repos', {
      credentials: 'include',
      query: { namespace: me.username, limit: 100 },
      ignoreResponseError: true,
    })
    if (res.code !== 0) throw new Error(res.message || '加载失败')
    repos.value = res.data?.items ?? []
  }
  catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  }
  finally {
    loading.value = false
  }
}

/** 编辑需要 tags/sources 完整数据，列表项不含，故单独取详情 */
async function startEdit(repo: RepoSummary) {
  notice.value = null
  try {
    const res = await $fetch<ApiEnvelope<RepoDetail>>(
      `/api/v1/repos/${repo.namespace}/${repo.name}`,
      { credentials: 'include', ignoreResponseError: true },
    )
    if (res.code !== 0 || !res.data) throw new Error(res.message || '无法加载镜像详情')
    editing.value = res.data
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
  catch (e) {
    flash('error', e instanceof Error ? e.message : '无法加载镜像详情')
  }
}

function cancelEdit() {
  editing.value = null
}

function onUpdated(repo: RepoDetail) {
  editing.value = null
  flash('ok', `已保存 ${repo.namespace}/${repo.name} 的修改。`)
  void load()
}

async function confirmDelete(repo: RepoSummary) {
  const full = `${repo.namespace}/${repo.name}`
  // 删除不可恢复，且会级联删掉所有标签与下载源
  if (!window.confirm(`确定删除 ${full}？\n该操作会同时删除它的全部标签与下载源，且不可恢复。`)) {
    return
  }

  deleting.value = repo.id
  try {
    await apiWrite(`/api/v1/repos/${repo.namespace}/${repo.name}`, { method: 'DELETE' })
    repos.value = repos.value.filter(r => r.id !== repo.id)
    flash('ok', `已删除 ${full}。`)
  }
  catch (e) {
    flash('error', e instanceof Error ? e.message : '删除失败')
  }
  finally {
    deleting.value = null
  }
}

onMounted(async () => {
  now.value = Date.now()
  await ensure()
  await load()
})

useHead({ title: '用户中心 · Boxli Hub' })
</script>
