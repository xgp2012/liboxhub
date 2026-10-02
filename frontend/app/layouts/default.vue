<template>
  <div class="flex min-h-[100dvh] flex-col overflow-x-hidden bg-zinc-950 text-zinc-100">
    <SiteHeader />

    <main class="flex-1">
      <slot />
    </main>

    <SiteFooter />
  </div>
</template>
