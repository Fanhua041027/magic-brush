<template>
  <div class="listen-overlay" @click.self="$emit('close')">
    <div class="listen-container">
      <!-- Header -->
      <div class="listen-header">
        <h2>🎧 面试聆听助手</h2>
        <button class="close-btn" @click="$emit('close')">
          <Icon name="x" :size="18" />
        </button>
      </div>

      <div class="listen-body">
        <!-- Status Area -->
        <div class="status-area">
          <div class="status-row">
            <button
              class="listen-btn"
              :class="{ listening: isListening, disabled: isProcessing }"
              @click="toggleListen"
              :disabled="isProcessing"
            >
              <span class="btn-icon">{{ isListening ? '⏹' : '🎤' }}</span>
              <span class="btn-text">{{ isListening ? '停止聆听' : isProcessing ? '处理中...' : '开始聆听' }}</span>
            </button>
            <div class="status-info">
              <div class="status-dot-group">
                <span class="s-dot" :class="statusClass"></span>
                <span class="s-label">{{ statusText }}</span>
              </div>
            </div>
          </div>
          <div class="device-info" v-if="!isListening && !transcribedText && !answerText">
            系统将捕获电脑内部声音（面试官的提问），自动转录并生成回答
          </div>
        </div>

        <!-- Pipeline Events -->
        <div v-if="pipelineEvents.length > 0" class="pipeline-bar">
          <div v-for="(evt, i) in pipelineEvents" :key="i" class="p-ev" :class="evt.status">
            <span class="pe-stage">{{ evt.stage }}</span>
            <span class="pe-status">{{ evt.status === 'done' ? '✓' : evt.status === 'running' ? '⟳' : '✗' }}</span>
            <span class="pe-detail">{{ evt.detail }}</span>
          </div>
        </div>

        <!-- Transcribed Question -->
        <div v-if="transcribedText" class="result-section">
          <div class="result-label">📝 识别到的问题</div>
          <div class="result-box question-box">{{ transcribedText }}</div>
        </div>

        <!-- AI Answer -->
        <div v-if="answerText" class="result-section">
          <div class="result-label">🤖 建议回答</div>
          <div class="result-box answer-box">{{ answerText }}</div>
          <div class="result-actions">
            <button class="btn-sm" @click="copyAnswer">📋 复制回答</button>
            <button class="btn-sm btn-outline" @click="reset">🔄 重新聆听</button>
          </div>
        </div>

        <!-- Initial state -->
        <div v-if="!transcribedText && !answerText && !isListening" class="empty-state">
          <div class="empty-icon">🎯</div>
          <p>点击「开始聆听」按钮，助手会监听系统声音</p>
          <p class="empty-sub">面试官提问后，再次点击停止，即可获得回答建议</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onUnmounted } from 'vue'
import Icon from './Icon.vue'
import { api } from '../services/api'
import { useAgentStore } from '../stores/agents'
import { on } from '../services/events'

const emit = defineEmits(['close'])
const agents = useAgentStore()

// State
const isListening = ref(false)
const isProcessing = ref(false)
const transcribedText = ref('')
const answerText = ref('')
const pipelineEvents = ref([])
const listenTimer = ref(null)
const audioDeviceId = ref(25) // Stereo Mix default

const statusText = computed(() => {
  if (isProcessing.value) return '正在处理...'
  if (isListening.value) return '正在聆听系统声音...'
  if (answerText.value) return '回答已生成'
  if (transcribedText.value) return '问题已识别'
  return '就绪'
})

const statusClass = computed(() => {
  if (isProcessing.value) return 'processing'
  if (isListening.value) return 'listening'
  if (answerText.value) return 'done'
  return 'idle'
})

// Listen for pipeline events
on('pipeline-event', (eventStr) => {
  const parts = eventStr.split('|')
  if (parts.length >= 2) {
    pipelineEvents.value.unshift({
      stage: parts[0],
      status: parts[1],
      detail: parts.slice(2).join('|'),
    })
    if (pipelineEvents.value.length > 10) {
      pipelineEvents.value = pipelineEvents.value.slice(0, 10)
    }
  }
})

