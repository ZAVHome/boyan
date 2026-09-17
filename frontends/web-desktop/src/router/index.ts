import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const routes = [
  {
    path: '/',
    name: 'catalog',
    component: () => import('@/views/CatalogView.vue')
  },
  {
    path: '/authors',
    name: 'authors',
    component: () => import('@/views/AuthorsView.vue')
  },
  {
    path: '/series',
    name: 'series',
    component: () => import('@/views/SeriesView.vue')
  },
  {
    path: '/shelves/:type?',
    name: 'shelves',
    component: () => import('@/views/ShelvesView.vue')
  },
  {
    path: '/read/:id',
    name: 'reader',
    component: () => import('@/views/ReaderView.vue'),
    props: true
  },
  {
    path: '/quarantine',
    name: 'quarantine',
    component: () => import('@/views/QuarantineView.vue'),
    meta: { requiresAdmin: true }
  },
  {
    path: '/admin',
    component: () => import('@/components/admin/AdminLayout.vue'),
    meta: { requiresAdmin: true },
    children: [
      {
        path: '',
        name: 'admin-dashboard',
        component: () => import('@/views/admin/AdminDashboardView.vue')
      },
      {
        path: 'users',
        name: 'admin-users',
        component: () => import('@/views/admin/AdminUsersView.vue')
      },
      {
        path: 'books',
        name: 'admin-books',
        component: () => import('@/views/admin/AdminBooksView.vue')
      },
      {
        path: 'quarantine',
        name: 'admin-quarantine',
        component: () => import('@/views/admin/AdminQuarantineView.vue')
      },
      {
        path: 'storage',
        name: 'admin-storage',
        component: () => import('@/views/admin/AdminStorageView.vue')
      },
      {
        path: 'settings',
        name: 'admin-settings',
        component: () => import('@/views/admin/AdminSettingsView.vue')
      },
      {
        path: 'logs',
        name: 'admin-logs',
        component: () => import('@/views/admin/AdminLogsView.vue')
      }
    ]
  },
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/LoginView.vue')
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/'
  }
]

export const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach(async (to, _from, next) => {
  const authStore = useAuthStore()
  if (!authStore.isLoaded) {
    await authStore.checkAuth()
  }

  if (to.meta.requiresAdmin && !authStore.isAdmin) {
    next({ name: 'login', query: { redirect: to.fullPath } })
    return
  }

  next()
})
