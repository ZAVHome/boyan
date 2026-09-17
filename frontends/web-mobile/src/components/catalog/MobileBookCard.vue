<template>
  <div
    @click="$emit('select', book)"
    class="flex items-start gap-3 p-3 bg-theme-card border border-theme rounded-xl active:bg-theme-bg/60 cursor-pointer select-none relative overflow-hidden transition-colors"
  >
    <!-- Cover -->
    <div
      class="w-16 h-24 flex-shrink-0 rounded-lg overflow-hidden bg-theme-bg border border-theme flex items-center justify-center relative shadow-sm"
    >
      <img
        v-if="!imgError"
        :src="api.getCoverUrl(book.id)"
        :alt="book.title"
        class="w-full h-full object-cover"
        loading="lazy"
        @error="imgError = true"
      />
      <div v-else class="text-theme-muted flex flex-col items-center justify-center p-1 text-center">
        <BookIcon class="w-6 h-6 mb-1 opacity-40" />
        <span class="text-[9px] uppercase tracking-wider font-mono opacity-60">
          {{ book.files?.[0]?.format || 'BOOK' }}
        </span>
      </div>

      <!-- Offline badge on cover -->
      <div
        v-if="isOffline"
        class="absolute bottom-1 right-1 bg-emerald-600 text-white rounded-full p-0.5 shadow"
        title="Доступно офлайн"
      >
        <Check class="w-2.5 h-2.5 stroke-[3]" />
      </div>
    </div>

    <!-- Details -->
    <div class="flex-1 min-w-0 flex flex-col justify-between self-stretch py-0.5">
      <div>
        <h2 class="text-sm font-semibold text-theme-text line-clamp-2 leading-snug mb-1">
          {{ book.title }}
        </h2>
        <p class="text-xs text-theme-muted truncate mb-1">
          {{ authorNames }}
        </p>
        <p
          v-if="book.series && book.series.length > 0"
          class="text-[11px] text-primary-600 dark:text-primary-400 truncate"
        >
          {{ book.series[0].name }}
          <span v-if="book.series[0].index">#{{ book.series[0].index }}</span>
        </p>
      </div>

      <!-- Footer / Formats -->
      <div class="flex items-center justify-between gap-1 mt-2">
        <div class="flex items-center gap-1 flex-wrap">
          <span
            v-for="file in book.files"
            :key="file.id"
            class="px-1.5 py-0.5 text-[10px] uppercase font-mono font-medium rounded bg-theme-bg text-theme-muted border border-theme"
          >
            {{ file.format }}
          </span>
        </div>

        <button
          @click.stop="$emit('read', book)"
          class="flex items-center gap-1 px-2.5 py-1 text-xs font-medium rounded-lg bg-primary-600 text-white hover:bg-primary-700 active:scale-95 transition-transform"
        >
          <BookOpen class="w-3.5 h-3.5" />
          <span>{{ $t('common.read') }}</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { Book as BookIcon, BookOpen, Check } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { Book } from '@/api/types'
import { useOfflineStore } from '@/stores/offline'

const props = defineProps<{
  book: Book
}>()

const imgError = ref(false)

defineEmits<{
  (e: 'select', book: Book): void
  (e: 'read', book: Book): void
}>()

const offlineStore = useOfflineStore()

const authorNames = computed(() => {
  if (!props.book.authors || props.book.authors.length === 0) {
    return 'Неизвестный автор'
  }
  return props.book.authors.map((a) => a.name).join(', ')
})

const isOffline = computed(() => offlineStore.isBookSaved(props.book.id))
</script>
