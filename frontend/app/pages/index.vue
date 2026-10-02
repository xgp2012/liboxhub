<template>
  <div class="mx-auto w-full max-w-6xl px-4">
    <!-- Hero：手机优先，窄屏居中、lg 左对齐 -->
    <section class="flex flex-col items-center gap-6 py-12 text-center lg:items-start lg:py-20 lg:text-left">
      <span class="rounded-full border border-zinc-800 bg-zinc-900/60 px-3 py-1 text-xs text-zinc-400">
        AGPL-3.0 · 自研 .boxli 镜像格式
      </span>

      <h1 class="max-w-2xl text-3xl font-semibold leading-tight tracking-tight sm:text-4xl lg:text-5xl">
        轻量级容器引擎
        <span class="block text-zinc-500">镜像索引，去中心化分发</span>
      </h1>

      <p class="max-w-xl text-base leading-relaxed text-zinc-400">
        Boxli 是 Go 编写的轻量容器引擎，镜像不强制存 GitHub。Hub 只收录元数据，
        镜像文件留在你自己的 GitHub / Gitee / OSS / IPFS 上。
      </p>

      <div class="flex w-full max-w-md flex-col gap-3 sm:flex-row lg:max-w-lg">
        <CopyButton class="flex-1" text="boxli pull alice/myapp:v1" />
      </div>

      <div class="flex flex-col gap-3 sm:flex-row">
        <NuxtLink
          to="/explore"
          class="flex min-h-11 items-center justify-center rounded-md bg-zinc-100 px-6 text-sm font-medium text-zinc-900 transition-colors hover:bg-white"
        >
          浏览镜像
        </NuxtLink>
        <NuxtLink
          to="/docs"
          class="flex min-h-11 items-center justify-center rounded-md border border-zinc-800 px-6 text-sm text-zinc-300 transition-colors hover:bg-zinc-900"
        >
          查看文档
        </NuxtLink>
      </div>
    </section>

    <!-- 快速开始 -->
    <section class="py-8">
      <h2 class="mb-4 text-lg font-medium">快速开始</h2>
      <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <div
          v-for="step in quickStart"
          :key="step.title"
          class="rounded-lg border border-zinc-800 bg-zinc-900/40 p-4"
        >
          <p class="text-sm font-medium text-zinc-200">{{ step.title }}</p>
          <code class="mt-2 block overflow-x-auto whitespace-nowrap font-mono text-xs text-zinc-400">{{ step.cmd }}</code>
        </div>
      </div>
    </section>

    <!-- 特性 -->
    <section class="py-8">
      <h2 class="mb-4 text-lg font-medium">为什么用 Boxli</h2>
      <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <div
          v-for="f in features"
          :key="f.title"
          class="rounded-lg border border-zinc-800 bg-zinc-900/40 p-4"
        >
          <span class="text-2xl" aria-hidden="true">{{ f.icon }}</span>
          <p class="mt-2 text-sm font-medium text-zinc-200">{{ f.title }}</p>
          <p class="mt-1 text-sm leading-relaxed text-zinc-500">{{ f.desc }}</p>
        </div>
      </div>
    </section>

    <!-- 热门镜像 -->
    <section class="py-8">
      <div class="mb-4 flex items-center justify-between gap-4">
        <h2 class="text-lg font-medium">热门镜像</h2>
        <NuxtLink to="/explore" class="-mr-2 flex min-h-11 min-w-11 items-center justify-end px-2 text-sm text-zinc-400 hover:text-zinc-100">
          全部 →
        </NuxtLink>
      </div>

      <div v-if="pending" class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <CardSkeleton v-for="i in 3" :key="i" />
      </div>

      <EmptyState
        v-else-if="error || !repos?.items?.length"
        icon="🔍"
        title="暂时拿不到镜像列表"
        description="后端服务可能未启动，请稍后重试。"
      >
        <NuxtLink
          to="/explore"
          class="flex min-h-11 items-center justify-center rounded-md border border-zinc-800 px-4 text-sm text-zinc-300 hover:bg-zinc-900"
        >
          去浏览页重试
        </NuxtLink>
      </EmptyState>

      <div v-else class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <RepoCard v-for="repo in repos.items" :key="repo.id" :repo="repo" />
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
const { data: repos, pending, error } = await useRepoList({ limit: 6 })

useHead({
  title: 'Boxli Hub · 轻量级容器引擎镜像索引',
  meta: [
    {
      name: 'description',
      content: 'Boxli 是 Go 编写的轻量级容器引擎。Hub 只收录社区镜像元数据，不存储镜像本体。',
    },
  ],
})

const quickStart = [
  { title: '1. 拉取镜像', cmd: 'boxli pull alice/myapp:v1' },
  { title: '2. 运行容器', cmd: 'boxli run alice/myapp:v1' },
  { title: '3. 查看本地镜像', cmd: 'boxli images' },
]

const features = [
  { icon: '🪶', title: '轻量', desc: 'Go 编写，单二进制，无守护进程依赖，启动迅速。' },
  { icon: '📦', title: '自研镜像格式', desc: '.boxli 格式专为精简设计，不兼容 Docker/OCI。' },
  { icon: '🌐', title: '多源下载', desc: '同一镜像可挂 GitHub、Gitee、OSS、IPFS 等多个源。' },
  { icon: '🔗', title: '去中心化', desc: 'Hub 只记录下载地址，镜像本体由用户自己托管。' },
]
</script>
