<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useRoute } from 'vue-router'
import {
  BookCopy,
  Users,
  Layers,
  BookOpen,
  CheckCircle2,
  Bookmark,
  ShieldAlert,
  Compass,
  Rss,
  ChevronRight
} from 'lucide-vue-next'

const emit = defineEmits<{
  (e: 'openConnect'): void
}>()

const { t } = useI18n()
const authStore = useAuthStore()
const route = useRoute()

const mainNav = [
  { name: 'catalog', path: '/', labelKey: 'nav.catalog', icon: Compass },
  { name: 'authors', path: '/authors', labelKey: 'nav.authors', icon: Users },
  { name: 'series', path: '/series', labelKey: 'nav.series', icon: Layers },
]

const shelfNav = [
  { path: '/shelves/reading', type: 'reading', labelKey: 'nav.shelf_reading', icon: BookOpen },
  { path: '/shelves/finished', type: 'finished', labelKey: 'nav.shelf_finished', icon: CheckCircle2 },
  { path: '/shelves/favorite', type: 'favorite', labelKey: 'nav.shelf_favorite', icon: Bookmark },
]
</script>

<template>
  <aside class="w-64 shrink-0 hidden lg:block p-4">
    <div class="sticky top-20 space-y-6">
      <!-- Основные разделы -->
      <nav class="space-y-1">
        <router-link
          v-for="item in mainNav"
          :key="item.path"
          :to="item.path"
          class="flex items-center gap-3 px-3.5 py-2.5 rounded-xl text-sm font-medium transition-colors"
          :class="route.path === item.path ? 'bg-accent text-white shadow-sm shadow-accent/20 font-semibold' : 'text-fg-secondary hover:text-fg-primary hover:bg-bg-hover'"
        >
          <component :is="item.icon" class="w-4 h-4" />
          <span>{{ t(item.labelKey) }}</span>
        </router-link>
      </nav>

      <!-- Полки пользователя -->
      <div v-if="authStore.isAuthenticated" class="pt-4 border-t border-border">
        <span class="px-3.5 text-xs font-semibold uppercase tracking-wider text-fg-muted block mb-2">
          {{ t('nav.shelves') }}
        </span>
        <nav class="space-y-1">
          <router-link
            v-for="shelf in shelfNav"
            :key="shelf.path"
            :to="shelf.path"
            class="flex items-center gap-3 px-3.5 py-2 rounded-xl text-sm font-medium transition-colors"
            :class="route.path === shelf.path ? 'bg-bg-surface text-accent font-semibold border border-border' : 'text-fg-secondary hover:text-fg-primary hover:bg-bg-hover'"
          >
            <component :is="shelf.icon" class="w-4 h-4" />
            <span>{{ t(shelf.labelKey) }}</span>
          </router-link>
        </nav>
      </div>

      <!-- Административная панель (Карантин) -->
      <div v-if="authStore.isAdmin" class="pt-4 border-t border-border">
        <span class="px-3.5 text-xs font-semibold uppercase tracking-wider text-fg-muted block mb-2">
          Администрирование
        </span>
        <router-link
          to="/quarantine"
          class="flex items-center gap-3 px-3.5 py-2.5 rounded-xl text-sm font-medium transition-colors"
          :class="route.path === '/quarantine' ? 'bg-amber-500/15 text-amber-500 font-semibold border border-amber-500/30' : 'text-fg-secondary hover:text-fg-primary hover:bg-bg-hover'"
        >
          <ShieldAlert class="w-4 h-4 text-amber-500" />
          <span>{{ t('nav.quarantine') }}</span>
        </router-link>
      </div>

      <!-- Подключение внешних читалок и приложений (OPDS) -->
      <div class="pt-4 border-t border-border">
        <button
          @click="emit('openConnect')"
          class="w-full flex items-center justify-between p-3 rounded-xl bg-bg-surface hover:bg-bg-hover border border-border text-left transition-all group shadow-xs hover:border-accent/40"
        >
          <div class="flex items-center gap-2.5 min-w-0">
            <div class="p-1.5 rounded-lg bg-accent/10 text-accent group-hover:bg-accent group-hover:text-white transition-colors shrink-0">
              <Rss class="w-4 h-4" />
            </div>
            <div class="min-w-0">
              <span class="text-xs font-semibold text-fg-primary block leading-tight truncate">
                {{ t('connect.title') }}
              </span>
              <span class="text-[10px] text-fg-muted block truncate mt-0.5">
                {{ t('connect.subtitle_short') }}
              </span>
            </div>
          </div>
          <ChevronRight class="w-3.5 h-3.5 text-fg-muted group-hover:text-fg-primary transition-colors shrink-0" />
        </button>
      </div>
    </div>
  </aside>
</template>
