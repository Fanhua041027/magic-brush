<template>
  <Teleport to="body">
    <Transition name="chat-fade">
      <div v-if="chatStore.isVisible" class="chat-overlay" @click.self="close">
        <div class="chat-dialog">
          <!-- Header -->
          <div class="chat-header">
            <div class="chat-title">
              <Icon name="message-square" :size="18" />
              <span>AI 对话</span>
              <span v-if="settingsStore.settings.resumeContent" class="resume-badge">
                <Icon name="file-text" :size="12" />
                已加载简历
              </span>
            </div>
            <button class="chat-close" @click="close">
              <Icon name="x" :size="18" />
            </button>
          </div>

          <!-- Messages -->
          <div class="chat-messages" ref="messagesRef">
            <div v-if="chatStore.messages.length === 0" class="chat-empty">
              <Icon name="mic" :size="32" class="chat-empty-icon" />
              <p>语音输入已就绪，按住左 Alt 键开始说话</p>
            </div>
            <div v-for="(msg, i) in chatStore.messages" :key="i" class="chat-message" :class="msg.role">
              <div class="message-avatar">
                <Icon :name="msg.role === 'user' ? 'user' : 'bot'" :size="16" />
              </div>
              <div class="message-content">
                <div class="message-text" v-html="renderMarkdown(msg.content)"></div>
                <div class="message-time">{{ msg.time }}</div>
              </div>
            </div>
            <div v-if="chatStore.isLoading" class="chat-message assistant">
              <div class="message-avatar">
                <Icon name="bot" :size="16" />
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
          <div class="chat-input-area">
            <div class="chat-input-wrapper">
              <textarea
                v-model="inputText"
                class="chat-input"
                placeholder="输入消息或按住左 Alt 语音输入..."
                @keydown.enter.exact="sendMessage"
                rows="1"
                ref="inputRef"
              ></textarea>
              <button class="chat-send" @click="sendMessage" :disabled="!inputText.trim() || chatStore.isLoading">
                <Icon name="send" :size="18" />
              </button>
            </div>
            <div class="chat-hint">
              <span v-if="voiceStore.isRecording" class="recording-hint">
                <span class="recording-dot"></span>
                录音中...松开左 Alt 结束
              </span>
              <span v-else>Enter 发送 · 左 Alt 语音输入</span>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, watch, nextTick, onMounted } from 'vue'
import Icon from './Icon.vue'
import { useChatStore } from '../stores/chat'
import { useVoiceStore } from '../stores/voice'
import { useSettingsStore } from '../stores/settings'
import { renderMarkdownWithLatex } from '../utils/markdown-latex'

const chatStore = useChatStore()
const voiceStore = useVoiceStore()
const settingsStore = useSettingsStore()

const inputText = ref('')
const messagesRef = ref(null)
const inputRef = ref(null)

function renderMarkdown(text) {
  if (!text) return ''
  return renderMarkdownWithLatex(text)
}

function close() {
  chatStore.hide()
}

async function sendMessage() {
  const text = inputText.value.trim()
  if (!text || chatStore.isLoading) return
  inputText.value = ''
  await chatStore.sendMessage(text)
}

// 监听语音转写结果
watch(() => voiceStore.transcribedText, (newText) => {
  if (newText && chatStore.isVisible) {
    inputText.value = newText
    // 自动发送语音转写的结果
    nextTick(() => {
      sendMessage()
    })
  }
})

// 监听消息列表变化，自动滚动到底部
watch(() => chatStore.messages.length, () => {
  nextTick(() => {
    if (messagesRef.value) {
      messagesRef.value.scrollTop = messagesRef.value.scrollHeight
    }
  })
})

watch(() => chatStore.isVisible, (visible) => {
  if (visible) {
    nextTick(() => {
      inputRef.value?.focus()
    })
  }
})

onMounted(() => {
  // 监听流式输出
  window.addEventListener('chat-stream-chunk', (e) => {
    chatStore.handleStreamChunk(e.detail)
  })
  window.addEventListener('chat-stream-done', () => {
    chatStore.handleStreamDone()
  })
  window.addEventListener('chat-stream-error', (e) => {
    chatStore.handleStreamError(e.detail)
  })
})
</script>

<style scoped>
.chat-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(4px);
}

