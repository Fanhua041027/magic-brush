import { defineStore } from 'pinia'
import { ref, reactive, watch } from 'vue'
import { ChatWithDeepSeek, ChatWithDeepSeekStream } from '../../wailsjs/go/app/App'

// 本地存储键名
const STORAGE_KEY = 'magic-brush-chat-history'
const MAX_HISTORY = 50 // 最大历史记录数

export const useChatStore = defineStore('chat', () => {
  const isVisible = ref(false)
  const isLoading = ref(false)
  const messages = ref([])
  const currentStreamContent = ref('')

  // 从本地存储加载历史
  function loadHistory() {
    try {
      const stored = localStorage.getItem(STORAGE_KEY)
      if (stored) {
        const parsed = JSON.parse(stored)
        if (Array.isArray(parsed)) {
          messages.value = parsed.slice(-MAX_HISTORY)
        }
      }
    } catch (e) {
      console.error('Failed to load chat history:', e)
    }
  }

  // 保存到本地存储
  function saveHistory() {
    try {
      // 只保存最近的消息，避免存储过大
      const toSave = messages.value.slice(-MAX_HISTORY)
      localStorage.setItem(STORAGE_KEY, JSON.stringify(toSave))
    } catch (e) {
      console.error('Failed to save chat history:', e)
    }
  }

  // 清除历史
  function clearHistory() {
    messages.value = []
    localStorage.removeItem(STORAGE_KEY)
  }

  // 初始化时加载历史
  loadHistory()

  // 监听消息变化，自动保存
  watch(messages, () => {
    saveHistory()
  }, { deep: true })

  function show() {
    isVisible.value = true
  }

  function hide() {
    isVisible.value = false
  }

  function toggle() {
    isVisible.value = !isVisible.value
  }

  function clearMessages() {
    messages.value = []
  }

  function addMessage(role, content) {
    messages.value.push({
      role,
      content,
      time: new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
      timestamp: Date.now(),
    })
  }

  async function sendMessage(text) {
    if (!text.trim() || isLoading.value) return

    // 添加用户消息
    addMessage('user', text)
    isLoading.value = true
    currentStreamContent.value = ''

    try {
      // 使用流式输出
      await ChatWithDeepSeekStream(text)

      // 流式输出完成后，内容已经通过事件处理
      // 如果没有收到流式事件，使用普通模式作为备选
      if (currentStreamContent.value === '') {
        const result = await ChatWithDeepSeek(text)
        if (result && !currentStreamContent.value) {
          addMessage('assistant', result)
        }
      }
    } catch (error) {
      console.error('Chat error:', error)
      addMessage('assistant', `抱歉，发生了错误: ${error.message || '未知错误'}`)
    } finally {
      isLoading.value = false
      currentStreamContent.value = ''
    }
  }

  function handleStreamChunk(chunk) {
    if (!isLoading.value) return

    currentStreamContent.value += chunk

    // 更新或添加助手消息
    const lastMsg = messages.value[messages.value.length - 1]
    if (lastMsg && lastMsg.role === 'assistant' && lastMsg._streaming) {
      lastMsg.content = currentStreamContent.value
    } else {
      messages.value.push({
        role: 'assistant',
        content: currentStreamContent.value,
        time: new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
        timestamp: Date.now(),
        _streaming: true,
      })
    }
  }

  function handleStreamDone() {
    const lastMsg = messages.value[messages.value.length - 1]
    if (lastMsg && lastMsg._streaming) {
      delete lastMsg._streaming
    }
    isLoading.value = false
    currentStreamContent.value = ''
  }

  function handleStreamError(error) {
    console.error('Stream error:', error)
    const lastMsg = messages.value[messages.value.length - 1]
    if (lastMsg && lastMsg._streaming) {
      lastMsg.content += `\n\n[错误: ${error}]`
      delete lastMsg._streaming
    } else {
      addMessage('assistant', `抱歉，发生了错误: ${error}`)
    }
    isLoading.value = false
    currentStreamContent.value = ''
  }

  // 导出对话历史
  function exportHistory() {
    const data = {
      exportTime: new Date().toISOString(),
      messages: messages.value.map(m => ({
        role: m.role,
        content: m.content,
        time: m.time,
      })),
    }
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `chat-history-${new Date().toISOString().slice(0, 10)}.json`
    a.click()
    URL.revokeObjectURL(url)
  }

  // 导入对话历史
  function importHistory(file) {
    return new Promise((resolve, reject) => {
      const reader = new FileReader()
      reader.onload = (e) => {
        try {
          const data = JSON.parse(e.target.result)
          if (data.messages && Array.isArray(data.messages)) {
            messages.value = data.messages
            saveHistory()
            resolve(data.messages.length)
          } else {
            reject(new Error('Invalid file format'))
          }
        } catch (err) {
          reject(err)
        }
      }
      reader.onerror = () => reject(new Error('Failed to read file'))
      reader.readAsText(file)
    })
  }

  return {
    isVisible,
    isLoading,
    messages,
    show,
    hide,
    toggle,
    clearMessages,
    clearHistory,
    addMessage,
    sendMessage,
    handleStreamChunk,
    handleStreamDone,
    handleStreamError,
    exportHistory,
    importHistory,
  }
})
