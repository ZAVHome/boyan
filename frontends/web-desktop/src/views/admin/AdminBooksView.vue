<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { adminApi, type UpdateBookPayload } from '@/api/admin'
import { api } from '@/api/client'
import type { Book } from '@/api/types'
import { useI18n } from 'vue-i18n'
import {
  BookOpen,
  Search,
  Edit2,
  Trash2,
  RefreshCw,
  Plus,
  X,
  Loader2,
  Check,
  AlertTriangle,
  Layers,
  Tag
} from 'lucide-vue-next'

const { t } = useI18n()

const books = ref<Book[]>([])
const total = ref(0)
const limit = ref(20)
const offset = ref(0)
const searchQuery = ref('')
const isLoading = ref(false)

// Чекбоксы и пакетные операции
const selectedIds = ref<string[]>([])
const isBatchOperating = ref(false)
const showBatchGenreModal = ref(false)
const showBatchSeriesModal = ref(false)
const batchGenreCode = ref('')
const batchSeriesName = ref('')

// Модалка удаления книги (с выбором стирания с диска)
const showDeleteModal = ref(false)
const deleteTargetBook = ref<Book | null>(null)
const deleteFilesOption = ref(false)
const isDeleting = ref(false)

// Модалка редактирования книги
const showEditModal = ref(false)
const editBook = ref<Book | null>(null)
const editForm = ref<UpdateBookPayload>({
  title: '',
  original_title: '',
  annotation: '',
  language: '',
  publisher: '',
  published_date: '',
  isbn: '',
  authors: [],
  series: [],
  genres: []
})
const isSaving = ref(false)
const editError = ref<string | null>(null)

// Вспомогательные строковые поля для авторов/серий/жанров в форме
const rawAuthors = ref('')
const rawSeriesName = ref('')
const rawSeriesIndex = ref(1)
const rawGenres = ref('')

onMounted(() => {
  fetchBooks()
})

async function fetchBooks() {
  isLoading.value = true
  try {
    const res = await adminApi.listBooks({
      q: searchQuery.value,
      limit: limit.value,
      offset: offset.value
    })
    books.value = res.books
    total.value = res.total
  } catch (err: any) {
    alert(err.message || 'Failed to load books')
  } finally {
    isLoading.value = false
  }
}

const isAllSelected = computed(() => {
  return books.value.length > 0 && selectedIds.value.length === books.value.length
})

function toggleSelectAll() {
  if (isAllSelected.value) {
    selectedIds.value = []
  } else {
    selectedIds.value = books.value.map(b => b.id)
  }
}

function openEditModal(book: Book) {
  editBook.value = book
  rawAuthors.value = (book.authors || []).map(a => a.name).join(', ')
  const s = book.series && book.series.length > 0 ? book.series[0] : null
  rawSeriesName.value = s ? s.name : ''
  rawSeriesIndex.value = s?.index || 1
  rawGenres.value = (book.genres || []).map(g => g.code).join(', ')

  editForm.value = {
    title: book.title,
    original_title: book.original_title || '',
    annotation: book.annotation || '',
    language: book.language || '',
    publisher: book.publisher || '',
    published_date: book.published_date || '',
    isbn: book.isbn || '',
    authors: [],
    series: [],
    genres: []
  }
  editError.value = null
  showEditModal.value = true
}

async function handleSaveEdit() {
  if (!editBook.value) return
  if (!editForm.value.title.trim()) {
    editError.value = t('admin.books.title_required')
    return
  }

  // Парсим авторов из строки
  const authorsList = rawAuthors.value
    .split(',')
    .map(s => s.trim())
    .filter(Boolean)
    .map((name, idx) => ({ name, role: 'author', order: idx + 1 }))

  // Парсим серию
  const seriesList = rawSeriesName.value.trim()
    ? [{ name: rawSeriesName.value.trim(), index: Number(rawSeriesIndex.value) || 1 }]
    : []

  // Парсим жанры
  const genresList = rawGenres.value
    .split(',')
    .map(s => s.trim())
    .filter(Boolean)

  editForm.value.authors = authorsList
  editForm.value.series = seriesList
  editForm.value.genres = genresList

  isSaving.value = true
  editError.value = null
  try {
    await adminApi.updateBook(editBook.value.id, editForm.value)
    showEditModal.value = false
    fetchBooks()
  } catch (err: any) {
    editError.value = err.message || 'Failed to update book'
  } finally {
    isSaving.value = false
  }
}