.chat-dialog {
  width: 480px;
  max-width: 90vw;
  height: 600px;
  max-height: 80vh;
  background: var(--surface-elevated);
  border-radius: var(--radius-xl);
  border: 1px solid var(--border-default);
  box-shadow: var(--shadow-xl);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.chat-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-default);
  background: var(--surface-card);
}

.chat-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  font-size: 15px;
  color: var(--text-primary);
}

.resume-badge {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  background: var(--accent-muted);
  border: 1px solid var(--accent-border);
  border-radius: var(--radius-full);
  font-size: 11px;
  font-weight: 500;
  color: var(--accent);
}

.chat-close {
  width: 32px;
  height: 32px;
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

.chat-close:hover {
  background: var(--surface-card-hover);
  color: var(--text-primary);
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.chat-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
  gap: 12px;
}

.chat-empty-icon {
  opacity: 0.5;
}

.chat-empty p {
  font-size: 13px;
  margin: 0;
}

.chat-message {
  display: flex;
  gap: 10px;
  max-width: 85%;
}

.chat-message.user {
  align-self: flex-end;
  flex-direction: row-reverse;
}

.chat-message.assistant {
  align-self: flex-start;
}

.message-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.chat-message.user .message-avatar {
  background: linear-gradient(135deg, #6366f1, #4f46e5);
  color: white;
}

.chat-message.assistant .message-avatar {
  background: linear-gradient(135deg, #10b981, #059669);
  color: white;
}

.message-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.message-text {
  padding: 10px 14px;
  border-radius: var(--radius-lg);
  font-size: 13px;
  line-height: 1.6;
  word-break: break-word;
}

.chat-message.user .message-text {
  background: var(--accent);
  color: white;
  border-bottom-right-radius: 4px;
}

.chat-message.assistant .message-text {
  background: var(--surface-card);
  color: var(--text-primary);
  border: 1px solid var(--border-default);
  border-bottom-left-radius: 4px;
}

.message-text :deep(p) {
  margin: 0 0 8px 0;
}

.message-text :deep(p:last-child) {
  margin-bottom: 0;
}

.message-text :deep(code) {
  background: rgba(0, 0, 0, 0.1);
  padding: 2px 4px;
  border-radius: 3px;
  font-size: 12px;
}

.message-text :deep(pre) {
  background: rgba(0, 0, 0, 0.1);
  padding: 8px;
  border-radius: 6px;
  overflow-x: auto;
  margin: 8px 0;
}

.message-time {
  font-size: 11px;
  color: var(--text-muted);
  padding: 0 4px;
}

.chat-message.user .message-time {
  text-align: right;
}

.message-loading {
  display: flex;
  gap: 4px;
  padding: 12px 16px;
  background: var(--surface-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-default);
  border-bottom-left-radius: 4px;
}

.dot {
  width: 6px;
  height: 6px;
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

.chat-input-area {
  padding: 16px 20px;
  border-top: 1px solid var(--border-default);
  background: var(--surface-card);
}

.chat-input-wrapper {
  display: flex;
  gap: 8px;
  align-items: flex-end;
}

.chat-input {
  flex: 1;
  padding: 10px 14px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-default);
  background: var(--surface-elevated);
  color: var(--text-primary);
  font-size: 13px;
  line-height: 1.5;
  resize: none;
  outline: none;
  transition: border-color 0.2s;
  max-height: 100px;
  font-family: inherit;
}

.chat-input:focus {
  border-color: var(--accent);
}

.chat-input::placeholder {
  color: var(--text-muted);
}

.chat-send {
  width: 36px;
  height: 36px;
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

.chat-send:hover:not(:disabled) {
  background: var(--accent-hover);
}

.chat-send:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.chat-hint {
  font-size: 11px;
  color: var(--text-muted);
  margin-top: 8px;
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
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #ef4444;
  animation: recording-pulse 1s infinite;
}

@keyframes recording-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

/* Transitions */
.chat-fade-enter-active {
  transition: all 0.3s var(--ease-out);
}

.chat-fade-leave-active {
  transition: all 0.2s ease-in;
}

.chat-fade-enter-from,
.chat-fade-leave-to {
  opacity: 0;
}

.chat-fade-enter-from .chat-dialog,
.chat-fade-leave-to .chat-dialog {
  transform: scale(0.95) translateY(10px);
}
</style>
