<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useCatalogStore } from '@/stores/catalog'
import { useI18n } from 'vue-i18n'
import { Users, BookOpen, ArrowRight } from 'lucide-vue-next'

const { t } = useI18n()
const router = useRouter()
const catalogStore = useCatalogStore()

// Алфавитный указатель
const letters = 'АБВГДЕЖЗИЙКЛМНОПРСТУФХЦЧШЩЭЮЯABCDEFGHIJKLMNOPQRSTUVWXYZ'.split('')
const activeLetter = ref('А')

function searchAuthor(name: string) {
  catalogStore.setSearch(name)
  router.push({ name: 'catalog' })
}
</script>

<template>
  <div class="space-y-6">
    <div class="pb-2 border-b border-border">
      <h1 class="text-xl font-bold text-fg-primary tracking-tight">
        {{ t('nav.authors') }}
      </h1>
      <p class="text-xs text-fg-muted mt-0.5">Алфавитный указатель авторов библиотеки</p>
    </div>

    <!-- Алфавит -->
    <div class="flex flex-wrap gap-1 bg-bg-surface p-2 rounded-2xl border border-border">
      <button
        v-for="l in letters"
        :key="l"
        @click="activeLetter = l"
        class="w-8 h-8 rounded-lg text-xs font-bold transition-colors flex items-center justify-center"
        :class="activeLetter === l ? 'bg-accent text-white shadow-sm' : 'text-fg-secondary hover:bg-bg-hover hover:text-fg-primary'"
      >
        {{ l }}
      </button>
    </div>

    <div class="p-8 rounded-2xl bg-bg-surface border border-border text-center space-y-3">
      <Users class="w-12 h-12 text-accent/40 mx-auto" />
      <h3 class="text-base font-bold text-fg-primary">Авторы на букву "{{ activeLetter }}"</h3>
      <p class="text-xs text-fg-muted max-w-md mx-auto">
        Для просмотра произведений конкретного автора воспользуйтесь строкой поиска или выберите книгу на витрине.
      </p>
    </div>
  </div>
</template>
