<template>
  <div class="mx-auto w-full max-w-6xl px-4 py-6 sm:py-8">
    <!-- 面包屑：纯文字链接补 min-w-11 + px-2 保证 ≥44px 触达区 -->
    <nav class="mb-4 flex flex-wrap items-center gap-1 text-sm text-zinc-500" aria-label="面包屑">
      <NuxtLink to="/" class="-ml-2 flex min-h-11 min-w-11 items-center px-2 hover:text-zinc-200">首页</NuxtLink>
      <span aria-hidden="true">/</span>
      <NuxtLink to="/docs" class="flex min-h-11 min-w-11 items-center px-2 hover:text-zinc-200">文档</NuxtLink>
      <template v-if="current && current.path !== '/docs'">
        <span aria-hidden="true">/</span>
        <span class="flex min-h-11 items-center px-2 text-zinc-300">{{ current.navTitle }}</span>
      </template>
    </nav>

    <!-- 手机端：章节折叠导航（不用 hover，点击展开） -->
    <div class="lg:hidden">
      <button
        type="button"
        class="flex min-h-11 w-full items-center justify-between rounded-md border border-zinc-800 bg-zinc-900/40 px-3 text-base text-zinc-200"
        :aria-expanded="mobileNavOpen"
        aria-controls="docs-mobile-nav"
        @click="mobileNavOpen = !mobileNavOpen"
      >
        <span class="flex items-center gap-2">
          <span aria-hidden="true">☰</span>
          <span>文档目录</span>
        </span>
        <span aria-hidden="true">{{ mobileNavOpen ? '▲' : '▼' }}</span>
      </button>

      <nav
        v-show="mobileNavOpen"
        id="docs-mobile-nav"
        class="mt-2 flex flex-col gap-1 rounded-md border border-zinc-800 bg-zinc-900/30 p-2"
        aria-label="文档导航"
      >
        <NuxtLink
          v-for="item in nav"
          :key="item.path"
          :to="item.path"
          class="flex min-h-11 items-center rounded-md px-3 text-base text-zinc-300 hover:bg-zinc-900"
          :class="isActive(item.path) ? 'bg-zinc-900 !text-zinc-100' : ''"
        >
          {{ item.label }}
        </NuxtLink>
      </nav>
    </div>

    <div class="mt-6 flex gap-8 lg:mt-2">
      <!-- 桌面端侧边导航：sticky -->
      <aside class="hidden w-56 shrink-0 lg:block">
        <nav class="sticky top-20 flex flex-col gap-1" aria-label="文档导航">
          <p class="px-3 pb-2 text-xs font-medium uppercase tracking-wider text-zinc-600">文档</p>
          <NuxtLink
            v-for="item in nav"
            :key="item.path"
            :to="item.path"
            class="flex min-h-11 items-center rounded-md px-3 text-sm text-zinc-400 transition-colors hover:bg-zinc-900 hover:text-zinc-100"
            :class="isActive(item.path) ? 'bg-zinc-900 !text-zinc-100' : ''"
          >
            {{ item.label }}
          </NuxtLink>

          <!-- 本页小节锚点 -->
          <template v-if="current && current.sections.length > 1">
            <p class="px-3 pb-1 pt-4 text-xs font-medium uppercase tracking-wider text-zinc-600">本页</p>
            <a
              v-for="s in current.sections"
              :key="s.id"
              :href="`#${s.id}`"
              class="flex min-h-11 items-center rounded-md py-2 pl-6 pr-3 text-sm text-zinc-500 transition-colors hover:text-zinc-200"
            >
              {{ s.title }}
            </a>
          </template>
        </nav>
      </aside>

      <!-- 正文 -->
      <div class="min-w-0 flex-1">
        <article class="min-w-0">
          <header class="mb-6">
            <h1 class="text-2xl font-semibold tracking-tight sm:text-3xl">{{ heading }}</h1>
            <p v-if="current" class="mt-2 text-base leading-relaxed text-zinc-400">{{ current.description }}</p>
          </header>

          <!-- 手机端：本页小节跳转（横向滚动，不撑破窄屏） -->
          <div
            v-if="current && current.sections.length > 1"
            class="-mx-4 mb-6 overflow-x-auto px-4 lg:hidden"
          >
            <div class="flex w-max gap-2">
              <a
                v-for="s in current.sections"
                :key="s.id"
                :href="`#${s.id}`"
                class="flex min-h-11 items-center whitespace-nowrap rounded-md border border-zinc-800 bg-zinc-900/40 px-3 text-sm text-zinc-400"
              >
                {{ s.title }}
              </a>
            </div>
          </div>

          <section
            v-for="s in sections"
            :id="s.id"
            :key="s.id"
            class="scroll-mt-20"
          >
            <!-- eslint-disable-next-line vue/no-v-html -- 内容已在 useMarkdown 中经 markdown-it 解析 + DOMPurify 清洗，html:false 禁用原始 HTML -->
            <div class="docs-prose" v-html="renderSection(s.body)" />
          </section>
        </article>

        <!-- 上一页 / 下一页 -->
        <nav v-if="prev || next" class="mt-10 flex flex-col gap-3 border-t border-zinc-800 pt-6 sm:flex-row" aria-label="翻页">
          <NuxtLink
            v-if="prev"
            :to="prev.path"
            class="flex min-h-11 flex-1 flex-col justify-center rounded-md border border-zinc-800 px-4 py-3 hover:bg-zinc-900"
          >
            <span class="text-xs text-zinc-500">上一页</span>
            <span class="text-sm text-zinc-200">← {{ prev.navTitle }}</span>
          </NuxtLink>
          <NuxtLink
            v-if="next"
            :to="next.path"
            class="flex min-h-11 flex-1 flex-col justify-center rounded-md border border-zinc-800 px-4 py-3 hover:bg-zinc-900 sm:items-end"
          >
            <span class="text-xs text-zinc-500">下一页</span>
            <span class="text-sm text-zinc-200">{{ next.navTitle }} →</span>
          </NuxtLink>
        </nav>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { docNav, docNeighbors, findDocPage, renderDocMarkdown } from '~/utils/docs'

