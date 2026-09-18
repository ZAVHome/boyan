<template>
  <div class="min-h-screen bg-theme-bg pb-20">
    <TopHeader />

    <main class="max-w-md mx-auto px-4 py-3 space-y-4">
      <!-- Search Input Bar (Touch friendly) -->
      <router-link
        to="/search"
        class="flex items-center gap-2.5 px-3.5 py-2.5 rounded-xl bg-theme-card border border-theme text-theme-muted text-xs shadow-sm"
      >
        <Search class="w-4 h-4 text-theme-muted" />
        <span>{{ $t('catalog.search_placeholder') }}</span>
      </router-link>

      <!-- Section Header with Count and Sort -->
      <div class="flex items-center justify-between gap-2 pt-1">
        <div>
          <h2 class="text-base font-bold text-theme-text">{{ $t('catalog.title') }}</h2>
          <p class="text-xs text-theme-muted">{{ $t('catalog.total_books', { count: catalogStore.total }) }}</p>
        </div>

        <div class="flex items-center gap-1.5 shrink-0">
          <select
            :value="catalogStore.sortBy"
            @change="onSortChange"
            class="bg-theme-card border border-theme rounded-xl px-2 py-1.5 text-xs font-semibold text-theme-text focus:outline-none shadow-xs"
          >
            <option value="recent">{{ $t('catalog.sort_recent') }}</option>
            <option value="year">{{ $t('catalog.sort_year') }}</option>
            <option value="title">{{ $t('catalog.sort_title') }}</option>
            <option value="author">{{ $t('catalog.sort_author') }}</option>
            <option value="series">{{ $t('catalog.sort_series') }}</option>
          </select>

          <button
            @click="catalogStore.toggleDirection"
            class="p-1.5 rounded-xl bg-theme-card border border-theme text-theme-text hover:bg-theme-bg active:scale-95 transition-all shadow-xs"
            :title="dirLabel"
            :aria-label="dirLabel"
          >
            <component
              :is="catalogStore.sortDirection === 'desc' ? ArrowDown : ArrowUp"
              class="w-4 h-4 text-primary-600 dark:text-primary-400"
            />
          </button>
        </div>
      </div>

      <!-- Loading skeleton -->
      <div v-if="catalogStore.loading && catalogStore.books.length === 0" class="space-y-3">
        <div
          v-for="i in 5"
          :key="i"
          class="h-24 rounded-xl bg-theme-card/60 animate-pulse border border-theme"
        ></div>
      </div>

      <!-- Error view -->
      <div
        v-else-if="catalogStore.error"
        class="p-4 rounded-xl bg-rose-500/10 border border-rose-500/20 text-center space-y-2"
      >
        <p class="text-xs text-rose-600 dark:text-rose-400">{{ catalogStore.error }}</p>
        <button
          @click="catalogStore.fetchBooks(true)"
          class="px-3 py-1.5 rounded-lg bg-theme-card border border-theme text-xs font-medium text-theme-text"
        >
          {{ $t('common.retry') }}
        </button>
      </div>

      <!-- Empty state -->
      <div
        v-else-if="catalogStore.books.length === 0"
        class="text-center py-12 space-y-2 text-theme-muted"
      >
        <BookOpen class="w-10 h-10 mx-auto opacity-30" />
        <p class="text-xs">{{ $t('catalog.no_books') }}</p>
      </div>

      <!-- Grouped books list -->
      <div v-else class="space-y-6">
        <section
          v-for="group in catalogStore.groupedBooks"
          :key="group.key"
          class="space-y-2.5"
        >
          <!-- Sticky header for each group -->
          <div
            class="sticky top-[48px] z-20 py-1.5 px-3 -mx-2 bg-theme-bg/95 backdrop-blur-md flex items-center justify-between border-b border-theme/60 transition-colors shadow-2xs"
          >
            <span class="text-xs font-bold text-theme-text tracking-tight truncate pr-2">
              {{ group.title }}
            </span>
            <span
              class="px-2 py-0.5 rounded-full text-[10px] font-semibold bg-theme-card border border-theme text-theme-muted shrink-0"
            >
              {{ $t('catalog.group_books_count', { count: group.books.length }) }}
            </span>
          </div>

          <!-- Cards list -->
          <MobileBookCard
            v-for="book in group.books"
            :key="book.id"
            :book="book"
            @select="catalogStore.openBookSheet"
            @read="onReadBook"
          />
        </section>

        <!-- Sentinel & Load more trigger -->
        <div ref="sentinelRef" class="py-6 flex flex-col items-center justify-center min-h-[60px]">
          <div v-if="catalogStore.loadingMore" class="flex items-center gap-2 text-xs font-semibold text-theme-muted">
            <Loader2 class="w-4 h-4 text-primary-600 dark:text-primary-400 animate-spin" />
            <span>{{ $t('catalog.loading_more') }}</span>
          </div>
          <button
            v-else-if="catalogStore.hasMore"
            @click="catalogStore.loadMore"
            class="w-full py-2.5 rounded-xl bg-theme-card border border-theme text-xs font-medium text-theme-text active:bg-theme-bg shadow-xs"
          >
            {{ $t('catalog.load_more') }}
          </button>
          <p v-else-if="catalogStore.books.length > 0" class="text-xs text-theme-muted">
            {{ $t('catalog.all_books_loaded') }}
          </p>
        </div>
      </div>
    </main>

    <!-- Book Action Sheet -->
    <BookActionSheet
      :book="catalogStore.selectedBook"
      @close="catalogStore.closeBookSheet"
    />

    <BottomNav />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Search, BookOpen, ArrowDown, ArrowUp, Loader2 } from 'lucide-vue-next'
import TopHeader from '@/components/common/TopHeader.vue'
import BottomNav from '@/components/common/BottomNav.vue'
import MobileBookCard from '@/components/catalog/MobileBookCard.vue'
import BookActionSheet from '@/components/catalog/BookActionSheet.vue'
import { useCatalogStore, type SortCriterion } from '@/stores/catalog'
import type { Book } from '@/api/types'

const { t } = useI18n()
const router = useRouter()
const catalogStore = useCatalogStore()
const sentinelRef = ref<HTMLElement | null>(null)
let observer: IntersectionObserver | null = null

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

onMounted(() => {
  if (catalogStore.books.length === 0) {
    catalogStore.fetchBooks(true)
  }
  catalogStore.fetchShelves()

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
    { rootMargin: '250px' }
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

function onSortChange(event: Event) {
  const target = event.target as HTMLSelectElement
  catalogStore.setSort(target.value as SortCriterion)
}

function onReadBook(book: Book) {
  const fmt = book.files?.[0]?.format.toLowerCase() || 'fb2'
  router.push(`/reader/${book.id}?format=${fmt}`)
}
</script>
