<template>
  <div class="flex flex-col gap-4">
    <!-- 标签 / 架构选择器：窄屏横向滚动，避免挤压 -->
    <div>
      <p class="mb-2 text-xs font-medium uppercase tracking-wider text-zinc-500">版本与架构</p>
      <div
        class="-mx-4 flex gap-2 overflow-x-auto px-4 pb-1"
        role="tablist"
        aria-label="选择版本与架构"
      >
        <button
          v-for="t in tags"
          :key="keyOf(t)"
          type="button"
          role="tab"
          :aria-selected="keyOf(t) === keyOf(active)"
          class="flex min-h-11 shrink-0 flex-col items-start justify-center rounded-md border px-3 text-left transition-colors"
          :class="keyOf(t) === keyOf(active)
            ? 'border-zinc-600 bg-zinc-800 text-zinc-100'
            : 'border-zinc-800 bg-zinc-900/40 text-zinc-400 hover:bg-zinc-900'"
          @click="activeKey = keyOf(t)"
        >
          <span class="text-sm font-medium">{{ t.tag }}</span>
          <span class="text-xs text-zinc-500">{{ t.os }}/{{ t.arch }}</span>
        </button>
      </div>
    </div>

    <!-- 当前选中标签的元信息 + pull 命令 -->
    <div v-if="active" class="flex flex-col gap-3">
      <div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-zinc-400">
        <span>{{ active.tag }} · {{ active.os }}/{{ active.arch }}</span>
        <span v-if="active.size_bytes">{{ formatBytes(active.size_bytes) }}</span>
        <span class="text-zinc-600">{{ active.sources.length }} 个下载源</span>
      </div>

      <CopyButton :text="pullCmd" />
    </div>

    <!-- 下载源列表 -->
    <div>
      <p class="mb-2 text-xs font-medium uppercase tracking-wider text-zinc-500">
        下载源（按优先级排序）
      </p>
      <ul class="flex flex-col gap-2">
        <li
          v-for="src in sortedSources"
          :key="src.id ?? src.url"
          class="rounded-lg border border-zinc-800 bg-zinc-900/40 p-3"
        >
          <div class="flex items-center gap-3">
            <span class="text-lg" aria-hidden="true">{{ sourceMeta(src.type).icon }}</span>
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-2">
                <span class="text-sm font-medium text-zinc-200">{{ sourceMeta(src.type).label }}</span>
                <span
                  v-if="sourceMeta(src.type).hint"
                  class="rounded bg-zinc-800 px-1.5 py-0.5 text-xs text-zinc-400"
                >{{ sourceMeta(src.type).hint }}</span>
                <span v-if="regionLabel(src.region)" class="text-xs text-zinc-500">
                  {{ regionLabel(src.region) }}
                </span>
                <span v-if="src.priority === topPriority" class="text-xs text-emerald-400">优先级最高</span>
              </div>
              <!-- 长 URL 用 break-all 避免横向溢出 -->
              <p class="mt-1 break-all font-mono text-xs leading-relaxed text-zinc-500">{{ src.url }}</p>
            </div>
          </div>

          <div class="mt-3 flex items-center gap-2">
            <a
              :href="src.url"
              target="_blank"
              rel="noopener noreferrer nofollow"
              class="flex min-h-11 flex-1 items-center justify-center rounded-md bg-zinc-100 px-4 text-sm font-medium text-zinc-900 transition-colors hover:bg-white active:bg-zinc-200"
            >
              {{ isDirectLink(src.type) ? '下载' : '打开链接' }}
            </a>
            <button
              type="button"
              class="flex min-h-11 shrink-0 items-center rounded-md border border-zinc-800 px-4 text-sm text-zinc-300 hover:bg-zinc-900 active:bg-zinc-800"
              :aria-label="`复制链接：${src.url}`"
              @click="copySource(src.url)"
            >
              {{ copiedUrl === src.url ? '✓ 已复制' : '复制链接' }}
            </button>
          </div>

          <p v-if="src.size_bytes" class="mt-2 text-xs text-zinc-600">
            文件大小 {{ formatBytes(src.size_bytes) }}
          </p>
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Tag } from '~/types/api'

const props = defineProps<{
  tags: Tag[]
  namespace: string
  name: string
}>()

const activeKey = ref('')

function keyOf(t: Tag) {
  return `${t.tag}|${t.os}|${t.arch}`
}

// 默认选中第一个标签
watchEffect(() => {
  if (!activeKey.value && props.tags.length > 0) {
    activeKey.value = keyOf(props.tags[0]!)
  }
})

const active = computed(() => props.tags.find(t => keyOf(t) === activeKey.value) ?? props.tags[0])

const sortedSources = computed(() =>
  [...(active.value?.sources ?? [])].sort((a, b) => (a.priority ?? 100) - (b.priority ?? 100)),
)

const topPriority = computed(() =>
  sortedSources.value.length > 0 ? sortedSources.value[0]!.priority : -1,
)

const pullCmd = computed(() => {
  const t = active.value
  if (!t) return ''
  return pullCommand(props.namespace, props.name, t.tag)
})

/** http 直链类源可直接下载；ipfs/magnet 需本地客户端接管，改称「打开链接」 */
function isDirectLink(type: string) {
  return ['github', 'gitlab', 'gitee', 'http', 's3', 'oss', 'cos', 'direct'].includes(type)
}

const copiedUrl = ref('')
let timer: ReturnType<typeof setTimeout> | undefined
async function copySource(url: string) {
  const ok = await copyText(url)
  if (!ok) return
  copiedUrl.value = url
  clearTimeout(timer)
  timer = setTimeout(() => {
    copiedUrl.value = ''
  }, 2000)
}
onBeforeUnmount(() => clearTimeout(timer))
</script>
