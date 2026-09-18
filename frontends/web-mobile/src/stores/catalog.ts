import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '@/api/client'
import { i18n } from '@/i18n'
import type { Book, BookListResponse, ShelfItem } from '@/api/types'

export type SortCriterion = 'recent' | 'year' | 'title' | 'author' | 'series'
export type SortDirection = 'asc' | 'desc'

export interface BookGroup {
  key: string
  title: string
  books: Book[]
}

export const useCatalogStore = defineStore('catalog', () => {
  const books = ref<Book[]>([])
  const total = ref(0)
  const page = ref(1)
  const perPage = ref(20)
  const totalPages = ref(1)
  const loading = ref(false)
  const loadingMore = ref(false)
  const error = ref<string | null>(null)

  const sortBy = ref<SortCriterion>('recent')
  const sortDirection = ref<SortDirection>('desc')

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

  const hasMore = computed(() => books.value.length < total.value)

  function getGroupForBook(
    book: Book,
    criterion: SortCriterion,
    locale: string,
    t: (k: string) => string
  ): { key: string; title: string } {
    switch (criterion) {
      case 'recent': {
        if (!book.created_at) {
          return { key: '__no_date__', title: t('catalog.group_without_year') }
        }
        const d = new Date(book.created_at)
        if (isNaN(d.getTime())) {
          return { key: '__invalid_date__', title: t('catalog.group_without_year') }
        }
        const month = d.toLocaleString(locale, { month: 'long' })
        const monthCapitalized = month.charAt(0).toUpperCase() + month.slice(1)
        const year = d.getFullYear()
        return { key: `${year}-${d.getMonth()}`, title: `${monthCapitalized} ${year}` }
      }
      case 'year': {
        if (!book.published_date || !book.published_date.trim()) {
          return { key: '__no_year__', title: t('catalog.group_without_year') }
        }
        const m = book.published_date.match(/\b\d{4}\b/)
        if (m) {
          const yearStr = m[0]
          return { key: `year_${yearStr}`, title: locale === 'ru' ? `${yearStr} год` : `${yearStr}` }
        }
        const raw = book.published_date.trim()
        return { key: `year_${raw}`, title: raw }
      }
      case 'title': {
        const tStr = (book.title || '').trim()
        if (!tStr) return { key: 'other', title: '#' }
        const first = tStr.charAt(0).toUpperCase()
        if (/\d/.test(first)) {
          return { key: 'digits', title: '0 — 9' }
        }
        return { key: `title_${first}`, title: first }
      }
      case 'author': {
        const firstAuthor = book.authors?.[0]
        const aName = firstAuthor ? (firstAuthor.sort_name || firstAuthor.name || '').trim() : ''
        if (!aName) {
          return { key: '__no_author__', title: t('catalog.group_without_author') }
        }
        const first = aName.charAt(0).toUpperCase()
        if (/\d/.test(first)) {
          return { key: 'author_digits', title: '0 — 9' }
        }
        return { key: `author_${first}`, title: first }
      }
      case 'series': {
        const firstSeries = book.series?.[0]
        const sName = firstSeries ? (firstSeries.name || '').trim() : ''
        if (!sName) {
          return { key: '__no_series__', title: t('catalog.group_without_series') }
        }
        const first = sName.charAt(0).toUpperCase()
        if (/\d/.test(first)) {
          return { key: 'series_digits', title: '0 — 9' }
        }
        return { key: `series_${first}`, title: first }
      }
      default:
        return { key: 'all', title: t('catalog.sort_recent') }
    }
  }

  const groupedBooks = computed<BookGroup[]>(() => {
    const list = books.value
    if (list.length === 0) return []

    const currentLocale =
      (typeof (i18n.global as any).locale === 'string'
        ? (i18n.global as any).locale
        : (i18n.global as any).locale?.value) || 'ru'
    const t = (key: string): string => (i18n.global as any).t(key)

    const groups: BookGroup[] = []
    let currentGroup: BookGroup | null = null

    for (const book of list) {
      const { key, title } = getGroupForBook(book, sortBy.value, currentLocale, t)
      if (!currentGroup || currentGroup.key !== key) {
        currentGroup = { key, title, books: [book] }
        groups.push(currentGroup)
      } else {
        currentGroup.books.push(book)
      }
    }

    return groups
  })

  async function fetchBooks(reset = true) {
    if (reset) {
      page.value = 1
      loading.value = true
    } else {
      loadingMore.value = true
    }
    error.value = null

    try {
      const res = await api.get<BookListResponse>('/api/v1/books', {
        sort: sortBy.value,
        dir: sortDirection.value,
        page: page.value,
        per_page: perPage.value
      })

      const newItems = res.items || []
      if (reset) {
        books.value = newItems
      } else {
        const existingIds = new Set(books.value.map(b => b.id))
        for (const item of newItems) {
          if (!existingIds.has(item.id)) {
            books.value.push(item)
          }
        }
      }
      total.value = res.total
      totalPages.value = res.total_pages || 1
    } catch (err: any) {
      error.value = err.message || 'Ошибка загрузки книг'
      if (reset) books.value = []
    } finally {
      loading.value = false
      loadingMore.value = false
    }
  }

  async function loadMore() {
    if (loading.value || loadingMore.value || !hasMore.value) return
    page.value += 1
    await fetchBooks(false)
  }

  function setSort(criterion: SortCriterion) {
    if (sortBy.value === criterion) return
    sortBy.value = criterion
    if (criterion === 'recent' || criterion === 'year') {
      sortDirection.value = 'desc'
    } else {
      sortDirection.value = 'asc'
    }
    fetchBooks(true)
  }

  function toggleDirection() {
    sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
    fetchBooks(true)
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
    loadingMore,
    hasMore,
    error,
    sortBy,
    sortDirection,
    groupedBooks,
    selectedBook,
    shelves,
    shelvesLoading,
    fetchBooks,
    loadMore,
    setSort,
    toggleDirection,
    fetchShelves,
    addToShelf,
    removeFromShelf,
    openBookSheet,
    closeBookSheet
  }
})
