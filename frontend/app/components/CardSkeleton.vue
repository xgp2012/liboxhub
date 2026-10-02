<template>
  <div class="rounded-lg border border-zinc-800 bg-zinc-900/40 p-4" role="status" aria-live="polite">
    <div class="flex animate-pulse items-start gap-3">
      <div class="h-9 w-9 shrink-0 rounded-md bg-zinc-800" />
      <div class="flex-1 space-y-2">
        <div class="h-4 w-1/3 rounded bg-zinc-800" />
        <div class="h-3 w-full rounded bg-zinc-800/70" />
        <div class="h-3 w-2/3 rounded bg-zinc-800/70" />
      </div>
    </div>
    <span class="sr-only">加载中</span>
  </div>
</template>
