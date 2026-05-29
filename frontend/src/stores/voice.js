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

  async function start() {
    if (isRecording.value || isTranscribing.value) return
    transcribedText.value = ''
    try {
      const result = await api.sttStart()
      if (result.status === 'recording') {
        isRecording.value = true
      }
    } catch (e) {
      console.error('STT start error:', e)
    }
  }

  async function stop() {
    if (!isRecording.value || isTranscribing.value) return
    isTranscribing.value = true
    try {
      const result = await api.sttStop()
      isRecording.value = false
      isTranscribing.value = false
      if (result.status === 'transcribed') {
        transcribedText.value = result.text || ''
        if (result.text) {
          api.setUserMessage(result.text)
        }
      }
    } catch (e) {
      isRecording.value = false
      isTranscribing.value = false
      console.error('STT stop error:', e)
    }
  }

  return { isRecording, isTranscribing, isReady, transcribedText, checkStatus, toggle, start, stop }
})
