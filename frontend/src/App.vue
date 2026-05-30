<template>
  <TopBar @openSettings="settingsStore.openSettings" />

  <WelcomeView v-if="!ui.hasStarted && solution.history.length === 0" />
  <SolveView v-else />

  <ScreenshotDock />
  <KBPanel @select-result="onKBSelect" />
  <SettingsModal />
  <ChatDialog />
  <ScreenshotFollowUp ref="followUpRef" />

  <Teleport to="body">
    <Transition name="overlay-fade">
      <div v-if="ui.showResumeWarning" class="overlay">
        <div class="warn-dialog">
          <div class="warn-icon-wrap">
            <Icon name="alert-triangle" :size="32" class="warn-svg-icon" />
          </div>
          <div class="warn-title">简历尚未解析</div>
          <p class="warn-desc">你已选择简历文件，但还没有解析成 Markdown。继续解题会跳过简历内容，建议先在设置中完成解析。</p>
          <div class="warn-actions">
            <button class="btn btn-ghost" @click="cancelSolve">取消</button>
            <button class="btn btn-accent" @click="continueSolve">继续解题</button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>

  <div id="toast-container">
    <div v-for="(t, i) in ui.toasts" :key="t.id || i" class="toast" :class="[t.type, { show: t.show }]">
      <Icon :name="toastIcon(t.type)" :size="14" class="toast-icon" />
      <span>{{ t.text }}</span>
    </div>
  </div>

  <div class="disclaimer">AI-Assistant 的回答仅供参考。</div>

  <!-- 全局录音指示器 -->
  <Teleport to="body">
    <Transition name="overlay-fade">
      <div v-if="voice.isRecording" class="global-recording-indicator">
        <span class="recording-dot"></span>
        <span>录音中...松开左 Alt 结束</span>
      </div>
    </Transition>
  </Teleport>

  <ResizeHandle />
</template>

<script setup>
import { onMounted, ref } from 'vue'
import TopBar from './components/TopBar.vue'
import WelcomeView from './components/WelcomeView.vue'
import SolveView from './components/SolveView.vue'
import ScreenshotDock from './components/ScreenshotDock.vue'
import SettingsModal from './components/SettingsModal.vue'
import ResizeHandle from './components/ResizeHandle.vue'
import Icon from './components/Icon.vue'
import KBPanel from './components/KBPanel.vue'
import ChatDialog from './components/ChatDialog.vue'
import ScreenshotFollowUp from './components/ScreenshotFollowUp.vue'

import { useUIStore } from './stores/ui'
import { useSettingsStore } from './stores/settings'
import { useSolutionStore } from './stores/solution'
import { useVoiceStore } from './stores/voice'
import { useChatStore } from './stores/chat'
import { on } from './services/events'
import { api } from './services/api'
import { wsService } from './services/websocket'
import { initCodeBlockInteractions } from './utils/markdown-latex'

const ui = useUIStore()
const settingsStore = useSettingsStore()
const solution = useSolutionStore()
const voice = useVoiceStore()
const chatStore = useChatStore()

const followUpRef = ref(null)
let pendingSolveCallback = null
let currentScreenshot = null

function toastIcon(type) {
  if (type === 'error') return 'x-circle'
  if (type === 'success') return 'check-circle'
  if (type === 'warning') return 'alert-triangle'
  return 'info'
}

function cancelSolve() {
  ui.showResumeWarning = false
  pendingSolveCallback = null
}

function continueSolve() {
  ui.showResumeWarning = false
  if (pendingSolveCallback) {
    pendingSolveCallback()
    pendingSolveCallback = null
  }
}

function onKBSelect(item) {
  console.log('KB selected:', item)
}

