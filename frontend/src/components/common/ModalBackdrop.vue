<template>
  <div class="modal-backdrop" @click.self="$emit('close')">
    <div class="modal" :class="modalClass">
      <div class="modal-header">
        <div class="modal-title">{{ title }}</div>
        <button class="modal-close-icon" @click="$emit('close')" aria-label="Close">
          <X class="icon icon--small" />
        </button>
      </div>
      <slot />
      <div v-if="$slots.actions" class="modal-actions">
        <slot name="actions" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { X } from 'lucide-vue-next'

defineProps<{
  title: string
  modalClass?: string
}>()

defineEmits<{
  close: []
}>()
</script>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.3);
  display: grid;
  place-items: center;
  z-index: 50;
}

.modal {
  background: var(--card-bg);
  border-radius: 16px;
  padding: 18px;
  width: min(420px, 92vw);
  box-shadow: 0 20px 40px rgba(15, 23, 42, 0.3);
  display: grid;
  gap: 12px;
  border: 1px solid var(--border);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.modal-title {
  font-size: 14px;
  font-weight: 600;
}

.modal-close-icon {
  border: 0;
  background: transparent;
  cursor: pointer;
  padding: 4px;
  border-radius: 8px;
  display: grid;
  place-items: center;
  color: var(--muted);
}

.modal-close-icon:hover {
  background: var(--hover);
  color: var(--fg);
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
}

.icon--small {
  width: 14px;
  height: 14px;
}
</style>
