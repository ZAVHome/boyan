<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '@/api/client'
import type { Book, ReadProgress } from '@/api/types'
import { useI18n } from 'vue-i18n'
import JSZip from 'jszip'
import ePub, { type Book as EpubBook, type Rendition } from 'epubjs'
import {
  ArrowLeft,
  Settings,
  List,
  Type,
  ChevronLeft,
  ChevronRight,
  Loader2,
  X
} from 'lucide-vue-next'
import ThemeToggle from '@/components/common/ThemeToggle.vue'

const props = defineProps<{
  bookId: string
}>()

const router = useRouter()
const { t } = useI18n()

// Состояние книги и ридера
const book = ref<Book | null>(null)
const loading = ref(true)
const loadingText = ref('Загрузка книги...')
const activeFormat = ref<'fb2' | 'epub' | 'unsupported'>('fb2')

// Содержание и навигация
interface TocItem {
  id: string
  title: string
}
const toc = ref<TocItem[]>([])
const showToc = ref(false)
const showSettings = ref(false)

// FB2 контент
const fb2Html = ref('')
const readerContainer = ref<HTMLElement | null>(null)

// EPUB
let epubBook: EpubBook | null = null
let epubRendition: Rendition | null = null
const epubViewerRef = ref<HTMLElement | null>(null)

// Настройки отображения
const fontSize = ref(Number(localStorage.getItem('boyan_reader_font_size')) || 18)
const fontFamily = ref(localStorage.getItem('boyan_reader_font_family') || 'font-serif')
const lineHeight = ref(localStorage.getItem('boyan_reader_line_height') || '1.7')
const columnWidth = ref(localStorage.getItem('boyan_reader_width') || '760px')

// Прогресс
const readingProgress = ref(0)
let saveProgressTimeout: ReturnType<typeof setTimeout> | null = null

onMounted(async () => {
  try {
    book.value = await api.get<Book>(`/api/v1/books/${props.bookId}`)
    await loadInitialProgress()
    await initReader()
  } catch (err) {
    console.error('Failed to init book reader:', err)
  } finally {
    loading.value = false
  }
})

onUnmounted(() => {
  if (saveProgressTimeout) clearTimeout(saveProgressTimeout)
  if (epubRendition) {
    epubRendition.destroy()
  }
})

async function loadInitialProgress() {
  try {
    const p = await api.get<ReadProgress>(`/api/v1/books/${props.bookId}/progress`)
    if (p && p.progress_percent > 0) {
      readingProgress.value = p.progress_percent
    }
  } catch {
    // Пользователь может быть не авторизован
  }
}

async function initReader() {
  if (!book.value?.files || book.value.files.length === 0) {
    activeFormat.value = 'unsupported'
    return
  }

  // Приоритет форматов: fb2.zip -> fb2 -> epub
  const files = book.value.files
  const hasFb2Zip = files.some(f => f.format === 'fb2.zip')
  const hasFb2 = files.some(f => f.format === 'fb2')
  const hasEpub = files.some(f => f.format === 'epub')

  if (hasFb2Zip) {
    activeFormat.value = 'fb2'
    await loadFb2Book(true)
  } else if (hasFb2) {
    activeFormat.value = 'fb2'
    await loadFb2Book(false)
  } else if (hasEpub) {
    activeFormat.value = 'epub'
    await loadEpubBook()
  } else {
    activeFormat.value = 'unsupported'
  }
}

async function loadFb2Book(isZip: boolean) {
  loadingText.value = 'Загрузка и парсинг FB2...'
  const fmt = isZip ? 'fb2.zip' : 'fb2'
  const url = api.getDownloadUrl(props.bookId, fmt)

  const response = await fetch(url)
  const buffer = await response.arrayBuffer()

  let fb2XmlText = ''
  if (isZip) {
    const zip = await JSZip.loadAsync(buffer)
    // Ищем файл с расширением .fb2
    let entryName = ''
    zip.forEach((path, file) => {
      if (!file.dir && path.toLowerCase().endsWith('.fb2') && !entryName) {
        entryName = path
      }
    })
    if (!entryName) throw new Error('No .fb2 file in archive')
    const uint8 = await zip.file(entryName)!.async('uint8array')
    fb2XmlText = decodeXmlBytes(uint8)
  } else {
    fb2XmlText = decodeXmlBytes(new Uint8Array(buffer))
  }

  renderFb2(fb2XmlText)
}

