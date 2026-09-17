<script setup lang="ts">
import { useRoute } from 'vue-router'
import BookReader from '@/components/reader/BookReader.vue'

const route = useRoute()
const bookId = route.params.id as string
</script>

<template>
  <BookReader :book-id="bookId" />
</template>
