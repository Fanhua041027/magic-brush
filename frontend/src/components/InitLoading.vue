<template>
  <div class="init-loading-overlay">
    <div class="loading-content">
      <div class="loader-wrapper">
        <div class="loader-ring"></div>
        <div class="loader-ring inner"></div>
        <div class="loader-glow"></div>
      </div>
      <div class="text-container">
        <div class="status-text">{{ statusText }}</div>
        <div class="sub-text">
          <span v-if="status === 'loading-model'">正在初始化神经网络...</span>
          <span v-else-if="status === 'initializing'">系统启动中...</span>
          <span v-else>请稍候</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  status: { type: String, default: 'initializing' }
})

const statusText = computed(() => {
  switch (props.status) {
    case 'initializing': return 'GHOST SLOVE'
    case 'loading-model': return 'LOADING MODELS'
    case 'ready': return 'READY'
    default: return 'PROCESSING'
  }
})
</script>

<style scoped>
.init-loading-overlay {
  position: fixed;
  inset: 0;
  background: var(--surface-overlay);
  backdrop-filter: blur(20px);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 9999;
  font-family: var(--font-sans);
  border-radius: var(--radius-xl);
  border: 1px solid var(--border-subtle);
}

.loading-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 40px;
}

.loader-wrapper {
  position: relative;
  width: 80px; height: 80px;
  display: flex;
  justify-content: center;
  align-items: center;
}

.loader-ring {
  position: absolute;
  width: 100%; height: 100%;
  border-radius: 50%;
  border: 2px solid transparent;
  border-top-color: var(--accent);
  border-right-color: var(--accent-glow);
  animation: spin 1.2s cubic-bezier(0.5, 0, 0.5, 1) infinite;
}

.loader-ring.inner {
  width: 70%; height: 70%;
  border-top-color: var(--color-info);
  border-right-color: transparent;
  border-left-color: var(--color-info);
  animation: spin 1.5s cubic-bezier(0.5, 0, 0.5, 1) infinite reverse;
}

.loader-glow {
  position: absolute;
  width: 100%; height: 100%;
  background: radial-gradient(circle, var(--accent-glow) 0%, transparent 70%);
  animation: pulse 2s ease-in-out infinite;
}

.text-container {
  text-align: center;
  display: flex;
  flex-direction: column;
  gap: var(--sp-2);
  animation: fadeInUp 0.6s ease-out;
}

.status-text {
  font-size: var(--text-base);
  font-weight: 700;
  letter-spacing: 3px;
  color: var(--text-primary);
  text-transform: uppercase;
}

.sub-text {
  font-size: var(--text-xs);
  color: var(--text-muted);
  font-weight: 400;
  letter-spacing: 0.5px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

@keyframes pulse {
  0%, 100% { transform: scale(0.8); opacity: 0.5; }
  50% { transform: scale(1.2); opacity: 0.8; }
}

@keyframes fadeInUp {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>