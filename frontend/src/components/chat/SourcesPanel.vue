<template>
  <div class="sources">
    <div class="sources-title">Просмотр источников: {{ sources.length }}</div>
    <div v-if="showNote" class="sources-note">Источники показаны для последнего запуска в этом чате.</div>
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
</template>

<script setup lang="ts">
import { computed } from 'vue'

export interface Source {
  url: string
  title?: string
}

export interface Snippet {
  url: string
  quote: string
  ref: number
}

const props = defineProps<{
  sources: Source[]
  snippets: Snippet[]
  showNote?: boolean
}>()

const sourceDetails = computed(() =>
  props.sources.map((source) => ({
    ...source,
    snippets: props.snippets.filter((snip) => snip.url === source.url)
  }))
)

function getDomain(url: string) {
  try {
    return new URL(url).hostname
  } catch {
    return url
  }
}

function isPDF(url: string) {
  return /\.pdf($|[?#])/i.test(url)
}
</script>

<style scoped>
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
  background: var(--card-bg);
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
  color: var(--fg);
}

.snippet-ref {
  font-weight: 600;
  color: var(--accent);
}

.source-item a {
  color: var(--accent);
  text-decoration: none;
}

.source-item a:hover {
  text-decoration: underline;
}
</style>
