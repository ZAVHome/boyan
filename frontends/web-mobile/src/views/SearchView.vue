<template>
  <div class="min-h-screen bg-theme-bg pb-20">
    <TopHeader :title="$t('search.title')" />

    <main class="max-w-md mx-auto px-4 py-3 space-y-4">
      <!-- Search Bar -->
      <div class="relative flex items-center">
        <Search class="absolute left-3.5 w-4 h-4 text-theme-muted pointer-events-none" />
        <input
          v-model="query"
          @input="onSearchInput"
          type="text"
          :placeholder="$t('search.placeholder')"
          class="w-full pl-10 pr-10 py-2.5 rounded-xl bg-theme-card border border-theme text-xs text-theme-text placeholder:text-theme-muted focus:outline-none focus:border-primary-500"
          autofocus
        />
        <button
          v-if="query"
          @click="clearSearch"
          class="absolute right-3 p-1 text-theme-muted hover:text-theme-text rounded-full"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <!-- Search results -->
      <div v-if="query.trim()">
        <div class="flex items-center justify-between py-1 mb-2">
          <span class="text-xs text-theme-muted font-medium">
            {{ $t('search.results', { count: results.length }) }}
          </span>
          <Loader2 v-if="loading" class="w-3.5 h-3.5 text-primary-600 animate-spin" />
        </div>

        <div v-if="results.length > 0" class="space-y-2.5">
          <MobileBookCard
            v-for="book in results"
            :key="book.id"
            :book="book"
            @select="catalogStore.openBookSheet"
            @read="onReadBook"
          />
        </div>

        <div v-else-if="!loading" class="text-center py-12 text-theme-muted space-y-2">
          <p class="text-xs">{{ $t('common.empty') }}</p>
        </div>
      </div>

      <!-- Search History if no active query -->
      <div v-else-if="searchHistory.length > 0" class="space-y-2">
        <div class="flex items-center justify-between text-xs text-theme-muted">
          <span>{{ $t('search.history') }}</span>
          <button @click="clearHistory" class="text-primary-600 hover:underline">
            {{ $t('search.clear_history') }}
          </button>
        </div>

        <div class="flex flex-wrap gap-1.5">
          <button
            v-for="(item, idx) in searchHistory"
            :key="idx"
            @click="selectHistoryItem(item)"
            class="px-2.5 py-1 text-xs rounded-lg bg-theme-card border border-theme text-theme-text hover:bg-theme-bg transition-colors"
          >
            {{ item }}
          </button>
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
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Search, X, Loader2 } from 'lucide-vue-next'
import TopHeader from '@/components/common/TopHeader.vue'
import BottomNav from '@/components/common/BottomNav.vue'
import MobileBookCard from '@/components/catalog/MobileBookCard.vue'
import BookActionSheet from '@/components/catalog/BookActionSheet.vue'
import { api } from '@/api/client'
import { useCatalogStore } from '@/stores/catalog'
import type { Book, BookListResponse } from '@/api/types'

const router = useRouter()
const catalogStore = useCatalogStore()

const query = ref('')
const results = ref<Book[]>([])
const loading = ref(false)
const searchHistory = ref<string[]>([])

let debounceTimer: ReturnType<typeof setTimeout> | null = null

onMounted(() => {
  const saved = localStorage.getItem('boyan_search_history')
  if (saved) {
    try {
      searchHistory.value = JSON.parse(saved)
    } catch {
      searchHistory.value = []
    }
  }
})

function onSearchInput() {
  if (debounceTimer) clearTimeout(debounceTimer)
  if (!query.value.trim()) {
    results.value = []
    return
  }

  debounceTimer = setTimeout(async () => {
    loading.value = true
    try {
      const q = query.value.trim()
      const res = await api.get<BookListResponse>('/api/v1/books', { q, per_page: 30 })
      results.value = res.items || []

      // Save to history
      if (q && !searchHistory.value.includes(q)) {
        searchHistory.value = [q, ...searchHistory.value.slice(0, 9)]
        localStorage.setItem('boyan_search_history', JSON.stringify(searchHistory.value))
      }
    } catch (err) {
      console.error('Search failed:', err)
    } finally {
      loading.value = false
    }
  }, 350)
}

function selectHistoryItem(item: string) {
  query.value = item
  onSearchInput()
}

function clearSearch() {
  query.value = ''
  results.value = []
}

function clearHistory() {
  searchHistory.value = []
  localStorage.removeItem('boyan_search_history')
}

function onReadBook(book: Book) {
  const fmt = book.files?.[0]?.format.toLowerCase() || 'fb2'
  router.push(`/reader/${book.id}?format=${fmt}`)
}
</script>
