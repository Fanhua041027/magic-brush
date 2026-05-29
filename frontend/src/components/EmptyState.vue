<template>
  <div class="empty-state" :style="{ opacity: contentOpacity }">
    <div class="empty-glow"></div>
    <div class="empty-content">
      <div class="empty-icon-wrap">
        <Icon name="lightbulb" :size="28" class="empty-icon" />
      </div>
      <h3 class="empty-title">准备就绪</h3>
      <p class="empty-desc">
        按 <kbd class="shortcut-key">{{ shortcut }}</kbd> 截图，<kbd class="shortcut-key">{{ sendShortcut }}</kbd> 发送解题
      </p>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useSettingsStore } from '../stores/settings'
import Icon from './Icon.vue'

const settingsStore = useSettingsStore()

defineProps({
  shortcut: { type: String, default: 'F8' },
  sendShortcut: { type: String, default: 'Ctrl+J' }
})

const contentOpacity = computed(() => {
  const t = settingsStore.settings.transparency ?? 0
  if (t >= 0.5) return 0
  return 1.0 - t * 2
})
</script>

<style scoped>
.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  position: relative;
  overflow: hidden;
}
.empty-glow {
  position: absolute;
  width: 180px;
  height: 180px;
  background: radial-gradient(circle, var(--accent-glow) 0%, transparent 70%);
  border-radius: 50%;
  filter: blur(40px);
  animation: breathe 4s ease-in-out infinite;
}
.empty-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--sp-3);
  text-align: center;
  z-index: 1;
}
.empty-icon-wrap {
  width: 52px;
  height: 52px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--accent-muted);
  border: 1px solid var(--accent-border);
  border-radius: var(--radius-lg);
  color: var(--accent);
  animation: float 3s ease-in-out infinite;
}
.empty-title {
  margin: 0;
  font-size: var(--text-xl);
  font-weight: var(--weight-semibold);
  color: var(--text-primary);
}
.empty-desc {
  margin: 0;
  font-size: var(--text-sm);
  color: var(--text-muted);
  display: flex;
  align-items: center;
  gap: 6px;
}
.shortcut-key {
  background: var(--accent-muted);
  color: var(--accent);
  padding: 3px 8px;
  border-radius: var(--radius-xs);
  font-family: var(--font-mono);
  font-size: var(--text-xs);
  font-weight: var(--weight-semibold);
  border: 1px solid var(--accent-border);
}

@keyframes breathe {
  0%, 100% { transform: scale(1); opacity: 0.5; }
  50% { transform: scale(1.15); opacity: 0.7; }
}
@keyframes float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-5px); }
}
</style>
