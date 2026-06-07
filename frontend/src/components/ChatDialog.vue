<template>
  <Teleport to="body">
    <Transition name="interview-fade">
      <div v-if="chatStore.isVisible" class="interview-overlay" :style="overlayStyle">
        <div class="interview-container" :style="[containerStyle, posStyle]">
          <!-- ═══ Top Bar (Drag Handle) ═══ -->
          <header class="interview-topbar" @mousedown="onTopbarMouseDown">
            <div class="topbar-left">
              <span class="timer-display">
                <Icon name="clock" :size="13" />
                <span>{{ formattedTime }}</span>
              </span>
            </div>
            <div class="topbar-center">
              <Icon name="sparkles" :size="16" class="brand-icon" />
              <span class="brand-name">AI 辅助面试</span>
              <span v-if="settingsStore.settings.resumeContent" class="brand-badge">
                <Icon name="file-text" :size="11" />
                简历已加载
              </span>
            </div>
            <div class="topbar-right">
              <div class="opacity-control">
                <button class="tb-btn" @click="showBgSlider = !showBgSlider" title="背景透明度（越大越透明）">
                  <Icon name="sun" :size="13" />
                </button>
                <Transition name="slide-fade">
                  <div v-if="showBgSlider" class="opacity-slider-wrap">
                    <span class="slider-label">背景</span>
                    <input type="range" class="opacity-slider" min="0" max="100" step="1"
                      v-model.number="transparencyLevel" @input="saveOpacity" />
                    <span class="opacity-value">{{ transparencyLevel }}%</span>
                  </div>
                </Transition>
              </div>
              <div class="opacity-control">
                <button class="tb-btn" @click="showFontSlider = !showFontSlider" title="文字透明度">
                  <Icon name="type" :size="13" />
                </button>
                <Transition name="slide-fade">
                  <div v-if="showFontSlider" class="opacity-slider-wrap">
                    <span class="slider-label">文字</span>
                    <input type="range" class="opacity-slider" min="0" max="100" step="1"
                      v-model.number="fontOpacity" @input="saveFontOpacity" />
                    <span class="opacity-value">{{ fontOpacity }}%</span>
                  </div>
                </Transition>
              </div>
              <button class="tb-btn" @click="chatStore.clearHistory" title="清除对话">
                <Icon name="trash" :size="14" />
              </button>
              <button class="tb-btn" @click="chatStore.exportHistory" title="导出历史">
                <Icon name="download" :size="14" />
              </button>
              <button class="tb-btn tb-close" @click="close">
                <Icon name="x" :size="16" />
              </button>
            </div>
          </header>

          <!-- ═══ Main Content: 三栏布局 ═══ -->
          <main class="interview-main" :style="textStyle">
            <!-- ─── 左栏：对话转录 ─── -->
            <section class="col-transcript">
              <div class="col-header">
                <Icon name="mic" :size="13" />
                <span>对话转录</span>
                <span class="col-badge">{{ transcripts.length }}</span>
              </div>
              <div class="col-body" ref="transcriptRef">
                <div v-if="transcripts.length === 0" class="col-empty">
                  <Icon name="message-circle" :size="24" class="empty-icon" />
                  <p>面试对话将实时显示在此处</p>
                  <p class="empty-hint">左侧 Alt 键录音 · 自动区分说话人</p>
                </div>
                <div
                  v-for="(t, i) in transcripts"
                  :key="i"
                  class="transcript-bubble"
                  :class="t.speaker"
                >
                  <div class="bubble-label">{{ t.speaker === 'interviewer' ? '面试官' : '我' }}</div>
                  <div class="bubble-content">{{ t.content }}</div>
                  <div class="bubble-time">{{ t.time }}</div>
                </div>
              </div>
            </section>

            <!-- ─── 中栏：AI 智能体 ─── -->
            <section class="col-agent">
              <div class="col-header">
                <Icon name="brain" :size="13" />
                <span>AI 智能体</span>
                <span v-if="chatStore.isLoading" class="thinking-badge">
                  <span class="think-dot"></span>
                  思考中
                </span>
              </div>
              <div class="col-body" ref="agentRef">
                <div v-if="chatStore.messages.length === 0 && !chatStore.isLoading" class="col-empty">
                  <Icon name="sparkles" :size="24" class="empty-icon" />
                  <p>AI 建议将显示在此处</p>
                  <p class="empty-hint">输入问题或点击快捷按钮开始</p>
                </div>
                <div v-for="(msg, i) in chatStore.messages" :key="i" class="agent-message" :class="msg.role">
                  <!-- User message -->
                  <div v-if="msg.role === 'user'" class="agent-user-msg">
                    <div class="user-label">你的问题</div>
                    <div class="user-text">{{ msg.content }}</div>
                  </div>
                  <!-- AI response -->
                  <div v-else class="agent-ai-block">
                    <div class="ai-header">
                      <span class="ai-tag">
                        <Icon name="sparkles" :size="10" />
                        AI 回答
                      </span>
                      <span v-if="msg._streaming" class="streaming-indicator">生成中...</span>
                    </div>
                    <div class="ai-content" v-html="renderMarkdown(msg.content)"></div>
                    <div v-if="msg.time && !msg._streaming" class="ai-time">{{ msg.time }}</div>
                  </div>
                </div>
                <!-- Loading state -->
                <div v-if="chatStore.isLoading && chatStore.messages[chatStore.messages.length - 1]?.role !== 'assistant'" class="agent-thinking">
                  <div class="thinking-line"><span class="think-tag">分析</span>理解问题中...</div>
                  <div class="thinking-line"><span class="think-tag">检索</span>搜索知识库...</div>
                  <div class="thinking-line active"><span class="think-tag">生成</span>
                    <span class="typing-dots">
                      <span class="tdot"></span><span class="tdot"></span><span class="tdot"></span>
                    </span>
                  </div>
                </div>
              </div>
            </section>

            <!-- ─── 右栏：输入与控制 ─── -->
            <section class="col-input">
              <!-- Quick Actions -->
              <div class="quick-actions">
                <button class="qa-btn" @click="insertQuestion('请解释这道题的思路')">
                  <Icon name="lightbulb" :size="12" />
                  <span>解题思路</span>
                </button>
                <button class="qa-btn" @click="insertQuestion('请帮我优化这段代码')">
                  <Icon name="code" :size="12" />
                  <span>代码优化</span>
                </button>
                <button class="qa-btn" @click="insertQuestion('用 STAR 法则回答这个问题')">
                  <Icon name="star" :size="12" />
                  <span>STAR 回答</span>
                </button>
              </div>

              <!-- Input Area -->
              <div class="input-area">
                <div v-if="attachedScreenshot" class="attach-preview">
                  <img :src="attachedScreenshot" class="attach-img" alt="截图" />
                  <button class="attach-remove" @click="attachedScreenshot = null">
                    <Icon name="x" :size="14" />
                  </button>
                </div>
                <div class="input-wrapper">
                  <textarea
                    v-model="inputText"
                    class="ai-input"
                    placeholder="输入你的问题..."
                    @keydown.enter.exact="sendMessage"
                    rows="1"
                    ref="inputRef"
                  ></textarea>
                  <button class="input-action-btn" @click="takeScreenshot" title="附截图 (F8)">
                    <Icon name="camera" :size="16" />
                  </button>
                </div>
              </div>

              <!-- Control Bar -->
              <div class="control-bar">
                <div class="voice-indicator" :class="{ recording: voiceStore.isRecording }">
                  <Icon :name="voiceStore.isRecording ? 'mic' : 'mic-off'" :size="13" />
                  <span>{{ voiceStore.isRecording ? '录音中...松开 Alt' : '左 Alt 语音' }}</span>
                </div>
                <button
                  class="send-btn"
                  :disabled="!inputText.trim() || chatStore.isLoading"
                  @click="sendMessage"
                >
                  <Icon name="send" :size="14" />
                  <span>发送</span>
                </button>
              </div>

              <!-- ─── 快捷键面板 ─── -->
              <div class="shortcuts-panel" :class="{ collapsed: shortcutsCollapsed }">
                <div class="sp-header" @click="shortcutsCollapsed = !shortcutsCollapsed">
                  <Icon name="keyboard" :size="12" />
                  <span>快捷键</span>
                  <Icon :name="shortcutsCollapsed ? 'chevron-up' : 'chevron-down'" :size="12" class="sp-chevron" />
                </div>
                <div v-if="!shortcutsCollapsed" class="sp-body">
                  <div class="sp-item"><kbd>F8</kbd><span>截图</span></div>
                  <div class="sp-item"><kbd>F6</kbd><span>追问截图</span></div>
                  <div class="sp-item"><kbd>左 Alt</kbd><span>语音输入</span></div>
                  <div class="sp-item"><kbd>Enter</kbd><span>发送消息</span></div>
                  <div class="sp-item"><kbd>Esc</kbd><span>关闭面板</span></div>
                </div>
              </div>
            </section>
          </main>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, watch, nextTick, onMounted, onUnmounted, computed, reactive } from 'vue'
