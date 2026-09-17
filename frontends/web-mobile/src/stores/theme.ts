import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export type ThemeName = 'light' | 'dark' | 'oled' | 'sepia' | 'eink'

export const useThemeStore = defineStore('theme', () => {
  const currentTheme = ref<ThemeName>(
    (localStorage.getItem('boyan_theme') as ThemeName) || 'light'
  )

  const isEInk = computed(() => currentTheme.value === 'eink')

  function setTheme(theme: ThemeName) {
    currentTheme.value = theme
    localStorage.setItem('boyan_theme', theme)
    document.documentElement.setAttribute('data-theme', theme)
  }

  function toggleEInk() {
    if (isEInk.value) {
      const prev = (localStorage.getItem('boyan_prev_theme') as ThemeName) || 'light'
      setTheme(prev === 'eink' ? 'light' : prev)
    } else {
      localStorage.setItem('boyan_prev_theme', currentTheme.value)
      setTheme('eink')
    }
  }

  // Initialize theme on start
  setTheme(currentTheme.value)

  return {
    currentTheme,
    isEInk,
    setTheme,
    toggleEInk
  }
})
