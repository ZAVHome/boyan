<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useThemeStore, type ThemeMode } from '@/stores/theme'
import { useI18n } from 'vue-i18n'
import { Sun, Moon, Sparkles, Coffee } from 'lucide-vue-next'

const { t } = useI18n()
const themeStore = useThemeStore()
const isOpen = ref(false)
const dropdownRef = ref<HTMLElement | null>(null)

const themes: { id: ThemeMode; labelKey: string; icon: any }[] = [
  { id: 'light', labelKey: 'theme.light', icon: Sun },
  { id: 'dark', labelKey: 'theme.dark', icon: Moon },
  { id: 'oled', labelKey: 'theme.oled', icon: Sparkles },
  { id: 'sepia', labelKey: 'theme.sepia', icon: Coffee },
]

function selectTheme(theme: ThemeMode) {
  themeStore.setTheme(theme)
  isOpen.value = false
}

function handleClickOutside(e: MouseEvent) {
  if (dropdownRef.value && !dropdownRef.value.contains(e.target as Node)) {
    isOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<template>
  <div class="relative inline-block text-left" ref="dropdownRef">
    <button
      type="button"
      @click.stop="isOpen = !isOpen"
      class="p-2 rounded-lg bg-bg-surface border border-border hover:bg-bg-hover text-fg-primary transition-colors flex items-center gap-1.5 focus:outline-none focus:ring-2 focus:ring-accent"
      :title="t(`theme.${themeStore.currentTheme}`)"
    >
      <Sun v-if="themeStore.currentTheme === 'light'" class="w-4 h-4 text-amber-500" />
      <Moon v-else-if="themeStore.currentTheme === 'dark'" class="w-4 h-4 text-indigo-400" />
      <Sparkles v-else-if="themeStore.currentTheme === 'oled'" class="w-4 h-4 text-sky-400" />
      <Coffee v-else class="w-4 h-4 text-amber-700" />
    </button>

    <transition
      enter-active-class="transition duration-100 ease-out"
      enter-from-class="transform scale-95 opacity-0"
      enter-to-class="transform scale-100 opacity-100"
      leave-active-class="transition duration-75 ease-in"
      leave-from-class="transform scale-100 opacity-100"
      leave-to-class="transform scale-95 opacity-0"
    >
      <div
        v-if="isOpen"
        class="absolute right-0 top-full mt-2 w-40 rounded-xl bg-bg-surface border border-border shadow-xl py-1 z-50 overflow-hidden"
      >
        <button
          v-for="th in themes"
          :key="th.id"
          type="button"
          @click.stop="selectTheme(th.id)"
          class="w-full px-3 py-2 text-left text-sm flex items-center gap-2.5 hover:bg-bg-hover transition-colors cursor-pointer"
          :class="themeStore.currentTheme === th.id ? 'font-semibold text-accent' : 'text-fg-primary'"
        >
          <component :is="th.icon" class="w-4 h-4 shrink-0 pointer-events-none" />
          <span class="pointer-events-none">{{ t(th.labelKey) }}</span>
        </button>
      </div>
    </transition>
  </div>
</template>
