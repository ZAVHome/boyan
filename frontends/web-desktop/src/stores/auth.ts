import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api, tokenStorage } from '@/api/client'
import type { User } from '@/api/types'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const isLoaded = ref(false)

  const isAuthenticated = computed(() => !!user.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  async function checkAuth() {
    const token = tokenStorage.get()
    if (!token) {
      user.value = null
      isLoaded.value = true
      return
    }

    try {
      const res = await api.get<{ user: User }>('/api/v1/auth/me')
      user.value = res.user
    } catch {
      user.value = null
      tokenStorage.clear()
    } finally {
      isLoaded.value = true
    }
  }

  async function login(username: string, password: string):Promise<boolean> {
    try {
      const res = await api.post<{ token: string; user: User }>('/api/v1/auth/login', {
        username,
        password
      })
      tokenStorage.set(res.token)
      user.value = res.user
      return true
    } catch (err) {
      throw err
    }
  }

  async function logout() {
    try {
      await api.post('/api/v1/auth/logout')
    } catch {
      // Игнорируем сетевые сбои при выходе
    } finally {
      tokenStorage.clear()
      user.value = null
    }
  }

  return {
    user,
    isLoaded,
    isAuthenticated,
    isAdmin,
    checkAuth,
    login,
    logout
  }
})
