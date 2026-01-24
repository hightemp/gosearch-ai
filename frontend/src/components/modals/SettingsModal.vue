<template>
  <ModalBackdrop title="Settings" @close="$emit('close')">
    <div class="settings-content">
      <div class="settings-section">
        <div class="settings-section-title">Appearance</div>
        <div class="settings-item">
          <div class="settings-item-label">Theme</div>
          <div class="theme-switcher">
            <button
              v-for="opt in themeOptions"
              :key="opt.value"
              class="theme-option"
              :class="{ 'theme-option--active': theme === opt.value }"
              @click="$emit('setTheme', opt.value)"
            >
              <component :is="opt.icon" class="icon icon--small" />
              <span>{{ opt.label }}</span>
            </button>
          </div>
        </div>
      </div>

      <div class="settings-section">
        <div class="settings-section-title">Model</div>
        <div class="settings-item">
          <div class="settings-item-label">Response model</div>
          <div class="model-options">
            <button
              v-for="m in models"
              :key="m"
              class="model-option"
              :class="{ 'model-option--active': m === selectedModel }"
              @click="$emit('setModel', m)"
            >
              {{ m }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <template #actions>
      <button class="modal-close" @click="$emit('close')">Close</button>
    </template>
  </ModalBackdrop>
</template>

<script setup lang="ts">
import { Monitor, Moon, Sun } from 'lucide-vue-next'
import ModalBackdrop from '../common/ModalBackdrop.vue'

export type Theme = 'light' | 'dark' | 'system'

const themeOptions: { value: Theme; label: string; icon: typeof Sun }[] = [
  { value: 'light', label: 'Light', icon: Sun },
  { value: 'dark', label: 'Dark', icon: Moon },
  { value: 'system', label: 'System', icon: Monitor }
]

defineProps<{
  theme: Theme
  models: string[]
  selectedModel: string
}>()

defineEmits<{
  close: []
  setTheme: [theme: Theme]
  setModel: [model: string]
}>()
</script>

<style scoped>
.settings-content {
  overflow-y: auto;
  display: grid;
  gap: 20px;
  padding: 8px 0;
}

.settings-section {
  display: grid;
  gap: 12px;
}

.settings-section-title {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--muted);
}

.settings-item {
  display: grid;
  gap: 8px;
}

.settings-item-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--fg);
}

.theme-switcher {
  display: flex;
  gap: 8px;
}

.theme-option {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 12px 8px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--card-bg);
  cursor: pointer;
  color: var(--fg);
  font-size: 12px;
  transition: all 0.15s;
}

.theme-option:hover {
  border-color: var(--accent);
  background: var(--accent-light);
}

.theme-option--active {
  border-color: var(--accent);
  background: var(--accent-light);
  color: var(--accent);
}

.model-options {
  display: grid;
  gap: 6px;
  max-height: 220px;
  overflow-y: auto;
}

.model-option {
  border: 0;
  background: transparent;
  text-align: left;
  padding: 8px 10px;
  border-radius: 10px;
  cursor: pointer;
  font-size: 12px;
  color: var(--fg);
}

.model-option:hover {
  background: var(--hover);
}

.model-option--active {
  background: var(--accent-light);
  color: var(--accent);
}

.modal-close {
  border: 1px solid var(--border);
  background: var(--card-bg);
  padding: 6px 12px;
  border-radius: 10px;
  cursor: pointer;
  color: var(--fg);
  font-size: 13px;
}

.modal-close:hover {
  background: var(--hover);
}

.icon--small {
  width: 14px;
  height: 14px;
}
</style>
