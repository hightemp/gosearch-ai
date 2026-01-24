<template>
  <div class="message" :class="`message--${message.role}`">
    <div class="message-header">
      <div class="message-role">
        {{ roleLabel }}
        <span v-if="message.modelLabel" class="model-badge">{{ message.modelLabel }}</span>
      </div>
    </div>
    <div class="message-body" v-if="message.role === 'assistant'" v-html="message.html" />
    <div class="message-body" v-else>{{ message.content }}</div>
    
    <!-- Action buttons for assistant messages -->
    <div v-if="message.role === 'assistant'" class="message-actions">
      <div class="action-buttons">
        <button class="action-btn" @click="copyUrl" title="Copy URL">
          <Link2 :size="16" />
        </button>
        <button class="action-btn" @click="downloadMarkdown" title="Download as Markdown">
          <Download :size="16" />
        </button>
        <button class="action-btn" @click="copyText(message.content)" title="Copy answer">
          <Copy :size="16" />
        </button>
        <button class="action-btn" @click="regenerate" title="Regenerate">
          <RefreshCw :size="16" />
        </button>
      </div>
      
      <!-- Sources button -->
      <button 
        v-if="message.sourcesCount && message.sourcesCount > 0" 
        class="sources-btn"
        @click="emit('show-sources', message.runId)"
      >
        <FileText :size="14" />
        {{ message.sourcesCount }} sources
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Copy, FileText, Link2, Download, RefreshCw } from 'lucide-vue-next'
import { computed } from 'vue'

export interface MessageData {
  id: string
  role: 'user' | 'assistant'
  content: string
  html?: string
  modelLabel?: string
  runId?: string
  sourcesCount?: number
}

const props = defineProps<{
  message: MessageData
}>()

const emit = defineEmits<{
  'show-sources': [runId: string]
  'regenerate': [runId: string]
}>()

const roleLabel = computed(() => {
  return props.message.role === 'assistant' ? 'Assistant' : 'You'
})

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

function copyUrl() {
  const url = window.location.href
  if (navigator.clipboard?.writeText) {
    void navigator.clipboard.writeText(url)
    return
  }
  const textarea = document.createElement('textarea')
  textarea.value = url
  textarea.style.position = 'fixed'
  textarea.style.left = '-9999px'
  document.body.appendChild(textarea)
  textarea.select()
  document.execCommand('copy')
  document.body.removeChild(textarea)
}

function downloadMarkdown() {
  const content = props.message.content
  const blob = new Blob([content], { type: 'text/markdown' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `answer-${Date.now()}.md`
  a.click()
  URL.revokeObjectURL(url)
}

function regenerate() {
  if (props.message.runId) {
    emit('regenerate', props.message.runId)
  }
}
</script>

<style scoped>
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
  border: 1px solid var(--border);
  background: var(--card-bg);
  color: var(--fg);
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 999px;
  text-transform: none;
  letter-spacing: 0.02em;
}

.message-body {
  font-size: 14px;
  color: var(--fg);
  line-height: 1.6;
  word-wrap: break-word;
  overflow-wrap: break-word;
}

.message--user .message-body {
  white-space: pre-wrap;
  word-break: break-word;
}

.message--assistant .message-body :deep(p),
.message--assistant .message-body :deep(h1),
.message--assistant .message-body :deep(h2),
.message--assistant .message-body :deep(h3),
.message--assistant .message-body :deep(h4),
.message--assistant .message-body :deep(h5),
.message--assistant .message-body :deep(h6),
.message--assistant .message-body :deep(ul),
.message--assistant .message-body :deep(blockquote),
.message--assistant .message-body :deep(pre) {
  margin: 0 0 12px 0;
}

.message--assistant .message-body :deep(ul) {
  margin-left: 20px;
}

.message--assistant .message-body :deep(blockquote) {
  padding-left: 12px;
  border-left: 3px solid var(--border);
  color: var(--muted);
}

.message--assistant .message-body :deep(pre) {
  background: var(--hover);
  border-radius: 10px;
  padding: 12px;
  overflow-x: auto;
}

.message--assistant .message-body :deep(.code-block) {
  position: relative;
  margin: 0 0 12px 0;
}

.message--assistant .message-body :deep(.code-copy) {
  position: absolute;
  top: 8px;
  right: 8px;
  border: 1px solid var(--border);
  background: var(--card-bg);
  color: var(--fg);
  font-size: 10px;
  padding: 4px 8px;
  border-radius: 999px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.message--assistant .message-body :deep(.code-copy:hover) {
  background: var(--hover);
}

.message--assistant .message-body :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 0 0 12px 0;
  font-size: 13px;
}

.message--assistant .message-body :deep(th),
.message--assistant .message-body :deep(td) {
  border: 1px solid var(--border);
  padding: 8px 10px;
  text-align: left;
  vertical-align: top;
}

.message--assistant .message-body :deep(th) {
  background: var(--hover);
  font-weight: 600;
}

.message--assistant .message-body :deep(code) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 12px;
}

.message--assistant .message-body :deep(.citation) {
  color: var(--accent);
  text-decoration: none;
  font-weight: 600;
}

.message--assistant .message-body :deep(.citation:hover) {
  text-decoration: underline;
}

.message--assistant {
  border-left: 3px solid var(--accent);
  padding-left: 12px;
}

.message--user {
  border-left: 3px solid var(--border);
  padding-left: 12px;
}

/* Message actions */
.message-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 12px;
}

.action-buttons {
  display: flex;
  align-items: center;
  gap: 4px;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 8px;
  color: var(--muted);
  cursor: pointer;
  transition: all 0.15s ease;
}

.action-btn:hover {
  background: var(--hover);
  color: var(--fg);
  border-color: var(--accent);
}

/* Sources button */
.sources-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  font-size: 12px;
  color: var(--accent);
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 999px;
  cursor: pointer;
  transition: all 0.15s ease;
  width: fit-content;
}

.sources-btn:hover {
  background: var(--hover);
  border-color: var(--accent);
}
</style>
