<template>
  <nav
    class="fixed bottom-0 left-0 right-0 z-30 bg-theme-bg/95 backdrop-blur border-t border-theme pb-safe"
  >
    <div class="flex items-center justify-around h-14 max-w-lg mx-auto">
      <!-- Catalog -->
      <router-link
        to="/"
        class="flex flex-col items-center justify-center flex-1 h-full py-1 text-xs transition-colors"
        :class="isActive('/') ? 'text-primary-600 font-semibold' : 'text-theme-muted hover:text-theme-text'"
      >
        <BookOpen class="w-5 h-5 mb-0.5" />
        <span>{{ $t('nav.catalog') }}</span>
      </router-link>

      <!-- Search -->
      <router-link
        to="/search"
        class="flex flex-col items-center justify-center flex-1 h-full py-1 text-xs transition-colors"
        :class="isActive('/search') ? 'text-primary-600 font-semibold' : 'text-theme-muted hover:text-theme-text'"
      >
        <Search class="w-5 h-5 mb-0.5" />
        <span>{{ $t('nav.search') }}</span>
      </router-link>

      <!-- Shelves -->
      <router-link
        to="/shelves"
        class="flex flex-col items-center justify-center flex-1 h-full py-1 text-xs transition-colors"
        :class="isActive('/shelves') ? 'text-primary-600 font-semibold' : 'text-theme-muted hover:text-theme-text'"
      >
        <Bookmark class="w-5 h-5 mb-0.5" />
        <span>{{ $t('nav.shelves') }}</span>
      </router-link>

      <!-- Offline -->
      <router-link
        to="/offline"
        class="relative flex flex-col items-center justify-center flex-1 h-full py-1 text-xs transition-colors"
        :class="isActive('/offline') ? 'text-primary-600 font-semibold' : 'text-theme-muted hover:text-theme-text'"
      >
        <DownloadCloud class="w-5 h-5 mb-0.5" />
        <span>{{ $t('nav.offline') }}</span>
        <!-- Count Badge -->
        <span
          v-if="offlineStore.count > 0"
          class="absolute top-1 right-[20%] min-w-4 h-4 px-1 text-[10px] font-bold text-white bg-primary-600 rounded-full flex items-center justify-center border-2 border-theme-bg"
        >
          {{ offlineStore.count }}
        </span>
      </router-link>

      <!-- Settings -->
      <router-link
        to="/settings"
        class="flex flex-col items-center justify-center flex-1 h-full py-1 text-xs transition-colors"
        :class="isActive('/settings') ? 'text-primary-600 font-semibold' : 'text-theme-muted hover:text-theme-text'"
      >
        <Settings class="w-5 h-5 mb-0.5" />
        <span>{{ $t('nav.settings') }}</span>
      </router-link>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router'
import { BookOpen, Search, Bookmark, DownloadCloud, Settings } from 'lucide-vue-next'
import { useOfflineStore } from '@/stores/offline'

const route = useRoute()
const offlineStore = useOfflineStore()

function isActive(path: string) {
  if (path === '/') return route.path === '/'
  return route.path.startsWith(path)
}
</script>
