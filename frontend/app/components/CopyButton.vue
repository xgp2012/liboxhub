<template>
  <div class="flex items-stretch gap-2">
    <code
      class="min-w-0 flex-1 overflow-x-auto whitespace-nowrap rounded-md border border-zinc-800 bg-zinc-950 px-3 py-3 font-mono text-sm text-zinc-300"
    >{{ text }}</code>
    <button
      type="button"
      class="flex min-h-11 min-w-11 shrink-0 items-center justify-center gap-1.5 rounded-md border border-zinc-800 px-3 text-sm text-zinc-300 transition-colors hover:bg-zinc-900 active:bg-zinc-800"
      :aria-label="`复制：${text}`"
      @click="onCopy"
    >
      <span aria-hidden="true">{{ copied ? '✓' : '⧉' }}</span>
      <span class="hidden sm:inline">{{ copied ? '已复制' : '复制' }}</span>
    </button>
    <span class="sr-only" role="status" aria-live="polite">{{ copied ? '已复制到剪贴板' : '' }}</span>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{ text: string }>()

const copied = ref(false)
let timer: ReturnType<typeof setTimeout> | undefined

async function onCopy() {
  const ok = await copyText(props.text)
  if (!ok) return
  copied.value = true
  clearTimeout(timer)
  timer = setTimeout(() => {
    copied.value = false
  }, 2000)
}

onBeforeUnmount(() => clearTimeout(timer))
</script>
