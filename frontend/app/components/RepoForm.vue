<template>
  <form class="flex flex-col gap-5" @submit.prevent="onSubmit">
    <div
      v-if="errorMessage"
      class="rounded-md border border-red-900/60 bg-red-950/40 px-4 py-3 text-sm text-red-300"
      role="alert"
    >
      {{ errorMessage }}
    </div>

    <!-- 基本信息 -->
    <div class="rounded-lg border border-zinc-800 bg-zinc-900/30 p-4">
      <h3 class="mb-3 text-sm font-medium text-zinc-300">基本信息</h3>

      <div class="grid gap-3 sm:grid-cols-2">
        <div>
          <label for="repo-ns" class="mb-1 block text-xs text-zinc-500">命名空间</label>
          <input
            id="repo-ns"
            v-model.trim="form.namespace"
            type="text"
            :readonly="mode === 'edit'"
            :class="mode === 'edit' ? 'opacity-60' : ''"
            placeholder="你的用户名"
            class="h-11 w-full rounded-md border border-zinc-800 bg-zinc-900/60 px-3 text-base text-zinc-100 placeholder:text-zinc-600 focus:border-zinc-600 focus:outline-none"
          >
          <p class="mt-1 text-xs text-zinc-600">
            {{ mode === 'edit' ? '命名空间不可修改' : '留空则使用你的 GitHub 用户名' }}
          </p>
        </div>
        <div>
          <label for="repo-name" class="mb-1 block text-xs text-zinc-500">镜像名</label>
          <input
            id="repo-name"
            v-model.trim="form.name"
            type="text"
            :readonly="mode === 'edit'"
            :class="mode === 'edit' ? 'opacity-60' : ''"
            placeholder="myapp"
            required
            class="h-11 w-full rounded-md border border-zinc-800 bg-zinc-900/60 px-3 text-base text-zinc-100 placeholder:text-zinc-600 focus:border-zinc-600 focus:outline-none"
          >
          <p class="mt-1 text-xs text-zinc-600">
            {{ mode === 'edit' ? '镜像名不可修改' : '小写字母、数字、- 和 _' }}
          </p>
        </div>
      </div>

      <div class="mt-3">
        <label for="repo-desc" class="mb-1 block text-xs text-zinc-500">简介</label>
        <input
          id="repo-desc"
          v-model.trim="form.description"
          type="text"
          placeholder="一句话说明这个镜像是什么"
          class="h-11 w-full rounded-md border border-zinc-800 bg-zinc-900/60 px-3 text-base text-zinc-100 placeholder:text-zinc-600 focus:border-zinc-600 focus:outline-none"
        >
      </div>

      <div class="mt-3">
        <label for="repo-readme" class="mb-1 block text-xs text-zinc-500">README（Markdown）</label>
        <textarea
          id="repo-readme"
          v-model="form.readme"
          rows="6"
          placeholder="# myapp&#10;&#10;使用方法…"
          class="w-full rounded-md border border-zinc-800 bg-zinc-900/60 px-3 py-2 text-base leading-relaxed text-zinc-100 placeholder:text-zinc-600 focus:border-zinc-600 focus:outline-none"
        />
      </div>
    </div>

    <!-- 标签与架构 -->
    <div class="rounded-lg border border-zinc-800 bg-zinc-900/30 p-4">
      <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3 class="text-sm font-medium text-zinc-300">标签</h3>
        <button
          type="button"
          class="flex min-h-11 items-center rounded-md border border-zinc-800 px-3 text-sm text-zinc-300 hover:bg-zinc-900"
          @click="addTag"
        >
          + 添加标签
        </button>
      </div>

      <ul class="flex flex-col gap-4">
        <li v-for="(tag, ti) in form.tags" :key="tag.key" class="rounded-md border border-zinc-800 bg-zinc-950/60 p-3">
          <div class="mb-2 flex items-center justify-between gap-2">
            <span class="text-xs text-zinc-500">标签 {{ ti + 1 }}</span>
            <button
              v-if="form.tags.length > 1"
              type="button"
              class="flex min-h-11 min-w-11 items-center justify-center rounded-md px-2 text-sm text-zinc-500 hover:bg-zinc-900 hover:text-red-400"
              :aria-label="`删除第 ${ti + 1} 个标签`"
              @click="removeTag(ti)"
            >
              ✕
            </button>
          </div>

          <div class="grid gap-2 sm:grid-cols-3">
            <div>
              <label :for="`tag-name-${tag.key}`" class="mb-1 block text-xs text-zinc-500">标签名</label>
              <input
                :id="`tag-name-${tag.key}`"
                v-model.trim="tag.tag"
                type="text"
                placeholder="v1.0.0"
                class="h-11 w-full rounded-md border border-zinc-800 bg-zinc-900/60 px-3 text-base text-zinc-100 placeholder:text-zinc-600 focus:border-zinc-600 focus:outline-none"
              >
            </div>
            <div>
              <label :for="`tag-os-${tag.key}`" class="mb-1 block text-xs text-zinc-500">系统</label>
              <select
                :id="`tag-os-${tag.key}`"
                v-model="tag.os"
                class="h-11 w-full rounded-md border border-zinc-800 bg-zinc-900/60 px-3 text-base text-zinc-200 focus:border-zinc-600 focus:outline-none"
              >
                <option value="linux">linux</option>
                <option value="windows">windows</option>
                <option value="darwin">darwin</option>
              </select>
            </div>
            <div>
              <label :for="`tag-arch-${tag.key}`" class="mb-1 block text-xs text-zinc-500">架构</label>
              <select
                :id="`tag-arch-${tag.key}`"
                v-model="tag.arch"
                class="h-11 w-full rounded-md border border-zinc-800 bg-zinc-900/60 px-3 text-base text-zinc-200 focus:border-zinc-600 focus:outline-none"
              >
                <option value="amd64">amd64</option>
                <option value="arm64">arm64</option>
                <option value="riscv64">riscv64</option>
              </select>
            </div>
          </div>

          <div class="mt-3">
            <SourceEditor :sources="tag.sources" @change="(next) => updateSources(ti, next)" />
          </div>
        </li>
      </ul>
    </div>

    <!-- 提交 -->
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
      <button
        type="submit"
        class="flex min-h-12 items-center justify-center rounded-md bg-zinc-100 px-6 text-sm font-medium text-zinc-900 transition-colors hover:bg-white disabled:opacity-60"
        :disabled="submitting"
      >
        {{ submitting ? '提交中…' : submitLabel }}
      </button>
      <NuxtLink
        to="/dashboard"
        class="flex min-h-12 items-center justify-center rounded-md border border-zinc-800 px-6 text-sm text-zinc-300 hover:bg-zinc-900"
      >
        取消
      </NuxtLink>
    </div>
  </form>