import Icon from './Icon.vue'
import { useChatStore } from '../stores/chat'
import { useVoiceStore } from '../stores/voice'
import { useSettingsStore } from '../stores/settings'
import { useUIStore } from '../stores/ui'
import { renderMarkdownWithLatex } from '../utils/markdown-latex'
import { api } from '../services/api'

const chatStore = useChatStore()
const voiceStore = useVoiceStore()
const settingsStore = useSettingsStore()
const ui = useUIStore()

const inputText = ref('')
const inputRef = ref(null)
const transcriptRef = ref(null)
const agentRef = ref(null)
const attachedScreenshot = ref(null)
const shortcutsCollapsed = ref(false)
const streamingText = ref('')

// ── 面试计时器 ──────────────────────────────────────────────
const timerSeconds = ref(0)
let timerInterval = null

const formattedTime = computed(() => {
  const m = String(Math.floor(timerSeconds.value / 60)).padStart(2, '0')
  const s = String(timerSeconds.value % 60).padStart(2, '0')
  return `${m}:${s}`
})

function startTimer() {
  if (timerInterval) return
  timerInterval = setInterval(() => { timerSeconds.value++ }, 1000)
}

function stopTimer() {
  if (timerInterval) {
    clearInterval(timerInterval)
    timerInterval = null
  }
}

