<template>
  <Teleport to="body">
    <Transition name="tutorial-fade">
      <div v-if="tutorial.isVisible" class="tutorial-overlay" @click.self="handleOverlayClick">
        <div class="tutorial-modal">
          <!-- ── Left Sidebar ──────────────────────────────────── -->
          <aside class="tutorial-sidebar">
            <div class="sidebar-header">
              <div class="sidebar-logo">
                <span class="logo-icon">M</span>
              </div>
              <h2 class="sidebar-title">先了解<br />Magic Brush</h2>
              <div class="sidebar-progress">
                <div class="progress-bar-track">
                  <div class="progress-bar-fill" :style="{ width: tutorial.progressPercent + '%' }"></div>
                </div>
                <span class="progress-text">{{ tutorial.completedSteps.size }} / {{ tutorial.totalSteps }}</span>
              </div>
            </div>

            <nav class="sidebar-nav">
              <!-- Phase 1: 面试前准备 -->
              <div class="phase-section">
                <div class="phase-label">面试前准备</div>
                <button
                  v-for="(step, i) in phase1Steps"
                  :key="step.id"
                  class="step-item"
                  :class="{
                    active: tutorial.isStepActive(i),
                    completed: tutorial.isStepCompleted(i),
                    locked: !tutorial.isStepAvailable(i),
                  }"
                  @click="tutorial.goToStep(i)"
                  :disabled="!tutorial.isStepAvailable(i)"
                >
                  <span class="step-indicator">
                    <Icon v-if="tutorial.isStepCompleted(i)" name="check" :size="14" />
                    <span v-else class="step-num">{{ i + 1 }}</span>
                  </span>
                  <div class="step-info">
                    <span class="step-title">{{ step.title }}</span>
                    <span class="step-subtitle">{{ step.subtitle }}</span>
                  </div>
                </button>
              </div>

              <!-- Phase 2: 面试中使用 -->
              <div class="phase-section">
                <div class="phase-label">面试中使用</div>
                <button
                  v-for="(step, i) in phase2Steps"
                  :key="step.id"
                  class="step-item"
                  :class="{
                    active: tutorial.isStepActive(phase1Steps.length + i),
                    completed: tutorial.isStepCompleted(phase1Steps.length + i),
                    locked: !tutorial.isStepAvailable(phase1Steps.length + i),
                  }"
                  @click="tutorial.goToStep(phase1Steps.length + i)"
                  :disabled="!tutorial.isStepAvailable(phase1Steps.length + i)"
                >
                  <span class="step-indicator">
                    <Icon v-if="tutorial.isStepCompleted(phase1Steps.length + i)" name="check" :size="14" />
                    <span v-else class="step-num">{{ phase1Steps.length + i + 1 }}</span>
                  </span>
                  <div class="step-info">
                    <span class="step-title">{{ step.title }}</span>
                    <span class="step-subtitle">{{ step.subtitle }}</span>
                  </div>
                </button>
              </div>
            </nav>

            <div class="sidebar-footer">
              <button class="btn-skip" @click="tutorial.skipToEnd">
                <Icon name="skip-forward" :size="14" />
                <span>跳过教程</span>
              </button>
            </div>
          </aside>

          <!-- ── Right Content Area ──────────────────────────────── -->
          <main class="tutorial-content">
            <!-- Header -->
            <div class="content-header">
              <div class="content-breadcrumb">
                <span class="phase-tag">{{ tutorial.phaseLabel }}</span>
                <span class="step-counter">{{ tutorial.currentPhaseIndex }} / {{ tutorial.currentPhaseTotal }}</span>
              </div>
              <button class="btn-close" @click="handleClose" title="关闭">
                <Icon name="x" :size="18" />
              </button>
            </div>

            <!-- Body -->
            <div class="content-body" :key="tutorial.currentStep">
              <Transition name="step-slide" mode="out-in">
                <!-- Step 0: Language -->
                <div v-if="tutorial.currentStep === 0" class="step-content" key="step-0">
                  <div class="step-header">
                    <div class="step-header-icon indigo"><Icon name="globe" :size="18" /></div>
                    <h3>设置语言</h3>
                    <p>选择语音识别（ASR）和 AI 回答的语言，确保后续交互的语言一致性。</p>
                  </div>
                  <div class="demo-card gradient-1">
                    <div class="demo-card-header">
                      <div class="demo-dot-group">
                        <span class="demo-dot red"></span>
                        <span class="demo-dot yellow"></span>
                        <span class="demo-dot green"></span>
                      </div>
                      <span class="demo-card-title">语言设置</span>
                    </div>
                    <div class="demo-card-body">
                      <div class="demo-field">
                        <label class="demo-label">🎤 语音识别语言</label>
                        <div class="demo-select-row">
                          <select class="demo-select" v-model="sttLang">
                            <option value="zh">中文</option>
                            <option value="en">英文</option>
                            <option value="auto">自动检测</option>
                          </select>
                        </div>
                      </div>
                      <div class="demo-field">
                        <label class="demo-label">🤖 AI 回答语言</label>
                        <div class="demo-select-row">
                          <select class="demo-select" v-model="llmLang">
                            <option value="zh">中文</option>
                            <option value="en">英文</option>
                            <option value="mix">中英混合</option>
                          </select>
                        </div>
                      </div>
                      <div class="demo-field">
                        <label class="demo-label">🌐 识别服务</label>
                        <div class="demo-select-row">
                          <select class="demo-select" v-model="sttService">
                            <option value="qwen_local">千问本地 Qwen3-ASR-Flash</option>
                            <option value="qwen_cloud">千问云端 Paraformer v2</option>
                            <option value="local_whisper">本地 Whisper</option>
                          </select>
                        </div>
                      </div>
                      <div class="demo-toast" v-if="langSaved">
                        <Icon name="check-circle" :size="14" />
                        <span>语言设置已保存</span>
                      </div>
                    </div>
                  </div>
                  <div class="step-tip">
                    <Icon name="info" :size="14" />
                    <span>提示：语音识别语言决定了录音转文字的准确性，建议与实际面试语言一致。</span>
                  </div>
                </div>

                <!-- Step 1: Resume + JD -->
                <div v-else-if="tutorial.currentStep === 1" class="step-content" key="step-1">
                  <div class="step-header">
                    <div class="step-header-icon emerald"><Icon name="file-text" :size="18" /></div>
                    <h3>添加简历和岗位描述</h3>
                    <p>上传你的简历 PDF / Markdown 文件，并粘贴目标岗位描述，让 AI 了解你的背景与目标。</p>
                  </div>
                  <div class="demo-card gradient-2">
                    <div class="demo-card-header">
                      <div class="demo-dot-group">
                        <span class="demo-dot red"></span>
                        <span class="demo-dot yellow"></span>
                        <span class="demo-dot green"></span>
                      </div>
                      <span class="demo-card-title">简历 & 岗位配置</span>
                    </div>
                    <div class="demo-card-body">
                      <div class="demo-upload-zone">
                        <Icon name="upload" :size="24" class="upload-icon" />
                        <span class="upload-text">点击或拖拽上传简历（PDF / Markdown）</span>
                        <span class="upload-hint">支持 .pdf, .md 格式</span>
                      </div>
                      <div class="demo-divider"></div>
                      <div class="demo-field">
                        <label class="demo-label">📋 岗位描述 (JD)</label>
                        <textarea
                          class="demo-textarea"
                          placeholder="粘贴目标岗位的 JD 内容..."
                          rows="3"
                          v-model="jdContent"
                        ></textarea>
                      </div>
                      <div class="demo-toast" v-if="resumeSaved">
                        <Icon name="check-circle" :size="14" />
                        <span>简历和岗位已就绪</span>
                      </div>
                    </div>
                  </div>
                  <div class="step-tip">
                    <Icon name="info" :size="14" />
                    <span>简历解析后，AI 将基于你的经历和岗位需求生成更具针对性的回答。</span>
                  </div>
                </div>

                <!-- Step 2: Knowledge Base -->
                <div v-else-if="tutorial.currentStep === 2" class="step-content" key="step-2">
                  <div class="step-header">
                    <div class="step-header-icon blue"><Icon name="database" :size="18" /></div>
                    <h3>添加知识和题库</h3>
                    <p>导入你的知识库和面试题库。支持 Markdown 格式文件，系统会自动建立索引。</p>
                  </div>
                  <div class="demo-card gradient-3">
                    <div class="demo-card-header">
                      <div class="demo-dot-group">
                        <span class="demo-dot red"></span>
                        <span class="demo-dot yellow"></span>
                        <span class="demo-dot green"></span>
                      </div>
                      <span class="demo-card-title">知识库管理</span>
                    </div>
                    <div class="demo-card-body">
                      <div class="kb-stats">
                        <div class="kb-stat-item">
                          <span class="kb-stat-value">42</span>
                          <span class="kb-stat-label">知识文件</span>
                        </div>
                        <div class="kb-stat-item">
                          <span class="kb-stat-value">168</span>
                          <span class="kb-stat-label">章节</span>
                        </div>
                        <div class="kb-stat-item">
                          <span class="kb-stat-value">3</span>
                          <span class="kb-stat-label">知识来源</span>
                        </div>
                      </div>
                      <div class="kb-source-list">
                        <div class="kb-source-item">
                          <Icon name="book-open" :size="14" />
                          <span>面试题库（通用）</span>
                          <span class="kb-source-badge">已接入</span>
                        </div>
                        <div class="kb-source-item">
                          <Icon name="folder" :size="14" />
                          <span>个人项目资料</span>
                          <span class="kb-source-badge">已接入</span>
                        </div>
                        <div class="kb-source-item">
                          <Icon name="code" :size="14" />
                          <span>技术笔记</span>
                          <span class="kb-source-badge">已接入</span>
                        </div>
                      </div>
                      <div class="demo-toast" v-if="kbSaved">
                        <Icon name="check-circle" :size="14" />
                        <span>知识来源已接入</span>
                      </div>
                    </div>
                  </div>
                  <div class="step-tip">
                    <Icon name="info" :size="14" />
                    <span>知识库中的内容会在对话时自动检索并注入上下文，让回答更精准。</span>
                  </div>
                </div>

                <!-- Step 3: Profile Optimization -->
                <div v-else-if="tutorial.currentStep === 3" class="step-content" key="step-3">
                  <div class="step-header">
                    <div class="step-header-icon purple"><Icon name="star" :size="18" /></div>
                    <h3>保存技能偏好</h3>
                    <p>配置你的岗位匹配偏好、表达风格和项目故事，让 AI 的回答更贴合你的个人特色。</p>
                  </div>
                  <div class="demo-card gradient-4">
                    <div class="demo-card-header">
                      <div class="demo-dot-group">
                        <span class="demo-dot red"></span>
                        <span class="demo-dot yellow"></span>
                        <span class="demo-dot green"></span>
                      </div>
                      <span class="demo-card-title">技能偏好设置</span>
                    </div>
                    <div class="demo-card-body">
                      <div class="profile-chips">
                        <div class="chip-row-label">岗位匹配</div>
                        <div class="chip-row">
                          <span class="chip active">Agent 开发</span>
                          <span class="chip">大模型应用</span>
                          <span class="chip">全栈开发</span>
                        </div>
                      </div>
                      <div class="profile-chips">
                        <div class="chip-row-label">表达风格</div>
                        <div class="chip-row">
                          <span class="chip active">结构化</span>
                          <span class="chip">简洁</span>
                          <span class="chip">STAR 法则</span>
                        </div>
                      </div>
                      <div class="profile-chips">
                        <div class="chip-row-label">项目故事</div>
                        <div class="chip-row">
                          <span class="chip active">智能客服系统</span>
                          <span class="chip">RAG 知识库</span>
                          <span class="chip">AI 面试助手</span>
                        </div>
                      </div>
                    </div>
                  </div>
                  <div class="step-tip">
                    <Icon name="info" :size="14" />
                    <span>这些偏好会作为 System Prompt 注入到每次对话中，确保回答风格一致。</span>
                  </div>
                </div>

                <!-- Step 4: Real-time Interview -->
                <div v-else-if="tutorial.currentStep === 4" class="step-content" key="step-4">
                  <div class="step-header">
                    <div class="step-header-icon pink"><Icon name="message-circle" :size="18" /></div>
                    <h3>用「提问」回答问题</h3>
                    <p>面试进行中，使用快捷键快速唤起 AI 助手，获取实时回答指引。</p>
                  </div>
                  <div class="demo-card gradient-5">
                    <div class="demo-card-header">
                      <div class="demo-dot-group">
                        <span class="demo-dot red"></span>
                        <span class="demo-dot yellow"></span>
                        <span class="demo-dot green"></span>
                      </div>
                      <span class="demo-card-title">实时面试辅助</span>
                    </div>
                    <div class="demo-card-body">
                      <div class="floating-window-preview">
                        <div class="floating-header">
                          <span class="floating-dot red"></span>
                          <span class="floating-dot yellow"></span>
                          <span class="floating-dot green"></span>
                          <span class="floating-title">AI 助手</span>
                        </div>
                        <div class="floating-body">
                          <div class="floating-msg assistant">
                            <span>你好！我是你的面试助手，有什么问题可以随时问我。</span>
                          </div>
                          <div class="floating-msg user">
                            <span>请解释 RAG 的工作原理</span>
                          </div>
                          <div class="floating-msg assistant typing">
                            <span class="typing-dot"></span>
                            <span class="typing-dot"></span>
                            <span class="typing-dot"></span>
                          </div>
                        </div>
                      </div>
                      <div class="hotkey-guide">
                        <div class="hotkey-item">
                          <kbd>Ctrl+Q</kbd>
                          <span>切换窗口</span>
                        </div>
                        <div class="hotkey-item">
                          <kbd>左 Alt</kbd>
                          <span>语音提问</span>
                        </div>
                        <div class="hotkey-item">
                          <kbd>Esc</kbd>
                          <span>隐藏窗口</span>
                        </div>
                      </div>
                      <div class="demo-toast" v-if="interviewReady">
                        <Icon name="check-circle" :size="14" />
                        <span>实时辅助已就绪 · 悬浮窗默认不进录屏</span>
                      </div>
                    </div>
                  </div>
                  <div class="step-tip">
                    <Icon name="info" :size="14" />
                    <span>悬浮窗采用隐身防御模式，不会出现在截屏或录屏中，保护你的隐私。</span>
                  </div>
                </div>

                <!-- Step 5: Screenshot Solving -->
                <div v-else-if="tutorial.currentStep === 5" class="step-content" key="step-5">
                  <div class="step-header">
                    <div class="step-header-icon orange"><Icon name="camera" :size="18" /></div>
                    <h3>用「截图」处理屏幕题</h3>
                    <p>遇到编程题或视觉题时，一键截图交给 AI 处理，支持代码识别、图文分析。</p>
                  </div>
                  <div class="demo-card gradient-6">
                    <div class="demo-card-header">
                      <div class="demo-dot-group">
                        <span class="demo-dot red"></span>
                        <span class="demo-dot yellow"></span>
                        <span class="demo-dot green"></span>
                      </div>
                      <span class="demo-card-title">截图解题</span>
                    </div>
                    <div class="demo-card-body">
                      <div class="screenshot-preview-demo">
                        <div class="screenshot-img-placeholder">
                          <Icon name="camera" :size="32" class="screenshot-icon" />
                          <span class="screenshot-hint">按 F8 截图</span>
                        </div>
                        <div class="screenshot-result-demo">
                          <div class="result-line code">
                            <span class="line-num">1</span>
                            <span class="line-code">function twoSum(nums, target) &#123;</span>
                          </div>
                          <div class="result-line code">
                            <span class="line-num">2</span>
                            <span class="line-code">&nbsp;&nbsp;const map = new Map()</span>
                          </div>
                          <div class="result-line code">
                            <span class="line-num">3</span>
                            <span class="line-code">&nbsp;&nbsp;for (let i = 0; i &lt; nums.length; i++) &#123;</span>
                          </div>
                          <div class="result-line code highlight">
                            <span class="line-num">4</span>
                            <span class="line-code">&nbsp;&nbsp;&nbsp;&nbsp;const complement = target - nums[i]</span>
                          </div>
                          <div class="result-line code">
                            <span class="line-num">5</span>
                            <span class="line-code">&nbsp;&nbsp;&nbsp;&nbsp;if (map.has(complement)) &#123;</span>
                          </div>
                          <div class="result-line code highlight">
                            <span class="line-num">6</span>
                            <span class="line-code">&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;return [map.get(complement), i]</span>
                          </div>
                          <div class="result-line code">
                            <span class="line-num">7</span>
                            <span class="line-code">&nbsp;&nbsp;&nbsp;&nbsp;&#125;</span>
                          </div>
                          <div class="result-line code">
                            <span class="line-num">8</span>
                            <span class="line-code">&nbsp;&nbsp;&nbsp;&nbsp;map.set(nums[i], i)</span>
                          </div>
                          <div class="result-line code">
                            <span class="line-num">9</span>
                            <span class="line-code">&nbsp;&nbsp;&#125;</span>
                          </div>
                        </div>
                      </div>
                      <div class="hotkey-guide">
                        <div class="hotkey-item">
                          <kbd>F8</kbd>
                          <span>截图解题</span>
                        </div>
                        <div class="hotkey-item">
                          <kbd>F6</kbd>
                          <span>追问截图</span>
                        </div>
                      </div>
                    </div>
                  </div>
                  <div class="step-tip">
                    <Icon name="info" :size="14" />
                    <span>支持多轮截图追问，每次截图都会结合之前的上下文给出更精准的回答。</span>
                  </div>
                </div>
              </Transition>
            </div>

            <!-- Footer Controls -->
            <div class="content-footer">
              <div class="footer-left">
                <button v-if="!tutorial.isFirstStep" class="btn-nav btn-prev" @click="tutorial.prevStep">
                  <Icon name="chevron-left" :size="16" />
                  <span>上一步</span>
                </button>
              </div>
              <div class="footer-right">
                <button
                  v-if="tutorial.isLastStep"
                  class="btn-nav btn-primary btn-complete"
                  @click="finishTutorial"
                >
                  <Icon name="check" :size="16" />
                  <span>完成教程</span>
                </button>
                <button v-else class="btn-nav btn-primary" @click="handleNext">
                  <span>{{ tutorial.isStepCompleted(tutorial.currentStep) ? '下一步' : '完成并继续' }}</span>
                  <Icon name="chevron-right" :size="16" />
                </button>
              </div>
            </div>
          </main>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import Icon from './Icon.vue'
