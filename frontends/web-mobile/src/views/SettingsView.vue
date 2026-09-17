<template>
  <div class="min-h-screen bg-theme-bg pb-20">
    <TopHeader :title="$t('settings.title')" />

    <main class="max-w-md mx-auto px-4 py-3 space-y-4">
      <!-- E-Ink Special Mode Card -->
      <div
        class="p-4 rounded-xl border flex flex-col gap-2.5 transition-colors"
        :class="
          themeStore.isEInk
            ? 'bg-black text-white border-black'
            : 'bg-theme-card border-theme text-theme-text'
        "
      >
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <Sparkles class="w-5 h-5 text-amber-500" />
            <h3 class="text-sm font-bold">{{ $t('eink.title') }}</h3>
          </div>
          <EInkToggle />
        </div>
        <p class="text-xs leading-relaxed opacity-80">
          {{ $t('eink.description') }}
        </p>
      </div>

      <!-- Color Theme Switcher -->
      <div class="p-4 rounded-xl bg-theme-card border border-theme space-y-3">
        <h3 class="text-xs font-semibold text-theme-muted uppercase tracking-wider">
          {{ $t('settings.theme') }}
        </h3>
        <div class="grid grid-cols-2 gap-2">
          <button
            v-for="t in themeOptions"
            :key="t.id"
            @click="themeStore.setTheme(t.id)"
            :class="[
              'flex items-center gap-2.5 p-2.5 rounded-xl border text-xs font-medium transition-colors text-left',
              themeStore.currentTheme === t.id
                ? 'border-primary-600 bg-primary-500/10 text-primary-600 font-semibold ring-1 ring-primary-600'
                : 'border-theme text-theme-text hover:bg-theme-bg'
            ]"
          >
            <span
              class="w-4 h-4 rounded-full border border-theme flex-shrink-0"
              :style="{ background: t.color }"
            ></span>
            <span class="truncate">{{ t.label }}</span>
          </button>
        </div>
      </div>

      <!-- Language Selector (Extensible i18n) -->
      <div class="p-4 rounded-xl bg-theme-card border border-theme space-y-3">
        <div class="flex items-center justify-between">
          <h3 class="text-xs font-semibold text-theme-muted uppercase tracking-wider">
            {{ $t('settings.language') }}
          </h3>
          <Globe class="w-4 h-4 text-theme-muted" />
        </div>
        <div class="grid grid-cols-2 gap-2">
          <button
            v-for="lang in availableLanguages"
            :key="lang.code"
            @click="changeLang(lang.code)"
            :class="[
              'py-2 px-3 rounded-xl border text-xs font-medium transition-colors text-center',
              currentLang === lang.code
                ? 'border-primary-600 bg-primary-500/10 text-primary-600 font-semibold'
                : 'border-theme text-theme-text hover:bg-theme-bg'
            ]"
          >
            {{ lang.name }}
          </button>
        </div>
      </div>

      <!-- Offline Cache Management -->
      <div class="p-4 rounded-xl bg-theme-card border border-theme space-y-3">
        <div class="flex items-center justify-between">
          <h3 class="text-xs font-semibold text-theme-muted uppercase tracking-wider">
            {{ $t('settings.offline_cache_size') }}
          </h3>
          <HardDrive class="w-4 h-4 text-theme-muted" />
        </div>
        <div class="flex items-center justify-between text-xs">
          <span class="text-theme-text">{{ $t('offline.count_books', { count: offlineStore.count }) }}</span>
          <span class="font-mono text-theme-muted">{{ offlineStore.formattedTotalSize }}</span>
        </div>
        <router-link
          to="/offline"
          class="block text-center py-2 rounded-lg bg-theme-bg border border-theme text-xs font-medium text-theme-text hover:text-primary-600"
        >
          Управление офлайн-книгами
        </router-link>
      </div>

      <!-- Network Protocols (OPDS) -->
      <div class="p-4 rounded-xl bg-theme-card border border-theme space-y-3">
        <div class="flex items-center justify-between">
          <h3 class="text-xs font-semibold text-theme-muted uppercase tracking-wider">
            {{ $t('settings.protocols_title') }}
          </h3>
          <Rss class="w-4 h-4 text-theme-muted" />
        </div>
        <p class="text-xs text-theme-muted leading-relaxed">
          {{ $t('settings.protocols_desc') }}
        </p>

        <!-- OPDS v1.2 -->
        <div class="space-y-1">
          <div class="flex items-center justify-between text-xs">
            <span class="font-medium text-theme-text">OPDS v1.2 (XML)</span>
            <span class="text-[10px] text-theme-muted">PocketBook, KOReader</span>
          </div>
          <div class="flex items-center gap-1.5 p-2 rounded-lg bg-theme-bg border border-theme text-xs font-mono">
            <span class="truncate flex-1 select-all text-[11px]">{{ opdsV1Url }}</span>
            <button
              @click="copyUrl(opdsV1Url, 'v1')"
              class="px-2 py-1 rounded bg-theme-card border border-theme text-[10px] font-sans text-theme-text active:scale-95 shrink-0"
            >
              {{ copiedField === 'v1' ? $t('settings.copied') : $t('settings.copy') }}
            </button>
          </div>
        </div>

        <!-- OPDS v2.0 -->
        <div class="space-y-1">
          <div class="flex items-center justify-between text-xs">
            <span class="font-medium text-theme-text">OPDS v2.0 (JSON-LD)</span>
            <span class="text-[10px] text-theme-muted">Thorium, Cantook</span>
          </div>
          <div class="flex items-center gap-1.5 p-2 rounded-lg bg-theme-bg border border-theme text-xs font-mono">
            <span class="truncate flex-1 select-all text-[11px]">{{ opdsV2Url }}</span>
            <button
              @click="copyUrl(opdsV2Url, 'v2')"
              class="px-2 py-1 rounded bg-theme-card border border-theme text-[10px] font-sans text-theme-text active:scale-95 shrink-0"
            >
              {{ copiedField === 'v2' ? $t('settings.copied') : $t('settings.copy') }}
            </button>
          </div>
        </div>
      </div>

      <!-- PWA Install Prompt Button if available -->
      <div v-if="canInstallPwa" class="p-4 rounded-xl bg-primary-600 text-white space-y-2 shadow-md">
        <h3 class="text-xs font-bold uppercase tracking-wider">Прогрессивное веб-приложение</h3>
        <p class="text-xs opacity-90">Установите Боян на домашний экран для мгновенного доступа и чтения книг без интернета.</p>
        <button
          @click="installPwa"
          class="w-full py-2.5 rounded-lg bg-white text-primary-900 text-xs font-bold active:scale-98 shadow"
        >
          {{ $t('settings.pwa_install_btn') }}
        </button>
      </div>

      <!-- App Info -->
      <div class="text-center py-4 text-xs text-theme-muted space-y-1 flex flex-col items-center">
        <img src="/logo.svg" alt="Боян" class="w-12 h-12 rounded-full mb-1 shadow-sm" />
        <p class="font-semibold text-theme-text">Боян (Mobile PWA)</p>
        <p class="font-mono text-[10px]">Версия 1.0.0 (Stage 5)</p>
      </div>
    </main>

    <BottomNav />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Sparkles, Globe, HardDrive, Rss } from 'lucide-vue-next'
