<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useCatalogStore } from '@/stores/catalog'
import { useI18n } from 'vue-i18n'
import { api } from '@/api/client'
import type { Series, SeriesWithCount, SeriesListResponse, SeriesBooksResponse, Book, LetterCount } from '@/api/types'
import BookCard from '@/components/catalog/BookCard.vue'
import BookDetailModal from '@/components/catalog/BookDetailModal.vue'
import {
  Layers,
  Search,
  ArrowLeft,
  ChevronLeft,
  ChevronRight,
  Loader2,
  BookX,
  ExternalLink,
  X
} from 'lucide-vue-next'

const { t } = useI18n()
const router = useRouter()
const catalogStore = useCatalogStore()

const seriesList = ref<SeriesWithCount[]>([])
const alphabet = ref<LetterCount[]>([])
const total = ref(0)
const page = ref(1)
const perPage = ref(36)
const totalPages = ref(1)
const activeLetter = ref('')
const searchQuery = ref('')
const loading = ref(false)

// Просмотр книг выбранной серии
const selectedSeries = ref<Series | null>(null)
const seriesBooks = ref<Book[]>([])
const booksLoading = ref(false)
const selectedBook = ref<Book | null>(null)

let searchDebounceTimer: any = null

onMounted(() => {
  fetchSeries()
})

async function fetchSeries(resetPage = false) {
  if (resetPage) {
    page.value = 1
  }
  loading.value = true
  try {
    const res = await api.get<SeriesListResponse>('/api/v1/series', {
      letter: activeLetter.value,
      q: searchQuery.value,
      page: page.value,
      per_page: perPage.value
    })
    seriesList.value = res.items || []
    if (res.letters && res.letters.length > 0) {
      alphabet.value = res.letters
    }
    total.value = res.total
    totalPages.value = res.total_pages || 1
  } catch (err) {
    console.error('Failed to load series:', err)
    seriesList.value = []
  } finally {
    loading.value = false
  }
}

function selectLetter(letter: string) {
  if (activeLetter.value === letter) {
    activeLetter.value = ''
  } else {
    activeLetter.value = letter
    searchQuery.value = ''
  }
  fetchSeries(true)
}

function onSearchInput() {
  clearTimeout(searchDebounceTimer)
  searchDebounceTimer = setTimeout(() => {
    if (searchQuery.value) {
      activeLetter.value = ''
    }
    fetchSeries(true)
  }, 300)
}

function clearSearch() {
  searchQuery.value = ''
  fetchSeries(true)
}

function setPage(p: number) {
  if (p >= 1 && p <= totalPages.value) {
    page.value = p
    fetchSeries()
  }
}

async function openSeries(series: SeriesWithCount) {
  selectedSeries.value = series
  booksLoading.value = true
  try {
    const res = await api.get<SeriesBooksResponse>(`/api/v1/series/${series.id}/books`)
    seriesBooks.value = res.items || []
  } catch (err) {
    console.error('Failed to load series books:', err)
    seriesBooks.value = []
  } finally {
    booksLoading.value = false
  }
}

function backToSeries() {
  selectedSeries.value = null
  seriesBooks.value = []
}

function openInCatalog(seriesName: string) {
  catalogStore.setSearch(seriesName)
  router.push({ name: 'catalog' })
}

function onBookClick(book: Book) {
  selectedBook.value = book
}

function onBookRead(book: Book) {
  router.push({ name: 'reader', params: { id: book.id } })
}
</script>

