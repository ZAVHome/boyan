<template>
  <div class="min-h-screen bg-theme-bg pb-20">
    <TopHeader :title="$t('shelves.title')" />

    <main class="max-w-md mx-auto px-4 py-3 space-y-4">
      <!-- Tabs for Shelves -->
      <div class="grid grid-cols-3 gap-1.5 p-1 bg-theme-card border border-theme rounded-xl">
        <button
          v-for="s in shelfTabs"
          :key="s.type"
          @click="activeShelf = s.type"
          :class="[
            'py-1.5 px-1 text-[11px] font-medium rounded-lg text-center transition-colors flex flex-col items-center justify-center gap-0.5',
            activeShelf === s.type
              ? 'bg-primary-600 text-white font-semibold shadow-sm'
              : 'text-theme-muted hover:text-theme-text'
          ]"
        >
          <span>{{ s.label }}</span>
          <span class="text-[9px] opacity-80">({{ shelfItemsCount(s.type) }})</span>
        </button>
      </div>

      <!-- Loading skeleton -->
      <div v-if="catalogStore.shelvesLoading" class="space-y-3">
        <div
          v-for="i in 3"
          :key="i"
          class="h-24 rounded-xl bg-theme-card/60 animate-pulse border border-theme"
        ></div>
      </div>

      <!-- Empty state -->
      <div
        v-else-if="currentItems.length === 0"
        class="text-center py-16 text-theme-muted space-y-2"
      >
        <Bookmark class="w-10 h-10 mx-auto opacity-30" />
        <p class="text-xs">{{ $t('shelves.empty_shelf') }}</p>
      </div>

      <!-- Items List -->
      <div v-else class="space-y-2.5">
        <div
          v-for="item in currentItems"
          :key="item.id"
          class="relative"
        >
          <MobileBookCard
            v-if="item.book"
            :book="item.book"
            @select="catalogStore.openBookSheet"
            @read="onReadBook"
          />
          <div
            v-else
            class="p-3 bg-theme-card border border-theme rounded-xl text-xs text-theme-muted"
          >
            ID книги: {{ item.book_id }}
          </div>
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
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Bookmark } from 'lucide-vue-next'
import TopHeader from '@/components/common/TopHeader.vue'
import BottomNav from '@/components/common/BottomNav.vue'
import MobileBookCard from '@/components/catalog/MobileBookCard.vue'
import BookActionSheet from '@/components/catalog/BookActionSheet.vue'
import { useCatalogStore } from '@/stores/catalog'
import type { Book } from '@/api/types'

type ShelfType = 'reading' | 'finished' | 'favorite'

const router = useRouter()
const { t } = useI18n()
const catalogStore = useCatalogStore()

const activeShelf = ref<ShelfType>('reading')

const shelfTabs = computed(() => [
  { type: 'reading' as const, label: t('shelves.reading') },
  { type: 'finished' as const, label: t('shelves.finished') },
  { type: 'favorite' as const, label: t('shelves.favorites') }
])

function shelfItemsCount(type: ShelfType) {
  return catalogStore.shelves[type]?.length || 0
}

const currentItems = computed(() => {
  return catalogStore.shelves[activeShelf.value] || []
})

onMounted(() => {
  catalogStore.fetchShelves()
})

function onReadBook(book: Book) {
  const fmt = book.files?.[0]?.format.toLowerCase() || 'fb2'
  router.push(`/reader/${book.id}?format=${fmt}`)
}
</script>