// ── 透明度控制 + 窗口拖动 ──────────────────────────────────
const OPACITY_KEY = 'magic-brush-interview-opacity'
const FONT_OPACITY_KEY = 'magic-brush-interview-font-opacity'
const POS_KEY = 'magic-brush-interview-pos'
const showBgSlider = ref(false)
const showFontSlider = ref(false)
const transparencyLevel = ref(8)
const fontOpacity = ref(100)
const t = () => 1 - transparencyLevel.value / 100
const containerStyle = computed(() => ({
  background: `rgba(20, 22, 30, ${t()})`,
}))
const overlayStyle = computed(() => {
  // 自由悬浮模式：不显示遮罩，点击穿透
  return {
    background: 'transparent',
    pointerEvents: 'none',
  }
})
const textStyle = computed(() => ({
  opacity: fontOpacity.value / 100,
  transition: 'opacity 0.2s ease',
}))

function loadOpacity() {
  try {
    const saved = localStorage.getItem(OPACITY_KEY)
    if (saved) {
      const val = parseFloat(saved)
      if (val > 1) {
        transparencyLevel.value = Math.max(0, Math.min(100, val))
      } else if (val >= 0.3 && val <= 0.98) {
        transparencyLevel.value = Math.round((1 - val) * 100)
      }
    }
    const fontSaved = localStorage.getItem(FONT_OPACITY_KEY)
    if (fontSaved) {
      const v = parseFloat(fontSaved)
      if (v >= 0 && v <= 100) fontOpacity.value = v
    }
  } catch (e) { /* ignore */ }
}
function saveOpacity() {
  try { localStorage.setItem(OPACITY_KEY, String(transparencyLevel.value)) } catch (e) { /* ignore */ }
}
function saveFontOpacity() {
  try { localStorage.setItem(FONT_OPACITY_KEY, String(fontOpacity.value)) } catch (e) { /* ignore */ }
}

// ── 窗口拖动（长按触发） ──────────────────────────────────
const dialogPos = reactive({ x: 0, y: 0 })
const posStyle = computed(() => ({
  left: `${dialogPos.x}px`,
  top: `${dialogPos.y}px`,
}))
let isDragging = false
let isDraggableReady = false
let dragOffset = { x: 0, y: 0 }
let longPressTimer = null
const LONG_PRESS_MS = 150

