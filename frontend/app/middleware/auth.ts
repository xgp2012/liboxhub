// 登录守卫中间件（阶段 4）
//
// 用法：在页面里 declare `definePageMeta({ middleware: 'auth' })`。
//
// 关键点：会话存在 httpOnly Cookie 里，**SSR 阶段拿不到**（Cookie 不会随
// 服务端内部请求自动携带），因此登录态**只能在客户端**判定。
// 若在 SSR 就跳转，会把已登录用户也误判为未登录。
export default defineNuxtRouteMiddleware(async (to) => {
  if (import.meta.server) return

  const { ensure } = useAuth()
  const user = await ensure()
  if (user) return

  // 未登录：带上原目标，登录成功后原路返回
  return navigateTo({
    path: '/login',
    query: { redirect: to.fullPath },
  })
})