import { useTutorialStore } from '../stores/tutorial'
import { useUIStore } from '../stores/ui'
import { useSettingsStore } from '../stores/settings'

const tutorial = useTutorialStore()
const ui = useUIStore()
const settingsStore = useSettingsStore()

// ── Interactive demo state ──────────────────────────────────
const sttLang = ref('zh')
const llmLang = ref('zh')
const sttService = ref('qwen_local')
const jdContent = ref('')

// ── Toast animations ────────────────────────────────────────
const langSaved = ref(false)
const resumeSaved = ref(false)
const kbSaved = ref(false)
const interviewReady = ref(false)

// ── Phase steps ─────────────────────────────────────────────
const phase1Steps = computed(() => tutorial.STEPS.filter(s => s.phase === 0))
const phase2Steps = computed(() => tutorial.STEPS.filter(s => s.phase === 1))

// ── Handlers ────────────────────────────────────────────────
function handleNext() {
  if (!tutorial.isStepCompleted(tutorial.currentStep)) {
    // Mark current as completed with visual feedback
    const step = tutorial.currentStep
    if (step === 0) langSaved.value = true
    else if (step === 1) resumeSaved.value = true
    else if (step === 2) kbSaved.value = true
    else if (step === 4) interviewReady.value = true

    tutorial.markCurrentCompleted()

    // Show toast feedback
    const stepMeta = tutorial.currentStepMeta
    ui.showToast(`✅ ${stepMeta.title} 已完成`, 'success', 1500)

    // Auto-advance after a brief delay for the visual feedback
    setTimeout(() => {
      if (tutorial.currentStep < tutorial.totalSteps - 1) {
        tutorial.nextStep()
      }
    }, 400)
  } else {
    tutorial.nextStep()
  }
}

