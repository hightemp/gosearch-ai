<template>
  <div class="pagination">
    <button
      class="pagination-btn"
      :disabled="page === 1"
      @click="$emit('prev')"
    >
      <ChevronLeft class="icon icon--small" />
    </button>
    <span class="pagination-info">Page {{ page }} of {{ totalPages }}</span>
    <button
      class="pagination-btn"
      :disabled="page >= totalPages"
      @click="$emit('next')"
    >
      <ChevronRight class="icon icon--small" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { ChevronLeft, ChevronRight } from 'lucide-vue-next'

defineProps<{
  page: number
  totalPages: number
}>()

defineEmits<{
  prev: []
  next: []
}>()
</script>

<style scoped>
.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--border);
}

.pagination-btn {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  border: 1px solid var(--border);
  background: var(--card-bg);
  display: grid;
  place-items: center;
  cursor: pointer;
  color: var(--fg);
}

.pagination-btn:hover:not(:disabled) {
  background: var(--hover);
}

.pagination-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.pagination-info {
  font-size: 12px;
  color: var(--muted);
}

.icon--small {
  width: 14px;
  height: 14px;
}
</style>
