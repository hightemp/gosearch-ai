<template>
  <div class="layout">
    <aside class="sidebar">
      <div class="brand">
        <span class="brand-title">
          <Library class="icon icon--brand" />
          Библиотека
        </span>
        <button class="home-link" @click="goHome">
          <Home class="icon icon--small" />
          На главную
        </button>
      </div>
      <nav class="nav">
        <div class="nav-section">
          <Bookmark class="icon icon--small" />
          Закладки
        </div>
        <div v-if="!bookmarks.length" class="nav-muted">Пока нет закладок.</div>
        <div v-for="item in bookmarks" :key="item.id" class="nav-row">
          <button class="nav-link" :class="{ 'nav-item--active': isActiveChat(item.id) }" @click="openChat(item.id)">
            <span class="nav-title">{{ item.title || 'Без названия' }}</span>
            <span class="nav-meta">{{ formatDate(item.bookmarked_at || item.updated_at) }}</span>
          </button>
        </div>

        <div class="nav-section">
          <MessageSquare class="icon icon--small" />
          Недавние
        </div>
        <div v-if="!recentChats.length" class="nav-muted">Пока нет запросов.</div>
        <div v-for="item in recentChats" :key="item.id" class="nav-row">
          <button class="nav-link" :class="{ 'nav-item--active': isActiveChat(item.id) }" @click="openChat(item.id)">
            <span class="nav-title">{{ item.title || 'Без названия' }}</span>
            <span class="nav-meta">{{ formatDate(item.updated_at) }}</span>
          </button>
          <button
            class="nav-action"
            :class="{ 'nav-action--active': item.bookmarked }"
            :title="item.bookmarked ? 'Убрать из закладок' : 'Добавить в закладки'"
            :aria-label="item.bookmarked ? 'Убрать из закладок' : 'Добавить в закладки'"
            @click="toggleBookmark(item)"
          >
            <Bookmark class="icon icon--tiny" />
          </button>
        </div>
      </nav>
    </aside>

    <main class="main">
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
import { Bookmark, Home, Library, MessageSquare } from 'lucide-vue-next'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { apiFetch } from './api'

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
      apiFetch('/chats?limit=20'),
      apiFetch('/bookmarks?limit=20')
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

async function goHome() {
  await router.push({ name: 'home' })
}

async function toggleBookmark(item: ChatItem) {
  const url = `/bookmarks/${item.id}`
  const resp = await apiFetch(url, { method: item.bookmarked ? 'DELETE' : 'POST' })
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
.home-link {
  border: 0;
  background: transparent;
  font-size: 11px;
  color: #0f766e;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.home-link:hover {
  text-decoration: underline;
}
.nav-section {
  margin-top: 18px;
  font-size: 12px;
  color: var(--muted);
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.nav-row {
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: center;
  gap: 8px;
  width: 100%;
}
.nav-link {
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
}
.nav-link:hover {
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
.nav-action {
  border: 0;
  background: transparent;
  font-size: 11px;
  color: #0f766e;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px;
  border-radius: 8px;
}
.nav-action:hover {
  background: var(--hover);
}
.nav-action--active {
  color: #0f766e;
  background: rgba(15, 118, 110, 0.1);
}
.icon {
  width: 16px;
  height: 16px;
}
.icon--small {
  width: 14px;
  height: 14px;
}
.icon--tiny {
  width: 12px;
  height: 12px;
}
.icon--brand {
  width: 18px;
  height: 18px;
}
.icon--active {
  color: #0f766e;
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
