<template>
  <MobileReader :book-id="bookId" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import MobileReader from '@/components/reader/MobileReader.vue'

const route = useRoute()
const bookId = computed(() => route.params.id as string)
</script>
