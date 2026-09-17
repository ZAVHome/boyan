<script setup lang="ts">
import { onMounted, onUnmounted, computed } from 'vue'
import { useAdminStore } from '@/stores/admin'
import { useAuthStore } from '@/stores/auth'
import { useI18n } from 'vue-i18n'
import AdminSidebar from './AdminSidebar.vue'
import ThemeToggle from '@/components/common/ThemeToggle.vue'
import LanguageSelector from '@/components/common/LanguageSelector.vue'
import { ArrowLeft, Loader2, Shield } from 'lucide-vue-next'

const { t } = useI18n()
const adminStore = useAdminStore()
const authStore = useAuthStore()

onMounted(() => {
  adminStore.fetchDashboard()
})

onUnmounted(() => {
  adminStore.stopPolling()
})

const activeTaskCount = computed(() => {
  return adminStore.activeTasks.filter(
    t => t.status === 'running' || t.status === 'pending'
  ).length
})
</script>

<template>
  <div class="min-h-screen flex flex-col bg-bg-primary text-fg-primary">
    <!-- Верхняя шапка панели администратора -->
    <header class="sticky top-0 z-40 bg-bg-surface/90 backdrop-blur-md border-b border-border transition-colors h-16">
      <div class="px-4 sm:px-6 lg:px-8 h-full flex items-center justify-between gap-4">
        <!-- Логотип и значок администратора -->
        <div class="flex items-center gap-3">
          <router-link to="/" class="flex items-center gap-2 group">
            <img
              src="/logo.svg"
              alt="Боян"
              class="w-9 h-9 rounded-full object-cover shadow-sm group-hover:scale-105 transition-transform"
            />
            <div>
              <span class="text-lg font-bold tracking-tight text-fg-primary leading-none block">
                {{ t('app.title') }}
              </span>
              <span class="text-[10px] font-bold text-accent tracking-wider uppercase block mt-0.5">
                {{ t('admin.header_badge') }}
              </span>
            </div>
          </router-link>

          <!-- Индикатор активности фоновых задач -->
          <router-link
            v-if="activeTaskCount > 0"
            to="/admin/storage"
            class="ml-4 hidden sm:flex items-center gap-2 px-3 py-1 rounded-full bg-accent/15 text-accent text-xs font-semibold animate-pulse border border-accent/25 hover:bg-accent/25 transition-colors"
          >
            <Loader2 class="w-3.5 h-3.5 animate-spin" />
            <span>{{ t('admin.tasks_running', { count: activeTaskCount }) }}</span>
          </router-link>
        </div>

        <!-- Правый блок: переключатели и профиль -->
        <div class="flex items-center gap-3">
          <ThemeToggle />
          <LanguageSelector />

          <div class="h-6 w-px bg-border mx-1" />

          <!-- Профиль админа -->
          <div class="flex items-center gap-2">
            <div class="w-8 h-8 rounded-lg bg-accent/10 border border-accent/20 flex items-center justify-center text-accent">
              <Shield class="w-4 h-4" />
            </div>
            <span class="hidden md:inline text-xs font-semibold text-fg-primary">
              {{ authStore.user?.username }}
            </span>
          </div>

          <!-- Кнопка перехода в каталог -->
          <router-link
            to="/"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-bg-surface border border-border hover:bg-bg-hover text-fg-primary text-xs font-medium transition-colors ml-1"
          >
            <ArrowLeft class="w-3.5 h-3.5" />
            <span class="hidden sm:inline">{{ t('admin.nav.back_to_catalog') }}</span>
          </router-link>
        </div>
      </div>
    </header>

    <!-- Основной контейнер: сайдбар слева, контент справа -->
    <div class="flex-1 flex overflow-hidden">
      <AdminSidebar />
      <main class="flex-1 overflow-y-auto p-4 sm:p-6 lg:p-8">
        <router-view />
      </main>
    </div>
  </div>
</template>
