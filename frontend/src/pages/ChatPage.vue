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

    <div v-if="activeTab === 'answer'" class="answer">
      <div class="answer-title">Ответ</div>
      <div class="answer-card">
        <div v-if="messages.length" class="message-list">
          <div v-for="msg in messages" :key="msg.id" class="message" :class="`message--${msg.role}`">
            <div class="message-role">{{ msg.roleLabel }}</div>
            <div class="message-body" v-if="msg.role === 'assistant'" v-html="msg.html" />
            <div class="message-body" v-else>{{ msg.content }}</div>
          </div>
        </div>
        <div v-else class="steps-card">
          <div v-if="!steps.length" class="sources-empty">Пока нет шагов…</div>
          <div v-for="(st, idx) in steps" :key="idx" class="step">
            <div class="step-type">{{ st.type }}</div>
            <div class="step-title">{{ st.title }}</div>
          </div>
        </div>
      </div>
    </div>

    <div v-if="activeTab === 'answer'" class="sources">
      <div class="sources-title">Просмотр источников: {{ sources.length }}</div>
      <div class="sources-card">
        <div v-if="!sources.length" class="sources-empty">Источники появятся после запуска run (SSE).</div>
        <ul v-else class="sources-list">
          <li v-for="s in sourceDetails" :key="s.url" class="source-item">
            <div class="source-header">
              <a :href="s.url" target="_blank" rel="noreferrer">{{ s.title || s.url }}</a>
              <span class="source-domain">{{ getDomain(s.url) }}</span>
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
        <div v-for="(st, idx) in steps" :key="idx" class="step">
          <div class="step-type">{{ st.type }}</div>
          <div class="step-title">{{ st.title }}</div>
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
              <div class="link-domain">{{ getDomain(s.url) }}</div>
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
      <button class="composer-send" :disabled="!followup.trim()" @click="submitFollowup">
        <ArrowRight class="composer-icon" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ArrowRight, Image, Link, ListChecks, MessageSquare } from 'lucide-vue-next'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

type Step = { type: string; title: string; payload: any; created_at?: string }
type Source = { url: string; title?: string }
type Snippet = { url: string; quote: string; ref: number }
type AnswerDelta = { delta?: string }
type AnswerFinal = { answer?: string }
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
const answerText = ref('')
const activeTab = ref<'answer' | 'steps' | 'links' | 'images'>('answer')
const sourceTitles = ref<Record<string, string>>({})
const followup = ref('')
const eventSource = ref<EventSource | null>(null)
const currentChatId = ref('')
const lastQuery = ref('')
const history = ref<ChatMessage[]>([])

const statusText = computed(() => {
  if (isRunning.value) return 'Активен…'
  if (runId.value) return 'Завершено'
  return 'Ожидание…'
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
    const html = msg.role === 'assistant' ? renderMarkdown(msg.content, sources.value) : ''
    return { ...msg, roleLabel, html }
  })
})

async function startRun(queryText: string, model: string, chatId?: string) {
  if (!queryText) return

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
  const resp = await fetch('/api/runs/start', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query: queryText, model, chat_id: chatId })
  })
  if (!resp.ok) {
    isRunning.value = false
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

  const es = new EventSource(`/api/runs/${runId.value}/stream`)
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
    upsertAssistant(answerText.value)
    isRunning.value = false
  })

  es.onerror = () => {
    isRunning.value = false
  }
}

onMounted(() => {
  void hydrateFromRoute()
})

watch(
  () => [route.params.chatId, route.query.q],
  () => {
    void hydrateFromRoute()
  }
)

