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
      <div class="flex items-center justify-between pt-1">
        <div>
          <h2 class="text-base font-bold text-theme-text">{{ $t('catalog.title') }}</h2>
          <p class="text-xs text-theme-muted">{{ $t('catalog.total_books', { count: catalogStore.total }) }}</p>
        </div>

        <select
          v-model="sortOrder"
          @change="onSortChange"
          class="bg-theme-card border border-theme rounded-lg px-2 py-1 text-xs text-theme-text focus:outline-none"
        >
          <option value="created_at_desc">{{ $t('catalog.sort_recent') }}</option>
          <option value="title_asc">{{ $t('catalog.sort_title') }}</option>
        </select>
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
          @click="catalogStore.fetchBooks({ sort: sortOrder })"
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

      <!-- Books list -->
      <div v-else class="space-y-2.5">
        <MobileBookCard
          v-for="book in catalogStore.books"
          :key="book.id"
          :book="book"
          @select="catalogStore.openBookSheet"
          @read="onReadBook"
        />

        <!-- Load More button if more pages exist -->
        <div v-if="catalogStore.page < catalogStore.totalPages" class="pt-2 text-center">
          <button
            @click="loadMore"
            :disabled="catalogStore.loading"
            class="w-full py-2.5 rounded-xl bg-theme-card border border-theme text-xs font-medium text-theme-text active:bg-theme-bg"
          >
            <span v-if="catalogStore.loading">{{ $t('common.loading') }}</span>
            <span v-else>Загрузить ещё</span>
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
import { Search, BookOpen } from 'lucide-vue-next'
import TopHeader from '@/components/common/TopHeader.vue'
import BottomNav from '@/components/common/BottomNav.vue'
import MobileBookCard from '@/components/catalog/MobileBookCard.vue'
import BookActionSheet from '@/components/catalog/BookActionSheet.vue'
import { useCatalogStore } from '@/stores/catalog'
import type { Book } from '@/api/types'

const router = useRouter()
const catalogStore = useCatalogStore()
const sortOrder = ref('created_at_desc')

onMounted(() => {
  if (catalogStore.books.length === 0) {
    catalogStore.fetchBooks({ sort: sortOrder.value })
  }
  catalogStore.fetchShelves()
})

function onSortChange() {
  catalogStore.fetchBooks({ sort: sortOrder.value, page: 1 })
}

function loadMore() {
  if (catalogStore.page < catalogStore.totalPages) {
    catalogStore.fetchBooks({
      page: catalogStore.page + 1,
      sort: sortOrder.value,
      append: true
    })
  }
}

function onReadBook(book: Book) {
  const fmt = book.files?.[0]?.format.toLowerCase() || 'fb2'
  router.push(`/reader/${book.id}?format=${fmt}`)
}
</script>
