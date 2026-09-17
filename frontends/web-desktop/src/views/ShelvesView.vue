<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '@/api/client'
import type { Book } from '@/api/types'
import { useI18n } from 'vue-i18n'
import BookCard from '@/components/catalog/BookCard.vue'
import BookDetailModal from '@/components/catalog/BookDetailModal.vue'
import { BookOpen, CheckCircle2, Bookmark, Loader2, BookX } from 'lucide-vue-next'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const activeType = ref<'reading' | 'finished' | 'favorite'>((route.params.type as any) || 'reading')
const books = ref<Book[]>([])
const loading = ref(false)
const selectedBook = ref<Book | null>(null)

watch(() => route.params.type, (newType) => {
  if (newType) {
    activeType.value = newType as any
    loadShelfBooks()
  }
})

onMounted(() => {
  loadShelfBooks()
})

async function loadShelfBooks() {
  loading.value = true
  try {
    const res = await api.get<{ items: Book[]; total: number }>(`/api/v1/shelves/${activeType.value}`)
    books.value = res.items || []
  } catch (err) {
    console.error('Failed to load shelf books:', err)
    books.value = []
  } finally {
    loading.value = false
  }
}

function switchShelf(type: 'reading' | 'finished' | 'favorite') {
  activeType.value = type
  router.push(`/shelves/${type}`)
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
    <div class="flex items-center justify-between pb-2 border-b border-border">
      <div>
        <h1 class="text-xl font-bold text-fg-primary tracking-tight">
          {{ t('nav.shelves') }}
        </h1>
        <p class="text-xs text-fg-muted mt-0.5">Персональные списки чтения</p>
      </div>

      <!-- Вкладки полок -->
      <div class="flex items-center gap-1 bg-bg-surface border border-border p-1 rounded-xl">
        <button
          @click="switchShelf('reading')"
          class="px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-colors"
          :class="activeType === 'reading' ? 'bg-accent text-white shadow-sm' : 'text-fg-secondary hover:text-fg-primary'"
        >
          <BookOpen class="w-3.5 h-3.5" />
          <span>{{ t('nav.shelf_reading') }}</span>
        </button>

        <button
          @click="switchShelf('finished')"
          class="px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-colors"
          :class="activeType === 'finished' ? 'bg-emerald-500 text-white shadow-sm' : 'text-fg-secondary hover:text-fg-primary'"
        >
          <CheckCircle2 class="w-3.5 h-3.5" />
          <span>{{ t('nav.shelf_finished') }}</span>
        </button>

        <button
          @click="switchShelf('favorite')"
          class="px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-colors"
          :class="activeType === 'favorite' ? 'bg-amber-500 text-white shadow-sm' : 'text-fg-secondary hover:text-fg-primary'"
        >
          <Bookmark class="w-3.5 h-3.5" />
          <span>{{ t('nav.shelf_favorite') }}</span>
        </button>
      </div>
    </div>

    <!-- Загрузка -->
    <div v-if="loading" class="flex justify-center items-center py-20">
      <Loader2 class="w-8 h-8 text-accent animate-spin" />
    </div>

    <!-- Пустая полка -->
    <div v-else-if="books.length === 0" class="text-center py-20 rounded-2xl bg-bg-surface border border-border p-8">
      <BookX class="w-12 h-12 text-fg-muted mx-auto mb-3" />
      <h3 class="text-base font-bold text-fg-primary">Полка пуста</h3>
      <p class="text-xs text-fg-muted mt-1">Добавляйте книги на полку при просмотре карточки книги или во время чтения.</p>
    </div>

    <!-- Список книг -->
    <div v-else class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 xl:grid-cols-6 gap-4">
      <BookCard
        v-for="book in books"
        :key="book.id"
        :book="book"
        @click="onBookClick"
        @read="onBookRead"
      />
    </div>

    <BookDetailModal
      v-if="selectedBook"
      :book="selectedBook"
      @close="selectedBook = null"
    />
  </div>
</template>
