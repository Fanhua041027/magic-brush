import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import { ChatWithDeepSeek, ChatWithDeepSeekStream, ChatWithScreenshot, ChatWithScreenshotSync, ChatWithDeepSeekStreamWithContext } from '../../wailsjs/go/app/App'
import { useSettingsStore } from './settings'

// 面试助手个人化 System Prompt
function buildInterviewSystemPrompt(resumeContent) {
  let prompt = `你是一位 AI 面试辅助助手。你的核心任务是以用户「朱晋辉」的身份和口吻，用第一人称「我」来回答面试问题。

<身份背景>
姓名：朱晋辉
年龄：21岁
学校：吕梁学院 — 数据科学与大数据技术专业（2027届）
求职意向：AI Agent 开发工程师（实习生）
核心能力：AI Agent 开发（LangChain/LangGraph）、大模型应用（RAG、Prompt工程、模型微调）、Python/Java 全栈、K8s/Docker 部署
荣誉：国家级竞赛奖项5+项、软件著作权3项、华为鸿蒙校园大使
</身份背景>

<回答风格>
1. 用第一人称「我」回答问题，模仿用户本人在面试现场的表达方式
2. 语气自信、专业、谦逊 — 像一个有实战经验的在校生
3. 技术表达直接、精准，用技术术语沟通（面试官是懂技术的）
4. 回答结构清晰：先给出核心结论，再展开具体细节
5. 项目经验用 STAR 法则（情境-任务-行动-结果）组织
6. 遇到不熟悉的技术领域坦诚说「了解不多，但我的理解是…」，不要编造
7. 避免空泛套话、避免过度谦虚、避免过于冗长
</回答风格>

<项目经验>
1. 金融情绪感知与决策 AI Agent（负责人 | 2025.07-2025.10）
   基于 LangGraph 设计「分析-反思-决策」三阶段状态机，引入自洽性机制进行多角色投票仲裁
   使用 INT8 量化将模型显存从 14GB 降至 7.5GB，推理延迟从 1.2s 优化至 0.6s
   编写 Dockerfile 与 K8s 配置，利用 HPA 实现弹性扩容，在 50 QPS 下保持系统稳定
   在消费级显卡上实现实时分析，保持 98% 原始精度

2. 金融资讯智能采集与推理 Agent 系统（多智能体架构师 | 2025.09-2025.12）
   基于 MoA 架构设计「总指挥+专家团」协作模式，研报生成效率提升 5 倍
   研发 RAG 幻觉抑制引擎，结合自纠错机制验证财务指标，多源冲突仲裁准确率达 92%
   利用 NLP 提取实体关系，搭建动态更新的金融知识图谱底座

3. 面向工业场景的 AI Agent 编排底座与事件驱动原型（AI全栈 | 2026.02-2026.04）
   基于 Python Dataclass 定义标准化事件 Schema，实现三大职能模块解耦
   依托 FastAPI Background Tasks 搭建异步队列，保障数据一致性达 100%
   基于 Streamlit 搭建全链路可视化监控面板，预留自然语言交互接口
</项目经验>

<实习经历>
- 临汾市商巢科技 — AI Agent 算法实习生（2025.07-2025.12）：基于 LangGraph 构建 TradingAgents 系统，搭建 GraphRAG 知识图谱，构建日均处理 2 万+条资讯的分布式管道
- 上海言楚实业 — 大模型应用开发工程师（2026.02-2026.05）：基于 FastAPI 构建高可用后端，研发 RAG 系统与自纠错机制，主导 AI 与企业 ERP/CRM 系统集成
</实习经历>

<行为规则>
1. 结合用户的简历和项目经历来回答问题，让回答有具体案例支撑
2. 如果是算法或技术题，先给出解题思路，再写代码
3. 如果是行为面试题（如"请介绍你自己"），用 STAR 法则组织回答
4. 回答中适当展现技术深度（如提到量化、推理框架、架构设计等）
5. 不要输出与面试无关的寒暄或闲聊内容
6. 保持回答简洁，重点突出，便于面试时口头表达
7. **直接输出回答内容，不要添加任何前缀、后缀、说明或思考过程**
8. **不要出现"根据你的简历"、"以下是你对于XX的回答"、"你可以这样表达"、"这是我的回答"等框架性语句**
9. **不要解释你要怎么回答，不要输出"首先"、"其次"等思考标记——直接以第一人称「我」开始说话**
10. **你的输出就是面试者本人在面试现场说出的原话，不是建议，不是模板，直接说出来即可**
</行为规则>`

  if (resumeContent) {
    prompt += '\n\n【用户简历】\n' + resumeContent
  }

  return prompt
}

// 本地存储键名
const STORAGE_KEY = 'magic-brush-chat-history'
const MAX_HISTORY = 50 // 最大历史记录数

