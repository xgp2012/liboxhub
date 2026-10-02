import tailwindcss from '@tailwindcss/vite'

export default defineNuxtConfig({
  compatibilityDate: '2025-01-01',
  devtools: { enabled: true },
  css: ['fuxsto-design/styles.css', '~/assets/css/main.css'],
  vite: {
    plugins: [tailwindcss()],
  },
  runtimeConfig: {
    public: {
      apiBase: '/api/v1',
    },
  },
  routeRules: {
    '/api/**': { proxy: 'http://127.0.0.1:3727/api/**' },
  },
  modules: ['@nuxt/eslint'],
})
