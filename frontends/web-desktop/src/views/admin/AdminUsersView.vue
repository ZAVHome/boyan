<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { adminApi, type AdminUser } from '@/api/admin'
import { useAuthStore } from '@/stores/auth'
import { useI18n } from 'vue-i18n'
import type { Role } from '@/api/types'
import {
  Users,
  UserPlus,
  Search,
  Key,
  Edit2,
  Trash2,
  CheckCircle2,
  XCircle,
  Shield,
  User as UserIcon,
  X,
  Loader2
} from 'lucide-vue-next'

const { t } = useI18n()
const authStore = useAuthStore()

const users = ref<AdminUser[]>([])
const total = ref(0)
const isLoading = ref(false)
const searchQuery = ref('')
const roleFilter = ref<Role | ''>('')

// Модалка создания/редактирования
const showCreateModal = ref(false)
const showEditModal = ref(false)
const showPasswordModal = ref(false)
const selectedUser = ref<AdminUser | null>(null)

// Поля формы создания
const createForm = ref({
  username: '',
  password: '',
  role: 'user' as Role,
  is_active: true
})

// Поля формы редактирования
const editForm = ref({
  role: 'user' as Role,
  is_active: true
})

// Поля формы пароля
const newPassword = ref('')
const formError = ref<string | null>(null)
const isSaving = ref(false)

onMounted(() => {
  fetchUsers()
})

async function fetchUsers() {
  isLoading.value = true
  try {
    const res = await adminApi.listUsers({
      q: searchQuery.value,
      role: roleFilter.value || undefined,
      limit: 50
    })
    users.value = res.users
    total.value = res.total
  } catch (err: any) {
    alert(err.message || 'Failed to load users')
  } finally {
    isLoading.value = false
  }
}

function openCreateModal() {
  createForm.value = {
    username: '',
    password: '',
    role: 'user',
    is_active: true
  }
  formError.value = null
  showCreateModal.value = true
}

async function handleCreateUser() {
  if (!createForm.value.username || !createForm.value.password) {
    formError.value = t('admin.users.fields_required')
    return
  }

  isSaving.value = true
  formError.value = null
  try {
    await adminApi.createUser(createForm.value)
    showCreateModal.value = false
    fetchUsers()
  } catch (err: any) {
    formError.value = err.message || 'Failed to create user'
  } finally {
    isSaving.value = false
  }
}

function openEditModal(user: AdminUser) {
  selectedUser.value = user
  editForm.value = {
    role: user.role,
    is_active: user.is_active
  }
  formError.value = null
  showEditModal.value = true
}

async function handleUpdateUser() {
  if (!selectedUser.value) return

  isSaving.value = true
  formError.value = null
  try {
    await adminApi.updateUser(selectedUser.value.id, editForm.value)
    showEditModal.value = false
    fetchUsers()
  } catch (err: any) {
    formError.value = err.message || 'Failed to update user'
  } finally {
    isSaving.value = false
  }
}

function openPasswordModal(user: AdminUser) {
  selectedUser.value = user
  newPassword.value = ''
  formError.value = null
  showPasswordModal.value = true
}

async function handleUpdatePassword() {
  if (!selectedUser.value || !newPassword.value) {
    formError.value = t('admin.users.password_required')
    return
  }

  isSaving.value = true
  formError.value = null
  try {
    await adminApi.updatePassword(selectedUser.value.id, newPassword.value)
    showPasswordModal.value = false
    alert(t('admin.users.password_changed_success'))
  } catch (err: any) {
    formError.value = err.message || 'Failed to change password'
  } finally {
    isSaving.value = false
  }
}

async function handleDeleteUser(user: AdminUser) {
  if (!confirm(t('admin.users.confirm_delete', { name: user.username }))) return

  try {
    await adminApi.deleteUser(user.id)
    fetchUsers()
  } catch (err: any) {
    alert(err.message || 'Failed to delete user')
  }
}
</script>