onMounted(() => {
  initCodeBlockInteractions()

  // 初始化 WebSocket 连接
  wsService.connect()
  wsService.startHeartbeat()

  // 监听 WebSocket 事件
  wsService.on('stt-streaming', (text) => {
    window.dispatchEvent(new CustomEvent('stt-streaming-text', { detail: text }))
  })

  wsService.on('error', (error) => {
    console.error('[App] WebSocket error:', error)
    // 不显示错误提示，避免频繁弹出
    // WebSocket 会自动重连
  })

  wsService.on('connected', () => {
    console.log('[App] WebSocket connected')
  })

  wsService.on('disconnected', () => {
    console.log('[App] WebSocket disconnected')
  })

  api.getInitStatus().then(s => { ui.initStatus = s })
  on('init-status', (s) => { ui.initStatus = s })

  settingsStore.loadSettings().then(() => {
    settingsStore.resetStatus()
  })

  on('key-recorded', (data) => {
    if (data && data.action) {
      if (settingsStore.tempShortcuts[data.action]) {
        settingsStore.tempShortcuts[data.action].keyName = data.keyName
        settingsStore.tempShortcuts[data.action].vkCode = data.comboID
      } else {
        settingsStore.tempShortcuts[data.action] = { keyName: data.keyName, vkCode: data.comboID }
      }
      if (settingsStore.recordingAction === data.action) {
        settingsStore.recordingText = data.keyName
      }
    }
  })

  on('shortcut-error', async (msg) => {
    ui.showToast(msg, 'error', 2000)
    const targetAction = settingsStore.recordingAction
    settingsStore.recordingAction = null
    settingsStore.recordingText = ''
    api.stopRecordingKey()
    if (!targetAction) return
    try {
      if (settingsStore.shortcuts[targetAction]?.keyName) {
        settingsStore.tempShortcuts[targetAction] = JSON.parse(JSON.stringify(settingsStore.shortcuts[targetAction]))
      } else {
        delete settingsStore.tempShortcuts[targetAction]
      }
    } catch (e) {
      console.error('回滚快捷键配置失败', e)
    }
  })

  on('shortcut-saved', (action) => {
    if (settingsStore.recordingAction === action) {
      settingsStore.recordingAction = null
      ui.showToast('快捷键已保存', 'success')
    }
  })

  on('user-message', (screenshot) => {
    solution.setUserScreenshot(screenshot)
    currentScreenshot = screenshot
  })

  on('start-solving', () => {
    const s = settingsStore.settings
    if (s.resumePath && !s.resumeContent) {
      pendingSolveCallback = proceedWithSolve
      ui.showResumeWarning = true
      return
    }
    proceedWithSolve()
  })

  function proceedWithSolve() {
    solution.errorState.show = false
    ui.flash('solve')
    settingsStore.statusText = '正在思考...'
    settingsStore.statusIcon = '...'
    ui.mainVisible = true
    ui.hasStarted = true
    const s = settingsStore.settings
    if (s.keepContext && solution.history.length > 0 && solution.activeHistoryIndex === 0) {
      solution.isLoading = false
      solution.isAppending = true
      setTimeout(() => {
        const el = document.getElementById('content')
        if (el) el.scrollTop = el.scrollHeight
      }, 50)
    } else {
      solution.isLoading = true
      solution.isAppending = false
    }
  }

  on('toggle-visibility', (isVisibleToCapture) => {
    ui.flash('toggle')
    ui.isStealthMode = isVisibleToCapture
    ui.showToast(isVisibleToCapture ? '隐身模式已开启' : '隐身模式已关闭', isVisibleToCapture ? 'info' : 'success')
  })

  on('solution', (data) => {
    settingsStore.statusText = '解题完成'
    settingsStore.statusIcon = '✓'
    solution.handleSolution(data)
    // 解题完成后显示追问对话框
    if (currentScreenshot && followUpRef.value) {
      setTimeout(() => {
        followUpRef.value.show(currentScreenshot, data, '')
        currentScreenshot = null
      }, 500)
    }
  })

  on('copy-code', () => {
    const old = settingsStore.statusText
    settingsStore.statusText = '已复制'
    setTimeout(() => (settingsStore.statusText = old), 2000)
  })

  on('click-through-state', (enabled) => {
    ui.isClickThrough = enabled
    const el = document.getElementById('main-interface')
    if (el) el.style.pointerEvents = enabled ? 'none' : 'auto'
  })

  on('scroll-content', (direction) => {
    const el = document.getElementById('content')
    if (!el) return
    el.scrollBy({ top: direction === 'up' ? -50 : 50, behavior: 'smooth' })
  })

  on('solution-stream-start', () => {
    ui.hasStarted = true
    solution.handleStreamStart(settingsStore.settings.keepContext)
  })

  on('solution-stream-chunk', (token) => solution.handleStreamChunk(token))
  on('solution-stream-thinking', (token) => solution.handleThinkingChunk(token))

  on('solution-error', (rawErrMsg) => {
    if (rawErrMsg && (rawErrMsg.includes('context canceled') || rawErrMsg.includes('canceled'))) {
      handleUserCancellation()
      return
    }
    let title = '请求出错'
    let desc = rawErrMsg || '未知错误'
    const icon = 'x-circle'
    try {
      const errObj = JSON.parse(rawErrMsg)
      if (errObj.message) desc = errObj.message
      if (errObj.statusCode) title = `错误 ${errObj.statusCode}`
    } catch (_) {}
    settingsStore.statusText = '出错'
    settingsStore.statusIcon = '!'
    const usedInline = solution.handleInlineError({ title, desc, icon })
    if (!usedInline) {
      Object.assign(solution.errorState, { show: true, title, desc, icon, rawError: rawErrMsg, showDetails: false })
      solution.isLoading = false
      solution.isAppending = false
      solution.shouldOverwriteHistory = true
    }
  })

  function handleUserCancellation() {
    if (solution.history.length > 0 && solution.activeHistoryIndex === 0) {
      const current = solution.history[0]
      if (settingsStore.settings.keepContext && current.rounds?.length > 1) {
        current.rounds.pop()
        solution.setStreamBuffer('')
        solution.isAppending = true
        solution.isLoading = false
        solution.shouldOverwriteHistory = false
      } else {
        if (current.rounds?.length) current.rounds[0].aiResponse = ''
        solution.setStreamBuffer('')
        solution.isLoading = true
        settingsStore.statusText = '正在思考...'
        settingsStore.statusIcon = '...'
        solution.shouldOverwriteHistory = true
      }
    }
  }

  on('stt-recording-started', () => {
    voice.isRecording = true
    voice.transcribedText = ''
    // 触发自定义事件
    window.dispatchEvent(new CustomEvent('stt-recording-started'))
  })

  on('stt-transcribed', (text) => {
    voice.isRecording = false
    voice.transcribedText = text || ''
    // 触发自定义事件
    window.dispatchEvent(new CustomEvent('stt-transcribed', { detail: text }))
    // 语音转写后自动打开对话框
    if (text && text.trim()) {
      chatStore.show()
    }
  })

  on('stt-recording-stopped', () => {
    voice.isRecording = false
    // 触发自定义事件
    window.dispatchEvent(new CustomEvent('stt-recording-stopped'))
  })

  on('stt-streaming-text', (text) => {
    // 触发流式转写事件
    window.dispatchEvent(new CustomEvent('stt-streaming-text', { detail: text }))
  })

  on('require-api-key', () => {
    if (!ui.showSettings) settingsStore.openSettings()
    ui.activeTab = 'api'
    ui.showToast('请先配置 API Key', 'warning')
  })

  on('open-settings', (tab) => {
    if (!ui.showSettings) settingsStore.openSettings()
    if (tab) ui.activeTab = tab
  })

  on('toast', (msg) => {
    ui.showToast(msg)
  })

  on('open-chat', () => {
    chatStore.show()
  })

  document.addEventListener('keydown', event => {
    if (
      event.key === 'F12' ||
      (event.ctrlKey && event.shiftKey && event.key === 'I') ||
      (event.ctrlKey && event.shiftKey && event.key === 'J') ||
      (event.ctrlKey && event.key === 'U')
    ) {
      event.preventDefault()
    }
  })
})
</script>

