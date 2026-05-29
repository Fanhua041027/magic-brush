<template>
  <div class="top-bar-wrapper" style="--wails-draggable: drag">
    <div class="top-bar">
      <div class="bar-left" style="--wails-draggable: no-drag">
        <div class="logo">Q</div>
        <div class="scene-pill">
          <Icon name="camera" :size="14" />
          <span>截图解题</span>
        </div>
      </div>

      <div class="bar-right" style="--wails-draggable: no-drag">
        <div class="shortcuts-hint">
          <kbd>{{ settingsStore.solveShortcut }}</kbd>
          <kbd>{{ settingsStore.sendShortcut }}</kbd>
          <kbd>{{ settingsStore.toggleShortcut }}</kbd>
        </div>

        <div class="bar-divider" />

        <button class="bar-btn" ref="statusBtnRef" @click="toggleStatusPanel" :title="'状态: ' + settingsStore.statusText">
          <span class="status-dot" :class="statusClass"></span>
        </button>

        <div class="bar-divider" />

        <ThemeToggle />

        <button class="bar-btn" @click="$emit('openSettings')" title="设置"
          @mouseenter="showSettingsTooltip" @mouseleave="showSettingsTip = false" ref="settingsBtnRef">
          <Icon name="settings" :size="15" />
        </button>

        <button class="bar-btn bar-btn-quit" @click="ui.quit" title="退出">
          <Icon name="power" :size="15" />
        </button>
      </div>
    </div>
  </div>

  <Teleport to="body">
    <Transition name="panel-fade">
      <div class="status-panel" v-if="showStatusPanel" :style="panelStyle" @click.stop>
        <div class="sp-header">
          <span class="sp-title">运行状态</span>
          <button class="sp-close" @click="showStatusPanel = false">
            <Icon name="x" :size="14" />
          </button>
        </div>
        <div class="sp-body">
          <div class="sp-row">
            <span class="sp-label">API Key</span>
            <span class="sp-value" :class="settingsStore.settings.apiKey ? 'ok' : 'warn'">
              {{ settingsStore.settings.apiKey ? '已配置' : '未配置' }}
            </span>
          </div>
          <div class="sp-row">
            <span class="sp-label">Base URL</span>
            <span class="sp-value model">{{ settingsStore.settings.baseURL || 'https://api.openai.com/v1' }}</span>
          </div>
          <div class="sp-row">
            <span class="sp-label">使用模型</span>
            <span class="sp-value model">{{ settingsStore.settings.model || '未设置' }}</span>
          </div>
          <div class="sp-row">
            <span class="sp-label">隐身模式</span>
            <span class="sp-value" :class="ui.isStealthMode ? 'ok' : 'err'">{{ ui.isStealthMode ? '已开启' : '已关闭' }}</span>
          </div>
        </div>
      </div>
    </Transition>

    <div class="settings-tooltip" v-if="showSettingsTip" :style="settingsTooltipStyle">
      <Icon name="alert-triangle" :size="13" class="tooltip-icon" />
      <span>打开设置会获取焦点，录屏期间请避免操作</span>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useUIStore } from '../stores/ui'
import { useSettingsStore } from '../stores/settings'
import Icon from './Icon.vue'
import ThemeToggle from './ThemeToggle.vue'

defineEmits(['openSettings'])

const ui = useUIStore()
const settingsStore = useSettingsStore()

const statusClass = computed(() => {
  const text = settingsStore.statusText || ''
  if (text === '已连接' || text === '就绪' || text === '解题完成' || text.includes('思考') || text.includes('复制')) return 'connected'
  if (text.includes('未配置')) return 'unconfigured'
  if (text.includes('无效') || text.includes('Key')) return 'invalid-key'
  if (text.includes('失败') || text.includes('出错')) return 'disconnected'
  return 'unconfigured'
})

const showStatusPanel = ref(false)
const statusBtnRef = ref(null)
const panelStyle = reactive({ top: '0px', left: '0px' })

function toggleStatusPanel() {
  if (showStatusPanel.value) {
    showStatusPanel.value = false
    return
  }
  if (statusBtnRef.value) {
    const rect = statusBtnRef.value.getBoundingClientRect()
    panelStyle.top = `${rect.bottom + 8}px`
    const pw = 280
    let left = rect.left + rect.width / 2 - pw / 2
    if (left + pw > window.innerWidth - 10) left = window.innerWidth - pw - 10
    if (left < 10) left = 10
    panelStyle.left = `${left}px`
    showStatusPanel.value = true
  }
}

function handleClickOutside(e) {
  if (showStatusPanel.value && statusBtnRef.value && !statusBtnRef.value.contains(e.target)) {
    showStatusPanel.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})

const showSettingsTip = ref(false)
const settingsBtnRef = ref(null)
const settingsTooltipStyle = reactive({ top: '0px', left: '0px' })

function showSettingsTooltip() {
  if (settingsBtnRef.value) {
    const rect = settingsBtnRef.value.getBoundingClientRect()
    settingsTooltipStyle.top = `${rect.bottom + 10}px`
    const tooltipW = 310
    const halfW = tooltipW / 2
    let centerX = rect.left + rect.width / 2
    if (centerX + halfW > window.innerWidth - 8) {
      centerX = window.innerWidth - 8 - halfW
    }
    if (centerX - halfW < 8) {
      centerX = 8 + halfW
    }
    settingsTooltipStyle.left = `${centerX}px`
    showSettingsTip.value = true
  }
}
</script>

