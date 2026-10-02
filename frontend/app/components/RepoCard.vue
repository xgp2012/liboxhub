<template>
  <NuxtLink
    :to="`/explore/${repo.namespace}/${repo.name}`"
    class="block rounded-lg border border-zinc-800 bg-zinc-900/40 p-4 transition-colors hover:border-zinc-700 hover:bg-zinc-900 active:bg-zinc-900"
  >
    <div class="flex items-start gap-3">
      <span
        class="mt-0.5 grid h-9 w-9 shrink-0 place-items-center rounded-md bg-zinc-800 text-sm font-semibold text-zinc-300"
        aria-hidden="true"
      >
        {{ repo.name.slice(0, 1).toUpperCase() }}
      </span>
      <div class="min-w-0 flex-1">
        <p class="truncate text-base font-medium text-zinc-100">
          <span class="text-zinc-500">{{ repo.namespace }}/</span>{{ repo.name }}
        </p>
        <p class="mt-1 line-clamp-2 text-sm leading-relaxed text-zinc-400">
          {{ repo.description || '暂无描述' }}
        </p>
        <div class="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-zinc-500">
          <span>★ {{ formatCount(repo.stars) }}</span>
          <span>↓ {{ formatCount(repo.pulls) }}</span>
          <ClientOnly>
            <span>{{ formatRelativeTime(repo.updated_at, mountTime) }}</span>
            <template #fallback>
              <span class="text-zinc-600">—</span>
            </template>
          </ClientOnly>
        </div>
      </div>
    </div>
  </NuxtLink>
</template>

<script setup lang="ts">
import type { RepoSummary } from '~/types/api'

defineProps<{ repo: RepoSummary }>()

// 相对时间在 SSR 与客户端会有微小差异，用挂载时刻固定基准避免 hydration 报错
const mountTime = ref(0)
onMounted(() => {
  mountTime.value = Date.now()
})
</script>
