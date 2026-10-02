<template>
  <footer class="mt-12 border-t border-zinc-800/80 bg-zinc-950">
    <!-- pb 叠加 safe-area，避免 iPhone 横条遮挡 -->
    <div class="mx-auto w-full max-w-6xl px-4 py-8 pb-[calc(2rem+env(safe-area-inset-bottom))]">
      <div class="flex flex-col gap-6 sm:flex-row sm:items-start sm:justify-between">
        <div class="flex flex-col gap-2">
          <div class="flex items-center gap-2 text-sm font-semibold">
            <span class="grid h-6 w-6 place-items-center rounded bg-zinc-100 text-xs font-bold text-zinc-900">B</span>
            Boxli Hub
          </div>
          <p class="max-w-xs text-sm leading-relaxed text-zinc-500">
            轻量级容器引擎 Boxli 的官网与镜像索引站。只收录元数据，不存储镜像本体。
          </p>
        </div>

        <nav class="grid grid-cols-2 gap-x-8 gap-y-2 text-sm sm:flex sm:gap-8" aria-label="页脚导航">
          <div class="flex flex-col gap-2">
            <span class="text-xs font-medium uppercase tracking-wider text-zinc-600">产品</span>
            <NuxtLink to="/explore" class="-ml-2 flex min-h-11 min-w-11 items-center px-2 text-zinc-400 hover:text-zinc-100 sm:min-h-0">浏览镜像</NuxtLink>
            <NuxtLink to="/docs" class="-ml-2 flex min-h-11 min-w-11 items-center px-2 text-zinc-400 hover:text-zinc-100 sm:min-h-0">文档</NuxtLink>
            <NuxtLink to="/submit" class="-ml-2 flex min-h-11 min-w-11 items-center px-2 text-zinc-400 hover:text-zinc-100 sm:min-h-0">提交镜像</NuxtLink>
            <NuxtLink to="/dashboard" class="-ml-2 flex min-h-11 min-w-11 items-center px-2 text-zinc-400 hover:text-zinc-100 sm:min-h-0">用户中心</NuxtLink>
          </div>
          <div class="flex flex-col gap-2">
            <span class="text-xs font-medium uppercase tracking-wider text-zinc-600">关于</span>
            <a
              href="https://github.com/LiStudioorg/boxli"
              target="_blank"
              rel="noopener noreferrer"
              class="-ml-2 flex min-h-11 min-w-11 items-center px-2 text-zinc-400 hover:text-zinc-100 sm:min-h-0"
            >GitHub</a>
            <NuxtLink to="/about" class="-ml-2 flex min-h-11 min-w-11 items-center px-2 text-zinc-400 hover:text-zinc-100 sm:min-h-0">项目背景</NuxtLink>
          </div>
        </nav>
      </div>

      <div class="mt-8 flex flex-col gap-2 border-t border-zinc-900 pt-6 text-xs text-zinc-600 sm:flex-row sm:items-center sm:justify-between">
        <p>© {{ year }} Boxli · AGPL-3.0 License</p>
        <p>去中心化镜像索引 · 不存储镜像文件</p>
      </div>
    </div>
  </footer>
</template>

<script setup lang="ts">
// 用 useState 固定年份，保证 SSR 与客户端一致
const year = useState('footer-year', () => new Date().getFullYear())
</script>