<style scoped>
.top-bar-wrapper {
  align-self: center;
  padding: 44px var(--sp-4) var(--sp-2);
  pointer-events: auto;
  display: flex;
  justify-content: center;
  margin-top: calc(-1 * var(--sp-3));
  user-select: none;
  width: 100%;
  position: relative;
  z-index: 50;
}
.top-bar {
  background: var(--surface-elevated);
  backdrop-filter: blur(24px);
  padding: var(--sp-1) var(--sp-2);
  border-radius: var(--radius-full);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sp-2);
  pointer-events: auto;
  box-shadow: var(--shadow-md), 0 0 0 1px var(--border-subtle);
  border: 1px solid var(--border-default);
  transition: border-color var(--duration-base) ease;
  white-space: nowrap;
  width: 100%;
  max-width: 95vw;
}
.top-bar:hover { border-color: var(--border-hover); }
.bar-left {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
}
.logo {
  width: 26px;
  height: 26px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--accent-gradient);
  border-radius: var(--radius-sm);
  font-size: 13px;
  font-weight: 800;
  color: white;
  flex-shrink: 0;
  letter-spacing: -0.5px;
}
.scene-pill {
  display: flex;
  align-items: center;
  gap: var(--sp-1);
  background: var(--surface-card);
  border-radius: var(--radius-full);
  padding: var(--sp-1) var(--sp-2-5);
  color: var(--text-primary);
  font-size: var(--text-xs);
  font-weight: var(--weight-semibold);
  white-space: nowrap;
}
.bar-right {
  display: flex;
  align-items: center;
  gap: 2px;
}
.shortcuts-hint {
  display: flex;
  gap: var(--sp-1);
  align-items: center;
}
.shortcuts-hint kbd {
  padding: 2px 5px;
  border-radius: var(--radius-xs);
  background: var(--surface-card);
  border: 1px solid var(--border-subtle);
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: var(--weight-semibold);
  color: var(--text-muted);
  line-height: 1;
}
.bar-divider {
  width: 1px;
  height: 14px;
  background: var(--border-default);
  margin: 0 var(--sp-1);
  flex-shrink: 0;
}
.bar-btn {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  border-radius: var(--radius-sm);
  color: var(--text-muted);
  cursor: pointer;
  transition: all var(--duration-fast) ease;
  flex-shrink: 0;
}
.bar-btn:hover {
  background: var(--surface-card-hover);
  color: var(--text-primary);
}
.bar-btn-quit:hover { color: var(--color-error); }
.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
  transition: all var(--duration-fast) ease;
}
.status-dot.connected { background: var(--color-success); box-shadow: 0 0 6px var(--color-success); }
.status-dot.unconfigured { background: var(--color-warning); box-shadow: 0 0 6px var(--color-warning); }
.status-dot.invalid-key,
.status-dot.disconnected { background: var(--color-error); box-shadow: 0 0 6px var(--color-error); }
.status-panel {
  position: fixed;
  width: 280px;
  background: var(--surface-popover);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-xl);
  backdrop-filter: blur(24px);
  z-index: var(--z-tooltip);
  overflow: hidden;
  pointer-events: auto;
  transform-origin: top center;
}
.sp-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--sp-3) var(--sp-4);
  border-bottom: 1px solid var(--border-subtle);
}
.sp-title { font-size: var(--text-sm); font-weight: var(--weight-bold); color: var(--text-primary); }
.sp-close {
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  border-radius: var(--radius-xs);
  color: var(--text-muted);
  cursor: pointer;
  transition: all var(--duration-fast) ease;
}
.sp-close:hover { background: var(--surface-card-hover); color: var(--text-primary); }
.sp-body { padding: var(--sp-2) var(--sp-4) var(--sp-3); }
.sp-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--sp-2) 0;
  border-bottom: 1px solid var(--border-subtle);
}
.sp-row:last-child { border-bottom: none; }
.sp-label { font-size: var(--text-xs); color: var(--text-muted); }
.sp-value {
  font-size: var(--text-xs);
  font-weight: var(--weight-semibold);
  font-family: var(--font-mono);
  white-space: nowrap;
  color: var(--text-primary);
}
.sp-value.ok { color: var(--color-success); }
.sp-value.warn { color: var(--color-warning); }
.sp-value.err { color: var(--color-error); }
.sp-value.model { max-width: 160px; overflow: hidden; text-overflow: ellipsis; }
.panel-fade-enter-active { transition: all 0.2s var(--ease-out); }
.panel-fade-leave-active { transition: all 0.15s ease-in; pointer-events: none; }
.panel-fade-enter-from, .panel-fade-leave-to {
  opacity: 0;
  transform: translateY(-8px) scale(0.96);
}
.settings-tooltip {
  position: fixed;
  transform: translateX(-50%);
  background: var(--surface-popover);
  border: 1px solid var(--warning-border);
  border-radius: var(--radius-md);
  padding: var(--sp-2) var(--sp-3);
  z-index: var(--z-tooltip);
  box-shadow: var(--shadow-lg);
  backdrop-filter: blur(16px);
  pointer-events: none;
  animation: tooltipIn 0.2s var(--ease-out);
  display: flex;
  align-items: center;
  gap: var(--sp-1-5);
  font-size: var(--text-xs);
  color: var(--color-warning);
  font-weight: var(--weight-semibold);
  white-space: nowrap;
  max-width: calc(100vw - 16px);
}
.settings-tooltip::before {
  content: '';
  position: absolute;
  top: -5px;
  left: 50%;
  transform: translateX(-50%);
  border-width: 0 5px 5px 5px;
  border-style: solid;
  border-color: transparent transparent var(--warning-border) transparent;
}
.tooltip-icon { flex-shrink: 0; }
@keyframes tooltipIn {
  from { opacity: 0; transform: translate(-50%, -4px); }
  to { opacity: 1; transform: translate(-50%, 0); }
}
</style>