function loadPosition() {
  try {
    const saved = localStorage.getItem(POS_KEY)
    if (saved) {
      const p = JSON.parse(saved)
      if (typeof p.x === 'number' && typeof p.y === 'number') {
        dialogPos.x = p.x; dialogPos.y = p.y
        return
      }
    }
  } catch (e) { /* ignore */ }
  dialogPos.x = Math.max(0, (window.innerWidth - 860) / 2)
  dialogPos.y = Math.max(0, (window.innerHeight - 640) / 2)
}
function savePosition() {
  try { localStorage.setItem(POS_KEY, JSON.stringify({ x: dialogPos.x, y: dialogPos.y })) } catch (e) { /* ignore */ }
}

function onTopbarMouseDown(e) {
  // 点击按钮时不触发拖动
  if (e.target.closest('.topbar-right, .tb-btn, .opacity-control, .opacity-slider-wrap')) return
  isDraggableReady = false
  longPressTimer = setTimeout(() => {
    isDraggableReady = true
    beginDrag(e)
  }, LONG_PRESS_MS)
  document.addEventListener('mousemove', onTopbarMove)
  document.addEventListener('mouseup', cancelLongPress)
}

function onTopbarMove(e) {
  if (!isDraggableReady) {
    // 鼠标移动超过阈值，立即开始拖动
    if (Math.abs(e.movementX) > 3 || Math.abs(e.movementY) > 3) {
      clearTimeout(longPressTimer)
      isDraggableReady = true
      beginDrag(e)
    }
  } else if (isDragging) {
    onDrag(e)
  }
}

function cancelLongPress() {
  clearTimeout(longPressTimer)
  isDraggableReady = false
  document.removeEventListener('mousemove', onTopbarMove)
  document.removeEventListener('mouseup', cancelLongPress)
}

function beginDrag(e) {
  isDragging = true
  const rect = document.querySelector('.interview-container')?.getBoundingClientRect()
  dragOffset.x = rect ? e.clientX - rect.left : e.clientX - dialogPos.x
  dragOffset.y = rect ? e.clientY - rect.top : e.clientY - dialogPos.y
  document.removeEventListener('mousemove', onTopbarMove)
  document.addEventListener('mousemove', onDrag)
  document.addEventListener('mouseup', stopDrag)
}

function onDrag(e) {
  if (!isDragging) return
  dialogPos.x = e.clientX - dragOffset.x
  dialogPos.y = e.clientY - dragOffset.y
  // 无边界限制，可随意拖到桌面任意位置
  // 只做极端的越界保护，防止完全拖出视线无法拉回
  dialogPos.x = Math.max(-window.innerWidth + 100, Math.min(window.innerWidth - 100, dialogPos.x))
  dialogPos.y = Math.max(-window.innerHeight + 60, Math.min(window.innerHeight - 60, dialogPos.y))
}

function stopDrag() {
  isDragging = false
  isDraggableReady = false
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
  savePosition()
}
const transcripts = ref([])

function addTranscript(speaker, content) {
  transcripts.value.push({
    speaker,
    content,
    time: new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
    timestamp: Date.now(),
  })
  // 自动滚动
  nextTick(() => {
    if (transcriptRef.value) {
      transcriptRef.value.scrollTop = transcriptRef.value.scrollHeight
    }
  })
}

// ── 截图 ────────────────────────────────────────────────────
async function takeScreenshot() {
  try {
    const result = await api.triggerScreenshot()
    if (result) {
      attachedScreenshot.value = result
      ui.showToast('截图已附加', 'success', 1500)
    }
  } catch (e) {
    console.error('Screenshot error:', e)
  }
}

// ── 快捷问题 ────────────────────────────────────────────────
function insertQuestion(text) {
  inputText.value = text
  nextTick(() => { inputRef.value?.focus() })
}

// ── Markdown ────────────────────────────────────────────────
function renderMarkdown(text) {
  if (!text) return ''
  return renderMarkdownWithLatex(text)
}

