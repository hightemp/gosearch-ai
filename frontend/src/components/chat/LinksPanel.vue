<template>
  <div class="links">
    <div class="links-title">Ссылки: {{ sources.length }}</div>
    <div class="links-card">
      <div v-if="!sources.length" class="links-empty">Список ссылок появится после поиска.</div>
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
</template>

<script setup lang="ts">
export interface Source {
  url: string
  title?: string
}

defineProps<{
  sources: Source[]
}>()

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
</script>

<style scoped>
.links {
  margin-top: 16px;
}

.links-title {
  font-size: 12px;
  color: var(--muted);
  margin-bottom: 8px;
}

.links-card {
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 14px;
  background: var(--card-bg);
}

.links-empty {
  color: var(--muted);
  font-size: 13px;
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
  background: var(--hover);
}

.link-meta a {
  color: var(--accent);
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
</style>
