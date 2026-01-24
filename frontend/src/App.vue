<template>
  <div class="layout">
    <div class="icon-bar">
      <button class="icon-btn" @click="goHome" title="Новый запрос">
        <Plus class="icon" />
      </button>
      <button class="icon-btn" @click="showLibrary = true" title="Библиотека">
        <Library class="icon" />
      </button>
      <button class="icon-btn" @click="showBookmarksModal = true" title="Закладки">
        <Bookmark class="icon" />
      </button>
    </div>
    <aside class="sidebar">
      <div class="brand">
        <span class="brand-title">
          <Library class="icon icon--brand" />
          Библиотека
        </span>
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
          <div class="nav-actions">
            <button
              class="nav-action nav-action--active"
              title="Убрать из закладок"
              aria-label="Убрать из закладок"
              @click="removeBookmark(item.id)"
            >
              <Bookmark class="icon icon--tiny" />
            </button>
            <button
              class="nav-action nav-action--danger"
              title="Удалить диалог"
              aria-label="Удалить диалог"
              @click="deleteChat(item.id)"
            >
              <Trash2 class="icon icon--tiny" />
            </button>
          </div>
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
          <div class="nav-actions">
            <button
              class="nav-action"
              :class="{ 'nav-action--active': item.bookmarked }"
              :title="item.bookmarked ? 'Убрать из закладок' : 'Добавить в закладки'"
              :aria-label="item.bookmarked ? 'Убрать из закладок' : 'Добавить в закладки'"
              @click="toggleBookmark(item)"
            >
              <Bookmark class="icon icon--tiny" />
            </button>
            <button
              class="nav-action nav-action--danger"
              title="Удалить диалог"
              aria-label="Удалить диалог"
              @click="deleteChat(item.id)"
            >
              <Trash2 class="icon icon--tiny" />
            </button>
          </div>
        </div>
      </nav>
    </aside>

    <main class="main">
      <div class="main-header">
        <button class="settings-btn" @click="showSettings = true" aria-label="Настройки">
          <Settings class="icon icon--small" />
        </button>
      </div>
      <router-view />
    </main>
    <div v-if="showSettings" class="modal-backdrop" @click.self="showSettings = false">
      <div class="modal settings-modal">
        <div class="modal-header">
          <div class="modal-title">Настройки</div>
          <button class="modal-close-icon" @click="showSettings = false" aria-label="Закрыть">
            <X class="icon icon--small" />
          </button>
        </div>

        <div class="settings-content">
          <div class="settings-section">
            <div class="settings-section-title">Внешний вид</div>
            <div class="settings-item">
              <div class="settings-item-label">Тема оформления</div>
              <div class="theme-switcher">
                <button
                  v-for="opt in themeOptions"
                  :key="opt.value"
                  class="theme-option"
                  :class="{ 'theme-option--active': theme === opt.value }"
                  @click="setTheme(opt.value)"
                >
                  <component :is="opt.icon" class="icon icon--small" />
                  <span>{{ opt.label }}</span>
                </button>
              </div>
            </div>
          </div>

          <div class="settings-section">
            <div class="settings-section-title">Модель</div>
            <div class="settings-item">
              <div class="settings-item-label">Модель ответа</div>
              <div class="modal-options">
                <button
                  v-for="m in models"
                  :key="m"
                  class="model-option"
                  :class="{ 'model-option--active': m === selectedModel }"
                  @click="setModel(m)"
                >
                  {{ m }}
                </button>
              </div>
            </div>
          </div>
        </div>

        <div class="modal-actions">
          <button class="modal-close" @click="showSettings = false">Закрыть</button>
        </div>
      </div>
    </div>

    <div v-if="showLibrary" class="modal-backdrop" @click.self="showLibrary = false">
      <div class="modal library-modal">
        <div class="modal-header">
          <div class="modal-title">Все диалоги</div>
          <button class="modal-close-icon" @click="showLibrary = false" aria-label="Закрыть">
            <X class="icon icon--small" />
          </button>
        </div>
        <div class="library-list">
          <div v-if="libraryLoading" class="library-loading">Загрузка...</div>
          <div v-else-if="!libraryChats.length" class="library-empty">Нет диалогов</div>
          <div
            v-for="chat in libraryChats"
            :key="chat.id"
            class="library-row"
          >
            <button
              class="library-item"
              :class="{ 'library-item--active': isActiveChat(chat.id) }"
              @click="openChatFromLibrary(chat.id)"
            >
              <span class="library-item-title">{{ chat.title || 'Без названия' }}</span>
              <span class="library-item-date">{{ formatDate(chat.updated_at) }}</span>
            </button>
            <div class="library-actions">
              <button
                class="library-action"
                :class="{ 'library-action--active': chat.bookmarked }"
                :title="chat.bookmarked ? 'Убрать из закладок' : 'Добавить в закладки'"
                :aria-label="chat.bookmarked ? 'Убрать из закладок' : 'Добавить в закладки'"
                @click="toggleBookmarkInLibrary(chat)"
              >
                <Bookmark class="icon icon--tiny" />
              </button>
              <button
                class="library-action library-action--danger"
                title="Удалить диалог"
                aria-label="Удалить диалог"
                @click="deleteChatFromLibrary(chat.id)"
              >
                <Trash2 class="icon icon--tiny" />
              </button>
            </div>
          </div>
        </div>
        <div class="library-pagination">
          <button class="pagination-btn" :disabled="libraryPage === 1" @click="loadLibraryPage(libraryPage - 1)">
            <ChevronLeft class="icon icon--small" />
          </button>
          <span class="pagination-info">Страница {{ libraryPage }} из {{ libraryTotalPages }}</span>
          <button class="pagination-btn" :disabled="libraryPage >= libraryTotalPages" @click="loadLibraryPage(libraryPage + 1)">
            <ChevronRight class="icon icon--small" />
          </button>
        </div>
      </div>
    </div>

    <div v-if="showBookmarksModal" class="modal-backdrop" @click.self="showBookmarksModal = false">
      <div class="modal library-modal">
        <div class="modal-header">
          <div class="modal-title">Закладки</div>
          <button class="modal-close-icon" @click="showBookmarksModal = false" aria-label="Закрыть">
            <X class="icon icon--small" />
          </button>
        </div>
        <div class="library-list">
          <div v-if="bookmarksModalLoading" class="library-loading">Загрузка...</div>
          <div v-else-if="!bookmarksModalList.length" class="library-empty">Нет закладок</div>
          <div
            v-for="item in bookmarksModalList"
            :key="item.id"
            class="library-row"
          >
            <button
              class="library-item"
              :class="{ 'library-item--active': isActiveChat(item.id) }"
              @click="openChatFromBookmarksModal(item.id)"
            >
              <span class="library-item-title">{{ item.title || 'Без названия' }}</span>
              <span class="library-item-date">{{ formatDate(item.bookmarked_at) }}</span>
            </button>
            <div class="library-actions">
              <button
                class="library-action library-action--active"
                title="Убрать из закладок"
                aria-label="Убрать из закладок"
                @click="removeBookmarkFromModal(item.id)"
              >
                <Bookmark class="icon icon--tiny" />
              </button>
              <button
                class="library-action library-action--danger"
                title="Удалить диалог"
                aria-label="Удалить диалог"
                @click="deleteChatFromBookmarksModal(item.id)"
              >
                <Trash2 class="icon icon--tiny" />
              </button>
            </div>
          </div>
        </div>
        <div class="library-pagination">
          <button class="pagination-btn" :disabled="bookmarksModalPage === 1" @click="loadBookmarksModalPage(bookmarksModalPage - 1)">
            <ChevronLeft class="icon icon--small" />
          </button>
          <span class="pagination-info">Страница {{ bookmarksModalPage }} из {{ bookmarksModalTotalPages }}</span>
          <button class="pagination-btn" :disabled="bookmarksModalPage >= bookmarksModalTotalPages" @click="loadBookmarksModalPage(bookmarksModalPage + 1)">
            <ChevronRight class="icon icon--small" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Bookmark, ChevronLeft, ChevronRight, Library, MessageSquare, Monitor, Moon, Plus, Settings, Sun, Trash2, X } from 'lucide-vue-next'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { apiFetch } from './api'