<style scoped>
.overlay {
  position: fixed;
  z-index: var(--z-modal);
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: auto;
  background: var(--surface-overlay);
  backdrop-filter: blur(6px);
}

.warn-dialog {
  background: var(--surface-elevated);
  border-radius: var(--radius-xl);
  padding: var(--sp-8);
  max-width: 380px;
  text-align: center;
  border: 1px solid var(--border-default);
  box-shadow: var(--shadow-xl);
}

.warn-icon-wrap {
  width: 56px;
  height: 56px;
  margin: 0 auto var(--sp-4);
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--warning-bg);
  border-radius: var(--radius-lg);
  border: 1px solid var(--warning-border);
}

.warn-svg-icon { color: var(--color-warning); }
.warn-title {
  font-size: var(--text-lg);
  font-weight: var(--weight-bold);
  color: var(--text-primary);
  margin-bottom: var(--sp-2);
}

.warn-desc {
  font-size: var(--text-sm);
  color: var(--text-secondary);
  line-height: var(--leading-relaxed);
  margin-bottom: var(--sp-6);
}

.warn-actions {
  display: flex;
  gap: var(--sp-3);
  justify-content: center;
}

.btn {
  padding: var(--sp-2) var(--sp-5);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  font-weight: var(--weight-semibold);
  cursor: pointer;
  transition: all var(--duration-fast) ease;
  border: none;
}

.btn-ghost {
  background: var(--surface-card);
  color: var(--text-primary);
  border: 1px solid var(--border-default);
}

.btn-ghost:hover { background: var(--surface-card-hover); }

.btn-accent {
  background: var(--accent);
  color: white;
}

.btn-accent:hover { background: var(--accent-hover); }

.overlay-fade-enter-active { transition: all 0.25s var(--ease-out); }
.overlay-fade-leave-active { transition: all 0.2s ease-in; }
.overlay-fade-enter-from,
.overlay-fade-leave-to { opacity: 0; }
.overlay-fade-enter-from .warn-dialog,
.overlay-fade-leave-to .warn-dialog { transform: scale(0.95) translateY(8px); }

.disclaimer {
  text-align: center;
  font-size: 10px;
  color: var(--text-muted);
  padding: 2px 0 0;
  pointer-events: none;
  user-select: none;
  flex-shrink: 0;
  letter-spacing: 0.3px;
  opacity: 0.7;
}
/* 全局录音指示器 */
.global-recording-indicator {
  position: fixed;
  top: 20px;
  left: 50%;
  transform: translateX(-50%);
  background: rgba(255, 59, 48, 0.9);
  color: white;
  padding: 8px 16px;
  border-radius: 20px;
  font-size: 14px;
  display: flex;
  align-items: center;
  gap: 8px;
  z-index: 99999;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.3);
  pointer-events: none;
}

.recording-dot {
  width: 10px;
  height: 10px;
  background: white;
  border-radius: 50%;
  animation: pulse 1s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}
</style>