function decodeXmlBytes(bytes: Uint8Array): string {
  // Проверяем XML заголовок на кодировку (windows-1251 vs utf-8)
  const headerPreview = new TextDecoder('ascii').decode(bytes.slice(0, 100))
  let encoding = 'utf-8'
  const match = headerPreview.match(/encoding=["']([^"']+)["']/i)
  if (match && match[1]) {
    encoding = match[1].toLowerCase()
  }

  try {
    return new TextDecoder(encoding).decode(bytes)
  } catch {
    return new TextDecoder('utf-8').decode(bytes)
  }
}

function renderFb2(xmlText: string) {
  const parser = new DOMParser()
  const doc = parser.parseFromString(xmlText, 'text/xml')

  const bodies = doc.querySelectorAll('body')
  let mainBody: Element | null = null
  for (let i = 0; i < bodies.length; i++) {
    if (!bodies[i].getAttribute('name')) {
      mainBody = bodies[i]
      break
    }
  }
  if (!mainBody && bodies.length > 0) mainBody = bodies[0]
  if (!mainBody) {
    fb2Html.value = '<p>Не удалось разобрать содержимое книги.</p>'
    return
  }

  // Извлекаем встроенные изображения из <binary>
  const binaries: Record<string, string> = {}
  doc.querySelectorAll('binary').forEach(bin => {
    const id = bin.getAttribute('id')
    const ctype = bin.getAttribute('content-type') || 'image/jpeg'
    const raw = bin.textContent?.replace(/\s+/g, '') || ''
    if (id && raw) {
      binaries[id] = `data:${ctype};base64,${raw}`
    }
  })

  // Парсинг секций и заголовков
  const tocList: TocItem[] = []
  let sectionIndex = 0

  function processNode(node: Node): string {
    if (node.nodeType === Node.TEXT_NODE) {
      return node.textContent || ''
    }

    if (node.nodeType !== Node.ELEMENT_NODE) return ''
    const el = node as Element
    const tag = el.tagName.toLowerCase()

    if (tag === 'section') {
      sectionIndex++
      const secId = `section-${sectionIndex}`
      let inner = ''
      el.childNodes.forEach(child => {
        inner += processNode(child)
      })
      return `<section id="${secId}" class="my-8">${inner}</section>`
    }

    if (tag === 'title') {
      let titleText = ''
      el.childNodes.forEach(child => {
        titleText += processNode(child)
      })
      const cleanTitle = el.textContent?.trim() || `Глава ${sectionIndex}`
      tocList.push({ id: `section-${sectionIndex}`, title: cleanTitle })
      return `<h2 class="text-xl font-bold my-4 text-center">${titleText}</h2>`
    }

    if (tag === 'p') {
      let pText = ''
      el.childNodes.forEach(child => {
        pText += processNode(child)
      })
      return `<p class="my-3 text-justify indent-6">${pText}</p>`
    }

    if (tag === 'strong') {
      return `<strong>${el.textContent}</strong>`
    }

    if (tag === 'emphasis') {
      return `<em>${el.textContent}</em>`
    }

    if (tag === 'image') {
      const href = (el.getAttribute('l:href') || el.getAttribute('xlink:href') || el.getAttribute('href') || '').replace(/^#/, '')
      if (href && binaries[href]) {
        return `<img src="${binaries[href]}" alt="illustration" class="max-w-full mx-auto my-4 rounded-lg shadow" />`
      }
      return ''
    }

    let out = ''
    el.childNodes.forEach(c => {
      out += processNode(c)
    })
    return out
  }

  let fullHtml = ''
  mainBody.childNodes.forEach(c => {
    fullHtml += processNode(c)
  })

  fb2Html.value = fullHtml
  toc.value = tocList

  // Восстановление позиции чтения FB2
  setTimeout(() => {
    if (readerContainer.value && readingProgress.value > 0) {
      const maxScroll = readerContainer.value.scrollHeight - readerContainer.value.clientHeight
      readerContainer.value.scrollTop = (readingProgress.value / 100) * maxScroll
    }
  }, 100)
}

async function loadEpubBook() {
  loadingText.value = 'Загрузка и рендеринг EPUB...'
  const url = api.getDownloadUrl(props.bookId, 'epub')

  epubBook = ePub(url)
  await epubBook.ready

  const nav = await epubBook.loaded.navigation
  toc.value = nav.toc.map((item: any) => ({
    id: item.href,
    title: item.label.trim()
  }))

  if (epubViewerRef.value) {
    epubRendition = epubBook.renderTo(epubViewerRef.value, {
      width: '100%',
      height: '100%',
      spread: 'none'
    })

    await epubRendition.display()

    epubRendition.on('relocated', (location: any) => {
      if (location && location.start && location.start.percentage) {
        const pct = location.start.percentage * 100
        handleProgressUpdate(pct, location.start.cfi)
      }
    })
  }
}

function handleFb2Scroll() {
  if (!readerContainer.value) return
  const el = readerContainer.value
  const maxScroll = el.scrollHeight - el.clientHeight
  if (maxScroll <= 0) return

  const pct = Math.min(100, Math.max(0, (el.scrollTop / maxScroll) * 100))
  handleProgressUpdate(pct, `scroll-${el.scrollTop}`)
}

function handleProgressUpdate(pct: number, position: string) {
  readingProgress.value = Math.round(pct)

  if (saveProgressTimeout) clearTimeout(saveProgressTimeout)
  saveProgressTimeout = setTimeout(async () => {
    try {
      await api.put(`/api/v1/books/${props.bookId}/progress`, {
        format: activeFormat.value,
        progress_percent: pct,
        position
      })
    } catch {
      // Игнорируем сетевые ошибки сохранения позиции
    }
  }, 1000)
}

function scrollToChapter(id: string) {
  showToc.value = false
  if (activeFormat.value === 'fb2') {
    const el = document.getElementById(id)
    if (el) {
      el.scrollIntoView({ behavior: 'smooth' })
    }
  } else if (epubRendition) {
    epubRendition.display(id)
  }
}

function changeFontSize(delta: number) {
  const n = fontSize.value + delta
  if (n >= 12 && n <= 36) {
    fontSize.value = n
    localStorage.setItem('boyan_reader_font_size', String(n))
  }
}

function setFontFamily(f: string) {
  fontFamily.value = f
  localStorage.setItem('boyan_reader_font_family', f)
}

function setLineHeight(h: string) {
  lineHeight.value = h
  localStorage.setItem('boyan_reader_line_height', h)
}

function setWidth(w: string) {
  columnWidth.value = w
  localStorage.setItem('boyan_reader_width', w)
}
</script>

<template>
  <div class="fixed inset-0 z-50 flex flex-col bg-bg-primary text-fg-primary select-text">
    <!-- Верхняя плавающая панель навигации ридера -->
    <header class="h-14 px-4 border-b border-border bg-bg-surface/90 backdrop-blur-md flex items-center justify-between gap-4 shrink-0">
      <div class="flex items-center gap-3">
        <button
          @click="router.back()"
          class="p-2 rounded-xl border border-border hover:bg-bg-hover text-fg-secondary hover:text-fg-primary transition-colors"
          :title="t('reader.back_to_catalog')"
        >
          <ArrowLeft class="w-5 h-5" />
        </button>
        <div class="truncate max-w-sm sm:max-w-md md:max-w-lg">
          <h1 class="text-sm font-bold text-fg-primary truncate">{{ book?.title }}</h1>
          <p class="text-[11px] text-fg-muted truncate">
            {{ book?.authors?.map(a => a.name).join(', ') }}
          </p>
        </div>
      </div>

      <!-- Действия читалки -->
      <div class="flex items-center gap-2">
        <!-- Индикатор прогресса -->
        <span class="text-xs font-semibold text-accent px-2 py-1 rounded-lg bg-bg-secondary border border-border">
          {{ readingProgress }}%
        </span>

        <!-- Содержание -->
        <button
          @click="showToc = !showToc"
          class="p-2 rounded-xl border border-border hover:bg-bg-hover text-fg-secondary hover:text-fg-primary transition-colors"
          :title="t('reader.toc')"
        >
          <List class="w-5 h-5" />
        </button>

        <!-- Настройки типографики -->
        <button
          @click="showSettings = !showSettings"
          class="p-2 rounded-xl border border-border hover:bg-bg-hover text-fg-secondary hover:text-fg-primary transition-colors"
          :title="t('reader.settings')"
        >
          <Settings class="w-5 h-5" />
        </button>

        <!-- Тема оформления -->
        <ThemeToggle />
      </div>
    </header>

    <!-- Основная область чтения -->
    <main class="flex-1 relative overflow-hidden flex">
      <!-- Лоадер -->
      <div v-if="loading" class="absolute inset-0 z-20 flex flex-col items-center justify-center bg-bg-primary gap-3">
        <Loader2 class="w-8 h-8 text-accent animate-spin" />
        <span class="text-sm text-fg-secondary">{{ loadingText }}</span>
      </div>

      <!-- Неподдерживаемый формат -->
      <div v-else-if="activeFormat === 'unsupported'" class="flex-1 flex flex-col items-center justify-center p-6 text-center">
        <p class="text-fg-secondary text-base mb-4">
          {{ t('reader.unsupported_format', { format: book?.files?.[0]?.format || 'N/A' }) }}
        </p>
        <button @click="router.back()" class="px-4 py-2 rounded-xl bg-accent text-white font-medium">
          {{ t('reader.back_to_catalog') }}
        </button>
      </div>

      <!-- FB2 рендерер -->
      <div
        v-else-if="activeFormat === 'fb2'"
        ref="readerContainer"
        @scroll="handleFb2Scroll"
        class="flex-1 overflow-y-auto px-4 sm:px-8 py-10 transition-colors"
      >
        <div
          class="reader-content"
          :class="[fontFamily]"
          :style="{
            '--reader-font-size': `${fontSize}px`,
            '--reader-line-height': lineHeight,
            '--reader-max-width': columnWidth
          }"
          v-html="fb2Html"
        ></div>
      </div>

      <!-- EPUB рендерер -->
      <div v-else-if="activeFormat === 'epub'" class="flex-1 relative overflow-hidden">
        <div ref="epubViewerRef" class="w-full h-full"></div>
      </div>

      <!-- Боковая панель Содержания (TOC) -->
      <transition
        enter-active-class="transition duration-200 ease-out"
        enter-from-class="transform -translate-x-full"
        enter-to-class="transform translate-x-0"
        leave-active-class="transition duration-150 ease-in"
        leave-from-class="transform translate-x-0"
        leave-to-class="transform -translate-x-full"
      >
        <div
          v-if="showToc"
          class="absolute inset-y-0 left-0 w-80 bg-bg-surface border-r border-border shadow-2xl z-30 flex flex-col"
        >
          <div class="h-14 px-4 border-b border-border flex items-center justify-between">
            <h3 class="font-bold text-sm text-fg-primary">{{ t('reader.toc') }}</h3>
            <button @click="showToc = false" class="p-1 text-fg-muted hover:text-fg-primary">
              <X class="w-5 h-5" />
            </button>
          </div>
          <div class="flex-1 overflow-y-auto p-2 space-y-1">
            <div v-if="toc.length === 0" class="p-4 text-xs text-fg-muted text-center">
              Оглавление недоступно
            </div>
            <button
              v-for="(item, idx) in toc"
              :key="idx"
              @click="scrollToChapter(item.id)"
              class="w-full text-left px-3 py-2 rounded-xl text-xs text-fg-secondary hover:text-fg-primary hover:bg-bg-hover truncate transition-colors"
            >
              {{ item.title }}
            </button>
          </div>
        </div>
      </transition>

      <!-- Выпадающая панель Настроек ридера -->
      <transition
        enter-active-class="transition duration-150 ease-out"
        enter-from-class="transform opacity-0 scale-95"
        enter-to-class="transform opacity-100 scale-100"
        leave-active-class="transition duration-100 ease-in"
        leave-from-class="transform opacity-100 scale-100"
        leave-to-class="transform opacity-0 scale-95"
      >
        <div
          v-if="showSettings"
          class="absolute top-4 right-4 w-72 bg-bg-surface border border-border shadow-2xl rounded-2xl p-4 z-30 space-y-4"
        >
          <div class="flex items-center justify-between border-b border-border pb-2">
            <h4 class="font-bold text-xs uppercase tracking-wider text-fg-primary">{{ t('reader.settings') }}</h4>
            <button @click="showSettings = false" class="text-fg-muted hover:text-fg-primary">
              <X class="w-4 h-4" />
            </button>
          </div>

          <!-- Гарнитура шрифта -->
          <div>
            <label class="text-xs text-fg-muted block mb-1.5">{{ t('reader.font_family') }}</label>
            <div class="grid grid-cols-3 gap-1 bg-bg-primary p-1 rounded-xl border border-border">
              <button
                @click="setFontFamily('font-serif')"
                class="py-1 text-xs rounded-lg transition-colors font-serif"
                :class="fontFamily === 'font-serif' ? 'bg-bg-surface text-accent font-bold shadow' : 'text-fg-secondary'"
              >
                Serif
              </button>
              <button
                @click="setFontFamily('font-sans')"
                class="py-1 text-xs rounded-lg transition-colors font-sans"
                :class="fontFamily === 'font-sans' ? 'bg-bg-surface text-accent font-bold shadow' : 'text-fg-secondary'"
              >
                Sans
              </button>
              <button
                @click="setFontFamily('font-mono')"
                class="py-1 text-xs rounded-lg transition-colors font-mono"
                :class="fontFamily === 'font-mono' ? 'bg-bg-surface text-accent font-bold shadow' : 'text-fg-secondary'"
              >
                Mono
              </button>
            </div>
          </div>

          <!-- Размер шрифта -->
          <div>
            <label class="text-xs text-fg-muted block mb-1.5">{{ t('reader.font_size') }}: {{ fontSize }}px</label>
            <div class="flex items-center gap-2">
              <button
                @click="changeFontSize(-2)"
                class="flex-1 py-1.5 rounded-xl border border-border bg-bg-primary hover:bg-bg-hover text-fg-primary font-bold text-sm"
              >
                A-
              </button>
              <button
                @click="changeFontSize(2)"
                class="flex-1 py-1.5 rounded-xl border border-border bg-bg-primary hover:bg-bg-hover text-fg-primary font-bold text-sm"
              >
                A+
              </button>
            </div>
          </div>

          <!-- Ширина колонки -->
          <div>
            <label class="text-xs text-fg-muted block mb-1.5">{{ t('reader.column_width') }}</label>
            <div class="grid grid-cols-3 gap-1 bg-bg-primary p-1 rounded-xl border border-border">
              <button
                @click="setWidth('600px')"
                class="py-1 text-xs rounded-lg transition-colors"
                :class="columnWidth === '600px' ? 'bg-bg-surface text-accent font-bold shadow' : 'text-fg-secondary'"
              >
                Узкая
              </button>
              <button
                @click="setWidth('760px')"
                class="py-1 text-xs rounded-lg transition-colors"
                :class="columnWidth === '760px' ? 'bg-bg-surface text-accent font-bold shadow' : 'text-fg-secondary'"
              >
                Средняя
              </button>
              <button
                @click="setWidth('960px')"
                class="py-1 text-xs rounded-lg transition-colors"
                :class="columnWidth === '960px' ? 'bg-bg-surface text-accent font-bold shadow' : 'text-fg-secondary'"
              >
                Широкая
              </button>
            </div>
          </div>
        </div>
      </transition>
    </main>
  </div>
</template>
