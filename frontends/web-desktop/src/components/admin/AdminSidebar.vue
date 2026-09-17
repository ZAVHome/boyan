<script setup lang="ts">
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  LayoutDashboard,
  Users,
  BookOpen,
  ShieldAlert,
  HardDrive,
  Settings,
  Terminal,
  ArrowLeft,
  ShieldCheck
} from 'lucide-vue-next'

const { t } = useI18n()
const route = useRoute()

const navItems = [
  { path: '/admin', exact: true, name: 'dashboard', labelKey: 'admin.nav.dashboard', icon: LayoutDashboard },
  { path: '/admin/users', name: 'users', labelKey: 'admin.nav.users', icon: Users },
  { path: '/admin/books', name: 'books', labelKey: 'admin.nav.books', icon: BookOpen },
  { path: '/admin/quarantine', name: 'quarantine', labelKey: 'admin.nav.quarantine', icon: ShieldAlert },
  { path: '/admin/storage', name: 'storage', labelKey: 'admin.nav.storage', icon: HardDrive },
  { path: '/admin/settings', name: 'settings', labelKey: 'admin.nav.settings', icon: Settings },
  { path: '/admin/logs', name: 'logs', labelKey: 'admin.nav.logs', icon: Terminal }
]

function isActive(item: typeof navItems[0]) {
  if (item.exact) {
    return route.path === item.path
  }
  return route.path.startsWith(item.path)
}
</script>

<template>
  <aside class="w-64 bg-bg-surface border-r border-border flex flex-col shrink-0 min-h-[calc(100vh-4rem)]">
    <!-- Шапка сайдбара -->
    <div class="p-4 border-b border-border flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-fg-muted">
      <ShieldCheck class="w-4 h-4 text-accent" />
      <span>{{ t('admin.sidebar_title') }}</span>
    </div>

    <!-- Список пунктов навигации -->
    <nav class="flex-1 p-3 space-y-1.5 overflow-y-auto">
      <router-link
        v-for="item in navItems"
        :key="item.path"
        :to="item.path"
        class="flex items-center gap-3 px-3.5 py-2.5 rounded-xl text-sm font-medium transition-all"
        :class="isActive(item)
          ? 'bg-accent text-white font-semibold shadow-sm shadow-accent/20'
          : 'text-fg-secondary hover:text-fg-primary hover:bg-bg-hover'"
      >
        <component :is="item.icon" class="w-4 h-4 shrink-0" />
        <span>{{ t(item.labelKey) }}</span>
      </router-link>
    </nav>

    <!-- Нижняя панель с кнопкой возврата -->
    <div class="p-4 border-t border-border mt-auto">
      <router-link
        to="/"
        class="flex items-center gap-2 px-3.5 py-2 rounded-xl text-sm font-medium text-fg-secondary hover:text-fg-primary hover:bg-bg-hover transition-colors"
      >
        <ArrowLeft class="w-4 h-4" />
        <span>{{ t('admin.nav.back_to_catalog') }}</span>
      </router-link>
    </div>
  </aside>
</template>
