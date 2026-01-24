<template>
  <div class="icon-bar">
    <button class="icon-btn" @click="$emit('newChat')" title="Новый запрос">
      <Plus class="icon" />
    </button>
    <button class="icon-btn" @click="$emit('openLibrary')" title="Библиотека">
      <Library class="icon" />
    </button>
    <button class="icon-btn" @click="$emit('openBookmarks')" title="Закладки">
      <Bookmark class="icon" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { Bookmark, Library, Plus } from 'lucide-vue-next'

defineEmits<{
  newChat: []
  openLibrary: []
  openBookmarks: []
}>()
</script>

<style scoped>
.icon-bar {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 16px 8px;
  background: var(--sidebar-bg);
  border-right: 1px solid var(--border);
}

.icon-btn {
  width: 40px;
  height: 40px;
  border-radius: 12px;
  border: 1px solid var(--border);
  background: var(--card-bg);
  display: grid;
  place-items: center;
  cursor: pointer;
  color: var(--fg);
  transition: background 0.15s, color 0.15s, border-color 0.15s;
}

.icon-btn:hover {
  background: var(--accent);
  color: #fff;
  border-color: var(--accent);
}

.icon {
  width: 16px;
  height: 16px;
}
</style>
