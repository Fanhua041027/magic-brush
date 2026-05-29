<template>
  <div class="error-container">
    <div class="error-icon-wrap">
      <Icon :name="errorState.icon || 'x-circle'" :size="32" class="error-icon" />
    </div>
    <div class="error-title">{{ errorState.title }}</div>
    <div class="error-desc">{{ errorState.desc }}</div>
    <div class="error-hint">
      按 <kbd class="shortcut-key">{{ solveShortcut }}</kbd> 重试
    </div>
  </div>
</template>

<script setup>
import Icon from './Icon.vue'

defineProps({
  errorState: Object,
  solveShortcut: String
})
</script>

<style scoped>
.error-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  text-align: center;
  padding: var(--sp-5);
  animation: fadeIn 0.3s var(--ease-out);
}
@keyframes fadeIn {
  from { opacity: 0; transform: translateY(6px); }
  to { opacity: 1; transform: translateY(0); }
}

.error-icon-wrap {
  width: 64px;
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--error-bg);
  border: 1px solid var(--error-border);
  border-radius: var(--radius-xl);
  margin-bottom: var(--sp-5);
}
.error-icon {
  color: var(--color-error);
}
.error-title {
  font-size: var(--text-xl);
  font-weight: var(--weight-bold);
  color: var(--text-primary);
  margin-bottom: var(--sp-2);
}
.error-desc {
  font-size: var(--text-sm);
  color: var(--text-secondary);
  margin-bottom: var(--sp-6);
  max-width: 80%;
  line-height: var(--leading-relaxed);
}
.error-hint {
  font-size: var(--text-sm);
  color: var(--text-tertiary);
  display: flex;
  align-items: center;
  gap: 6px;
  animation: pulseOpacity 2s infinite;
}
@keyframes pulseOpacity {
  0% { opacity: 0.5; }
  50% { opacity: 1; }
  100% { opacity: 0.5; }
}
.shortcut-key {
  background: var(--accent-muted);
  padding: 2px 6px;
  border-radius: var(--radius-xs);
  font-family: var(--font-mono);
  font-weight: var(--weight-semibold);
  color: var(--accent);
  border: 1px solid var(--accent-border);
  font-size: var(--text-xs);
}
</style>
