<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useI18n } from 'vue-i18n'
import { KeyRound, User as UserIcon, Loader2, AlertCircle } from 'lucide-vue-next'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()
const authStore = useAuthStore()

const username = ref('admin')
const password = ref('')
const loading = ref(false)
const errorMessage = ref('')

async function handleLogin() {
  if (!username.value || !password.value) return

  loading.value = true
  errorMessage.value = ''

  try {
    await authStore.login(username.value, password.value)
    const redirect = (route.query.redirect as string) || '/'
    router.push(redirect)
  } catch (err: any) {
    errorMessage.value = err.message || t('auth.error')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-[80vh] flex items-center justify-center p-4">
    <div class="w-full max-w-md rounded-3xl bg-bg-surface border border-border shadow-2xl p-8 space-y-6">
      <div class="text-center space-y-2">
        <img
          src="/logo.svg"
          alt="Боян"
          class="w-16 h-16 rounded-full mx-auto mb-3 shadow-lg shadow-amber-900/20 hover:scale-105 transition-transform"
        />
        <h2 class="text-2xl font-bold text-fg-primary tracking-tight">
          {{ t('auth.title') }}
        </h2>
        <p class="text-xs text-fg-muted">
          Введите учетные данные для доступа к библиотеке
        </p>
      </div>

      <div v-if="errorMessage" class="p-3.5 rounded-xl bg-red-500/10 text-red-500 border border-red-500/30 text-xs flex items-center gap-2">
        <AlertCircle class="w-4 h-4 shrink-0" />
        <span>{{ errorMessage }}</span>
      </div>

      <form @submit.prevent="handleLogin" class="space-y-4">
        <div>
          <label class="text-xs font-semibold uppercase tracking-wider text-fg-muted block mb-1.5">
            {{ t('auth.username') }}
          </label>
          <div class="relative flex items-center">
            <UserIcon class="w-4 h-4 text-fg-muted absolute left-3.5 pointer-events-none" />
            <input
              v-model="username"
              type="text"
              required
              class="w-full pl-10 pr-4 py-2.5 rounded-xl bg-bg-primary border border-border focus:border-accent focus:ring-2 focus:ring-accent/20 text-fg-primary text-sm transition-all focus:outline-none"
            />
          </div>
        </div>

        <div>
          <label class="text-xs font-semibold uppercase tracking-wider text-fg-muted block mb-1.5">
            {{ t('auth.password') }}
          </label>
          <div class="relative flex items-center">
            <KeyRound class="w-4 h-4 text-fg-muted absolute left-3.5 pointer-events-none" />
            <input
              v-model="password"
              type="password"
              required
              class="w-full pl-10 pr-4 py-2.5 rounded-xl bg-bg-primary border border-border focus:border-accent focus:ring-2 focus:ring-accent/20 text-fg-primary text-sm transition-all focus:outline-none"
            />
          </div>
        </div>

        <button
          type="submit"
          :disabled="loading"
          class="w-full py-2.5 rounded-xl bg-accent hover:bg-accent-hover text-white font-semibold text-sm shadow-md shadow-accent/25 transition-all flex items-center justify-center gap-2 disabled:opacity-50"
        >
          <Loader2 v-if="loading" class="w-4 h-4 animate-spin" />
          <span>{{ t('auth.submit') }}</span>
        </button>
      </form>
    </div>
  </div>
</template>
