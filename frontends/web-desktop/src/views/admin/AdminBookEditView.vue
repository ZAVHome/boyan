<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { api } from '@/api/client'
import { adminApi, type UpdateBookPayload } from '@/api/admin'
import type { Book } from '@/api/types'
import {
  ArrowLeft,
  Save,
  Loader2,
  CheckCircle2,
  AlertTriangle,
  Trash2,
  RefreshCw,
  BookOpen,
  Download,
  Layers,
  User,
  Tag,
  Building2,
  FileText
} from 'lucide-vue-next'

const props = defineProps<{
  id: string
}>()

const router = useRouter()
const route = useRoute()
const { t } = useI18n()

const bookId = computed(() => props.id || (route.params.id as string))

const book = ref<Book | null>(null)
const isLoading = ref(true)
const loadError = ref<string | null>(null)

// Form fields
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

const rawAuthors = ref('')
const rawSeriesName = ref('')
const rawSeriesIndex = ref(1)
const rawGenres = ref('')

// State tracking
const isSaving = ref(false)
const saveSuccess = ref(false)
const saveError = ref<string | null>(null)
const isRegeneratingCover = ref(false)
const coverRegenSuccess = ref(false)
const coverKey = ref(Date.now())
const imgError = ref(false)

// Delete modal
const showDeleteModal = ref(false)
const deleteFilesOption = ref(false)
const isDeleting = ref(false)

const coverUrl = computed(() => {
  if (imgError.value || !bookId.value) return ''
  return `${api.getCoverUrl(bookId.value)}?t=${coverKey.value}`
})

const popularGenres = [
  { code: 'sf', label: 'Фантастика' },
  { code: 'sf_space', label: 'Космоопера' },
  { code: 'fantasy', label: 'Фэнтези' },
  { code: 'detective', label: 'Детектив' },
  { code: 'thriller', label: 'Триллер' },
  { code: 'prose_classic', label: 'Классика' },
  { code: 'history', label: 'История' },
  { code: 'science', label: 'Наука' }
]

onMounted(async () => {
  await fetchBook()
})

async function fetchBook() {
  isLoading.value = true
  loadError.value = null
  try {
    const data = await api.get<Book>(`/api/v1/books/${bookId.value}`)
    book.value = data

    // Populate form
    rawAuthors.value = (data.authors || []).map(a => a.name).join(', ')
    const s = data.series && data.series.length > 0 ? data.series[0] : null
    rawSeriesName.value = s ? s.name : ''
    rawSeriesIndex.value = s?.index || 1
    rawGenres.value = (data.genres || []).map(g => g.code).join(', ')

    editForm.value = {
      title: data.title || '',
      original_title: data.original_title || '',
      annotation: data.annotation || '',
      language: data.language || '',
      publisher: data.publisher || '',
      published_date: data.published_date || '',
      isbn: data.isbn || '',
      authors: [],
      series: [],
      genres: []
    }
  } catch (err: any) {
    loadError.value = err.message || 'Не удалось загрузить книгу'
  } finally {
    isLoading.value = false
  }
}

function addGenreCode(code: string) {
  const current = rawGenres.value
    .split(',')
    .map(g => g.trim())
    .filter(Boolean)
  if (!current.includes(code)) {
    current.push(code)
    rawGenres.value = current.join(', ')
  }
}

async function handleSave() {
  if (!editForm.value.title.trim()) {
    saveError.value = t('admin.books.title_required')
    return
  }

  // Parse authors
  const authorsList = rawAuthors.value
    .split(',')
    .map(s => s.trim())
    .filter(Boolean)
    .map((name, idx) => ({ name, role: 'author', order: idx + 1 }))

  // Parse series
  const seriesList = rawSeriesName.value.trim()
    ? [{ name: rawSeriesName.value.trim(), index: Number(rawSeriesIndex.value) || 1 }]
    : []

  // Parse genres
  const genresList = rawGenres.value
    .split(',')
    .map(s => s.trim())
    .filter(Boolean)

  editForm.value.authors = authorsList
  editForm.value.series = seriesList
  editForm.value.genres = genresList

  isSaving.value = true
  saveError.value = null
  saveSuccess.value = false

  try {
    const updated = await adminApi.updateBook(bookId.value, editForm.value)
    book.value = updated
    saveSuccess.value = true
    setTimeout(() => {
      saveSuccess.value = false
    }, 4000)
  } catch (err: any) {
    saveError.value = err.message || 'Ошибка сохранения метаданных'
  } finally {
    isSaving.value = false
  }
}

