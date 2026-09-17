import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  adminApi,
  type StorageStats,
  type HostMetrics,
  type DatabaseMetrics,
  type CacheMetrics,
  type TelegramStatus,
  type AdminTask
} from '@/api/admin'

export const useAdminStore = defineStore('admin', () => {
  const stats = ref<StorageStats | null>(null)
  const hostMetrics = ref<HostMetrics | null>(null)
  const dbMetrics = ref<DatabaseMetrics | null>(null)
  const cacheMetrics = ref<CacheMetrics | null>(null)
  const telegramStatus = ref<TelegramStatus | null>(null)
  const activeTasks = ref<AdminTask[]>([])

  const isLoading = ref(false)
  const error = ref<string | null>(null)

  let taskPollTimer: any = null

  async function fetchDashboard() {
    isLoading.value = true
    error.value = null
    try {
      const [s, hm, dbm, cm, tg, tasksRes] = await Promise.all([
        adminApi.getStats(),
        adminApi.getHostMetrics(),
        adminApi.getDatabaseMetrics(),
        adminApi.getCacheMetrics(),
        adminApi.getTelegramStatus(),
        adminApi.listTasks(10)
      ])

      stats.value = s
      hostMetrics.value = hm
      dbMetrics.value = dbm
      cacheMetrics.value = cm
      telegramStatus.value = tg
      activeTasks.value = tasksRes.tasks

      checkPolling()
    } catch (err: any) {
      error.value = err.message || 'Failed to load dashboard data'
    } finally {
      isLoading.value = false
    }
  }

  async function fetchTasks() {
    try {
      const res = await adminApi.listTasks(15)
      activeTasks.value = res.tasks
      checkPolling()
    } catch {
      // Игнорируем сетевые сбои при опросе задач
    }
  }

  function checkPolling() {
    const hasRunning = activeTasks.value.some(
      t => t.status === 'running' || t.status === 'pending'
    )

    if (hasRunning && !taskPollTimer) {
      taskPollTimer = setInterval(() => {
        fetchTasks()
      }, 1500)
    } else if (!hasRunning && taskPollTimer) {
      clearInterval(taskPollTimer)
      taskPollTimer = null
    }
  }

  function stopPolling() {
    if (taskPollTimer) {
      clearInterval(taskPollTimer)
      taskPollTimer = null
    }
  }

  async function checkpointDB() {
    await adminApi.checkpointDatabase()
    const dbm = await adminApi.getDatabaseMetrics()
    dbMetrics.value = dbm
  }

  async function purgeCoverCache() {
    await adminApi.purgeCache()
    const cm = await adminApi.getCacheMetrics()
    cacheMetrics.value = cm
  }

  return {
    stats,
    hostMetrics,
    dbMetrics,
    cacheMetrics,
    telegramStatus,
    activeTasks,
    isLoading,
    error,
    fetchDashboard,
    fetchTasks,
    stopPolling,
    checkpointDB,
    purgeCoverCache
  }
})
