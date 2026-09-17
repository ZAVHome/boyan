<script setup lang="ts">
import { onMounted } from 'vue'
import { useQuarantineStore } from '@/stores/quarantine'
import { useI18n } from 'vue-i18n'
import QuarantineTable from '@/components/quarantine/QuarantineTable.vue'
import { ShieldAlert, RefreshCw } from 'lucide-vue-next'

const { t } = useI18n()
const store = useQuarantineStore()

onMounted(() => {
  store.fetchQuarantine()
})
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between pb-2 border-b border-border">
      <div>
        <h1 class="text-xl font-bold text-fg-primary tracking-tight flex items-center gap-2">
          <ShieldAlert class="w-5 h-5 text-amber-500" />
          <span>{{ t('quarantine.title') }}</span>
        </h1>
        <p class="text-xs text-fg-muted mt-0.5">
          {{ t('quarantine.subtitle') }}
        </p>
      </div>

      <button
        @click="store.fetchQuarantine"
        class="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-bg-surface border border-border hover:bg-bg-hover text-fg-secondary text-xs font-semibold transition-colors"
      >
        <RefreshCw class="w-3.5 h-3.5" :class="{ 'animate-spin': store.loading }" />
        <span>Обновить</span>
      </button>
    </div>

    <QuarantineTable />
  </div>
</template>
