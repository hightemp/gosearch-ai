<template>
  <div class="chat">
    <ChatHeader
      v-model:active-tab="activeTab"
      :show-actions="!!currentChatId"
      :is-bookmarked="isBookmarked"
      @toggle-bookmark="toggleBookmark"
      @delete="deleteCurrentChat"
    />

    <div class="progress">
      <div class="dot" />
      <div class="text">{{ statusText }}</div>
    </div>
    <div v-if="runError" class="error-text">{{ runError }}</div>

    <!-- Answer Tab -->
    <div v-if="activeTab === 'answer'" class="answer">
      <div class="answer-title">Ответ</div>
      <div class="answer-card" ref="answerRef">
        <MessageList
          v-if="messages.length"
          :messages="messages"
          :is-running="isRunning"
        />
        <StepsList
          v-else
          :steps="steps"
          :is-running="isRunning"
        />
      </div>
    </div>

    <SourcesPanel
      v-if="activeTab === 'answer'"
      :sources="sources"
      :snippets="snippets"
      :show-note="showSourcesNote"
    />

    <!-- Steps Tab -->
    <div v-if="activeTab === 'steps'" class="steps">
      <div class="steps-title">Шаги</div>
      <StepsList :steps="steps" :is-running="isRunning" />
    </div>

    <!-- Links Tab -->
    <LinksPanel v-if="activeTab === 'links'" :sources="sources" />

    <!-- Images Tab -->
    <div v-if="activeTab === 'images'" class="links">
      <div class="sources-title">Изображения</div>
      <div class="sources-card">
        <div class="sources-empty">Поддержка изображений появится позже.</div>
      </div>
    </div>

    <ChatComposer
      v-model="followup"
      placeholder="Добавить детали или пояснения..."
      :can-submit="!!followup.trim()"
      :models="models"
      :selected-model="selectedModel"
      :is-loading-models="isLoadingModels"
      @submit="submitFollowup"
      @select-model="setModel"
    />
  </div>
</template>

<script setup lang="ts">
import hljs from 'highlight.js'
import MarkdownIt from 'markdown-it'
import mk from 'markdown-it-katex'
import { storeToRefs } from 'pinia'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { apiFetch, apiUrl } from '../api'
import ChatComposer from '../components/chat/ChatComposer.vue'
import ChatHeader, { type TabValue } from '../components/chat/ChatHeader.vue'
import LinksPanel from '../components/chat/LinksPanel.vue'
import MessageList from '../components/chat/MessageList.vue'
import SourcesPanel from '../components/chat/SourcesPanel.vue'
import StepsList, { type Step } from '../components/chat/StepsList.vue'
import { useModelStore } from '../stores/modelStore'

type Source = { url: string; title?: string }
type Snippet = { url: string; quote: string; ref: number }
type AnswerDelta = { delta?: string }
type AnswerFinal = { answer?: string; model?: string }
type ChatMessage = { id: string; role: 'user' | 'assistant'; content: string }
type ChatMeta = { last_run_id?: string; bookmarked?: boolean }

const route = useRoute()
const router = useRouter()
const steps = ref<Step[]>([])
const sources = ref<Source[]>([])
const snippets = ref<Snippet[]>([])
const runId = ref<string>('')
const isRunning = ref(false)
const runError = ref('')
const answerText = ref('')
const answerModel = ref('')
const activeTab = ref<TabValue>('answer')
const sourceTitles = ref<Record<string, string>>({})
const followup = ref('')
const eventSource = ref<EventSource | null>(null)
const currentChatId = ref('')
const lastQuery = ref('')
const history = ref<ChatMessage[]>([])
const answerRef = ref<HTMLElement | null>(null)
const isBookmarked = ref(false)
const modelStore = useModelStore()
const { models, selectedModel, isLoadingModels } = storeToRefs(modelStore)
const { loadModels, setModel } = modelStore

const md = new MarkdownIt({
  html: false,
  linkify: true
}).use(mk)