// ── 发送消息 ────────────────────────────────────────────────
async function sendMessage() {
  const text = inputText.value.trim()
  if (!text || chatStore.isLoading) return
  inputText.value = ''

  // 如果有截图，附加上
  if (attachedScreenshot.value) {
    await chatStore.sendMessageWithScreenshot(text, attachedScreenshot.value)
    attachedScreenshot.value = null
  } else {
    await chatStore.sendMessage(text)
  }
}

function close() {
  stopTimer()
  chatStore.hide()
}

// ── 事件监听 ────────────────────────────────────────────────
onMounted(() => {
  startTimer()
  loadOpacity()
  loadPosition()

  // 流式输出
  window.addEventListener('chat-stream-chunk', onStreamChunk)
  window.addEventListener('chat-stream-done', onStreamDone)
  window.addEventListener('chat-stream-error', onStreamError)

  // 语音转写
  window.addEventListener('stt-streaming-text', (e) => {
    if (e.detail) {
      streamingText.value += e.detail
      inputText.value = streamingText.value
    }
  })
  window.addEventListener('stt-recording-started', () => {
    streamingText.value = ''
    addTranscript('interviewer', '...')
  })
  window.addEventListener('stt-recording-stopped', () => { streamingText.value = '' })
  window.addEventListener('stt-transcribed', (e) => {
    if (e.detail) {
      inputText.value = e.detail
      // 将最终转写加入转录区
      addTranscript('me', e.detail)
      streamingText.value = ''
    }
  })
})

onUnmounted(() => {
  stopTimer()
  window.removeEventListener('chat-stream-chunk', onStreamChunk)
  window.removeEventListener('chat-stream-done', onStreamDone)
  window.removeEventListener('chat-stream-error', onStreamError)
})

function onStreamChunk(e) { chatStore.handleStreamChunk(e.detail) }
function onStreamDone() { chatStore.handleStreamDone() }
function onStreamError(e) { chatStore.handleStreamError(e.detail) }

// ── 自动滚动 ────────────────────────────────────────────────
watch(() => chatStore.messages.length, () => {
  nextTick(() => {
    if (agentRef.value) agentRef.value.scrollTop = agentRef.value.scrollHeight
  })
})

watch(() => chatStore.isVisible, (v) => {
  if (v) {
    startTimer()
    nextTick(() => { inputRef.value?.focus() })
  } else {
    stopTimer()
  }
})
</script>

<style scoped>
/* ═══ Overlay ═══ */
.interview-overlay {
  position: fixed;
  inset: 0;
  z-index: 999;
  pointer-events: none;
}

.interview-container {
  position: fixed;
  z-index: 1000;
  width: 860px;
  max-width: 94vw;
  height: 640px;
  max-height: 85vh;
  background: rgba(24, 26, 35, 0.92);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 16px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  pointer-events: auto;
  user-select: none;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(20px);
}

/* ═══ Top Bar ═══ */
.interview-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  background: rgba(15, 17, 23, 0.8);
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  flex-shrink: 0;
  height: 40px;
  cursor: grab;
}
.interview-topbar:active { cursor: grabbing; }

.topbar-left, .topbar-right { display: flex; align-items: center; gap: 6px; }
.topbar-center { display: flex; align-items: center; gap: 8px; }

.timer-display {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.4);
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.5px;
  background: rgba(0, 0, 0, 0.3);
  padding: 3px 10px;
  border-radius: 6px;
}

