<script setup lang="ts">
import { ref } from 'vue'
import { useQuarantineStore } from '@/stores/quarantine'
import { useI18n } from 'vue-i18n'
import type { QuarantineItem } from '@/api/types'
import {
  FileText,
  AlertTriangle,
  Trash2,
  RefreshCw,
  FolderPlus,
  CopyCheck,
  Loader2
} from 'lucide-vue-next'

const { t } = useI18n()
const store = useQuarantineStore()
const resolvingId = ref<string | null>(null)

async function resolve(item: QuarantineItem, action: 'discard' | 'replace' | 'attach_format' | 'keep_both') {
  resolvingId.value = item.id
  try {
    await store.resolveConflict(item.id, action)
  } catch (err) {
    alert('Failed to resolve: ' + err)
  } finally {
    resolvingId.value = null
  }
}

function formatConflict(type: string): string {
  if (type === 'exact_hash') return t('quarantine.conflict_exact_hash')
  if (type === 'same_format') return t('quarantine.conflict_same_format')
  return t('quarantine.conflict_fuzzy')
}
</script>

<template>
  <div class="space-y-4">
    <div v-if="store.loading && store.items.length === 0" class="flex justify-center py-12">
      <Loader2 class="w-8 h-8 text-accent animate-spin" />
    </div>

    <div v-else-if="store.items.length === 0" class="p-12 text-center rounded-2xl bg-bg-surface border border-border">
      <CopyCheck class="w-12 h-12 text-accent/40 mx-auto mb-3" />
      <h3 class="font-bold text-base text-fg-primary">{{ t('quarantine.empty') }}</h3>
      <p class="text-xs text-fg-muted mt-1">Все входящие книги успешно добавлены без коллизий.</p>
    </div>

    <div v-else class="grid grid-cols-1 gap-4">
      <div
        v-for="item in store.items"
        :key="item.id"
        class="p-5 rounded-2xl bg-bg-surface border border-border shadow-sm flex flex-col md:flex-row items-start md:items-center justify-between gap-4 transition-all hover:border-accent/40"
      >
        <div class="space-y-1.5 flex-1 min-w-0">
          <div class="flex items-center gap-2 flex-wrap">
            <span class="px-2 py-0.5 rounded-lg text-xs font-bold bg-amber-500/15 text-amber-500 border border-amber-500/30">
              {{ formatConflict(item.conflict_type) }}
            </span>
            <span class="px-2 py-0.5 rounded text-[11px] font-semibold uppercase bg-bg-secondary text-fg-secondary border border-border">
              {{ item.format }}
            </span>
            <span class="text-xs text-fg-muted">
              {{ (item.file_size / 1024).toFixed(0) }} KB
            </span>
          </div>

          <h4 class="font-bold text-base text-fg-primary truncate">
            {{ item.parsed_title || 'Без названия' }}
          </h4>

          <div class="text-xs text-fg-secondary truncate">
            <span class="text-fg-muted">Автор:</span> {{ item.parsed_authors || 'Неизвестен' }}
          </div>

          <div class="text-[11px] text-fg-muted font-mono truncate max-w-xl">
            SHA-256: {{ item.sha256 }}
          </div>
        </div>

        <!-- Кнопки действий -->
        <div class="flex items-center flex-wrap gap-2 shrink-0">
          <!-- Заменить существующий файл -->
          <button
            v-if="item.existing_book_id"
            :disabled="resolvingId === item.id"
            @click="resolve(item, 'replace')"
            class="px-3 py-1.5 rounded-xl text-xs font-semibold bg-bg-secondary hover:bg-accent hover:text-white border border-border transition-colors flex items-center gap-1.5"
            :title="t('quarantine.action_replace')"
          >
            <RefreshCw class="w-3.5 h-3.5" />
            <span>{{ t('quarantine.action_replace') }}</span>
          </button>

          <!-- Объединить форматы -->
          <button
            v-if="item.existing_book_id"
            :disabled="resolvingId === item.id"
            @click="resolve(item, 'attach_format')"
            class="px-3 py-1.5 rounded-xl text-xs font-semibold bg-bg-secondary hover:bg-emerald-600 hover:text-white border border-border transition-colors flex items-center gap-1.5"
            :title="t('quarantine.action_attach')"
          >
            <FolderPlus class="w-3.5 h-3.5" />
            <span>{{ t('quarantine.action_attach') }}</span>
          </button>

          <!-- Оставить обе -->
          <button
            :disabled="resolvingId === item.id"
            @click="resolve(item, 'keep_both')"
            class="px-3 py-1.5 rounded-xl text-xs font-semibold bg-bg-secondary hover:bg-sky-600 hover:text-white border border-border transition-colors flex items-center gap-1.5"
            :title="t('quarantine.action_keep_both')"
          >
            <CopyCheck class="w-3.5 h-3.5" />
            <span>{{ t('quarantine.action_keep_both') }}</span>
          </button>

          <!-- Отклонить и удалить -->
          <button
            :disabled="resolvingId === item.id"
            @click="resolve(item, 'discard')"
            class="p-2 rounded-xl text-xs font-semibold bg-bg-secondary hover:bg-red-500 hover:text-white text-red-400 border border-border transition-colors"
            :title="t('quarantine.action_discard')"
          >
            <Trash2 class="w-4 h-4" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
