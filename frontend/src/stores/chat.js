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
求职意向：AI应用开发工程师（实习生）
核心能力：AI Agent 开发与编排（LangChain/LangGraph）、大模型应用（RAG、Prompt工程、模型微调）、Python/Java 全栈、K8s/Docker 部署
主修课程：人工智能、机器学习、深度学习、大模型(LLM)应用开发、LangChain/LangGraph框架、数据结构与算法
荣誉：省级以上竞赛奖项13+项、软件著作权4项、华为鸿蒙校园大使
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
1. 金融智能决策与投研多智能体系统（币星人）（负责人 | 2025.06-2025.12）
   基于 LangGraph 设计「总指挥+专家团」协作模式与 FSM 状态机，MoA 架构使研报生成效率提升 5 倍
   AWQ-4bit 量化将显存从 64GB 压缩至单卡可承载范围，单句推理延迟优化至 0.7s，精度损失<2%
   K8s 弹性伸缩确保单副本 50 QPS 无请求堆积，GPT-4o-mini 异步兜底保障服务 99.9% 可用性
   构建 RAG+代码解释器+知识图谱三元幻觉抑制引擎，多源冲突仲裁准确率达 92%
   Playwright 采集 4 个财经网站，NLP 提取实体关系，日均处理 5000 条原始文本

2. 工业级ERP系统Agent调度中间件（AI全栈 | 2026.02-2026.05）
   采用 FastAPI 构建独立 Agent 网关，定义基于 Pydantic 的强类型请求/响应 Schema 对接 Java 后端
   实现内存队列重试+超时熔断机制，构造 20+ 异常用例验证系统降级表现，无脏数据产生
   Agent 指令执行成功率从 72% 提升至 96%，在采购审批、库存预警、工单派发场景完成端到端集成
   开发 Streamlit 可视化控制台及 NLP-to-API 映射中间件，简单指令解析准确率达 90%
</项目经验>

<实习经历>
- 临汾市商巢科技 — 后端开发实习生（2025.06-2025.12）：搭建 GraphRAG 与金融知识图谱，关键资讯筛选准确率从 78% 提升至 92%；基于 LangGraph 构建 TradingAgents 多智能体系统；运维日均 2 万+条资讯的分布式采集管道
- 上海言楚实业 — AI全栈开发实习生（2026.02-2026.06）：基于 Python/FastAPI 构建高可用后端，设计多 Agent 协同架构与标准化通信协议；落地 Saga 模式补偿机制与深度容错策略，实现异常场景自动回滚与数据最终一致性
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
