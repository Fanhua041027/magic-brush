<template>
  <div class="welcome-screen">
    <div v-if="shouldShow" class="welcome-content" :style="{ opacity: contentOpacity }">
      <!-- Logo -->
      <div class="logo-section">
        <div class="logo-badge">
          <span class="logo-letter">Q</span>
        </div>
        <div class="logo-glow"></div>
      </div>

      <!-- Title -->
      <div class="title-section">
        <h1 class="main-title">Q-SOLVER</h1>
        <p class="subtitle">AI 智能助手 · 即刻开始</p>
      </div>

      <!-- Status -->
      <Transition name="fade-slide" mode="out-in">
        <div v-if="ui.initStatus !== 'ready'" class="status-pill loading" key="loading">
          <div class="spinner"></div>
          <span>{{ ui.initStatus === 'loading-model' ? '正在加载模型...' : '系统初始化中...' }}</span>
        </div>

        <div v-else-if="showSuccess" class="status-pill success" key="success">
          <Icon name="check-circle" :size="15" />
          <span>系统就绪</span>
        </div>

        <div v-else class="shortcuts-group" key="hints">
          <div class="shortcut-pill">
            <kbd>{{ settingsStore.solveShortcut }}</kbd>
            <span>截图</span>
          </div>
          <span class="shortcut-sep">·</span>
          <div class="shortcut-pill">
            <kbd>{{ settingsStore.sendShortcut }}</kbd>
            <span>发送解题</span>
          </div>
          <span class="shortcut-sep">·</span>
          <div class="shortcut-pill">
            <kbd>{{ settingsStore.toggleShortcut }}</kbd>
            <span>隐藏窗口</span>
          </div>
        </div>
      </Transition>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useUIStore } from '../stores/ui'
import { useSettingsStore } from '../stores/settings'
import Icon from './Icon.vue'

const ui = useUIStore()
const settingsStore = useSettingsStore()

const showSuccess = ref(false)

const contentOpacity = computed(() => {
  const t = settingsStore.settings.transparency ?? 0
  if (t >= 0.5) return 0
  return 1.0 - t * 2
})

const shouldShow = computed(() => {
  return (settingsStore.settings.transparency ?? 0) < 0.5
})

watch(() => ui.initStatus, (newVal, oldVal) => {
  if (newVal === 'ready' && oldVal !== 'ready') {
    showSuccess.value = true
    setTimeout(() => { showSuccess.value = false }, 1500)
  }
})
</script>

<style scoped>
.welcome-screen {
  flex: 1;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  min-height: 0;
  pointer-events: auto;
}

.welcome-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--sp-6);
  z-index: 1;
  pointer-events: auto;
  transition: opacity 0.3s ease;
}

/* Logo */
.logo-section {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}
.logo-badge {
  width: 64px;
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--accent-gradient);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-accent);
  position: relative;
  z-index: 1;
}
.logo-letter {
  font-size: 28px;
  font-weight: 800;
  color: white;
  font-family: var(--font-display);
  letter-spacing: -1px;
}
.logo-glow {
  position: absolute;
  width: 100px;
  height: 100px;
  background: radial-gradient(circle, var(--accent-glow) 0%, transparent 70%);
  border-radius: 50%;
  filter: blur(20px);
  animation: glowBreathe 4s ease-in-out infinite;
}
@keyframes glowBreathe {
  0%, 100% { transform: scale(1); opacity: 0.5; }
  50% { transform: scale(1.15); opacity: 0.8; }
}

/* Title */
.title-section { text-align: center; }
.main-title {
  font-size: var(--text-3xl);
  font-weight: var(--weight-bold);
  letter-spacing: 5px;
  color: var(--text-primary);
  margin: 0 0 var(--sp-2) 0;
  font-family: var(--font-display);
}
.subtitle {
  font-size: var(--text-sm);
  color: var(--text-muted);
  margin: 0;
  letter-spacing: 1.5px;
}

/* Status pills */
.status-pill {
  display: flex;
  align-items: center;
  gap: var(--sp-2-5);
  padding: var(--sp-2-5) var(--sp-5);
  border-radius: var(--radius-full);
  border: 1px solid var(--border-subtle);
  font-size: var(--text-sm);
  font-weight: var(--weight-medium);
}
.status-pill.loading {
  background: var(--surface-card);
  color: var(--text-secondary);
}
.status-pill.success {
  background: var(--accent-muted);
  border-color: var(--accent-border);
  color: var(--accent);
}
.spinner {
  width: 14px;
  height: 14px;
  border: 2px solid transparent;
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

/* Shortcuts */
.shortcuts-group {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  user-select: none;
}
.shortcut-pill {
  display: flex;
  align-items: center;
  gap: var(--sp-1-5);
  font-size: var(--text-xs);
  color: var(--text-muted);
}
.shortcut-pill kbd {
  padding: 3px 7px;
  border-radius: var(--radius-xs);
  background: var(--surface-card);
  border: 1px solid var(--border-subtle);
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: var(--weight-semibold);
  color: var(--text-secondary);
  line-height: 1;
}
.shortcut-sep {
  color: var(--text-muted);
  opacity: 0.3;
}

/* Transitions */
.fade-slide-enter-active, .fade-slide-leave-active { transition: all 0.4s var(--ease-out); }
.fade-slide-enter-from { opacity: 0; transform: translateY(8px); }
.fade-slide-leave-to { opacity: 0; transform: translateY(-8px); }
</style>
