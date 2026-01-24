<template>
  <div class="message-list">
    <MessageItem
      v-for="msg in messages"
      :key="msg.id"
      :message="msg"
    />
    <div v-if="isRunning" class="message message--assistant">
      <div class="message-role">Ассистент</div>
      <div class="message-body">
        <LoadingDots label="Идет генерация" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import LoadingDots from '../common/LoadingDots.vue'
import MessageItem, { type MessageData } from './MessageItem.vue'

defineProps<{
  messages: MessageData[]
  isRunning?: boolean
}>()
</script>

<style scoped>
.message-list {
  display: grid;
  gap: 16px;
}

.message {
  display: grid;
  gap: 6px;
}

.message--assistant {
  border-left: 3px solid var(--accent);
  padding-left: 12px;
}

.message-role {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--muted);
}

.message-body {
  font-size: 14px;
  color: var(--fg);
  line-height: 1.6;
}
</style>
