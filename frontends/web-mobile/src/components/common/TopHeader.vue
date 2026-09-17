<template>
  <header
    class="sticky top-0 z-30 flex items-center justify-between px-4 py-2.5 bg-theme-bg/95 backdrop-blur border-b border-theme pt-safe"
  >
    <div class="flex items-center gap-2">
      <button
        v-if="showBack"
        @click="onBack"
        class="p-1 -ml-1 text-theme-muted hover:text-theme-text rounded-md focus:outline-none"
        :aria-label="$t('common.back')"
      >
        <ArrowLeft class="w-5 h-5" />
      </button>

      <div class="flex items-center gap-2" v-if="!title">
        <img
          src="/logo.png"
          alt="Боян"
          class="w-7 h-7 rounded-full object-cover shadow-sm"
        />
        <span class="font-bold tracking-tight text-base text-theme-text">БОЯН</span>
      </div>

      <h1 v-else class="font-bold text-base text-theme-text truncate max-w-[200px]">
        {{ title }}
      </h1>
    </div>

    <div class="flex items-center gap-2">
      <!-- Network indicator if offline -->
      <span
        v-if="!isOnline"
        class="flex items-center gap-1 text-[11px] font-medium text-amber-600 dark:text-amber-400 bg-amber-500/10 px-2 py-0.5 rounded-full border border-amber-500/20"
      >
        <WifiOff class="w-3 h-3" />
        <span>{{ $t('common.offline') }}</span>
      </span>

      <EInkToggle />
    </div>
  </header>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowLeft, WifiOff } from 'lucide-vue-next'
import EInkToggle from '@/components/common/EInkToggle.vue'

defineProps<{
  title?: string
  showBack?: boolean
}>()

const router = useRouter()

function onBack() {
  if (window.history.length > 1) {
    router.back()
  } else {
    router.push('/')
  }
}

const isOnline = ref(navigator.onLine)

function handleOnline() {
  isOnline.value = true
}

function handleOffline() {
  isOnline.value = false
}

onMounted(() => {
  window.addEventListener('online', handleOnline)
  window.addEventListener('offline', handleOffline)
})

onUnmounted(() => {
  window.removeEventListener('online', handleOnline)
  window.removeEventListener('offline', handleOffline)
})
</script>
