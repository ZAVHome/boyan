<script setup lang="ts">
import { ref } from 'vue'
import { api } from '@/api/client'
import type { ProcessResult } from '@/api/types'
import { useI18n } from 'vue-i18n'
import { UploadCloud, CheckCircle2, AlertTriangle, X, Loader2 } from 'lucide-vue-next'

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'uploaded'): void
}>()

const { t } = useI18n()
const fileInputRef = ref<HTMLInputElement | null>(null)
const isDragging = ref(false)
const isUploading = ref(false)
const result = ref<ProcessResult | null>(null)
const errorMessage = ref('')

async function handleFiles(files: FileList | null) {
  if (!files || files.length === 0) return
  const file = files[0]

  isUploading.value = true
  result.value = null
  errorMessage.value = ''

  const formData = new FormData()
  formData.append('file', file)

  try {
    const res = await api.post<ProcessResult>('/api/v1/books/upload', formData)
    result.value = res
    emit('uploaded')
  } catch (err: any) {
    errorMessage.value = err.message || 'Upload failed'
  } finally {
    isUploading.value = false
  }
}

function onDrop(e: DragEvent) {
  isDragging.value = false
  handleFiles(e.dataTransfer?.files || null)
}

function onFileSelect(e: Event) {
  const input = e.target as HTMLInputElement
  handleFiles(input.files)
}
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm" @click.self="emit('close')">
    <div class="relative w-full max-w-lg rounded-2xl bg-bg-surface border border-border shadow-2xl p-6 space-y-5">
      <div class="flex items-center justify-between">
        <h3 class="text-lg font-bold text-fg-primary">
          {{ t('upload.title') }}
        </h3>
        <button
          @click="emit('close')"
          class="p-1.5 rounded-lg text-fg-secondary hover:text-fg-primary hover:bg-bg-hover transition-colors"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Зона Drag & Drop -->
      <div
        @dragover.prevent="isDragging = true"
        @dragleave.prevent="isDragging = false"
        @drop.prevent="onDrop"
        @click="fileInputRef?.click()"
        class="border-2 border-dashed rounded-2xl p-8 flex flex-col items-center justify-center text-center cursor-pointer transition-all"
        :class="isDragging ? 'border-accent bg-accent/5' : 'border-border hover:border-accent/60 hover:bg-bg-secondary/50'"
      >
        <input
          ref="fileInputRef"
          type="file"
          accept=".fb2,.zip,.epub"
          class="hidden"
          @change="onFileSelect"
        />

        <div class="w-14 h-14 rounded-2xl bg-accent/10 text-accent flex items-center justify-center mb-3">
          <Loader2 v-if="isUploading" class="w-7 h-7 animate-spin" />
          <UploadCloud v-else class="w-7 h-7" />
        </div>

        <p class="text-sm font-medium text-fg-primary max-w-xs">
          {{ isUploading ? t('upload.uploading') : t('upload.drop_zone') }}
        </p>
        <span class="text-xs text-fg-muted mt-2">
          Поддерживаются: .fb2, .fb2.zip, .epub
        </span>
      </div>

      <!-- Результат / Уведомления -->
      <div v-if="result" class="p-4 rounded-xl text-sm flex items-start gap-3" :class="{
        'bg-emerald-500/10 text-emerald-500 border border-emerald-500/30': result.status === 'imported',
        'bg-sky-500/10 text-sky-400 border border-sky-500/30': result.status === 'format_attached',
        'bg-amber-500/10 text-amber-500 border border-amber-500/30': result.status === 'quarantined'
      }">
        <CheckCircle2 v-if="result.status === 'imported' || result.status === 'format_attached'" class="w-5 h-5 shrink-0 mt-0.5" />
        <AlertTriangle v-else class="w-5 h-5 shrink-0 mt-0.5" />
        <div>
          <div class="font-semibold">
            <span v-if="result.status === 'imported'">{{ t('upload.success') }}</span>
            <span v-else-if="result.status === 'format_attached'">{{ t('upload.format_attached') }}</span>
            <span v-else>{{ t('upload.quarantined') }}</span>
          </div>
          <div v-if="result.message" class="text-xs opacity-90 mt-0.5">
            {{ result.message }}
          </div>
        </div>
      </div>

      <div v-if="errorMessage" class="p-4 rounded-xl text-sm bg-red-500/10 text-red-500 border border-red-500/30 flex items-center gap-2">
        <AlertTriangle class="w-5 h-5 shrink-0" />
        <span>{{ t('upload.error', { msg: errorMessage }) }}</span>
      </div>

      <div class="flex justify-end pt-2">
        <button
          @click="emit('close')"
          class="px-4 py-2 rounded-xl bg-bg-secondary hover:bg-bg-hover text-fg-primary text-sm font-medium transition-colors"
        >
          {{ t('upload.close') }}
        </button>
      </div>
    </div>
  </div>
</template>
