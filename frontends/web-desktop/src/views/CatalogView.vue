<script setup lang="ts">
import { onMounted } from 'vue'
import { useCatalogStore } from '@/stores/catalog'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import type { Book } from '@/api/types'
import BookCard from '@/components/catalog/BookCard.vue'
import BookDetailModal from '@/components/catalog/BookDetailModal.vue'
import {
  LayoutGrid,
  List as ListIcon,
  ChevronLeft,
  ChevronRight,
  Loader2,
  BookX
} from 'lucide-vue-next'

const { t } = useI18n()
const router = useRouter()
const catalogStore = useCatalogStore()

onMounted(() => {
  if (catalogStore.books.length === 0) {
    catalogStore.fetchBooks()
  }
})

function onBookClick(book: Book) {
  catalogStore.openBookDetail(book.id)
}

function onBookRead(book: Book) {
  router.push({ name: 'reader', params: { id: book.id } })
}
</script>

<template>
  <div class="space-y-6">
    <!-- Верхняя панель витрины: общее количество и переключатель вида -->
    <div class="flex items-center justify-between gap-4 pb-2 border-b border-border">
      <div>
        <h1 class="text-xl font-bold text-fg-primary tracking-tight">
          {{ catalogStore.searchQuery ? `Поиск: "${catalogStore.searchQuery}"` : t('nav.catalog') }}
        </h1>
        <p class="text-xs text-fg-muted mt-0.5">
          {{ t('catalog.total_books', { count: catalogStore.total }) }}
        </p>
      </div>

      <!-- Переключатель Grid / List -->
      <div class="flex items-center gap-1 bg-bg-surface border border-border p-1 rounded-xl">
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

    <!-- Лоадер -->
    <div v-if="catalogStore.loading" class="flex justify-center items-center py-20">
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

    <!-- Сетка книг -->
    <div
      v-else-if="catalogStore.viewMode === 'grid'"
      class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 xl:grid-cols-6 gap-4"
    >
      <BookCard
        v-for="book in catalogStore.books"
        :key="book.id"
        :book="book"
        @click="onBookClick"
        @read="onBookRead"
      />
    </div>

    <!-- Список книг -->
    <div v-else class="space-y-2">
      <div
        v-for="book in catalogStore.books"
        :key="book.id"
        @click="onBookClick(book)"
        class="p-4 rounded-xl bg-bg-surface border border-border hover:border-accent/40 cursor-pointer flex items-center justify-between gap-4 transition-all"
      >
        <div class="flex items-center gap-4 min-w-0">
          <div class="w-12 h-16 shrink-0 rounded-lg overflow-hidden bg-bg-secondary border border-border">
            <img
              v-if="book.cover_cached"
              :src="`/covers/${book.id}`"
              :alt="book.title"
              class="w-full h-full object-cover"
            />
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

    <!-- Пагинация -->
    <div v-if="catalogStore.totalPages > 1" class="flex items-center justify-center gap-2 pt-6">
      <button
        :disabled="catalogStore.page <= 1"
        @click="catalogStore.setPage(catalogStore.page - 1)"
        class="p-2 rounded-xl bg-bg-surface border border-border hover:bg-bg-hover disabled:opacity-40 disabled:pointer-events-none text-fg-primary transition-colors"
      >
        <ChevronLeft class="w-5 h-5" />
      </button>

      <span class="text-xs font-semibold text-fg-secondary px-3">
        {{ t('catalog.page', { current: catalogStore.page, total: catalogStore.totalPages }) }}
      </span>

      <button
        :disabled="catalogStore.page >= catalogStore.totalPages"
        @click="catalogStore.setPage(catalogStore.page + 1)"
        class="p-2 rounded-xl bg-bg-surface border border-border hover:bg-bg-hover disabled:opacity-40 disabled:pointer-events-none text-fg-primary transition-colors"
      >
        <ChevronRight class="w-5 h-5" />
      </button>
    </div>

    <!-- Модальное окно книги -->
    <BookDetailModal
      v-if="catalogStore.selectedBook"
      :book="catalogStore.selectedBook"
      @close="catalogStore.closeBookDetail"
    />
  </div>
</template>
