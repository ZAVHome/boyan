<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { adminApi } from '@/api/admin'
import { useI18n } from 'vue-i18n'
import {
  Settings,
  Save,
  RefreshCw,
  Server,
  Lock,
  HardDrive,
  BookOpen,
  Send,
  Loader2,
  Check,
  AlertTriangle
} from 'lucide-vue-next'

const { t } = useI18n()

const configData = ref<any>(null)
const isLoading = ref(false)
const loadError = ref<string | null>(null)
const isSaving = ref(false)
const isReloading = ref(false)
const activeTab = ref<'server' | 'security' | 'storage' | 'opds' | 'telegram'>('server')
const successMessage = ref<string | null>(null)

// Вспомогательное строковое поле для CORS
const corsString = ref('')
const tgUsersString = ref('')

function normalizeConfig(res: any) {
  if (!res) return null
  const server = res.server || res.Server || {}
  const admin = res.admin || res.Admin || {}
  const storage = res.storage || res.Storage || {}
  const watcher = storage.watcher || storage.Watcher || {}
  const opds = res.opds || res.OPDS || {}
  const telegram = res.telegram || res.Telegram || {}
  const metadata = res.metadata || res.Metadata || {}
  const logging = res.logging || res.Logging || {}
  const database = res.database || res.Database || {}

  return {
    server: {
      host: server.host ?? server.Host ?? '0.0.0.0',
      port: server.port ?? server.Port ?? 8080,
      base_url: server.base_url ?? server.BaseURL ?? '',
      default_language: server.default_language ?? server.DefaultLanguage ?? 'ru',
      jwt_secret: server.jwt_secret ?? server.JWTSecret ?? '',
      jwt_expiration_hours: server.jwt_expiration_hours ?? server.JWTExpirationHours ?? 24,
      cors_allowed_origins: Array.isArray(server.cors_allowed_origins ?? server.CORSAllowedOrigins)
        ? (server.cors_allowed_origins ?? server.CORSAllowedOrigins)
        : []
    },
    admin: {
      default_username: admin.default_username ?? admin.DefaultUsername ?? 'admin',
      default_password: admin.default_password ?? admin.DefaultPassword ?? '',
      allow_public_registration: admin.allow_public_registration ?? admin.AllowPublicRegistration ?? false
    },
    storage: {
      library_dir: storage.library_dir ?? storage.LibraryDir ?? '',
      watch_dir: storage.watch_dir ?? storage.WatchDir ?? '',
      path_template: storage.path_template ?? storage.PathTemplate ?? '',
      watcher: {
        enabled: watcher.enabled ?? watcher.Enabled ?? true,
        settle_delay_seconds: watcher.settle_delay_seconds ?? watcher.SettleDelaySeconds ?? 5,
        delete_source_after_import: watcher.delete_source_after_import ?? watcher.DeleteSourceAfterImport ?? false,
        quarantine_duplicates: watcher.quarantine_duplicates ?? watcher.QuarantineDuplicates ?? true
      }
    },
    opds: {
      title: opds.title ?? opds.Title ?? 'Боян OPDS',
      subtitle: opds.subtitle ?? opds.Subtitle ?? '',
      page_size: opds.page_size ?? opds.PageSize ?? 20,
      enable_opds_v1: opds.enable_opds_v1 ?? opds.EnableOPDSv1 ?? true,
      enable_opds_v2: opds.enable_opds_v2 ?? opds.EnableOPDSv2 ?? true,
      allow_anonymous_reading: opds.allow_anonymous_reading ?? opds.AllowAnonymousReading ?? true,
      stream_from_zip: opds.stream_from_zip ?? opds.StreamFromZIP ?? true
    },
    telegram: {
      enabled: telegram.enabled ?? telegram.Enabled ?? false,
      bot_token: telegram.bot_token ?? telegram.BotToken ?? '',
      allowed_user_ids: Array.isArray(telegram.allowed_user_ids ?? telegram.AllowedUserIDs)
        ? (telegram.allowed_user_ids ?? telegram.AllowedUserIDs)
        : []
    },
    metadata: {
      auto_enrich_on_import: metadata.auto_enrich_on_import ?? metadata.AutoEnrichOnImport ?? true,
      providers_priority: metadata.providers_priority ?? metadata.ProvidersPriority ?? [],
      cache_covers_locally: metadata.cache_covers_locally ?? metadata.CacheCoversLocally ?? true,
      cover_thumbnail_size: metadata.cover_thumbnail_size ?? metadata.CoverThumbnailSize ?? 300,
      cover_cache_max_mb: metadata.cover_cache_max_mb ?? metadata.CoverCacheMaxMB ?? 512,
      cover_cache_dir: metadata.cover_cache_dir ?? metadata.CoverCacheDir ?? ''
    },
    logging: {
      level: logging.level ?? logging.Level ?? 'info',
      format: logging.format ?? logging.Format ?? 'text'
    },
    database: {
      driver: database.driver ?? database.Driver ?? 'sqlite',
      sqlite: {
        path: database.sqlite?.path ?? database.SQLite?.Path ?? '',
        enable_wal: database.sqlite?.enable_wal ?? database.SQLite?.EnableWAL ?? true,
        busy_timeout_ms: database.sqlite?.busy_timeout_ms ?? database.SQLite?.BusyTimeoutMS ?? 5000,
        cache_size_kb: database.sqlite?.cache_size_kb ?? database.SQLite?.CacheSizeKB ?? 16000
      }
    }
  }
}

