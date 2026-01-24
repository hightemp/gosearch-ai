<template>
  <div class="model-picker">
    <button
      class="model-trigger"
      :disabled="isLoading"
      @click="toggleMenu"
      aria-label="Выбрать модель"
    >
      <Cpu class="icon icon--small" />
    </button>
    <div v-if="showMenu" class="model-menu" :class="`model-menu--${position}`">
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
</template>

<script setup lang="ts">
import { Cpu } from 'lucide-vue-next'
import { ref } from 'vue'

const props = defineProps<{
  models: string[]
  selectedModel: string
  isLoading?: boolean
  position?: 'top' | 'bottom'
}>()

const emit = defineEmits<{
  select: [model: string]
}>()

const showMenu = ref(false)

function toggleMenu() {
  if (props.isLoading) return
  showMenu.value = !showMenu.value
}

function selectModel(model: string) {
  emit('select', model)
  showMenu.value = false
}
</script>

<style scoped>
.model-picker {
  position: relative;
}

.model-trigger {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  border: 1px solid var(--border);
  background: var(--card-bg);
  display: grid;
  place-items: center;
  cursor: pointer;
  color: var(--fg);
}

.model-trigger:hover:not(:disabled) {
  background: var(--hover);
}

.model-trigger:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.model-menu {
  position: absolute;
  right: 0;
  background: var(--card-bg);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 8px;
  display: grid;
  gap: 6px;
  min-width: 220px;
  z-index: 20;
  box-shadow: 0 10px 30px rgba(15, 23, 42, 0.15);
}

.model-menu--top {
  bottom: 52px;
}

.model-menu--bottom {
  top: 52px;
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

.icon--small {
  width: 14px;
  height: 14px;
}
</style>
