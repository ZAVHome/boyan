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
  Compass
} from 'lucide-vue-next'

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
    </div>
  </aside>
</template>
