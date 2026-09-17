<template>
  <div class="min-h-screen bg-theme-bg pb-20">
    <TopHeader :title="$t('offline.title')" />

    <main class="max-w-md mx-auto px-4 py-3 space-y-4">
      <!-- Storage Summary Card -->
      <div class="p-4 rounded-xl bg-theme-card border border-theme flex items-center justify-between shadow-sm">
        <div class="space-y-0.5">
          <div class="text-xs font-semibold text-theme-text flex items-center gap-1.5">
            <DownloadCloud class="w-4 h-4 text-primary-600" />
            <span>{{ $t('offline.count_books', { count: offlineStore.count }) }}</span>
          </div>
          <p class="text-[11px] text-theme-muted">
            {{ $t('offline.total_size', { size: offlineStore.formattedTotalSize }) }}
          </p>
        </div>

        <div class="flex items-center gap-1">
          <span class="w-2.5 h-2.5 rounded-full bg-emerald-500 animate-pulse"></span>
          <span class="text-xs font-mono text-emerald-600 dark:text-emerald-400 font-medium">Ready</span>
        </div>
      </div>

      <!-- Empty state -->
      <div
        v-if="offlineStore.savedBooks.length === 0"
        class="text-center py-16 px-4 space-y-3 text-theme-muted"
      >
        <HardDriveDownload class="w-12 h-12 mx-auto opacity-30" />
        <h3 class="text-sm font-semibold text-theme-text">{{ $t('offline.no_offline_books') }}</h3>
        <p class="text-xs leading-relaxed max-w-xs mx-auto">
          {{ $t('offline.tip_offline') }}
        </p>
        <router-link
          to="/"
          class="inline-block px-4 py-2 mt-2 rounded-xl bg-primary-600 text-white text-xs font-medium"
        >
          {{ $t('nav.catalog') }}
        </router-link>
      </div>

      <!-- Offline Books List -->
      <div v-else class="space-y-3">
        <div
          v-for="b in offlineStore.savedBooks"
          :key="b.id"
          class="flex items-start gap-3 p-3 bg-theme-card border border-theme rounded-xl relative shadow-sm"
        >
          <!-- Cover -->
          <div
            class="w-14 h-20 flex-shrink-0 rounded-lg overflow-hidden bg-theme-bg border border-theme flex items-center justify-center relative shadow-sm"
          >
            <img
              v-if="b.coverUrl"
              :src="b.coverUrl"
              :alt="b.title"
              class="w-full h-full object-cover"
            />
            <BookIcon v-else class="w-6 h-6 text-theme-muted opacity-40" />
          </div>

          <!-- Info -->
          <div class="flex-1 min-w-0 flex flex-col justify-between self-stretch py-0.5">
            <div>
              <h3 class="text-xs font-bold text-theme-text line-clamp-2 leading-snug">
                {{ b.title }}
              </h3>
              <p class="text-[11px] text-theme-muted truncate mt-0.5">
                {{ b.author }}
              </p>
              <div class="flex items-center gap-2 mt-1">
                <span class="px-1.5 py-0.5 text-[9px] uppercase font-mono font-bold rounded bg-theme-bg border border-theme text-theme-text">
                  {{ b.format }}
                </span>
                <span class="text-[10px] text-theme-muted">
                  {{ formatSize(b.size) }}
                </span>
                <span v-if="b.progress" class="text-[10px] text-primary-600 font-medium">
                  {{ b.progress.percentage }}%
                </span>
              </div>
            </div>

            <!-- Actions -->
            <div class="flex items-center justify-between gap-2 mt-2 pt-1 border-t border-theme/40">
              <button
                @click="removeBook(b.id)"
                class="p-1 rounded text-theme-muted hover:text-rose-500"
                :title="$t('common.delete')"
              >
                <Trash2 class="w-3.5 h-3.5" />
              </button>

              <button
                @click="readOfflineBook(b)"
                class="flex items-center gap-1 px-3 py-1 text-xs font-semibold rounded-lg bg-primary-600 text-white active:scale-95 shadow-sm"
              >
                <BookOpen class="w-3.5 h-3.5" />
                <span>{{ $t('common.read') }}</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </main>

    <BottomNav />
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  DownloadCloud,
  HardDriveDownload,
  Book as BookIcon,
  BookOpen,
  Trash2
} from 'lucide-vue-next'
import TopHeader from '@/components/common/TopHeader.vue'
import BottomNav from '@/components/common/BottomNav.vue'
import { useOfflineStore } from '@/stores/offline'
import type { OfflineBook } from '@/db/offline'

const router = useRouter()
const offlineStore = useOfflineStore()

onMounted(() => {
  offlineStore.loadOfflineBooks()
})

function formatSize(bytes: number) {
  if (!bytes) return '0 B'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

async function removeBook(id: string) {
  if (confirm('Удалить эту книгу из офлайн-хранилища?')) {
    await offlineStore.removeBook(id)
  }
}

function readOfflineBook(book: OfflineBook) {
  router.push(`/reader/${book.id}?format=${book.format}`)
}
</script>
