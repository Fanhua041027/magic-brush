<template>
  <Teleport to="body">
    <Transition name="followup-fade">
      <div v-if="isVisible" class="followup-overlay" @click.self="close">
        <div class="followup-dialog">
          <!-- Header -->
          <div class="followup-header">
            <div class="followup-title">
              <Icon name="message-circle" :size="18" />
              <span>追问</span>
            </div>
            <button class="followup-close" @click="close">
              <Icon name="x" :size="18" />
            </button>
          </div>

          <!-- Screenshot Preview -->
          <div class="followup-preview">
            <div class="preview-label">截图内容</div>
            <div class="preview-image-wrapper">
              <img v-if="screenshot" :src="screenshot" class="preview-image" alt="截图预览" />
              <div v-else class="preview-placeholder">无截图</div>
            </div>
          </div>

          <!-- Previous Answer -->
          <div v-if="previousAnswer" class="followup-context">
            <div class="context-label">之前的回答</div>
            <div class="context-content" v-html="renderMarkdown(previousAnswer)"></div>
          </div>

          <!-- Messages -->
          <div class="followup-messages" ref="messagesRef">
            <div v-for="(msg, i) in messages" :key="i" class="followup-message" :class="msg.role">
              <div class="message-avatar">
                <Icon :name="msg.role === 'user' ? 'user' : 'bot'" :size="14" />
              </div>
              <div class="message-content">
                <div class="message-text" v-html="renderMarkdown(msg.content)"></div>
              </div>
            </div>
            <div v-if="isLoading" class="followup-message assistant">
              <div class="message-avatar">
                <Icon name="bot" :size="14" />
              </div>
              <div class="message-content">
                <div class="message-loading">
                  <span class="dot"></span>
                  <span class="dot"></span>
                  <span class="dot"></span>
                </div>
              </div>
            </div>
          </div>

          <!-- Input -->
          <div class="followup-input-area">
            <div class="followup-input-wrapper">
              <textarea
                v-model="inputText"
                class="followup-input"
                placeholder="输入追问内容或按住左 Alt 语音输入..."
                @keydown.enter.exact="sendMessage"
                rows="1"
                ref="inputRef"
              ></textarea>
              <button class="followup-send" @click="sendMessage" :disabled="!inputText.trim() || isLoading">
                <Icon name="send" :size="16" />
              </button>
            </div>
            <div class="followup-hint">
              <span v-if="isRecording" class="recording-hint">
                <span class="recording-dot"></span>
                录音中...松开左 Alt 结束
              </span>
              <span v-else>Enter 发送 · 左 Alt 语音输入 · Esc 关闭</span>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, watch, nextTick, onMounted, onUnmounted } from 'vue'
import Icon from './Icon.vue'
import { renderMarkdownWithLatex } from '../utils/markdown-latex'
import { ChatWithScreenshot } from '../../wailsjs/go/app/App'

const isVisible = ref(false)
const isLoading = ref(false)
const isRecording = ref(false)
const screenshot = ref('')
const previousAnswer = ref('')
const previousContext = ref('')
const messages = ref([])
const inputText = ref('')
const messagesRef = ref(null)
const inputRef = ref(null)

let streamContent = ''

function renderMarkdown(text) {
  if (!text) return ''
  return renderMarkdownWithLatex(text)
}

function show(screenshotData, answer, context) {
  screenshot.value = screenshotData || ''
  previousAnswer.value = answer || ''
  previousContext.value = context || ''
  messages.value = []
  isVisible.value = true
  nextTick(() => {
    inputRef.value?.focus()
  })
}

function close() {
  isVisible.value = false
  messages.value = []
  streamContent = ''
}

async function sendMessage() {
  const text = inputText.value.trim()
  if (!text || isLoading.value) return

  inputText.value = ''
  messages.value.push({
    role: 'user',
    content: text,
  })

  isLoading.value = true
  streamContent = ''

  try {
    await ChatWithScreenshot(text, screenshot.value, previousContext.value)
  } catch (error) {
    console.error('Follow-up error:', error)
    messages.value.push({
      role: 'assistant',
      content: `抱歉，发生了错误: ${error.message || '未知错误'}`,
    })
  } finally {
    isLoading.value = false
  }
}

