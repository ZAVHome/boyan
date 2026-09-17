<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { adminApi, type AdminTask } from '@/api/admin'
import { api } from '@/api/client'
import { useI18n } from 'vue-i18n'
import {
  HardDrive,
  FolderSync,
  Play,
  StopCircle,
  Loader2,
  CheckCircle2,
  XCircle,
  Clock,
  FileArchive,
  RefreshCw,
  FolderOpen
} from 'lucide-vue-next'

const { t } = useI18n()

const tasks = ref<AdminTask[]>([])
const settings = ref<any>(null)
const isLoading = ref(false)
const isStartingScan = ref(false)

// Calibre форма
const calibrePath = ref('')
const calibreCopyFiles = ref(true)
const isImportingCalibre = ref(false)
const calibreResult = ref<string | null>(null)

let timer: any = null

onMounted(async () => {
  await loadData()
  timer = setInterval(fetchTasks, 1500)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

async function loadData() {
  isLoading.value = true
  try {
    const [cfg, tRes] = await Promise.all([
      adminApi.getSettings(),
      adminApi.listTasks(20)
    ])
    settings.value = cfg
    tasks.value = tRes.tasks
  } catch (err: any) {
    console.error('Failed to load storage admin data', err)
  } finally {
    isLoading.value = false
  }
}

async function fetchTasks() {
  try {
    const res = await adminApi.listTasks(20)
    tasks.value = res.tasks
  } catch {}
}

async function handleRunScan(target: 'watch_dir' | 'library_dir') {
  isStartingScan.value = true
  try {
    await adminApi.runScan(target)
    await fetchTasks()
  } catch (err: any) {
    alert(err.message || 'Failed to start scan')
  } finally {
    isStartingScan.value = false
  }
}

async function handleCancelTask(id: string) {
  try {
    await adminApi.cancelTask(id)
    await fetchTasks()
  } catch (err: any) {
    alert(err.message || 'Failed to cancel task')
  }
}

async function handleCalibreImport() {
  if (!calibrePath.value.trim()) return
  isImportingCalibre.value = true
  calibreResult.value = null
  try {
    const res = await api.post<any>('/api/v1/admin/import/calibre', {
      path: calibrePath.value.trim(),
      copy_files: calibreCopyFiles.value
    })
    calibreResult.value = t('admin.storage.calibre_imported', {
      imported: res.imported,
      total: res.total
    })
  } catch (err: any) {
    calibreResult.value = 'Error: ' + (err.message || 'Calibre import failed')
  } finally {
    isImportingCalibre.value = false
  }
}
</script>

<template>
  <div class="space-y-6 max-w-7xl mx-auto">
    <!-- Заголовок -->
    <div class="flex items-center justify-between pb-3 border-b border-border">
      <div>
        <h1 class="text-2xl font-bold tracking-tight text-fg-primary flex items-center gap-2">
          <HardDrive class="w-6 h-6 text-accent" />
          {{ t('admin.storage.title') }}
        </h1>
        <p class="text-sm text-fg-secondary mt-1">
          {{ t('admin.storage.subtitle') }}
        </p>
      </div>

      <button
        @click="fetchTasks"
        class="flex items-center gap-1.5 px-3 py-2 rounded-xl bg-bg-surface border border-border hover:bg-bg-hover text-fg-primary text-xs font-medium transition-colors"
      >
        <RefreshCw class="w-3.5 h-3.5" :class="{ 'animate-spin': isLoading }" />
        <span>{{ t('admin.refresh') }}</span>
      </button>
    </div>

    <!-- БЛОК 1: Конфигурация путей и ручной запуск сканирования -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
      <!-- Папка входящих (Watch Dir) -->
      <div class="bg-bg-surface rounded-2xl border border-border p-6 flex flex-col justify-between">
        <div>
          <div class="flex items-center justify-between mb-3">
            <span class="text-xs font-semibold uppercase tracking-wider text-fg-muted">
              {{ t('admin.storage.watch_dir_title') }}
            </span>
            <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-emerald-500/15 text-emerald-500">
              <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
              fsnotify active
            </span>
          </div>

          <div class="p-3 rounded-xl bg-bg-primary border border-border font-mono text-xs text-fg-primary break-all">
            {{ settings?.storage?.watch_dir || 'data/import' }}
          </div>
          <p class="text-xs text-fg-muted mt-2">
            {{ t('admin.storage.watch_dir_hint') }}
          </p>
        </div>

        <button
          @click="handleRunScan('watch_dir')"
          :disabled="isStartingScan"
          class="mt-4 flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl bg-accent text-white hover:bg-accent-hover text-sm font-medium transition-colors disabled:opacity-50"
        >
          <FolderSync class="w-4 h-4" />
          <span>{{ t('admin.storage.scan_watch_btn') }}</span>
        </button>
      </div>

      <!-- Папка библиотеки (Library Dir) -->
      <div class="bg-bg-surface rounded-2xl border border-border p-6 flex flex-col justify-between">
        <div>
          <div class="flex items-center justify-between mb-3">
            <span class="text-xs font-semibold uppercase tracking-wider text-fg-muted">
              {{ t('admin.storage.library_dir_title') }}
            </span>
            <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-accent/15 text-accent">
              Managed
            </span>
          </div>

          <div class="p-3 rounded-xl bg-bg-primary border border-border font-mono text-xs text-fg-primary break-all">
            {{ settings?.storage?.library_dir || 'data/library' }}
          </div>
          <p class="text-xs text-fg-muted mt-2">
            {{ t('admin.storage.library_dir_hint') }}
          </p>
        </div>

        <button
          @click="handleRunScan('library_dir')"
          :disabled="isStartingScan"
          class="mt-4 flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl bg-bg-surface border border-border hover:bg-bg-hover text-fg-primary text-sm font-medium transition-colors disabled:opacity-50"
        >
          <FolderOpen class="w-4 h-4 text-accent" />
          <span>{{ t('admin.storage.rescan_library_btn') }}</span>
        </button>
      </div>
    </div>

    <!-- БЛОК 2: Импорт библиотеки Calibre -->
    <div class="bg-bg-surface rounded-2xl border border-border p-6">
      <h2 class="text-base font-semibold text-fg-primary mb-2 flex items-center gap-2">
        <FileArchive class="w-4 h-4 text-accent" />
        {{ t('admin.storage.calibre_title') }}
      </h2>
      <p class="text-xs text-fg-secondary mb-4">
        {{ t('admin.storage.calibre_subtitle') }}
      </p>

      <div class="flex flex-col sm:flex-row gap-3">
        <input
          v-model="calibrePath"
          type="text"
          placeholder="/path/to/calibre/library (containing metadata.db)"
          class="flex-1 px-3.5 py-2 rounded-xl bg-bg-primary border border-border text-fg-primary text-sm focus:outline-none focus:border-accent"
        />
        <label class="flex items-center gap-2 px-2 text-xs text-fg-secondary cursor-pointer">
          <input v-model="calibreCopyFiles" type="checkbox" class="rounded border-border text-accent focus:ring-accent" />
          <span>{{ t('admin.storage.calibre_copy_files') }}</span>
        </label>
        <button
          @click="handleCalibreImport"
          :disabled="isImportingCalibre || !calibrePath"
          class="px-4 py-2 rounded-xl bg-accent text-white hover:bg-accent-hover text-sm font-medium transition-colors disabled:opacity-50 flex items-center gap-2 shrink-0"
        >
          <Loader2 v-if="isImportingCalibre" class="w-4 h-4 animate-spin" />
          <Play v-else class="w-4 h-4" />
          <span>{{ t('admin.storage.calibre_run_btn') }}</span>
        </button>
      </div>

      <div v-if="calibreResult" class="mt-3 p-3 rounded-xl bg-bg-primary border border-border text-xs font-mono">
        {{ calibreResult }}
      </div>
    </div>

    <!-- БЛОК 3: Менеджер фоновых задач (Task Manager) -->
    <div class="bg-bg-surface rounded-2xl border border-border overflow-hidden">
      <div class="p-4 border-b border-border flex items-center justify-between">
        <h2 class="text-base font-semibold text-fg-primary flex items-center gap-2">
          <Clock class="w-4 h-4 text-accent" />
          {{ t('admin.storage.tasks_title') }}
        </h2>
        <span class="text-xs text-fg-muted">
          {{ t('admin.storage.tasks_live_polling') }}
        </span>
      </div>

      <div class="divide-y divide-border">
        <div v-if="tasks.length === 0" class="py-8 text-center text-xs text-fg-muted">
          {{ t('admin.storage.no_tasks') }}
        </div>

        <div v-for="tItem in tasks" :key="tItem.id" class="p-4 space-y-2 hover:bg-bg-hover/30 transition-colors">
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
            <div class="flex items-center gap-2.5">
              <Loader2 v-if="tItem.status === 'running'" class="w-4 h-4 animate-spin text-accent" />
              <CheckCircle2 v-else-if="tItem.status === 'completed'" class="w-4 h-4 text-emerald-500" />
              <XCircle v-else-if="tItem.status === 'failed' || tItem.status === 'cancelled'" class="w-4 h-4 text-red-500" />
              <Clock v-else class="w-4 h-4 text-fg-muted" />

              <span class="font-bold text-sm text-fg-primary uppercase tracking-wider">
                {{ tItem.type }}
              </span>

              <span
                class="px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider"
                :class="{
                  'bg-accent/15 text-accent': tItem.status === 'running',
                  'bg-emerald-500/15 text-emerald-500': tItem.status === 'completed',
                  'bg-red-500/15 text-red-500': tItem.status === 'failed',
                  'bg-bg-primary text-fg-muted': tItem.status === 'pending' || tItem.status === 'cancelled'
                }"
              >
                {{ tItem.status }}
              </span>
            </div>

            <div class="flex items-center gap-3 text-xs text-fg-muted">
              <span>{{ new Date(tItem.created_at).toLocaleTimeString() }}</span>
              <button
                v-if="tItem.status === 'running' || tItem.status === 'pending'"
                @click="handleCancelTask(tItem.id)"
                class="flex items-center gap-1 text-red-500 hover:text-red-600 font-medium px-2 py-1 rounded bg-red-500/10 border border-red-500/20"
              >
                <StopCircle class="w-3.5 h-3.5" />
                <span>{{ t('admin.storage.cancel_task') }}</span>
              </button>
            </div>
          </div>

          <!-- Прогресс-бар -->
          <div v-if="tItem.status === 'running' || tItem.progress > 0" class="space-y-1">
            <div class="flex justify-between text-xs text-fg-secondary">
              <span class="truncate max-w-md font-mono text-[11px]">{{ tItem.current_item || tItem.message || 'Processing...' }}</span>
              <span class="font-bold">{{ tItem.progress }}% ({{ tItem.processed_count }} / {{ tItem.total_count || '?' }})</span>
            </div>
            <div class="w-full bg-bg-primary rounded-full h-2 overflow-hidden border border-border">
              <div
                class="bg-accent h-full rounded-full transition-all duration-300"
                :style="{ width: `${tItem.progress}%` }"
              />
            </div>
          </div>

          <div v-if="tItem.message && tItem.status !== 'running'" class="text-xs text-fg-secondary">
            {{ tItem.message }}
          </div>

          <!-- Ошибки задачи, если есть -->
          <div v-if="tItem.errors && tItem.errors.length > 0" class="mt-2 p-2 rounded-lg bg-red-500/10 border border-red-500/20 text-[11px] text-red-500 font-mono max-h-24 overflow-y-auto">
            <div v-for="(err, eIdx) in tItem.errors" :key="eIdx">{{ err }}</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