onMounted(() => {
  loadSettings()
})

async function loadSettings() {
  isLoading.value = true
  loadError.value = null
  try {
    const res = await adminApi.getSettings()
    configData.value = normalizeConfig(res)
    corsString.value = (configData.value.server.cors_allowed_origins || []).join(', ')
    tgUsersString.value = (configData.value.telegram.allowed_user_ids || []).join(', ')
  } catch (err: any) {
    loadError.value = err.message || 'Failed to load settings'
    console.error('Failed to load settings', err)
  } finally {
    isLoading.value = false
  }
}

async function handleSaveSettings() {
  if (!configData.value) return
  isSaving.value = true
  successMessage.value = null

  try {
    // Преобразуем строковые поля обратно
    configData.value.server.cors_allowed_origins = corsString.value
      .split(',')
      .map(s => s.trim())
      .filter(Boolean)

    configData.value.telegram.allowed_user_ids = tgUsersString.value
      .split(',')
      .map(s => Number(s.trim()))
      .filter(n => !isNaN(n) && n > 0)

    await adminApi.updateSettings(configData.value)
    successMessage.value = t('admin.settings.saved_success')
    setTimeout(() => (successMessage.value = null), 4000)
  } catch (err: any) {
    alert(err.message || 'Failed to save settings')
  } finally {
    isSaving.value = false
  }
}

async function handleReloadServices() {
  isReloading.value = true
  successMessage.value = null
  try {
    await adminApi.reloadServices()
    successMessage.value = t('admin.settings.reload_success')
    setTimeout(() => (successMessage.value = null), 4000)
  } catch (err: any) {
    alert(err.message || 'Failed to reload services')
  } finally {
    isReloading.value = false
  }
}
</script>