md.renderer.rules.fence = (tokens, idx) => {
  const token = tokens[idx]
  const content = token.content || ''
  const lang = (token.info || '').trim().split(/\s+/)[0]
  const highlighted = lang && hljs.getLanguage(lang)
    ? hljs.highlight(content, { language: lang }).value
    : hljs.highlightAuto(content).value
  const escaped = md.utils.escapeHtml(content)
  const data = escaped.replace(/"/g, '&quot;')
  const icon =
    '<svg viewBox="0 0 24 24" width="14" height="14" aria-hidden="true">' +
    '<path d="M8 7a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2h-8a2 2 0 0 1-2-2z" fill="none" stroke="currentColor" stroke-width="2"/>' +
    '<path d="M6 17H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2v1" fill="none" stroke="currentColor" stroke-width="2"/>' +
    '</svg>'
  return (
    `<div class="code-block">` +
    `<button class="code-copy" data-copy="code" data-content="${data}" aria-label="Copy">${icon}</button>` +
    `<pre><code class="hljs${lang ? ` language-${lang}` : ''}">${highlighted || escaped}</code></pre>` +
    `</div>`
  )
}

const statusText = computed(() => {
  if (runError.value) return 'Ошибка'
  if (isRunning.value) return 'Активен...'
  if (runId.value) return 'Завершено'
  return 'Ожидание...'
})

const lastAssistantId = computed(() => {
  for (let i = history.value.length - 1; i >= 0; i -= 1) {
    if (history.value[i].role === 'assistant') return history.value[i].id
  }
  return ''
})

const citationSources = computed<Source[]>(() => {
  if (!snippets.value.length) return []
  const ordered = [...snippets.value].sort((a, b) => a.ref - b.ref)
  return ordered.map((snip) => ({
    url: snip.url,
    title: sourceTitles.value[snip.url] || ''
  }))
})

const messages = computed(() => {
  return history.value.map((msg) => {
    const roleLabel = msg.role === 'assistant' ? 'Ассистент' : 'Вы'
    const isLatestAssistant = msg.id === lastAssistantId.value
    const citations = isLatestAssistant ? citationSources.value : []
    const html = msg.role === 'assistant' ? renderMarkdown(msg.content, citations) : ''
    const modelLabel = isLatestAssistant ? answerModel.value : ''
    return { ...msg, roleLabel, html, modelLabel }
  })
})

const showSourcesNote = computed(() => history.value.filter((msg) => msg.role === 'assistant').length > 1)

async function startRun(queryText: string, model: string, chatId?: string) {
  if (!queryText) return

  runError.value = ''
  if (!chatId) {
    steps.value = []
    sources.value = []
    snippets.value = []
    sourceTitles.value = {}
    history.value = []
  }
  answerText.value = ''
  isRunning.value = true
  if (eventSource.value) {
    eventSource.value.close()
    eventSource.value = null
  }
  const resp = await apiFetch('/runs/start', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query: queryText, model, chat_id: chatId })
  })
  if (!resp.ok) {
    isRunning.value = false
    runError.value = 'Не удалось запустить поиск.'
    return
  }
  const data = await resp.json()
  runId.value = data.run_id
  if (!currentChatId.value) {
    currentChatId.value = data.chat_id
    if (String(route.params.chatId) !== currentChatId.value) {
      await router.replace({ name: 'chat', params: { chatId: currentChatId.value }, query: { model } })
    }
  }
  history.value.push({ id: `user-${runId.value}`, role: 'user', content: queryText })

  if (currentChatId.value) {
    await loadHistory(currentChatId.value)
  }

  const es = new EventSource(apiUrl(`/runs/${runId.value}/stream`))
  eventSource.value = es

  es.addEventListener('step', (ev: MessageEvent) => {
    const obj = JSON.parse(ev.data) as Step
    steps.value.push(obj)

    if (obj.type === 'search.results' && obj.payload?.results) {
      const mapped = (obj.payload.results as any[]).map((r) => ({ url: r.url, title: r.title }))
      sources.value = mapped
      sourceTitles.value = mapped.reduce<Record<string, string>>((acc, item) => {
        if (item.url && item.title) acc[item.url] = item.title
        return acc
      }, {})
    }
    if ((obj.type === 'source.selected' || obj.type === 'sources.selected') && obj.payload?.urls) {
      sources.value = (obj.payload.urls as string[]).map((url, i) => ({
        url,
        title: sourceTitles.value[url] || `Источник ${i + 1}`
      }))
    }
    if ((obj.type === 'snippet.extracted' || obj.type === 'snippets.extracted') && obj.payload?.snippets) {
      const payloadSnippets = obj.payload.snippets as any[]
      snippets.value = payloadSnippets
        .filter((snip) => snip?.url && snip?.quote)
        .map((snip) => ({ url: snip.url, quote: snip.quote, ref: Number(snip.ref) || 0 }))
        .sort((a, b) => a.ref - b.ref)
    }
  })

  es.addEventListener('answer.delta', (ev: MessageEvent) => {
    const obj = JSON.parse(ev.data) as AnswerDelta
    if (obj.delta) {
      answerText.value += obj.delta
      upsertAssistant(answerText.value)
    }
  })

  es.addEventListener('answer.final', (ev: MessageEvent) => {
    const obj = JSON.parse(ev.data) as AnswerFinal
    if (obj.answer) {
      answerText.value = obj.answer
    }
    if (obj.model) {
      answerModel.value = obj.model
    }
    upsertAssistant(answerText.value)
    isRunning.value = false
  })

  es.addEventListener('run.error', (ev: MessageEvent) => {
    const obj = JSON.parse(ev.data) as { error?: string }
    isRunning.value = false
    runError.value = obj.error || 'Ошибка выполнения pipeline.'
    es.close()
  })

  es.onerror = () => {
    isRunning.value = false
    runError.value = 'Потеряно соединение SSE. Обновите страницу и попробуйте снова.'
    es.close()
  }
}

