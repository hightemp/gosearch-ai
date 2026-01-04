import { ref } from 'vue'
import { apiFetch } from './api'

const models = ref<string[]>([])
const selectedModel = ref('')
const isLoadingModels = ref(false)
const loaded = ref(false)
const storageKey = 'gosearch.selectedModel'

function readStoredModel() {
  try {
    return localStorage.getItem(storageKey) || ''
  } catch {
    return ''
  }
}

function persistModel(model: string) {
  try {
    localStorage.setItem(storageKey, model)
  } catch {
    // ignore storage errors
  }
}

function setModel(model: string) {
  const trimmed = model.trim()
  if (!trimmed) return
  selectedModel.value = trimmed
  persistModel(trimmed)
}

async function loadModels() {
  if (loaded.value || isLoadingModels.value) return
  isLoadingModels.value = true
  try {
    const resp = await apiFetch('/models')
    if (resp.ok) {
      const data = (await resp.json()) as { models?: string[] }
      if (Array.isArray(data.models)) {
        models.value = data.models
      }
    }
  } catch {
    // ignore, fallback below
  } finally {
    if (!models.value.length) {
      models.value = ['openai/gpt-4.1-mini']
    }
    if (!selectedModel.value) {
      const stored = readStoredModel()
      selectedModel.value = stored || models.value[0]
    }
    loaded.value = true
    isLoadingModels.value = false
  }
}

export function useModelStore() {
  return {
    models,
    selectedModel,
    isLoadingModels,
    loadModels,
    setModel
  }
}
