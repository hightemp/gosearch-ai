<template>
  <Teleport to="body">
    <Transition name="slide">
      <div v-if="isOpen" class="sources-sidebar-overlay" @click.self="$emit('close')">
        <aside class="sources-sidebar">
          <header class="sidebar-header">
            <div class="header-content">
              <span class="header-title">{{ sources.length }} источников</span>
              <span class="header-subtitle">{{ queryContext }}</span>
            </div>
            <button class="close-btn" @click="$emit('close')" aria-label="Закрыть">
              <X :size="20" />
            </button>
          </header>
          
          <div class="sources-list">
            <article v-for="(source, index) in sources" :key="source.url" class="source-card">
              <div class="source-header">
                <span class="source-number">[{{ index + 1 }}]</span>
                <img :src="getFavicon(source.url)" class="favicon" alt="" />
                <div class="source-meta">
                  <a :href="source.url" target="_blank" rel="noreferrer" class="source-title">
                    {{ source.title || getDomain(source.url) }}
                  </a>
                  <span class="source-domain">{{ getDomain(source.url) }}</span>
                </div>
              </div>
              <p v-if="source.markdownContent" class="source-preview">
                {{ getPreview(source.markdownContent) }}
              </p>
            </article>

            <div v-if="sources.length === 0" class="sources-empty">
              Источники не найдены
            </div>
          </div>
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { X } from 'lucide-vue-next'

export interface Source {
  url: string
  title?: string
  domain?: string
  faviconUrl?: string
  markdownContent?: string
}

defineProps<{
  isOpen: boolean
  sources: Source[]
  queryContext: string
}>()

defineEmits<{
  close: []
}>()

function getFavicon(url: string): string {
  try {
    const domain = new URL(url).hostname
    return `https://www.google.com/s2/favicons?domain=${domain}&sz=32`
  } catch {
    return ''
  }
}

function getDomain(url: string): string {
  try {
    return new URL(url).hostname
  } catch {
    return url
  }
}

function getPreview(content: string, maxLength = 200): string {
  if (!content) return ''
  // Remove markdown formatting for preview
  const clean = content
    .replace(/^#+\s+/gm, '') // headers
    .replace(/\*\*([^*]+)\*\*/g, '$1') // bold
    .replace(/\*([^*]+)\*/g, '$1') // italic
    .replace(/`([^`]+)`/g, '$1') // inline code
    .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1') // links
    .replace(/!\[([^\]]*)\]\([^)]+\)/g, '') // images
    .replace(/^\s*[-*]\s+/gm, '') // list items
    .replace(/^\s*\d+\.\s+/gm, '') // numbered list
    .replace(/\n{2,}/g, ' ') // multiple newlines
    .replace(/\n/g, ' ') // single newlines
    .trim()
  
  return clean.length > maxLength ? clean.slice(0, maxLength) + '...' : clean
}
</script>

<style scoped>
.sources-sidebar-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.3);
  z-index: 1000;
}

.sources-sidebar {
  position: fixed;
  top: 0;
  right: 0;
  width: 420px;
  max-width: 90vw;
  height: 100vh;
  background: var(--bg);
  border-left: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: -4px 0 20px rgba(0, 0, 0, 0.1);
}

.sidebar-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 20px;
  border-bottom: 1px solid var(--border);
  background: var(--card-bg);
}

.header-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.header-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--fg);
}

.header-subtitle {
  font-size: 13px;
  color: var(--muted);
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.close-btn {
  background: none;
  border: none;
  cursor: pointer;
  color: var(--muted);
  padding: 4px;
  border-radius: 4px;
  transition: all 0.15s ease;
}

.close-btn:hover {
  color: var(--fg);
  background: var(--hover);
}

.sources-list {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.source-card {
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 14px;
  background: var(--card-bg);
  transition: border-color 0.15s ease;
}

.source-card:hover {
  border-color: var(--accent);
}

.source-header {
  display: flex;
  gap: 10px;
  align-items: flex-start;
}

.source-number {
  font-size: 13px;
  font-weight: 600;
  color: var(--accent);
  min-width: 24px;
  flex-shrink: 0;
}

.favicon {
  width: 24px;
  height: 24px;
  border-radius: 4px;
  flex-shrink: 0;
  background: var(--hover);
}

.source-meta {
  flex: 1;
  min-width: 0;
}

.source-title {
  color: var(--accent);
  text-decoration: none;
  font-weight: 500;
  font-size: 14px;
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.source-title:hover {
  text-decoration: underline;
}

.source-domain {
  font-size: 12px;
  color: var(--muted);
  display: block;
  margin-top: 2px;
}

.source-preview {
  margin-top: 10px;
  font-size: 13px;
  color: var(--muted);
  line-height: 1.5;
  border-top: 1px dashed var(--border);
  padding-top: 10px;
}

.sources-empty {
  text-align: center;
  padding: 40px 20px;
  color: var(--muted);
  font-size: 14px;
}

/* Slide animation */
.slide-enter-active,
.slide-leave-active {
  transition: all 0.3s ease;
}

.slide-enter-active .sources-sidebar,
.slide-leave-active .sources-sidebar {
  transition: transform 0.3s ease;
}

.slide-enter-from .sources-sidebar,
.slide-leave-to .sources-sidebar {
  transform: translateX(100%);
}

.slide-enter-from,
.slide-leave-to {
  opacity: 0;
}
</style>