function handleStreamChunk(chunk) {
  if (!isLoading.value) return

  streamContent += chunk

  const lastMsg = messages.value[messages.value.length - 1]
  if (lastMsg && lastMsg.role === 'assistant' && lastMsg._streaming) {
    lastMsg.content = streamContent
  } else {
    messages.value.push({
      role: 'assistant',
      content: streamContent,
      _streaming: true,
    })
  }

  nextTick(() => {
    if (messagesRef.value) {
      messagesRef.value.scrollTop = messagesRef.value.scrollHeight
    }
  })
}

function handleStreamDone() {
  const lastMsg = messages.value[messages.value.length - 1]
  if (lastMsg && lastMsg._streaming) {
    delete lastMsg._streaming
  }
  isLoading.value = false
  streamContent = ''
}

function handleStreamError(error) {
  const lastMsg = messages.value[messages.value.length - 1]
  if (lastMsg && lastMsg._streaming) {
    lastMsg.content += `\n\n[错误: ${error}]`
    delete lastMsg._streaming
  } else {
    messages.value.push({
      role: 'assistant',
      content: `抱歉，发生了错误: ${error}`,
    })
  }
  isLoading.value = false
  streamContent = ''
}

function handleKeydown(e) {
  if (e.key === 'Escape' && isVisible.value) {
    close()
  }
}

onMounted(() => {
  window.addEventListener('chat-stream-chunk', (e) => {
    if (isVisible.value) {
      handleStreamChunk(e.detail)
    }
  })
  window.addEventListener('chat-stream-done', () => {
    if (isVisible.value) {
      handleStreamDone()
    }
  })
  window.addEventListener('chat-stream-error', (e) => {
    if (isVisible.value) {
      handleStreamError(e.detail)
    }
  })
  window.addEventListener('stt-recording-started', () => {
    isRecording.value = true
  })
  window.addEventListener('stt-recording-stopped', () => {
    isRecording.value = false
  })
  window.addEventListener('stt-transcribed', (e) => {
    if (isVisible.value && e.detail) {
      inputText.value = e.detail
      nextTick(() => {
        sendMessage()
      })
    }
  })
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
})

defineExpose({ show, close })
</script>

<style scoped>
.followup-overlay {
  position: fixed;
  inset: 0;
  z-index: 1001;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(4px);
}

.followup-dialog {
  width: 520px;
  max-width: 90vw;
  height: 700px;
  max-height: 85vh;
  background: var(--surface-elevated);
  border-radius: var(--radius-xl);
  border: 1px solid var(--border-default);
  box-shadow: var(--shadow-xl);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.followup-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 18px;
  border-bottom: 1px solid var(--border-default);
  background: var(--surface-card);
}

.followup-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  font-size: 14px;
  color: var(--text-primary);
}

