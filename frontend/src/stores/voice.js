import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../services/api'

export const useVoiceStore = defineStore('voice', () => {
  const isRecording = ref(false)
  const isTranscribing = ref(false)
  const isReady = ref(false)
  const transcribedText = ref('')

  async function checkStatus() {
    try {
      const status = await api.getSTTStatus()
      isRecording.value = status.recording || false
      isReady.value = status.ready || false
    } catch (e) {
      isReady.value = false
    }
  }

  async function toggle() {
    if (isTranscribing.value) return
    try {
      const result = await api.toggleSTT()
      if (result.status === 'recording') {
        isRecording.value = true
        transcribedText.value = ''
      } else if (result.status === 'transcribed') {
        isRecording.value = false
        transcribedText.value = result.text || ''
        // Auto-set as pending user message for next solve
        if (result.text) {
          api.setUserMessage(result.text)
        }
      } else if (result.error) {
        isRecording.value = false
        console.error('STT error:', result.error)
      }
    } catch (e) {
      isRecording.value = false
      console.error('STT toggle error:', e)
    }
  }

  return { isRecording, isTranscribing, isReady, transcribedText, checkStatus, toggle }
})
