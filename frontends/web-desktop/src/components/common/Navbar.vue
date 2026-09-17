<script setup lang="ts">
import { ref, watch } from 'vue'
import { useCatalogStore } from '@/stores/catalog'
import { useAuthStore } from '@/stores/auth'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { Search, Plus, BookOpen, LogOut, User as UserIcon, X } from 'lucide-vue-next'
import ThemeToggle from './ThemeToggle.vue'
import LanguageSelector from './LanguageSelector.vue'

const emit = defineEmits<{
  (e: 'openUpload'): void
}>()

const { t } = useI18n()
const router = useRouter()
const catalogStore = useCatalogStore()
const authStore = useAuthStore()

const searchInput = ref(catalogStore.searchQuery)
let debounceTimeout: ReturnType<typeof setTimeout> | null = null

watch(searchInput, (val) => {
  if (debounceTimeout) clearTimeout(debounceTimeout)
  debounceTimeout = setTimeout(() => {
    catalogStore.setSearch(val)
    if (router.currentRoute.value.name !== 'catalog') {
      router.push({ name: 'catalog' })
    }
  }, 300)
})

function clearSearch() {
  searchInput.value = ''
  catalogStore.setSearch('')
}

function handleAuthAction() {
  if (authStore.isAuthenticated) {
    authStore.logout()
  } else {
    router.push({ name: 'login' })
  }
}
</script>

<template>
  <header class="sticky top-0 z-40 bg-bg-surface/80 backdrop-blur-md border-b border-border transition-colors">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between gap-4">
      <!-- Логотип и название -->
      <router-link to="/" class="flex items-center gap-3 shrink-0 group focus:outline-none">
        <div class="w-10 h-10 rounded-xl bg-gradient-to-tr from-accent to-indigo-500 flex items-center justify-center text-white shadow-md shadow-accent/20 group-hover:scale-105 transition-transform">
          <BookOpen class="w-5 h-5" />
        </div>
        <div>
          <span class="text-xl font-bold tracking-tight text-fg-primary block leading-none group-hover:text-accent transition-colors">
            {{ t('app.title') }}
          </span>
          <span class="text-[11px] font-medium text-fg-muted tracking-wider uppercase block mt-0.5">
            {{ t('app.subtitle') }}
          </span>
        </div>
      </router-link>

      <!-- Поисковая строка -->
      <div class="flex-1 max-w-xl relative">
        <div class="relative flex items-center">
          <Search class="w-4 h-4 text-fg-muted absolute left-3.5 pointer-events-none" />
          <input
            v-model="searchInput"
            type="text"
            :placeholder="t('app.search_placeholder')"
            class="w-full pl-10 pr-9 py-2 rounded-xl bg-bg-primary border border-border focus:border-accent focus:ring-2 focus:ring-accent/20 text-fg-primary placeholder:text-fg-muted text-sm transition-all focus:outline-none"
          />
          <button
            v-if="searchInput"
            @click="clearSearch"
            class="absolute right-3 p-0.5 text-fg-muted hover:text-fg-primary rounded-md focus:outline-none"
          >
            <X class="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      <!-- Действия и переключатели -->
      <div class="flex items-center gap-2 sm:gap-3 shrink-0">
        <!-- Кнопка загрузки книги -->
        <button
          v-if="authStore.isAuthenticated"
          @click="emit('openUpload')"
          class="hidden sm:flex items-center gap-1.5 px-3.5 py-2 rounded-xl bg-accent text-white hover:bg-accent-hover text-sm font-medium shadow-sm shadow-accent/25 transition-all focus:outline-none focus:ring-2 focus:ring-accent/50"
        >
          <Plus class="w-4 h-4" />
          <span>{{ t('app.upload_btn') }}</span>
        </button>

        <!-- Переключатели темы и языка -->
        <ThemeToggle />
        <LanguageSelector />

        <!-- Профиль пользователя / Выход -->
        <div v-if="authStore.isAuthenticated" class="flex items-center gap-2 pl-2 border-l border-border">
          <div class="hidden md:flex flex-col items-end leading-tight">
            <span class="text-xs font-semibold text-fg-primary flex items-center gap-1">
              {{ authStore.user?.username }}
              <span v-if="authStore.isAdmin" class="px-1.5 py-0.2 text-[10px] font-bold uppercase rounded bg-accent/15 text-accent">
                {{ t('app.admin_badge') }}
              </span>
            </span>
          </div>
          <button
            @click="handleAuthAction"
            class="p-2 rounded-lg bg-bg-surface border border-border hover:bg-red-500/10 hover:text-red-500 text-fg-secondary transition-colors"
            :title="t('app.logout_btn')"
          >
            <LogOut class="w-4 h-4" />
          </button>
        </div>

        <button
          v-else
          @click="handleAuthAction"
          class="flex items-center gap-1.5 px-3.5 py-2 rounded-xl bg-bg-surface border border-border hover:bg-bg-hover text-fg-primary text-sm font-medium transition-colors"
        >
          <UserIcon class="w-4 h-4" />
          <span class="hidden sm:inline">{{ t('app.login_btn') }}</span>
        </button>
      </div>
    </div>
  </header>
</template>
