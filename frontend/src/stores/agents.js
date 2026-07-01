/**
 * Agent Store — 管理 Profile/Interview/Exam/Mock 等 Agent 状态
 */
import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'
import { api } from '../services/api'

export const useAgentStore = defineStore('agents', () => {
  // ---- Mode ----
  const activeMode = ref('solve') // 'solve' | 'interview' | 'exam' | 'mock' | 'profile'

  // ---- Pipeline Events ----
  const pipelineEvents = ref([])
  const isPipelineRunning = ref(false)

  // ---- Profile ----
  const profile = reactive({
    resumeRaw: '',
    resumeSummary: '',
    jdSummary: '',
    skillCards: [],
    language: 'zh-CN',
  })
  const hasSnapshot = ref(false)

  // ---- Knowledge Base ----
  const kbDocuments = ref([])

  // ---- Exam ----
  const examResult = reactive({
    ocrText: '',
    type: '',
    summary: '',
    steps: [],
  })

  // ---- Mock Interview ----
  const mockState = reactive({
    questions: [],
    currentIdx: 0,
    total: 0,
    scores: [],
    finished: false,
    averageScore: 0,
  })

  // ---- Interview ----
  const interviewActive = ref(false)

  // ---- Methods ----

  function setMode(mode) {
    activeMode.value = mode
  }

  function addPipelineEvent(eventStr) {
    // event format: "stage|status|detail"
    const parts = eventStr.split('|')
    if (parts.length >= 2) {
      pipelineEvents.value.unshift({
        stage: parts[0],
        status: parts[1],
        detail: parts.slice(2).join('|') || '',
        time: new Date().toLocaleTimeString(),
      })
      // Keep only last 20 events
      if (pipelineEvents.value.length > 20) {
        pipelineEvents.value = pipelineEvents.value.slice(0, 20)
      }
      isPipelineRunning.value = parts[1] === 'running'
      if (parts[1] === 'running' && parts[0] === 'agent') {
        isPipelineRunning.value = true
      }
      if (parts[1] === 'done' || parts[1] === 'error') {
        isPipelineRunning.value = false
      }
    }
  }

  function clearPipelineEvents() {
    pipelineEvents.value = []
    isPipelineRunning.value = false
  }

  // ---- Profile Operations ----
  async function loadProfile() {
    try {
      const result = await api.profileGet()
      if (result) {
        Object.assign(profile, JSON.parse(result))
      }
    } catch (e) {
      console.error('加载Profile失败', e)
    }
  }

  async function generateSummary(rawText, type) {
    try {
      const result = await api.profileGenerateSummary(rawText, type)
      if (result) {
        const parsed = JSON.parse(result)
        return parsed.summary || parsed.error || ''
      }
    } catch (e) {
      console.error('生成摘要失败', e)
    }
    return ''
  }

  async function generateSkillCard(projectDesc) {
    try {
      const result = await api.profileGenerateSkillCard(projectDesc)
      if (result) {
        return JSON.parse(result)
      }
    } catch (e) {
      console.error('生成技能卡片失败', e)
    }
    return null
  }

  async function addSkillCard(card) {
    try {
      await api.profileAddSkillCard(JSON.stringify(card))
      await loadProfile()
    } catch (e) {
      console.error('添加技能卡片失败', e)
    }
  }

  async function removeSkillCard(id) {
    try {
      await api.profileRemoveSkillCard(id)
      await loadProfile()
    } catch (e) {
      console.error('删除技能卡片失败', e)
    }
  }

  async function createSnapshot() {
    try {
      const result = await api.profileCreateSnapshot()
      if (result) {
        hasSnapshot.value = true
        return JSON.parse(result)
      }
    } catch (e) {
      console.error('创建快照失败', e)
    }
    return null
  }

  // ---- Knowledge Base Operations ----
  async function loadKBDocuments() {
    try {
      const result = await api.kbListDocuments()
      if (result) {
        kbDocuments.value = JSON.parse(result)
      }
    } catch (e) {
      console.error('加载知识库失败', e)
    }
  }

  async function addKBDocument(title, content, source) {
    try {
      await api.kbAddDocument(title, content, source)
      await loadKBDocuments()
    } catch (e) {
      console.error('添加文档失败', e)
    }
  }

  async function deleteKBDocument(id) {
    try {
      await api.kbDeleteDocument(id)
      await loadKBDocuments()
    } catch (e) {
      console.error('删除文档失败', e)
    }
  }

  // ---- Mock Interview ----
  async function startMock() {
    try {
      const result = await api.mockStart()
      if (result) {
        const parsed = JSON.parse(result)
        if (parsed.error) return parsed
        mockState.questions = parsed.questions || []
        mockState.total = parsed.total || 0
        mockState.currentIdx = 0
        mockState.scores = []
        mockState.finished = false
        mockState.averageScore = 0
        return parsed
      }
    } catch (e) {
      console.error('启动模拟面试失败', e)
    }
    return null
  }

  async function submitMockAnswer(answer) {
    try {
      const result = await api.mockSubmitAnswer(answer)
      if (result) {
        const parsed = JSON.parse(result)
        if (parsed.error) return parsed
        if (parsed.score) {
          mockState.scores.push(parsed.score)
        }
        mockState.currentIdx++
        mockState.finished = parsed.finished
        if (mockState.finished) {
          await loadMockSummary()
        }
        return parsed
      }
    } catch (e) {
      console.error('提交答案失败', e)
    }
    return null
  }

  async function loadMockSummary() {
    try {
      const result = await api.mockGetSummary()
      if (result) {
        const parsed = JSON.parse(result)
        mockState.averageScore = parsed.average || 0
        return parsed
      }
    } catch (e) {
      console.error('加载总结失败', e)
    }
    return null
  }

  return {
    activeMode, pipelineEvents, isPipelineRunning,
    profile, hasSnapshot, kbDocuments,
    examResult, mockState, interviewActive,
    setMode, addPipelineEvent, clearPipelineEvents,
    loadProfile, generateSummary, generateSkillCard, addSkillCard, removeSkillCard, createSnapshot,
    loadKBDocuments, addKBDocument, deleteKBDocument,
    startMock, submitMockAnswer, loadMockSummary,
  }
})