</template>

<script setup lang="ts">
import type { RepoDetail, RepoInput, SourceInput, TagInput } from '~/types/api'
import { apiWrite } from '~/composables/useAuth'
import type { SourceRow } from '~/components/SourceEditor.vue'

/** 内部标签行：带稳定 key + 源行 */
interface TagRow extends Omit<TagInput, 'sources'> {
  key: string
  sources: SourceRow[]
}

const props = withDefaults(defineProps<{
  mode?: 'create' | 'edit'
  /** edit 模式下要编辑的仓库（用于预填与决定 PUT 地址） */
  repo?: RepoDetail | null
  /** create 模式下预填的默认命名空间 */
  defaultNamespace?: string
}>(), {
  mode: 'create',
  repo: null,
  defaultNamespace: '',
})

const emit = defineEmits<{ (e: 'success', repo: RepoDetail): void }>()

let seq = 0
const nextKey = (p: string) => `${p}-${Date.now()}-${++seq}`

function emptyTag(): TagRow {
  return {
    key: nextKey('tag'),
    tag: '',
    os: 'linux',
    arch: 'amd64',
    sources: [
      {
        key: nextKey('src'),
        type: 'github',
        url: '',
        priority: 1,
        region: 'global',
      },
    ],
  }
}

const form = reactive({
  namespace: props.defaultNamespace,
  name: '',
  description: '',
  readme: '',
  tags: [emptyTag()] as TagRow[],
})

