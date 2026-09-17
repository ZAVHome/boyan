<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { availableLanguages, setLocale } from '@/i18n'
import { useI18n } from 'vue-i18n'
import { Globe } from 'lucide-vue-next'

const { locale } = useI18n()
const isOpen = ref(false)
const dropdownRef = ref<HTMLElement | null>(null)

function selectLanguage(code: string) {
  setLocale(code)
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
      :title="locale.toUpperCase()"
    >
      <Globe class="w-4 h-4 text-fg-secondary pointer-events-none" />
      <span class="text-xs font-semibold uppercase pointer-events-none">{{ locale }}</span>
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
        class="absolute right-0 top-full mt-2 w-36 rounded-xl bg-bg-surface border border-border shadow-xl py-1 z-50 overflow-hidden"
      >
        <button
          v-for="lang in availableLanguages"
          :key="lang.code"
          type="button"
          @click.stop="selectLanguage(lang.code)"
          class="w-full px-3 py-2 text-left text-sm flex items-center justify-between hover:bg-bg-hover transition-colors cursor-pointer"
          :class="locale === lang.code ? 'font-semibold text-accent' : 'text-fg-primary'"
        >
          <span class="pointer-events-none">{{ lang.name }}</span>
          <span class="text-xs uppercase text-fg-muted pointer-events-none">{{ lang.code }}</span>
        </button>
      </div>
    </transition>
  </div>
</template>
