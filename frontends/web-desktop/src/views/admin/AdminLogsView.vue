<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { adminApi, type LogEntry } from '@/api/admin'
import { useI18n } from 'vue-i18n'
import {
  Terminal,
  RefreshCw,
  Play,
  Pause,
  Search,
  Filter,
  Trash2
} from 'lucide-vue-next'

const { t } = useI18n()

const logs = ref<LogEntry[]>([])
const levelFilter = ref('')
const searchFilter = ref('')
const isAutoRefresh = ref(true)
const isLoading = ref(false)

let timer: any = null

onMounted(() => {
  fetchLogs()
  timer = setInterval(() => {
    if (isAutoRefresh.value) {
      fetchLogs()
    }
  }, 2000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

async function fetchLogs() {
  try {
    const res = await adminApi.getLogs({
      limit: 200,
      level: levelFilter.value || undefined
    })
    logs.value = res.entries
  } catch (err) {
    console.error('Failed to load logs', err)
  }
}

const filteredLogs = computed(() => {
  if (!searchFilter.value.trim()) return logs.value
  const q = searchFilter.value.toLowerCase()
  return logs.value.filter(l =>
    l.message.toLowerCase().includes(q) ||
    (l.attrs && JSON.stringify(l.attrs).toLowerCase().includes(q))
  )
})

function formatTime(iso: string) {
  try {
    return new Date(iso).toLocaleTimeString()
  } catch {
    return iso
  }
}
</script>

<template>
  <div class="space-y-4 max-w-7xl mx-auto flex flex-col h-[calc(100vh-8rem)]">
    <!-- Шапка и фильтры -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-3 border-b border-border shrink-0">
      <div>
        <h1 class="text-2xl font-bold tracking-tight text-fg-primary flex items-center gap-2">
          <Terminal class="w-6 h-6 text-accent" />
          {{ t('admin.logs.title') }}
        </h1>
        <p class="text-sm text-fg-secondary mt-1">
          {{ t('admin.logs.subtitle') }}
        </p>
      </div>

      <!-- Контролы -->
      <div class="flex items-center gap-2.5">
        <!-- Уровень -->
        <select
          v-model="levelFilter"
          @change="fetchLogs"
          class="px-3 py-1.5 rounded-xl bg-bg-surface border border-border text-xs font-semibold uppercase tracking-wider focus:outline-none focus:border-accent"
        >
          <option value="">ALL LEVELS</option>
          <option value="INFO">INFO</option>
          <option value="WARN">WARN</option>
          <option value="ERROR">ERROR</option>
          <option value="DEBUG">DEBUG</option>
        </select>

        <!-- Поиск -->
        <div class="relative">
          <Search class="w-3.5 h-3.5 text-fg-muted absolute left-3 top-2.5 pointer-events-none" />
          <input
            v-model="searchFilter"
            type="text"
            placeholder="Search..."
            class="pl-8 pr-3 py-1.5 rounded-xl bg-bg-surface border border-border text-xs focus:outline-none focus:border-accent w-36 sm:w-48"
          />
        </div>

        <!-- Пауза / Автообновление -->
        <button
          @click="isAutoRefresh = !isAutoRefresh"
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold border border-border transition-colors"
          :class="isAutoRefresh ? 'bg-emerald-500/15 text-emerald-500 border-emerald-500/30' : 'bg-bg-surface text-fg-muted'"
        >
          <component :is="isAutoRefresh ? Pause : Play" class="w-3.5 h-3.5" />
          <span>{{ isAutoRefresh ? 'Auto (2s)' : 'Paused' }}</span>
        </button>

        <!-- Ручное обновление -->
        <button
          @click="fetchLogs"
          class="p-2 rounded-xl bg-bg-surface border border-border hover:bg-bg-hover text-fg-secondary"
        >
          <RefreshCw class="w-3.5 h-3.5" />
        </button>
      </div>
    </div>

    <!-- Окно терминала логов -->
    <div class="flex-1 bg-black/90 rounded-2xl border border-border p-4 font-mono text-xs overflow-y-auto space-y-1 shadow-inner text-neutral-300">
      <div v-if="filteredLogs.length === 0" class="py-12 text-center text-neutral-500">
        {{ t('admin.logs.empty') }}
      </div>

      <div
        v-for="(log, idx) in filteredLogs"
        :key="idx"
        class="flex items-start gap-3 py-0.5 hover:bg-neutral-900/60 px-1.5 rounded transition-colors"
      >
        <span class="text-neutral-500 shrink-0 select-none">{{ formatTime(log.timestamp) }}</span>

        <!-- Значок уровня -->
        <span
          class="px-1.5 py-0.2 rounded text-[10px] font-bold uppercase shrink-0"
          :class="{
            'bg-blue-500/20 text-blue-400': log.level === 'INFO',
            'bg-amber-500/20 text-amber-400': log.level === 'WARN',
            'bg-red-500/20 text-red-400': log.level === 'ERROR',
            'bg-neutral-700 text-neutral-400': log.level === 'DEBUG'
          }"
        >
          {{ log.level }}
        </span>

        <!-- Сообщение и атрибуты -->
        <div class="flex-1 break-all">
          <span class="text-neutral-100 font-medium">{{ log.message }}</span>
          <span v-if="log.attrs && Object.keys(log.attrs).length > 0" class="text-neutral-400 ml-2">
            <span v-for="(v, k) in log.attrs" :key="k" class="mr-2 inline-block">
              <span class="text-neutral-500">{{ k }}=</span><span class="text-amber-300/80">"{{ v }}"</span>
            </span>
          </span>
        </div>
      </div>
    </div>
  </div>
</template>
