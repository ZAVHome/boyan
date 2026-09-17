<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Layers } from 'lucide-vue-next'

const { t } = useI18n()
</script>

<template>
  <div class="space-y-6">
    <div class="pb-2 border-b border-border">
      <h1 class="text-xl font-bold text-fg-primary tracking-tight">
        {{ t('nav.series') }}
      </h1>
      <p class="text-xs text-fg-muted mt-0.5">Литературные серии и циклы произведений</p>
    </div>

    <div class="p-12 text-center rounded-2xl bg-bg-surface border border-border space-y-3">
      <Layers class="w-12 h-12 text-accent/40 mx-auto" />
      <h3 class="text-base font-bold text-fg-primary">Каталог книжных серий</h3>
      <p class="text-xs text-fg-muted max-w-md mx-auto">
        Книжные циклы автоматически группируются и сортируются по порядку томов при просмотре книг.
      </p>
    </div>
  </div>
</template>
