<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Rss,
  BookOpen,
  Sparkles,
  Code2,
  Copy,
  Check,
  ExternalLink,
  ChevronDown,
  X,
  Tablet,
  Smartphone,
  HelpCircle
} from 'lucide-vue-next'

const emit = defineEmits<{
  (e: 'close'): void
}>()

const { t } = useI18n()

const copiedField = ref<string | null>(null)
const selectedFeedLang = ref<string>('default')
const showGuide = ref(false)

const baseUrl = computed(() => {
  if (typeof window !== 'undefined' && window.location.origin) {
    return window.location.origin
  }
  return 'http://localhost:8080'
})

const opdsV1Url = computed(() => {
  let url = `${baseUrl.value}/opds/v1/feed.xml`
  if (selectedFeedLang.value === 'ru') {
    url += '?lang=ru'
  } else if (selectedFeedLang.value === 'en') {
    url += '?lang=en'
  }
  return url
})

const opdsV2Url = computed(() => {
  return `${baseUrl.value}/opds/v2/catalog.json`
})

const swaggerUrl = computed(() => {
  return `${baseUrl.value}/api/v1/docs`
})

function copyToClipboard(text: string, fieldId: string) {
  navigator.clipboard.writeText(text)
  copiedField.value = fieldId
  setTimeout(() => {
    if (copiedField.value === fieldId) {
      copiedField.value = null
    }
  }, 2000)
}
</script>

