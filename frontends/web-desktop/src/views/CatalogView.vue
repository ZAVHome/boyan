<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useCatalogStore, type SortCriterion } from '@/stores/catalog'
import { useI18n } from 'vue-i18n'
import { useRouter, useRoute } from 'vue-router'
import type { Book } from '@/api/types'
import BookCard from '@/components/catalog/BookCard.vue'
import BookDetailModal from '@/components/catalog/BookDetailModal.vue'
import {
  LayoutGrid,
  List as ListIcon,
  Loader2,
  BookX,
  BookOpen,
  ArrowDown,
  ArrowUp,
  X
} from 'lucide-vue-next'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const catalogStore = useCatalogStore()
const imgErrors = ref<Record<string, boolean>>({})
const sentinelRef = ref<HTMLElement | null>(null)
let observer: IntersectionObserver | null = null

function syncFiltersFromRoute() {
  const q = typeof route.query.q === 'string' ? route.query.q : ''
  const genre = typeof route.query.genre === 'string' ? route.query.genre : ''
  const publisher = typeof route.query.publisher === 'string' ? route.query.publisher : ''
  const year = typeof route.query.year === 'string' ? route.query.year : ''
  const language = typeof route.query.language === 'string' ? route.query.language : ''

  catalogStore.setFilters({ q, genre, publisher, year, language })
}

watch(
  () => route.query,
  () => {
    syncFiltersFromRoute()
  },
  { deep: true }
)

onMounted(() => {
  if (route.query.q || route.query.genre || route.query.publisher || route.query.year || route.query.language) {
    syncFiltersFromRoute()
  } else if (catalogStore.books.length === 0) {
    catalogStore.fetchBooks(true)
  }

  observer = new IntersectionObserver(
    (entries) => {
      if (
        entries[0]?.isIntersecting &&
        catalogStore.hasMore &&
        !catalogStore.loading &&
        !catalogStore.loadingMore
      ) {
        catalogStore.loadMore()
      }
    },
    { rootMargin: '300px' }
  )

  if (sentinelRef.value) {
    observer.observe(sentinelRef.value)
  }
})

onUnmounted(() => {
  if (observer) {
    observer.disconnect()
    observer = null
  }
})

watch(sentinelRef, (el) => {
  if (observer && el) {
    observer.observe(el)
  }
})

const pageTitle = computed(() => {
  if (catalogStore.genreFilter) {
    return t('catalog.filter_genre', { value: catalogStore.genreFilter })
  }
  if (catalogStore.publisherFilter) {
    return t('catalog.filter_publisher', { value: catalogStore.publisherFilter })
  }
  if (catalogStore.yearFilter) {
    return t('catalog.filter_year', { value: catalogStore.yearFilter })
  }
  if (catalogStore.languageFilter) {
    return t('catalog.filter_language', { value: catalogStore.languageFilter })
  }
  if (catalogStore.searchQuery) {
    return t('catalog.filter_search', { value: catalogStore.searchQuery })
  }
  return t('nav.catalog')
})

function clearFilter(key: 'q' | 'genre' | 'publisher' | 'year' | 'language' | 'all') {
  const q = { ...route.query }
  if (key === 'all') {
    delete q.q
    delete q.genre
    delete q.publisher
    delete q.year
    delete q.language
  } else {
    delete q[key]
  }
  router.push({ path: '/', query: q })
}

const dirLabel = computed(() => {
  if (catalogStore.sortBy === 'recent' || catalogStore.sortBy === 'year') {
    return catalogStore.sortDirection === 'desc'
      ? t('catalog.sort_tooltip_newest')
      : t('catalog.sort_tooltip_oldest')
  }
  return catalogStore.sortDirection === 'asc'
    ? t('catalog.sort_tooltip_az')
    : t('catalog.sort_tooltip_za')
})

function onSortChange(event: Event) {
  const target = event.target as HTMLSelectElement
  catalogStore.setSort(target.value as SortCriterion)
}

function onBookClick(book: Book) {
  catalogStore.openBookDetail(book.id)
}

function onBookRead(book: Book) {
  router.push({ name: 'reader', params: { id: book.id } })
}
</script>

