<script setup lang="ts">
import { ref, computed } from 'vue'
import { useAdminStore } from '@/stores/admin'
import { useI18n } from 'vue-i18n'
import {
  Cpu,
  HardDrive,
  Database,
  BookOpen,
  Image as ImageIcon,
  Send,
  RefreshCw,
  Trash2,
  CheckCircle2,
  Layers,
  Users,
  Activity,
  Check
} from 'lucide-vue-next'

const { t } = useI18n()
const adminStore = useAdminStore()

const isCheckpointing = ref(false)
const isPurging = ref(false)
const actionSuccess = ref<string | null>(null)

function formatBytes(bytes: number) {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function formatUptime(seconds: number) {
  if (!seconds) return '0s'
  const d = Math.floor(seconds / (3600 * 24))
  const h = Math.floor((seconds % (3600 * 24)) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = seconds % 60
  if (d > 0) return `${d}d ${h}h ${m}m`
  if (h > 0) return `${h}h ${m}m ${s}s`
  return `${m}m ${s}s`
}

async function handleCheckpoint() {
  isCheckpointing.value = true
  actionSuccess.value = null
  try {
    await adminStore.checkpointDB()
    actionSuccess.value = t('admin.dashboard.checkpoint_success')
    setTimeout(() => (actionSuccess.value = null), 4000)
  } catch (err: any) {
    alert(err.message || 'Checkpoint failed')
  } finally {
    isCheckpointing.value = false
  }
}

async function handlePurgeCache() {
  if (!confirm(t('admin.dashboard.confirm_purge_cache'))) return
  isPurging.value = true
  actionSuccess.value = null
  try {
    await adminStore.purgeCoverCache()
    actionSuccess.value = t('admin.dashboard.purge_success')
    setTimeout(() => (actionSuccess.value = null), 4000)
  } catch (err: any) {
    alert(err.message || 'Purge failed')
  } finally {
    isPurging.value = false
  }
}
</script>

<template>
  <div class="space-y-6 max-w-7xl mx-auto">
    <!-- Заголовок страницы и кнопка обновления -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold tracking-tight text-fg-primary">
          {{ t('admin.dashboard.title') }}
        </h1>
        <p class="text-sm text-fg-secondary mt-1">
          {{ t('admin.dashboard.subtitle') }}
        </p>
      </div>

      <div class="flex items-center gap-3">
        <span v-if="actionSuccess" class="text-xs text-emerald-500 font-medium flex items-center gap-1 bg-emerald-500/10 px-3 py-1.5 rounded-lg border border-emerald-500/20 animate-fade-in">
          <Check class="w-3.5 h-3.5" />
          {{ actionSuccess }}
        </span>
        <button
          @click="adminStore.fetchDashboard"
          :disabled="adminStore.isLoading"
          class="flex items-center gap-1.5 px-3 py-2 rounded-xl bg-bg-surface border border-border hover:bg-bg-hover text-fg-primary text-xs font-medium transition-colors disabled:opacity-50"
        >
          <RefreshCw class="w-3.5 h-3.5" :class="{ 'animate-spin': adminStore.isLoading }" />
          <span>{{ t('admin.refresh') }}</span>
        </button>
      </div>
    </div>

    <!-- БЛОК 1: Ресурсы хоста (RAM, CPU, Uptime, Disk) -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      <!-- RAM Gauge -->
      <div class="bg-bg-surface rounded-2xl border border-border p-5 flex flex-col justify-between">
        <div class="flex items-center justify-between">
          <span class="text-xs font-medium uppercase tracking-wider text-fg-muted">{{ t('admin.host.ram') }}</span>
          <div class="w-8 h-8 rounded-xl bg-blue-500/10 text-blue-500 flex items-center justify-center">
            <Activity class="w-4 h-4" />
          </div>
        </div>
        <div class="mt-4">
          <div class="text-2xl font-bold text-fg-primary">
            {{ formatBytes(adminStore.hostMetrics?.alloc_bytes || 0) }}
          </div>
          <p class="text-xs text-fg-muted mt-1">
            Sys: {{ formatBytes(adminStore.hostMetrics?.sys_bytes || 0) }} / Heap: {{ formatBytes(adminStore.hostMetrics?.heap_alloc_bytes || 0) }}
          </p>
        </div>
        <div class="mt-3 text-[11px] text-fg-secondary">
          GC cycles: {{ adminStore.hostMetrics?.num_gc || 0 }}
        </div>
      </div>

      <!-- CPU & Runtime -->
      <div class="bg-bg-surface rounded-2xl border border-border p-5 flex flex-col justify-between">
        <div class="flex items-center justify-between">
          <span class="text-xs font-medium uppercase tracking-wider text-fg-muted">{{ t('admin.host.cpu') }}</span>
          <div class="w-8 h-8 rounded-xl bg-purple-500/10 text-purple-500 flex items-center justify-center">
            <Cpu class="w-4 h-4" />
          </div>
        </div>
        <div class="mt-4">
          <div class="text-2xl font-bold text-fg-primary">
            {{ adminStore.hostMetrics?.num_cpu || 1 }} Cores
          </div>
          <p class="text-xs text-fg-muted mt-1">
            Goroutines: {{ adminStore.hostMetrics?.num_goroutine || 0 }}
          </p>
        </div>
        <div class="mt-3 text-[11px] text-fg-secondary">
          Uptime: {{ formatUptime(adminStore.hostMetrics?.uptime_seconds || 0) }} ({{ adminStore.hostMetrics?.go_version }})
        </div>
      </div>

      <!-- Библиотечный диск -->
      <div class="bg-bg-surface rounded-2xl border border-border p-5 flex flex-col justify-between">
        <div class="flex items-center justify-between">
          <span class="text-xs font-medium uppercase tracking-wider text-fg-muted">{{ t('admin.host.disk') }}</span>
          <div class="w-8 h-8 rounded-xl bg-amber-500/10 text-amber-500 flex items-center justify-center">
            <HardDrive class="w-4 h-4" />
          </div>
        </div>
        <div class="mt-4">
          <div class="text-2xl font-bold text-fg-primary">
            {{ (adminStore.hostMetrics?.disk_used_percent || 0).toFixed(1) }}%
          </div>
          <div class="w-full bg-bg-primary rounded-full h-2 mt-2 overflow-hidden border border-border">
            <div
              class="bg-accent h-full rounded-full transition-all duration-500"
              :style="{ width: `${adminStore.hostMetrics?.disk_used_percent || 0}%` }"
            />
          </div>
        </div>
        <div class="mt-3 text-[11px] text-fg-secondary">
          Free: {{ formatBytes(adminStore.hostMetrics?.disk_free_bytes || 0) }} / Total: {{ formatBytes(adminStore.hostMetrics?.disk_total_bytes || 0) }}
        </div>
      </div>

      <!-- Статус Telegram-бота -->
      <div class="bg-bg-surface rounded-2xl border border-border p-5 flex flex-col justify-between">
        <div class="flex items-center justify-between">
          <span class="text-xs font-medium uppercase tracking-wider text-fg-muted">{{ t('admin.host.telegram') }}</span>
          <div class="w-8 h-8 rounded-xl bg-sky-500/10 text-sky-500 flex items-center justify-center">
            <Send class="w-4 h-4" />
          </div>
        </div>
        <div class="mt-4">
          <div class="text-2xl font-bold flex items-center gap-2">
            <span v-if="adminStore.telegramStatus?.enabled" class="text-emerald-500 flex items-center gap-1.5 text-lg">
              <span class="w-2.5 h-2.5 rounded-full bg-emerald-500 animate-pulse" />
              {{ t('admin.status_active') }}
            </span>
            <span v-else class="text-fg-muted text-lg">
              {{ t('admin.status_disabled') }}
            </span>
          </div>
          <p class="text-xs text-fg-muted mt-1">
            Token: {{ adminStore.telegramStatus?.has_token ? t('admin.token_configured') : t('admin.no_token') }}
          </p>
        </div>
        <div class="mt-3 text-[11px] text-fg-secondary">
          Allowed users: {{ adminStore.telegramStatus?.allowed_users || 0 }}
        </div>
      </div>
    </div>

    <!-- БЛОК 2: Сводка библиотеки и распределение форматов -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Статистика библиотеки -->
      <div class="bg-bg-surface rounded-2xl border border-border p-6 lg:col-span-2">
        <h2 class="text-base font-semibold text-fg-primary mb-4 flex items-center gap-2">
          <BookOpen class="w-4 h-4 text-accent" />
          {{ t('admin.dashboard.library_summary') }}
        </h2>

        <div class="grid grid-cols-2 sm:grid-cols-4 gap-4 mb-6">
          <div class="p-4 rounded-xl bg-bg-primary border border-border">
            <span class="text-xs text-fg-muted block">{{ t('admin.stats.total_books') }}</span>
            <span class="text-xl font-bold text-fg-primary mt-1 block">{{ adminStore.stats?.total_books || 0 }}</span>
          </div>
          <div class="p-4 rounded-xl bg-bg-primary border border-border">
            <span class="text-xs text-fg-muted block">{{ t('admin.stats.total_authors') }}</span>
            <span class="text-xl font-bold text-fg-primary mt-1 block">{{ adminStore.stats?.total_authors || 0 }}</span>
          </div>
          <div class="p-4 rounded-xl bg-bg-primary border border-border">
            <span class="text-xs text-fg-muted block">{{ t('admin.stats.total_series') }}</span>
            <span class="text-xl font-bold text-fg-primary mt-1 block">{{ adminStore.stats?.total_series || 0 }}</span>
          </div>
          <div class="p-4 rounded-xl bg-bg-primary border border-border">
            <span class="text-xs text-fg-muted block">{{ t('admin.stats.total_size') }}</span>
            <span class="text-xl font-bold text-fg-primary mt-1 block">{{ formatBytes(adminStore.stats?.total_bytes || 0) }}</span>
          </div>
        </div>

        <!-- Распределение форматов -->
        <h3 class="text-xs font-semibold uppercase tracking-wider text-fg-muted mb-3">
          {{ t('admin.stats.formats_distribution') }}
        </h3>
        <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-6 gap-3">
          <div
            v-for="(count, format) in (adminStore.stats?.format_counts || {})"
            :key="format"
            class="p-3 rounded-xl bg-bg-primary border border-border text-center"
          >
            <span class="text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded bg-accent/15 text-accent inline-block mb-1">
              {{ format }}
            </span>
            <span class="text-base font-bold text-fg-primary block">{{ count }}</span>
          </div>
          <div v-if="!adminStore.stats || Object.keys(adminStore.stats.format_counts).length === 0" class="col-span-full py-4 text-center text-xs text-fg-muted">
            {{ t('admin.no_data') }}
          </div>
        </div>
      </div>

      <!-- Сервисные операции и обслуживание БД/кэша -->
      <div class="bg-bg-surface rounded-2xl border border-border p-6 flex flex-col justify-between">
        <div>
          <h2 class="text-base font-semibold text-fg-primary mb-4 flex items-center gap-2">
            <Database class="w-4 h-4 text-accent" />
            {{ t('admin.dashboard.maintenance') }}
          </h2>

          <!-- БД SQLite метрики -->
          <div class="p-4 rounded-xl bg-bg-primary border border-border mb-4 space-y-2 text-xs">
            <div class="flex justify-between">
              <span class="text-fg-muted">SQLite DB:</span>
              <span class="font-semibold text-fg-primary">{{ formatBytes(adminStore.dbMetrics?.db_size_bytes || 0) }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-fg-muted">WAL File:</span>
              <span class="font-semibold text-fg-primary">{{ formatBytes(adminStore.dbMetrics?.wal_size_bytes || 0) }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-fg-muted">FTS5 Indexed:</span>
              <span class="font-semibold text-fg-primary">{{ adminStore.dbMetrics?.fts_count || 0 }} items</span>
            </div>

            <button
              @click="handleCheckpoint"
              :disabled="isCheckpointing"
              class="w-full mt-3 flex items-center justify-center gap-2 py-2 px-3 rounded-lg bg-bg-surface border border-border hover:bg-bg-hover text-fg-primary text-xs font-medium transition-colors disabled:opacity-50"
            >
              <RefreshCw class="w-3.5 h-3.5" :class="{ 'animate-spin': isCheckpointing }" />
              <span>{{ t('admin.dashboard.wal_checkpoint_btn') }}</span>
            </button>
          </div>

          <!-- Кэш обложек -->
          <div class="p-4 rounded-xl bg-bg-primary border border-border space-y-2 text-xs">
            <div class="flex justify-between">
              <span class="text-fg-muted">Cover Cache:</span>
              <span class="font-semibold text-fg-primary">{{ formatBytes(adminStore.cacheMetrics?.size_bytes || 0) }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-fg-muted">Cached Thumbnails:</span>
              <span class="font-semibold text-fg-primary">{{ adminStore.cacheMetrics?.total_files || 0 }}</span>
            </div>

            <button
              @click="handlePurgeCache"
              :disabled="isPurging"
              class="w-full mt-3 flex items-center justify-center gap-2 py-2 px-3 rounded-lg bg-red-500/10 border border-red-500/20 hover:bg-red-500/20 text-red-500 text-xs font-medium transition-colors disabled:opacity-50"
            >
              <Trash2 class="w-3.5 h-3.5" :class="{ 'animate-spin': isPurging }" />
              <span>{{ t('admin.dashboard.purge_cache_btn') }}</span>
            </button>
          </div>
        </div>

        <div class="mt-6 pt-4 border-t border-border flex justify-between items-center text-xs text-fg-muted">
          <span>{{ t('admin.dashboard.quick_links') }}:</span>
          <router-link to="/admin/logs" class="text-accent hover:underline">
            {{ t('admin.nav.logs') }} &rarr;
          </router-link>
        </div>
      </div>
    </div>
  </div>
</template>
