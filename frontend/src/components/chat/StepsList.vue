<template>
  <div class="steps-card">
    <div v-if="!steps.length && !isRunning" class="steps-empty">Пока нет шагов...</div>
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
        <LoadingDots label="Выполняется" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import LoadingDots from '../common/LoadingDots.vue'

export interface Step {
  type: string
  title: string
  payload: any
  created_at?: string
}

interface StepGroup extends Step {
  label: string
  detail?: string
  detailUrl?: string
  detailDomain?: string
  items?: { url: string; title?: string }[]
}

const props = defineProps<{
  steps: Step[]
  isRunning?: boolean
}>()

const stepGroups = computed<StepGroup[]>(() => {
  return props.steps.map((st) => {
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
</script>

<style scoped>
.steps-card {
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 14px;
  background: var(--card-bg);
  display: grid;
  gap: 10px;
}

.steps-empty {
  color: var(--muted);
  font-size: 13px;
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
  color: var(--muted);
}

.step-title {
  font-size: 13px;
  color: var(--fg);
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
  background: var(--hover);
}

.step-link a {
  color: var(--accent);
  text-decoration: none;
  font-weight: 600;
}

.step-link a:hover {
  text-decoration: underline;
}

.step-title a {
  color: var(--accent);
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
</style>
