<template>
  <aside class="sidebar">
    <div class="brand">
      <span class="brand-title">
        <Library class="icon icon--brand" />
        Library
      </span>
    </div>
    <nav class="nav">
      <div class="nav-section">
        <Bookmark class="icon icon--small" />
        Bookmarks
      </div>
      <div v-if="!bookmarks.length" class="nav-muted">No bookmarks yet.</div>
      <ChatListItem
        v-for="item in bookmarks"
        :key="item.id"
        :item="{ ...item, bookmarked: true }"
        :is-active="isActiveChat(item.id)"
        date-field="bookmarked_at"
        @open="$emit('openChat', $event)"
        @toggle-bookmark="$emit('removeBookmark', item.id)"
        @delete="$emit('deleteChat', $event)"
      />

      <div class="nav-section">
        <MessageSquare class="icon icon--small" />
        Recent
      </div>
      <div v-if="!recentChats.length" class="nav-muted">No queries yet.</div>
      <ChatListItem
        v-for="item in recentChats"
        :key="item.id"
        :item="item"
        :is-active="isActiveChat(item.id)"
        @open="$emit('openChat', $event)"
        @toggle-bookmark="$emit('toggleBookmark', $event)"
        @delete="$emit('deleteChat', $event)"
      />
    </nav>
  </aside>
</template>

<script setup lang="ts">
import { Bookmark, Library, MessageSquare } from 'lucide-vue-next'
import ChatListItem, { type ChatItemData } from './ChatListItem.vue'

export interface BookmarkItemData extends ChatItemData {
  bookmarked_at?: string
}

const props = defineProps<{
  bookmarks: BookmarkItemData[]
  recentChats: ChatItemData[]
  activeChatId: string
}>()

defineEmits<{
  openChat: [id: string]
  toggleBookmark: [item: ChatItemData]
  removeBookmark: [id: string]
  deleteChat: [id: string]
}>()

function isActiveChat(id: string) {
  return props.activeChatId === id
}
</script>

<script lang="ts">
// Re-export for convenience
export { type ChatItemData } from './ChatListItem.vue'
</script>

<style scoped>
.sidebar {
  border-right: 1px solid var(--border);
  padding: 20px 16px;
  overflow: hidden;
  background: var(--sidebar-bg);
}

.brand {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 14px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 10px;
}

.brand-title {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.nav-section {
  margin-top: 18px;
  font-size: 12px;
  color: var(--muted);
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.nav-muted {
  margin-top: 10px;
  font-size: 12px;
  color: var(--muted);
}

.icon {
  width: 16px;
  height: 16px;
}

.icon--small {
  width: 14px;
  height: 14px;
}

.icon--brand {
  width: 18px;
  height: 18px;
}
</style>