function openDeleteModal(book: Book) {
  deleteTargetBook.value = book
  deleteFilesOption.value = false
  showDeleteModal.value = true
}

async function handleConfirmDelete() {
  if (!deleteTargetBook.value) return
  isDeleting.value = true
  try {
    await adminApi.deleteBook(deleteTargetBook.value.id, deleteFilesOption.value)
    showDeleteModal.value = false
    deleteTargetBook.value = null
    fetchBooks()
  } catch (err: any) {
    alert(err.message || 'Failed to delete book')
  } finally {
    isDeleting.value = false
  }
}

async function handleRegenerateCover(bookId: string) {
  try {
    await adminApi.regenerateCover(bookId)
    alert(t('admin.books.cover_regen_success'))
    fetchBooks()
  } catch (err: any) {
    alert(err.message || 'Failed to regenerate cover')
  }
}

// Пакетные действия
async function handleBatchDelete() {
  if (!confirm(t('admin.books.confirm_batch_delete', { count: selectedIds.value.length }))) return
  const deleteFiles = confirm(t('admin.books.confirm_batch_delete_files'))
  isBatchOperating.value = true
  try {
    await adminApi.batchAction({
      book_ids: selectedIds.value,
      action: 'delete',
      delete_files: deleteFiles
    })
    selectedIds.value = []
    fetchBooks()
  } catch (err: any) {
    alert(err.message || 'Batch delete failed')
  } finally {
    isBatchOperating.value = false
  }
}

async function handleBatchAssignGenre() {
  if (!batchGenreCode.value.trim()) return
  isBatchOperating.value = true
  try {
    await adminApi.batchAction({
      book_ids: selectedIds.value,
      action: 'set_genre',
      genre_code: batchGenreCode.value.trim()
    })
    showBatchGenreModal.value = false
    batchGenreCode.value = ''
    selectedIds.value = []
    fetchBooks()
  } catch (err: any) {
    alert(err.message || 'Batch assign genre failed')
  } finally {
    isBatchOperating.value = false
  }
}

async function handleBatchAssignSeries() {
  if (!batchSeriesName.value.trim()) return
  isBatchOperating.value = true
  try {
    await adminApi.batchAction({
      book_ids: selectedIds.value,
      action: 'set_series',
      series_name: batchSeriesName.value.trim()
    })
    showBatchSeriesModal.value = false
    batchSeriesName.value = ''
    selectedIds.value = []
    fetchBooks()
  } catch (err: any) {
    alert(err.message || 'Batch assign series failed')
  } finally {
    isBatchOperating.value = false
  }
}

async function handleBatchRegenerateCovers() {
  if (!confirm(t('admin.books.confirm_batch_covers', { count: selectedIds.value.length }))) return
  isBatchOperating.value = true
  try {
    const res = await adminApi.batchAction({
      book_ids: selectedIds.value,
      action: 'regenerate_cover'
    })
    alert(t('admin.books.batch_covers_result', { success: res.success_count, errors: res.error_count }))
    selectedIds.value = []
    fetchBooks()
  } catch (err: any) {
    alert(err.message || 'Batch cover regeneration failed')
  } finally {
    isBatchOperating.value = false
  }
}
</script>

