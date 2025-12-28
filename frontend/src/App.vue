<template>
  <div class="layout">
    <aside class="sidebar">
      <div class="brand">Библиотека</div>
      <nav class="nav">
        <div class="nav-section">Закладки</div>
        <div v-if="!bookmarks.length" class="nav-muted">Пока нет закладок.</div>
        <button
          v-for="item in bookmarks"
          :key="item.id"
          class="nav-item"
          :class="{ 'nav-item--active': isActiveChat(item.id) }"
          @click="openChat(item.id)"
        >
          <span class="nav-title">{{ item.title || 'Без названия' }}</span>
          <span class="nav-meta">{{ formatDate(item.bookmarked_at || item.updated_at) }}</span>
        </button>

        <div class="nav-section">Недавние</div>
        <div v-if="!recentChats.length" class="nav-muted">Пока нет запросов.</div>
        <button
          v-for="item in recentChats"
          :key="item.id"
          class="nav-item"
          :class="{ 'nav-item--active': isActiveChat(item.id) }"
          @click="openChat(item.id)"
        >
          <span class="nav-title">{{ item.title || 'Без названия' }}</span>
          <span class="nav-meta">{{ formatDate(item.updated_at) }}</span>
          <span class="nav-actions">
            <button class="nav-action" @click.stop="toggleBookmark(item)">
              {{ item.bookmarked ? 'Убрать' : 'В закладки' }}
            </button>
          </span>
        </button>
      </nav>
    </aside>

    <main class="main">
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

type ChatItem = {
  id: string
  title: string
  pinned: boolean
  bookmarked: boolean
  created_at: string
  updated_at: string
}

type BookmarkItem = {
  id: string
  title: string
  pinned: boolean
  created_at: string
  updated_at: string
  bookmarked_at: string
}

const route = useRoute()
const router = useRouter()
const recentChats = ref<ChatItem[]>([])
const bookmarks = ref<BookmarkItem[]>([])
const isLoading = ref(false)

const activeChatId = computed(() => String(route.params.chatId || '').trim())

async function loadSidebar() {
  if (isLoading.value) return
  isLoading.value = true
  try {
    const [chatsResp, bookmarksResp] = await Promise.all([
      fetch('/api/chats?limit=20'),
      fetch('/api/bookmarks?limit=20')
    ])
    if (chatsResp.ok) {
      const data = (await chatsResp.json()) as { items?: ChatItem[] }
      if (Array.isArray(data.items)) {
        recentChats.value = data.items
      }
    }
    if (bookmarksResp.ok) {
      const data = (await bookmarksResp.json()) as { items?: BookmarkItem[] }
      if (Array.isArray(data.items)) {
        bookmarks.value = data.items
      }
    }
  } finally {
    isLoading.value = false
  }
}

function isActiveChat(id: string) {
  return activeChatId.value === id
}

async function openChat(id: string) {
  await router.push({ name: 'chat', params: { chatId: id } })
}

async function toggleBookmark(item: ChatItem) {
  const url = `/api/bookmarks/${item.id}`
  const resp = await fetch(url, { method: item.bookmarked ? 'DELETE' : 'POST' })
  if (!resp.ok) return
  await loadSidebar()
}

function formatDate(input?: string) {
  if (!input) return ''
  const dt = new Date(input)
  if (Number.isNaN(dt.getTime())) return ''
  return dt.toLocaleDateString('ru-RU', { month: 'short', day: 'numeric' })
}

onMounted(() => {
  void loadSidebar()
})

watch(
  () => route.params.chatId,
  () => {
    void loadSidebar()
  }
)
</script>

<style scoped>
.layout {
  display: grid;
  grid-template-columns: 280px 1fr;
  min-height: 100vh;
  background: var(--bg);
  color: var(--fg);
}
.sidebar {
  border-right: 1px solid var(--border);
  padding: 20px 16px;
}
.brand {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 14px;
}
.nav-section {
  margin-top: 18px;
  font-size: 12px;
  color: var(--muted);
}
.nav-item {
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
}
.nav-item:hover {
  background: var(--hover);
}
.nav-item--active {
  background: #eef2f2;
}
.nav-title {
  font-size: 13px;
  font-weight: 600;
}
.nav-meta {
  font-size: 11px;
  color: var(--muted);
}
.nav-actions {
  display: flex;
  justify-content: flex-end;
}
.nav-action {
  border: 0;
  background: transparent;
  font-size: 11px;
  color: #0f766e;
  cursor: pointer;
}
.nav-action:hover {
  text-decoration: underline;
}
.nav-muted {
  margin-top: 10px;
  font-size: 12px;
  color: var(--muted);
}
.main {
  padding: 24px;
}
</style>