function handleClose() {
  if (tutorial.allCompleted) {
    tutorial.hide()
  } else {
    ui.showToast('建议完成所有步骤以充分了解功能', 'info', 2000)
  }
}

function handleOverlayClick() {
  // Don't close on overlay click for modal
}

function finishTutorial() {
  if (!tutorial.isStepCompleted(tutorial.currentStep)) {
    tutorial.markCurrentCompleted()
  }
  tutorial.skipToEnd()
  ui.showToast('🎉 教程完成，祝你面试顺利！', 'success', 3000)
}
</script>

<style scoped>
/* ── Overlay ──────────────────────────────────────────────── */
.tutorial-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.75);
  backdrop-filter: blur(8px);
  pointer-events: auto;
}

.tutorial-modal {
  display: flex;
  width: 860px;
  max-width: 92vw;
  height: 640px;
  max-height: 88vh;
  background: rgba(20, 22, 30, 0.95);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 20px;
  overflow: hidden;
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.6), 0 0 0 1px rgba(255, 255, 255, 0.05);
}

/* ── Sidebar ──────────────────────────────────────────────── */
.tutorial-sidebar {
  width: 260px;
  flex-shrink: 0;
  background: rgba(15, 17, 23, 0.9);
  border-right: 1px solid rgba(255, 255, 255, 0.06);
  display: flex;
  flex-direction: column;
  padding: 24px 0 16px;
}