import TopHeader from '@/components/common/TopHeader.vue'
import BottomNav from '@/components/common/BottomNav.vue'
import EInkToggle from '@/components/common/EInkToggle.vue'
import { useThemeStore, type ThemeName } from '@/stores/theme'
import { useOfflineStore } from '@/stores/offline'
import { availableLanguages, setLocale } from '@/i18n'

const { t, locale } = useI18n()
const themeStore = useThemeStore()
const offlineStore = useOfflineStore()

const copiedField = ref<string | null>(null)
const baseUrl = computed(() => {
  if (typeof window !== 'undefined' && window.location.origin) {
    return window.location.origin
  }
  return 'http://localhost:8080'
})
const opdsV1Url = computed(() => `${baseUrl.value}/opds/v1/feed.xml`)
const opdsV2Url = computed(() => `${baseUrl.value}/opds/v2/catalog.json`)

function copyUrl(text: string, field: string) {
  navigator.clipboard.writeText(text)
  copiedField.value = field
  setTimeout(() => {
    if (copiedField.value === field) copiedField.value = null
  }, 2000)
}

const currentLang = computed(() => locale.value)

function changeLang(code: string) {
  setLocale(code)
}

const themeOptions = computed(() => [
  { id: 'light' as ThemeName, label: t('settings.theme_light'), color: '#ffffff' },
  { id: 'dark' as ThemeName, label: t('settings.theme_dark'), color: '#1e293b' },
  { id: 'oled' as ThemeName, label: t('settings.theme_oled'), color: '#000000' },
  { id: 'sepia' as ThemeName, label: t('settings.theme_sepia'), color: '#f4ecd8' },
  { id: 'eink' as ThemeName, label: t('settings.theme_eink'), color: '#f3f4f6' }
])

// PWA Install prompt capture
const deferredPrompt = ref<any>(null)
const canInstallPwa = computed(() => !!deferredPrompt.value)

onMounted(() => {
  window.addEventListener('beforeinstallprompt', (e) => {
    e.preventDefault()
    deferredPrompt.value = e
  })
})

async function installPwa() {
  if (deferredPrompt.value) {
    deferredPrompt.value.prompt()
    const choice = await deferredPrompt.value.userChoice
    if (choice.outcome === 'accepted') {
      deferredPrompt.value = null
    }
  }
}
</script>
