import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/api/client'
import type { QuarantineItem, QuarantineListResponse } from '@/api/types'

export const useQuarantineStore = defineStore('quarantine', () => {
  const items = ref<QuarantineItem[]>([])
  const total = ref(0)
  const loading = ref(false)

  async function fetchQuarantine() {
    loading.value = true
    try {
      const res = await api.get<QuarantineListResponse>('/api/v1/quarantine', {
        per_page: 50
      })
      items.value = res.items || []
      total.value = res.total
    } catch (err) {
      console.error('Failed to fetch quarantine items:', err)
      items.value = []
    } finally {
      loading.value = false
    }
  }

  async function resolveConflict(id: string, action: 'discard' | 'replace' | 'attach_format' | 'keep_both') {
    try {
      await api.post(`/api/v1/quarantine/${id}/resolve`, { action })
      await fetchQuarantine()
      return true
    } catch (err) {
      console.error('Failed to resolve quarantine item:', err)
      throw err
    }
  }

  return {
    items,
    total,
    loading,
    fetchQuarantine,
    resolveConflict
  }
})
