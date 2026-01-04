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
        <input
          v-model="q"
          class="search-input"
          placeholder="Спросите что угодно..."
          @keydown.enter.exact.prevent="submit"
        />

        <div class="model-picker">
          <button class="model-trigger" :disabled="isLoadingModels" @click="toggleModelMenu">
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

        <button class="send" @click="submit">
          <ArrowRight class="icon icon--inverse" />
        </button>
      </div>
      <div class="hint">Enter — отправить (Shift+Enter добавим позже)</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ArrowRight, Cpu, Search } from 'lucide-vue-next'
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useModelStore } from '../modelStore'

const router = useRouter()
const q = ref('')

const { models, selectedModel, isLoadingModels, loadModels, setModel } = useModelStore()
const showModelMenu = ref(false)

function toggleModelMenu() {
  showModelMenu.value = !showModelMenu.value
}

function selectModel(model: string) {
  setModel(model)
  showModelMenu.value = false
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
  color: #111827;
  letter-spacing: -0.02em;
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
.model-picker {
  position: relative;
}
.model-trigger {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  border: 1px solid var(--border);
  background: #fff;
  display: grid;
  place-items: center;
  cursor: pointer;
}
.model-trigger:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}
.model-menu {
  position: absolute;
  right: 0;
  top: 52px;
  background: #fff;
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 8px;
  display: grid;
  gap: 6px;
  min-width: 220px;
  z-index: 10;
  box-shadow: 0 10px 30px rgba(15, 23, 42, 0.08);
}
.model-option {
  border: 0;
  background: transparent;
  text-align: left;
  padding: 8px 10px;
  border-radius: 10px;
  cursor: pointer;
  font-size: 12px;
  color: #111827;
}
.model-option:hover {
  background: var(--hover);
}
.model-option--active {
  background: rgba(15, 118, 110, 0.12);
  color: #0f766e;
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