.sidebar-header {
  padding: 0 20px 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.sidebar-logo {
  width: 36px;
  height: 36px;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 12px;
}

.logo-icon {
  font-size: 18px;
  font-weight: 800;
  color: white;
}

.sidebar-title {
  font-size: 16px;
  font-weight: 700;
  color: #f3f4f6;
  line-height: 1.4;
  margin: 0 0 16px;
}

.sidebar-progress {
  display: flex;
  align-items: center;
  gap: 10px;
}

.progress-bar-track {
  flex: 1;
  height: 4px;
  background: rgba(255, 255, 255, 0.08);
  border-radius: 2px;
  overflow: hidden;
}

.progress-bar-fill {
  height: 100%;
  background: linear-gradient(90deg, #6366f1, #8b5cf6);
  border-radius: 2px;
  transition: width 0.5s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.progress-text {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.4);
  font-weight: 600;
  white-space: nowrap;
}

/* ── Navigation ────────────────────────────────────────────── */
.sidebar-nav {
  flex: 1;
  overflow-y: auto;
  padding: 16px 8px;
  scrollbar-width: thin;
  scrollbar-color: rgba(255,255,255,0.08) transparent;
}

.phase-section {
  margin-bottom: 20px;
}

.phase-section:last-child {
  margin-bottom: 0;
}

.phase-label {
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 1.2px;
  color: rgba(255, 255, 255, 0.25);
  padding: 0 12px;
  margin-bottom: 8px;
}

.step-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 10px 12px;
  border: none;
  border-radius: 10px;
  background: transparent;
  cursor: pointer;
  transition: all 0.2s ease;
  text-align: left;
  position: relative;
}

.step-item:hover:not(.locked) {
  background: rgba(255, 255, 255, 0.05);
}

.step-item.active {
  background: rgba(99, 102, 241, 0.12);
  box-shadow: inset 0 0 0 1px rgba(99, 102, 241, 0.25);
}

.step-item.locked {
  opacity: 0.35;
  cursor: not-allowed;
}

.step-item.completed {
  opacity: 0.7;
}

.step-indicator {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  font-size: 12px;
  font-weight: 700;
  transition: all 0.3s ease;
}

.step-item:not(.active):not(.completed) .step-indicator {
  background: rgba(255, 255, 255, 0.08);
  color: rgba(255, 255, 255, 0.35);
}

.step-item.active .step-indicator {
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  color: white;
}

.step-item.completed .step-indicator {
  background: rgba(16, 185, 129, 0.2);
  color: #10b981;
}

.step-num {
  font-size: 11px;
}

.step-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.step-title {
  font-size: 13px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.85);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.step-item.active .step-title {
  color: white;
}

.step-item.completed .step-title {
  color: rgba(255, 255, 255, 0.6);
}

.step-subtitle {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.35);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ── Sidebar Footer ────────────────────────────────────────── */
.sidebar-footer {
  padding: 12px 20px 0;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
}

.btn-skip {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  background: transparent;
  color: rgba(255, 255, 255, 0.4);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
  width: 100%;
  justify-content: center;
}

.btn-skip:hover {
  background: rgba(255, 255, 255, 0.05);
  color: rgba(255, 255, 255, 0.6);
}

/* ── Main Content ──────────────────────────────────────────── */
.tutorial-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.content-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 24px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  flex-shrink: 0;
}

