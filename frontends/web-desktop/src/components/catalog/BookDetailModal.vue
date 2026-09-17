<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import type { Book } from '@/api/types'
import {
  X,
  BookOpen,
  Download,
  Bookmark,
  CheckCircle2,
  BookMarked,
  Layers,
  User as UserIcon,
  Tag
} from 'lucide-vue-next'

const props = defineProps<{
  book: Book
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()

const currentProgress = ref<{ progress_percent: number } | null>(null)
const activeShelf = ref<string | null>(null)
const shelfLoading = ref(false)

const coverUrl = computed(() => {
  if (props.book.cover_cached) {
    return api.getCoverUrl(props.book.id)
  }
  return ''
})

const authorNames = computed(() => {
  return props.book.authors?.map(a => a.name).join(', ') || t('catalog.author_unknown')
})

onMounted(async () => {
  if (authStore.isAuthenticated) {
    try {
      const p = await api.get<{ progress_percent: number }>(`/api/v1/books/${props.book.id}/progress`)
      if (p && p.progress_percent > 0) {
        currentProgress.value = p
      }
    } catch {
      // Игнорируем
    }
  }
})

async function toggleShelf(type: 'reading' | 'finished' | 'favorite') {
  if (!authStore.isAuthenticated) {
    router.push({ name: 'login' })
    return
  }

  shelfLoading.value = true
  try {
    if (activeShelf.value === type) {
      await api.delete(`/api/v1/books/${props.book.id}/shelf/${type}`)
      activeShelf.value = null
    } else {
      await api.post(`/api/v1/books/${props.book.id}/shelf`, { shelf_type: type })
      activeShelf.value = type
    }
  } catch (err) {
    console.error('Failed to toggle shelf:', err)
  } finally {
    shelfLoading.value = false
  }
}

function startReading() {
  emit('close')
  router.push({ name: 'reader', params: { id: props.book.id } })
}
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm" @click.self="emit('close')">
    <div class="relative w-full max-w-3xl rounded-2xl bg-bg-surface border border-border shadow-2xl overflow-hidden max-h-[90vh] flex flex-col">
      <!-- Кнопка закрытия -->
      <button
        @click="emit('close')"
        class="absolute top-4 right-4 z-10 p-2 rounded-xl bg-bg-primary/80 border border-border text-fg-secondary hover:text-fg-primary transition-colors focus:outline-none"
      >
        <X class="w-5 h-5" />
      </button>

      <div class="overflow-y-auto p-6 md:p-8 space-y-6">
        <div class="flex flex-col sm:flex-row gap-6">
          <!-- Обложка -->
          <div class="w-full sm:w-48 shrink-0 aspect-[2/3] rounded-xl overflow-hidden bg-bg-secondary border border-border shadow-md">
            <img
              v-if="coverUrl"
              :src="coverUrl"
              :alt="book.title"
              class="w-full h-full object-cover"
            />
            <div v-else class="w-full h-full p-4 flex flex-col justify-center items-center text-center bg-gradient-to-br from-bg-secondary to-bg-primary">
              <BookOpen class="w-10 h-10 text-accent/50 mb-2" />
              <span class="text-xs font-semibold text-fg-secondary">{{ book.title }}</span>
            </div>
          </div>

          <!-- Метаданные -->
          <div class="flex-1 space-y-3">
            <h2 class="text-2xl font-bold text-fg-primary leading-tight">
              {{ book.title }}
            </h2>

            <div class="flex items-center gap-2 text-sm text-fg-secondary">
              <UserIcon class="w-4 h-4 text-accent" />
              <span class="font-medium text-fg-primary">{{ authorNames }}</span>
            </div>

            <!-- Серия -->
            <div v-if="book.series && book.series.length > 0" class="flex items-center gap-2 text-sm text-fg-secondary">
              <Layers class="w-4 h-4 text-accent" />
              <span>
                {{ book.series[0].name }}
                <span v-if="book.series[0].index" class="font-semibold text-accent ml-1">
                  #{{ book.series[0].index }}
                </span>
              </span>
            </div>

            <!-- Жанры -->
            <div v-if="book.genres && book.genres.length > 0" class="flex flex-wrap gap-1.5 pt-1">
              <span
                v-for="g in book.genres"
                :key="g.code"
                class="px-2 py-0.5 rounded-lg text-xs bg-bg-secondary text-fg-secondary border border-border"
              >
                {{ g.name_ru || g.name_en || g.code }}
              </span>
            </div>

            <!-- Прогресс чтения -->
            <div v-if="currentProgress" class="pt-2">
              <div class="flex items-center justify-between text-xs text-fg-muted mb-1">
                <span>{{ t('book.progress', { percent: currentProgress.progress_percent.toFixed(0) }) }}</span>
              </div>
              <div class="w-full h-1.5 rounded-full bg-bg-secondary overflow-hidden">
                <div class="h-full bg-accent transition-all duration-300" :style="{ width: `${currentProgress.progress_percent}%` }"></div>
              </div>
            </div>

            <!-- Кнопки действий -->
            <div class="pt-3 flex flex-wrap gap-2.5">
              <button
                @click="startReading"
                class="flex items-center gap-2 px-5 py-2.5 rounded-xl bg-accent text-white hover:bg-accent-hover font-semibold shadow-md shadow-accent/25 transition-all"
              >
                <BookOpen class="w-4 h-4" />
                <span>{{ t('book.read_online') }}</span>
              </button>

              <!-- Полки -->
              <div class="flex items-center gap-1 bg-bg-secondary border border-border rounded-xl p-1">
                <button
                  @click="toggleShelf('reading')"
                  class="p-2 rounded-lg transition-colors"
                  :class="activeShelf === 'reading' ? 'bg-accent text-white' : 'text-fg-secondary hover:text-fg-primary'"
                  :title="t('book.mark_reading')"
                >
                  <BookMarked class="w-4 h-4" />
                </button>
                <button
                  @click="toggleShelf('finished')"
                  class="p-2 rounded-lg transition-colors"
                  :class="activeShelf === 'finished' ? 'bg-emerald-500 text-white' : 'text-fg-secondary hover:text-fg-primary'"
                  :title="t('book.mark_finished')"
                >
                  <CheckCircle2 class="w-4 h-4" />
                </button>
                <button
                  @click="toggleShelf('favorite')"
                  class="p-2 rounded-lg transition-colors"
                  :class="activeShelf === 'favorite' ? 'bg-amber-500 text-white' : 'text-fg-secondary hover:text-fg-primary'"
                  :title="t('book.mark_favorite')"
                >
                  <Bookmark class="w-4 h-4" />
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- Доступные форматы для скачивания -->
        <div v-if="book.files && book.files.length > 0" class="pt-4 border-t border-border">
          <h4 class="text-xs font-semibold uppercase tracking-wider text-fg-muted mb-2.5">
            {{ t('book.download') }}
          </h4>
          <div class="flex flex-wrap gap-2">
            <a
              v-for="f in book.files"
              :key="f.id"
              :href="api.getDownloadUrl(book.id, f.format)"
              target="_blank"
              download
              class="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-bg-secondary border border-border hover:border-accent text-fg-primary hover:text-accent text-xs font-medium transition-colors"
            >
              <Download class="w-3.5 h-3.5" />
              <span class="uppercase font-bold">{{ f.format }}</span>
              <span class="text-fg-muted text-[11px]">({{ (f.file_size / 1024).toFixed(0) }} KB)</span>
            </a>
          </div>
        </div>

        <!-- Аннотация -->
        <div class="pt-4 border-t border-border">
          <h4 class="text-xs font-semibold uppercase tracking-wider text-fg-muted mb-2">
            {{ t('book.annotation') }}
          </h4>
          <div class="text-sm text-fg-secondary leading-relaxed whitespace-pre-line">
            {{ book.annotation || t('book.no_annotation') }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
