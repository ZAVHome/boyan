<template>
  <div class="min-h-screen bg-theme-bg text-theme-text font-sans antialiased">
    <router-view />
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useThemeStore } from '@/stores/theme'
import { useAuthStore } from '@/stores/auth'

const themeStore = useThemeStore()
const authStore = useAuthStore()

onMounted(async () => {
  themeStore.setTheme(themeStore.currentTheme)
  await authStore.checkAuth()
})
</script>
