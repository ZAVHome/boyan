import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import HomeView from '@/views/HomeView.vue'
import SearchView from '@/views/SearchView.vue'
import ShelvesView from '@/views/ShelvesView.vue'
import OfflineView from '@/views/OfflineView.vue'
import SettingsView from '@/views/SettingsView.vue'
import ReaderView from '@/views/ReaderView.vue'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'home',
    component: HomeView
  },
  {
    path: '/search',
    name: 'search',
    component: SearchView
  },
  {
    path: '/shelves',
    name: 'shelves',
    component: ShelvesView
  },
  {
    path: '/offline',
    name: 'offline',
    component: OfflineView
  },
  {
    path: '/settings',
    name: 'settings',
    component: SettingsView
  },
  {
    path: '/reader/:id',
    name: 'reader',
    component: ReaderView
  }
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior(_to, _from, savedPosition) {
    if (savedPosition) {
      return savedPosition
    } else {
      return { top: 0 }
    }
  }
})