async function toggleListen() {
  if (isListening.value) {
    // Stop listening and process
    isListening.value = false
    isProcessing.value = true
    pipelineEvents.value = []

    try {
      // Find available audio device
      let deviceId = audioDeviceId.value
      try {
        const avail = await api.audioIsAvailable()
        if (avail) {
          const parsed = JSON.parse(avail)
          if (parsed.available && parsed.deviceId >= 0) {
            deviceId = parsed.deviceId
          }
        }
      } catch (_) {}

      // Capture system audio (8 seconds)
      const captureResult = await api.audioCapture(deviceId, 8)
      if (!captureResult) throw new Error('音频捕获失败')

      const capture = JSON.parse(captureResult)
      if (capture.error) throw new Error(capture.error)
      if (!capture.base64) throw new Error('未捕获到音频数据')

      // Transcribe
      const transcribeResult = await api.audioTranscribe(capture.base64)
      if (!transcribeResult) throw new Error('转录失败')

      const transcribe = JSON.parse(transcribeResult)
      if (transcribe.error) throw new Error(transcribe.error)
      if (!transcribe.text || transcribe.text.trim().length < 3) {
        throw new Error('未检测到语音内容，请重试')
      }

      transcribedText.value = transcribe.text

      // Generate answer directly via Interview Agent (non-streaming)
      const answerResult = await api.generateInterviewAnswer(transcribe.text)
      if (answerResult) {
        const parsed = JSON.parse(answerResult)
        if (parsed.error) throw new Error(parsed.error)
        if (parsed.answer) {
          answerText.value = parsed.answer
        }
      }

    } catch (e) {
      transcribedText.value = ''
      answerText.value = ''
      const errMsg = e.message || '处理失败'
      pipelineEvents.value.unshift({
        stage: 'error',
        status: 'error',
        detail: errMsg,
      })
    } finally {
      isProcessing.value = false
    }
  } else {
    // Start listening
    pipelineEvents.value = []
    transcribedText.value = ''
    answerText.value = ''
    isListening.value = true

    // Auto-stop after 30 seconds
    listenTimer.value = setTimeout(() => {
      if (isListening.value) toggleListen()
    }, 30000)
  }
}



function copyAnswer() {
  if (answerText.value) {
    navigator.clipboard.writeText(answerText.value)
    // Toast
    const event = new CustomEvent('app-toast', {
      detail: { text: '回答已复制', type: 'success' }
    })
    window.dispatchEvent(event)
  }
}

function reset() {
  transcribedText.value = ''
  answerText.value = ''
  pipelineEvents.value = []
  isListening.value = false
  isProcessing.value = false
}

onUnmounted(() => {
  if (listenTimer.value) clearTimeout(listenTimer.value)
})
</script>

<style scoped>
.listen-overlay {
  position: fixed;
  inset: 0;
  z-index: 5000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(8px);
  pointer-events: auto;
}
.listen-container {
  width: 520px;
  max-width: 90vw;
  max-height: 85vh;
  background: var(--surface-popover);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-xl);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  animation: slideIn 0.25s var(--ease-out);
}
@keyframes slideIn {
  from { opacity: 0; transform: translateY(20px) scale(0.96); }
  to { opacity: 1; transform: translateY(0) scale(1); }
}
.listen-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--sp-4) var(--sp-5);
  border-bottom: 1px solid var(--border-subtle);
}
.listen-header h2 { margin: 0; font-size: var(--text-lg); color: var(--text-primary); }
.close-btn {
  width: 32px; height: 32px;
  display: flex; align-items: center; justify-content: center;
  background: transparent; border: none;
  border-radius: var(--radius-sm);
  color: var(--text-muted);
  cursor: pointer;
}
.close-btn:hover { background: var(--surface-card-hover); color: var(--text-primary); }
.listen-body {
  flex: 1;
  padding: var(--sp-5);
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: var(--sp-4);
}

