<template>
  <div class="rounded-lg border border-zinc-800 bg-zinc-900/30 p-4">
    <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
      <h3 class="text-sm font-medium text-zinc-300">
        下载源
        <span class="ml-1 text-xs font-normal text-zinc-500">（按优先级从高到低排列）</span>
      </h3>
      <button
        type="button"
        class="flex min-h-11 items-center rounded-md border border-zinc-800 px-3 text-sm text-zinc-300 hover:bg-zinc-900"
        @click="addSource"
      >
        + 添加下载源
      </button>
    </div>

    <p v-if="!sources.length" class="rounded-md border border-dashed border-zinc-800 px-4 py-6 text-center text-sm text-zinc-500">
      至少添加 1 个下载源；建议同时提供国内与海外源。
    </p>

    <ul v-else class="flex flex-col gap-3">
      <li
        v-for="(source, index) in sources"
        :key="source.key"
        class="rounded-md border border-zinc-800 bg-zinc-950/60 p-3"
      >
        <div class="mb-2 flex items-center justify-between gap-2">
          <span class="text-xs text-zinc-500">
            <span aria-hidden="true">{{ sourceMeta(source.type).icon }}</span>
            源 {{ index + 1 }}
            <span v-if="index === 0" class="ml-1 text-zinc-400">· 最高优先级</span>
          </span>
          <button
            type="button"
            class="flex min-h-11 min-w-11 items-center justify-center rounded-md px-2 text-sm text-zinc-500 hover:bg-zinc-900 hover:text-red-400"
            :aria-label="`删除第 ${index + 1} 个下载源`"
            @click="removeSource(index)"
          >
            ✕
          </button>
        </div>

        <div class="grid gap-2 sm:grid-cols-2">
          <div>
            <label :for="`src-type-${source.key}`" class="mb-1 block text-xs text-zinc-500">类型</label>
            <select
              :id="`src-type-${source.key}`"
              :value="source.type"
              class="h-11 w-full rounded-md border border-zinc-800 bg-zinc-900/60 px-3 text-base text-zinc-200 focus:border-zinc-600 focus:outline-none"
              @change="update(index, { type: ($event.target as HTMLSelectElement).value as SourceType })"
            >
              <option v-for="opt in SOURCE_OPTIONS" :key="opt.value" :value="opt.value">
                {{ opt.icon }} {{ opt.label }}
              </option>
            </select>
          </div>
          <div>
            <label :for="`src-region-${source.key}`" class="mb-1 block text-xs text-zinc-500">区域</label>
            <select
              :id="`src-region-${source.key}`"
              :value="source.region"
              class="h-11 w-full rounded-md border border-zinc-800 bg-zinc-900/60 px-3 text-base text-zinc-200 focus:border-zinc-600 focus:outline-none"
              @change="update(index, { region: ($event.target as HTMLSelectElement).value })"
            >
              <option v-for="opt in REGION_OPTIONS" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </option>
            </select>
          </div>
        </div>

        <div class="mt-2">
          <label :for="`src-url-${source.key}`" class="mb-1 block text-xs text-zinc-500">下载地址</label>
          <input
            :id="`src-url-${source.key}`"
            :value="source.url"
            type="url"
            inputmode="url"
            autocapitalize="off"
            autocomplete="off"
            spellcheck="false"
            placeholder="https://github.com/…/releases/download/v1/app.boxli"
            class="h-11 w-full rounded-md border border-zinc-800 bg-zinc-900/60 px-3 text-base text-zinc-100 placeholder:text-zinc-600 focus:border-zinc-600 focus:outline-none"
            @input="update(index, { url: ($event.target as HTMLInputElement).value })"
          >
        </div>
      </li>
    </ul>

    <p class="mt-3 text-xs leading-relaxed text-zinc-600">
      Hub 只记录「去哪下载」，不存储镜像文件本体。
    </p>
  </div>
</template>

<script setup lang="ts">
import type { SourceInput, SourceType } from '~/types/api'
import { SOURCE_META, sourceMeta } from '~/utils/format'

/** 带稳定 key 的源行（key 仅用于渲染，不提交给后端） */
export interface SourceRow extends SourceInput {
  key: string
}

// 不直接改 prop：源列表由父组件持有，这里只发事件，
// 避免隐式依赖「数组按引用传递」这种脆弱假设。
const props = defineProps<{ sources: SourceRow[] }>()
const emit = defineEmits<{ (e: 'change', sources: SourceRow[]): void }>()

const SOURCE_OPTIONS = (Object.keys(SOURCE_META) as SourceType[]).map(value => ({
  value,
  label: SOURCE_META[value].label,
  icon: SOURCE_META[value].icon,
}))

const REGION_OPTIONS = [
  { value: 'global', label: '全球' },
  { value: 'cn', label: '中国大陆' },
  { value: 'us', label: '美国' },
  { value: 'eu', label: '欧洲' },
]

let seq = 0

/** 复制为新数组并重排 priority，保证「展示顺序 == 优先级」 */
function commit(next: SourceRow[]) {
  emit('change', next.map((s, i) => ({ ...s, priority: i + 1 })))
}

function addSource() {
  seq += 1
  const row: SourceRow = {
    key: `src-${Date.now()}-${seq}`,
    type: 'github',
    url: '',
    priority: props.sources.length + 1,
    region: 'global',
  }
  commit([...props.sources, row])
}

function removeSource(index: number) {
  commit(props.sources.filter((_, i) => i !== index))
}

function update(index: number, patch: Partial<SourceInput>) {
  commit(props.sources.map((s, i) => (i === index ? { ...s, ...patch } : s)))
}
</script>
