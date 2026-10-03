<template>
  <div class="mx-auto w-full max-w-md px-4 py-12">
    <header class="mb-6 text-center">
      <h1 class="text-2xl font-semibold tracking-tight">登录 Boxli Hub</h1>
      <p class="mt-2 text-sm leading-relaxed text-zinc-500">
        使用 GitHub 账号登录，即可提交与管理镜像元数据。
      </p>
    </header>

    <!-- 回调失败提示：后端 302 回 /login?error=xxx，此处翻译为用户可读文案 -->
    <div
      v-if="errorMessage"
      class="mb-4 rounded-md border border-red-900/60 bg-red-950/40 px-4 py-3 text-sm text-red-300"
      role="alert"
    >
      {{ errorMessage }}
    </div>

    <!-- 已登录：给出明确的下一步入口，而不是让用户面对一个无意义的登录按钮 -->
    <div v-if="isLoggedIn" class="rounded-lg border border-zinc-800 bg-zinc-900/40 p-6 text-center">
      <img
        v-if="user?.avatar_url"
        :src="user.avatar_url"
        :alt="`${user.username} 的头像`"
        class="mx-auto h-16 w-16 rounded-full border border-zinc-800"
        width="64"
        height="64"
      >
      <p class="mt-3 text-base font-medium">已登录为 {{ user?.username }}</p>
      <div class="mt-5 flex flex-col gap-3">
        <NuxtLink
          to="/dashboard"
          class="flex min-h-11 items-center justify-center rounded-md bg-zinc-100 px-4 text-sm font-medium text-zinc-900 hover:bg-white"
        >
          进入用户中心
        </NuxtLink>
        <NuxtLink
          to="/submit"
          class="flex min-h-11 items-center justify-center rounded-md border border-zinc-800 px-4 text-sm text-zinc-300 hover:bg-zinc-900"
        >
          提交新镜像
        </NuxtLink>
      </div>
    </div>

    <!-- 未登录：GitHub OAuth 入口 -->
    <div v-else class="rounded-lg border border-zinc-800 bg-zinc-900/40 p-6">
      <button
        type="button"
        class="flex min-h-12 w-full items-center justify-center gap-2 rounded-md bg-zinc-100 px-4 text-sm font-medium text-zinc-900 transition-colors hover:bg-white disabled:opacity-60"
        :disabled="loading || submitting"
        @click="onLogin"
      >
        <span aria-hidden="true">🐙</span>
        {{ submitting ? '正在跳转…' : '使用 GitHub 登录' }}
      </button>

      <p class="mt-4 text-xs leading-relaxed text-zinc-500">
        登录后会在你的浏览器写入一个
        <span class="text-zinc-400">HttpOnly Cookie</span>
        作为会话凭证；本站不存储你的 GitHub 密码，也无法读取该 Cookie。
      </p>

      <p v-if="loginUnavailable" class="mt-3 text-xs leading-relaxed text-amber-400/90">
        后端未配置 OAuth 凭证（或未开启本地模拟登录），当前无法登录。
        请检查 <code class="text-zinc-400">hub.toml</code> 里的
        <code class="text-zinc-400">[github] client_id / secret</code>。
      </p>
    </div>

    <p class="mt-6 text-center text-xs text-zinc-600">
      登录即表示同意以 AGPL-3.0 协议使用本站服务。
    </p>
  </div>
</template>

<script setup lang="ts">
const route = useRoute()
const { user, isLoggedIn, loading, ensure, login } = useAuth()

const submitting = ref(false)
const loginUnavailable = ref(false)

/** 后端返回的错误码 → 用户可读文案 */
const ERROR_MESSAGES: Record<string, string> = {
  invalid_state: '登录会话已失效或已被使用，请重新发起登录。',
  state_error: '登录状态校验失败，请重试。',
  access_denied: '你取消了 GitHub 授权。',
  missing_code: 'GitHub 未返回授权码，请重试。',
  exchange_failed: '换取访问令牌失败，请稍后重试。',
  github_failed: '获取 GitHub 账号信息失败，请稍后重试。',
  identity_taken: '该用户名已被其他账号占用，请联系管理员。',
  user_failed: '创建账号失败，请稍后重试。',
  issue_failed: '签发会话失败，请稍后重试。',
}

const errorMessage = computed(() => {
  const code = typeof route.query.error === 'string' ? route.query.error : ''
  if (!code) return ''
  return ERROR_MESSAGES[code] ?? '登录失败，请重试。'
})

/** 登录成功后的回跳目标（仅接受站内相对路径，与后端校验保持一致） */
const redirectTarget = computed(() => {
  const r = route.query.redirect
  if (typeof r !== 'string') return '/dashboard'
  if (!r.startsWith('/') || r.startsWith('//')) return '/dashboard'
  return r
})

onMounted(async () => {
  // 登录态只在客户端可判定（Cookie 不参与 SSR）
  await ensure()
  // 清理 URL 上的 error 参数，避免刷新时重复提示
  if (route.query.error) {
    const { error, ...rest } = route.query
    void error
    await navigateTo({ path: '/login', query: rest }, { replace: true })
  }
})

async function onLogin() {
  if (submitting.value) return
  submitting.value = true
  loginUnavailable.value = false
  try {
    await login(redirectTarget.value)
    // 成功后浏览器会跳转到 GitHub，此处不重置 submitting，
    // 避免跳转前按钮被再次点击。
  }
  catch {
    loginUnavailable.value = true
    submitting.value = false
  }
}

useHead({ title: '登录 · Boxli Hub' })
</script>
