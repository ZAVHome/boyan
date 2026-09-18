<template>
  <div
    class="fixed inset-0 z-40 flex flex-col bg-theme-bg select-none overflow-hidden"
    :class="themeStore.currentTheme"
  >
    <!-- Top Header Overlay (collapsible) -->
    <transition name="fade">
      <header
        v-if="showOverlay"
        class="absolute top-0 left-0 right-0 z-50 flex items-center justify-between px-4 py-2 bg-theme-bg/95 backdrop-blur border-b border-theme pt-safe shadow-sm"
      >
        <div class="flex items-center gap-2">
          <button
            @click="goBack"
            class="p-2 -ml-2 rounded-full text-theme-muted hover:text-theme-text"
            :aria-label="$t('common.back')"
          >
            <ArrowLeft class="w-5 h-5" />
          </button>
          <div class="min-w-0">
            <h1 class="text-sm font-bold text-theme-text truncate max-w-[200px]">
              {{ book?.title || offlineBook?.title || $t('reader.loading') }}
            </h1>
            <p class="text-[11px] text-theme-muted truncate">
              {{ readingProgress }}% {{ isOfflineSource ? `• ${$t('common.offline')}` : '' }}
            </p>
          </div>
        </div>

        <div class="flex items-center gap-1">
          <!-- Contents / TOC button -->
          <button
            v-if="toc.length > 0"
            @click="showToc = !showToc"
            class="p-2 rounded-lg text-theme-muted hover:text-theme-text"
            :title="$t('reader.contents')"
          >
            <List class="w-5 h-5" />
          </button>

          <!-- Settings button -->
          <button
            @click="showSettings = !showSettings"
            class="p-2 rounded-lg text-theme-muted hover:text-theme-text"
            :title="$t('reader.settings')"
          >
            <Type class="w-5 h-5" />
          </button>
        </div>
      </header>
    </transition>

    <!-- Loading State -->
    <div v-if="loading" class="flex-1 flex flex-col items-center justify-center p-6 text-center">
      <Loader2 class="w-8 h-8 text-primary-600 animate-spin mb-3" />
      <p class="text-sm text-theme-muted">{{ loadingText }}</p>
    </div>

    <!-- Error State -->
    <div
      v-else-if="errorMessage"
      class="flex-1 flex flex-col items-center justify-center p-6 text-center"
    >
      <div class="max-w-xs p-5 rounded-2xl bg-theme-card border border-theme shadow-lg">
        <p class="text-sm font-semibold text-rose-600 mb-2">Не удалось открыть книгу</p>
        <p class="text-xs text-theme-muted mb-4">{{ errorMessage }}</p>
        <button
          @click="goBack"
          class="w-full py-2 rounded-xl bg-primary-600 text-white text-xs font-medium active:scale-95 transition-transform"
        >
          {{ $t('common.back') }}
        </button>
      </div>
    </div>

    <!-- Error / Unsupported State -->
    <div
      v-else-if="activeFormat === 'unsupported'"
      class="flex-1 flex flex-col items-center justify-center p-6 text-center"
    >
      <p class="text-sm font-semibold text-rose-600 mb-2">Формат книги не поддерживается</p>
      <p class="text-xs text-theme-muted mb-4">Для чтения в веб-приложении доступны FB2 и EPUB.</p>
      <button
        @click="goBack"
        class="px-4 py-2 rounded-lg bg-theme-card border border-theme text-xs font-medium text-theme-text"
      >
        {{ $t('common.back') }}
      </button>
    </div>

    <!-- Reading Area with Touch Gestures and Tap Zones -->
    <div
      v-else
      class="relative flex-1 w-full h-full overflow-hidden"
      @touchstart="handleTouchStart"
      @touchend="handleTouchEnd"
    >
      <!-- EPUB Viewer -->
      <div
        v-show="activeFormat === 'epub'"
        ref="epubViewerRef"
        class="w-full h-full"
      ></div>

      <!-- FB2 HTML Reader -->
      <div
        v-show="activeFormat === 'fb2'"
        ref="readerContainer"
        @scroll="handleFb2Scroll"
        class="w-full h-full overflow-y-auto px-4 py-8 max-w-xl mx-auto reading-content font-serif"
        :style="{
          fontSize: `${fontSize}px`,
          lineHeight: lineHeight,
          fontFamily: fontFamily === 'sans' ? 'sans-serif' : 'serif'
        }"
        v-html="fb2Html"
      ></div>

      <!-- Invisible Touch Tap Zones -->
      <!-- Left 25% Tap Zone (Previous Page) -->
      <div
        class="absolute top-0 bottom-0 left-0 w-1/4 z-10"
        @click.stop="prevPage"
      ></div>

      <!-- Center 50% Tap Zone (Toggle Controls) -->
      <div
        class="absolute top-0 bottom-0 left-1/4 w-2/4 z-10"
        @click.stop="toggleOverlay"
      ></div>

      <!-- Right 25% Tap Zone (Next Page) -->
      <div
        class="absolute top-0 bottom-0 right-0 w-1/4 z-10"
        @click.stop="nextPage"
      ></div>
    </div>

    <!-- Bottom Progress Bar (collapsible) -->
    <transition name="fade">
      <footer
        v-if="showOverlay"
        class="absolute bottom-0 left-0 right-0 z-50 flex items-center justify-between px-4 py-2.5 bg-theme-bg/95 backdrop-blur border-t border-theme pb-safe shadow-sm"
      >
        <button
          @click="prevPage"
          class="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-theme-card border border-theme text-xs font-medium text-theme-text active:scale-95"
        >
          <ChevronLeft class="w-4 h-4" />
          <span>{{ $t('reader.prev') }}</span>
        </button>

        <div class="flex flex-col items-center">
          <span class="text-xs font-semibold text-theme-text">{{ readingProgress }}%</span>
          <span class="text-[10px] text-theme-muted">{{ $t('reader.tap_zones_hint') }}</span>
        </div>

        <button
          @click="nextPage"
          class="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-theme-card border border-theme text-xs font-medium text-theme-text active:scale-95"
        >
          <span>{{ $t('reader.next') }}</span>
          <ChevronRight class="w-4 h-4" />
        </button>
      </footer>
    </transition>

    <!-- Table of Contents Sheet -->
    <div
      v-if="showToc"
      class="fixed inset-0 z-50 flex items-end justify-center bg-black/60 backdrop-blur-sm"
      @click="showToc = false"
    >
      <div
        class="relative w-full max-w-md max-h-[75vh] bg-theme-bg rounded-t-2xl border-t border-theme p-4 flex flex-col pb-safe shadow-2xl"
        @click.stop
      >
        <div class="flex items-center justify-between pb-3 border-b border-theme mb-2">
          <h3 class="text-sm font-bold text-theme-text">{{ $t('reader.contents') }}</h3>
          <button @click="showToc = false" class="p-1 text-theme-muted hover:text-theme-text">
            <X class="w-5 h-5" />
          </button>
        </div>

        <div class="overflow-y-auto space-y-1 flex-1">
          <button
            v-for="(item, idx) in toc"
            :key="idx"
            @click="goToChapter(item.id)"
            class="w-full text-left px-3 py-2.5 rounded-lg text-xs hover:bg-theme-card text-theme-text transition-colors border-b border-theme/40"
          >
            {{ item.title }}
          </button>
        </div>
      </div>
    </div>

    <!-- Reading Settings Sheet -->
    <div
      v-if="showSettings"
      class="fixed inset-0 z-50 flex items-end justify-center bg-black/60 backdrop-blur-sm"
      @click="showSettings = false"
    >
      <div
        class="relative w-full max-w-md bg-theme-bg rounded-t-2xl border-t border-theme p-4 space-y-4 pb-safe shadow-2xl"
        @click.stop
      >
        <div class="flex items-center justify-between pb-2 border-b border-theme">
          <h3 class="text-sm font-bold text-theme-text">{{ $t('reader.settings') }}</h3>
          <button @click="showSettings = false" class="p-1 text-theme-muted hover:text-theme-text">
            <X class="w-5 h-5" />
          </button>
        </div>

        <!-- Font size -->
        <div class="space-y-1.5">
          <div class="flex justify-between text-xs text-theme-muted font-medium">
            <span>{{ $t('reader.font_size') }}</span>
            <span>{{ fontSize }}px</span>
          </div>
          <div class="flex items-center gap-3">
            <button
              @click="changeFontSize(-1)"
              class="w-9 h-9 rounded-lg bg-theme-card border border-theme flex items-center justify-center text-sm font-bold text-theme-text"
            >
              A-
            </button>
            <input
              type="range"
              min="14"
              max="28"
              v-model.number="fontSize"
              class="flex-1 accent-primary-600"
            />
            <button
              @click="changeFontSize(1)"
              class="w-9 h-9 rounded-lg bg-theme-card border border-theme flex items-center justify-center text-base font-bold text-theme-text"
            >
              A+
            </button>
          </div>
        </div>

        <!-- Line height -->
        <div class="space-y-1.5">
          <div class="text-xs text-theme-muted font-medium">{{ $t('reader.line_height') }}</div>
          <div class="grid grid-cols-3 gap-2">
            <button
              v-for="lh in ['1.4', '1.6', '1.8']"
              :key="lh"
              @click="lineHeight = lh"
              :class="[
                'py-1.5 text-xs rounded-lg border font-mono transition-colors',
                lineHeight === lh
                  ? 'bg-primary-600 text-white border-primary-600'
                  : 'bg-theme-card text-theme-text border-theme'
              ]"
            >
              {{ lh }}
            </button>
          </div>
        </div>

        <!-- Font Family -->
        <div class="space-y-1.5">
          <div class="text-xs text-theme-muted font-medium">{{ $t('reader.font_family') }}</div>
          <div class="grid grid-cols-2 gap-2">
            <button
              @click="fontFamily = 'serif'"
              :class="[
                'py-2 px-3 text-xs rounded-lg border font-serif transition-colors',
                fontFamily === 'serif'
                  ? 'bg-primary-600 text-white border-primary-600'
                  : 'bg-theme-card text-theme-text border-theme'
              ]"
            >
              {{ $t('reader.serif') }}
            </button>
            <button
              @click="fontFamily = 'sans'"
              :class="[
                'py-2 px-3 text-xs rounded-lg border font-sans transition-colors',
                fontFamily === 'sans'
                  ? 'bg-primary-600 text-white border-primary-600'
                  : 'bg-theme-card text-theme-text border-theme'
              ]"
            >
              {{ $t('reader.sans') }}
            </button>
          </div>
        </div>

        <!-- Theme Switcher -->
        <div class="space-y-1.5 pt-2 border-t border-theme">
          <div class="text-xs text-theme-muted font-medium">{{ $t('settings.theme') }}</div>
          <div class="grid grid-cols-5 gap-1.5">
            <button
              v-for="tName in (['light', 'dark', 'oled', 'sepia', 'eink'] as const)"
              :key="tName"
              @click="themeStore.setTheme(tName)"
              :class="[
                'py-1.5 text-[11px] font-medium rounded-lg border text-center transition-colors',
                themeStore.currentTheme === tName
                  ? 'ring-2 ring-primary-600 border-primary-600'
                  : 'border-theme'
              ]"
              :style="{
                background: tName === 'light' ? '#ffffff' : tName === 'dark' ? '#1e293b' : tName === 'oled' ? '#000000' : tName === 'sepia' ? '#f4ecd8' : '#ffffff',
                color: tName === 'light' ? '#0f172a' : tName === 'dark' ? '#f8fafc' : tName === 'oled' ? '#ffffff' : tName === 'sepia' ? '#5b4636' : '#000000'
              }"
            >
              {{ tName.toUpperCase() }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import JSZip from 'jszip'
import ePub, { type Book as EpubBook, type Rendition } from 'epubjs'
import {
  ArrowLeft,
  List,
  Type,
  ChevronLeft,
  ChevronRight,
  Loader2,
  X
} from 'lucide-vue-next'
import { api } from '@/api/client'
import type { Book, ReadProgress } from '@/api/types'
import { getOfflineBook, updateOfflineProgress, type OfflineBook } from '@/db/offline'
import { useThemeStore } from '@/stores/theme'

const props = defineProps<{
  bookId: string
}>()

const router = useRouter()
const route = useRoute()
const themeStore = useThemeStore()

// State
const book = ref<Book | null>(null)
const offlineBook = ref<OfflineBook | null>(null)
const isOfflineSource = ref(false)
const loading = ref(true)
const loadingText = ref('Загрузка книги...')
const errorMessage = ref('')
const activeFormat = ref<'fb2' | 'epub' | 'unsupported'>('fb2')

const authorNames = computed(() => {
  if (book.value?.authors && book.value.authors.length > 0) {
    return book.value.authors.map((a) => a.name).join(', ')
  }
  if (offlineBook.value?.author) {
    return offlineBook.value.author
  }
  return ''
})

// Overlays
const showOverlay = ref(true)
const showToc = ref(false)
const showSettings = ref(false)

// Reader Display Settings
const fontSize = ref(Number(localStorage.getItem('boyan_m_font_size')) || 17)
const lineHeight = ref(localStorage.getItem('boyan_m_line_height') || '1.6')
const fontFamily = ref(localStorage.getItem('boyan_m_font_family') || 'serif')

watch(fontSize, (v) => localStorage.setItem('boyan_m_font_size', String(v)))
watch(lineHeight, (v) => localStorage.setItem('boyan_m_line_height', v))
watch(fontFamily, (v) => localStorage.setItem('boyan_m_font_family', v))

function changeFontSize(delta: number) {
  const next = fontSize.value + delta
  if (next >= 13 && next <= 32) {
    fontSize.value = next
  }
}

// FB2
const fb2Html = ref('')
const readerContainer = ref<HTMLElement | null>(null)

// EPUB
let epubBook: EpubBook | null = null
let epubRendition: Rendition | null = null
const epubViewerRef = ref<HTMLElement | null>(null)

// TOC & Progress
interface TocItem {
  id: string
  title: string
}
const toc = ref<TocItem[]>([])
const readingProgress = ref(0)
let saveProgressTimeout: ReturnType<typeof setTimeout> | null = null

// Touch Gestures
let touchStartX = 0
let touchStartY = 0

function handleTouchStart(e: TouchEvent) {
  touchStartX = e.changedTouches[0].screenX
  touchStartY = e.changedTouches[0].screenY
}

function handleTouchEnd(e: TouchEvent) {
  const diffX = e.changedTouches[0].screenX - touchStartX
  const diffY = e.changedTouches[0].screenY - touchStartY

  // Only trigger horizontal swipe if movement is predominantly horizontal
  if (Math.abs(diffX) > 60 && Math.abs(diffY) < 50) {
    if (diffX > 0) {
      prevPage()
    } else {
      nextPage()
    }
  }
}

function toggleOverlay() {
  showOverlay.value = !showOverlay.value
}

function prevPage() {
  if (activeFormat.value === 'epub' && epubRendition) {
    epubRendition.prev()
  } else if (activeFormat.value === 'fb2' && readerContainer.value) {
    const pageHeight = window.innerHeight * 0.85
    const behavior = themeStore.isEInk ? 'auto' : 'smooth'
    readerContainer.value.scrollBy({ top: -pageHeight, behavior })
  }
}

function nextPage() {
  if (activeFormat.value === 'epub' && epubRendition) {
    epubRendition.next()
  } else if (activeFormat.value === 'fb2' && readerContainer.value) {
    const pageHeight = window.innerHeight * 0.85
    const behavior = themeStore.isEInk ? 'auto' : 'smooth'
    readerContainer.value.scrollBy({ top: pageHeight, behavior })
  }
}

function goBack() {
  router.back()
}

onMounted(async () => {
  try {
    // 1. Check if saved offline in IndexedDB
    const cached = await getOfflineBook(props.bookId)
    if (cached) {
      offlineBook.value = cached
      isOfflineSource.value = true
      if (cached.progress) {
        readingProgress.value = cached.progress.percentage
      }
    }

    // 2. If online, try to fetch book details and server progress
    if (navigator.onLine && !isOfflineSource.value) {
      try {
        book.value = await api.get<Book>(`/api/v1/books/${props.bookId}`)
        const p = await api.get<ReadProgress>(`/api/v1/books/${props.bookId}/progress`)
        if (p && p.progress_percent > 0) {
          readingProgress.value = p.progress_percent
        }
      } catch {
        // Fallback to offline if request fails
      }
    }

    // 3. Initialize Reader
    await initReader()
  } catch (err: any) {
    console.error('Failed to init mobile reader:', err)
    errorMessage.value = err?.message || 'Не удалось открыть книгу'
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

async function initReader() {
  const preferredFormat = (route.query.format as string)?.toLowerCase()

  // Case A: Offline source
  if (offlineBook.value) {
    const fmt = offlineBook.value.format.toLowerCase()
    if (fmt.includes('epub')) {
      activeFormat.value = 'epub'
      await loadEpubFromBlob(offlineBook.value.blob)
    } else {
      activeFormat.value = 'fb2'
      const isZip = fmt.includes('zip')
      const buffer = await offlineBook.value.blob.arrayBuffer()
      await parseAndRenderFb2Buffer(buffer, isZip)
    }
    return
  }

  // Case B: Online source
  if (!book.value?.files || book.value.files.length === 0) {
    activeFormat.value = 'unsupported'
    return
  }

  const files = book.value.files
  const hasFb2Zip = files.some((f) => f.format === 'fb2.zip')
  const hasFb2 = files.some((f) => f.format === 'fb2')
  const hasEpub = files.some((f) => f.format === 'epub')

  if (preferredFormat === 'epub' && hasEpub) {
    activeFormat.value = 'epub'
    await loadEpubOnline()
  } else if (hasFb2Zip) {
    activeFormat.value = 'fb2'
    await loadFb2Online(true)
  } else if (hasFb2) {
    activeFormat.value = 'fb2'
    await loadFb2Online(false)
  } else if (hasEpub) {
    activeFormat.value = 'epub'
    await loadEpubOnline()
  } else {
    activeFormat.value = 'unsupported'
  }
}

async function loadFb2Online(isZip: boolean) {
  loadingText.value = 'Загрузка и парсинг FB2...'
  const fmt = isZip ? 'fb2.zip' : 'fb2'
  const url = api.getDownloadUrl(props.bookId, fmt)
  const response = await fetch(url)
  if (!response.ok) {
    let msg = `Не удалось загрузить книгу (${response.status})`
    try {
      const errData = await response.json()
      if (errData.error) msg = errData.error
    } catch {
      // response is not json
    }
    throw new Error(msg)
  }
  const buffer = await response.arrayBuffer()
  await parseAndRenderFb2Buffer(buffer, isZip)
}

async function parseAndRenderFb2Buffer(buffer: ArrayBuffer, isZip: boolean) {
  let fb2XmlText = ''
  if (isZip) {
    const zip = await JSZip.loadAsync(buffer)
    let entryName = ''
    zip.forEach((path, file) => {
      if (!file.dir && path.toLowerCase().endsWith('.fb2') && !entryName) {
        entryName = path
      }
    })
    if (!entryName) throw new Error('В архиве не найден файл .fb2')
    const uint8 = await zip.file(entryName)!.async('uint8array')
    fb2XmlText = decodeXmlBytes(uint8)
  } else {
    fb2XmlText = decodeXmlBytes(new Uint8Array(buffer))
  }

  renderFb2(fb2XmlText)
}

function decodeXmlBytes(bytes: Uint8Array): string {
  const headerPreview = new TextDecoder('ascii').decode(bytes.slice(0, 100))
  let encoding = 'utf-8'
  const match = headerPreview.match(/encoding=["']([^"']+)["']/i)
  if (match && match[1]) {
    encoding = match[1].toLowerCase()
  }
  let text = ''
  try {
    text = new TextDecoder(encoding).decode(bytes)
  } catch {
    text = new TextDecoder('utf-8').decode(bytes)
  }
  return sanitizeFb2Xml(text)
}

function sanitizeFb2Xml(xmlText: string): string {
  // 1. Очищаем пробелы и UTF-8 BOM (\uFEFF) перед <?xml
  xmlText = xmlText.trimStart()
  if (xmlText.charCodeAt(0) === 0xFEFF) {
    xmlText = xmlText.slice(1).trimStart()
  }

  // 2. Убеждаемся, что стандартные namespace FB2 объявлены на корневом теге FictionBook.
  // В книгах Calibre и старых редакторах часто используются атрибуты xlink:href или l:href
  // без соответствующего объявления xmlns:xlink или xmlns:l на корневом теге FictionBook,
  // из-за чего строгий XML-парсер браузера завершается с ошибкой 'Namespace prefix xlink/l is not defined'.
  xmlText = xmlText.replace(/<FictionBook([^>]*)>/i, (match, attrs) => {
    let newAttrs = attrs
    if (!newAttrs.includes('xmlns:xlink')) {
      newAttrs += ' xmlns:xlink="http://www.w3.org/1999/xlink"'
    }
    if (!newAttrs.includes('xmlns:l')) {
      newAttrs += ' xmlns:l="http://www.w3.org/1999/xlink"'
    }
    return `<FictionBook${newAttrs}>`
  })

  // 3. Заменяем именованные HTML-сущности на числовые XML entities, чтобы XML-парсер не падал
  xmlText = xmlText.replace(/&nbsp;/g, '&#160;')
                   .replace(/&copy;/g, '&#169;')
                   .replace(/&mdash;/g, '&#8212;')
                   .replace(/&ndash;/g, '&#8211;')
                   .replace(/&laquo;/g, '&#171;')
                   .replace(/&raquo;/g, '&#187;')
                   .replace(/&hellip;/g, '&#8230;')

  return xmlText
}

function renderFb2(xmlText: string) {
  const parser = new DOMParser()
  let doc: Document = parser.parseFromString(xmlText, 'text/xml')
  let parserError = doc.querySelector('parsererror')

  // Толерантный Fallback: если строгий XML-парсер споткнулся на невалидном XML (синтаксические ошибки и т.д.),
  // используем HTML-парсер браузера, который толерантен к любым погрешностям разметки
  if (parserError) {
    console.warn('Strict XML parsing failed, attempting tolerant HTML fallback parser:', parserError.textContent)
    const htmlDoc = parser.parseFromString(xmlText, 'text/html')
    if (htmlDoc.querySelector('body')) {
      doc = htmlDoc
      parserError = null
    }
  }

  if (parserError) {
    console.error('FB2 XML parse error:', parserError.textContent)
    errorMessage.value = 'Ошибка структуры книги (некорректный XML формат)'
    return
  }

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
    errorMessage.value = 'Не удалось разобрать содержимое книги.'
    return
  }

  // Binaries
  const binaries: Record<string, string> = {}
  doc.querySelectorAll('binary').forEach((bin) => {
    const id = bin.getAttribute('id')
    const ctype = bin.getAttribute('content-type') || 'image/jpeg'
    const raw = bin.textContent?.replace(/\s+/g, '') || ''
    if (id && raw) {
      binaries[id] = `data:${ctype};base64,${raw}`
    }
  })

  // TOC and node tree
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
      el.childNodes.forEach((child) => {
        inner += processNode(child)
      })
      return `<section id="${secId}" class="my-6">${inner}</section>`
    }

    if (tag === 'title') {
      let titleText = ''
      el.childNodes.forEach((child) => {
        titleText += processNode(child)
      })
      const cleanTitle = el.textContent?.trim() || `Раздел ${sectionIndex}`
      tocList.push({ id: `section-${sectionIndex}`, title: cleanTitle })
      return `<h2 class="text-lg font-bold my-4 text-center text-theme-text leading-snug">${titleText}</h2>`
    }

    if (tag === 'p') {
      let pText = ''
      el.childNodes.forEach((child) => {
        pText += processNode(child)
      })
      return `<p class="my-2.5 text-justify indent-5">${pText}</p>`
    }

    if (tag === 'strong') return `<strong>${el.textContent}</strong>`
    if (tag === 'emphasis') return `<em>${el.textContent}</em>`

    if (tag === 'image') {
      const href = (
        el.getAttribute('l:href') ||
        el.getAttribute('xlink:href') ||
        el.getAttribute('href') ||
        ''
      ).replace(/^#/, '')
      if (href && binaries[href]) {
        return `<img src="${binaries[href]}" alt="illustration" class="max-w-full mx-auto my-4 rounded-lg" />`
      }
      return ''
    }

    let out = ''
    el.childNodes.forEach((c) => {
      out += processNode(c)
    })
    return out
  }

  let fullHtml = ''

  // Титульная страница с обложкой книги
  let coverSrc = ''
  const coverEl = doc.querySelector('title-info coverpage image, coverpage image')
  const coverHref = (
    coverEl?.getAttribute('l:href') ||
    coverEl?.getAttribute('xlink:href') ||
    coverEl?.getAttribute('href') ||
    ''
  ).replace(/^#/, '')

  if (coverHref && binaries[coverHref]) {
    coverSrc = binaries[coverHref]
  } else if (props.bookId) {
    coverSrc = api.getCoverUrl(props.bookId)
  }

  if (coverSrc) {
    const bookTitle = book.value?.title || offlineBook.value?.title || ''
    const bookAuthors = authorNames.value
    fullHtml += `
      <div class="text-center my-6 pb-6 border-b border-theme/60">
        <img src="${coverSrc}" alt="${bookTitle}" class="max-h-[50vh] max-w-[240px] mx-auto rounded-xl shadow-lg object-contain mb-4" />
        <h1 class="text-xl font-bold my-2 text-center text-theme-text leading-snug">${bookTitle}</h1>
        ${bookAuthors ? `<p class="text-xs text-theme-muted text-center">${bookAuthors}</p>` : ''}
      </div>
    `
  }

  mainBody.childNodes.forEach((c) => {
    fullHtml += processNode(c)
  })

  fb2Html.value = fullHtml
  toc.value = tocList

  // Restore scroll
  setTimeout(() => {
    if (readerContainer.value && readingProgress.value > 0) {
      const maxScroll = readerContainer.value.scrollHeight - readerContainer.value.clientHeight
      readerContainer.value.scrollTop = (readingProgress.value / 100) * maxScroll
    }
  }, 100)
}

function applyEpubTheme(theme: string) {
  if (!epubRendition) return
  const themesMap: Record<string, { body: Record<string, string> }> = {
    dark: { body: { background: '#0f172a !important', color: '#f8fafc !important' } },
    oled: { body: { background: '#000000 !important', color: '#ffffff !important' } },
    sepia: { body: { background: '#fbf0d9 !important', color: '#433422 !important' } },
    light: { body: { background: '#f8fafc !important', color: '#0f172a !important' } },
    eink: { body: { background: '#ffffff !important', color: '#000000 !important' } }
  }
  const rules = themesMap[theme] || themesMap.light
  epubRendition.themes.default(rules)
}

watch(
  () => themeStore.currentTheme,
  (newTheme) => {
    if (activeFormat.value === 'epub' && epubRendition) {
      applyEpubTheme(newTheme)
    }
  }
)

async function loadEpubOnline() {
  loadingText.value = 'Загрузка и рендеринг EPUB...'
  const url = api.getDownloadUrl(props.bookId, 'epub')
  const testRes = await fetch(url, { method: 'HEAD' })
  if (!testRes.ok) {
    let msg = `Файл EPUB недоступен (${testRes.status})`
    try {
      const getRes = await fetch(url)
      const data = await getRes.json()
      if (data.error) msg = data.error
    } catch {
      // ignore
    }
    throw new Error(msg)
  }
  await renderEpubFromSource(url)
}

async function loadEpubFromBlob(blob: Blob) {
  loadingText.value = 'Загрузка EPUB из памяти...'
  const buffer = await blob.arrayBuffer()
  await renderEpubFromSource(buffer)
}

async function renderEpubFromSource(source: string | ArrayBuffer) {
  epubBook = ePub(source as any)
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
    applyEpubTheme(themeStore.currentTheme)

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
    // 1. Update local offline DB
    await updateOfflineProgress(props.bookId, { position, percentage: Math.round(pct) })

    // 2. If online, sync with server
    if (navigator.onLine) {
      try {
        await api.put(`/api/v1/books/${props.bookId}/progress`, {
          format: activeFormat.value,
          progress_percent: pct,
          position
        })
      } catch {
        // Network sync silent fallback
      }
    }
  }, 1000)
}

function goToChapter(id: string) {
  showToc.value = false
  if (activeFormat.value === 'epub' && epubRendition) {
    epubRendition.display(id)
  } else if (activeFormat.value === 'fb2') {
    const el = document.getElementById(id)
    if (el) {
      el.scrollIntoView({ behavior: themeStore.isEInk ? 'auto' : 'smooth' })
    }
  }
}
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

[data-theme="eink"] .fade-enter-active,
[data-theme="eink"] .fade-leave-active {
  transition: none !important;
}
</style>
