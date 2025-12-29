<template>
  <div class="home">
    <div class="logo">
      <span class="logo-text">perplexity</span>
      <span class="logo-badge">pro</span>
    </div>

    <div class="search-card">
      <div class="search-row">
        <div class="search-icon">
          <Search class="icon" />
        </div>
        <input
          v-model="q"
          class="search-input"
          placeholder="Спросите что угодно..."
          @keydown.enter.exact.prevent="submit"
        />

        <select v-model="model" class="model" :disabled="isLoadingModels">
          <option v-for="m in models" :key="m" :value="m">{{ m }}</option>
        </select>

        <button class="send" @click="submit">
          <ArrowRight class="icon icon--inverse" />
        </button>
      </div>
      <div class="hint">Enter — отправить (Shift+Enter добавим позже)</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ArrowRight, Search } from 'lucide-vue-next'
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { apiFetch } from '../api'

const router = useRouter()
const q = ref('')

const models = ref<string[]>([])
const model = ref('')
const isLoadingModels = ref(true)

async function loadModels() {
  try {
    const resp = await apiFetch('/models')
    if (!resp.ok) return
    const data = (await resp.json()) as { models?: string[] }
    if (Array.isArray(data.models)) {
      models.value = data.models
    }
  } catch {
    // fallback to default below
  } finally {
    if (!models.value.length) {
      models.value = ['openai/gpt-4.1-mini']
    }
    if (!model.value) {
      model.value = models.value[0]
    }
    isLoadingModels.value = false
  }
}

async function submit() {
  const text = q.value.trim()
  if (!text) return
  if (!model.value) {
    model.value = models.value[0] || 'openai/gpt-4.1-mini'
  }

  const tmpId = crypto.randomUUID()
  await router.push({ name: 'chat', params: { chatId: tmpId }, query: { q: text, model: model.value } })
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
  font-size: 44px;
  font-weight: 400;
  color: #111827;
}
.logo-badge {
  font-size: 14px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 999px;
  border: 1px solid var(--border);
  color: #0f766e;
}
.search-card {
  width: min(860px, 92%);
  background: #fff;
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
  color: #0f766e;
}
.search-input {
  font-size: 16px;
  padding: 14px 14px;
  border-radius: 12px;
  border: 1px solid var(--border);
  outline: none;
}
.search-input:focus {
  border-color: #0f766e;
  box-shadow: 0 0 0 3px rgba(15, 118, 110, 0.15);
}
.model {
  font-size: 12px;
  padding: 12px 10px;
  border-radius: 12px;
  border: 1px solid var(--border);
  background: #fff;
}
.send {
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
