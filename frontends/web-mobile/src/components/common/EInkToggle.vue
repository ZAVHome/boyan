<template>
  <button
    @click="themeStore.toggleEInk()"
    :class="[
      'flex items-center gap-1.5 px-2.5 py-1 text-xs font-semibold rounded-full transition-colors border',
      themeStore.isEInk
        ? 'bg-black text-white border-black font-mono uppercase tracking-wider'
        : 'bg-theme-card text-theme-muted border-theme hover:text-theme-text'
    ]"
    :title="themeStore.isEInk ? $t('eink.disable') : $t('eink.enable')"
  >
    <span
      :class="[
        'w-2 h-2 rounded-full',
        themeStore.isEInk ? 'bg-white animate-none' : 'bg-emerald-500'
      ]"
    ></span>
    <span>E-INK</span>
  </button>
</template>

<script setup lang="ts">
import { useThemeStore } from '@/stores/theme'

const themeStore = useThemeStore()
</script>