<template>
  <div class="space-y-6">
    <!-- Режим 1: Просмотр книг выбранной серии -->
    <div v-if="selectedSeries" class="space-y-6">
      <!-- Навигация назад и заголовок серии -->
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-4 border-b border-border">
        <div class="flex items-center gap-3">
          <button
            @click="backToSeries"
            class="p-2 rounded-xl bg-bg-surface border border-border hover:bg-bg-hover text-fg-primary transition-colors"
            :title="t('series.back_to_series')"
          >
            <ArrowLeft class="w-5 h-5" />
          </button>
          <div>
            <h1 class="text-xl font-bold text-fg-primary tracking-tight">
              {{ selectedSeries.name }}
            </h1>
            <p class="text-xs text-fg-muted mt-0.5">
              {{ t('series.series_books') }}: {{ seriesBooks.length }}
            </p>
          </div>
        </div>

        <button
          @click="openInCatalog(selectedSeries.name)"
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-bg-surface border border-border hover:bg-bg-hover text-xs font-semibold text-fg-primary transition-colors self-start sm:self-auto"
        >
          <ExternalLink class="w-3.5 h-3.5 text-accent" />
          <span>{{ t('series.view_in_catalog') }}</span>
        </button>
      </div>

      <!-- Лоадер книг серии -->
      <div v-if="booksLoading" class="flex justify-center items-center py-20">
        <Loader2 class="w-8 h-8 text-accent animate-spin" />
      </div>

      <!-- Пустой список книг серии -->
      <div
        v-else-if="seriesBooks.length === 0"
        class="text-center py-16 rounded-2xl bg-bg-surface border border-border p-8"
      >
        <BookX class="w-12 h-12 text-fg-muted mx-auto mb-3" />
        <h3 class="text-base font-bold text-fg-primary">{{ t('catalog.no_books') }}</h3>
      </div>

      <!-- Сетка книг серии -->
      <div
        v-else
        class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 xl:grid-cols-6 gap-4"
      >
        <BookCard
          v-for="book in seriesBooks"
          :key="book.id"
          :book="book"
          @click="onBookClick"
          @read="onBookRead"
        />
      </div>
    </div>

    <!-- Режим 2: Каталог всех серий -->
    <div v-else class="space-y-6">
      <!-- Верхняя панель: Заголовок и поиск -->
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-2 border-b border-border">
        <div>
          <h1 class="text-xl font-bold text-fg-primary tracking-tight">
            {{ t('series.title') }}
          </h1>
          <p class="text-xs text-fg-muted mt-0.5">
            {{ t('series.total_series', { count: total }) }}
          </p>
        </div>

        <!-- Поисковая строка -->
        <div class="relative w-full sm:w-72">
          <Search class="w-4 h-4 text-fg-muted absolute left-3 top-1/2 -translate-y-1/2" />
          <input
            v-model="searchQuery"
            @input="onSearchInput"
            type="text"
            :placeholder="t('series.search_placeholder')"
            class="w-full pl-9 pr-8 py-1.5 bg-bg-surface border border-border rounded-xl text-xs text-fg-primary placeholder-fg-muted focus:outline-none focus:border-accent transition-colors"
          />
          <button
            v-if="searchQuery"
            @click="clearSearch"
            class="absolute right-2.5 top-1/2 -translate-y-1/2 text-fg-muted hover:text-fg-primary"
          >
            <X class="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      <!-- Алфавитный указатель -->
      <div class="flex flex-wrap items-center gap-1 bg-bg-surface p-2 rounded-2xl border border-border shadow-xs">
        <button
          @click="selectLetter('')"
          class="px-2.5 h-8 rounded-lg text-xs font-semibold transition-colors flex items-center justify-center"
          :class="!activeLetter ? 'bg-accent text-white shadow-sm' : 'text-fg-secondary hover:bg-bg-hover hover:text-fg-primary'"
        >
          {{ t('series.all_letters') }}
        </button>

        <button
          v-for="item in alphabet"
          :key="item.letter"
          @click="selectLetter(item.letter)"
          class="min-w-7 h-8 px-1.5 rounded-lg text-xs font-bold transition-colors flex items-center justify-center gap-0.5"
          :class="activeLetter === item.letter ? 'bg-accent text-white shadow-sm' : 'text-fg-secondary hover:bg-bg-hover hover:text-fg-primary'"
        >
          <span>{{ item.letter }}</span>
        </button>
      </div>

      <!-- Лоадер -->
      <div v-if="loading" class="flex justify-center items-center py-20">
        <Loader2 class="w-8 h-8 text-accent animate-spin" />
      </div>

      <!-- Пустое состояние -->
      <div
        v-else-if="seriesList.length === 0"
        class="text-center py-16 rounded-2xl bg-bg-surface border border-border p-8"
      >
        <Layers class="w-12 h-12 text-fg-muted mx-auto mb-3" />
        <h3 class="text-base font-bold text-fg-primary">{{ t('series.no_series') }}</h3>
      </div>

      <!-- Список карточек серий -->
      <div v-else class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
        <div
          v-for="series in seriesList"
          :key="series.id"
          @click="openSeries(series)"
          class="group flex items-center justify-between p-3.5 rounded-xl bg-bg-surface border border-border hover:border-accent/50 hover:bg-bg-hover/80 cursor-pointer transition-all shadow-xs"
        >
          <div class="flex items-center gap-3 min-w-0">
            <div class="w-9 h-9 rounded-xl bg-accent/10 border border-accent/20 flex items-center justify-center text-accent shrink-0 group-hover:scale-105 transition-transform">
              <Layers class="w-4 h-4" />
            </div>
            <div class="min-w-0">
              <h3 class="text-xs font-bold text-fg-primary group-hover:text-accent truncate transition-colors">
                {{ series.name }}
              </h3>
              <p class="text-[11px] text-fg-muted">
                {{ t('series.books_count', { count: series.book_count }) }}
              </p>
            </div>
          </div>
          <ChevronRight class="w-4 h-4 text-fg-muted group-hover:text-accent group-hover:translate-x-0.5 transition-all shrink-0 ml-2" />
        </div>
      </div>

      <!-- Пагинация -->
      <div v-if="totalPages > 1" class="flex items-center justify-center gap-2 pt-4">
        <button
          :disabled="page <= 1"
          @click="setPage(page - 1)"
          class="p-2 rounded-xl bg-bg-surface border border-border hover:bg-bg-hover disabled:opacity-40 disabled:pointer-events-none text-fg-primary transition-colors"
        >
          <ChevronLeft class="w-5 h-5" />
        </button>

        <span class="text-xs font-semibold text-fg-secondary px-3">
          {{ t('catalog.page', { current: page, total: totalPages }) }}
        </span>

        <button
          :disabled="page >= totalPages"
          @click="setPage(page + 1)"
          class="p-2 rounded-xl bg-bg-surface border border-border hover:bg-bg-hover disabled:opacity-40 disabled:pointer-events-none text-fg-primary transition-colors"
        >
          <ChevronRight class="w-5 h-5" />
        </button>
      </div>
    </div>

    <!-- Модальное окно книги при просмотре книг серии -->
    <BookDetailModal
      v-if="selectedBook"
      :book="selectedBook"
      @close="selectedBook = null"
    />
  </div>
</template>
