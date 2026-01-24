<template>
  <div class="chat-row">
    <button
      class="chat-link"
      :class="{ 'chat-link--active': isActive }"
      @click="$emit('open', item.id)"
    >
      <span class="chat-title">{{ item.title || 'Без названия' }}</span>
      <span class="chat-meta">{{ formatDate(displayDate) }}</span>
    </button>
    <div class="chat-actions">
      <button
        class="chat-action"
        :class="{ 'chat-action--active': item.bookmarked }"
        :title="item.bookmarked ? 'Убрать из закладок' : 'Добавить в закладки'"
        :aria-label="item.bookmarked ? 'Убрать из закладок' : 'Добавить в закладки'"
        @click="$emit('toggleBookmark', item)"
      >
        <Bookmark class="icon icon--tiny" />
      </button>
      <button
        class="chat-action chat-action--danger"
        title="Удалить диалог"
        aria-label="Удалить диалог"
        @click="$emit('delete', item.id)"
      >
        <Trash2 class="icon icon--tiny" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Bookmark, Trash2 } from 'lucide-vue-next'
import { computed } from 'vue'

export interface ChatItemData {
  id: string
  title: string
  bookmarked?: boolean
  created_at?: string
  updated_at?: string
  bookmarked_at?: string
}

const props = defineProps<{
  item: ChatItemData
  isActive?: boolean
  dateField?: 'updated_at' | 'bookmarked_at'
}>()

defineEmits<{
  open: [id: string]
  toggleBookmark: [item: ChatItemData]
  delete: [id: string]
}>()

const displayDate = computed(() => {
  if (props.dateField === 'bookmarked_at') {
    return props.item.bookmarked_at || props.item.updated_at
  }
  return props.item.updated_at
})

function formatDate(input?: string) {
  if (!input) return ''
  const dt = new Date(input)
  if (Number.isNaN(dt.getTime())) return ''
  return dt.toLocaleDateString('ru-RU', { month: 'short', day: 'numeric' })
}
</script>

<style scoped>
.chat-row {
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: center;
  gap: 4px;
  width: 100%;
}

.chat-actions {
  display: flex;
  gap: 2px;
}

.chat-link {
  display: grid;
  gap: 6px;
  padding: 10px 10px;
  border-radius: 10px;
  text-decoration: none;
  border: 0;
  background: transparent;
  color: var(--fg);
  text-align: left;
  cursor: pointer;
  width: 100%;
  min-width: 0;
  overflow: hidden;
}

.chat-link:hover {
  background: var(--hover);
}

.chat-link--active {
  background: var(--accent-light);
}

.chat-title {
  font-size: 13px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-meta {
  font-size: 11px;
  color: var(--muted);
}

.chat-action {
  border: 0;
  background: transparent;
  font-size: 11px;
  color: var(--accent);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px;
  border-radius: 8px;
}

.chat-action:hover {
  background: var(--hover);
}

.chat-action--active {
  color: var(--accent);
  background: var(--accent-light);
}

.chat-action--danger {
  color: var(--muted);
}

.chat-action--danger:hover {
  color: var(--danger);
  background: var(--danger-light);
}

.icon--tiny {
  width: 12px;
  height: 12px;
}
</style>