.brand-icon { color: #818cf8; }
.brand-name {
  font-size: 14px;
  font-weight: 700;
  color: rgba(255, 255, 255, 0.9);
  letter-spacing: 0.3px;
}
.brand-badge {
  display: flex; align-items: center; gap: 3px;
  padding: 2px 7px;
  background: rgba(16, 185, 129, 0.1);
  border: 1px solid rgba(16, 185, 129, 0.15);
  border-radius: 5px;
  font-size: 10px;
  font-weight: 600;
  color: #34d399;
}

.tb-btn {
  width: 28px; height: 28px;
  border-radius: 6px;
  border: none;
  background: transparent;
  color: rgba(255, 255, 255, 0.35);
  cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  transition: all 0.15s;
}
.tb-btn:hover { background: rgba(255, 255, 255, 0.06); color: rgba(255, 255, 255, 0.6); }
.tb-close:hover { background: rgba(239, 68, 68, 0.1); color: #ef4444; }

/* ─── 透明度滑块 ─── */
.opacity-control { position: relative; display: flex; align-items: center; }
.opacity-slider-wrap {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  background: rgba(30, 32, 42, 0.95);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(0,0,0,0.4);
  z-index: 10;
  white-space: nowrap;
}
.opacity-slider {
  width: 80px;
  height: 4px;
  -webkit-appearance: none;
  appearance: none;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 2px;
  outline: none;
  cursor: pointer;
}
.opacity-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  width: 14px; height: 14px;
  border-radius: 50%;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  border: 2px solid rgba(255,255,255,0.15);
  cursor: pointer;
  transition: box-shadow 0.15s;
}
.opacity-slider::-webkit-slider-thumb:hover { box-shadow: 0 0 12px rgba(99,102,241,0.4); }
.opacity-value {
  font-size: 11px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.4);
  min-width: 30px;
  text-align: right;
  font-variant-numeric: tabular-nums;
}

.slider-label {
  font-size: 10px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.25);
  margin-right: 4px;
  letter-spacing: 0.3px;
}
.slide-fade-enter-active { transition: all 0.2s ease; }
.slide-fade-leave-active { transition: all 0.15s ease; }
.slide-fade-enter-from,
.slide-fade-leave-to { opacity: 0; transform: translateY(-4px); }

/* ═══ Main: 三栏 ═══ */
.interview-main {
  flex: 1;
  display: grid;
  grid-template-columns: 1fr 1.6fr 1fr;
  gap: 1px;
  background: rgba(255, 255, 255, 0.04);
  overflow: hidden;
}

.col-transcript,
.col-agent,
.col-input {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  background: rgba(0, 0, 0, 0.15);
}

/* ─── Column Header ─── */
.col-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 14px;
  font-size: 11px;
  font-weight: 700;
  color: rgba(255, 255, 255, 0.35);
  text-transform: uppercase;
  letter-spacing: 0.8px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
  flex-shrink: 0;
  background: rgba(0, 0, 0, 0.15);
}
.col-badge {
  margin-left: auto;
  font-size: 10px;
  background: rgba(255, 255, 255, 0.06);
  padding: 1px 7px;
  border-radius: 5px;
  color: rgba(255, 255, 255, 0.3);
}
.thinking-badge {
  display: flex; align-items: center; gap: 4px;
  margin-left: auto;
  font-size: 10px;
  font-weight: 600;
  text-transform: none;
  letter-spacing: normal;
  color: #818cf8;
  background: rgba(99, 102, 241, 0.1);
  padding: 2px 8px;
  border-radius: 5px;
}
.think-dot {
  width: 5px; height: 5px;
  border-radius: 50%;
  background: #818cf8;
  animation: think-pulse 1s infinite;
}
@keyframes think-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.2; }
}

/* ─── Column Body ─── */
.col-body {
  flex: 1;
  overflow-y: auto;
  padding: 10px;
  scrollbar-width: thin;
  scrollbar-color: rgba(255,255,255,0.06) transparent;
}
.col-body::-webkit-scrollbar { width: 3px; }
.col-body::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.06); border-radius: 2px; }

.col-empty {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  height: 100%; min-height: 200px;
  text-align: center;
  color: rgba(255, 255, 255, 0.2);
  gap: 6px;
}
.col-empty p { margin: 0; font-size: 13px; }
.empty-icon { color: rgba(255, 255, 255, 0.08); }
.empty-hint { font-size: 11px !important; color: rgba(255, 255, 255, 0.15); }

/* ═══ 左栏：转录气泡 ═══ */
.transcript-bubble {
  padding: 8px 10px;
  margin-bottom: 6px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.03);
  border-left: 2px solid transparent;
}
.transcript-bubble.interviewer {
  border-left-color: rgba(255, 255, 255, 0.15);
}
.transcript-bubble.me {
  border-left-color: rgba(99, 102, 241, 0.4);
  background: rgba(99, 102, 241, 0.04);
}
.bubble-label {
  font-size: 10px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.25);
  margin-bottom: 3px;
}
.transcript-bubble.me .bubble-label { color: rgba(99, 102, 241, 0.5); }
.bubble-content {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.7);
  line-height: 1.5;
  word-break: break-word;
}
.bubble-time {
  font-size: 10px;
  color: rgba(255, 255, 255, 0.15);
  margin-top: 4px;
}