async function hydrateFromRoute() {
  const q = String(route.query.q || '').trim()
  const model = String(route.query.model || '').trim()
  const chatId = String(route.params.chatId || '').trim()
  if (q) {
    if (q !== lastQuery.value) {
      lastQuery.value = q
      await startRun(q, model, chatId || undefined)
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
    await loadHistory(chatId)
    await loadRunData(chatId)
  }
}

async function loadRunData(chatId: string) {
  const metaResp = await fetch(`/api/chats/${chatId}`)
  if (!metaResp.ok) return
  const meta = (await metaResp.json()) as ChatMeta
  const runId = meta.last_run_id
  if (!runId) return

  const [stepsResp, sourcesResp, snippetsResp] = await Promise.all([
    fetch(`/api/runs/${runId}/steps`),
    fetch(`/api/runs/${runId}/sources`),
    fetch(`/api/runs/${runId}/snippets`)
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
      snippets.value = data.items
    }
  }
}

async function submitFollowup() {
  const text = followup.value.trim()
  if (!text || isRunning.value) return
  followup.value = ''
  const model = String(route.query.model || '').trim()
  const chatId = currentChatId.value || String(route.params.chatId || '').trim()
  await startRun(text, model, chatId || undefined)
}

async function loadHistory(chatId: string) {
  const resp = await fetch(`/api/chats/${chatId}/messages?limit=50`)
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

function escapeHtml(input: string) {
  return input
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

function escapeAttr(input: string) {
  return escapeHtml(input)
}

function renderInline(input: string, sourceList: Source[]) {
  const escaped = escapeHtml(input)
  const formatted = escaped
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
    .replace(/\*([^*]+)\*/g, '<em>$1</em>')
  return formatted.replace(/\[(\d+)\]/g, (match, raw) => {
    const idx = Number(raw) - 1
    const source = sourceList[idx]
    if (!source) return match
    const url = escapeAttr(source.url)
    const title = escapeAttr(source.title || source.url)
    return `<a class="citation" href="${url}" target="_blank" rel="noreferrer" title="${title}">[${raw}]</a>`
  })
}

function renderMarkdown(input: string, sourceList: Source[]) {
  const lines = input.replace(/\r\n/g, '\n').split('\n')
  const html: string[] = []
  let inCode = false
  let code: string[] = []
  let list: string[] = []
  let paragraph: string[] = []

  const flushParagraph = () => {
    if (!paragraph.length) return
    html.push(`<p>${renderInline(paragraph.join(' '), sourceList)}</p>`)
    paragraph = []
  }

  const flushList = () => {
    if (!list.length) return
    html.push(`<ul>${list.map((item) => `<li>${renderInline(item, sourceList)}</li>`).join('')}</ul>`)
    list = []
  }

  for (const line of lines) {
    const trimmed = line.trim()
    if (trimmed.startsWith('```')) {
      if (inCode) {
        html.push(`<pre><code>${escapeHtml(code.join('\n'))}</code></pre>`)
        code = []
        inCode = false
      } else {
        flushParagraph()
        flushList()
        inCode = true
      }
      continue
    }

    if (inCode) {
      code.push(line)
      continue
    }

    if (!trimmed) {
      flushParagraph()
      flushList()
      continue
    }

    const headingMatch = trimmed.match(/^(#{1,6})\s+(.+)$/)
    if (headingMatch) {
      flushParagraph()
      flushList()
      const level = headingMatch[1].length
      html.push(`<h${level}>${renderInline(headingMatch[2], sourceList)}</h${level}>`)
      continue
    }

    if (trimmed.startsWith('>')) {
      flushParagraph()
      flushList()
      const quote = trimmed.replace(/^>\s?/, '')
      html.push(`<blockquote>${renderInline(quote, sourceList)}</blockquote>`)
      continue
    }

    const listMatch = line.match(/^\s*([-*]|\d+\.)\s+(.+)$/)
    if (listMatch) {
      flushParagraph()
      list.push(listMatch[2])
      continue
    }

    paragraph.push(trimmed)
  }

  if (inCode) {
    html.push(`<pre><code>${escapeHtml(code.join('\n'))}</code></pre>`)
  }
  flushParagraph()
  flushList()

  return html.join('\n')
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
.message-role {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--muted);
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
.composer {
  position: sticky;
  bottom: 18px;
  margin-top: 26px;
  display: grid;
  grid-template-columns: 1fr auto;
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
</style>
