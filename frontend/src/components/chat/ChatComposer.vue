<template>
  <div class="composer">
    <textarea
      ref="textareaRef"
      :value="modelValue"
      class="composer-input"
      :placeholder="placeholder"
      rows="1"
      @input="onInput"
      @keydown.enter.exact.prevent="$emit('submit')"
    />
    <ModelPicker
      :models="models"
      :selected-model="selectedModel"
      :is-loading="isLoadingModels"
      position="top"
      @select="$emit('selectModel', $event)"
    />
    <button
      class="composer-send"
      :disabled="!canSubmit"
      @click="$emit('submit')"
    >
      <ArrowRight class="composer-icon" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { ArrowRight } from 'lucide-vue-next'
import { ref, watch } from 'vue'
import ModelPicker from '../ModelPicker.vue'

const props = defineProps<{
  modelValue: string
  placeholder?: string
  canSubmit?: boolean
  models: string[]
  selectedModel: string
  isLoadingModels?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  submit: []
  selectModel: [model: string]
}>()

const textareaRef = ref<HTMLTextAreaElement | null>(null)

function onInput(e: Event) {
  const target = e.target as HTMLTextAreaElement
  emit('update:modelValue', target.value)
  autoResize()
}

function autoResize() {
  const el = textareaRef.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 200) + 'px'
}

function resetHeight() {
  const el = textareaRef.value
  if (!el) return
  el.style.height = 'auto'
}

watch(() => props.modelValue, (newVal) => {
  if (!newVal) {
    resetHeight()
  }
})

defineExpose({ resetHeight })
</script>

<style scoped>
.composer {
  position: sticky;
  bottom: 18px;
  margin-top: 26px;
  display: grid;
  grid-template-columns: 1fr auto auto;
  gap: 10px;
  background: var(--card-bg);
  border: 1px solid var(--border);
  border-radius: 16px;
  padding: 12px;
}

.composer-input {
  border: 0;
  outline: none;
  font-size: 14px;
  resize: none;
  min-height: 44px;
  max-height: 200px;
  overflow-y: auto;
  line-height: 1.5;
  padding: 10px 0;
  font-family: inherit;
  background: transparent;
  color: var(--fg);
}

.composer-input::placeholder {
  color: var(--muted);
}

.composer-send {
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

.composer-send:hover:not(:disabled) {
  background: var(--accent-hover);
}

.composer-send:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.composer-icon {
  width: 18px;
  height: 18px;
}
</style>
