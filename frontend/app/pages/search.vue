<template>
  <div class="mx-auto w-full max-w-6xl px-4 py-8">
    <header class="mb-6">
      <h1 class="text-2xl font-semibold tracking-tight">搜索镜像</h1>
      <p class="mt-1 text-sm text-zinc-500">按名称、描述或命名空间搜索</p>
    </header>

    <!-- 搜索框：输入框字号 ≥16px 防 iOS 缩放；键盘弹出不遮挡结果（结果在下方正常流） -->
    <form class="mb-6" role="search" @submit.prevent="submit">
      <label class="sr-only" for="search-input">搜索关键词</label>
      <div class="flex gap-2">
        <input
          id="search-input"
          v-model="keyword"
          type="search"
          placeholder="例如 myapp、postgres、redis…"
          enterkeyhint="search"
          class="h-12 min-w-0 flex-1 rounded-md border border-zinc-800 bg-zinc-900/60 px-3 text-base text-zinc-100 placeholder:text-zinc-500 focus:border-zinc-600 focus:outline-none"
        >
        <button
          type="submit"
          class="flex min-h-12 shrink-0 items-center rounded-md bg-zinc-100 px-5 text-sm font-medium text-zinc-900 hover:bg-white"
        >
          搜索
        </button>
      </div>
    </form>

    <!-- 未输入关键词：给推荐入口 -->
    <div v-if="!activeQuery">
      <p class="mb-3 text-sm text-zinc-500">试试这些关键词：</p>
      <div class="flex flex-wrap gap-2">
        <NuxtLink
          v-for="s in suggestions"
          :key="s"
          :to="{ path: '/search', query: { q: s } }"
          class="flex min-h-11 items-center rounded-md border border-zinc-800 bg-zinc-900/40 px-4 text-sm text-zinc-300 hover:bg-zinc-900"
        >
          {{ s }}
        </NuxtLink>
      </div>
    </div>

    <template v-else>
      <p v-if="!pending && !error" class="mb-3 text-sm text-zinc-500">
        关键词「{{ activeQuery }}」· 找到 {{ results.length }} 个结果
      </p>

      <div v-if="pending" class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <CardSkeleton v-for="i in 3" :key="i" />
      </div>

      <EmptyState
        v-else-if="error"
        icon="⚠️"
        title="搜索失败"
        description="无法连接到 Hub 后端，请稍后重试。"
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
        v-else-if="!results.length"
        icon="📭"
        title="没有找到匹配的镜像"
        description="换个关键词，或者去浏览全部镜像。"
      >
        <NuxtLink
          to="/explore"
          class="flex min-h-11 items-center justify-center rounded-md border border-zinc-800 px-4 text-sm text-zinc-300 hover:bg-zinc-900"
        >
          浏览全部
        </NuxtLink>
      </EmptyState>

      <div v-else class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <RepoCard v-for="repo in results" :key="repo.id" :repo="repo" />
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
const route = useRoute()
const router = useRouter()

const keyword = ref(typeof route.query.q === 'string' ? route.query.q : '')
const activeQuery = ref(keyword.value)

const { data, pending, error, refresh } = await useSearch(activeQuery, 40)

const results = computed(() => data.value?.items ?? [])

const suggestions = ['myapp', 'postgres', 'redis', 'nginx', 'riscv']

watch(() => route.query.q, (q) => {
  const next = typeof q === 'string' ? q : ''
  keyword.value = next
  activeQuery.value = next
})

function submit() {
  const q = keyword.value.trim()
  activeQuery.value = q
  router.replace({ path: '/search', query: q ? { q } : {} })
}

useHead(() => ({
  title: activeQuery.value ? `搜索「${activeQuery.value}」· Boxli Hub` : '搜索镜像 · Boxli Hub',
}))
</script>
