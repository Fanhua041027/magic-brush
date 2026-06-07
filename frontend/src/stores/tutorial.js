import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useTutorialStore = defineStore('tutorial', () => {
  const isVisible = ref(false)
  const currentStep = ref(0) // 0-indexed: 0-5
  const completedSteps = ref(new Set())

  // ── 步骤定义 ──────────────────────────────────────────────
  const STEPS = [
    {
      id: 'language',
      title: '设置语言',
      subtitle: '初始化环境变量',
      phase: 0, // Phase 0 = 面试前准备
      icon: 'globe',
    },
    {
      id: 'context',
      title: '添加简历和岗位描述',
      subtitle: '构建 RAG 索引',
      phase: 0,
      icon: 'file-text',
    },
    {
      id: 'knowledge',
      title: '添加知识和题库',
      subtitle: '扩展上下文窗口',
      phase: 0,
      icon: 'database',
    },
    {
      id: 'profile',
      title: '保存技能偏好',
      subtitle: '固化回答策略',
      phase: 0,
      icon: 'star',
    },
    {
      id: 'interview',
      title: '用「提问」回答问题',
      subtitle: '实时辅助模式',
      phase: 1, // Phase 1 = 面试中使用
      icon: 'message-circle',
    },
    {
      id: 'screenshot',
      title: '用「截图」处理屏幕题',
      subtitle: 'OCR + 代码生成',
      phase: 1,
      icon: 'camera',
    },
  ]

  // ── 计算属性 ──────────────────────────────────────────────
  const totalSteps = computed(() => STEPS.length)
  const phase1Steps = computed(() => STEPS.filter(s => s.phase === 0).length)
  const phase2Steps = computed(() => STEPS.filter(s => s.phase === 1).length)

  const currentStepMeta = computed(() => STEPS[currentStep.value])

  const currentPhaseIndex = computed(() => {
    const phaseSteps = STEPS.filter(s => s.phase === currentStepMeta.value.phase)
    return phaseSteps.findIndex(s => s.id === currentStepMeta.value.id) + 1
  })

  const currentPhaseTotal = computed(() =>
    currentStepMeta.value.phase === 0 ? phase1Steps.value : phase2Steps.value
  )

  const phaseLabel = computed(() =>
    currentStepMeta.value.phase === 0 ? '面试前准备' : '面试中使用'
  )

  const isFirstStep = computed(() => currentStep.value === 0)
  const isLastStep = computed(() => currentStep.value === STEPS.length - 1)
  const allCompleted = computed(() => completedSteps.value.size >= STEPS.length)

  const progressPercent = computed(() =>
    Math.round((completedSteps.value.size / STEPS.length) * 100)
  )

  function isStepCompleted(index) {
    return completedSteps.value.has(index)
  }

  function isStepActive(index) {
    return currentStep.value === index
  }

  function isStepAvailable(index) {
    // Steps can be navigated to if completed or the next uncompleted
    if (isStepCompleted(index)) return true
    // Allow navigating to the current step
    if (index === currentStep.value) return true
    // Allow navigating to the first uncompleted step
    for (let i = 0; i < STEPS.length; i++) {
      if (!completedSteps.value.has(i)) return i === index
    }
    return false
  }

  // ── 操作方法 ──────────────────────────────────────────────
  function show() {
    isVisible.value = true
  }

  function hide() {
    isVisible.value = false
  }

  function goToStep(index) {
    if (index >= 0 && index < STEPS.length && isStepAvailable(index)) {
      currentStep.value = index
    }
  }

  function nextStep() {
    if (currentStep.value < STEPS.length - 1) {
      markCurrentCompleted()
      currentStep.value++
    }
  }

  function prevStep() {
    if (currentStep.value > 0) {
      currentStep.value--
    }
  }

  function markCurrentCompleted() {
    completedSteps.value.add(currentStep.value)
    // Trigger reactivity by creating a new Set
    completedSteps.value = new Set(completedSteps.value)
  }

  function markStepCompleted(index) {
    completedSteps.value.add(index)
    completedSteps.value = new Set(completedSteps.value)
  }

  function skipToEnd() {
    // Mark all as completed and hide
    for (let i = 0; i < STEPS.length; i++) {
      completedSteps.value.add(i)
    }
    completedSteps.value = new Set(completedSteps.value)
    isVisible.value = false
  }

  function reset() {
    currentStep.value = 0
    completedSteps.value = new Set()
  }

  return {
    isVisible,
    currentStep,
    completedSteps,
    STEPS,
    totalSteps,
    phase1Steps,
    phase2Steps,
    currentStepMeta,
    currentPhaseIndex,
    currentPhaseTotal,
    phaseLabel,
    isFirstStep,
    isLastStep,
    allCompleted,
    progressPercent,
    isStepCompleted,
    isStepActive,
    isStepAvailable,
    show,
    hide,
    goToStep,
    nextStep,
    prevStep,
    markCurrentCompleted,
    markStepCompleted,
    skipToEnd,
    reset,
  }
})
