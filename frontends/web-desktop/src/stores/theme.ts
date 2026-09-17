import { defineStore } from 'pinia'
import { ref } from 'vue'

export type ThemeMode = 'light' | 'dark' | 'oled' | 'sepia'

export const useThemeStore = defineStore('theme', () => {
  const saved = localStorage.getItem('boyan_theme') as ThemeMode | null
  const currentTheme = ref<ThemeMode>(saved || 'dark')

  function applyTheme(theme: ThemeMode) {
    currentTheme.value = theme
    localStorage.setItem('boyan_theme', theme)
    document.documentElement.setAttribute('data-theme', theme)
  }

  // Применяем при старте
  applyTheme(currentTheme.value)

  function setTheme(theme: ThemeMode) {
    applyTheme(theme)
  }

  return {
    currentTheme,
    setTheme
  }
})
