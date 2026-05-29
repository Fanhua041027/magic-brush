import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'
import { api } from '../services/api'

export const useUIStore = defineStore('ui', () => {
  const toasts = ref([])

  function showToast(text, type = 'error', duration = 2000) {
    const id = Date.now() + Math.random()
    const toast = reactive({ id, text, type, show: false })
    toasts.value.push(toast)
    setTimeout(() => { toast.show = true }, 50)
    setTimeout(() => {
      toast.show = false
      setTimeout(() => {
        const idx = toasts.value.findIndex(t => t.id === id)
        if (idx !== -1) toasts.value.splice(idx, 1)
      }, 400)
    }, duration)
  }

  const activeButtons = reactive({ toggle: false, solve: false, clickthrough: false })
  function flash(which) {
    activeButtons[which] = true
    setTimeout(() => { activeButtons[which] = false }, 200)
  }

  const isClickThrough = ref(false)
  const mainVisible = ref(true)
  const isStealthMode = ref(true)
  const hasStarted = ref(false)

  const showSettings = ref(false)
  const activeTab = ref('api')
  const availableModels = ref([])
  const isLoadingModels = ref(false)
  const isTestingConnection = ref(false)
  const connectionStatus = ref(null)

  const initStatus = ref('initializing')
  const showResumeWarning = ref(false)

  function quit() {
    api.quit()
  }

  return {
    toasts, showToast,
    activeButtons, flash,
    isClickThrough, mainVisible, isStealthMode, hasStarted,
    showSettings, activeTab, availableModels, isLoadingModels,
    isTestingConnection, connectionStatus,
    initStatus, showResumeWarning,
    quit,
  }
})
