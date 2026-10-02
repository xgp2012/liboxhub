<template>
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>
</template>

<script setup lang="ts">
useHead({
  htmlAttrs: { lang: 'zh-CN' },
  // viewport 由 Nuxt 默认注入（width=device-width, initial-scale=1），此处补 theme-color
  meta: [
    { name: 'theme-color', content: '#09090b' },
  ],
})
</script>
