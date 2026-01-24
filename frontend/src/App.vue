<template>
  <div class="layout">
    <IconBar
      @new-chat="goHome"
      @open-library="showLibrary = true"
      @open-bookmarks="showBookmarksModal = true"
    />

    <Sidebar
      :bookmarks="bookmarks"
      :recent-chats="recentChats"
      :active-chat-id="activeChatId"
      @open-chat="openChat"
      @toggle-bookmark="toggleBookmark"
      @remove-bookmark="removeBookmark"
      @delete-chat="deleteChat"
    />

    <main class="main">
      <div class="main-header">
        <button class="settings-btn" @click="showSettings = true" aria-label="Settings">
          <Settings class="icon icon--small" />
        </button>
      </div>
      <router-view />
    </main>

    <!-- Settings Modal -->
    <SettingsModal
      v-if="showSettings"
      :theme="theme"
      :models="models"
      :selected-model="selectedModel"
      @close="showSettings = false"
      @set-theme="setTheme"
      @set-model="setModel"
    />

    <!-- Library Modal -->
    <PaginatedListModal
      v-if="showLibrary"
      title="All chats"
      :items="libraryChats"
      :loading="libraryLoading"
      empty-text="No chats"
      :page="libraryPage"
      :total-pages="libraryTotalPages"
      :active-id="activeChatId"
      @close="showLibrary = false"
      @open="openChatFromLibrary"
      @toggle-bookmark="toggleBookmarkInLibrary"
      @delete="deleteChatFromLibrary"
      @prev-page="loadLibraryPage(libraryPage - 1)"
      @next-page="loadLibraryPage(libraryPage + 1)"
    />

    <!-- Bookmarks Modal -->
    <PaginatedListModal
      v-if="showBookmarksModal"
      title="Bookmarks"
      :items="bookmarksModalList"
      :loading="bookmarksModalLoading"
      empty-text="No bookmarks"
      :page="bookmarksModalPage"
      :total-pages="bookmarksModalTotalPages"
      :active-id="activeChatId"
      date-field="bookmarked_at"
      @close="showBookmarksModal = false"
      @open="openChatFromBookmarksModal"
      @toggle-bookmark="removeBookmarkFromModal"
      @delete="deleteChatFromBookmarksModal"
      @prev-page="loadBookmarksModalPage(bookmarksModalPage - 1)"
      @next-page="loadBookmarksModalPage(bookmarksModalPage + 1)"
    />
  </div>
</template>

<script setup lang="ts">
import { Settings } from 'lucide-vue-next'
import { storeToRefs } from 'pinia'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { apiFetch } from './api'
import PaginatedListModal, { type ListItem } from './components/modals/PaginatedListModal.vue'
import SettingsModal, { type Theme } from './components/modals/SettingsModal.vue'
import IconBar from './components/sidebar/IconBar.vue'
import Sidebar from './components/sidebar/Sidebar.vue'
import type { ChatItemData } from './components/sidebar/ChatListItem.vue'
import { useModelStore } from './stores/modelStore'
import { useSettingsStore } from './settingsStore'

interface BookmarkItem extends ChatItemData {
  bookmarked_at?: string
}

const route = useRoute()
const router = useRouter()
const recentChats = ref<ChatItemData[]>([])
const bookmarks = ref<BookmarkItem[]>([])
const isLoading = ref(false)
const showSettings = ref(false)
const showLibrary = ref(false)
const libraryChats = ref<ListItem[]>([])
const libraryPage = ref(1)
const libraryTotalPages = ref(1)
const libraryLoading = ref(false)
const libraryLimit = 20
const showBookmarksModal = ref(false)
const bookmarksModalList = ref<ListItem[]>([])
const bookmarksModalPage = ref(1)
const bookmarksModalTotalPages = ref(1)
const bookmarksModalLoading = ref(false)
const bookmarksModalLimit = 20
const modelStore = useModelStore()
const { models, selectedModel } = storeToRefs(modelStore)
const { loadModels, setModel } = modelStore
const settingsStore = useSettingsStore()
const { theme, setTheme, initTheme } = settingsStore

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
      const data = (await chatsResp.json()) as { items?: ChatItemData[] }
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

async function openChat(id: string) {
  await router.push({ name: 'chat', params: { chatId: id } })
}

async function goHome() {
  await router.push({ name: 'home' })
}

async function toggleBookmark(item: ChatItemData) {
  const url = `/bookmarks/${item.id}`
  const resp = await apiFetch(url, { method: item.bookmarked ? 'DELETE' : 'POST' })
  if (!resp.ok) return
  await loadSidebar()
}