<template>
  <div class="space-y-6 max-w-7xl mx-auto pb-20">
    <!-- Заголовок -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold tracking-tight text-fg-primary flex items-center gap-2">
          <BookOpen class="w-6 h-6 text-accent" />
          {{ t('admin.books.title') }}
        </h1>
        <p class="text-sm text-fg-secondary mt-1">
          {{ t('admin.books.subtitle') }} ({{ t('admin.total_items', { count: total }) }})
        </p>
      </div>
    </div>

    <!-- Поиск -->
    <div class="bg-bg-surface rounded-2xl border border-border p-4">
      <div class="relative">
        <Search class="w-4 h-4 text-fg-muted absolute left-3.5 top-3 pointer-events-none" />
        <input
          v-model="searchQuery"
          @keyup.enter="fetchBooks"
          type="text"
          :placeholder="t('admin.books.search_placeholder')"
          class="w-full pl-10 pr-4 py-2 rounded-xl bg-bg-primary border border-border text-fg-primary text-sm focus:outline-none focus:border-accent"
        />
      </div>
    </div>

    <!-- Таблица книг -->
    <div class="bg-bg-surface rounded-2xl border border-border overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-left text-sm">
          <thead class="bg-bg-primary text-xs uppercase tracking-wider text-fg-muted border-b border-border">
            <tr>
              <th class="py-3 px-4 w-10 text-center">
                <input
                  type="checkbox"
                  :checked="isAllSelected"
                  @change="toggleSelectAll"
                  class="rounded border-border text-accent focus:ring-accent"
                />
              </th>
              <th class="py-3 px-4 w-16">{{ t('admin.books.col_cover') }}</th>
              <th class="py-3 px-4">{{ t('admin.books.col_title') }}</th>
              <th class="py-3 px-4">{{ t('admin.books.col_authors') }}</th>
              <th class="py-3 px-4">{{ t('admin.books.col_series') }}</th>
              <th class="py-3 px-4">{{ t('admin.books.col_formats') }}</th>
              <th class="py-3 px-4 text-right">{{ t('admin.actions') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <tr v-if="isLoading">
              <td colspan="7" class="py-12 text-center text-fg-muted">
                <Loader2 class="w-6 h-6 animate-spin mx-auto text-accent" />
              </td>
            </tr>
            <tr v-else-if="books.length === 0">
              <td colspan="7" class="py-12 text-center text-fg-muted">
                {{ t('admin.books.no_books_found') }}
              </td>
            </tr>
            <tr v-for="b in books" :key="b.id" class="hover:bg-bg-hover/50 transition-colors">
              <!-- Чекбокс выбора -->
              <td class="py-3 px-4 text-center">
                <input
                  type="checkbox"
                  :value="b.id"
                  v-model="selectedIds"
                  class="rounded border-border text-accent focus:ring-accent"
                />
              </td>
              <!-- Обложка -->
              <td class="py-3 px-4">
                <img
                  :src="api.getCoverUrl(b.id)"
                  alt=""
                  class="w-10 h-14 object-cover rounded-md bg-bg-primary border border-border shadow-xs"
                  loading="lazy"
                  @error="(e: any) => (e.target.style.display = 'none')"
                />
              </td>
              <!-- Название -->
              <td class="py-3 px-4 font-medium text-fg-primary max-w-xs">
                <div class="truncate font-semibold">{{ b.title }}</div>
                <div v-if="b.original_title" class="text-xs text-fg-muted truncate">
                  {{ b.original_title }}
                </div>
              </td>
              <!-- Авторы -->
              <td class="py-3 px-4 text-xs text-fg-secondary max-w-xs truncate">
                {{ (b.authors || []).map(a => a.name).join(', ') || '—' }}
              </td>
              <!-- Серия -->
              <td class="py-3 px-4 text-xs text-fg-secondary">
                <span v-if="b.series && b.series.length > 0">
                  {{ b.series[0].name }} #{{ b.series[0].index }}
                </span>
                <span v-else class="text-fg-muted">—</span>
              </td>
              <!-- Форматы -->
              <td class="py-3 px-4">
                <div class="flex flex-wrap gap-1">
                  <span
                    v-for="f in (b.files || [])"
                    :key="f.id"
                    class="px-1.5 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-accent/10 text-accent border border-accent/20"
                  >
                    {{ f.format }}
                  </span>
                </div>
              </td>
              <!-- Действия -->
              <td class="py-3 px-4 text-right">
                <div class="inline-flex items-center gap-1">
                  <!-- Перегенерировать обложку -->
                  <button
                    @click="handleRegenerateCover(b.id)"
                    class="p-1.5 rounded-lg hover:bg-bg-hover text-fg-secondary hover:text-fg-primary"
                    :title="t('admin.books.action_regen_cover')"
                  >
                    <RefreshCw class="w-4 h-4" />
                  </button>
                  <!-- Редактировать -->
                  <router-link
                    :to="{ name: 'admin-book-edit', params: { id: b.id } }"
                    class="p-1.5 rounded-lg hover:bg-bg-hover text-fg-secondary hover:text-fg-primary inline-flex items-center justify-center"
                    :title="t('admin.books.action_edit')"
                  >
                    <Edit2 class="w-4 h-4" />
                  </router-link>
                  <!-- Удалить -->
                  <button
                    @click="openDeleteModal(b)"
                    class="p-1.5 rounded-lg hover:bg-red-500/15 text-fg-secondary hover:text-red-500"
                    :title="t('admin.books.action_delete')"
                  >
                    <Trash2 class="w-4 h-4" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Пагинация -->
      <div v-if="total > limit" class="p-4 border-t border-border flex items-center justify-between text-xs text-fg-secondary">
        <span>{{ t('admin.showing_range', { from: offset + 1, to: Math.min(offset + limit, total), total }) }}</span>
        <div class="flex gap-2">
          <button
            :disabled="offset === 0"
            @click="offset -= limit; fetchBooks()"
            class="px-3 py-1.5 rounded-lg border border-border hover:bg-bg-hover disabled:opacity-50"
          >
            &larr; {{ t('admin.prev') }}
          </button>
          <button
            :disabled="offset + limit >= total"
            @click="offset += limit; fetchBooks()"
            class="px-3 py-1.5 rounded-lg border border-border hover:bg-bg-hover disabled:opacity-50"
          >
            {{ t('admin.next') }} &rarr;
          </button>
        </div>
      </div>
    </div>

    <!-- ПЛАВАЮЩАЯ ПАНЕЛЬ ПАКЕТНЫХ ДЕЙСТВИЙ (появляется при выборе чекбоксов) -->
    <div
      v-if="selectedIds.length > 0"
      class="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 bg-bg-surface border border-accent/40 shadow-2xl rounded-2xl px-6 py-3.5 flex items-center gap-4 animate-bounce-short backdrop-blur-md"
    >
      <span class="text-xs font-bold text-fg-primary">
        {{ t('admin.books.selected_count', { count: selectedIds.length }) }}
      </span>

      <div class="h-4 w-px bg-border" />

      <!-- Массовая смена жанра -->
      <button
        @click="showBatchGenreModal = true"
        :disabled="isBatchOperating"
        class="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-bg-primary hover:bg-bg-hover text-xs font-medium border border-border"
      >
        <Tag class="w-3.5 h-3.5 text-accent" />
        <span>{{ t('admin.books.batch_genre_btn') }}</span>
      </button>

      <!-- Массовая смена серии -->
      <button
        @click="showBatchSeriesModal = true"
        :disabled="isBatchOperating"
        class="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-bg-primary hover:bg-bg-hover text-xs font-medium border border-border"
      >
        <Layers class="w-3.5 h-3.5 text-accent" />
        <span>{{ t('admin.books.batch_series_btn') }}</span>
      </button>

      <!-- Массовая регенерация обложек -->
      <button
        @click="handleBatchRegenerateCovers"
        :disabled="isBatchOperating"
        class="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-bg-primary hover:bg-bg-hover text-xs font-medium border border-border"
      >
        <RefreshCw class="w-3.5 h-3.5 text-blue-500" />
        <span>{{ t('admin.books.batch_cover_btn') }}</span>
      </button>

      <!-- Массовое удаление -->
      <button
        @click="handleBatchDelete"
        :disabled="isBatchOperating"
        class="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-red-500/15 hover:bg-red-500/25 text-red-500 text-xs font-semibold border border-red-500/30"
      >
        <Trash2 class="w-3.5 h-3.5" />
        <span>{{ t('admin.books.batch_delete_btn') }}</span>
      </button>

      <!-- Снять выделение -->
      <button
        @click="selectedIds = []"
        class="p-1.5 rounded-lg text-fg-muted hover:text-fg-primary"
      >
        <X class="w-4 h-4" />
      </button>
    </div>

    <!-- МОДАЛКА: Редактирование метаданных книги -->
    <div v-if="showEditModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fade-in">
      <div class="bg-bg-surface border border-border rounded-2xl max-w-2xl w-full p-6 space-y-4 shadow-xl max-h-[90vh] flex flex-col">
        <div class="flex items-center justify-between pb-3 border-b border-border">
          <h2 class="text-lg font-bold text-fg-primary">{{ t('admin.books.modal_edit_title') }}</h2>
          <button @click="showEditModal = false" class="text-fg-muted hover:text-fg-primary"><X class="w-5 h-5" /></button>
        </div>

        <div v-if="editError" class="p-3 rounded-xl bg-red-500/10 border border-red-500/20 text-red-500 text-xs font-medium">
          {{ editError }}
        </div>

        <div class="flex-1 overflow-y-auto space-y-3.5 pr-1">
          <div>
            <label class="text-xs font-semibold text-fg-muted block mb-1">{{ t('admin.books.field_title') }} *</label>
            <input v-model="editForm.title" type="text" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm focus:outline-none focus:border-accent" />
          </div>

          <div>
            <label class="text-xs font-semibold text-fg-muted block mb-1">{{ t('admin.books.field_original_title') }}</label>
            <input v-model="editForm.original_title" type="text" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm focus:outline-none focus:border-accent" />
          </div>

          <div>
            <label class="text-xs font-semibold text-fg-muted block mb-1">{{ t('admin.books.field_authors') }} (через запятую)</label>
            <input v-model="rawAuthors" type="text" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm focus:outline-none focus:border-accent" />
          </div>

          <div class="grid grid-cols-3 gap-3">
            <div class="col-span-2">
              <label class="text-xs font-semibold text-fg-muted block mb-1">{{ t('admin.books.field_series') }}</label>
              <input v-model="rawSeriesName" type="text" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm focus:outline-none focus:border-accent" />
            </div>
            <div>
              <label class="text-xs font-semibold text-fg-muted block mb-1">{{ t('admin.books.field_series_index') }}</label>
              <input v-model.number="rawSeriesIndex" type="number" step="0.1" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm focus:outline-none focus:border-accent" />
            </div>
          </div>

          <div>
            <label class="text-xs font-semibold text-fg-muted block mb-1">{{ t('admin.books.field_genres') }} (коды через запятую, напр: sf_space, prose)</label>
            <input v-model="rawGenres" type="text" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm focus:outline-none focus:border-accent" />
          </div>

          <div class="grid grid-cols-3 gap-3">
            <div>
              <label class="text-xs font-semibold text-fg-muted block mb-1">{{ t('admin.books.field_language') }}</label>
              <input v-model="editForm.language" type="text" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm focus:outline-none focus:border-accent" />
            </div>
            <div>
              <label class="text-xs font-semibold text-fg-muted block mb-1">{{ t('admin.books.field_publisher') }}</label>
              <input v-model="editForm.publisher" type="text" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm focus:outline-none focus:border-accent" />
            </div>
            <div>
              <label class="text-xs font-semibold text-fg-muted block mb-1">{{ t('admin.books.field_year') }}</label>
              <input v-model="editForm.published_date" type="text" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm focus:outline-none focus:border-accent" />
            </div>
          </div>

          <div>
            <label class="text-xs font-semibold text-fg-muted block mb-1">{{ t('admin.books.field_annotation') }}</label>
            <textarea v-model="editForm.annotation" rows="4" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm focus:outline-none focus:border-accent" />
          </div>
        </div>

        <div class="flex justify-end gap-3 pt-3 border-t border-border">
          <button @click="showEditModal = false" class="px-4 py-2 rounded-xl border border-border hover:bg-bg-hover text-sm font-medium">{{ t('admin.cancel') }}</button>
          <button @click="handleSaveEdit" :disabled="isSaving" class="px-4 py-2 rounded-xl bg-accent text-white hover:bg-accent-hover text-sm font-medium disabled:opacity-50 flex items-center gap-1.5">
            <Loader2 v-if="isSaving" class="w-4 h-4 animate-spin" />
            <span>{{ t('admin.save') }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- МОДАЛКА: Удаление книги (с выбором стирания с диска) -->
    <div v-if="showDeleteModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fade-in">
      <div class="bg-bg-surface border border-border rounded-2xl max-w-md w-full p-6 space-y-4 shadow-xl">
        <div class="flex items-center gap-3 text-red-500">
          <AlertTriangle class="w-6 h-6 shrink-0" />
          <h2 class="text-lg font-bold text-fg-primary">{{ t('admin.books.modal_delete_title') }}</h2>
        </div>

        <p class="text-sm text-fg-secondary">
          {{ t('admin.books.delete_book_prompt', { title: deleteTargetBook?.title }) }}
        </p>

        <!-- Чекбокс удаления файлов с диска -->
        <div class="p-3.5 rounded-xl bg-red-500/10 border border-red-500/20 space-y-2">
          <label class="flex items-start gap-2.5 cursor-pointer">
            <input
              type="checkbox"
              v-model="deleteFilesOption"
              class="mt-1 rounded border-red-500/40 text-red-500 focus:ring-red-500"
            />
            <div class="text-xs">
              <span class="font-bold text-red-500 block">{{ t('admin.books.delete_files_label') }}</span>
              <span class="text-fg-muted block mt-0.5">{{ t('admin.books.delete_files_hint') }}</span>
            </div>
          </label>
        </div>

        <div class="flex justify-end gap-3 pt-3 border-t border-border">
          <button @click="showDeleteModal = false" class="px-4 py-2 rounded-xl border border-border hover:bg-bg-hover text-sm font-medium">{{ t('admin.cancel') }}</button>
          <button @click="handleConfirmDelete" :disabled="isDeleting" class="px-4 py-2 rounded-xl bg-red-500 text-white hover:bg-red-600 text-sm font-medium disabled:opacity-50 flex items-center gap-1.5">
            <Loader2 v-if="isDeleting" class="w-4 h-4 animate-spin" />
            <span>{{ t('admin.delete') }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- МОДАЛКА: Пакетное присвоение жанра -->
    <div v-if="showBatchGenreModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fade-in">
      <div class="bg-bg-surface border border-border rounded-2xl max-w-md w-full p-6 space-y-4 shadow-xl">
        <h2 class="text-lg font-bold text-fg-primary">{{ t('admin.books.batch_genre_title') }}</h2>
        <div>
          <label class="text-xs font-semibold text-fg-muted block mb-1">{{ t('admin.books.field_genre_code') }}</label>
          <input v-model="batchGenreCode" type="text" placeholder="sf_space" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm focus:outline-none focus:border-accent" />
        </div>
        <div class="flex justify-end gap-3 pt-3 border-t border-border">
          <button @click="showBatchGenreModal = false" class="px-4 py-2 rounded-xl border border-border hover:bg-bg-hover text-sm font-medium">{{ t('admin.cancel') }}</button>
          <button @click="handleBatchAssignGenre" :disabled="isBatchOperating" class="px-4 py-2 rounded-xl bg-accent text-white hover:bg-accent-hover text-sm font-medium disabled:opacity-50">
            {{ t('admin.apply') }}
          </button>
        </div>
      </div>
    </div>

    <!-- МОДАЛКА: Пакетное присвоение серии -->
    <div v-if="showBatchSeriesModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fade-in">
      <div class="bg-bg-surface border border-border rounded-2xl max-w-md w-full p-6 space-y-4 shadow-xl">
        <h2 class="text-lg font-bold text-fg-primary">{{ t('admin.books.batch_series_title') }}</h2>
        <div>
          <label class="text-xs font-semibold text-fg-muted block mb-1">{{ t('admin.books.field_series_name') }}</label>
          <input v-model="batchSeriesName" type="text" placeholder="Космоолухи" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm focus:outline-none focus:border-accent" />
        </div>
        <div class="flex justify-end gap-3 pt-3 border-t border-border">
          <button @click="showBatchSeriesModal = false" class="px-4 py-2 rounded-xl border border-border hover:bg-bg-hover text-sm font-medium">{{ t('admin.cancel') }}</button>
          <button @click="handleBatchAssignSeries" :disabled="isBatchOperating" class="px-4 py-2 rounded-xl bg-accent text-white hover:bg-accent-hover text-sm font-medium disabled:opacity-50">
            {{ t('admin.apply') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
