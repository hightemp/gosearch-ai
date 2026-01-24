<template>
  <div class="chat-header">
    <div class="tabs">
      <button
        v-for="tab in tabs"
        :key="tab.value"
        class="tab"
        :class="{ 'tab--active': activeTab === tab.value }"
        @click="$emit('update:activeTab', tab.value)"
      >
        <component :is="tab.icon" class="tab-icon" />
        {{ tab.label }}
      </button>
    </div>
    <div v-if="showActions" class="chat-actions">
      <button
        class="bookmark-btn"
        :class="{ 'bookmark-btn--active': isBookmarked }"
        :title="isBookmarked ? 'Remove from favorites' : 'Add to favorites'"
        @click="$emit('toggleBookmark')"
      >
        <Bookmark class="bookmark-icon" />
        {{ isBookmarked ? 'In favorites' : 'Favorite' }}
      </button>
      <button
        class="delete-btn"
        title="Delete chat"
        @click="$emit('delete')"
      >
        <Trash2 class="delete-icon" />
        Delete
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Bookmark, Image, Link, ListChecks, MessageSquare, Trash2 } from 'lucide-vue-next'

export type TabValue = 'answer' | 'steps' | 'links' | 'images'

const tabs: { value: TabValue; label: string; icon: typeof MessageSquare }[] = [
  { value: 'answer', label: 'Answer', icon: MessageSquare },
  { value: 'steps', label: 'Steps', icon: ListChecks },
  { value: 'links', label: 'Links', icon: Link },
  { value: 'images', label: 'Images', icon: Image }
]

defineProps<{
  activeTab: TabValue
  showActions?: boolean
  isBookmarked?: boolean
}>()

defineEmits<{
  'update:activeTab': [tab: TabValue]
  toggleBookmark: []
  delete: []
}>()
</script>

<style scoped>
.chat-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  border-bottom: 1px solid var(--border);
  padding-bottom: 10px;
  gap: 16px;
}

.tabs {
  display: flex;
  gap: 18px;
}

.tab {
  border: 0;
  background: transparent;
  padding: 10px 6px;
  cursor: pointer;
  color: var(--muted);
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.tab--active {
  color: var(--fg);
  border-bottom: 2px solid var(--fg);
}

.tab-icon {
  width: 14px;
  height: 14px;
}

.chat-actions {
  display: flex;
  gap: 8px;
}

.bookmark-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--card-bg);
  font-size: 12px;
  color: var(--muted);
  cursor: pointer;
  white-space: nowrap;
}

.bookmark-btn:hover {
  background: var(--hover);
  color: var(--fg);
}

.bookmark-btn--active {
  background: var(--accent-light);
  border-color: var(--accent);
  color: var(--accent);
}

.bookmark-btn--active:hover {
  background: var(--accent-light);
}

.bookmark-icon {
  width: 14px;
  height: 14px;
}

.delete-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--card-bg);
  font-size: 12px;
  color: var(--muted);
  cursor: pointer;
  white-space: nowrap;
}

.delete-btn:hover {
  background: var(--danger-light);
  border-color: var(--danger);
  color: var(--danger);
}

.delete-icon {
  width: 14px;
  height: 14px;
}
</style>
