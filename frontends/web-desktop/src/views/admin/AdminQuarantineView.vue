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
  <div class="space-y-6 max-w-7xl mx-auto">
    <div class="flex items-center justify-between pb-3 border-b border-border">
      <div>
        <h1 class="text-2xl font-bold tracking-tight text-fg-primary flex items-center gap-2">
          <ShieldAlert class="w-6 h-6 text-amber-500" />
          <span>{{ t('admin.quarantine.title') }}</span>
        </h1>
        <p class="text-sm text-fg-secondary mt-1">
          {{ t('admin.quarantine.subtitle') }}
        </p>
      </div>

      <button
        @click="store.fetchQuarantine"
        class="flex items-center gap-1.5 px-3 py-2 rounded-xl bg-bg-surface border border-border hover:bg-bg-hover text-fg-primary text-xs font-medium transition-colors"
      >
        <RefreshCw class="w-3.5 h-3.5" :class="{ 'animate-spin': store.loading }" />
        <span>{{ t('admin.refresh') }}</span>
      </button>
    </div>

    <div class="bg-bg-surface rounded-2xl border border-border p-4">
      <QuarantineTable />
    </div>
  </div>
</template>
