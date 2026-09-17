import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '@/api/client'
import type { Book } from '@/api/types'
import {
  listOfflineBooks,
  saveOfflineBook,
  removeOfflineBook,
  type OfflineBook
} from '@/db/offline'

export const useOfflineStore = defineStore('offline', () => {
  const savedBooks = ref<OfflineBook[]>([])
  const loading = ref(false)
  const savingBookId = ref<string | null>(null)

  const count = computed(() => savedBooks.value.length)
  const totalBytes = computed(() =>
    savedBooks.value.reduce((acc, b) => acc + (b.size || 0), 0)
  )

  const formattedTotalSize = computed(() => {
    const bytes = totalBytes.value
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  })

  function isBookSaved(id: string): boolean {
    return savedBooks.value.some((b) => b.id === id)
  }

  async function loadOfflineBooks() {
    loading.value = true
    try {
      savedBooks.value = await listOfflineBooks()
    } finally {
      loading.value = false
    }
  }

  async function saveBook(book: Book, format: string) {
    savingBookId.value = book.id
    try {
      const blob = await api.downloadBookBlob(book.id, format)
      const offlineBook: OfflineBook = {
        id: book.id,
        title: book.title,
        author: book.authors?.map((a) => a.name).join(', ') || 'Неизвестный автор',
        format,
        coverUrl: book.cover_cached ? api.getCoverUrl(book.id) : undefined,
        blob,
        savedAt: Date.now(),
        size: blob.size
      }

      await saveOfflineBook(offlineBook)
      await loadOfflineBooks()
    } finally {
      savingBookId.value = null
    }
  }

  async function removeBook(id: string) {
    await removeOfflineBook(id)
    savedBooks.value = savedBooks.value.filter((b) => b.id !== id)
  }

  // Initialize offline books on start
  loadOfflineBooks()

  return {
    savedBooks,
    loading,
    savingBookId,
    count,
    totalBytes,
    formattedTotalSize,
    isBookSaved,
    loadOfflineBooks,
    saveBook,
    removeBook
  }
})