function decodeHtmlEntities(input: string) {
  return input
    .replace(/&quot;/g, '"')
    .replace(/&apos;/g, "'")
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&amp;/g, '&')
}

function copyText(text: string) {
  const clean = decodeHtmlEntities(text)
  if (navigator.clipboard?.writeText) {
    void navigator.clipboard.writeText(clean)
    return
  }
  const textarea = document.createElement('textarea')
  textarea.value = clean
  textarea.style.position = 'fixed'
  textarea.style.left = '-9999px'
  document.body.appendChild(textarea)
  textarea.select()
  document.execCommand('copy')
  document.body.removeChild(textarea)
}

const codeCopyHandler = (ev: Event) => {
  const target = ev.target as HTMLElement | null
  if (!target) return
  const button = target.closest<HTMLButtonElement>('[data-copy="code"]')
  if (!button) return
  const content = button.getAttribute('data-content') || ''
  copyText(content)
}

onMounted(() => {
  void hydrateFromRoute()
  void loadModels()
  if (answerRef.value) {
    answerRef.value.addEventListener('click', codeCopyHandler)
  }
})

onBeforeUnmount(() => {
  if (answerRef.value) {
    answerRef.value.removeEventListener('click', codeCopyHandler)
  }
})

watch(
  () => [route.params.chatId, route.query.q],
  () => {
    void hydrateFromRoute()
  }
)

async function hydrateFromRoute() {
  const q = String(route.query.q || '').trim()
  const routeModel = String(route.query.model || '').trim()
  if (routeModel) {
    setModel(routeModel)
  }
  const model = String(routeModel || selectedModel.value || '').trim()
  const chatId = String(route.params.chatId || '').trim()
  if (q) {
    if (q !== lastQuery.value) {
      lastQuery.value = q
      await startRun(q, model)
    }
    return
  }
  if (chatId) {
    steps.value = []
    sources.value = []
    snippets.value = []
    sourceTitles.value = {}
    answerText.value = ''
    currentChatId.value = chatId
    runError.value = ''
    await loadHistory(chatId)
    await loadRunData(chatId)
  }
}

async function loadRunData(chatId: string) {
  const metaResp = await apiFetch(`/chats/${chatId}`)
  if (!metaResp.ok) return
  const meta = (await metaResp.json()) as ChatMeta
  isBookmarked.value = meta.bookmarked || false
  const lastRunId = meta.last_run_id
  if (!lastRunId) return
  runId.value = lastRunId

  const [stepsResp, sourcesResp, snippetsResp] = await Promise.all([
    apiFetch(`/runs/${lastRunId}/steps`),
    apiFetch(`/runs/${lastRunId}/sources`),
    apiFetch(`/runs/${lastRunId}/snippets`)
  ])

  if (stepsResp.ok) {
    const data = (await stepsResp.json()) as { items?: Step[] }
    if (Array.isArray(data.items)) {
      steps.value = data.items
    }
  }
  if (sourcesResp.ok) {
    const data = (await sourcesResp.json()) as { items?: Source[] }
    if (Array.isArray(data.items)) {
      sources.value = data.items
      sourceTitles.value = data.items.reduce<Record<string, string>>((acc, item) => {
        if (item.url && item.title) acc[item.url] = item.title
        return acc
      }, {})
    }
  }
  if (snippetsResp.ok) {
    const data = (await snippetsResp.json()) as { items?: Snippet[] }
    if (Array.isArray(data.items)) {
      snippets.value = [...data.items].sort((a, b) => a.ref - b.ref)
    }
  }
}