.content-breadcrumb {
  display: flex;
  align-items: center;
  gap: 10px;
}

.phase-tag {
  font-size: 11px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.5);
  letter-spacing: 0.5px;
}

.step-counter {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.3);
  background: rgba(255, 255, 255, 0.06);
  padding: 2px 10px;
  border-radius: 10px;
  font-weight: 600;
}

.btn-close {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  border: none;
  background: transparent;
  color: rgba(255, 255, 255, 0.4);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.btn-close:hover {
  background: rgba(255, 255, 255, 0.08);
  color: rgba(255, 255, 255, 0.7);
}

/* ── Content Body ──────────────────────────────────────────── */
.content-body {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
  scrollbar-width: thin;
  scrollbar-color: rgba(255,255,255,0.08) transparent;
}

.step-content {
  animation: content-fade-in 0.3s ease;
}

@keyframes content-fade-in {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}

/* ── Step Header ──────────────────────────────────────────── */
.step-header {
  margin-bottom: 20px;
}

.step-header-icon {
  width: 40px;
  height: 40px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 12px;
}

.step-header-icon.indigo {
  background: rgba(99, 102, 241, 0.15);
  color: #818cf8;
}
.step-header-icon.emerald {
  background: rgba(16, 185, 129, 0.15);
  color: #34d399;
}
.step-header-icon.blue {
  background: rgba(59, 130, 246, 0.15);
  color: #60a5fa;
}
.step-header-icon.purple {
  background: rgba(139, 92, 246, 0.15);
  color: #a78bfa;
}
.step-header-icon.pink {
  background: rgba(236, 72, 153, 0.15);
  color: #f472b6;
}
.step-header-icon.orange {
  background: rgba(249, 115, 22, 0.15);
  color: #fb923c;
}

.step-header h3 {
  font-size: 20px;
  font-weight: 700;
  color: white;
  margin: 0 0 6px;
}

.step-header p {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.5);
  line-height: 1.6;
  margin: 0;
}