async function handleRegenerateCover() {
  isRegeneratingCover.value = true
  coverRegenSuccess.value = false
  try {
    await adminApi.regenerateCover(bookId.value)
    imgError.value = false
    coverKey.value = Date.now()
    coverRegenSuccess.value = true
    setTimeout(() => {
      coverRegenSuccess.value = false
    }, 3000)
  } catch (err: any) {
    alert(err.message || 'Не удалось перегенерировать обложку')
  } finally {
    isRegeneratingCover.value = false
  }
}

async function handleConfirmDelete() {
  isDeleting.value = true
  try {
    await adminApi.deleteBook(bookId.value, deleteFilesOption.value)
    showDeleteModal.value = false
    router.push({ name: 'admin-books' })
  } catch (err: any) {
    alert(err.message || 'Ошибка при удалении книги')
  } finally {
    isDeleting.value = false
  }
}

function goBack() {
  if (window.history.length > 1) {
    router.back()
  } else {
    router.push({ name: 'admin-books' })
  }
}

function openReader() {
  router.push({ name: 'reader', params: { id: bookId.value } })
}
</script>

<template>
  <div class="space-y-6 max-w-6xl mx-auto pb-12">
    <!-- Верхняя панель навигации -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-4 border-b border-border">
      <div class="flex items-center gap-3">
        <button
          @click="goBack"
          class="p-2 rounded-xl bg-bg-surface border border-border text-fg-secondary hover:text-fg-primary hover:bg-bg-hover transition-colors shadow-xs"
          :title="t('admin.books.back_to_books')"
        >
          <ArrowLeft class="w-5 h-5" />
        </button>
        <div>
          <div class="flex items-center gap-2 text-xs text-fg-muted mb-0.5">
            <router-link to="/admin" class="hover:text-accent transition-colors">Панель управления</router-link>
            <span>/</span>
            <router-link to="/admin/books" class="hover:text-accent transition-colors">Книги</router-link>
            <span>/</span>
            <span class="text-fg-secondary font-medium">Редактирование</span>
          </div>
          <h1 class="text-xl font-bold text-fg-primary tracking-tight truncate max-w-xl">
            {{ book ? book.title : t('admin.books.edit_page_title') }}
          </h1>
        </div>
      </div>

      <div class="flex items-center gap-2 self-end sm:self-auto">
        <button
          v-if="book"
          @click="openReader"
          class="flex items-center gap-1.5 px-3.5 py-2 rounded-xl bg-bg-surface border border-border text-fg-secondary hover:text-fg-primary hover:bg-bg-hover text-xs font-semibold shadow-xs transition-colors"
        >
          <BookOpen class="w-4 h-4 text-accent" />
          <span>{{ t('admin.books.read_online') }}</span>
        </button>

        <button
          @click="handleSave"
          :disabled="isSaving || isLoading"
          class="flex items-center gap-1.5 px-5 py-2 rounded-xl bg-accent text-white hover:bg-accent-hover text-xs font-semibold shadow-md shadow-accent/25 transition-all disabled:opacity-50"
        >
          <Loader2 v-if="isSaving" class="w-4 h-4 animate-spin" />
          <Save v-else class="w-4 h-4" />
          <span>{{ t('admin.save') }}</span>
        </button>
      </div>
    </div>

    <!-- Индикатор загрузки -->
    <div v-if="isLoading" class="flex flex-col items-center justify-center py-24 space-y-3">
      <Loader2 class="w-8 h-8 animate-spin text-accent" />
      <span class="text-sm text-fg-muted">{{ t('common.loading') }}</span>
    </div>

    <!-- Ошибка загрузки -->
    <div v-else-if="loadError" class="p-6 rounded-2xl bg-red-500/10 border border-red-500/20 text-center space-y-3">
      <AlertTriangle class="w-8 h-8 text-red-500 mx-auto" />
      <p class="text-sm text-red-500 font-medium">{{ loadError }}</p>
      <button
        @click="fetchBook"
        class="px-4 py-2 rounded-xl bg-bg-surface border border-border text-xs font-semibold hover:bg-bg-hover transition-colors"
      >
        Повторить попытку
      </button>
    </div>

    <!-- Основной контент редактирования -->
    <div v-else-if="book" class="grid grid-cols-1 lg:grid-cols-12 gap-6">
      <!-- Левая колонка: обложка, файлы, быстрые действия -->
      <div class="lg:col-span-4 space-y-6">
        <!-- Карточка обложки -->
        <div class="p-5 rounded-2xl bg-bg-surface border border-border shadow-xs space-y-4">
          <h3 class="text-xs font-bold uppercase tracking-wider text-fg-muted">
            {{ t('admin.books.col_cover') }}
          </h3>

          <div class="relative aspect-[2/3] w-full rounded-xl overflow-hidden bg-bg-secondary border border-border shadow-md flex items-center justify-center group">
            <img
              v-if="coverUrl"
              :src="coverUrl"
              :alt="book.title"
              class="w-full h-full object-cover"
              @error="imgError = true"
            />
            <div v-else class="w-full h-full p-4 flex flex-col justify-center items-center text-center">
              <BookOpen class="w-12 h-12 text-accent/40 mb-2" />
              <span class="text-xs text-fg-muted">Обложка отсутствует</span>
            </div>
          </div>

          <button
            @click="handleRegenerateCover"
            :disabled="isRegeneratingCover"
            class="w-full flex items-center justify-center gap-2 py-2.5 px-3 rounded-xl bg-bg-secondary hover:bg-bg-hover border border-border text-fg-primary text-xs font-semibold transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="isRegeneratingCover" class="w-4 h-4 animate-spin text-accent" />
            <RefreshCw v-else class="w-4 h-4 text-accent" />
            <span>{{ t('admin.books.regenerate_cover') }}</span>
          </button>

          <div v-if="coverRegenSuccess" class="p-2.5 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-500 text-xs font-medium text-center flex items-center justify-center gap-1.5 animate-fade-in">
            <CheckCircle2 class="w-4 h-4 shrink-0" />
            <span>{{ t('admin.books.cover_regen_success') }}</span>
          </div>
        </div>

        <!-- Карточка прикрепленных файлов -->
        <div class="p-5 rounded-2xl bg-bg-surface border border-border shadow-xs space-y-3">
          <h3 class="text-xs font-bold uppercase tracking-wider text-fg-muted">
            {{ t('admin.books.attached_files') }}
          </h3>

          <div v-if="book.files && book.files.length > 0" class="space-y-2">
            <div
              v-for="f in book.files"
              :key="f.id"
              class="flex items-center justify-between p-3 rounded-xl bg-bg-secondary border border-border/70 text-xs"
            >
              <div class="flex items-center gap-2.5 min-w-0">
                <FileText class="w-4 h-4 text-accent shrink-0" />
                <div class="min-w-0">
                  <div class="font-bold text-fg-primary uppercase">{{ f.format }}</div>
                  <div class="text-[11px] text-fg-muted truncate" :title="f.file_path">
                    {{ (f.file_size / 1024).toFixed(0) }} KB
                  </div>
                </div>
              </div>

              <a
                :href="api.getDownloadUrl(book.id, f.format)"
                download
                target="_blank"
                class="p-1.5 rounded-lg bg-bg-surface border border-border text-fg-secondary hover:text-accent hover:border-accent transition-colors"
                :title="t('book.download')"
              >
                <Download class="w-3.5 h-3.5" />
              </a>
            </div>
          </div>
          <div v-else class="text-xs text-fg-muted">
            Файлы книги не найдены
          </div>
        </div>

        <!-- Опасная зона -->
        <div class="p-5 rounded-2xl bg-red-500/5 border border-red-500/20 shadow-xs space-y-3">
          <h3 class="text-xs font-bold uppercase tracking-wider text-red-500">
            {{ t('admin.books.danger_zone') }}
          </h3>
          <p class="text-xs text-fg-muted">
            Удаление книги из каталога или полное стирание файлов с носителя.
          </p>
          <button
            @click="showDeleteModal = true"
            class="w-full flex items-center justify-center gap-2 py-2 px-3 rounded-xl bg-red-500/10 hover:bg-red-500/20 text-red-500 border border-red-500/30 text-xs font-semibold transition-colors"
          >
            <Trash2 class="w-4 h-4" />
            <span>{{ t('admin.books.action_delete') }}</span>
          </button>
        </div>
      </div>

      <!-- Правая колонка: форма метаданных -->
      <div class="lg:col-span-8 space-y-6">
        <!-- Уведомления об успехе/ошибке сохранения -->
        <div
          v-if="saveSuccess"
          class="p-4 rounded-2xl bg-emerald-500/10 border border-emerald-500/30 text-emerald-500 text-sm font-semibold flex items-center gap-2.5 animate-fade-in shadow-xs"
        >
          <CheckCircle2 class="w-5 h-5 shrink-0" />
          <span>{{ t('admin.books.save_success') }}</span>
        </div>

        <div
          v-if="saveError"
          class="p-4 rounded-2xl bg-red-500/10 border border-red-500/30 text-red-500 text-sm font-semibold flex items-center gap-2.5 animate-fade-in shadow-xs"
        >
          <AlertTriangle class="w-5 h-5 shrink-0" />
          <span>{{ saveError }}</span>
        </div>

        <!-- Секция: Основные метаданные -->
        <div class="p-6 rounded-2xl bg-bg-surface border border-border shadow-xs space-y-4">
          <div class="flex items-center gap-2 pb-2 border-b border-border">
            <User class="w-4 h-4 text-accent" />
            <h2 class="text-sm font-bold text-fg-primary">{{ t('admin.books.meta_general') }}</h2>
          </div>

          <div class="space-y-3.5">
            <div>
              <label class="text-xs font-semibold text-fg-muted block mb-1">
                {{ t('admin.books.field_title') }} <span class="text-red-500">*</span>
              </label>
              <input
                v-model="editForm.title"
                type="text"
                class="w-full px-3.5 py-2 rounded-xl bg-bg-primary border border-border text-sm text-fg-primary focus:outline-none focus:border-accent transition-colors"
                placeholder="Название книги"
              />
            </div>

            <div>
              <label class="text-xs font-semibold text-fg-muted block mb-1">
                {{ t('admin.books.field_original_title') }}
              </label>
              <input
                v-model="editForm.original_title"
                type="text"
                class="w-full px-3.5 py-2 rounded-xl bg-bg-primary border border-border text-sm text-fg-primary focus:outline-none focus:border-accent transition-colors"
                placeholder="Оригинальное название (если переводное)"
              />
            </div>

            <div>
              <label class="text-xs font-semibold text-fg-muted block mb-1">
                {{ t('admin.books.field_authors') }}
              </label>
              <input
                v-model="rawAuthors"
                type="text"
                class="w-full px-3.5 py-2 rounded-xl bg-bg-primary border border-border text-sm text-fg-primary focus:outline-none focus:border-accent transition-colors"
                placeholder="Аркадий Стругацкий, Борис Стругацкий"
              />
              <p class="text-[11px] text-fg-muted mt-1">Укажите авторов через запятую</p>
            </div>
          </div>
        </div>

        <!-- Секция: Серия и цикл -->
        <div class="p-6 rounded-2xl bg-bg-surface border border-border shadow-xs space-y-4">
          <div class="flex items-center gap-2 pb-2 border-b border-border">
            <Layers class="w-4 h-4 text-accent" />
            <h2 class="text-sm font-bold text-fg-primary">{{ t('admin.books.meta_series') }}</h2>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-3 gap-3.5">
            <div class="sm:col-span-2">
              <label class="text-xs font-semibold text-fg-muted block mb-1">
                {{ t('admin.books.field_series') }}
              </label>
              <input
                v-model="rawSeriesName"
                type="text"
                class="w-full px-3.5 py-2 rounded-xl bg-bg-primary border border-border text-sm text-fg-primary focus:outline-none focus:border-accent transition-colors"
                placeholder="Мир Полудня"
              />
            </div>
            <div>
              <label class="text-xs font-semibold text-fg-muted block mb-1">
                {{ t('admin.books.field_series_index') }}
              </label>
              <input
                v-model.number="rawSeriesIndex"
                type="number"
                step="1"
                min="1"
                class="w-full px-3.5 py-2 rounded-xl bg-bg-primary border border-border text-sm text-fg-primary focus:outline-none focus:border-accent transition-colors"
                placeholder="1"
              />
            </div>
          </div>
        </div>

        <!-- Секция: Жанры -->
        <div class="p-6 rounded-2xl bg-bg-surface border border-border shadow-xs space-y-4">
          <div class="flex items-center gap-2 pb-2 border-b border-border">
            <Tag class="w-4 h-4 text-accent" />
            <h2 class="text-sm font-bold text-fg-primary">{{ t('admin.books.meta_genres') }}</h2>
          </div>

          <div class="space-y-2.5">
            <div>
              <label class="text-xs font-semibold text-fg-muted block mb-1">
                {{ t('admin.books.field_genres') }}
              </label>
              <input
                v-model="rawGenres"
                type="text"
                class="w-full px-3.5 py-2 rounded-xl bg-bg-primary border border-border text-sm text-fg-primary focus:outline-none focus:border-accent transition-colors"
                placeholder="sf_space, sf, detectives"
              />
            </div>

            <!-- Быстрые теги жанров для вставки -->
            <div>
              <span class="text-[11px] font-semibold text-fg-muted block mb-1.5">{{ t('admin.books.popular_genres') }}:</span>
              <div class="flex flex-wrap gap-1.5">
                <button
                  v-for="pg in popularGenres"
                  :key="pg.code"
                  type="button"
                  @click="addGenreCode(pg.code)"
                  class="px-2.5 py-1 rounded-lg text-xs bg-bg-primary hover:bg-accent/15 hover:text-accent border border-border hover:border-accent/40 text-fg-secondary transition-colors"
                >
                  + {{ pg.label }} ({{ pg.code }})
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- Секция: Издательские данные -->
        <div class="p-6 rounded-2xl bg-bg-surface border border-border shadow-xs space-y-4">
          <div class="flex items-center gap-2 pb-2 border-b border-border">
            <Building2 class="w-4 h-4 text-accent" />
            <h2 class="text-sm font-bold text-fg-primary">{{ t('admin.books.meta_publishing') }}</h2>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3.5">
            <div>
              <label class="text-xs font-semibold text-fg-muted block mb-1">
                {{ t('admin.books.field_language') }}
              </label>
              <input
                v-model="editForm.language"
                type="text"
                class="w-full px-3.5 py-2 rounded-xl bg-bg-primary border border-border text-sm text-fg-primary focus:outline-none focus:border-accent transition-colors"
                placeholder="ru"
              />
            </div>

            <div>
              <label class="text-xs font-semibold text-fg-muted block mb-1">
                {{ t('admin.books.field_year') }}
              </label>
              <input
                v-model="editForm.published_date"
                type="text"
                class="w-full px-3.5 py-2 rounded-xl bg-bg-primary border border-border text-sm text-fg-primary focus:outline-none focus:border-accent transition-colors"
                placeholder="2024"
              />
            </div>

            <div class="sm:col-span-2">
              <label class="text-xs font-semibold text-fg-muted block mb-1">
                {{ t('admin.books.field_publisher') }}
              </label>
              <input
                v-model="editForm.publisher"
                type="text"
                class="w-full px-3.5 py-2 rounded-xl bg-bg-primary border border-border text-sm text-fg-primary focus:outline-none focus:border-accent transition-colors"
                placeholder="АСТ, Эксмо, Молодая гвардия"
              />
            </div>

            <div class="sm:col-span-2">
              <label class="text-xs font-semibold text-fg-muted block mb-1">
                ISBN
              </label>
              <input
                v-model="editForm.isbn"
                type="text"
                class="w-full px-3.5 py-2 rounded-xl bg-bg-primary border border-border text-sm text-fg-primary focus:outline-none focus:border-accent transition-colors"
                placeholder="978-5-17-123456-7"
              />
            </div>
          </div>
        </div>

        <!-- Секция: Аннотация -->
        <div class="p-6 rounded-2xl bg-bg-surface border border-border shadow-xs space-y-4">
          <div class="flex items-center gap-2 pb-2 border-b border-border">
            <FileText class="w-4 h-4 text-accent" />
            <h2 class="text-sm font-bold text-fg-primary">{{ t('admin.books.meta_annotation') }}</h2>
          </div>

          <div>
            <textarea
              v-model="editForm.annotation"
              rows="6"
              class="w-full px-3.5 py-2.5 rounded-xl bg-bg-primary border border-border text-sm text-fg-primary focus:outline-none focus:border-accent transition-colors leading-relaxed"
              placeholder="Текст аннотации книги..."
            />
          </div>
        </div>

        <!-- Нижняя панель действий -->
        <div class="flex items-center justify-end gap-3 pt-2">
          <button
            @click="goBack"
            class="px-5 py-2.5 rounded-xl border border-border hover:bg-bg-hover text-fg-secondary hover:text-fg-primary text-sm font-semibold transition-colors"
          >
            {{ t('admin.cancel') }}
          </button>

          <button
            @click="handleSave"
            :disabled="isSaving"
            class="flex items-center gap-2 px-6 py-2.5 rounded-xl bg-accent text-white hover:bg-accent-hover text-sm font-semibold shadow-md shadow-accent/25 transition-all disabled:opacity-50"
          >
            <Loader2 v-if="isSaving" class="w-4 h-4 animate-spin" />
            <Save v-else class="w-4 h-4" />
            <span>{{ t('admin.save') }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- Модальное окно подтверждения удаления книги -->
    <div v-if="showDeleteModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fade-in">
      <div class="bg-bg-surface border border-border rounded-2xl max-w-md w-full p-6 space-y-4 shadow-xl">
        <div class="flex items-center gap-3 text-red-500">
          <AlertTriangle class="w-6 h-6 shrink-0" />
          <h2 class="text-lg font-bold text-fg-primary">{{ t('admin.books.modal_delete_title') }}</h2>
        </div>

        <p class="text-sm text-fg-secondary">
          {{ t('admin.books.delete_book_prompt', { title: book?.title }) }}
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
          <button
            @click="showDeleteModal = false"
            class="px-4 py-2 rounded-xl border border-border hover:bg-bg-hover text-sm font-medium transition-colors"
          >
            {{ t('admin.cancel') }}
          </button>
          <button
            @click="handleConfirmDelete"
            :disabled="isDeleting"
            class="px-4 py-2 rounded-xl bg-red-500 text-white hover:bg-red-600 text-sm font-medium disabled:opacity-50 flex items-center gap-1.5 transition-colors"
          >
            <Loader2 v-if="isDeleting" class="w-4 h-4 animate-spin" />
            <span>{{ t('admin.delete') }}</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