<template>
  <div class="space-y-6 max-w-7xl mx-auto">
    <!-- Заголовок и кнопка создания -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold tracking-tight text-fg-primary flex items-center gap-2">
          <Users class="w-6 h-6 text-accent" />
          {{ t('admin.users.title') }}
        </h1>
        <p class="text-sm text-fg-secondary mt-1">
          {{ t('admin.users.subtitle') }}
        </p>
      </div>

      <button
        @click="openCreateModal"
        class="flex items-center gap-2 px-4 py-2.5 rounded-xl bg-accent text-white font-medium text-sm shadow-sm hover:bg-accent-hover transition-colors"
      >
        <UserPlus class="w-4 h-4" />
        <span>{{ t('admin.users.add_user_btn') }}</span>
      </button>
    </div>

    <!-- Фильтры и поиск -->
    <div class="bg-bg-surface rounded-2xl border border-border p-4 flex flex-col sm:flex-row gap-4">
      <div class="flex-1 relative">
        <Search class="w-4 h-4 text-fg-muted absolute left-3.5 top-3 pointer-events-none" />
        <input
          v-model="searchQuery"
          @keyup.enter="fetchUsers"
          type="text"
          :placeholder="t('admin.users.search_placeholder')"
          class="w-full pl-10 pr-4 py-2 rounded-xl bg-bg-primary border border-border text-fg-primary text-sm focus:outline-none focus:border-accent"
        />
      </div>

      <select
        v-model="roleFilter"
        @change="fetchUsers"
        class="px-4 py-2 rounded-xl bg-bg-primary border border-border text-fg-primary text-sm focus:outline-none focus:border-accent"
      >
        <option value="">{{ t('admin.users.all_roles') }}</option>
        <option value="admin">Admin</option>
        <option value="user">User</option>
        <option value="restricted">Restricted</option>
      </select>
    </div>

    <!-- Таблица пользователей -->
    <div class="bg-bg-surface rounded-2xl border border-border overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-left text-sm">
          <thead class="bg-bg-primary text-xs uppercase tracking-wider text-fg-muted border-b border-border">
            <tr>
              <th class="py-3 px-4">{{ t('admin.users.col_username') }}</th>
              <th class="py-3 px-4">{{ t('admin.users.col_role') }}</th>
              <th class="py-3 px-4">{{ t('admin.users.col_status') }}</th>
              <th class="py-3 px-4">{{ t('admin.users.col_created') }}</th>
              <th class="py-3 px-4 text-right">{{ t('admin.actions') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <tr v-if="isLoading">
              <td colspan="5" class="py-8 text-center text-fg-muted">
                <Loader2 class="w-6 h-6 animate-spin mx-auto text-accent" />
              </td>
            </tr>
            <tr v-else-if="users.length === 0">
              <td colspan="5" class="py-8 text-center text-fg-muted">
                {{ t('admin.users.no_users_found') }}
              </td>
            </tr>
            <tr v-for="u in users" :key="u.id" class="hover:bg-bg-hover/50 transition-colors">
              <td class="py-3.5 px-4 font-medium text-fg-primary flex items-center gap-2">
                <div class="w-7 h-7 rounded-full bg-accent/15 text-accent flex items-center justify-center font-bold text-xs">
                  {{ u.username.charAt(0).toUpperCase() }}
                </div>
                <span>{{ u.username }}</span>
                <span v-if="u.id === authStore.user?.id" class="text-[10px] px-1.5 py-0.5 rounded bg-blue-500/15 text-blue-500 font-semibold">
                  {{ t('admin.users.badge_you') }}
                </span>
              </td>
              <td class="py-3.5 px-4">
                <span
                  class="px-2 py-0.5 rounded text-xs font-semibold uppercase"
                  :class="u.role === 'admin' ? 'bg-amber-500/15 text-amber-500' : 'bg-bg-primary text-fg-secondary border border-border'"
                >
                  {{ u.role }}
                </span>
              </td>
              <td class="py-3.5 px-4">
                <span
                  class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-xs font-medium"
                  :class="u.is_active ? 'text-emerald-500 bg-emerald-500/10' : 'text-red-500 bg-red-500/10'"
                >
                  <component :is="u.is_active ? CheckCircle2 : XCircle" class="w-3.5 h-3.5" />
                  {{ u.is_active ? t('admin.users.status_active') : t('admin.users.status_inactive') }}
                </span>
              </td>
              <td class="py-3.5 px-4 text-xs text-fg-muted">
                {{ new Date(u.created_at).toLocaleDateString() }}
              </td>
              <td class="py-3.5 px-4 text-right">
                <div class="inline-flex items-center gap-1">
                  <!-- Смена пароля -->
                  <button
                    @click="openPasswordModal(u)"
                    class="p-1.5 rounded-lg hover:bg-bg-hover text-fg-secondary hover:text-fg-primary"
                    :title="t('admin.users.action_change_password')"
                  >
                    <Key class="w-4 h-4" />
                  </button>
                  <!-- Редактирование -->
                  <button
                    @click="openEditModal(u)"
                    class="p-1.5 rounded-lg hover:bg-bg-hover text-fg-secondary hover:text-fg-primary"
                    :title="t('admin.users.action_edit')"
                  >
                    <Edit2 class="w-4 h-4" />
                  </button>
                  <!-- Удаление (нельзя удалить себя) -->
                  <button
                    v-if="u.id !== authStore.user?.id"
                    @click="handleDeleteUser(u)"
                    class="p-1.5 rounded-lg hover:bg-red-500/15 text-fg-secondary hover:text-red-500"
                    :title="t('admin.users.action_delete')"
                  >
                    <Trash2 class="w-4 h-4" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- МОДАЛКА 1: Создание пользователя -->
    <div v-if="showCreateModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fade-in">
      <div class="bg-bg-surface border border-border rounded-2xl max-w-md w-full p-6 space-y-4 shadow-xl">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-bold text-fg-primary">{{ t('admin.users.modal_create_title') }}</h2>
          <button @click="showCreateModal = false" class="text-fg-muted hover:text-fg-primary"><X class="w-5 h-5" /></button>
        </div>

        <div v-if="formError" class="p-3 rounded-xl bg-red-500/10 border border-red-500/20 text-red-500 text-xs font-medium">
          {{ formError }}
        </div>

        <div class="space-y-3">
          <div>
            <label class="text-xs font-semibold text-fg-muted block mb-1">{{ t('admin.users.field_username') }}</label>
            <input v-model="createForm.username" type="text" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm focus:outline-none focus:border-accent" />
          </div>
          <div>
            <label class="text-xs font-semibold text-fg-muted block mb-1">{{ t('admin.users.field_password') }}</label>
            <input v-model="createForm.password" type="password" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm focus:outline-none focus:border-accent" />
          </div>
          <div>
            <label class="text-xs font-semibold text-fg-muted block mb-1">{{ t('admin.users.field_role') }}</label>
            <select v-model="createForm.role" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm focus:outline-none focus:border-accent">
              <option value="user">User</option>
              <option value="admin">Admin</option>
              <option value="restricted">Restricted</option>
            </select>
          </div>
          <div class="flex items-center gap-2 pt-2">
            <input v-model="createForm.is_active" type="checkbox" id="active_check" class="rounded border-border text-accent focus:ring-accent" />
            <label for="active_check" class="text-sm text-fg-primary">{{ t('admin.users.field_active') }}</label>
          </div>
        </div>

        <div class="flex justify-end gap-3 pt-4 border-t border-border">
          <button @click="showCreateModal = false" class="px-4 py-2 rounded-xl border border-border hover:bg-bg-hover text-sm font-medium">{{ t('admin.cancel') }}</button>
          <button @click="handleCreateUser" :disabled="isSaving" class="px-4 py-2 rounded-xl bg-accent text-white hover:bg-accent-hover text-sm font-medium disabled:opacity-50 flex items-center gap-1.5">
            <Loader2 v-if="isSaving" class="w-4 h-4 animate-spin" />
            <span>{{ t('admin.save') }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- МОДАЛКА 2: Редактирование пользователя -->
    <div v-if="showEditModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fade-in">
      <div class="bg-bg-surface border border-border rounded-2xl max-w-md w-full p-6 space-y-4 shadow-xl">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-bold text-fg-primary">{{ t('admin.users.modal_edit_title', { name: selectedUser?.username }) }}</h2>
          <button @click="showEditModal = false" class="text-fg-muted hover:text-fg-primary"><X class="w-5 h-5" /></button>
        </div>

        <div v-if="formError" class="p-3 rounded-xl bg-red-500/10 border border-red-500/20 text-red-500 text-xs font-medium">
          {{ formError }}
        </div>

        <div class="space-y-3">
          <div>
            <label class="text-xs font-semibold text-fg-muted block mb-1">{{ t('admin.users.field_role') }}</label>
            <select v-model="editForm.role" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm focus:outline-none focus:border-accent">
              <option value="user">User</option>
              <option value="admin">Admin</option>
              <option value="restricted">Restricted</option>
            </select>
          </div>
          <div class="flex items-center gap-2 pt-2">
            <input v-model="editForm.is_active" type="checkbox" id="edit_active_check" class="rounded border-border text-accent focus:ring-accent" />
            <label for="edit_active_check" class="text-sm text-fg-primary">{{ t('admin.users.field_active') }}</label>
          </div>
        </div>

        <div class="flex justify-end gap-3 pt-4 border-t border-border">
          <button @click="showEditModal = false" class="px-4 py-2 rounded-xl border border-border hover:bg-bg-hover text-sm font-medium">{{ t('admin.cancel') }}</button>
          <button @click="handleUpdateUser" :disabled="isSaving" class="px-4 py-2 rounded-xl bg-accent text-white hover:bg-accent-hover text-sm font-medium disabled:opacity-50 flex items-center gap-1.5">
            <Loader2 v-if="isSaving" class="w-4 h-4 animate-spin" />
            <span>{{ t('admin.save') }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- МОДАЛКА 3: Смена пароля -->
    <div v-if="showPasswordModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fade-in">
      <div class="bg-bg-surface border border-border rounded-2xl max-w-md w-full p-6 space-y-4 shadow-xl">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-bold text-fg-primary">{{ t('admin.users.modal_password_title', { name: selectedUser?.username }) }}</h2>
          <button @click="showPasswordModal = false" class="text-fg-muted hover:text-fg-primary"><X class="w-5 h-5" /></button>
        </div>

        <div v-if="formError" class="p-3 rounded-xl bg-red-500/10 border border-red-500/20 text-red-500 text-xs font-medium">
          {{ formError }}
        </div>

        <div>
          <label class="text-xs font-semibold text-fg-muted block mb-1">{{ t('admin.users.field_new_password') }}</label>
          <input v-model="newPassword" type="password" class="w-full px-3 py-2 rounded-xl bg-bg-primary border border-border text-sm focus:outline-none focus:border-accent" />
        </div>

        <div class="flex justify-end gap-3 pt-4 border-t border-border">
          <button @click="showPasswordModal = false" class="px-4 py-2 rounded-xl border border-border hover:bg-bg-hover text-sm font-medium">{{ t('admin.cancel') }}</button>
          <button @click="handleUpdatePassword" :disabled="isSaving" class="px-4 py-2 rounded-xl bg-accent text-white hover:bg-accent-hover text-sm font-medium disabled:opacity-50 flex items-center gap-1.5">
            <Loader2 v-if="isSaving" class="w-4 h-4 animate-spin" />
            <span>{{ t('admin.users.btn_change_password') }}</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