/* ── Demo Card ─────────────────────────────────────────────── */
.demo-card {
  border-radius: 14px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  overflow: hidden;
  margin-bottom: 16px;
  backdrop-filter: blur(20px);
}

.demo-card.gradient-1 {
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.12), rgba(139, 92, 246, 0.08));
}
.demo-card.gradient-2 {
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.12), rgba(5, 150, 105, 0.08));
}
.demo-card.gradient-3 {
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.12), rgba(37, 99, 235, 0.08));
}
.demo-card.gradient-4 {
  background: linear-gradient(135deg, rgba(139, 92, 246, 0.12), rgba(192, 132, 252, 0.08));
}
.demo-card.gradient-5 {
  background: linear-gradient(135deg, rgba(236, 72, 153, 0.12), rgba(244, 114, 182, 0.08));
}
.demo-card.gradient-6 {
  background: linear-gradient(135deg, rgba(249, 115, 22, 0.12), rgba(251, 146, 60, 0.08));
}

.demo-card-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.demo-dot-group {
  display: flex;
  gap: 5px;
}

.demo-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.demo-dot.red { background: #ef4444; }
.demo-dot.yellow { background: #eab308; }
.demo-dot.green { background: #22c55e; }

.demo-card-title {
  font-size: 12px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.5);
}

.demo-card-body {
  padding: 16px;
}

/* ── Form Elements ─────────────────────────────────────────── */
.demo-field {
  margin-bottom: 14px;
}

.demo-field:last-child {
  margin-bottom: 0;
}

.demo-label {
  display: block;
  font-size: 12px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.6);
  margin-bottom: 6px;
}

.demo-select-row {
  position: relative;
}

.demo-select {
  width: 100%;
  padding: 8px 12px;
  background: rgba(0, 0, 0, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  color: rgba(255, 255, 255, 0.8);
  font-size: 13px;
  font-family: inherit;
  outline: none;
  cursor: pointer;
  appearance: none;
  transition: border-color 0.2s;
}

.demo-select:focus {
  border-color: rgba(99, 102, 241, 0.5);
}

.demo-select option {
  background: #1f2937;
  color: white;
}

.demo-textarea {
  width: 100%;
  padding: 10px 12px;
  background: rgba(0, 0, 0, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  color: rgba(255, 255, 255, 0.8);
  font-size: 13px;
  font-family: inherit;
  outline: none;
  resize: vertical;
  transition: border-color 0.2s;
  box-sizing: border-box;
}

.demo-textarea:focus {
  border-color: rgba(16, 185, 129, 0.5);
}

.demo-textarea::placeholder {
  color: rgba(255, 255, 255, 0.2);
}

/* ── Upload Zone ───────────────────────────────────────────── */
.demo-upload-zone {
  border: 2px dashed rgba(255, 255, 255, 0.1);
  border-radius: 10px;
  padding: 24px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.demo-upload-zone:hover {
  border-color: rgba(16, 185, 129, 0.4);
  background: rgba(16, 185, 129, 0.05);
}

.upload-icon {
  color: rgba(255, 255, 255, 0.3);
}

.upload-text {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.6);
  font-weight: 600;
}

.upload-hint {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.25);
}

.demo-divider {
  height: 1px;
  background: rgba(255, 255, 255, 0.06);
  margin: 16px 0;
}

/* ── Toast ──────────────────────────────────────────────────── */
.demo-toast {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  background: rgba(16, 185, 129, 0.1);
  border: 1px solid rgba(16, 185, 129, 0.2);
  border-radius: 8px;
  color: #34d399;
  font-size: 13px;
  font-weight: 600;
  margin-top: 14px;
  animation: toast-enter 0.3s ease;
}

@keyframes toast-enter {
  from { opacity: 0; transform: translateY(-6px); }
  to { opacity: 1; transform: translateY(0); }
}

/* ── KB Stats ──────────────────────────────────────────────── */
.kb-stats {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.kb-stat-item {
  flex: 1;
  background: rgba(0, 0, 0, 0.2);
  border-radius: 10px;
  padding: 12px;
  text-align: center;
}

.kb-stat-value {
  display: block;
  font-size: 20px;
  font-weight: 800;
  color: white;
  margin-bottom: 2px;
}

.kb-stat-label {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.35);
}

.kb-source-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.kb-source-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  background: rgba(0, 0, 0, 0.2);
  border-radius: 8px;
  font-size: 13px;
  color: rgba(255, 255, 255, 0.7);
}

.kb-source-badge {
  margin-left: auto;
  font-size: 10px;
  font-weight: 600;
  color: #34d399;
  background: rgba(16, 185, 129, 0.1);
  padding: 2px 8px;
  border-radius: 6px;
}

/* ── Profile Chips ──────────────────────────────────────────── */
.profile-chips {
  margin-bottom: 14px;
}

.profile-chips:last-child {
  margin-bottom: 0;
}

.chip-row-label {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.4);
  font-weight: 600;
  margin-bottom: 6px;
}

