import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/api/client'
import type { Book, BookListResponse } from '@/api/types'

export const useCatalogStore = defineStore('catalog', () => {
  const books = ref<Book[]>([])
  const total = ref(0)
  const page = ref(1)
  const perPage = ref(24)
  const totalPages = ref(1)
  const loading = ref(false)
  const searchQuery = ref('')
  const selectedBook = ref<Book | null>(null)
  const viewMode = ref<'grid' | 'list'>('grid')

  async function fetchBooks(resetPage = false) {
    if (resetPage) {
      page.value = 1
    }
    loading.value = true
    try {
      const res = await api.get<BookListResponse>('/api/v1/books', {
        q: searchQuery.value,
        page: page.value,
        per_page: perPage.value
      })
      books.value = res.items || []
      total.value = res.total
      totalPages.value = res.total_pages || 1
    } catch (err) {
      console.error('Failed to fetch books:', err)
      books.value = []
    } finally {
      loading.value = false
    }
  }

  function setPage(newPage: number) {
    if (newPage >= 1 && newPage <= totalPages.value) {
      page.value = newPage
      fetchBooks()
    }
  }

  function setSearch(query: string) {
    searchQuery.value = query
    fetchBooks(true)
  }

  async function openBookDetail(bookId: string) {
    try {
      const b = await api.get<Book>(`/api/v1/books/${bookId}`)
      selectedBook.value = b
    } catch (err) {
      console.error('Failed to get book details:', err)
    }
  }

  function closeBookDetail() {
    selectedBook.value = null
  }

  return {
    books,
    total,
    page,
    perPage,
    totalPages,
    loading,
    searchQuery,
    selectedBook,
    viewMode,
    fetchBooks,
    setPage,
    setSearch,
    openBookDetail,
    closeBookDetail
  }
})
