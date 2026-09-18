<template>
  <div v-if="book" class="fixed inset-0 z-50 flex items-end justify-center">
    <!-- Backdrop -->
    <div
      class="fixed inset-0 bg-black/60 backdrop-blur-sm transition-opacity"
      @click="close"
    ></div>

    <!-- Sheet Panel -->
    <div
      class="relative w-full max-w-lg max-h-[85vh] bg-theme-bg rounded-t-2xl border-t border-theme shadow-2xl flex flex-col z-10 overflow-hidden pb-safe animate-slide-up"
    >
      <!-- Drag handle -->
      <div class="w-full flex justify-center pt-3 pb-1 cursor-grab" @click="close">
        <div class="w-12 h-1.5 bg-theme-muted/30 rounded-full"></div>
      </div>

      <!-- Header with close button -->
      <div class="flex items-center justify-between px-4 pb-2 border-b border-theme">
        <span class="text-xs font-semibold text-theme-muted uppercase tracking-wider">
          {{ $t('book.details') }}
        </span>
        <button
          @click="close"
          class="p-1 rounded-full text-theme-muted hover:text-theme-text"
          :aria-label="$t('common.close')"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Scrollable content -->
      <div class="p-4 overflow-y-auto space-y-4 flex-1">
        <!-- Book Main Info -->
        <div class="flex gap-4 items-start">
          <div
            class="w-20 h-28 flex-shrink-0 rounded-lg overflow-hidden bg-theme-card border border-theme flex items-center justify-center relative shadow"
          >
            <img
              v-if="!imgError"
              :src="api.getCoverUrl(book.id)"
              :alt="book.title"
              class="w-full h-full object-cover"
              @error="imgError = true"
            />
            <BookIcon v-else class="w-8 h-8 text-theme-muted opacity-40" />
          </div>

          <div class="flex-1 min-w-0">
            <h3 class="text-base font-bold text-theme-text leading-snug">
              {{ book.title }}
            </h3>

            <!-- Авторы -->
            <div class="text-sm text-theme-muted mt-0.5">
              <span v-if="book.authors && book.authors.length > 0">
                <span v-for="(a, idx) in book.authors" :key="a.id || idx">
                  <button
                    type="button"
                    @click="searchBy(a.name)"
                    class="hover:underline hover:text-theme-text font-medium"
                  >
                    {{ a.name }}
                  </button>
                  <span v-if="idx < book.authors.length - 1">, </span>
                </span>
              </span>
              <span v-else>{{ authorNames }}</span>
            </div>

            <!-- Серия -->
            <div
              v-if="book.series && book.series.length > 0"
              class="mt-1"
            >
              <button
                type="button"
                @click="searchBy(book.series[0].name)"
                class="text-xs text-primary-600 dark:text-primary-400 hover:underline text-left"
              >
                {{ book.series[0].name }}
                <span v-if="book.series[0].index">#{{ book.series[0].index }}</span>
              </button>
            </div>

            <!-- Жанры -->
            <div v-if="book.genres && book.genres.length > 0" class="flex flex-wrap gap-1 mt-2">
              <button
                v-for="g in book.genres.slice(0, 4)"
                :key="g.code"
                type="button"
                @click="searchBy(g.name_ru || g.code)"
                class="px-1.5 py-0.5 text-[10px] rounded bg-theme-card border border-theme text-theme-muted hover:text-theme-text hover:border-primary-500 transition-colors"
              >
                {{ g.name_ru || g.code }}
              </button>
            </div>

            <!-- Дополнительно: издательство, год -->
            <div v-if="book.publisher || book.published_date" class="flex flex-wrap gap-2 text-[11px] text-theme-muted mt-2">
              <button
                v-if="book.publisher"
                type="button"
                @click="searchBy(book.publisher)"
                class="hover:underline hover:text-theme-text"
              >
                {{ book.publisher }}
              </button>
              <button
                v-if="book.published_date"
                type="button"
                @click="searchBy(book.published_date)"
                class="hover:underline hover:text-theme-text"
              >
                {{ book.published_date }}
              </button>
            </div>
          </div>
        </div>

        <!-- Annotation -->
        <div v-if="book.annotation" class="text-xs text-theme-text/80 leading-relaxed bg-theme-card/50 p-3 rounded-lg border border-theme line-clamp-6">
          {{ book.annotation }}
        </div>

        <!-- Shelves Actions -->
        <div>
          <div class="text-xs font-semibold text-theme-muted mb-2">
            {{ $t('shelves.add_to_shelf') }}
          </div>
          <div class="grid grid-cols-2 gap-2">
            <button
              v-for="shelf in shelfOptions"
              :key="shelf.type"
              @click="toggleShelf(shelf.type)"
              :class="[
                'flex items-center gap-2 px-3 py-2 text-xs rounded-lg border transition-colors',
                isOnShelf(shelf.type)
                  ? 'bg-primary-600 text-white border-primary-600'
                  : 'bg-theme-card text-theme-muted border-theme hover:text-theme-text'
              ]"
            >
              <component :is="shelf.icon" class="w-3.5 h-3.5" />
              <span>{{ shelf.label }}</span>
            </button>
          </div>
        </div>

        <!-- Offline Status Message if saved -->
        <div
          v-if="isOfflineSaved"
          class="flex items-center gap-2 p-2.5 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-700 dark:text-emerald-400 text-xs font-medium"
        >
          <CheckCircle2 class="w-4 h-4 flex-shrink-0" />
          <span>{{ $t('book.saved_offline') }}</span>
        </div>
      </div>

      <!-- Action Buttons Fixed at Sheet Bottom -->
      <div class="p-4 border-t border-theme bg-theme-bg space-y-2">
        <!-- Read Button -->
        <button
          @click="readBook"
          class="w-full flex items-center justify-center gap-2 py-3 rounded-xl bg-primary-600 text-white font-semibold text-sm hover:bg-primary-700 active:scale-[0.99] transition-transform shadow-md"
        >
          <BookOpen class="w-5 h-5" />
          <span>{{ isOfflineSaved ? $t('book.read_offline') : $t('book.read_now') }}</span>
        </button>

        <div class="grid grid-cols-2 gap-2">
          <!-- Offline Button -->
          <button
            v-if="!isOfflineSaved"
            @click="saveOffline"
            :disabled="isSaving"
            class="flex items-center justify-center gap-1.5 py-2.5 px-3 rounded-xl bg-theme-card border border-theme text-theme-text hover:bg-theme-bg text-xs font-medium transition-colors"
          >
            <Loader2 v-if="isSaving" class="w-4 h-4 animate-spin text-primary-600" />
            <DownloadCloud v-else class="w-4 h-4 text-theme-muted" />
            <span>{{ isSaving ? $t('common.loading') : $t('book.save_offline') }}</span>
          </button>

          <button
            v-else
            @click="removeOffline"
            class="flex items-center justify-center gap-1.5 py-2.5 px-3 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-600 dark:text-rose-400 hover:bg-rose-500/20 text-xs font-medium transition-colors"
          >
            <Trash2 class="w-4 h-4" />
            <span>{{ $t('book.remove_offline') }}</span>
          </button>

          <!-- Download File direct link -->
          <a
            :href="primaryDownloadUrl"
            download
            class="flex items-center justify-center gap-1.5 py-2.5 px-3 rounded-xl bg-theme-card border border-theme text-theme-text hover:bg-theme-bg text-xs font-medium transition-colors"
          >
            <Download class="w-4 h-4 text-theme-muted" />
            <span>{{ $t('common.download') }} ({{ primaryFormat.toUpperCase() }})</span>
          </a>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  X,
  Book as BookIcon,
  BookOpen,
  DownloadCloud,
  Download,
  Trash2,
  CheckCircle2,
  Loader2,
  Clock,
  Bookmark,
  CheckSquare,
  Heart
} from 'lucide-vue-next'
import { api } from '@/api/client'
import type { Book } from '@/api/types'
import { useOfflineStore } from '@/stores/offline'
import { useCatalogStore } from '@/stores/catalog'

