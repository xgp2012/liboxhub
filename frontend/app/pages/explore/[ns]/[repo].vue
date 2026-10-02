<template>
  <div class="mx-auto w-full max-w-4xl px-4 py-6">
    <!-- 面包屑 -->
    <nav class="mb-4 flex items-center gap-2 text-sm text-zinc-500" aria-label="面包屑">
      <NuxtLink to="/explore" class="-ml-2 flex min-h-11 min-w-11 items-center px-2 hover:text-zinc-300">浏览</NuxtLink>
      <span aria-hidden="true">/</span>
      <span class="truncate text-zinc-400">{{ ns }}/{{ name }}</span>
    </nav>

    <div v-if="pending" class="flex flex-col gap-4">
      <CardSkeleton />
      <CardSkeleton />
    </div>

    <EmptyState
      v-else-if="error || !repo"
      icon="🚫"
      title="镜像不存在"
      description="该仓库可能已被删除，或从未被收录。"
    >
      <NuxtLink
        to="/explore"
        class="flex min-h-11 items-center justify-center rounded-md border border-zinc-800 px-4 text-sm text-zinc-300 hover:bg-zinc-900"
      >
        返回浏览
      </NuxtLink>
    </EmptyState>

    <article v-else class="flex flex-col gap-6">
      <!-- 头部信息 -->
      <header class="flex flex-col gap-3">
        <div class="flex items-start gap-3">
          <span
            class="grid h-12 w-12 shrink-0 place-items-center rounded-lg bg-zinc-800 text-lg font-semibold text-zinc-300"
            aria-hidden="true"
          >
            {{ repo.name.slice(0, 1).toUpperCase() }}
          </span>
          <div class="min-w-0 flex-1">
            <h1 class="break-words text-xl font-semibold tracking-tight sm:text-2xl">
              <span class="text-zinc-500">{{ repo.namespace }}/</span>{{ repo.name }}
            </h1>
            <p class="mt-1 text-sm leading-relaxed text-zinc-400">
              {{ repo.description || '暂无描述' }}
            </p>
          </div>
        </div>

        <div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-zinc-500">
          <span>作者 {{ repo.author }}</span>
          <span>★ {{ formatCount(repo.stars) }}</span>
          <span>↓ {{ formatCount(repo.pulls) }}</span>
          <ClientOnly>
            <span>更新于 {{ formatRelativeTime(repo.updated_at, mountTime) }}</span>
            <template #fallback><span class="text-zinc-600">—</span></template>
          </ClientOnly>
        </div>
      </header>

      <!-- tags 与多源下载 -->
      <section v-if="repo.tags?.length" class="rounded-lg border border-zinc-800 bg-zinc-900/30 p-4">
        <h2 class="mb-4 text-base font-medium">下载</h2>
        <SourceList :tags="repo.tags" :namespace="repo.namespace" :name="repo.name" />
      </section>

      <EmptyState
        v-else
        icon="🏷️"
        title="还没有可用标签"
        description="该镜像尚未提交任何版本标签。"
      />

      <!-- 支持架构概览 -->
      <section v-if="repo.tags?.length">
        <h2 class="mb-2 text-xs font-medium uppercase tracking-wider text-zinc-500">支持架构</h2>
        <div class="flex flex-wrap gap-2">
          <span
            v-for="arch in archList"
            :key="arch"
            class="rounded-md border border-zinc-800 bg-zinc-900/40 px-2.5 py-1 font-mono text-xs text-zinc-400"
          >{{ arch }}</span>
        </div>
      </section>

      <!-- README -->
      <section v-if="repo.readme">
        <h2 class="mb-3 text-base font-medium">README</h2>
        <!-- eslint-disable-next-line vue/no-v-html -- 内容已在 useMarkdown 中经 markdown-it 解析 + DOMPurify 清洗，html:false 禁用原始 HTML -->
        <div class="readme rounded-lg border border-zinc-800 bg-zinc-900/30 p-4" v-html="readmeHtml" />
      </section>
    </article>
  </div>
</template>

<script setup lang="ts">
const route = useRoute()
const ns = computed(() => String(route.params.ns ?? ''))
const name = computed(() => String(route.params.repo ?? ''))

const { data: repo, pending, error } = await useRepoDetail(ns, name)

const readmeHtml = useMarkdown(() => repo.value?.readme ?? '')

const mountTime = ref(0)
onMounted(() => {
  mountTime.value = Date.now()
})

const archList = computed(() => {
  const set = new Set((repo.value?.tags ?? []).map(t => `${t.os}/${t.arch}`))
  return [...set]
})

useHead(() => ({
  title: repo.value
    ? `${repo.value.namespace}/${repo.value.name} · Boxli Hub`
    : '镜像详情 · Boxli Hub',
  meta: [
    {
      name: 'description',
      content: repo.value?.description ?? 'Boxli 镜像详情',
    },
  ],
}))
</script>

<style scoped>
/* README 渲染样式（scoped + :deep，避免影响全局） */
.readme :deep(h1) { font-size: 1.375rem; font-weight: 600; margin: 1.25rem 0 0.75rem; }
.readme :deep(h2) { font-size: 1.125rem; font-weight: 600; margin: 1.25rem 0 0.5rem; }
.readme :deep(h3) { font-size: 1rem; font-weight: 600; margin: 1rem 0 0.5rem; }
.readme :deep(p) { margin: 0.75rem 0; line-height: 1.7; color: #d4d4d8; }
.readme :deep(a) { color: #a1a1aa; text-decoration: underline; text-underline-offset: 2px; }
.readme :deep(ul), .readme :deep(ol) { margin: 0.75rem 0; padding-left: 1.25rem; line-height: 1.7; }
.readme :deep(ul) { list-style: disc; }
.readme :deep(ol) { list-style: decimal; }
.readme :deep(li) { margin: 0.25rem 0; color: #d4d4d8; }
.readme :deep(code) {
  background: #27272a; padding: 0.125rem 0.375rem; border-radius: 0.25rem;
  font-size: 0.85em; font-family: ui-monospace, monospace;
}
.readme :deep(pre) {
  background: #09090b; border: 1px solid #27272a; border-radius: 0.375rem;
  padding: 0.75rem; overflow-x: auto; margin: 0.75rem 0;
}
.readme :deep(pre code) { background: transparent; padding: 0; font-size: 0.8125rem; }
.readme :deep(blockquote) {
  border-left: 3px solid #3f3f46; padding-left: 0.75rem; margin: 0.75rem 0; color: #a1a1aa;
}
.readme :deep(table) { width: 100%; border-collapse: collapse; margin: 0.75rem 0; display: block; overflow-x: auto; }
.readme :deep(th), .readme :deep(td) { border: 1px solid #27272a; padding: 0.375rem 0.625rem; text-align: left; }
.readme :deep(th) { background: #18181b; font-weight: 600; }
.readme :deep(img) { max-width: 100%; height: auto; border-radius: 0.375rem; }
.readme :deep(hr) { border-color: #27272a; margin: 1.25rem 0; }
/* 首尾元素去掉多余外边距 */
.readme :deep(> :first-child) { margin-top: 0; }
.readme :deep(> :last-child) { margin-bottom: 0; }
</style>
