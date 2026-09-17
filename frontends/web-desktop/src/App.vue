<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import Navbar from '@/components/common/Navbar.vue'
import Sidebar from '@/components/common/Sidebar.vue'
import UploadModal from '@/components/catalog/UploadModal.vue'
import { useCatalogStore } from '@/stores/catalog'

const route = useRoute()
const authStore = useAuthStore()
const catalogStore = useCatalogStore()

const isReaderMode = computed(() => route.name === 'reader')
const isAdminMode = computed(() => route.path.startsWith('/admin'))
const showUploadModal = ref(false)

onMounted(() => {
  authStore.checkAuth()
})

function handleUploaded() {
  catalogStore.fetchBooks(true)
}
</script>

<template>
  <!-- Полноэкранный режим чтения или панели администратора -->
  <div v-if="isReaderMode || isAdminMode" class="min-h-screen bg-bg-primary">
    <router-view />
  </div>


  <!-- Стандартный макет приложения с навигацией и сайдбаром -->
  <div v-else class="min-h-screen flex flex-col bg-bg-primary text-fg-primary">
    <Navbar @open-upload="showUploadModal = true" />

    <div class="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-6 flex gap-6">
      <Sidebar />
      <main class="flex-1 min-w-0">
        <router-view />
      </main>
    </div>

    <!-- Модальное окно загрузки книг -->
    <UploadModal
      v-if="showUploadModal"
      @close="showUploadModal = false"
      @uploaded="handleUploaded"
    />
  </div>
</template>
