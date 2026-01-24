<template>
  <div class="home">
    <div class="logo">
      <span class="logo-text">gosearch.ai</span>
    </div>

    <div class="search-card">
      <div class="search-row">
        <div class="search-icon">
          <Search class="icon" />
        </div>
        <textarea
          ref="searchTextarea"
          v-model="q"
          class="search-input"
          placeholder="Ask anything..."
          rows="1"
          @keydown.enter.exact.prevent="submit"
          @input="autoResizeSearch"
        />

        <ModelPicker
          :models="models"
          :selected-model="selectedModel"
          :is-loading="isLoadingModels"
          position="bottom"
          @select="selectModel"
        />

        <button class="send" @click="submit">
          <ArrowRight class="icon icon--inverse" />
        </button>
      </div>
      <div class="hint">Enter - send, Shift+Enter - new line</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ArrowRight, Search } from 'lucide-vue-next'
import { storeToRefs } from 'pinia'
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import ModelPicker from '../components/ModelPicker.vue'
import { useModelStore } from '../stores/modelStore'

const router = useRouter()
const q = ref('')
const searchTextarea = ref<HTMLTextAreaElement | null>(null)

const modelStore = useModelStore()
const { models, selectedModel, isLoadingModels } = storeToRefs(modelStore)
const { loadModels, setModel } = modelStore

function selectModel(model: string) {
  setModel(model)
}

function autoResizeSearch() {
  const el = searchTextarea.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 200) + 'px'
}

async function submit() {
  const text = q.value.trim()
  if (!text) return
  const model = selectedModel.value || models.value[0] || 'openai/gpt-4.1-mini'

  const tmpId = crypto.randomUUID()
  await router.push({ name: 'chat', params: { chatId: tmpId }, query: { q: text, model } })
}

onMounted(() => {
  void loadModels()
})
</script>

<style scoped>
.home {
  display: grid;
  place-items: center;
  min-height: calc(100vh - 48px);
}

.logo {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin-bottom: 18px;
}

.logo-text {
  font-size: 42px;
  font-weight: 500;
  color: var(--fg);
  letter-spacing: -0.02em;
}

.search-card {
  width: min(860px, 92%);
  background: var(--card-bg);
  border: 1px solid var(--border);
  border-radius: 16px;
  padding: 14px;
}

.search-row {
  display: grid;
  grid-template-columns: auto 1fr auto auto;
  gap: 10px;
  align-items: center;
}

.search-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  border: 1px solid var(--border);
  display: grid;
  place-items: center;
  color: var(--accent);
  background: var(--card-bg);
}

.search-input {
  font-size: 16px;
  padding: 14px 14px;
  border-radius: 12px;
  border: 1px solid var(--border);
  outline: none;
  resize: none;
  min-height: 52px;
  max-height: 200px;
  overflow-y: auto;
  line-height: 1.5;
  font-family: inherit;
  background: var(--input-bg);
  color: var(--fg);
}

.search-input::placeholder {
  color: var(--muted);
}

.search-input:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-light);
}

.send {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  border: 0;
  background: var(--accent);
  color: #fff;
  cursor: pointer;
  display: grid;
  place-items: center;
}

.send:hover {
  filter: brightness(0.95);
}

.icon {
  width: 18px;
  height: 18px;
}

.icon--inverse {
  color: #fff;
}

.hint {
  margin-top: 10px;
  font-size: 12px;
  color: var(--muted);
}
</style>
