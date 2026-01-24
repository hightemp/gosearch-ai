<template>
  <ModalBackdrop :title="title" modal-class="paginated-modal" @close="$emit('close')">
    <div class="list-content">
      <div v-if="loading" class="list-loading">Загрузка...</div>
      <div v-else-if="!items.length" class="list-empty">{{ emptyText }}</div>
      <div
        v-for="item in items"
        :key="item.id"
        class="list-row"
      >
        <button
          class="list-item"
          :class="{ 'list-item--active': item.id === activeId }"
          @click="$emit('open', item.id)"
        >
          <span class="list-item-title">{{ item.title || 'Без названия' }}</span>
          <span class="list-item-date">{{ formatDate(getItemDate(item)) }}</span>
        </button>
        <div class="list-actions">
          <button
            class="list-action"
            :class="{ 'list-action--active': item.bookmarked }"
            :title="item.bookmarked ? 'Убрать из закладок' : 'Добавить в закладки'"
            :aria-label="item.bookmarked ? 'Убрать из закладок' : 'Добавить в закладки'"
            @click="$emit('toggleBookmark', item)"
          >
            <Bookmark class="icon icon--tiny" />
          </button>
          <button
            class="list-action list-action--danger"
            title="Удалить диалог"
            aria-label="Удалить диалог"
            @click="$emit('delete', item.id)"
          >
            <Trash2 class="icon icon--tiny" />
          </button>
        </div>
      </div>
    </div>
    <PaginationControls
      :page="page"
      :total-pages="totalPages"
      @prev="$emit('prevPage')"
      @next="$emit('nextPage')"
    />
  </ModalBackdrop>
</template>

<script setup lang="ts">
import { Bookmark, Trash2 } from 'lucide-vue-next'
import ModalBackdrop from '../common/ModalBackdrop.vue'
import PaginationControls from '../common/PaginationControls.vue'

export interface ListItem {
  id: string
  title: string
  bookmarked?: boolean
  updated_at?: string
  bookmarked_at?: string
}

const props = defineProps<{
  title: string
  items: ListItem[]
  loading?: boolean
  emptyText?: string
  page: number
  totalPages: number
  activeId?: string
  dateField?: 'updated_at' | 'bookmarked_at'
}>()

defineEmits<{
  close: []
  open: [id: string]
  toggleBookmark: [item: ListItem]
  delete: [id: string]
  prevPage: []
  nextPage: []
}>()

function getItemDate(item: ListItem) {
  if (props.dateField === 'bookmarked_at') {
    return item.bookmarked_at || item.updated_at
  }
  return item.updated_at
}

function formatDate(input?: string) {
  if (!input) return ''
  const dt = new Date(input)
  if (Number.isNaN(dt.getTime())) return ''
  return dt.toLocaleDateString('ru-RU', { month: 'short', day: 'numeric' })
}
</script>

<style scoped>
:deep(.paginated-modal) {
  width: min(560px, 92vw);
  max-height: 80vh;
  display: grid;
  grid-template-rows: auto 1fr auto;
}

.list-content {
  overflow-y: auto;
  max-height: 400px;
  display: grid;
  gap: 4px;
  padding: 8px 0;
}

.list-loading,
.list-empty {
  font-size: 13px;
  color: var(--muted);
  padding: 12px;
  text-align: center;
}

.list-row {
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: center;
  gap: 4px;
}

.list-actions {
  display: flex;
  gap: 2px;
}

.list-item {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 12px;
  align-items: center;
  padding: 10px 12px;
  border-radius: 10px;
  border: 0;
  background: transparent;
  text-align: left;
  cursor: pointer;
  color: var(--fg);
  min-width: 0;
}

.list-item:hover {
  background: var(--hover);
}

.list-item--active {
  background: var(--accent-light);
}

.list-action {
  border: 0;
  background: transparent;
  cursor: pointer;
  display: grid;
  place-items: center;
  padding: 8px;
  border-radius: 8px;
  color: var(--muted);
}

.list-action:hover {
  background: var(--hover);
}

.list-action--active {
  color: var(--accent);
  background: var(--accent-light);
}

.list-action--danger:hover {
  color: var(--danger);
  background: var(--danger-light);
}

.list-item-title {
  font-size: 13px;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.list-item-date {
  font-size: 11px;
  color: var(--muted);
}

.icon--tiny {
  width: 12px;
  height: 12px;
}
</style>
