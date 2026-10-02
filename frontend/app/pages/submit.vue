<template>
  <div class="mx-auto w-full max-w-3xl px-4 py-8">
    <header class="mb-6">
      <h1 class="text-2xl font-semibold tracking-tight">提交镜像</h1>
      <p class="mt-1 text-sm leading-relaxed text-zinc-500">
        只提交镜像的元数据与下载地址，镜像文件本体仍放在你自己的存储上。
      </p>
    </header>

    <div
      v-if="created"
      class="mb-6 rounded-md border border-emerald-900/60 bg-emerald-950/30 px-4 py-3 text-sm text-emerald-300"
      role="status"
    >
      <p class="font-medium">提交成功！</p>
      <p class="mt-1 text-emerald-400/80">
        已收录 {{ created.namespace }}/{{ created.name }}，共 {{ created.tags.length }} 个标签。
      </p>
      <div class="mt-3 flex flex-col gap-2 sm:flex-row">
        <NuxtLink
          :to="`/explore/${created.namespace}/${created.name}`"
          class="flex min-h-11 items-center justify-center rounded-md bg-emerald-500/20 px-4 text-sm font-medium text-emerald-200 hover:bg-emerald-500/30"
        >
          查看镜像详情
        </NuxtLink>
        <button
          type="button"
          class="flex min-h-11 items-center justify-center rounded-md border border-emerald-900/60 px-4 text-sm text-emerald-300 hover:bg-emerald-950/40"
          @click="resetForm"
        >
          继续提交下一个
        </button>
      </div>
    </div>

    <RepoForm
      v-if="!created"
      :key="formKey"
      mode="create"
      :default-namespace="user?.username ?? ''"
      @success="onSuccess"
    />
  </div>
</template>

<script setup lang="ts">
import type { RepoDetail } from '~/types/api'

// 服务端渲染时拿不到 httpOnly Cookie 里的登录态，必须走客户端守卫
definePageMeta({ middleware: 'auth' })

const { user, ensure } = useAuth()
const created = ref<RepoDetail | null>(null)
// 重置表单：换 key 强制重建 RepoForm，比手动清空各字段更可靠
const formKey = ref(0)

onMounted(() => {
  // 预填命名空间需要用户名
  void ensure()
})

function onSuccess(repo: RepoDetail) {
  created.value = repo
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

function resetForm() {
  created.value = null
  formKey.value += 1
}

useHead({ title: '提交镜像 · Boxli Hub' })
</script>