const props = defineProps<{
  book: Book | null
}>()

const imgError = ref(false)

const emit = defineEmits<{
  (e: 'close'): void
}>()

const router = useRouter()
const { t } = useI18n()
const offlineStore = useOfflineStore()
const catalogStore = useCatalogStore()

const authorNames = computed(() => {
  if (!props.book?.authors || props.book.authors.length === 0) {
    return 'Неизвестный автор'
  }
  return props.book.authors.map((a) => a.name).join(', ')
})

const primaryFormat = computed(() => {
  if (!props.book?.files || props.book.files.length === 0) return 'fb2'
  // Prefer fb2 or epub
  const formats = props.book.files.map((f) => f.format.toLowerCase())
  if (formats.includes('fb2')) return 'fb2'
  if (formats.includes('epub')) return 'epub'
  return formats[0]
})

const primaryDownloadUrl = computed(() => {
  if (!props.book) return '#'
  return api.getDownloadUrl(props.book.id, primaryFormat.value)
})

const isOfflineSaved = computed(() => {
  if (!props.book) return false
  return offlineStore.isBookSaved(props.book.id)
})

const isSaving = computed(() => {
  if (!props.book) return false
  return offlineStore.savingBookId === props.book.id
})

const shelfOptions = computed(() => [
  { type: 'reading' as const, label: t('shelves.reading'), icon: Clock },
  { type: 'finished' as const, label: t('shelves.finished'), icon: CheckSquare },
  { type: 'favorite' as const, label: t('shelves.favorites'), icon: Heart }
])

function isOnShelf(type: 'reading' | 'finished' | 'favorite') {
  if (!props.book) return false
  const list = catalogStore.shelves[type]
  return Array.isArray(list) && list.some((item) => item.book_id === props.book?.id)
}

async function toggleShelf(type: 'reading' | 'finished' | 'favorite') {
  if (!props.book) return
  if (isOnShelf(type)) {
    await catalogStore.removeFromShelf(type, props.book.id)
  } else {
    await catalogStore.addToShelf(props.book.id, type)
  }
}

async function saveOffline() {
  if (!props.book) return
  await offlineStore.saveBook(props.book, primaryFormat.value)
}

async function removeOffline() {
  if (!props.book) return
  await offlineStore.removeBook(props.book.id)
}

function readBook() {
  if (!props.book) return
  emit('close')
  router.push(`/reader/${props.book.id}?format=${primaryFormat.value}`)
}

function searchBy(query: string) {
  emit('close')
  router.push({ name: 'search', query: { q: query } })
}

function close() {
  emit('close')
}
</script>

<style scoped>
@keyframes slide-up {
  from {
    transform: translateY(100%);
  }
  to {
    transform: translateY(0);
  }
}

.animate-slide-up {
  animation: slide-up 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

[data-theme="eink"] .animate-slide-up {
  animation: none !important;
}
</style>
