<script setup lang="ts">
import { computed } from 'vue'
import { api } from '@/api/client'
import type { Book } from '@/api/types'
import { useI18n } from 'vue-i18n'
import { BookOpen, FileText } from 'lucide-vue-next'

const props = defineProps<{
  book: Book
}>()

const emit = defineEmits<{
  (e: 'click', book: Book): void
  (e: 'read', book: Book): void
}>()

const { t } = useI18n()

const coverUrl = computed(() => {
  if (props.book.cover_cached) {
    return api.getCoverUrl(props.book.id)
  }
  return ''
})

const authorSummary = computed(() => {
  if (props.book.authors && props.book.authors.length > 0) {
    return props.book.authors.map(a => a.name).join(', ')
  }
  return t('catalog.author_unknown')
})

const seriesSummary = computed(() => {
  if (props.book.series && props.book.series.length > 0) {
    const s = props.book.series[0]
    return s.index ? `#${s.index} в ${s.name}` : s.name
  }
  return ''
})

const availableFormats = computed(() => {
  if (!props.book.files) return []
  return props.book.files.map(f => f.format.toUpperCase())
})
</script>

<template>
  <div
    @click="emit('click', book)"
    class="group cursor-pointer rounded-2xl bg-bg-surface border border-border p-3 flex flex-col transition-all duration-200 hover:shadow-xl hover:shadow-accent/10 hover:border-accent/40 hover:-translate-y-1"
  >
    <!-- Обложка книги -->
    <div class="relative w-full aspect-[2/3] rounded-xl overflow-hidden bg-bg-secondary border border-border/50 mb-3 flex items-center justify-center">
      <img
        v-if="coverUrl"
        :src="coverUrl"
        :alt="book.title"
        loading="lazy"
        class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
      />
      <!-- Заглушка если обложки нет -->
      <div v-else class="w-full h-full p-4 flex flex-col justify-between bg-gradient-to-br from-bg-secondary to-bg-surface text-center">
        <div class="text-[10px] uppercase font-bold tracking-widest text-accent/70 truncate">
          {{ seriesSummary || authorSummary }}
        </div>
        <div class="font-bold text-fg-primary text-sm line-clamp-3 leading-snug">
          {{ book.title }}
        </div>
        <div class="text-xs text-fg-muted truncate">
          {{ authorSummary }}
        </div>
      </div>

      <!-- Бейджи форматов -->
      <div class="absolute top-2 right-2 flex flex-col gap-1 items-end">
        <span
          v-for="fmt in availableFormats"
          :key="fmt"
          class="px-1.5 py-0.5 rounded text-[10px] font-bold bg-bg-primary/80 backdrop-blur-md text-fg-primary border border-border shadow-sm uppercase tracking-wider"
        >
          {{ fmt }}
        </span>
      </div>

      <!-- Оверлей при наведении с кнопкой быстрой читки -->
      <div class="absolute inset-0 bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center gap-2 p-2">
        <button
          @click.stop="emit('read', book)"
          class="p-2.5 rounded-xl bg-accent text-white hover:bg-accent-hover shadow-lg hover:scale-110 transition-all"
          :title="t('book.read_online')"
        >
          <BookOpen class="w-5 h-5" />
        </button>
      </div>
    </div>

    <!-- Метаданные книги -->
    <div class="flex-1 flex flex-col justify-between">
      <div>
        <span v-if="seriesSummary" class="text-[11px] font-semibold text-accent block truncate mb-0.5">
          {{ seriesSummary }}
        </span>
        <h3 class="font-semibold text-sm text-fg-primary line-clamp-2 leading-tight group-hover:text-accent transition-colors" :title="book.title">
          {{ book.title }}
        </h3>
      </div>
      <p class="text-xs text-fg-secondary mt-1.5 truncate">
        {{ authorSummary }}
      </p>
    </div>
  </div>
</template>
