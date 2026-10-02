<template>
  <header class="sticky top-0 z-40 w-full border-b border-zinc-800/80 bg-zinc-950/90 backdrop-blur">
    <div class="mx-auto flex h-14 w-full max-w-6xl items-center gap-2 px-4">
      <!-- Logo：点击区 ≥44px -->
      <NuxtLink
        to="/"
        class="flex h-11 min-w-11 items-center gap-2 pr-1 text-base font-semibold tracking-tight"
        aria-label="Boxli Hub 首页"
      >
        <span class="grid h-7 w-7 place-items-center rounded-md bg-zinc-100 text-sm font-bold text-zinc-900">B</span>
        <span>Boxli</span>
        <span class="hidden text-xs font-normal text-zinc-500 sm:inline">Hub</span>
      </NuxtLink>

      <!-- 桌面导航：lg 以上显示 -->
      <nav class="ml-4 hidden items-center gap-1 lg:flex" aria-label="主导航">
        <NuxtLink
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          class="rounded-md px-3 py-2 text-sm text-zinc-400 transition-colors hover:bg-zinc-900 hover:text-zinc-100"
          active-class="!text-zinc-100"
        >
          {{ item.label }}
        </NuxtLink>
      </nav>

      <!-- 桌面搜索：md 以上显示 -->
      <form class="ml-auto hidden md:block" role="search" @submit.prevent="submitSearch">
        <label class="sr-only" for="header-search">搜索镜像</label>
        <div class="relative">
          <span class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-zinc-500" aria-hidden="true">⌕</span>
          <input
            id="header-search"
            v-model="keyword"
            type="search"
            placeholder="搜索镜像…"
            class="h-10 w-56 rounded-md border border-zinc-800 bg-zinc-900/60 pl-9 pr-3 text-base text-zinc-100 placeholder:text-zinc-500 focus:border-zinc-600 focus:outline-none lg:w-64"
          >
        </div>
      </form>

      <!-- 手机：搜索入口 + 汉堡，点击区 44px -->
      <div class="ml-auto flex items-center gap-1 md:hidden">
        <NuxtLink
          to="/search"
          class="grid h-11 w-11 place-items-center rounded-md text-zinc-300 hover:bg-zinc-900"
          aria-label="搜索"
        >
          <span class="text-lg" aria-hidden="true">⌕</span>
        </NuxtLink>
        <button
          type="button"
          class="grid h-11 w-11 place-items-center rounded-md text-zinc-300 hover:bg-zinc-900"
          :aria-expanded="menuOpen"
          aria-controls="mobile-menu"
          aria-label="打开菜单"
          @click="menuOpen = true"
        >
          <span class="flex flex-col gap-1" aria-hidden="true">
            <span class="block h-0.5 w-5 rounded bg-current" />
            <span class="block h-0.5 w-5 rounded bg-current" />
            <span class="block h-0.5 w-5 rounded bg-current" />
          </span>
        </button>
      </div>

      <!-- 桌面端：登录入口 / 用户菜单 -->
      <div class="hidden lg:ml-3 lg:flex lg:items-center lg:gap-2">
        <template v-if="isLoggedIn">
          <NuxtLink
            to="/dashboard"
            class="flex h-10 items-center gap-2 rounded-md border border-zinc-800 px-3 text-sm text-zinc-300 transition-colors hover:bg-zinc-900"
          >
            <img
              v-if="user?.avatar_url"
              :src="user.avatar_url"
              :alt="`${user.username} 的头像`"
              class="h-6 w-6 rounded-full border border-zinc-700"
              width="24"
              height="24"
            >
            <span v-else class="grid h-6 w-6 place-items-center rounded-full bg-zinc-800 text-xs">
              {{ user?.username?.charAt(0)?.toUpperCase() }}
            </span>
            <span class="max-w-24 truncate">{{ user?.username }}</span>
          </NuxtLink>
          <button
            type="button"
            class="flex h-10 items-center rounded-md px-3 text-sm text-zinc-500 transition-colors hover:bg-zinc-900 hover:text-zinc-200 disabled:opacity-50"
            :disabled="loggingOut"
            @click="onLogout"
          >
            {{ loggingOut ? '退出中…' : '退出' }}
          </button>
        </template>
        <NuxtLink
          v-else
          to="/login"
          class="inline-flex h-10 items-center rounded-md border border-zinc-800 px-4 text-sm text-zinc-300 transition-colors hover:bg-zinc-900"
        >
          登录
        </NuxtLink>
      </div>
    </div>

    <!-- 手机抽屉菜单 -->
    <Drawer
      v-model:open="menuOpen"
      title="菜单"
      placement="right"
      size="sm"
      :show-confirm="false"
      :show-cancel="false"
    >
      <nav class="flex flex-col gap-1" aria-label="移动端导航">
        <NuxtLink
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          class="flex min-h-11 items-center rounded-md px-3 text-base text-zinc-300 hover:bg-zinc-900"
          active-class="bg-zinc-900 !text-zinc-100"
          @click="menuOpen = false"
        >
          {{ item.label }}
        </NuxtLink>

        <!-- 已登录：用户信息 + 管理中心 + 退出（手机端不做 hover 依赖，全部可点） -->
        <template v-if="isLoggedIn">
          <div class="mt-3 flex items-center gap-3 border-t border-zinc-800 px-3 pt-4">
            <img
              v-if="user?.avatar_url"
              :src="user.avatar_url"
              :alt="`${user.username} 的头像`"
              class="h-9 w-9 rounded-full border border-zinc-700"
              width="36"
              height="36"
            >
            <span v-else class="grid h-9 w-9 place-items-center rounded-full bg-zinc-800 text-sm">
              {{ user?.username?.charAt(0)?.toUpperCase() }}
            </span>
            <span class="min-w-0 flex-1 truncate text-sm text-zinc-300">{{ user?.username }}</span>
          </div>
          <NuxtLink
            to="/dashboard"
            class="mt-2 flex min-h-11 items-center rounded-md px-3 text-base text-zinc-200 hover:bg-zinc-900"
            @click="menuOpen = false"
          >
            用户中心
          </NuxtLink>
          <NuxtLink
            to="/submit"
            class="flex min-h-11 items-center rounded-md px-3 text-base text-zinc-200 hover:bg-zinc-900"
            @click="menuOpen = false"
          >
            提交镜像
          </NuxtLink>
          <button
            type="button"
            class="flex min-h-11 items-center rounded-md px-3 text-left text-base text-red-400 hover:bg-zinc-900 disabled:opacity-50"
            :disabled="loggingOut"
            @click="onLogout"
          >
            {{ loggingOut ? '退出中…' : '退出登录' }}
          </button>
        </template>

        <template v-else>
          <NuxtLink
            to="/login"
            class="mt-2 flex min-h-11 items-center rounded-md border border-zinc-800 px-3 text-base text-zinc-200 hover:bg-zinc-900"
            @click="menuOpen = false"
          >
            登录
          </NuxtLink>
        </template>
      </nav>
    </Drawer>
  </header>
</template>

<script setup lang="ts">
import Drawer from 'fuxsto-design/drawer'

const menuOpen = ref(false)
const keyword = ref('')
const router = useRouter()
const { user, isLoggedIn, logout, ensure } = useAuth()

const loggingOut = ref(false)

const navItems = [
  { to: '/explore', label: '浏览镜像' },
  { to: '/docs', label: '文档' },
  { to: '/about', label: '关于' },
]

function submitSearch() {
  const q = keyword.value.trim()
  router.push(q ? { path: '/search', query: { q } } : '/search')
}

// 登录态存在 httpOnly Cookie 里，仅客户端可判定；挂载后拉一次，
// 未登录时静默保持 null（不报错、不跳转）。
onMounted(() => {
  void ensure()
})

async function onLogout() {
  if (loggingOut.value) return
  loggingOut.value = true
  try {
    await logout()
    menuOpen.value = false
    await navigateTo('/')
  }
  finally {
    loggingOut.value = false
  }
}

// 路由变化时收起抽屉，避免返回后菜单仍展开
watch(() => router.currentRoute.value.fullPath, () => {
  menuOpen.value = false
})
</script>