<template>
  <div class="space-y-6">
    <!-- Верхняя панель витрины: общее количество, сортировка и переключатель вида -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-2 border-b border-border">
      <div>
        <h1 class="text-xl font-bold text-fg-primary tracking-tight">
          {{ pageTitle }}
        </h1>
        <p class="text-xs text-fg-muted mt-0.5">
          {{ t('catalog.total_books', { count: catalogStore.total }) }}
        </p>
      </div>

      <!-- Элементы управления: сортировка, направление и вид -->
      <div class="flex flex-wrap items-center gap-2 self-end sm:self-auto">
        <!-- Сортировка -->
        <div class="flex items-center gap-1.5 bg-bg-surface border border-border px-2.5 py-1.5 rounded-xl shadow-xs">
          <span class="text-xs text-fg-muted hidden lg:inline">{{ t('catalog.sort_by') }}:</span>
          <select
            :value="catalogStore.sortBy"
            @change="onSortChange"
            class="bg-transparent text-xs font-semibold text-fg-primary focus:outline-none cursor-pointer pr-1"
          >
            <option value="recent" class="bg-bg-surface text-fg-primary">{{ t('catalog.sort_recent') }}</option>
            <option value="year" class="bg-bg-surface text-fg-primary">{{ t('catalog.sort_year') }}</option>
            <option value="title" class="bg-bg-surface text-fg-primary">{{ t('catalog.sort_title') }}</option>
            <option value="author" class="bg-bg-surface text-fg-primary">{{ t('catalog.sort_author') }}</option>
            <option value="series" class="bg-bg-surface text-fg-primary">{{ t('catalog.sort_series') }}</option>
          </select>
        </div>

        <!-- Кнопка переключения направления -->
        <button
          @click="catalogStore.toggleDirection"
          class="flex items-center gap-1.5 bg-bg-surface border border-border px-2.5 py-1.5 rounded-xl hover:bg-bg-hover text-fg-primary text-xs font-semibold shadow-xs transition-colors"
          :title="dirLabel"
        >
          <component
            :is="catalogStore.sortDirection === 'desc' ? ArrowDown : ArrowUp"
            class="w-3.5 h-3.5 text-accent shrink-0"
          />
          <span class="text-[11px] text-fg-secondary hidden md:inline">{{ dirLabel }}</span>
        </button>

        <!-- Переключатель Grid / List -->
        <div class="flex items-center gap-1 bg-bg-surface border border-border p-1 rounded-xl shadow-xs">
          <button
            @click="catalogStore.viewMode = 'grid'"
            class="p-1.5 rounded-lg transition-colors"
            :class="catalogStore.viewMode === 'grid' ? 'bg-accent text-white shadow-sm' : 'text-fg-secondary hover:text-fg-primary'"
            :title="t('catalog.view_grid')"
          >
            <LayoutGrid class="w-4 h-4" />
          </button>
          <button
            @click="catalogStore.viewMode = 'list'"
            class="p-1.5 rounded-lg transition-colors"
            :class="catalogStore.viewMode === 'list' ? 'bg-accent text-white shadow-sm' : 'text-fg-secondary hover:text-fg-primary'"
            :title="t('catalog.view_list')"
          >
            <ListIcon class="w-4 h-4" />
          </button>
        </div>
      </div>
    </div>

    <!-- Панель активных фильтров -->
    <div v-if="catalogStore.hasActiveFilters" class="flex flex-wrap items-center gap-2 p-2.5 rounded-2xl bg-bg-surface border border-border shadow-xs">
      <span class="text-xs font-semibold text-fg-muted ml-1">Фильтры:</span>

      <span
        v-if="catalogStore.searchQuery"
        class="inline-flex items-center gap-1.5 px-3 py-1 rounded-xl bg-accent/10 border border-accent/20 text-accent text-xs font-medium"
      >
        <span>Поиск: "{{ catalogStore.searchQuery }}"</span>
        <button @click="clearFilter('q')" class="hover:text-accent-hover"><X class="w-3.5 h-3.5" /></button>
      </span>

      <span
        v-if="catalogStore.genreFilter"
        class="inline-flex items-center gap-1.5 px-3 py-1 rounded-xl bg-accent/10 border border-accent/20 text-accent text-xs font-medium"
      >
        <span>Жанр: {{ catalogStore.genreFilter }}</span>
        <button @click="clearFilter('genre')" class="hover:text-accent-hover"><X class="w-3.5 h-3.5" /></button>
      </span>

      <span
        v-if="catalogStore.publisherFilter"
        class="inline-flex items-center gap-1.5 px-3 py-1 rounded-xl bg-accent/10 border border-accent/20 text-accent text-xs font-medium"
      >
        <span>Издательство: {{ catalogStore.publisherFilter }}</span>
        <button @click="clearFilter('publisher')" class="hover:text-accent-hover"><X class="w-3.5 h-3.5" /></button>
      </span>

      <span
        v-if="catalogStore.yearFilter"
        class="inline-flex items-center gap-1.5 px-3 py-1 rounded-xl bg-accent/10 border border-accent/20 text-accent text-xs font-medium"
      >
        <span>Год: {{ catalogStore.yearFilter }}</span>
        <button @click="clearFilter('year')" class="hover:text-accent-hover"><X class="w-3.5 h-3.5" /></button>
      </span>

      <span
        v-if="catalogStore.languageFilter"
        class="inline-flex items-center gap-1.5 px-3 py-1 rounded-xl bg-accent/10 border border-accent/20 text-accent text-xs font-medium uppercase"
      >
        <span>Язык: {{ catalogStore.languageFilter }}</span>
        <button @click="clearFilter('language')" class="hover:text-accent-hover"><X class="w-3.5 h-3.5" /></button>
      </span>

      <button
        @click="clearFilter('all')"
        class="ml-auto text-xs text-fg-muted hover:text-red-500 font-medium px-2 py-1 transition-colors"
      >
        {{ t('catalog.clear_filter') }}
      </button>
    </div>

    <!-- Лоадер первичной загрузки -->
    <div v-if="catalogStore.loading" class="flex justify-center items-center py-24">
      <Loader2 class="w-8 h-8 text-accent animate-spin" />
    </div>

    <!-- Пустое состояние -->
    <div
      v-else-if="catalogStore.books.length === 0"
      class="text-center py-20 rounded-2xl bg-bg-surface border border-border p-8"
    >
      <BookX class="w-12 h-12 text-fg-muted mx-auto mb-3" />
      <h3 class="text-base font-bold text-fg-primary">{{ t('catalog.no_books') }}</h3>
      <p class="text-xs text-fg-muted mt-1">Попробуйте изменить поисковый запрос или загрузите новые книги.</p>
    </div>

    <!-- Группированное отображение книг -->
    <div v-else class="space-y-8">
      <section
        v-for="group in catalogStore.groupedBooks"
        :key="group.key"
        class="space-y-4"
      >
        <!-- Sticky плашка заголовка группы -->
        <div class="sticky top-0 z-10 py-2.5 -mx-3 px-3 sm:-mx-6 sm:px-6 bg-bg-primary/95 backdrop-blur-md flex items-center justify-between border-b border-border/70 transition-colors">
          <div class="flex items-center gap-2.5">
            <h2 class="text-sm font-bold text-fg-primary tracking-tight">
              {{ group.title }}
            </h2>
            <span class="px-2 py-0.5 rounded-full text-[11px] font-semibold bg-bg-surface border border-border text-fg-muted">
              {{ t('catalog.group_books_count', { count: group.books.length }) }}
            </span>
          </div>
        </div>

        <!-- Сетка книг (Grid mode) -->
        <div
          v-if="catalogStore.viewMode === 'grid'"
          class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 xl:grid-cols-6 gap-4"
        >
          <BookCard
            v-for="book in group.books"
            :key="book.id"
            :book="book"
            @click="onBookClick"
            @read="onBookRead"
          />
        </div>

        <!-- Список книг (List mode) -->
        <div v-else class="space-y-2">
          <div
            v-for="book in group.books"
            :key="book.id"
            @click="onBookClick(book)"
            class="p-4 rounded-xl bg-bg-surface border border-border hover:border-accent/40 cursor-pointer flex items-center justify-between gap-4 transition-all shadow-xs"
          >
            <div class="flex items-center gap-4 min-w-0">
              <div class="w-12 h-16 shrink-0 rounded-lg overflow-hidden bg-bg-secondary border border-border flex items-center justify-center">
                <img
                  v-if="!imgErrors[book.id]"
                  :src="`/covers/${book.id}`"
                  :alt="book.title"
                  class="w-full h-full object-cover"
                  @error="imgErrors[book.id] = true"
                />
                <BookOpen v-else class="w-5 h-5 text-accent/40" />
              </div>
              <div class="min-w-0">
                <h4 class="font-bold text-sm text-fg-primary truncate">{{ book.title }}</h4>
                <p class="text-xs text-fg-secondary truncate">
                  {{ book.authors?.map(a => a.name).join(', ') || t('catalog.author_unknown') }}
                </p>
                <span v-if="book.series && book.series.length > 0" class="text-[11px] text-accent">
                  {{ book.series[0].name }} #{{ book.series[0].index }}
                </span>
              </div>
            </div>

            <div class="flex items-center gap-2 shrink-0">
              <span
                v-for="f in book.files"
                :key="f.id"
                class="px-2 py-0.5 rounded text-[10px] font-bold uppercase bg-bg-secondary border border-border text-fg-secondary"
              >
                {{ f.format }}
              </span>
            </div>
          </div>
        </div>
      </section>

      <!-- Индикатор подгрузки и sentinel для бесконечной прокрутки -->
      <div ref="sentinelRef" class="py-8 flex flex-col items-center justify-center min-h-[70px]">
        <div v-if="catalogStore.loadingMore" class="flex items-center gap-2 text-xs font-semibold text-fg-muted">
          <Loader2 class="w-5 h-5 text-accent animate-spin" />
          <span>{{ t('catalog.loading_more') }}</span>
        </div>
        <button
          v-else-if="catalogStore.hasMore"
          @click="catalogStore.loadMore"
          class="px-5 py-2 rounded-xl bg-bg-surface border border-border hover:bg-bg-hover text-xs font-semibold text-fg-primary transition-colors shadow-xs"
        >
          {{ t('catalog.load_more') }}
        </button>
        <p v-else-if="catalogStore.books.length > 0" class="text-xs text-fg-muted">
          {{ t('catalog.all_books_loaded') }}
        </p>
      </div>
    </div>

    <!-- Модальное окно книги -->
    <BookDetailModal
      v-if="catalogStore.selectedBook"
      :book="catalogStore.selectedBook"
      @close="catalogStore.closeBookDetail"
    />
  </div>
</template>