.chip-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.chip {
  padding: 5px 12px;
  background: rgba(0, 0, 0, 0.25);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.5);
  transition: all 0.2s;
}

.chip.active {
  background: rgba(139, 92, 246, 0.15);
  border-color: rgba(139, 92, 246, 0.3);
  color: #a78bfa;
}

/* ── Floating Window Preview ───────────────────────────────── */
.floating-window-preview {
  background: rgba(0, 0, 0, 0.35);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 12px;
  overflow: hidden;
  margin-bottom: 14px;
  backdrop-filter: blur(10px);
}

.floating-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background: rgba(0, 0, 0, 0.3);
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.floating-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
}

.floating-dot.red { background: #ef4444; }
.floating-dot.yellow { background: #eab308; }
.floating-dot.green { background: #22c55e; }

.floating-title {
  margin-left: 6px;
  font-size: 11px;
  color: rgba(255, 255, 255, 0.4);
  font-weight: 600;
}

.floating-body {
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.floating-msg {
  max-width: 85%;
  padding: 8px 12px;
  border-radius: 10px;
  font-size: 12px;
  line-height: 1.5;
}

.floating-msg.assistant {
  background: rgba(255, 255, 255, 0.06);
  color: rgba(255, 255, 255, 0.7);
  align-self: flex-start;
  border-bottom-left-radius: 4px;
}

.floating-msg.user {
  background: rgba(99, 102, 241, 0.2);
  color: rgba(255, 255, 255, 0.8);
  align-self: flex-end;
  border-bottom-right-radius: 4px;
}

.floating-msg.typing {
  display: flex;
  gap: 3px;
  align-items: center;
  background: rgba(255, 255, 255, 0.06);
  align-self: flex-start;
  border-bottom-left-radius: 4px;
  padding: 10px 14px;
}

.typing-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.4);
  animation: typing-bounce 1.4s infinite ease-in-out both;
}

.typing-dot:nth-child(1) { animation-delay: -0.32s; }
.typing-dot:nth-child(2) { animation-delay: -0.16s; }

@keyframes typing-bounce {
  0%, 80%, 100% { transform: scale(0.6); opacity: 0.3; }
  40% { transform: scale(1); opacity: 0.8; }
}

/* ── Hotkey Guide ──────────────────────────────────────────── */
.hotkey-guide {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.hotkey-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  background: rgba(0, 0, 0, 0.2);
  border-radius: 8px;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.5);
}

.hotkey-item kbd {
  padding: 2px 6px;
  background: rgba(255, 255, 255, 0.08);
  border-radius: 4px;
  font-family: var(--font-mono, 'Cascadia Code', monospace);
  font-size: 11px;
  color: rgba(255, 255, 255, 0.7);
  border: 1px solid rgba(255, 255, 255, 0.1);
}

/* ── Screenshot Preview ────────────────────────────────────── */
.screenshot-preview-demo {
  display: flex;
  gap: 12px;
  margin-bottom: 14px;
}

.screenshot-img-placeholder {
  width: 140px;
  flex-shrink: 0;
  background: rgba(0, 0, 0, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 20px;
}

.screenshot-icon {
  color: rgba(255, 255, 255, 0.2);
}

.screenshot-hint {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.3);
  font-weight: 600;
}

.screenshot-result-demo {
  flex: 1;
  background: rgba(0, 0, 0, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 10px;
  overflow: hidden;
  font-family: var(--font-mono, 'Cascadia Code', monospace);
  font-size: 11px;
  line-height: 1.7;
}

.result-line {
  display: flex;
  padding: 1px 8px;
}

.result-line.code {
  color: rgba(255, 255, 255, 0.5);
}

.result-line.highlight {
  background: rgba(99, 102, 241, 0.1);
  color: rgba(255, 255, 255, 0.8);
}

.line-num {
  width: 20px;
  flex-shrink: 0;
  color: rgba(255, 255, 255, 0.15);
  text-align: right;
  margin-right: 12px;
  user-select: none;
}

.line-code {
  white-space: pre;
}

/* ── Step Tip ──────────────────────────────────────────────── */
.step-tip {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 10px 14px;
  background: rgba(99, 102, 241, 0.06);
  border: 1px solid rgba(99, 102, 241, 0.12);
  border-radius: 10px;
  font-size: 12px;
  line-height: 1.5;
  color: rgba(255, 255, 255, 0.45);
}

/* ── Footer ────────────────────────────────────────────────── */
.content-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 24px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  flex-shrink: 0;
}

.footer-left,
.footer-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.btn-nav {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border-radius: 9px;
  border: none;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
  font-family: inherit;
}

.btn-prev {
  background: rgba(255, 255, 255, 0.06);
  color: rgba(255, 255, 255, 0.6);
}

.btn-prev:hover {
  background: rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.8);
}