async function removeBookmark(chatId: string) {
  const resp = await apiFetch(`/bookmarks/${chatId}`, { method: 'DELETE' })
  if (!resp.ok) return
  await loadSidebar()
}

async function deleteChat(chatId: string) {
  if (!confirm('Delete this chat?')) return
  const resp = await apiFetch(`/chats/${chatId}`, { method: 'DELETE' })
  if (!resp.ok) return
  if (activeChatId.value === chatId) {
    await goHome()
  }
  await loadSidebar()
}

async function loadLibraryPage(page: number) {
  if (libraryLoading.value) return
  libraryLoading.value = true
  try {
    const offset = (page - 1) * libraryLimit
    const resp = await apiFetch(`/chats?limit=${libraryLimit}&offset=${offset}`)
    if (resp.ok) {
      const data = (await resp.json()) as { items?: ListItem[]; total?: number }
      if (Array.isArray(data.items)) {
        libraryChats.value = data.items
      }
      const total = data.total || data.items?.length || 0
      libraryTotalPages.value = Math.max(1, Math.ceil(total / libraryLimit))
      libraryPage.value = page
    }
  } finally {
    libraryLoading.value = false
  }
}

async function openChatFromLibrary(id: string) {
  showLibrary.value = false
  await openChat(id)
}

async function deleteChatFromLibrary(chatId: string) {
  if (!confirm('Удалить этот диалог?')) return
  const resp = await apiFetch(`/chats/${chatId}`, { method: 'DELETE' })
  if (!resp.ok) return
  if (activeChatId.value === chatId) {
    showLibrary.value = false
    await goHome()
  }
  await loadLibraryPage(libraryPage.value)
  await loadSidebar()
}

async function toggleBookmarkInLibrary(chat: ListItem) {
  const url = `/bookmarks/${chat.id}`
  const resp = await apiFetch(url, { method: chat.bookmarked ? 'DELETE' : 'POST' })
  if (!resp.ok) return
  chat.bookmarked = !chat.bookmarked
  await loadSidebar()
}

async function loadBookmarksModalPage(page: number) {
  if (bookmarksModalLoading.value) return
  bookmarksModalLoading.value = true
  try {
    const offset = (page - 1) * bookmarksModalLimit
    const resp = await apiFetch(`/bookmarks?limit=${bookmarksModalLimit}&offset=${offset}`)
    if (resp.ok) {
      const data = (await resp.json()) as { items?: ListItem[]; total?: number }
      if (Array.isArray(data.items)) {
        bookmarksModalList.value = data.items.map(item => ({ ...item, bookmarked: true }))
      }
      const total = data.total || data.items?.length || 0
      bookmarksModalTotalPages.value = Math.max(1, Math.ceil(total / bookmarksModalLimit))
      bookmarksModalPage.value = page
    }
  } finally {
    bookmarksModalLoading.value = false
  }
}

async function openChatFromBookmarksModal(id: string) {
  showBookmarksModal.value = false
  await openChat(id)
}

async function removeBookmarkFromModal(item: ListItem) {
  const resp = await apiFetch(`/bookmarks/${item.id}`, { method: 'DELETE' })
  if (!resp.ok) return
  await loadBookmarksModalPage(bookmarksModalPage.value)
  await loadSidebar()
}

async function deleteChatFromBookmarksModal(chatId: string) {
  if (!confirm('Удалить этот диалог?')) return
  const resp = await apiFetch(`/chats/${chatId}`, { method: 'DELETE' })
  if (!resp.ok) return
  if (activeChatId.value === chatId) {
    showBookmarksModal.value = false
    await goHome()
  }
  await loadBookmarksModalPage(bookmarksModalPage.value)
  await loadSidebar()
}

onMounted(() => {
  void loadSidebar()
  void loadModels()
  initTheme()
})

watch(
  () => route.params.chatId,
  () => {
    void loadSidebar()
  }
)

watch(showLibrary, (val) => {
  if (val) {
    void loadLibraryPage(1)
  }
})

watch(showBookmarksModal, (val) => {
  if (val) {
    void loadBookmarksModalPage(1)
  }
})
</script>

<style scoped>
.layout {
  display: grid;
  grid-template-columns: 56px 280px 1fr;
  min-height: 100vh;
  background: var(--bg);
  color: var(--fg);
}

.main {
  padding: 24px;
}

.main-header {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 12px;
}

.settings-btn {
  border: 1px solid var(--border);
  background: var(--card-bg);
  border-radius: 12px;
  width: 36px;
  height: 36px;
  display: grid;
  place-items: center;
  cursor: pointer;
  color: var(--fg);
}

.settings-btn:hover {
  background: var(--hover);
}

.icon--small {
  width: 14px;
  height: 14px;
}
</style>