async function submitFollowup() {
  const text = followup.value.trim()
  if (!text || isRunning.value) return
  followup.value = ''
  const model = String(route.query.model || selectedModel.value || '').trim()
  const chatId = currentChatId.value || String(route.params.chatId || '').trim()
  await startRun(text, model, chatId || undefined)
}

async function toggleBookmark() {
  const chatId = currentChatId.value || String(route.params.chatId || '').trim()
  if (!chatId) return
  const url = `/bookmarks/${chatId}`
  const resp = await apiFetch(url, { method: isBookmarked.value ? 'DELETE' : 'POST' })
  if (!resp.ok) return
  isBookmarked.value = !isBookmarked.value
}

async function deleteCurrentChat() {
  const chatId = currentChatId.value || String(route.params.chatId || '').trim()
  if (!chatId) return
  if (!confirm('Удалить этот диалог?')) return
  const resp = await apiFetch(`/chats/${chatId}`, { method: 'DELETE' })
  if (!resp.ok) return
  await router.push({ name: 'home' })
}

async function loadHistory(chatId: string) {
  const resp = await apiFetch(`/chats/${chatId}/messages?limit=50`)
  if (!resp.ok) return
  const data = (await resp.json()) as { items?: ChatMessage[] }
  if (!Array.isArray(data.items)) return
  history.value = data.items
    .filter((msg) => msg.role === 'user' || msg.role === 'assistant')
    .map((msg) => ({ id: msg.id, role: msg.role, content: msg.content }))
  const lastAssistant = [...history.value].reverse().find((msg) => msg.role === 'assistant')
  answerText.value = lastAssistant?.content || ''
}

function upsertAssistant(content: string) {
  if (!content) return
  const last = history.value[history.value.length - 1]
  if (last && last.role === 'assistant') {
    last.content = content
    history.value = [...history.value.slice(0, -1), last]
    return
  }
  history.value.push({ id: `assistant-${runId.value}`, role: 'assistant', content })
}

function renderCitations(input: string, sourceList: Source[]) {
  const re = /\[(\d+)\]/g
  let out = ''
  let last = 0
  let match: RegExpExecArray | null
  while ((match = re.exec(input)) !== null) {
    out += md.utils.escapeHtml(input.slice(last, match.index))
    const idx = Number(match[1]) - 1
    const source = sourceList[idx]
    if (source) {
      const url = md.utils.escapeHtml(source.url)
      const title = md.utils.escapeHtml(source.title || source.url)
      out += `<a class="citation" href="${url}" target="_blank" rel="noreferrer" title="${title}">[${match[1]}]</a>`
    } else {
      out += md.utils.escapeHtml(match[0])
    }
    last = match.index + match[0].length
  }
  out += md.utils.escapeHtml(input.slice(last))
  return out
}

const defaultTextRule = md.renderer.rules.text
md.renderer.rules.text = (tokens, idx, options, env, self) => {
  const sources = (env && (env as { sources?: Source[] }).sources) || []
  if (sources.length === 0) {
    return defaultTextRule ? defaultTextRule(tokens, idx, options, env, self) : md.utils.escapeHtml(tokens[idx].content)
  }
  return renderCitations(tokens[idx].content, sources)
}

function renderMarkdown(input: string, sourceList: Source[]) {
  return md.render(input, { sources: sourceList })
}
</script>

<style scoped>
.chat {
  max-width: 980px;
  margin: 0 auto;
}

.progress {
  margin-top: 18px;
  display: flex;
  align-items: center;
  gap: 10px;
}

.error-text {
  margin-top: 6px;
  font-size: 12px;
  color: var(--danger);
}

.answer {
  margin-top: 16px;
}

.answer-title {
  font-size: 12px;
  color: var(--muted);
  margin-bottom: 8px;
}

.answer-card {
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 16px;
  background: var(--card-bg);
  overflow: hidden;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: var(--accent);
}

.text {
  color: var(--muted);
  font-size: 14px;
}

.steps {
  margin-top: 16px;
}

.steps-title {
  font-size: 12px;
  color: var(--muted);
  margin-bottom: 8px;
}

.links {
  margin-top: 16px;
}

.sources-title {
  font-size: 12px;
  color: var(--muted);
  margin-bottom: 8px;
}

.sources-card {
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 14px;
  background: var(--card-bg);
}

.sources-empty {
  color: var(--muted);
  font-size: 13px;
}
</style>