import { useModelStore } from './modelStore'
import { useSettingsStore, type Theme } from './settingsStore'

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
const showSettings = ref(false)
const showLibrary = ref(false)
const libraryChats = ref<ChatItem[]>([])
const libraryPage = ref(1)
const libraryTotalPages = ref(1)
const libraryLoading = ref(false)
const libraryLimit = 20
const showBookmarksModal = ref(false)
const bookmarksModalList = ref<BookmarkItem[]>([])
const bookmarksModalPage = ref(1)
const bookmarksModalTotalPages = ref(1)
const bookmarksModalLoading = ref(false)
const bookmarksModalLimit = 20
const { models, selectedModel, loadModels, setModel } = useModelStore()
const { theme, setTheme, initTheme } = useSettingsStore()

const themeOptions: { value: Theme; label: string; icon: typeof Sun }[] = [
  { value: 'light', label: 'Светлая', icon: Sun },
  { value: 'dark', label: 'Тёмная', icon: Moon },
  { value: 'system', label: 'Системная', icon: Monitor }
]

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

async function removeBookmark(chatId: string) {
  const resp = await apiFetch(`/bookmarks/${chatId}`, { method: 'DELETE' })
  if (!resp.ok) return
  await loadSidebar()
}

async function deleteChat(chatId: string) {
  if (!confirm('Удалить этот диалог?')) return
  const resp = await apiFetch(`/chats/${chatId}`, { method: 'DELETE' })
  if (!resp.ok) return
  // Если удаляем текущий чат, переходим на главную
  if (activeChatId.value === chatId) {
    await goHome()
  }
  await loadSidebar()
}