.btn-primary {
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  color: white;
}

.btn-primary:hover {
  box-shadow: 0 4px 20px rgba(99, 102, 241, 0.4);
  transform: translateY(-1px);
}

.btn-complete {
  background: linear-gradient(135deg, #10b981, #059669);
}

.btn-complete:hover {
  box-shadow: 0 4px 20px rgba(16, 185, 129, 0.4);
}

/* ── Transitions ───────────────────────────────────────────── */
.tutorial-fade-enter-active {
  transition: all 0.35s cubic-bezier(0.16, 1, 0.3, 1);
}
.tutorial-fade-leave-active {
  transition: all 0.25s ease-in;
}
.tutorial-fade-enter-from,
.tutorial-fade-leave-to {
  opacity: 0;
}
.tutorial-fade-enter-from .tutorial-modal,
.tutorial-fade-leave-to .tutorial-modal {
  transform: scale(0.92) translateY(20px);
  opacity: 0;
}

.step-slide-enter-active {
  transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}
.step-slide-leave-active {
  transition: all 0.2s ease-in;
}
.step-slide-enter-from {
  opacity: 0;
  transform: translateX(30px);
}
.step-slide-leave-to {
  opacity: 0;
  transform: translateX(-20px);
}

/* ── Scrollbar ─────────────────────────────────────────────── */
.content-body::-webkit-scrollbar,
.sidebar-nav::-webkit-scrollbar {
  width: 4px;
}
.content-body::-webkit-scrollbar-thumb,
.sidebar-nav::-webkit-scrollbar-thumb {
  background: rgba(255,255,255,0.08);
  border-radius: 2px;
}
.content-body::-webkit-scrollbar-track,
.sidebar-nav::-webkit-scrollbar-track {
  background: transparent;
}
</style>