export const useChatStore = defineStore('chat', () => {
  const isVisible = ref(false)
  const isLoading = ref(false)
  const messages = ref([])
  const currentStreamContent = ref('')
  let streamReceivedData = false // 标记流式是否产生过数据（防竞态）

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

  // ── 多对话管理 ──────────────────────────────────────────────
  const CONV_LIST_KEY = 'magic-brush-conversation-list'
  const ACTIVE_CONV_KEY = 'magic-brush-active-conversation'
  const savedConversations = ref([])
  const activeConversationId = ref(null)

  function loadConversationList() {
    try {
      const stored = localStorage.getItem(CONV_LIST_KEY)
      if (stored) savedConversations.value = JSON.parse(stored)
    } catch (e) { /* ignore */ }
    try {
      const active = localStorage.getItem(ACTIVE_CONV_KEY)
      if (active) activeConversationId.value = active
    } catch (e) { /* ignore */ }
  }

  function saveConversationList() {
    try { localStorage.setItem(CONV_LIST_KEY, JSON.stringify(savedConversations.value)) } catch (e) { /* ignore */ }
  }

  function saveCurrentConversation() {
    if (messages.value.length === 0) return
    const convId = Date.now().toString()
    const firstUserMsg = messages.value.find(m => m.role === 'user')
    const title = firstUserMsg ? firstUserMsg.content.slice(0, 30) + (firstUserMsg.content.length > 30 ? '...' : '') : '新对话'
    const conv = {
      id: convId,
      title,
      messages: JSON.parse(JSON.stringify(messages.value)),
      createdAt: new Date().toISOString(),
      messageCount: messages.value.length,
    }
    // 替换或新增
    const existingIdx = savedConversations.value.findIndex(c => c.id === activeConversationId.value)
    if (existingIdx >= 0) {
      savedConversations.value[existingIdx] = conv
    } else {
      savedConversations.value.unshift(conv)
    }
    // 限制保存数量
    if (savedConversations.value.length > 20) savedConversations.value = savedConversations.value.slice(0, 20)
    activeConversationId.value = convId
    saveConversationList()
    localStorage.setItem(ACTIVE_CONV_KEY, convId)
  }

  function startNewConversation() {
    if (messages.value.length > 0) saveCurrentConversation()
    messages.value = []
    activeConversationId.value = Date.now().toString()
    localStorage.setItem(ACTIVE_CONV_KEY, activeConversationId.value)
    localStorage.removeItem(STORAGE_KEY)
  }

  function loadConversation(id) {
    const conv = savedConversations.value.find(c => c.id === id)
    if (conv) {
      if (messages.value.length > 0) saveCurrentConversation()
      messages.value = JSON.parse(JSON.stringify(conv.messages))
      activeConversationId.value = id
      localStorage.setItem(ACTIVE_CONV_KEY, id)
      saveHistory()
    }
  }

  function deleteConversation(id) {
    savedConversations.value = savedConversations.value.filter(c => c.id !== id)
    saveConversationList()
    if (activeConversationId.value === id) {
      activeConversationId.value = null
      localStorage.removeItem(ACTIVE_CONV_KEY)
    }
  }

  // 初始化时加载
  loadHistory()
  loadConversationList()

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

    addMessage('user', text)
    isLoading.value = true
    currentStreamContent.value = ''
    streamReceivedData = false // 重置流式数据标记

    try {
      // 构建带个人化 System Prompt 的消息列表
      const settingsStore = useSettingsStore()
      const systemPrompt = buildInterviewSystemPrompt(settingsStore.settings.resumeContent)
      const msgs = [
        { role: 'system', content: systemPrompt },
      ]
      // 添加上文对话历史（最多保留最近 5 轮，避免超出上下文）
      const history = messages.value.slice(-10)
      for (const msg of history) {
        if (msg.role === 'user' || msg.role === 'assistant') {
          msgs.push({ role: msg.role, content: msg.content })
        }
      }
      // 加上当前用户消息
      msgs.push({ role: 'user', content: text })

      // 流式输出：传入完整消息列表，后端按序处理
      await ChatWithDeepSeekStreamWithContext(msgs)

      // 等一小段时间让流式事件到达（防止竞态条件下误判无输出）
      if (!streamReceivedData) {
        await new Promise(resolve => setTimeout(resolve, 800))
      }

      // 检查流式是否真的产生了助手消息
      const hasAssistantMsg = messages.value.some(
        m => m.role === 'assistant' && !m._streaming && m.content.length > 0
      )
      if (!hasAssistantMsg && !streamReceivedData) {
        // 流式无输出，降级为非流式
        const result = await ChatWithDeepSeek(text)
        if (result) {
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

    streamReceivedData = true // 标记已有流式数据
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
  async function sendMessageWithScreenshot(text, screenshotBase64) {
    if (!text.trim() || isLoading.value) return

    addMessage('user', text + ' [附截图]')
    isLoading.value = true
    currentStreamContent.value = ''
    streamReceivedData = false

    try {
      // 流式输出：Go 后端逐字推送，事件驱动更新 messages
      await ChatWithScreenshot(text, screenshotBase64, '')

      // 等一小段时间让流式事件到达（防竞态）
      if (!streamReceivedData) {
        await new Promise(resolve => setTimeout(resolve, 800))
      }

      // 检查流式是否真的产生了助手消息
      const hasAssistantMsg = messages.value.some(
        m => m.role === 'assistant' && !m._streaming && m.content.length > 0
      )
      if (!hasAssistantMsg && !streamReceivedData) {
        // 流式无输出，降级为非流式
        const result = await ChatWithScreenshotSync(text, screenshotBase64, '')
        if (result) {
          addMessage('assistant', result)
        }
      }
    } catch (error) {
      console.error('Chat with screenshot error:', error)
      addMessage('assistant', `抱歉，发生了错误: ${error.message || '未知错误'}`)
    } finally {
      isLoading.value = false
    }
  }

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
    sendMessageWithScreenshot,
    handleStreamChunk,
    handleStreamDone,
    handleStreamError,
    exportHistory,
    importHistory,
    savedConversations,
    activeConversationId,
    saveCurrentConversation,
    startNewConversation,
    loadConversation,
    deleteConversation,
  }
})