function formatDate(input?: string) {
  if (!input) return ''
  const dt = new Date(input)
  if (Number.isNaN(dt.getTime())) return ''
  return dt.toLocaleDateString('ru-RU', { month: 'short', day: 'numeric' })
}

async function loadLibraryPage(page: number) {
  if (libraryLoading.value) return
  libraryLoading.value = true
  try {
    const offset = (page - 1) * libraryLimit
    const resp = await apiFetch(`/chats?limit=${libraryLimit}&offset=${offset}`)
    if (resp.ok) {
      const data = (await resp.json()) as { items?: ChatItem[]; total?: number }
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
  // Если удаляем текущий чат, переходим на главную
  if (activeChatId.value === chatId) {
    showLibrary.value = false
    await goHome()
  }
  // Перезагружаем список библиотеки
  await loadLibraryPage(libraryPage.value)
  await loadSidebar()
}

async function toggleBookmarkInLibrary(chat: ChatItem) {
  const url = `/bookmarks/${chat.id}`
  const resp = await apiFetch(url, { method: chat.bookmarked ? 'DELETE' : 'POST' })
  if (!resp.ok) return
  // Обновляем состояние локально для мгновенного отклика
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
      const data = (await resp.json()) as { items?: BookmarkItem[]; total?: number }
      if (Array.isArray(data.items)) {
        bookmarksModalList.value = data.items
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

async function removeBookmarkFromModal(chatId: string) {
  const resp = await apiFetch(`/bookmarks/${chatId}`, { method: 'DELETE' })
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
.nav-row {
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: center;
  gap: 4px;
  width: 100%;
}
.nav-actions {
  display: flex;
  gap: 2px;
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
  min-width: 0;
  overflow: hidden;
}
.nav-link:hover {
  background: var(--hover);
}
.nav-item--active {
  background: var(--accent-light);
}
.nav-title {
  font-size: 13px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.nav-meta {
  font-size: 11px;
  color: var(--muted);
}
.nav-action {
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
.nav-action:hover {
  background: var(--hover);
}
.nav-action--active {
  color: var(--accent);
  background: var(--accent-light);
}
.nav-action--danger {
  color: var(--muted);
}
.nav-action--danger:hover {
  color: var(--danger);
  background: var(--danger-light);
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
  color: var(--accent);
}
.nav-muted {
  margin-top: 10px;
  font-size: 12px;
  color: var(--muted);
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
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.3);
  display: grid;
  place-items: center;
  z-index: 50;
}
.modal {
  background: var(--card-bg);
  border-radius: 16px;
  padding: 18px;
  width: min(420px, 92vw);
  box-shadow: 0 20px 40px rgba(15, 23, 42, 0.3);
  display: grid;
  gap: 12px;
  border: 1px solid var(--border);
}
.modal-title {
  font-size: 14px;
  font-weight: 600;
}
.modal-section {
  display: grid;
  gap: 8px;
}
.modal-label {
  font-size: 12px;
  color: var(--muted);
}
.modal-options {
  display: grid;
  gap: 6px;
  max-height: 220px;
  overflow-y: auto;
}
.model-option {
  border: 0;
  background: transparent;
  text-align: left;
  padding: 8px 10px;
  border-radius: 10px;
  cursor: pointer;
  font-size: 12px;
  color: var(--fg);
}
.model-option:hover {
  background: var(--hover);
}
.model-option--active {
  background: var(--accent-light);
  color: var(--accent);
}
.modal-actions {
  display: flex;
  justify-content: flex-end;
}
.modal-close {
  border: 1px solid var(--border);
  background: var(--card-bg);
  padding: 6px 12px;
  border-radius: 10px;
  cursor: pointer;
  color: var(--fg);
  font-size: 13px;
}
.modal-close:hover {
  background: var(--hover);
}
.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.modal-close-icon {
  border: 0;
  background: transparent;
  cursor: pointer;
  padding: 4px;
  border-radius: 8px;
  display: grid;
  place-items: center;
  color: var(--muted);
}
.modal-close-icon:hover {
  background: var(--hover);
  color: var(--fg);
}
.library-modal {
  width: min(560px, 92vw);
  max-height: 80vh;
  display: grid;
  grid-template-rows: auto 1fr auto;
}
.library-list {
  overflow-y: auto;
  max-height: 400px;
  display: grid;
  gap: 4px;
  padding: 8px 0;
}
.library-loading,
.library-empty {
  font-size: 13px;
  color: var(--muted);
  padding: 12px;
  text-align: center;
}
.library-row {
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: center;
  gap: 4px;
}
.library-actions {
  display: flex;
  gap: 2px;
}
.library-item {
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
.library-item:hover {
  background: var(--hover);
}
.library-item--active {
  background: var(--accent-light);
}
.library-action {
  border: 0;
  background: transparent;
  cursor: pointer;
  display: grid;
  place-items: center;
  padding: 8px;
  border-radius: 8px;
  color: var(--muted);
}
.library-action:hover {
  background: var(--hover);
}
.library-action--active {
  color: var(--accent);
  background: var(--accent-light);
}
.library-action--danger:hover {
  color: var(--danger);
  background: var(--danger-light);
}
.library-item-title {
  font-size: 13px;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.library-item-date {
  font-size: 11px;
  color: var(--muted);
}
.library-pagination {
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

/* Settings Modal */
.settings-modal {
  width: min(480px, 92vw);
  max-height: 80vh;
  display: grid;
  grid-template-rows: auto 1fr auto;
}
.settings-content {
  overflow-y: auto;
  display: grid;
  gap: 20px;
  padding: 8px 0;
}
.settings-section {
  display: grid;
  gap: 12px;
}
.settings-section-title {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--muted);
}
.settings-item {
  display: grid;
  gap: 8px;
}
.settings-item-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--fg);
}

/* Theme Switcher */
.theme-switcher {
  display: flex;
  gap: 8px;
}
.theme-option {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 12px 8px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--card-bg);
  cursor: pointer;
  color: var(--fg);
  font-size: 12px;
  transition: all 0.15s;
}
.theme-option:hover {
  border-color: var(--accent);
  background: var(--accent-light);
}
.theme-option--active {
  border-color: var(--accent);
  background: var(--accent-light);
  color: var(--accent);
}
</style>