/* ═══ 中栏：AI 消息 ═══ */
.agent-message { margin-bottom: 12px; }

.agent-user-msg {
  text-align: right;
  padding: 8px 10px;
  background: rgba(99, 102, 241, 0.08);
  border-radius: 8px;
  border: 1px solid rgba(99, 102, 241, 0.1);
}
.user-label {
  font-size: 10px;
  font-weight: 600;
  color: rgba(99, 102, 241, 0.5);
  margin-bottom: 4px;
}
.user-text {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.8);
  line-height: 1.5;
}

.agent-ai-block {
  border: 1px solid rgba(139, 92, 246, 0.12);
  border-radius: 8px;
  overflow: hidden;
}
.ai-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 6px 10px;
  background: rgba(139, 92, 246, 0.06);
  border-bottom: 1px solid rgba(139, 92, 246, 0.08);
}
.ai-tag {
  display: flex; align-items: center; gap: 4px;
  font-size: 10px;
  font-weight: 700;
  color: #a78bfa;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.streaming-indicator {
  font-size: 10px;
  color: rgba(255, 255, 255, 0.25);
  animation: stream-pulse 1s infinite;
}
@keyframes stream-pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.3; } }

.ai-content {
  padding: 10px 12px;
  font-size: 13px;
  line-height: 1.7;
  color: rgba(255, 255, 255, 0.85);
  word-break: break-word;
}
.ai-content :deep(p) { margin: 0 0 8px; }
.ai-content :deep(p:last-child) { margin-bottom: 0; }
.ai-content :deep(code) {
  background: rgba(0, 0, 0, 0.3);
  padding: 1px 5px;
  border-radius: 3px;
  font-family: var(--font-mono, monospace);
  font-size: 12px;
  color: #e2e8f0;
}
.ai-content :deep(pre) {
  background: rgba(0, 0, 0, 0.4);
  padding: 10px 12px;
  border-radius: 6px;
  overflow-x: auto;
  margin: 8px 0;
  border: 1px solid rgba(255, 255, 255, 0.05);
}
.ai-content :deep(pre code) {
  background: transparent;
  padding: 0;
  font-size: 12px;
  line-height: 1.6;
}
.ai-time {
  font-size: 10px;
  color: rgba(255, 255, 255, 0.12);
  padding: 4px 10px 6px;
}

/* ─── AI Loading / Thinking ─── */
.agent-thinking {
  padding: 10px 12px;
  border: 1px solid rgba(139, 92, 246, 0.12);
  border-radius: 8px;
  background: rgba(139, 92, 246, 0.03);
}
.thinking-line {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.3);
  padding: 3px 0;
  display: flex; align-items: center; gap: 6px;
}
.thinking-line.active { color: rgba(255, 255, 255, 0.6); }
.think-tag {
  font-size: 9px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  padding: 1px 5px;
  border-radius: 3px;
  background: rgba(255, 255, 255, 0.04);
  color: rgba(255, 255, 255, 0.25);
}
.thinking-line.active .think-tag { background: rgba(99, 102, 241, 0.15); color: #818cf8; }

.typing-dots { display: inline-flex; gap: 2px; margin-left: 4px; }
.tdot {
  width: 4px; height: 4px;
  border-radius: 50%;
  background: rgba(255,255,255,0.4);
  animation: tdot-bounce 1.4s infinite ease-in-out both;
}
.tdot:nth-child(1) { animation-delay: -0.32s; }
.tdot:nth-child(2) { animation-delay: -0.16s; }
@keyframes tdot-bounce { 0%,80%,100% { transform: scale(0.6); opacity: 0.3; } 40% { transform: scale(1); opacity: 0.8; } }

/* ═══ 右栏：输入与控制 ═══ */
/* ─── Quick Actions ─── */
.quick-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  padding: 8px 10px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
}
.qa-btn {
  display: flex; align-items: center; gap: 4px;
  padding: 4px 9px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 5px;
  color: rgba(255, 255, 255, 0.45);
  font-size: 11px;
  cursor: pointer;
  transition: all 0.15s;
  font-family: inherit;
}
.qa-btn:hover { background: rgba(255, 255, 255, 0.07); color: rgba(255, 255, 255, 0.65); }

