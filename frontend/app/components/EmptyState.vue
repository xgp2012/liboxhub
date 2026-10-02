<template>
  <div class="flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-zinc-800 px-6 py-12 text-center">
    <span class="text-3xl" aria-hidden="true">{{ icon }}</span>
    <p class="text-base font-medium text-zinc-300">{{ title }}</p>
    <p v-if="description" class="max-w-sm text-sm leading-relaxed text-zinc-500">{{ description }}</p>
    <div v-if="$slots.default" class="mt-2">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  title?: string
  description?: string
  icon?: string
}>(), {
  title: '暂无内容',
  description: '',
  icon: '📦',
})
</script>