.followup-close {
  width: 28px;
  height: 28px;
  border-radius: var(--radius-md);
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.followup-close:hover {
  background: var(--surface-card-hover);
  color: var(--text-primary);
}

.followup-preview {
  padding: 12px 18px;
  border-bottom: 1px solid var(--border-default);
  background: var(--surface-card);
}

.preview-label {
  font-size: 11px;
  color: var(--text-muted);
  margin-bottom: 8px;
  font-weight: 500;
}

.preview-image-wrapper {
  max-height: 120px;
  overflow: hidden;
  border-radius: var(--radius-md);
  border: 1px solid var(--border-default);
}

.preview-image {
  width: 100%;
  height: auto;
  max-height: 120px;
  object-fit: cover;
}

.preview-placeholder {
  padding: 20px;
  text-align: center;
  color: var(--text-muted);
  font-size: 12px;
}

.followup-context {
  padding: 12px 18px;
  border-bottom: 1px solid var(--border-default);
  background: var(--surface-card);
  max-height: 100px;
  overflow-y: auto;
}

.context-label {
  font-size: 11px;
  color: var(--text-muted);
  margin-bottom: 6px;
  font-weight: 500;
}

.context-content {
  font-size: 12px;
  color: var(--text-secondary);
  line-height: 1.5;
}

.context-content :deep(p) {
  margin: 0 0 4px 0;
}

.followup-messages {
  flex: 1;
  overflow-y: auto;
  padding: 16px 18px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.followup-message {
  display: flex;
  gap: 8px;
  max-width: 90%;
}

.followup-message.user {
  align-self: flex-end;
  flex-direction: row-reverse;
}

.followup-message.assistant {
  align-self: flex-start;
}

.message-avatar {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.followup-message.user .message-avatar {
  background: linear-gradient(135deg, #6366f1, #4f46e5);
  color: white;
}

.followup-message.assistant .message-avatar {
  background: linear-gradient(135deg, #10b981, #059669);
  color: white;
}

.message-content {
  display: flex;
  flex-direction: column;
}

.message-text {
  padding: 8px 12px;
  border-radius: var(--radius-lg);
  font-size: 13px;
  line-height: 1.5;
  word-break: break-word;
}

.followup-message.user .message-text {
  background: var(--accent);
  color: white;
  border-bottom-right-radius: 4px;
}

.followup-message.assistant .message-text {
  background: var(--surface-card);
  color: var(--text-primary);
  border: 1px solid var(--border-default);
  border-bottom-left-radius: 4px;
}

.message-text :deep(p) {
  margin: 0 0 6px 0;
}

.message-text :deep(p:last-child) {
  margin-bottom: 0;
}

.message-loading {
  display: flex;
  gap: 4px;
  padding: 10px 14px;
  background: var(--surface-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-default);
  border-bottom-left-radius: 4px;
}

.dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--text-muted);
  animation: dot-pulse 1.4s infinite ease-in-out both;
}

.dot:nth-child(1) { animation-delay: -0.32s; }
.dot:nth-child(2) { animation-delay: -0.16s; }

@keyframes dot-pulse {
  0%, 80%, 100% { transform: scale(0.6); opacity: 0.5; }
  40% { transform: scale(1); opacity: 1; }
}

.followup-input-area {
  padding: 14px 18px;
  border-top: 1px solid var(--border-default);
  background: var(--surface-card);
}

.followup-input-wrapper {
  display: flex;
  gap: 8px;
  align-items: flex-end;
}

.followup-input {
  flex: 1;
  padding: 8px 12px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-default);
  background: var(--surface-elevated);
  color: var(--text-primary);
  font-size: 13px;
  line-height: 1.4;
  resize: none;
  outline: none;
  transition: border-color 0.2s;
  max-height: 80px;
  font-family: inherit;
}

.followup-input:focus {
  border-color: var(--accent);
}

.followup-input::placeholder {
  color: var(--text-muted);
}

.followup-send {
  width: 32px;
  height: 32px;
  border-radius: var(--radius-md);
  border: none;
  background: var(--accent);
  color: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
  flex-shrink: 0;
}

.followup-send:hover:not(:disabled) {
  background: var(--accent-hover);
}

.followup-send:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.followup-hint {
  font-size: 11px;
  color: var(--text-muted);
  margin-top: 6px;
  text-align: center;
}

.recording-hint {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  color: #ef4444;
  font-weight: 500;
}

.recording-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: #ef4444;
  animation: recording-pulse 1s infinite;
}

@keyframes recording-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

/* Transitions */
.followup-fade-enter-active {
  transition: all 0.3s var(--ease-out);
}

.followup-fade-leave-active {
  transition: all 0.2s ease-in;
}

.followup-fade-enter-from,
.followup-fade-leave-to {
  opacity: 0;
}

.followup-fade-enter-from .followup-dialog,
.followup-fade-leave-to .followup-dialog {
  transform: scale(0.95) translateY(10px);
}
</style>