/* ─── Input ─── */
.input-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 8px 10px;
  min-height: 0;
}

.attach-preview {
  position: relative;
  margin-bottom: 8px;
  border-radius: 6px;
  overflow: hidden;
  max-height: 80px;
}
.attach-img {
  width: 100%;
  height: auto;
  max-height: 80px;
  object-fit: cover;
  border-radius: 6px;
}
.attach-remove {
  position: absolute;
  top: 4px; right: 4px;
  width: 22px; height: 22px;
  border-radius: 4px;
  border: none;
  background: rgba(0,0,0,0.6);
  color: white;
  cursor: pointer;
  display: flex; align-items: center; justify-content: center;
}

.input-wrapper {
  display: flex;
  gap: 6px;
  align-items: flex-end;
}

.ai-input {
  flex: 1;
  padding: 8px 10px;
  background: rgba(0, 0, 0, 0.35);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 8px;
  color: rgba(255, 255, 255, 0.8);
  font-size: 13px;
  font-family: inherit;
  outline: none;
  resize: none;
  max-height: 80px;
  line-height: 1.4;
  transition: border-color 0.15s;
}
.ai-input:focus { border-color: rgba(99, 102, 241, 0.3); }
.ai-input::placeholder { color: rgba(255, 255, 255, 0.15); }

.input-action-btn {
  width: 32px; height: 32px;
  border-radius: 6px;
  border: 1px solid rgba(255, 255, 255, 0.06);
  background: rgba(255, 255, 255, 0.03);
  color: rgba(255, 255, 255, 0.3);
  cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
  transition: all 0.15s;
}
.input-action-btn:hover { background: rgba(255, 255, 255, 0.06); color: rgba(255, 255, 255, 0.5); }

/* ─── Control Bar ─── */
.control-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 10px;
  border-top: 1px solid rgba(255, 255, 255, 0.04);
}
.voice-indicator {
  display: flex; align-items: center; gap: 5px;
  font-size: 11px;
  color: rgba(255, 255, 255, 0.25);
}
.voice-indicator.recording { color: #ef4444; }
.voice-indicator.recording .icon { animation: mic-pulse 1s infinite; }
@keyframes mic-pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.3; } }

.send-btn {
  display: flex; align-items: center; gap: 5px;
  padding: 6px 14px;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  border: none;
  border-radius: 7px;
  color: white;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s;
  font-family: inherit;
}
.send-btn:hover:not(:disabled) { box-shadow: 0 2px 12px rgba(99,102,241,0.3); }
.send-btn:disabled { opacity: 0.4; cursor: not-allowed; }

/* ─── 快捷键面板 ─── */
.shortcuts-panel {
  border-top: 1px solid rgba(255, 255, 255, 0.04);
  background: rgba(0, 0, 0, 0.1);
}
.shortcuts-panel.collapsed .sp-body { display: none; }

.sp-header {
  display: flex; align-items: center; gap: 5px;
  padding: 6px 10px;
  font-size: 10px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.2);
  cursor: pointer;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.sp-chevron { margin-left: auto; }

.sp-body {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 2px;
  padding: 2px 10px 8px;
}
.sp-item {
  display: flex; align-items: center; gap: 5px;
  font-size: 11px;
  color: rgba(255, 255, 255, 0.25);
}
.sp-item kbd {
  font-size: 10px;
  padding: 1px 5px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 3px;
  font-family: var(--font-mono, monospace);
  color: rgba(255, 255, 255, 0.35);
}

/* ═══ Transitions ═══ */
.interview-fade-enter-active { transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1); }
.interview-fade-leave-active { transition: all 0.2s ease-in; }
.interview-fade-enter-from,
.interview-fade-leave-to { opacity: 0; }
.interview-fade-enter-from .interview-container,
.interview-fade-leave-to .interview-container { transform: scale(0.95) translateY(10px); opacity: 0; }
</style>