// edit 模式：用已有仓库预填（整体替换语义，因此必须回填全部 tags/sources）
watch(() => props.repo, (repo) => {
  if (props.mode !== 'edit' || !repo) return
  form.namespace = repo.namespace
  form.name = repo.name
  form.description = repo.description ?? ''
  form.readme = repo.readme ?? ''
  form.tags = repo.tags.length
    ? repo.tags.map(t => ({
        key: nextKey('tag'),
        tag: t.tag,
        os: t.os,
        arch: t.arch,
        sources: (t.sources ?? []).map((s, i): SourceRow => ({
          key: nextKey('src'),
          type: s.type,
          url: s.url,
          priority: s.priority || i + 1,
          region: s.region ?? 'global',
        })),
      }))
    : [emptyTag()]
}, { immediate: true })

watch(() => props.defaultNamespace, (ns) => {
  if (props.mode === 'create' && !form.namespace) form.namespace = ns
})

const submitting = ref(false)
const errorMessage = ref('')
const submitLabel = computed(() => (props.mode === 'edit' ? '保存修改' : '提交镜像'))

function addTag() {
  form.tags.push(emptyTag())
}

function removeTag(index: number) {
  form.tags.splice(index, 1)
}

/** 接收 SourceEditor 的新源数组（不就地改 prop，保持单向数据流） */
function updateSources(tagIndex: number, next: SourceRow[]) {
  const tag = form.tags[tagIndex]
  if (tag) tag.sources = next
}

/** 提交前做客户端校验，尽早给出可读提示，减少一次无效往返 */
function validate(): string {
  if (!form.name.trim()) return '请填写镜像名。'
  if (props.mode === 'edit' && (!form.namespace || !form.name)) return '命名空间与镜像名不可为空。'
  if (!form.tags.length) return '至少需要一个标签。'

  for (const [i, t] of form.tags.entries()) {
    if (!t.tag.trim()) return `标签 ${i + 1} 缺少标签名。`
    if (!t.sources.length) return `标签「${t.tag || i + 1}」至少需要一个下载源。`
    for (const [j, s] of t.sources.entries()) {
      if (!s.url.trim()) return `标签「${t.tag}」的源 ${j + 1} 缺少下载地址。`
      if (!/^https?:\/\/.+/i.test(s.url.trim())) {
        return `标签「${t.tag}」的源 ${j + 1} 地址需以 http:// 或 https:// 开头。`
      }
    }
  }
  return ''
}

async function onSubmit() {
  if (submitting.value) return
  errorMessage.value = ''

  const invalid = validate()
  if (invalid) {
    errorMessage.value = invalid
    return
  }

  submitting.value = true
  try {
    // 只提交后端认识的字段（剔除仅用于渲染的 key）
    const tags: TagInput[] = form.tags.map(t => ({
      tag: t.tag.trim(),
      os: t.os,
      arch: t.arch,
      sources: t.sources.map((s, i): SourceInput => ({
        type: s.type,
        url: s.url.trim(),
        priority: i + 1,
        region: s.region,
      })),
    }))

    const payload: RepoInput = {
      namespace: form.namespace.trim(),
      name: form.name.trim(),
      description: form.description.trim(),
      readme: form.readme,
      tags,
    }

    const repo = props.mode === 'edit' && props.repo
      ? await apiWrite<RepoDetail>(`/api/v1/repos/${props.repo.namespace}/${props.repo.name}`, {
          method: 'PUT',
          body: payload,
        })
      : await apiWrite<RepoDetail>('/api/v1/repos', { method: 'POST', body: payload })

    emit('success', repo)
  }
  catch (e) {
    const msg = e instanceof Error ? e.message : '提交失败'
    // 常见错误翻译为可操作的提示
    if (/already exists/i.test(msg)) errorMessage.value = '该镜像已存在，请换个名字或到用户中心编辑。'
    else if (/unauthorized|401/i.test(msg)) errorMessage.value = '登录已过期，请重新登录后再提交。'
    else if (/forbidden|403/i.test(msg)) errorMessage.value = '你没有权限修改这个镜像。'
    else errorMessage.value = msg
  }
  finally {
    submitting.value = false
  }
}
</script>