<template>
  <div
    class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-md overflow-y-auto animate-fade-in"
    @click.self="emit('close')"
  >
    <div
      class="relative w-full max-w-2xl rounded-2xl bg-bg-surface border border-border shadow-2xl p-6 sm:p-7 space-y-6 my-8 text-fg-primary max-h-[90vh] flex flex-col"
    >
      <!-- Шапка модального окна -->
      <div class="flex items-start justify-between gap-4 pb-4 border-b border-border shrink-0">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-xl bg-accent/15 text-accent flex items-center justify-center shrink-0 border border-accent/25">
            <Rss class="w-5 h-5" />
          </div>
          <div>
            <h3 class="text-lg font-bold tracking-tight text-fg-primary">
              {{ t('connect.modal_title') }}
            </h3>
            <p class="text-xs text-fg-secondary mt-0.5 leading-relaxed">
              {{ t('connect.modal_desc') }}
            </p>
          </div>
        </div>

        <button
          @click="emit('close')"
          class="p-1.5 rounded-lg text-fg-muted hover:text-fg-primary hover:bg-bg-hover transition-colors shrink-0"
          :title="t('common.close') || 'Close'"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Содержимое со скроллом -->
      <div class="space-y-4 overflow-y-auto pr-1 flex-1">
        <!-- КАРТОЧКА 1: OPDS v1.2 -->
        <div class="p-4 rounded-xl bg-bg-primary border border-border hover:border-accent/40 transition-colors space-y-3">
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
            <div class="flex items-center gap-2">
              <BookOpen class="w-4 h-4 text-accent shrink-0" />
              <span class="font-semibold text-sm text-fg-primary">
                {{ t('connect.opds_v1_title') }}
              </span>
            </div>
            <span class="inline-flex items-center px-2 py-0.5 rounded-md text-[10px] font-semibold bg-accent/15 text-accent border border-accent/25 w-fit">
              {{ t('connect.opds_v1_badge') }}
            </span>
          </div>

          <p class="text-xs text-fg-secondary leading-relaxed">
            {{ t('connect.opds_v1_desc') }}
          </p>

          <!-- Выбор языка ленты -->
          <div class="flex items-center gap-2 text-xs pt-1">
            <span class="text-fg-muted text-[11px]">{{ t('connect.lang_feed_label') }}</span>
            <div class="inline-flex rounded-lg bg-bg-surface p-0.5 border border-border text-[11px]">
              <button
                @click="selectedFeedLang = 'default'"
                :class="selectedFeedLang === 'default' ? 'bg-accent text-white font-medium' : 'text-fg-secondary hover:text-fg-primary'"
                class="px-2 py-0.5 rounded transition-colors"
              >
                {{ t('connect.lang_feed_default') }}
              </button>
              <button
                @click="selectedFeedLang = 'ru'"
                :class="selectedFeedLang === 'ru' ? 'bg-accent text-white font-medium' : 'text-fg-secondary hover:text-fg-primary'"
                class="px-2 py-0.5 rounded transition-colors"
              >
                RU
              </button>
              <button
                @click="selectedFeedLang = 'en'"
                :class="selectedFeedLang === 'en' ? 'bg-accent text-white font-medium' : 'text-fg-secondary hover:text-fg-primary'"
                class="px-2 py-0.5 rounded transition-colors"
              >
                EN
              </button>
            </div>
          </div>

          <!-- URL и кнопка копирования -->
          <div class="relative flex items-center bg-bg-surface border border-border rounded-xl p-2.5 font-mono text-xs text-fg-primary">
            <span class="pr-20 truncate select-all">{{ opdsV1Url }}</span>
            <button
              @click="copyToClipboard(opdsV1Url, 'opds_v1')"
              type="button"
              class="absolute right-2 px-2.5 py-1 rounded-lg bg-bg-hover hover:bg-accent hover:text-white text-fg-secondary transition-all flex items-center gap-1 text-[11px] font-medium"
            >
              <Check v-if="copiedField === 'opds_v1'" class="w-3.5 h-3.5 text-emerald-400" />
              <Copy v-else class="w-3.5 h-3.5" />
              <span>{{ copiedField === 'opds_v1' ? t('connect.copied') : t('connect.copy_btn') }}</span>
            </button>
          </div>
        </div>

        <!-- КАРТОЧКА 2: OPDS v2.0 -->
        <div class="p-4 rounded-xl bg-bg-primary border border-border hover:border-border/80 transition-colors space-y-3">
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
            <div class="flex items-center gap-2">
              <Sparkles class="w-4 h-4 text-accent shrink-0" />
              <span class="font-semibold text-sm text-fg-primary">
                {{ t('connect.opds_v2_title') }}
              </span>
            </div>
            <span class="inline-flex items-center px-2 py-0.5 rounded-md text-[10px] font-semibold bg-bg-surface text-fg-secondary border border-border w-fit">
              {{ t('connect.opds_v2_badge') }}
            </span>
          </div>

          <p class="text-xs text-fg-secondary leading-relaxed">
            {{ t('connect.opds_v2_desc') }}
          </p>

          <!-- URL и кнопка копирования -->
          <div class="relative flex items-center bg-bg-surface border border-border rounded-xl p-2.5 font-mono text-xs text-fg-primary">
            <span class="pr-20 truncate select-all">{{ opdsV2Url }}</span>
            <button
              @click="copyToClipboard(opdsV2Url, 'opds_v2')"
              type="button"
              class="absolute right-2 px-2.5 py-1 rounded-lg bg-bg-hover hover:bg-accent hover:text-white text-fg-secondary transition-all flex items-center gap-1 text-[11px] font-medium"
            >
              <Check v-if="copiedField === 'opds_v2'" class="w-3.5 h-3.5 text-emerald-400" />
              <Copy v-else class="w-3.5 h-3.5" />
              <span>{{ copiedField === 'opds_v2' ? t('connect.copied') : t('connect.copy_btn') }}</span>
            </button>
          </div>
        </div>

        <!-- КАРТОЧКА 3: REST API & Swagger -->
        <div class="p-4 rounded-xl bg-bg-primary border border-border transition-colors space-y-3">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <Code2 class="w-4 h-4 text-accent shrink-0" />
              <span class="font-semibold text-sm text-fg-primary">
                {{ t('connect.api_title') }}
              </span>
            </div>
            <a
              :href="swaggerUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-accent/15 text-accent hover:bg-accent/25 border border-accent/25 text-xs font-semibold transition-colors"
            >
              <span>{{ t('connect.open_swagger') }}</span>
              <ExternalLink class="w-3 h-3" />
            </a>
          </div>

          <p class="text-xs text-fg-secondary leading-relaxed">
            {{ t('connect.api_desc') }}
          </p>
        </div>

        <!-- КАРТОЧКА 4: Памятка по настройке популярных читалок -->
        <div class="rounded-xl border border-border overflow-hidden bg-bg-primary">
          <button
            @click="showGuide = !showGuide"
            type="button"
            class="w-full p-4 flex items-center justify-between text-left hover:bg-bg-hover/40 transition-colors"
          >
            <div class="flex items-center gap-2">
              <HelpCircle class="w-4 h-4 text-accent shrink-0" />
              <span class="font-semibold text-xs text-fg-primary">
                {{ t('connect.guide_title') }}
              </span>
            </div>
            <ChevronDown
              class="w-4 h-4 text-fg-muted transition-transform duration-200"
              :class="{ 'rotate-180': showGuide }"
            />
          </button>

          <div v-if="showGuide" class="p-4 pt-1 border-t border-border/60 space-y-3 text-xs leading-relaxed">
            <div class="flex gap-2.5 items-start">
              <Tablet class="w-4 h-4 text-fg-muted shrink-0 mt-0.5" />
              <div>
                <span class="font-semibold text-fg-primary">{{ t('connect.guide_pocketbook_title') }}:</span>
                <p class="text-fg-secondary mt-0.5">{{ t('connect.guide_pocketbook_desc') }}</p>
              </div>
            </div>

            <div class="flex gap-2.5 items-start">
              <Tablet class="w-4 h-4 text-fg-muted shrink-0 mt-0.5" />
              <div>
                <span class="font-semibold text-fg-primary">{{ t('connect.guide_koreader_title') }}:</span>
                <p class="text-fg-secondary mt-0.5">{{ t('connect.guide_koreader_desc') }}</p>
              </div>
            </div>

            <div class="flex gap-2.5 items-start">
              <Smartphone class="w-4 h-4 text-fg-muted shrink-0 mt-0.5" />
              <div>
                <span class="font-semibold text-fg-primary">{{ t('connect.guide_moonreader_title') }}:</span>
                <p class="text-fg-secondary mt-0.5">{{ t('connect.guide_moonreader_desc') }}</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
