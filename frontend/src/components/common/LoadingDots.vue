<template>
  <span class="loading-dots" :aria-label="label">
    <span class="dot" />
    <span class="dot" />
    <span class="dot" />
  </span>
</template>

<script setup lang="ts">
defineProps<{
  label?: string
}>()
</script>

<style scoped>
.loading-dots {
  display: inline-flex;
  gap: 6px;
  align-items: center;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: var(--accent);
  animation: loading-pulse 1.1s infinite ease-in-out;
}

.dot:nth-child(2) {
  animation-delay: 0.15s;
}

.dot:nth-child(3) {
  animation-delay: 0.3s;
}

@keyframes loading-pulse {
  0%,
  80%,
  100% {
    opacity: 0.3;
    transform: scale(0.9);
  }
  40% {
    opacity: 1;
    transform: scale(1);
  }
}
</style>