const props = defineProps<{ path: string, heading?: string }>()

const route = useRoute()
const nav = docNav()
const mobileNavOpen = ref(false)

const current = computed(() => findDocPage(props.path))
// H1 用 heading（短标题），title 含 " · Boxli 文档" 后缀只适合 <title>
const heading = computed(() => props.heading ?? current.value?.heading ?? '文档')
const sections = computed(() => current.value?.sections ?? [])
// 用 computed 保持响应式（解构普通对象会丢失响应性）
const prev = computed(() => docNeighbors(props.path).prev)
const next = computed(() => docNeighbors(props.path).next)

function isActive(p: string) {
  return route.path === p
}

function renderSection(body: string) {
  return renderDocMarkdown(body)
}

// 侧边导航高亮依赖 route，SSR 与客户端一致；路由切换时收起手机目录
watch(() => route.path, () => {
  mobileNavOpen.value = false
})

useHead({
  title: current.value?.title ?? '文档 · Boxli Hub',
  meta: [{ name: 'description', content: current.value?.description ?? 'Boxli Hub 使用文档。' }],
})
</script>

<style scoped>
/* 文档正文排版：手机优先，不引入 typography 插件 */
.docs-prose :deep(h2) {
  margin-top: 2rem;
  margin-bottom: 0.75rem;
  font-size: 1.125rem;
  font-weight: 500;
  color: #e4e4e7;
  /* 锚点跳转时避开 sticky 头部（h-14 = 56px） */
  scroll-margin-top: 5rem;
}
.docs-prose :deep(h3) {
  margin-top: 1.5rem;
  margin-bottom: 0.5rem;
  font-size: 1rem;
  font-weight: 500;
  color: #e4e4e7;
}
.docs-prose :deep(p) {
  margin-bottom: 0.875rem;
  font-size: 1rem;
  line-height: 1.75;
  color: #a1a1aa;
}
.docs-prose :deep(strong) {
  font-weight: 500;
  color: #e4e4e7;
}
.docs-prose :deep(a) {
  color: #e4e4e7;
  text-decoration: underline;
  text-underline-offset: 2px;
  /* 行内链接保证 44px 触达高度 */
  display: inline-flex;
  align-items: center;
  min-height: 44px;
  word-break: break-all;
}
.docs-prose :deep(ul),
.docs-prose :deep(ol) {
  margin-bottom: 0.875rem;
  padding-left: 1.25rem;
  color: #a1a1aa;
}
.docs-prose :deep(ul) {
  list-style: disc;
}
.docs-prose :deep(ol) {
  list-style: decimal;
}
.docs-prose :deep(li) {
  margin-bottom: 0.375rem;
  font-size: 1rem;
  line-height: 1.75;
}
.docs-prose :deep(code) {
  /* 行内代码字号 16px，避免移动端过小 */
  font-size: 0.9375rem;
  background: #18181b;
  border: 1px solid #27272a;
  border-radius: 0.25rem;
  padding: 0.125rem 0.375rem;
  word-break: break-all;
}
.docs-prose :deep(pre) {
  margin-bottom: 1rem;
  padding: 0.875rem;
  background: #09090b;
  border: 1px solid #27272a;
  border-radius: 0.5rem;
  overflow-x: auto;
}
.docs-prose :deep(pre code) {
  background: transparent;
  border: 0;
  padding: 0;
  font-size: 0.875rem;
  line-height: 1.6;
  color: #d4d4d8;
  word-break: normal;
  white-space: pre;
}
/* 宽表格在窄屏内横向滚动，避免撑破页面 */
.docs-prose :deep(table) {
  display: block;
  width: 100%;
  margin-bottom: 1rem;
  overflow-x: auto;
  border-collapse: collapse;
  font-size: 0.9375rem;
}
.docs-prose :deep(th),
.docs-prose :deep(td) {
  border: 1px solid #27272a;
  padding: 0.5rem 0.75rem;
  text-align: left;
  color: #a1a1aa;
  white-space: nowrap;
}
.docs-prose :deep(th) {
  background: #18181b;
  color: #e4e4e7;
  font-weight: 500;
}
.docs-prose :deep(blockquote) {
  margin-bottom: 1rem;
  border-left: 2px solid #3f3f46;
  padding-left: 0.875rem;
  color: #a1a1aa;
}
.docs-prose :deep(blockquote p) {
  margin-bottom: 0.5rem;
}
.docs-prose :deep(hr) {
  margin: 2rem 0;
  border-color: #27272a;
}
</style>
