<template>
  <div class="chat">
    <div class="tabs">
      <button class="tab" :class="{ 'tab--active': activeTab === 'answer' }" @click="activeTab = 'answer'">
        <MessageSquare class="tab-icon" />
        Ответ
      </button>
      <button class="tab" :class="{ 'tab--active': activeTab === 'steps' }" @click="activeTab = 'steps'">
        <ListChecks class="tab-icon" />
        Шаги
      </button>
      <button class="tab" :class="{ 'tab--active': activeTab === 'links' }" @click="activeTab = 'links'">
        <Link class="tab-icon" />
        Ссылки
      </button>
      <button class="tab" :class="{ 'tab--active': activeTab === 'images' }" @click="activeTab = 'images'">
        <Image class="tab-icon" />
        Изображения
      </button>
    </div>

    <div class="progress">
      <div class="dot" />
      <div class="text">{{ statusText }}</div>
    </div>
    <div v-if="runError" class="error-text">{{ runError }}</div>

    <div v-if="activeTab === 'answer'" class="answer">
      <div class="answer-title">Ответ</div>
      <div class="answer-card" ref="answerRef">
        <div v-if="messages.length" class="message-list">
          <div v-for="msg in messages" :key="msg.id" class="message" :class="`message--${msg.role}`">
            <div class="message-header">
              <div class="message-role">
                {{ msg.roleLabel }}
                <span v-if="msg.modelLabel" class="model-badge">{{ msg.modelLabel }}</span>
              </div>
              <button v-if="msg.role === 'assistant'" class="copy-btn" @click="copyText(msg.content)" aria-label="Copy">
                <Copy class="copy-icon" />
              </button>
            </div>
            <div class="message-body" v-if="msg.role === 'assistant'" v-html="msg.html" />
            <div class="message-body" v-else>{{ msg.content }}</div>
          </div>
          <div v-if="isRunning" class="message message--assistant">
            <div class="message-role">Ассистент</div>
            <div class="message-body">
              <span class="loading-dots" aria-label="Идет генерация">
                <span class="dot" />
                <span class="dot" />
                <span class="dot" />
              </span>
            </div>
          </div>
        </div>
        <div v-else class="steps-card">
          <div v-if="!steps.length" class="sources-empty">Пока нет шагов…</div>
          <div v-for="(st, idx) in stepGroups" :key="idx" class="step">
            <div class="step-type">{{ st.label }}</div>
            <div class="step-title">
              <div v-if="st.detail">{{ st.detail }}</div>
              <div v-else-if="st.detailUrl" class="step-url">
                <div class="step-url-main">
                  <img class="step-favicon" :src="faviconUrl(st.detailUrl)" alt="" />
                  <a :href="st.detailUrl" target="_blank" rel="noreferrer">{{ st.detailDomain || st.detailUrl }}</a>
                </div>
                <span class="step-url-raw">{{ st.detailUrl }}</span>
              </div>
              <div v-else-if="st.type === 'agent.fetch'">
                <ul class="step-links">
                  <li v-for="item in st.items" :key="item.url" class="step-link">
                    <img class="step-favicon" :src="faviconUrl(item.url)" alt="" />
                    <a :href="item.url" target="_blank" rel="noreferrer">{{ item.title || item.url }}</a>
                    <span class="step-domain">{{ getDomain(item.url) }}</span>
                  </li>
                </ul>
              </div>
              <div v-else>{{ st.title }}</div>
            </div>
          </div>
          <div v-if="isRunning" class="step step--pending">
            <div class="step-type">В процессе</div>
            <div class="step-title">
              <span class="loading-dots" aria-label="Выполняется">
                <span class="dot" />
                <span class="dot" />
                <span class="dot" />
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-if="activeTab === 'answer'" class="sources">
      <div class="sources-title">Просмотр источников: {{ sources.length }}</div>
      <div v-if="showSourcesNote" class="sources-note">Источники показаны для последнего запуска в этом чате.</div>
      <div class="sources-card">
        <div v-if="!sources.length" class="sources-empty">Источники появятся после запуска run (SSE).</div>
        <ul v-else class="sources-list">
          <li v-for="s in sourceDetails" :key="s.url" class="source-item">
            <div class="source-header">
              <a :href="s.url" target="_blank" rel="noreferrer">{{ s.title || s.url }}</a>
              <span class="source-domain">{{ getDomain(s.url) }}</span>
              <span v-if="isPDF(s.url)" class="source-badge">PDF</span>
            </div>
            <div v-if="s.snippets.length" class="source-snippets">
              <div class="source-snippets-title">Цитаты</div>
              <ul class="snippet-list">
                <li v-for="snip in s.snippets" :key="`${s.url}-${snip.ref}`" class="snippet-item">
                  <span class="snippet-ref">[{{ snip.ref }}]</span>
                  <span class="snippet-text">{{ snip.quote }}</span>
                </li>
              </ul>
            </div>
          </li>
        </ul>
      </div>
    </div>

    <div v-if="activeTab === 'steps'" class="steps">
      <div class="steps-title">Шаги</div>
      <div class="steps-card">
        <div v-if="!steps.length" class="sources-empty">Пока нет шагов…</div>
        <div v-for="(st, idx) in stepGroups" :key="idx" class="step">
          <div class="step-type">{{ st.label }}</div>
          <div class="step-title">
            <div v-if="st.detail">{{ st.detail }}</div>
            <div v-else-if="st.detailUrl" class="step-url">
              <div class="step-url-main">
                <img class="step-favicon" :src="faviconUrl(st.detailUrl)" alt="" />
                <a :href="st.detailUrl" target="_blank" rel="noreferrer">{{ st.detailDomain || st.detailUrl }}</a>
              </div>
              <span class="step-url-raw">{{ st.detailUrl }}</span>
            </div>
            <div v-else-if="st.type === 'agent.fetch'">
              <ul class="step-links">
                <li v-for="item in st.items" :key="item.url" class="step-link">
                  <img class="step-favicon" :src="faviconUrl(item.url)" alt="" />
                  <a :href="item.url" target="_blank" rel="noreferrer">{{ item.title || item.url }}</a>
                  <span class="step-domain">{{ getDomain(item.url) }}</span>
                </li>
              </ul>
            </div>
            <div v-else>{{ st.title }}</div>
          </div>
        </div>
        <div v-if="isRunning" class="step step--pending">
          <div class="step-type">В процессе</div>
          <div class="step-title">
            <span class="loading-dots" aria-label="Выполняется">
              <span class="dot" />
              <span class="dot" />
              <span class="dot" />
            </span>
          </div>
        </div>
      </div>
    </div>

    <div v-if="activeTab === 'links'" class="links">
      <div class="sources-title">Ссылки: {{ sources.length }}</div>
      <div class="sources-card">
        <div v-if="!sources.length" class="sources-empty">Список ссылок появится после поиска.</div>
        <ul v-else class="links-list">
          <li v-for="(s, idx) in sources" :key="s.url" class="link-item">
            <img class="link-favicon" :src="faviconUrl(s.url)" alt="" />
            <div class="link-meta">
              <a :href="s.url" target="_blank" rel="noreferrer">{{ s.title || `Источник ${idx + 1}` }}</a>
              <div class="link-domain">
                {{ getDomain(s.url) }}
                <span v-if="isPDF(s.url)" class="link-badge">PDF</span>
              </div>
            </div>
          </li>
        </ul>
      </div>
    </div>

    <div v-if="activeTab === 'images'" class="links">
      <div class="sources-title">Изображения</div>
      <div class="sources-card">
        <div class="sources-empty">Поддержка изображений появится позже.</div>
      </div>
    </div>

    <div class="composer">
      <input
        v-model="followup"
        class="composer-input"
        placeholder="Добавить детали или пояснения…"
        @keydown.enter.exact.prevent="submitFollowup"
      />
      <div class="model-picker">
        <button class="model-trigger" :disabled="isLoadingModels" @click="toggleModelMenu" aria-label="Выбрать модель">
          <Cpu class="icon icon--small" />
        </button>
        <div v-if="showModelMenu" class="model-menu">
          <button
            v-for="m in models"
            :key="m"
            class="model-option"
            :class="{ 'model-option--active': m === selectedModel }"
            @click="selectModel(m)"
          >
            {{ m }}
          </button>
        </div>
      </div>
      <button class="composer-send" :disabled="!followup.trim()" @click="submitFollowup">
        <ArrowRight class="composer-icon" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ArrowRight, Copy, Cpu, Image, Link, ListChecks, MessageSquare } from 'lucide-vue-next'