/* Status Area */
.status-area { flex-shrink: 0; }
.status-row {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
}
.listen-btn {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  padding: 10px 24px;
  border-radius: var(--radius-md);
  border: none;
  font-size: var(--text-sm);
  font-weight: 700;
  cursor: pointer;
  transition: all var(--duration-fast) ease;
  background: var(--accent);
  color: white;
  white-space: nowrap;
}
.listen-btn:hover { background: var(--accent-hover); transform: translateY(-1px); }
.listen-btn.listening {
  background: var(--color-error);
  animation: btnPulse 1.5s ease-in-out infinite;
}
.listen-btn.disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none;
}
.listen-btn:disabled { opacity: 0.6; cursor: not-allowed; transform: none; }
.btn-icon { font-size: 18px; }
@keyframes btnPulse { 0%,100% { box-shadow: 0 0 0 0 rgba(239,68,68,0.4); } 50% { box-shadow: 0 0 0 8px rgba(239,68,68,0); } }
.status-info { flex: 1; }
.status-dot-group { display: flex; align-items: center; gap: var(--sp-2); }
.s-dot {
  width: 8px; height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.s-dot.idle { background: var(--text-muted); }
.s-dot.listening { background: var(--color-error); animation: dotPulse 1s ease-in-out infinite; }
.s-dot.processing { background: var(--accent); animation: dotPulse 0.5s ease-in-out infinite; }
.s-dot.done { background: var(--color-success); }
@keyframes dotPulse { 0%,100% { opacity: 1; } 50% { opacity: 0.3; } }
.s-label { font-size: var(--text-sm); color: var(--text-secondary); font-weight: 600; }
.device-info {
  margin-top: var(--sp-2);
  padding: var(--sp-2) var(--sp-3);
  background: var(--surface-card);
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
  color: var(--text-muted);
  line-height: 1.5;
}

/* Pipeline Events */
.pipeline-bar {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: var(--sp-2);
  background: var(--surface-card);
  border-radius: var(--radius-sm);
  max-height: 100px;
  overflow-y: auto;
}
.p-ev {
  display: flex;
  align-items: center;
  gap: var(--sp-1);
  font-size: 10px;
  font-family: var(--font-mono);
  padding: 2px 4px;
}
.p-ev.done { color: var(--color-success); }
.p-ev.running { color: var(--accent); }
.p-ev.error { color: var(--color-error); }
.pe-stage { font-weight: 700; min-width: 45px; text-transform: uppercase; }
.pe-status { font-size: 11px; }
.pe-detail { color: var(--text-muted); }

/* Results */
.result-section { flex-shrink: 0; }
.result-label {
  font-size: var(--text-xs);
  font-weight: 700;
  color: var(--text-secondary);
  margin-bottom: var(--sp-1);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.result-box {
  padding: var(--sp-3);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  line-height: 1.6;
  border: 1px solid;
}
.question-box {
  background: var(--surface-card);
  border-color: var(--border-default);
  color: var(--text-primary);
}
.answer-box {
  background: var(--success-bg);
  border-color: var(--success-border);
  color: var(--text-primary);
}
.result-actions {
  display: flex;
  gap: var(--sp-2);
  margin-top: var(--sp-2);
}
.btn-sm {
  padding: 6px 16px;
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
  font-weight: 600;
  border: none;
  cursor: pointer;
  transition: all var(--duration-fast) ease;
  background: var(--accent);
  color: white;
}
.btn-sm:hover { background: var(--accent-hover); }
.btn-outline {
  background: transparent;
  color: var(--accent);
  border: 1px solid var(--accent-border);
}
.btn-outline:hover { background: var(--accent-muted); }

/* Empty state */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--sp-2);
  padding: var(--sp-8) 0;
  text-align: center;
}
.empty-icon { font-size: 48px; }
.empty-state p { color: var(--text-secondary); font-size: var(--text-sm); margin: 0; }
.empty-sub { color: var(--text-muted) !important; font-size: var(--text-xs) !important; }
</style>
