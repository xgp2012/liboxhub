<template>
  <div class="mx-auto w-full max-w-3xl px-4 py-8">
    <header class="mb-8">
      <h1 class="text-2xl font-semibold tracking-tight sm:text-3xl">关于 Boxli</h1>
      <p class="mt-2 text-base leading-relaxed text-zinc-400">
        Boxli 是一个用 Go 编写的轻量级容器引擎，自研 .boxli 镜像格式，不兼容 Docker/OCI。
      </p>
    </header>

    <div class="flex flex-col gap-8">
      <section>
        <h2 class="mb-3 text-lg font-medium">项目背景</h2>
        <div class="flex flex-col gap-3 text-base leading-relaxed text-zinc-400">
          <p>
            Boxli 的目标是做一个足够轻的容器引擎：单个 Go 二进制、启动快、依赖少，
            为嵌入式和边缘设备提供容器能力。
          </p>
          <p>
            为了让镜像分发不被单一平台绑定，Boxli Hub 只做一件事——
            <strong class="font-medium text-zinc-200">收录镜像元数据</strong>。
            镜像文件本体放在用户自己的 GitHub / Gitee / OSS / S3 / IPFS / BT 上，
            Hub 只记录「去哪下载」。
          </p>
        </div>
      </section>

      <section>
        <h2 class="mb-3 text-lg font-medium">为什么只存元数据</h2>
        <div class="grid gap-3 sm:grid-cols-2">
          <div
            v-for="item in reasons"
            :key="item.title"
            class="rounded-lg border border-zinc-800 bg-zinc-900/40 p-4"
          >
            <p class="text-sm font-medium text-zinc-200">{{ item.title }}</p>
            <p class="mt-1 text-sm leading-relaxed text-zinc-500">{{ item.desc }}</p>
          </div>
        </div>
      </section>

      <section>
        <h2 class="mb-3 text-lg font-medium">支持的下载源</h2>
        <div class="flex flex-wrap gap-2">
          <span
            v-for="(meta, key) in SOURCE_META"
            :key="key"
            class="flex items-center gap-1.5 rounded-md border border-zinc-800 bg-zinc-900/40 px-2.5 py-1.5 text-sm text-zinc-400"
          >
            <span aria-hidden="true">{{ meta.icon }}</span>{{ meta.label }}
          </span>
        </div>
      </section>

      <section>
        <h2 class="mb-3 text-lg font-medium">开源与协议</h2>
        <div class="flex flex-col gap-3 text-base leading-relaxed text-zinc-400">
          <p>
            Boxli 以 <strong class="font-medium text-zinc-200">AGPL-3.0</strong> 协议开源，
            源代码托管在 GitHub。
          </p>
          <a
            href="https://github.com/LiStudioorg/boxli"
            target="_blank"
            rel="noopener noreferrer"
            class="flex min-h-11 w-full items-center justify-center rounded-md border border-zinc-800 px-4 text-sm text-zinc-300 hover:bg-zinc-900 sm:w-auto sm:self-start"
          >
            在 GitHub 上查看源码 →
          </a>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
const reasons = [
  { title: '去中心化', desc: 'Hub 挂了，下载地址依然有效，镜像不受影响。' },
  { title: '多区域加速', desc: '国内用户走 Gitee / OSS / COS，海外走 GitHub / S3。' },
  { title: '抗单点故障', desc: '一个源不可用时，可切换到同标签的其它源。' },
  { title: '零带宽成本', desc: 'Hub 不承担镜像分发带宽，只提供索引。' },
  { title: '格式灵活', desc: '网盘直链、IPFS、BT 磁力都可以作为下载源。' },
  { title: '可审计', desc: '下载源可附 SHA256 digest，便于校验完整性。' },
]

useHead({
  title: '关于 · Boxli Hub',
  meta: [{ name: 'description', content: 'Boxli 项目背景、开源协议与设计理念。' }],
})
</script>
