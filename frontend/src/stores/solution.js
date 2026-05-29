/**
 * Solution Store — 解题结果、流式输出、历史记录
 */
import { defineStore } from 'pinia'
import { ref, reactive, computed } from 'vue'
import { renderMarkdownWithLatex } from '../utils/markdown-latex'
import { api } from '../services/api'

export const useSolutionStore = defineStore('solution', () => {
  const history = ref([])
  const activeHistoryIndex = ref(0)
  const isLoading = ref(false)
  const isAppending = ref(false)
  const shouldOverwriteHistory = ref(false)

  // Thinking chain
  const isThinking = ref(false)
  const thinkingStatusText = ref('Thinking Process')
  const thinkingExpanded = ref(true)
  const isThinkingStalled = ref(false)

  // Stream state (not reactive — internal)
  let streamBuffer = ''
  let thinkingBuffer = ''
  let thinkingStartTime = 0
  let stallTimer = null
  let pendingUserScreenshot = ''

  // 流式渲染：节流 + 全量渲染
  let renderRafId = null
  let renderDirty = false
  const streamingHtml = ref('')

  // Error state
  const errorState = reactive({
    show: false,
    icon: '⚠️',
    title: '出错了',
    desc: '发生了一个未知错误',
    rawError: '',
    showDetails: false,
  })

  // ---- Helpers ----

  function renderMarkdown(md) {
    if (!md) return ''
    return renderMarkdownWithLatex(md)
  }

  function getSummary(item) {
    if (!item) return ''
    if (!item.rounds?.length) return item.summary || ''
    const text = item.rounds[item.rounds.length - 1]?.aiResponse || ''
    return text.substring(0, 30).replace(/\n/g, ' ') + '...'
  }

  function getRoundsCount(item) {
    return item?.rounds?.length || 1
  }

  function getFullContent(item) {
    if (!item) return ''
    if (!item.rounds?.length) return item.full || ''
    return item.rounds.map(r => r.aiResponse || '').join('\n\n---\n\n')
  }

  function createHistoryItem(userScreenshot) {
    return {
      time: new Date().toLocaleTimeString(),
      rounds: [{ userScreenshot: userScreenshot || '', thinking: '', aiResponse: '', error: null }],
    }
  }

  function addRoundToItem(item, userScreenshot) {
    if (!item.rounds) item.rounds = []
    item.rounds.push({
      userScreenshot: userScreenshot || '',
      thinking: '',
      thinkingStatus: 'Thinking Process',
      thinkingDuration: 0,
      aiResponse: '',
      error: null,
    })
  }

  function getCurrentRound(item) {
    if (!item?.rounds?.length) return null
    return item.rounds[item.rounds.length - 1]
  }

  function getThinkingPreview(thinking) {
    if (!thinking) return ''
    const lines = thinking.split('\n').filter(l => l.trim())
    if (lines.length === 0) return 'Just started...'
    const lastLine = lines[lines.length - 1]
    return lastLine.length > 80 ? lastLine.substring(0, 80) + '...' : lastLine
  }

  // ---- Computed ----
  const currentRounds = computed(() => {
    const item = history.value[activeHistoryIndex.value]
    return item?.rounds || []
  })

  // ---- Core actions ----

  function selectHistory(idx) {
    if (history.value[idx]) activeHistoryIndex.value = idx
  }

  function handleStreamStart(keepContext) {
    streamBuffer = ''
    thinkingBuffer = ''
    thinkingStartTime = 0
    isThinking.value = false
    thinkingStatusText.value = 'Thinking Process'
    thinkingExpanded.value = true
    isThinkingStalled.value = false
    if (renderRafId) { cancelAnimationFrame(renderRafId); renderRafId = null }
    renderDirty = false
    streamingHtml.value = ''
    if (stallTimer) clearTimeout(stallTimer)

    if (keepContext && history.value.length > 0 && !shouldOverwriteHistory.value) {
      const currentItem = history.value[0]
      const lastRound = getCurrentRound(currentItem)
      if (lastRound && lastRound.error) {
        lastRound.userScreenshot = pendingUserScreenshot || lastRound.userScreenshot
        lastRound.thinking = ''
        lastRound.thinkingStatus = 'Thinking Process'
        lastRound.aiResponse = ''
        lastRound.error = null
      } else {
        addRoundToItem(currentItem, pendingUserScreenshot)
      }
      activeHistoryIndex.value = 0
      pendingUserScreenshot = ''
    } else {
      if (shouldOverwriteHistory.value && history.value.length > 0) {
        history.value[0] = createHistoryItem(pendingUserScreenshot)
        shouldOverwriteHistory.value = false
      } else {
        history.value.unshift(createHistoryItem(pendingUserScreenshot))
      }
      activeHistoryIndex.value = 0
      pendingUserScreenshot = ''
    }
  }

  function handleStreamChunk(token) {
    if (isLoading.value) isLoading.value = false
    if (isAppending.value) isAppending.value = false
    if (isThinking.value) {
      isThinking.value = false
      thinkingExpanded.value = false
      if (stallTimer) clearTimeout(stallTimer)
    }
    streamBuffer += token

    if (history.value.length > 0) {
      const round = getCurrentRound(history.value[0])
      if (round) round.aiResponse = streamBuffer
    }

    // 用 rAF 节流渲染，避免每个 token 都触发 DOM 更新
    if (!renderDirty) {
      renderDirty = true
      renderRafId = requestAnimationFrame(() => {
        renderDirty = false
        renderRafId = null
        streamingHtml.value = renderMarkdownWithLatex(streamBuffer)
        scrollContentToBottom()
      })
    }
  }

  function handleThinkingChunk(token) {
    if (isLoading.value) isLoading.value = false
    if (isAppending.value) isAppending.value = false
    if (!isThinking.value) {
      thinkingStartTime = Date.now()
      thinkingExpanded.value = true
    }
    isThinking.value = true
    isThinkingStalled.value = false
    if (stallTimer) clearTimeout(stallTimer)
    stallTimer = setTimeout(() => {
      if (isThinking.value) isThinkingStalled.value = true
    }, 1000)

    thinkingBuffer += token

    if (token.length > 2) {
      if (token.match(/search|web|google/i)) thinkingStatusText.value = 'Searching Internet...'
      else if (token.match(/code|function|import|class/i)) thinkingStatusText.value = 'Generating Code...'
      else if (token.match(/error|bug|fix/i)) thinkingStatusText.value = 'Debugging...'
      else if (token.match(/analyse|analyze|review/i)) thinkingStatusText.value = 'Analyzing...'
    }

    if (history.value.length > 0) {
      const round = getCurrentRound(history.value[0])
      if (round) {
        round.thinking = thinkingBuffer
        round.thinkingStatus = thinkingStatusText.value
      }
    }
    scrollContentToBottom()
  }

  function handleSolution(data) {
    isLoading.value = false
    isAppending.value = false
    isThinking.value = false
    if (stallTimer) clearTimeout(stallTimer)
    thinkingStartTime = 0
    if (renderRafId) { cancelAnimationFrame(renderRafId); renderRafId = null }
    renderDirty = false
    streamingHtml.value = ''
    if (history.value.length > 0) {
      const round = getCurrentRound(history.value[0])
      if (round && !round.aiResponse) round.aiResponse = data
    }
  }

  function handleInlineError(errorInfo) {
    if (history.value.length > 0) {
      const round = getCurrentRound(history.value[0])
      if (round) {
        round.error = {
          title: errorInfo.title || '请求出错',
          desc: errorInfo.desc || '未知错误',
          icon: errorInfo.icon || '❌',
        }
        isLoading.value = false
        isAppending.value = false
        isThinking.value = false
        return true
      }
    }
    return false
  }

  function clearInlineError() {
    if (history.value.length > 0) {
      const round = getCurrentRound(history.value[0])
      if (round) round.error = null
    }
  }

  function setStreamBuffer(val) { streamBuffer = val }
  function setUserScreenshot(screenshot) { pendingUserScreenshot = screenshot }

  function deleteHistory(index) {
    if (index < 0 || index >= history.value.length) return
    history.value.splice(index, 1)
    if (history.value.length === 0) activeHistoryIndex.value = 0
    else if (index <= activeHistoryIndex.value) {
      activeHistoryIndex.value = Math.max(0, activeHistoryIndex.value - 1)
    }
  }

  function scrollContentToBottom() {
    const el = document.getElementById('content')
    if (el) el.scrollTop = el.scrollHeight
  }

  // ---- Export image (migrated from useSolution) ----

  function createRoundCard(round, roundIndex, totalRounds) {
    const card = document.createElement('div')
    card.style.cssText = `display:flex;gap:24px;align-items:stretch;margin-bottom:${roundIndex < totalRounds - 1 ? '24px' : '0'};padding-bottom:${roundIndex < totalRounds - 1 ? '24px' : '0'};border-bottom:${roundIndex < totalRounds - 1 ? '1px dashed #cbd5e1' : 'none'};`
    // Left panel - user input
    const leftPanel = document.createElement('div')
    leftPanel.style.cssText = 'flex:0 0 240px;background:white;border-radius:12px;padding:16px;box-shadow:0 2px 4px rgba(0,0,0,0.05);border:1px solid #e2e8f0;'
    const userHeader = document.createElement('div')
    userHeader.style.cssText = 'display:flex;align-items:center;gap:8px;margin-bottom:12px;padding-bottom:10px;border-bottom:1px solid #f1f5f9;'
    userHeader.innerHTML = `<div style="width:28px;height:28px;border-radius:50%;background:linear-gradient(135deg,#6366f1,#4f46e5);display:flex;align-items:center;justify-content:center;color:white;font-size:12px;">👤</div><div><div style="font-weight:600;font-size:12px;color:#334155;">问题 ${roundIndex + 1}</div></div>`
    leftPanel.appendChild(userHeader)
    if (round.userScreenshot) {
      const imgC = document.createElement('div')
      imgC.innerHTML = `<img src="${round.userScreenshot}" style="width:100%;border-radius:6px;border:1px solid #e2e8f0;" />`
      leftPanel.appendChild(imgC)
    } else {
      const ph = document.createElement('div')
      ph.style.cssText = 'padding:20px;text-align:center;color:#94a3b8;font-size:12px;background:#f8fafc;border-radius:6px;'
      ph.textContent = '无截图'
      leftPanel.appendChild(ph)
    }
    card.appendChild(leftPanel)

    // Right panel - AI response
    const rightPanel = document.createElement('div')
    rightPanel.style.cssText = 'flex:1;background:white;border-radius:12px;padding:16px;box-shadow:0 2px 4px rgba(0,0,0,0.05);border:1px solid #e2e8f0;overflow:hidden;'
    const aiHeader = document.createElement('div')
    aiHeader.style.cssText = 'display:flex;align-items:center;gap:8px;margin-bottom:12px;padding-bottom:10px;border-bottom:1px solid #f1f5f9;'
    aiHeader.innerHTML = `<div style="width:28px;height:28px;border-radius:50%;background:linear-gradient(135deg,#6366f1,#818cf8);display:flex;align-items:center;justify-content:center;color:white;font-size:12px;">🤖</div><div><div style="font-weight:600;font-size:12px;color:#334155;">AI 回复</div></div>`
    rightPanel.appendChild(aiHeader)
    const aiContent = document.createElement('div')
    aiContent.style.cssText = 'font-size:13px;line-height:1.6;color:#334155;'
    aiContent.innerHTML = renderMarkdown(round.aiResponse || '')
    rightPanel.appendChild(aiContent)
    card.appendChild(rightPanel)
    return card
  }

  async function exportImage(index) {
    const item = history.value[index]
    if (!item) return
    const rounds = item.rounds || []
    if (rounds.length === 0) return
    try {
      const { default: html2canvas } = await import('html2canvas')
      const container = document.createElement('div')
      container.style.cssText = 'position:fixed;left:-9999px;top:0;width:900px;padding:28px;background:linear-gradient(135deg,#f8fafc 0%,#e2e8f0 100%);font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;color:#1e293b;border-radius:16px;'
      if (rounds.length > 1) {
        const title = document.createElement('div')
        title.style.cssText = 'font-size:14px;font-weight:600;color:#64748b;margin-bottom:20px;padding-bottom:12px;border-bottom:1px solid #cbd5e1;'
        title.textContent = `共 ${rounds.length} 轮对话`
        container.appendChild(title)
      }
      rounds.forEach((round, idx) => container.appendChild(createRoundCard(round, idx, rounds.length)))
      const footer = document.createElement('div')
      footer.style.cssText = 'display:flex;justify-content:space-between;align-items:center;margin-top:20px;padding-top:14px;border-top:1px solid #cbd5e1;font-size:11px;color:#64748b;'
      footer.innerHTML = `<div style="display:flex;align-items:center;gap:6px;"><span style="font-weight:600;">Q-Solver</span></div><div>${new Date().toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' })}</div>`
      container.appendChild(footer)
      document.body.appendChild(container)
      const canvas = await html2canvas(container, { backgroundColor: null, scale: 2, useCORS: true, logging: false })
      document.body.removeChild(container)
      const b64 = canvas.toDataURL('image/png')
      await api.saveImageToFile(b64)
    } catch (e) {
      console.error('导出图片失败:', e)
    }
  }

  return {
    history, activeHistoryIndex, isLoading, isAppending, shouldOverwriteHistory,
    isThinking, thinkingStatusText, thinkingExpanded, isThinkingStalled,
    streamingHtml, errorState, currentRounds,
    renderMarkdown, getSummary, getRoundsCount, getFullContent, getThinkingPreview,
    selectHistory, handleStreamStart, handleStreamChunk, handleThinkingChunk,
    handleSolution, handleInlineError, clearInlineError,
    setStreamBuffer, setUserScreenshot, deleteHistory, exportImage,
  }
})
