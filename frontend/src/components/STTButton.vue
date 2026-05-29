<template>
  <button
    class="stt-btn"
    :class="{ recording: voice.isRecording, disabled: !voice.isReady }"
    :title="voice.isRecording ? '点击停止录音' : '语音输入'"
    @click="voice.toggle"
  >
    <svg
      v-if="voice.isRecording"
      class="stt-icon"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
    >
      <rect x="6" y="4" width="4" height="16" rx="1" />
      <rect x="14" y="4" width="4" height="16" rx="1" />
    </svg>
    <svg
      v-else
      class="stt-icon"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
    >
      <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z" />
      <path d="M19 10v2a7 7 0 0 1-14 0v-2" />
      <line x1="12" y1="19" x2="12" y2="23" />
      <line x1="8" y1="23" x2="16" y2="23" />
    </svg>
    <span v-if="voice.isRecording" class="stt-label">录音中</span>
  </button>
</template>

<script setup>
import { onMounted } from 'vue'
import { useVoiceStore } from '../stores/voice'

const voice = useVoiceStore()

onMounted(() => {
  voice.checkStatus()
})
</script>

<style scoped>
.stt-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  background: var(--surface-card);
  color: var(--text-secondary);
  cursor: pointer;
  font-size: var(--text-xs);
  transition: all var(--duration-fast) ease;
}
.stt-btn:hover:not(.disabled) {
  background: var(--surface-card-hover);
  color: var(--text-primary);
}
.stt-btn.recording {
  background: var(--color-danger);
  border-color: var(--color-danger);
  color: white;
  animation: pulse 1.5s ease-in-out infinite;
}
.stt-btn.disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.stt-icon {
  width: 16px;
  height: 16px;
}
.stt-label {
  font-weight: var(--weight-semibold);
}
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}
</style>
