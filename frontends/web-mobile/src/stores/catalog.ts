import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/api/client'
import type { Book, BookListResponse, ShelfItem } from '@/api/types'

export const useCatalogStore = defineStore('catalog', () => {
  const books = ref<Book[]>([])
  const total = ref(0)
  const page = ref(1)
  const perPage = ref(20)
  const totalPages = ref(1)
  const loading = ref(false)
  const error = ref<string | null>(null)

  // Selected book for ActionSheet
  const selectedBook = ref<Book | null>(null)

  // Shelves
  const shelves = ref<{
    reading: ShelfItem[]
    finished: ShelfItem[]
    favorite: ShelfItem[]
  }>({
    reading: [],
    finished: [],
    favorite: []
  })
  const shelvesLoading = ref(false)

  async function fetchBooks(params: {
    page?: number
    search?: string
    sort?: string
    append?: boolean
  } = {}) {
    loading.value = true
    error.value = null
    try {
      const targetPage = params.page || 1
      const res = await api.get<BookListResponse>('/api/v1/books', {
        page: targetPage,
        per_page: perPage.value,
        q: params.search,
        sort: params.sort || 'created_at_desc'
      })

      if (params.append) {
        books.value = [...books.value, ...res.items]
      } else {
        books.value = res.items
      }
      total.value = res.total
      page.value = res.page
      totalPages.value = res.total_pages
    } catch (err: any) {
      error.value = err.message || 'Ошибка загрузки книг'
    } finally {
      loading.value = false
    }
  }

  async function fetchShelves() {
    shelvesLoading.value = true
    try {
      const types: ('reading' | 'finished' | 'favorite')[] = [
        'reading',
        'finished',
        'favorite'
      ]
      for (const t of types) {
        try {
          const res = await api.get<{ items: ShelfItem[] }>(`/api/v1/shelves/${t}`)
          shelves.value[t] = Array.isArray(res?.items) ? res.items : (Array.isArray(res) ? (res as any) : [])
        } catch {
          shelves.value[t] = []
        }
      }
    } finally {
      shelvesLoading.value = false
    }
  }

  async function addToShelf(bookId: string, shelfType: 'reading' | 'finished' | 'favorite') {
    await api.post(`/api/v1/books/${bookId}/shelf`, { shelf_type: shelfType })
    await fetchShelves()
  }

  async function removeFromShelf(shelfType: 'reading' | 'finished' | 'favorite', bookId: string) {
    await api.delete(`/api/v1/books/${bookId}/shelf/${shelfType}`)
    await fetchShelves()
  }

  function openBookSheet(book: Book) {
    selectedBook.value = book
  }

  function closeBookSheet() {
    selectedBook.value = null
  }

  return {
    books,
    total,
    page,
    perPage,
    totalPages,
    loading,
    error,
    selectedBook,
    shelves,
    shelvesLoading,
    fetchBooks,
    fetchShelves,
    addToShelf,
    removeFromShelf,
    openBookSheet,
    closeBookSheet
  }
})