import hljs from 'highlight.js'
import MarkdownIt from 'markdown-it'
import mk from 'markdown-it-katex'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { apiFetch, apiUrl } from '../api'
import { useModelStore } from '../modelStore'

type Step = { type: string; title: string; payload: any; created_at?: string }
type Source = { url: string; title?: string }
type Snippet = { url: string; quote: string; ref: number }
type AnswerDelta = { delta?: string }
type AnswerFinal = { answer?: string; model?: string }
type ChatMessage = { id: string; role: 'user' | 'assistant'; content: string }
type ChatMeta = { last_run_id?: string }
type RunStep = { type: string; title: string; payload: any; created_at?: string }
type RunSource = { url: string; title?: string }
type RunSnippet = { url: string; quote: string; ref: number }

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
const activeTab = ref<'answer' | 'steps' | 'links' | 'images'>('answer')
const sourceTitles = ref<Record<string, string>>({})
const followup = ref('')
const eventSource = ref<EventSource | null>(null)
const currentChatId = ref('')
const lastQuery = ref('')
const history = ref<ChatMessage[]>([])
const answerRef = ref<HTMLElement | null>(null)
const showModelMenu = ref(false)
const { models, selectedModel, isLoadingModels, loadModels, setModel } = useModelStore()

const md = new MarkdownIt({
  html: false,
  linkify: true
}).use(mk)