<template>
  <div class="space-y-6 max-w-5xl mx-auto">
    <!-- Заголовок -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-3 border-b border-border">
      <div>
        <h1 class="text-2xl font-bold tracking-tight text-fg-primary flex items-center gap-2">
          <Settings class="w-6 h-6 text-accent" />
          {{ t('admin.settings.title') }}
        </h1>
        <p class="text-sm text-fg-secondary mt-1">
          {{ t('admin.settings.subtitle') }}
        </p>
      </div>

      <div class="flex items-center gap-3">
        <span v-if="successMessage" class="text-xs text-emerald-500 font-medium flex items-center gap-1 bg-emerald-500/10 px-3 py-1.5 rounded-lg border border-emerald-500/20">
          <Check class="w-3.5 h-3.5" />
          {{ successMessage }}
        </span>

        <button
          @click="handleReloadServices"
          :disabled="isReloading"
          class="flex items-center gap-1.5 px-3.5 py-2 rounded-xl bg-bg-surface border border-border hover:bg-bg-hover text-fg-primary text-xs font-medium transition-colors disabled:opacity-50"
        >
          <RefreshCw class="w-3.5 h-3.5" :class="{ 'animate-spin': isReloading }" />
          <span>{{ t('admin.settings.reload_services_btn') }}</span>
        </button>

        <button
          @click="handleSaveSettings"
          :disabled="isSaving || !configData"
          class="flex items-center gap-2 px-4 py-2 rounded-xl bg-accent text-white hover:bg-accent-hover text-xs font-semibold shadow-sm transition-colors disabled:opacity-50"
        >
          <Loader2 v-if="isSaving" class="w-4 h-4 animate-spin" />
          <Save v-else class="w-4 h-4" />
          <span>{{ t('admin.settings.save_btn') }}</span>
        </button>
      </div>
    </div>

    <!-- Вкладки настроек -->
    <div class="flex border-b border-border overflow-x-auto gap-2">
      <button
        @click="activeTab = 'server'"
        class="flex items-center gap-2 px-4 py-2.5 text-xs font-semibold uppercase tracking-wider border-b-2 transition-colors whitespace-nowrap"
        :class="activeTab === 'server' ? 'border-accent text-accent' : 'border-transparent text-fg-muted hover:text-fg-primary'"
      >
        <Server class="w-4 h-4" />
        <span>{{ t('admin.settings.tab_server') }}</span>
      </button>

      <button
        @click="activeTab = 'security'"
        class="flex items-center gap-2 px-4 py-2.5 text-xs font-semibold uppercase tracking-wider border-b-2 transition-colors whitespace-nowrap"
        :class="activeTab === 'security' ? 'border-accent text-accent' : 'border-transparent text-fg-muted hover:text-fg-primary'"
      >
        <Lock class="w-4 h-4" />
        <span>{{ t('admin.settings.tab_security') }}</span>
      </button>

      <button
        @click="activeTab = 'storage'"
        class="flex items-center gap-2 px-4 py-2.5 text-xs font-semibold uppercase tracking-wider border-b-2 transition-colors whitespace-nowrap"
        :class="activeTab === 'storage' ? 'border-accent text-accent' : 'border-transparent text-fg-muted hover:text-fg-primary'"
      >
        <HardDrive class="w-4 h-4" />
        <span>{{ t('admin.settings.tab_storage') }}</span>
      </button>

      <button
        @click="activeTab = 'opds'"
        class="flex items-center gap-2 px-4 py-2.5 text-xs font-semibold uppercase tracking-wider border-b-2 transition-colors whitespace-nowrap"
        :class="activeTab === 'opds' ? 'border-accent text-accent' : 'border-transparent text-fg-muted hover:text-fg-primary'"
      >
        <BookOpen class="w-4 h-4" />
        <span>{{ t('admin.settings.tab_opds') }}</span>
      </button>

      <button
        @click="activeTab = 'telegram'"
        class="flex items-center gap-2 px-4 py-2.5 text-xs font-semibold uppercase tracking-wider border-b-2 transition-colors whitespace-nowrap"
        :class="activeTab === 'telegram' ? 'border-accent text-accent' : 'border-transparent text-fg-muted hover:text-fg-primary'"
      >
        <Send class="w-4 h-4" />
        <span>{{ t('admin.settings.tab_telegram') }}</span>
      </button>
    </div>

    <!-- Индикатор загрузки -->
    <div v-if="isLoading" class="py-16 text-center text-fg-muted">
      <Loader2 class="w-8 h-8 animate-spin mx-auto text-accent mb-2" />
      <span class="text-xs">{{ t('admin.loading') }}</span>
    </div>

    <!-- Ошибка загрузки -->
    <div v-else-if="loadError" class="py-12 px-6 rounded-2xl bg-red-500/10 border border-red-500/20 text-center space-y-3">
      <AlertTriangle class="w-8 h-8 text-red-500 mx-auto" />
      <p class="text-sm font-semibold text-red-400">{{ loadError }}</p>
      <button @click="loadSettings" class="px-4 py-2 rounded-xl bg-bg-surface border border-border hover:bg-bg-hover text-xs font-semibold text-fg-primary">
        {{ t('admin.refresh') }}
      </button>
    </div>

    <!-- Содержимое вкладок -->
    <div v-else-if="configData" class="bg-bg-surface rounded-2xl border border-border p-6 space-y-6">
      <!-- ВКЛАДКА 1: Сервер -->
      <div v-if="activeTab === 'server'" class="space-y-4">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="text-xs font-semibold text-fg-muted block mb-1">Host</label>
            <input v-model="configData.server.host" type="text" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm font-mono focus:outline-none focus:border-accent" />
          </div>
          <div>
            <label class="text-xs font-semibold text-fg-muted block mb-1">Port</label>
            <input v-model.number="configData.server.port" type="number" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm font-mono focus:outline-none focus:border-accent" />
          </div>
          <div>
            <label class="text-xs font-semibold text-fg-muted block mb-1">Base URL</label>
            <input v-model="configData.server.base_url" type="text" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm font-mono focus:outline-none focus:border-accent" />
          </div>
          <div>
            <label class="text-xs font-semibold text-fg-muted block mb-1">Default Language</label>
            <select v-model="configData.server.default_language" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm focus:outline-none focus:border-accent">
              <option value="ru">Русский (ru)</option>
              <option value="en">English (en)</option>
            </select>
          </div>
        </div>
      </div>

      <!-- ВКЛАДКА 2: Регистрация и Безопасность -->
      <div v-if="activeTab === 'security'" class="space-y-4">
        <!-- Флаг публичной регистрации -->
        <div class="p-4 rounded-xl bg-bg-primary border border-border flex items-center justify-between">
          <div>
            <span class="text-sm font-bold text-fg-primary block">{{ t('admin.settings.allow_registration_title') }}</span>
            <span class="text-xs text-fg-muted block mt-0.5">{{ t('admin.settings.allow_registration_hint') }}</span>
          </div>
          <input
            type="checkbox"
            v-model="configData.admin.allow_public_registration"
            class="w-5 h-5 rounded border-border text-accent focus:ring-accent cursor-pointer"
          />
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="text-xs font-semibold text-fg-muted block mb-1">JWT Expiration (Hours)</label>
            <input v-model.number="configData.server.jwt_expiration_hours" type="number" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm font-mono focus:outline-none focus:border-accent" />
          </div>
          <div>
            <label class="text-xs font-semibold text-fg-muted block mb-1">CORS Allowed Origins (через запятую)</label>
            <input v-model="corsString" type="text" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm font-mono focus:outline-none focus:border-accent" />
          </div>
        </div>
      </div>

      <!-- ВКЛАДКА 3: Хранилище и инжест -->
      <div v-if="activeTab === 'storage'" class="space-y-4">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="text-xs font-semibold text-fg-muted block mb-1">Library Directory</label>
            <input v-model="configData.storage.library_dir" type="text" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm font-mono focus:outline-none focus:border-accent" />
          </div>
          <div>
            <label class="text-xs font-semibold text-fg-muted block mb-1">Watch Directory</label>
            <input v-model="configData.storage.watch_dir" type="text" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm font-mono focus:outline-none focus:border-accent" />
          </div>
          <div class="md:col-span-2">
            <label class="text-xs font-semibold text-fg-muted block mb-1">Path Template</label>
            <input v-model="configData.storage.path_template" type="text" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm font-mono focus:outline-none focus:border-accent" />
          </div>
          <div>
            <label class="text-xs font-semibold text-fg-muted block mb-1">Settle Delay (Seconds)</label>
            <input v-model.number="configData.storage.watcher.settle_delay_seconds" type="number" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm font-mono focus:outline-none focus:border-accent" />
          </div>
        </div>

        <div class="space-y-2 pt-2">
          <label class="flex items-center gap-2 text-xs text-fg-primary cursor-pointer">
            <input v-model="configData.storage.watcher.quarantine_duplicates" type="checkbox" class="rounded border-border text-accent focus:ring-accent" />
            <span>{{ t('admin.settings.quarantine_duplicates') }}</span>
          </label>
          <label class="flex items-center gap-2 text-xs text-fg-primary cursor-pointer">
            <input v-model="configData.storage.watcher.delete_source_after_import" type="checkbox" class="rounded border-border text-accent focus:ring-accent" />
            <span>{{ t('admin.settings.delete_source') }}</span>
          </label>
        </div>
      </div>

      <!-- ВКЛАДКА 4: OPDS протоколы -->
      <div v-if="activeTab === 'opds'" class="space-y-4">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="text-xs font-semibold text-fg-muted block mb-1">Catalog Title</label>
            <input v-model="configData.opds.title" type="text" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm focus:outline-none focus:border-accent" />
          </div>
          <div>
            <label class="text-xs font-semibold text-fg-muted block mb-1">Catalog Subtitle</label>
            <input v-model="configData.opds.subtitle" type="text" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm focus:outline-none focus:border-accent" />
          </div>
          <div>
            <label class="text-xs font-semibold text-fg-muted block mb-1">Page Size</label>
            <input v-model.number="configData.opds.page_size" type="number" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm font-mono focus:outline-none focus:border-accent" />
          </div>
        </div>

        <div class="space-y-2 pt-2">
          <label class="flex items-center gap-2 text-xs text-fg-primary cursor-pointer">
            <input v-model="configData.opds.enable_opds_v1" type="checkbox" class="rounded border-border text-accent focus:ring-accent" />
            <span>Enable OPDS v1.2 (Atom/XML)</span>
          </label>
          <label class="flex items-center gap-2 text-xs text-fg-primary cursor-pointer">
            <input v-model="configData.opds.enable_opds_v2" type="checkbox" class="rounded border-border text-accent focus:ring-accent" />
            <span>Enable OPDS v2.0 (JSON-LD)</span>
          </label>
          <label class="flex items-center gap-2 text-xs text-fg-primary cursor-pointer">
            <input v-model="configData.opds.allow_anonymous_reading" type="checkbox" class="rounded border-border text-accent focus:ring-accent" />
            <span>Allow Anonymous Reading (No Basic Auth required)</span>
          </label>
          <label class="flex items-center gap-2 text-xs text-fg-primary cursor-pointer">
            <input v-model="configData.opds.stream_from_zip" type="checkbox" class="rounded border-border text-accent focus:ring-accent" />
            <span>Stream FB2 on-the-fly from FB2.ZIP without extraction</span>
          </label>
        </div>
      </div>

      <!-- ВКЛАДКА 5: Telegram-бот -->
      <div v-if="activeTab === 'telegram'" class="space-y-4">
        <label class="flex items-center gap-2 text-xs text-fg-primary cursor-pointer mb-3">
          <input v-model="configData.telegram.enabled" type="checkbox" class="rounded border-border text-accent focus:ring-accent" />
          <span class="font-bold">Enable Telegram Bot</span>
        </label>

        <div>
          <label class="text-xs font-semibold text-fg-muted block mb-1">Bot Token</label>
          <input v-model="configData.telegram.bot_token" type="password" placeholder="123456:ABC-DEF..." class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm font-mono focus:outline-none focus:border-accent" />
        </div>

        <div>
          <label class="text-xs font-semibold text-fg-muted block mb-1">Allowed Telegram User IDs (через запятую)</label>
          <input v-model="tgUsersString" type="text" placeholder="12345678, 98765432" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm font-mono focus:outline-none focus:border-accent" />
        </div>
      </div>
    </div>
  </div>
</template>
