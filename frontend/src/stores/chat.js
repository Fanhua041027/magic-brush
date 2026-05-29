import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'
import { ChatWithDeepSeek, ChatWithDeepSeekStream } from '../../wailsjs/go/app/App'

export const useChatStore = defineStore('chat', () => {
  const isVisible = ref(false)
  const isLoading = ref(false)
  const messages = ref([])
  const currentStreamContent = ref('')

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

  return {
    isVisible,
    isLoading,
    messages,
    show,
    hide,
    toggle,
    clearMessages,
    addMessage,
    sendMessage,
    handleStreamChunk,
    handleStreamDone,
    handleStreamError,
  }
})