md.renderer.rules.fence = (tokens, idx, options, env, self) => {
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
  if (isRunning.value) return 'Активен…'
  if (runId.value) return 'Завершено'
  return 'Ожидание…'
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

const sourceDetails = computed(() =>
  sources.value.map((source) => ({
    ...source,
    title: source.title || sourceTitles.value[source.url],
    snippets: snippets.value.filter((snip) => snip.url === source.url)
  }))
)

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

const stepGroups = computed(() => {
  return steps.value.map((st) => {
    if (st.type === 'agent.fetch') {
      const items = Array.isArray(st.payload?.items) ? st.payload.items : []
      return { ...st, label: 'Чтение источников', items }
    }
    if (st.type === 'agent.message') {
      return { ...st, label: 'Сообщение агента', detail: st.payload?.content || '' }
    }
    if (st.type === 'agent.reasoning') {
      return { ...st, label: 'Рассуждение агента', detail: st.payload?.content || '' }
    }
    if (st.type === 'search.query') {
      return { ...st, label: 'Поиск', detail: st.payload?.query || '' }
    }
    if (st.type === 'page.fetch.started') {
      const url = st.payload?.url || ''
      return { ...st, label: 'Запрос страницы', detailUrl: url, detailDomain: url ? getDomain(url) : '' }
    }
    if (st.type === 'run.finished') {
      return { ...st, label: 'Pipeline завершен' }
    }
    return { ...st, label: st.title || st.type }
  })
})

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
  const lastRunId = meta.last_run_id
  if (!lastRunId) return
  runId.value = lastRunId

  const [stepsResp, sourcesResp, snippetsResp] = await Promise.all([
    apiFetch(`/runs/${lastRunId}/steps`),
    apiFetch(`/runs/${lastRunId}/sources`),
    apiFetch(`/runs/${lastRunId}/snippets`)
  ])

  if (stepsResp.ok) {
    const data = (await stepsResp.json()) as { items?: RunStep[] }
    if (Array.isArray(data.items)) {
      steps.value = data.items as Step[]
    }
  }
  if (sourcesResp.ok) {
    const data = (await sourcesResp.json()) as { items?: RunSource[] }
    if (Array.isArray(data.items)) {
      sources.value = data.items
      sourceTitles.value = data.items.reduce<Record<string, string>>((acc, item) => {
        if (item.url && item.title) acc[item.url] = item.title
        return acc
      }, {})
    }
  }
  if (snippetsResp.ok) {
    const data = (await snippetsResp.json()) as { items?: RunSnippet[] }
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

function toggleModelMenu() {
  if (isLoadingModels.value) return
  showModelMenu.value = !showModelMenu.value
}

function selectModel(model: string) {
  setModel(model)
  showModelMenu.value = false
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

function getDomain(url: string) {
  try {
    return new URL(url).hostname
  } catch {
    return url
  }
}

function faviconUrl(url: string) {
  const domain = getDomain(url)
  return `https://www.google.com/s2/favicons?domain=${encodeURIComponent(domain)}&sz=64`
}

function isPDF(url: string) {
  return /\.pdf($|[?#])/i.test(url)
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
</script>

<style scoped>
.chat {
  max-width: 980px;
  margin: 0 auto;
}
.tabs {
  display: flex;
  gap: 18px;
  border-bottom: 1px solid var(--border);
  padding-bottom: 10px;
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
.progress {
  margin-top: 18px;
  display: flex;
  align-items: center;
  gap: 10px;
}
.error-text {
  margin-top: 6px;
  font-size: 12px;
  color: #b91c1c;
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
  background: #fff;
}
.message-list {
  display: grid;
  gap: 16px;
}
.message {
  display: grid;
  gap: 6px;
}
.message-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.copy-btn {
  border: 1px solid var(--border);
  background: #fff;
  color: #111827;
  font-size: 11px;
  padding: 4px 8px;
  border-radius: 999px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.copy-btn:hover {
  background: #f9fafb;
}
.copy-icon {
  width: 14px;
  height: 14px;
}
.message-role {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--muted);
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
.model-badge {
  border: 1px solid #e5e7eb;
  background: #fff;
  color: #111827;
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 999px;
  text-transform: none;
  letter-spacing: 0.02em;
}
.message-body {
  font-size: 14px;
  color: #111827;
  line-height: 1.6;
}
.message--user .message-body {
  white-space: pre-wrap;
}
.message--assistant .message-body :global(p),
.message--assistant .message-body :global(h1),
.message--assistant .message-body :global(h2),
.message--assistant .message-body :global(h3),
.message--assistant .message-body :global(h4),
.message--assistant .message-body :global(h5),
.message--assistant .message-body :global(h6),
.message--assistant .message-body :global(ul),
.message--assistant .message-body :global(blockquote),
.message--assistant .message-body :global(pre) {
  margin: 0 0 12px 0;
}
.message--assistant .message-body :global(ul) {
  margin-left: 20px;
}
.message--assistant .message-body :global(blockquote) {
  padding-left: 12px;
  border-left: 3px solid #e5e7eb;
  color: #374151;
}
.message--assistant .message-body :global(pre) {
  background: #f3f4f6;
  border-radius: 10px;
  padding: 12px;
  overflow-x: auto;
}
.message--assistant .message-body :global(.code-block) {
  position: relative;
  margin: 0 0 12px 0;
}
.message--assistant .message-body :global(.code-copy) {
  position: absolute;
  top: 8px;
  right: 8px;
  border: 1px solid #e5e7eb;
  background: #fff;
  color: #111827;
  font-size: 10px;
  padding: 4px 8px;
  border-radius: 999px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.message--assistant .message-body :global(.code-copy:hover) {
  background: #f9fafb;
}
.message--assistant .message-body :global(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 0 0 12px 0;
  font-size: 13px;
}
.message--assistant .message-body :global(th),
.message--assistant .message-body :global(td) {
  border: 1px solid #e5e7eb;
  padding: 8px 10px;
  text-align: left;
  vertical-align: top;
}
.message--assistant .message-body :global(th) {
  background: #f9fafb;
  font-weight: 600;
}
.message--assistant .message-body :global(code) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 12px;
}
.message--assistant .message-body :global(.citation) {
  color: #0f766e;
  text-decoration: none;
  font-weight: 600;
}
.message--assistant .message-body :global(.citation:hover) {
  text-decoration: underline;
}
.message--assistant {
  border-left: 3px solid #0f766e;
  padding-left: 12px;
}
.message--user {
  border-left: 3px solid #e5e7eb;
  padding-left: 12px;
}
.answer-content :global(p) {
  margin: 0 0 12px 0;
  line-height: 1.6;
}
.answer-content :global(h1),
.answer-content :global(h2),
.answer-content :global(h3),
.answer-content :global(h4),
.answer-content :global(h5),
.answer-content :global(h6) {
  margin: 0 0 12px 0;
  font-weight: 600;
}
.answer-content :global(blockquote) {
  margin: 0 0 12px 0;
  padding-left: 12px;
  border-left: 3px solid #e5e7eb;
  color: #374151;
}
.answer-content :global(ul) {
  margin: 0 0 12px 20px;
  padding: 0;
}
.answer-content :global(pre) {
  background: #f3f4f6;
  border-radius: 10px;
  padding: 12px;
  overflow-x: auto;
  margin: 0 0 12px 0;
}
.answer-content :global(code) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 12px;
}
.answer-content :global(.citation) {
  color: #0f766e;
  text-decoration: none;
  font-weight: 600;
}
.answer-content :global(.citation:hover) {
  text-decoration: underline;
}
.dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: #0f766e;
}
.loading-dots {
  display: inline-flex;
  gap: 6px;
  align-items: center;
}
.loading-dots .dot {
  animation: loading-pulse 1.1s infinite ease-in-out;
}
.loading-dots .dot:nth-child(2) {
  animation-delay: 0.15s;
}
.loading-dots .dot:nth-child(3) {
  animation-delay: 0.3s;
}
@keyframes loading-pulse {
  0%,
  80%,
  100% {
    opacity: 0.3;
    transform: scale(0.9);
  }
  40% {
    opacity: 1;
    transform: scale(1);
  }
}
.text {
  color: var(--muted);
  font-size: 14px;
}
.sources {
  margin-top: 14px;
}
.sources-title {
  font-size: 12px;
  color: var(--muted);
  margin-bottom: 8px;
}
.sources-note {
  font-size: 12px;
  color: #6b7280;
  margin-bottom: 8px;
}
.sources-card {
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 14px;
  background: #fff;
}
.sources-empty {
  color: var(--muted);
  font-size: 13px;
}
.sources-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: grid;
  gap: 8px;
}
.source-header {
  display: flex;
  align-items: center;
  gap: 10px;
}
.source-domain {
  font-size: 12px;
  color: var(--muted);
}
.source-badge {
  border: 1px solid #c7d2fe;
  color: #4338ca;
  background: #eef2ff;
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 999px;
  letter-spacing: 0.02em;
}
.source-snippets {
  margin-top: 8px;
  border-top: 1px dashed var(--border);
  padding-top: 8px;
}
.source-snippets-title {
  font-size: 12px;
  color: var(--muted);
  margin-bottom: 6px;
}
.snippet-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: grid;
  gap: 6px;
}
.snippet-item {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 8px;
  font-size: 13px;
  color: #111827;
}
.snippet-ref {
  font-weight: 600;
  color: #0f766e;
}
.links-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: grid;
  gap: 12px;
}
.link-item {
  display: grid;
  grid-template-columns: 24px 1fr;
  gap: 10px;
  align-items: center;
}
.link-favicon {
  width: 20px;
  height: 20px;
  border-radius: 6px;
  background: #f3f4f6;
}
.link-meta a {
  color: #0f766e;
  text-decoration: none;
  font-weight: 600;
}
.link-meta a:hover {
  text-decoration: underline;
}
.link-domain {
  font-size: 12px;
  color: var(--muted);
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.link-badge {
  border: 1px solid #c7d2fe;
  color: #4338ca;
  background: #eef2ff;
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 999px;
  letter-spacing: 0.02em;
}
.source-item a {
  color: #0f766e;
  text-decoration: none;
}
.source-item a:hover {
  text-decoration: underline;
}
.steps {
  margin-top: 16px;
}
.steps-title {
  font-size: 12px;
  color: var(--muted);
  margin-bottom: 8px;
}
.steps-card {
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 14px;
  background: #fff;
  display: grid;
  gap: 10px;
}
.step {
  display: grid;
  grid-template-columns: 180px 1fr;
  gap: 12px;
  align-items: baseline;
}
.step-type {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 12px;
  color: #374151;
}
.step-title {
  font-size: 13px;
  color: #111827;
}
.step-links {
  list-style: none;
  padding: 0;
  margin: 0;
  display: grid;
  gap: 6px;
}
.step-link {
  display: grid;
  grid-template-columns: 16px auto 1fr;
  gap: 8px;
  align-items: center;
  font-size: 12px;
}
.step-favicon {
  width: 16px;
  height: 16px;
  border-radius: 4px;
  background: #f3f4f6;
}
.step-link a {
  color: #0f766e;
  text-decoration: none;
  font-weight: 600;
}
.step-link a:hover {
  text-decoration: underline;
}
.step-title a {
  color: #0f766e;
  text-decoration: none;
  word-break: break-all;
}
.step-title a:hover {
  text-decoration: underline;
}
.step-url {
  display: grid;
  gap: 4px;
}
.step-url-main {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.step-url-raw {
  font-size: 11px;
  color: var(--muted);
  word-break: break-all;
}
.step-domain {
  color: var(--muted);
  font-size: 11px;
}
.composer {
  position: sticky;
  bottom: 18px;
  margin-top: 26px;
  display: grid;
  grid-template-columns: 1fr auto auto;
  gap: 10px;
  background: #fff;
  border: 1px solid var(--border);
  border-radius: 16px;
  padding: 12px;
}
.composer-input {
  border: 0;
  outline: none;
  font-size: 14px;
}
.composer-send {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  border: 0;
  background: #0f766e;
  color: #fff;
  cursor: pointer;
  display: grid;
  place-items: center;
}
.composer-icon {
  width: 18px;
  height: 18px;
}
.composer .model-picker {
  position: relative;
}
.composer .model-trigger {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  border: 1px solid var(--border);
  background: #fff;
  display: grid;
  place-items: center;
  cursor: pointer;
}
.composer .model-trigger:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}
.composer .model-menu {
  position: absolute;
  right: 0;
  bottom: 52px;
  background: #fff;
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 8px;
  display: grid;
  gap: 6px;
  min-width: 220px;
  z-index: 20;
  box-shadow: 0 10px 30px rgba(15, 23, 42, 0.08);
}
.composer .model-option {
  border: 0;
  background: transparent;
  text-align: left;
  padding: 8px 10px;
  border-radius: 10px;
  cursor: pointer;
  font-size: 12px;
  color: #111827;
}
.composer .model-option:hover {
  background: var(--hover);
}
.composer .model-option--active {
  background: rgba(15, 118, 110, 0.12);
  color: #0f766e;
}
</style>
