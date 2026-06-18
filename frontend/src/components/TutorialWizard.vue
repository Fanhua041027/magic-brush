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
                <!-- ══════ Step 0: 设置语言 ══════ -->
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
                          <select class="demo-select" v-model="settingsStore.tempSettings.sttLanguage">
                            <option value="zh">中文</option>
                            <option value="en">英文</option>
                            <option value="auto">自动检测</option>
                          </select>
                        </div>
                      </div>
                      <div class="demo-field">
                        <label class="demo-label">🌐 识别服务</label>
                        <div class="demo-select-row">
                          <select class="demo-select" v-model="settingsStore.tempSettings.sttService">
                            <option value="qwen_local">千问本地 Qwen3-ASR-Flash（推荐）</option>
                            <option value="qwen_cloud">千问云端 Paraformer v2</option>
                            <option value="local_whisper">本地 Whisper</option>
                          </select>
                        </div>
                      </div>
                      <div class="demo-field">
                        <label class="demo-label">🔊 音频输入</label>
                        <div class="demo-select-row">
                          <select class="demo-select" v-model="audioInputType">
                            <option value="mic">🎤 麦克风</option>
                            <option value="stereo_mix">🔊 立体声混音（电脑内部音频）</option>
                          </select>
                        </div>
                      </div>
                      <div class="demo-field">
                        <label class="demo-label">🤖 AI 回答语言</label>
                        <div class="language-chips">
                          <span
                            class="chip"
                            :class="{ active: settingsStore.tempSettings.sttLanguage === 'zh' || settingsStore.tempSettings.sttLanguage === 'auto' }"
                          >中文</span>
                          <span class="chip">英文</span>
                          <span class="chip">中英混合</span>
                        </div>
                        <p class="demo-hint">AI 会根据你的问题语言自动匹配回答语言。</p>
                      </div>
                      <transition name="quick-fade">
                        <div v-if="langSaved" class="demo-toast">
                          <Icon name="check-circle" :size="14" />
                          <span>语言设置已保存</span>
                        </div>
                      </transition>
                    </div>
                  </div>
                  <div class="step-tip">
                    <Icon name="info" :size="14" />
                    <span>提示：语音识别语言决定了录音转文字的准确性，建议与实际面试语言一致。</span>
                  </div>
                </div>

                <!-- ══════ Step 1: 简历和岗位描述 ══════ -->
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
                      <!-- 简历区域 -->
                      <div class="demo-field">
                        <label class="demo-label">📄 简历文件</label>
                        <div v-if="!settingsStore.settings.resumePath" class="resume-upload-zone" @click="selectResume">
                          <Icon name="upload" :size="22" class="upload-icon" />
                          <span class="upload-text">点击选择简历文件</span>
                          <span class="upload-hint">支持 PDF / Markdown 格式</span>
                        </div>
                        <div v-else class="resume-file-info">
                          <div class="file-info-row">
                            <Icon name="file" :size="16" />
                            <span class="file-name">{{ settingsStore.settings.resumePath.split(/[/\\]/).pop() }}</span>
                            <span class="file-badge">{{ settingsStore.settings.resumeContent ? '已解析' : '待解析' }}</span>
                          </div>
                          <div class="file-actions-row">
                            <button class="btn-small" @click="selectResume">
                              <Icon name="refresh" :size="12" />
                              <span>更换</span>
                            </button>
                            <button v-if="settingsStore.settings.resumePath && !settingsStore.settings.resumeContent" class="btn-small btn-accent-small" @click="parseResume" :disabled="parsingResume">
                              <Icon name="loader" :size="12" :spinning="parsingResume" />
                              <span>{{ parsingResume ? '解析中...' : '解析简历' }}</span>
                            </button>
                          </div>
                        </div>
                      </div>

                      <!-- API Key 检查 -->
                      <div v-if="!settingsStore.settings.apiKey" class="api-warning">
                        <Icon name="alert-triangle" :size="14" />
                        <span>请先在设置中配置 API Key 后才能解析简历</span>
                        <button class="btn-small" @click="goToSettings">去设置</button>
                      </div>

                      <div class="demo-divider"></div>

                      <!-- 岗位描述 -->
                      <div class="demo-field">
                        <label class="demo-label">📋 岗位描述 (JD)</label>
                        <textarea
                          class="demo-textarea"
                          placeholder="粘贴目标岗位的 JD 内容，让 AI 了解岗位要求..."
                          rows="4"
                          v-model="jdInput"
                        ></textarea>
                      </div>

                      <transition name="quick-fade">
                        <div v-if="resumeSaved" class="demo-toast">
                          <Icon name="check-circle" :size="14" />
                          <span>{{ settingsStore.settings.resumeContent ? '简历和 JD 已就绪' : 'JD 内容已保存' }}</span>
                        </div>
                      </transition>
                    </div>
                  </div>
                  <div class="step-tip">
                    <Icon name="info" :size="14" />
                    <span>简历解析后，AI 将基于你的经历和岗位需求生成更具针对性的回答。</span>
                  </div>
                </div>

                <!-- ══════ Step 2: 知识和题库 ══════ -->
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
                      <!-- 知识库状态 -->
                      <div v-if="kbLoading" class="kb-loading">
                        <Icon name="loader" :size="18" :spinning="true" />
                        <span>正在加载知识库状态...</span>
                      </div>
                      <template v-else>
                        <div class="kb-stats">
                          <div class="kb-stat-item">
                            <span class="kb-stat-value">{{ kbStatus.fileCount ?? '--' }}</span>
                            <span class="kb-stat-label">知识文件</span>
                          </div>
                          <div class="kb-stat-item">
                            <span class="kb-stat-value">{{ kbStatus.sectionCount ?? '--' }}</span>
                            <span class="kb-stat-label">章节</span>
                          </div>
                          <div class="kb-stat-item">
                            <span class="kb-stat-value" :class="kbStatus.ready ? 'text-green' : 'text-muted'">
                              {{ kbStatus.ready ? '就绪' : '未导入' }}
                            </span>
                            <span class="kb-stat-label">状态</span>
                          </div>
                        </div>

                        <div class="demo-field">
                          <label class="demo-label">📂 知识库目录</label>
                          <div v-if="kbStatus.kbPath" class="kb-path-display">
                            <Icon name="folder" :size="14" />
                            <span class="kb-path-text">{{ kbStatus.kbPath }}</span>
                          </div>
                          <button class="btn-kb-import" @click="importKnowledgeBase">
                            <Icon :name="kbStatus.ready ? 'refresh' : 'download'" :size="16" />
                            <span>{{ kbStatus.ready ? '重新导入' : '选择知识库目录' }}</span>
                          </button>
                          <button v-if="kbStatus.ready" class="btn-kb-clear" @click="clearKnowledgeBase">
                            <Icon name="trash" :size="14" />
                            <span>清除</span>
                          </button>
                        </div>
                      </template>

                      <transition name="quick-fade">
                        <div v-if="kbSaved" class="demo-toast">
                          <Icon name="check-circle" :size="14" />
                          <span>知识库已导入 · {{ kbStatus.fileCount }} 个文件</span>
                        </div>
                      </transition>
                    </div>
                  </div>
                  <div class="step-tip">
                    <Icon name="info" :size="14" />
                    <span>知识库中的内容会在对话时自动检索并注入上下文，让回答更精准。</span>
                  </div>
                </div>

                <!-- ══════ Step 3: 保存技能偏好 ══════ -->
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
                      <!-- 岗位匹配 -->
                      <div class="profile-chips">
                        <div class="chip-row-label">🎯 目标岗位</div>
                        <div class="chip-row">
                          <span
                            v-for="role in roleOptions"
                            :key="role.value"
                            class="chip"
                            :class="{ active: selectedRole === role.value }"
                            @click="selectedRole = role.value"
                          >{{ role.label }}</span>
                        </div>
                        <input
                          v-if="selectedRole === 'custom'"
                          class="chip-input"
                          v-model="customRole"
                          placeholder="输入自定义岗位名称..."
                          @input="updatePromptContext"
                        />
                      </div>

                      <!-- 表达风格 -->
                      <div class="profile-chips">
                        <div class="chip-row-label">🎨 表达风格</div>
                        <div class="chip-row">
                          <span
                            v-for="style in styleOptions"
                            :key="style.value"
                            class="chip"
                            :class="{ active: selectedStyle === style.value }"
                            @click="selectedStyle = style.value"
                          >{{ style.label }}</span>
                        </div>
                      </div>

                      <!-- 自定义 Prompt -->
                      <div class="profile-chips">
                        <div class="chip-row-label">✏️ 自定义指令 (System Prompt)</div>
                        <textarea
                          class="demo-textarea"
                          v-model="customPrompt"
                          placeholder="例如：请用 STAR 法则回答问题，突出我的项目贡献..."
                          rows="3"
                        ></textarea>
                      </div>

                      <transition name="quick-fade">
                        <div v-if="profileSaved" class="demo-toast">
                          <Icon name="check-circle" :size="14" />
                          <span>技能偏好已保存</span>
                        </div>
                      </transition>
                    </div>
                  </div>
                  <div class="step-tip">
                    <Icon name="info" :size="14" />
                    <span>这些偏好会作为 System Prompt 注入到每次对话中，确保回答风格一致。</span>
                  </div>
                </div>

                <!-- ══════ Step 4: 实时面试 ══════ -->
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
                          <kbd>F7</kbd>
                          <span>打开对话</span>
                        </div>
                        <div class="hotkey-item">
                          <kbd>左 Alt</kbd>
                          <span>语音提问</span>
                        </div>
                        <div class="hotkey-item">
                          <kbd>F9</kbd>
                          <span>隐藏窗口</span>
                        </div>
                      </div>
                      <transition name="quick-fade">
                        <div v-if="interviewReady" class="demo-toast">
                          <Icon name="check-circle" :size="14" />
                          <span>实时辅助已就绪 · 悬浮窗默认不进录屏</span>
                        </div>
                      </transition>
                    </div>
                  </div>
                  <div class="step-tip">
                    <Icon name="info" :size="14" />
                    <span>悬浮窗采用隐身防御模式，不会出现在截屏或录屏中，保护你的隐私。</span>
                  </div>
                </div>

                <!-- ══════ Step 5: 截图解题 ══════ -->
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
                  <span>{{ tutorial.isStepCompleted(tutorial.currentStep) ? '下一步' : '保存并继续' }}</span>
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
import { ref, watch, computed, onMounted, reactive, nextTick } from 'vue'
import Icon from './Icon.vue'
import { useTutorialStore } from '../stores/tutorial'
import { useUIStore } from '../stores/ui'
import { useSettingsStore } from '../stores/settings'
import { useVoiceStore } from '../stores/voice'
import { api } from '../services/api'

const tutorial = useTutorialStore()
const ui = useUIStore()
const settingsStore = useSettingsStore()
const voiceStore = useVoiceStore()

// ── Phase steps ─────────────────────────────────────────────
const phase1Steps = computed(() => tutorial.STEPS.filter(s => s.phase === 0))
const phase2Steps = computed(() => tutorial.STEPS.filter(s => s.phase === 1))

// ── Step 0: 语言设置 ──────────────────────────────────────
const audioInputType = ref('mic')
const langSaved = ref(false)

// ── Step 1: 简历 + JD ──────────────────────────────────────
const jdInput = ref('')
const resumeSaved = ref(false)
const parsingResume = ref(false)

// ── Step 2: 知识库 ────────────────────────────────────────
const kbStatus = reactive({ ready: false, fileCount: 0, sectionCount: 0, kbPath: '' })
const kbLoading = ref(false)
const kbSaved = ref(false)

// ── Step 3: 技能偏好 ──────────────────────────────────────
const roleOptions = [
  { value: 'agent-dev', label: '🤖 Agent 开发' },
  { value: 'llm-app', label: '🧠 大模型应用' },
  { value: 'fullstack', label: '🌐 全栈开发' },
  { value: 'algorithm', label: '📐 算法工程师' },
  { value: 'custom', label: '✏️ 自定义' },
]
const styleOptions = [
  { value: 'structured', label: '结构化' },
  { value: 'concise', label: '简洁' },
  { value: 'star', label: 'STAR 法则' },
  { value: 'detailed', label: '详细分析' },
]
const selectedRole = ref('agent-dev')
const customRole = ref('')
const selectedStyle = ref('structured')
const customPrompt = ref('')
const profileSaved = ref(false)

// ── Step 4: 面试辅助 ──────────────────────────────────────
const interviewReady = ref(false)

// ── 初始化 ─────────────────────────────────────────────────
onMounted(() => {
  // 从已有设置中恢复值
  if (settingsStore.settings.prompt) {
    customPrompt.value = settingsStore.settings.prompt
  }
  // 从已有设置中恢复音频设备选择
  if (settingsStore.settings.sttDevice) {
    audioInputType.value = settingsStore.settings.sttDevice
  }
  // 加载知识库状态
  refreshKBStatus()
  // 加载音频设备列表
  voiceStore.loadDevices()
})

// ── 方法 ───────────────────────────────────────────────────
async function selectResume() {
  try {
    await api.selectResume()
    // 选择后自动解析
    if (settingsStore.settings.apiKey) {
      await parseResume()
    }
  } catch (e) {
    console.error('选择简历失败', e)
  }
}

async function parseResume() {
  if (!settingsStore.settings.apiKey) {
    ui.showToast('请先在设置中配置 API Key', 'warning')
    return
  }
  parsingResume.value = true
  try {
    await api.parseResume()
    ui.showToast('简历解析完成', 'success')
  } catch (e) {
    console.error('解析简历失败', e)
    ui.showToast('简历解析失败', 'error')
  } finally {
    parsingResume.value = false
  }
}

function goToSettings() {
  if (!ui.showSettings) settingsStore.openSettings()
  ui.activeTab = 'api'
}

async function importKnowledgeBase() {
  try {
    const result = await api.selectKBDirectory()
    if (result) {
      await refreshKBStatus()
      ui.showToast('知识库导入成功', 'success')
    }
  } catch (e) {
    console.error('导入知识库失败', e)
  }
}

async function clearKnowledgeBase() {
  try {
    await api.clearKB()
    kbStatus.ready = false
    kbStatus.fileCount = 0
    kbStatus.sectionCount = 0
    kbStatus.kbPath = ''
    ui.showToast('知识库已清除', 'info')
  } catch (e) {
    console.error('清除知识库失败', e)
  }
}

async function refreshKBStatus() {
  kbLoading.value = true
  try {
    const status = await api.getKBStatus()
    if (status) {
      kbStatus.ready = status.ready || false
      kbStatus.fileCount = status.file_count || status.fileCount || 0
      kbStatus.sectionCount = status.section_count || status.sectionCount || 0
      kbStatus.kbPath = status.kb_path || status.kbPath || ''
    }
  } catch (e) {
    console.error('获取知识库状态失败', e)
  } finally {
    kbLoading.value = false
  }
}

function updatePromptContext() {
  // 当自定义岗位输入时更新
}

async function applyAudioDevice(type) {
  // 根据音频输入类型切换设备
  try {
    // 先加载最新设备列表
    await voiceStore.loadDevices()
    const devices = voiceStore.devices
    // 找到对应类型的设备
    const target = devices.find(d => d.type === type)
    if (target) {
      voiceStore.selectedDeviceId = target.id
      await voiceStore.changeDevice()
      console.log(`[Audio] Switched to device: ${target.name} (${type})`)
    } else {
      console.warn(`[Audio] No device found for type: ${type}`)
    }
  } catch (e) {
    console.error('[Audio] Failed to switch device:', e)
  }
}

async function saveStepSettings() {
  // 构造包含角色和风格信息的 prompt 前缀
  const roleName = selectedRole === 'custom' ? customRole.value : roleOptions.find(r => r.value === selectedRole)?.label || ''
  const styleName = styleOptions.find(s => s.value === selectedStyle)?.label || ''
  let combinedPrompt = customPrompt.value || ''
  const prefixParts = []
  if (roleName) prefixParts.push(`目标岗位：${roleName}`)
  if (styleName) prefixParts.push(`表达风格：${styleName}`)
  if (prefixParts.length > 0) {
    combinedPrompt = prefixParts.join('；') + '\n' + combinedPrompt
  }

  // 追加 JD 内容
  if (jdInput.value) {
    const jdSection = `\n\n【岗位描述】\n${jdInput.value}`
    if (!combinedPrompt.includes('【岗位描述】')) {
      combinedPrompt += jdSection
    }
  }

  // 直接修改 settings 并持久化
  settingsStore.settings.prompt = combinedPrompt.trim()
  try {
    await settingsStore.saveSettingsSilent()
    // 同步到 tempSettings 保持一致性
    settingsStore.tempSettings.prompt = settingsStore.settings.prompt
    return true
  } catch (e) {
    console.error('保存设置失败', e)
    return false
  }
}

function handleNext() {
  if (!tutorial.isStepCompleted(tutorial.currentStep)) {
    const step = tutorial.currentStep

    if (step === 0) {
      // 同步 tempSettings 到 settings 并持久化
      settingsStore.settings.sttLanguage = settingsStore.tempSettings.sttLanguage
      settingsStore.settings.sttService = settingsStore.tempSettings.sttService
      settingsStore.settings.sttDevice = audioInputType.value
      // 应用音频设备切换
      applyAudioDevice(audioInputType.value).then(() => {
        return settingsStore.saveSettingsSilent()
      }).then(() => {
        langSaved.value = true
        proceedNext(step)
      })
    } else if (step === 1) {
      // 保存 JD 内容
      saveStepSettings().then(() => {
        resumeSaved.value = true
        proceedNext(step)
      })
    } else if (step === 2) {
      kbSaved.value = true
      proceedNext(step)
    } else if (step === 3) {
      saveStepSettings().then(() => {
        profileSaved.value = true
        proceedNext(step)
      })
    } else if (step === 4) {
      interviewReady.value = true
      proceedNext(step)
    } else {
      proceedNext(step)
    }
  } else {
    tutorial.nextStep()
  }
}

function proceedNext(step) {
  tutorial.markCurrentCompleted()
  const stepMeta = tutorial.currentStepMeta
  ui.showToast(`✅ ${stepMeta.title} 已完成`, 'success', 1500)

  setTimeout(() => {
    if (tutorial.currentStep < tutorial.totalSteps - 1) {
      tutorial.nextStep()
    }
  }, 400)
}

function handleClose() {
  if (tutorial.allCompleted) {
    tutorial.hide()
  } else {
    ui.showToast('建议完成所有步骤以充分了解功能', 'info', 2000)
  }
}

function handleOverlayClick() {}

function finishTutorial() {
  saveStepSettings().then(() => {
    if (!tutorial.isStepCompleted(tutorial.currentStep)) {
      tutorial.markCurrentCompleted()
    }
    tutorial.skipToEnd()
    ui.showToast('🎉 教程完成，祝你面试顺利！', 'success', 3000)
  })
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
.phase-section:last-child { margin-bottom: 0; }

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
}
.step-item:hover:not(.locked) { background: rgba(255, 255, 255, 0.05); }
.step-item.active { background: rgba(99, 102, 241, 0.12); box-shadow: inset 0 0 0 1px rgba(99, 102, 241, 0.25); }
.step-item.locked { opacity: 0.35; cursor: not-allowed; }
.step-item.completed { opacity: 0.7; }

.step-indicator {
  width: 24px; height: 24px;
  border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
  font-size: 12px; font-weight: 700;
  transition: all 0.3s ease;
}
.step-item:not(.active):not(.completed) .step-indicator { background: rgba(255,255,255,0.08); color: rgba(255,255,255,0.35); }
.step-item.active .step-indicator { background: linear-gradient(135deg, #6366f1, #8b5cf6); color: white; }
.step-item.completed .step-indicator { background: rgba(16,185,129,0.2); color: #10b981; }
.step-num { font-size: 11px; }

.step-info { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.step-title { font-size: 13px; font-weight: 600; color: rgba(255,255,255,0.85); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.step-item.active .step-title { color: white; }
.step-item.completed .step-title { color: rgba(255,255,255,0.6); }
.step-subtitle { font-size: 11px; color: rgba(255,255,255,0.35); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

/* ── Sidebar Footer ────────────────────────────────────────── */
.sidebar-footer {
  padding: 12px 20px 0;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
}
.btn-skip {
  display: flex; align-items: center; gap: 6px;
  padding: 8px 14px; border: 1px solid rgba(255,255,255,0.1); border-radius: 8px;
  background: transparent; color: rgba(255,255,255,0.4); font-size: 12px;
  cursor: pointer; transition: all 0.2s ease; width: 100%; justify-content: center;
}
.btn-skip:hover { background: rgba(255,255,255,0.05); color: rgba(255,255,255,0.6); }

/* ── Main Content ──────────────────────────────────────────── */
.tutorial-content { flex: 1; display: flex; flex-direction: column; min-width: 0; }
.content-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 16px 24px; border-bottom: 1px solid rgba(255,255,255,0.06); flex-shrink: 0;
}
.content-breadcrumb { display: flex; align-items: center; gap: 10px; }
.phase-tag { font-size: 11px; font-weight: 600; color: rgba(255,255,255,0.5); letter-spacing: 0.5px; }
.step-counter { font-size: 12px; color: rgba(255,255,255,0.3); background: rgba(255,255,255,0.06); padding: 2px 10px; border-radius: 10px; font-weight: 600; }
.btn-close {
  width: 32px; height: 32px; border-radius: 8px; border: none;
  background: transparent; color: rgba(255,255,255,0.4);
  cursor: pointer; display: flex; align-items: center; justify-content: center; transition: all 0.2s ease;
}
.btn-close:hover { background: rgba(255,255,255,0.08); color: rgba(255,255,255,0.7); }

/* ── Content Body ──────────────────────────────────────────── */
.content-body {
  flex: 1; overflow-y: auto; padding: 24px;
  scrollbar-width: thin; scrollbar-color: rgba(255,255,255,0.08) transparent;
}
.step-content { animation: content-fade-in 0.3s ease; }
@keyframes content-fade-in {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}

/* ── Step Header ──────────────────────────────────────────── */
.step-header { margin-bottom: 20px; }
.step-header-icon {
  width: 40px; height: 40px; border-radius: 12px;
  display: flex; align-items: center; justify-content: center; margin-bottom: 12px;
}
.step-header-icon.indigo { background: rgba(99,102,241,0.15); color: #818cf8; }
.step-header-icon.emerald { background: rgba(16,185,129,0.15); color: #34d399; }
.step-header-icon.blue { background: rgba(59,130,246,0.15); color: #60a5fa; }
.step-header-icon.purple { background: rgba(139,92,246,0.15); color: #a78bfa; }
.step-header-icon.pink { background: rgba(236,72,153,0.15); color: #f472b6; }
.step-header-icon.orange { background: rgba(249,115,22,0.15); color: #fb923c; }
.step-header h3 { font-size: 20px; font-weight: 700; color: white; margin: 0 0 6px; }
.step-header p { font-size: 13px; color: rgba(255,255,255,0.5); line-height: 1.6; margin: 0; }

/* ── Demo Card ─────────────────────────────────────────────── */
.demo-card {
  border-radius: 14px; border: 1px solid rgba(255,255,255,0.08);
  overflow: hidden; margin-bottom: 16px; backdrop-filter: blur(20px);
}
.demo-card.gradient-1 { background: linear-gradient(135deg, rgba(99,102,241,0.12), rgba(139,92,246,0.08)); }
.demo-card.gradient-2 { background: linear-gradient(135deg, rgba(16,185,129,0.12), rgba(5,150,105,0.08)); }
.demo-card.gradient-3 { background: linear-gradient(135deg, rgba(59,130,246,0.12), rgba(37,99,235,0.08)); }
.demo-card.gradient-4 { background: linear-gradient(135deg, rgba(139,92,246,0.12), rgba(192,132,252,0.08)); }
.demo-card.gradient-5 { background: linear-gradient(135deg, rgba(236,72,153,0.12), rgba(244,114,182,0.08)); }
.demo-card.gradient-6 { background: linear-gradient(135deg, rgba(249,115,22,0.12), rgba(251,146,60,0.08)); }
.demo-card-header {
  display: flex; align-items: center; gap: 10px;
  padding: 12px 16px; border-bottom: 1px solid rgba(255,255,255,0.06);
}
.demo-dot-group { display: flex; gap: 5px; }
.demo-dot { width: 8px; height: 8px; border-radius: 50%; }
.demo-dot.red { background: #ef4444; }
.demo-dot.yellow { background: #eab308; }
.demo-dot.green { background: #22c55e; }
.demo-card-title { font-size: 12px; font-weight: 600; color: rgba(255,255,255,0.5); }
.demo-card-body { padding: 16px; }

/* ── Form Elements ─────────────────────────────────────────── */
.demo-field { margin-bottom: 14px; }
.demo-field:last-child { margin-bottom: 0; }
.demo-label { display: block; font-size: 12px; font-weight: 600; color: rgba(255,255,255,0.6); margin-bottom: 6px; }
.demo-select-row { position: relative; }
.demo-select {
  width: 100%; padding: 8px 12px; background: rgba(0,0,0,0.3);
  border: 1px solid rgba(255,255,255,0.1); border-radius: 8px;
  color: rgba(255,255,255,0.8); font-size: 13px; font-family: inherit;
  outline: none; cursor: pointer; appearance: none;
  transition: border-color 0.2s;
}
.demo-select:focus { border-color: rgba(99,102,241,0.5); }
.demo-select option { background: #1f2937; color: white; }

.demo-textarea {
  width: 100%; padding: 10px 12px; box-sizing: border-box;
  background: rgba(0,0,0,0.3); border: 1px solid rgba(255,255,255,0.1);
  border-radius: 8px; color: rgba(255,255,255,0.8); font-size: 13px;
  font-family: inherit; outline: none; resize: vertical; transition: border-color 0.2s;
}
.demo-textarea:focus { border-color: rgba(99,102,241,0.5); }
.demo-textarea::placeholder { color: rgba(255,255,255,0.2); }

.demo-hint { font-size: 11px; color: rgba(255,255,255,0.25); margin-top: 4px; }

/* ── Language Chips ────────────────────────────────────────── */
.language-chips { display: flex; gap: 6px; margin-bottom: 4px; }
.chip {
  padding: 5px 12px; background: rgba(0,0,0,0.25);
  border: 1px solid rgba(255,255,255,0.08); border-radius: 8px;
  font-size: 12px; color: rgba(255,255,255,0.5); cursor: default;
}
.chip.active { background: rgba(99,102,241,0.15); border-color: rgba(99,102,241,0.3); color: #818cf8; }

/* ── Resume Upload ─────────────────────────────────────────── */
.resume-upload-zone {
  border: 2px dashed rgba(255,255,255,0.1); border-radius: 10px;
  padding: 20px; text-align: center; cursor: pointer;
  display: flex; flex-direction: column; align-items: center; gap: 6px;
  transition: all 0.2s;
}
.resume-upload-zone:hover { border-color: rgba(16,185,129,0.4); background: rgba(16,185,129,0.05); }
.upload-icon { color: rgba(255,255,255,0.3); }
.upload-text { font-size: 13px; color: rgba(255,255,255,0.6); font-weight: 600; }
.upload-hint { font-size: 11px; color: rgba(255,255,255,0.25); }

.resume-file-info {
  background: rgba(0,0,0,0.2); border-radius: 8px; padding: 12px;
}
.file-info-row { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
.file-name { font-size: 13px; color: rgba(255,255,255,0.7); flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.file-badge { font-size: 10px; font-weight: 600; padding: 2px 8px; border-radius: 6px; }
.file-badge:contains(已解析) { background: rgba(16,185,129,0.1); color: #34d399; }
.file-badge { background: rgba(255,255,255,0.06); color: rgba(255,255,255,0.4); }
.file-actions-row { display: flex; gap: 6px; }

.api-warning {
  display: flex; align-items: center; gap: 8px;
  padding: 8px 12px; background: rgba(245,158,11,0.08);
  border: 1px solid rgba(245,158,11,0.15); border-radius: 8px;
  font-size: 12px; color: #f59e0b; margin-top: 8px;
}

.demo-divider { height: 1px; background: rgba(255,255,255,0.06); margin: 16px 0; }

/* ── Toast ──────────────────────────────────────────────────── */
.demo-toast {
  display: flex; align-items: center; gap: 8px;
  padding: 10px 14px; background: rgba(16,185,129,0.1);
  border: 1px solid rgba(16,185,129,0.2); border-radius: 8px;
  color: #34d399; font-size: 13px; font-weight: 600; margin-top: 14px;
}

/* ── KB Section ──────────────────────────────────────────────── */
.kb-loading { display: flex; align-items: center; gap: 10px; justify-content: center; padding: 20px; color: rgba(255,255,255,0.4); font-size: 13px; }
.kb-stats { display: flex; gap: 12px; margin-bottom: 16px; }
.kb-stat-item { flex: 1; background: rgba(0,0,0,0.2); border-radius: 10px; padding: 12px; text-align: center; }
.kb-stat-value { display: block; font-size: 20px; font-weight: 800; color: white; margin-bottom: 2px; }
.kb-stat-value.text-green { color: #34d399; }
.kb-stat-value.text-muted { color: rgba(255,255,255,0.3); }
.kb-stat-label { font-size: 11px; color: rgba(255,255,255,0.35); }
.kb-path-display { display: flex; align-items: center; gap: 6px; padding: 6px 10px; background: rgba(0,0,0,0.15); border-radius: 6px; margin-bottom: 8px; font-size: 11px; color: rgba(255,255,255,0.35); overflow: hidden; }
.kb-path-text { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.btn-kb-import {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 8px 16px; background: rgba(59,130,246,0.15); border: 1px solid rgba(59,130,246,0.25);
  border-radius: 8px; color: #60a5fa; font-size: 13px; font-weight: 600; cursor: pointer; transition: all 0.2s; font-family: inherit;
}
.btn-kb-import:hover { background: rgba(59,130,246,0.25); }
.btn-kb-clear {
  display: inline-flex; align-items: center; gap: 4px;
  padding: 6px 12px; background: transparent; border: 1px solid rgba(255,255,255,0.08);
  border-radius: 6px; color: rgba(255,255,255,0.3); font-size: 12px; cursor: pointer; margin-left: 6px; font-family: inherit;
}
.btn-kb-clear:hover { background: rgba(239,68,68,0.1); color: #ef4444; border-color: rgba(239,68,68,0.2); }

/* ── Profile Chips ──────────────────────────────────────────── */
.profile-chips { margin-bottom: 14px; }
.profile-chips:last-child { margin-bottom: 0; }
.chip-row-label { font-size: 11px; color: rgba(255,255,255,0.4); font-weight: 600; margin-bottom: 6px; }
.chip-row { display: flex; flex-wrap: wrap; gap: 6px; }
.chip { padding: 5px 12px; background: rgba(0,0,0,0.25); border: 1px solid rgba(255,255,255,0.08); border-radius: 8px; font-size: 12px; color: rgba(255,255,255,0.5); cursor: pointer; transition: all 0.2s; }
.chip:hover { border-color: rgba(255,255,255,0.15); }
.chip.active { background: rgba(139,92,246,0.15); border-color: rgba(139,92,246,0.3); color: #a78bfa; }
.chip-input {
  width: 100%; padding: 6px 10px; box-sizing: border-box; margin-top: 6px;
  background: rgba(0,0,0,0.25); border: 1px solid rgba(255,255,255,0.08);
  border-radius: 6px; color: rgba(255,255,255,0.7); font-size: 12px; font-family: inherit; outline: none;
}

/* ── Floating Window Preview ───────────────────────────────── */
.floating-window-preview {
  background: rgba(0,0,0,0.35); border: 1px solid rgba(255,255,255,0.08);
  border-radius: 12px; overflow: hidden; margin-bottom: 14px; backdrop-filter: blur(10px);
}
.floating-header {
  display: flex; align-items: center; gap: 6px;
  padding: 8px 12px; background: rgba(0,0,0,0.3); border-bottom: 1px solid rgba(255,255,255,0.05);
}
.floating-dot { width: 7px; height: 7px; border-radius: 50%; }
.floating-dot.red { background: #ef4444; }
.floating-dot.yellow { background: #eab308; }
.floating-dot.green { background: #22c55e; }
.floating-title { margin-left: 6px; font-size: 11px; color: rgba(255,255,255,0.4); font-weight: 600; }
.floating-body { padding: 12px; display: flex; flex-direction: column; gap: 8px; }
.floating-msg { max-width: 85%; padding: 8px 12px; border-radius: 10px; font-size: 12px; line-height: 1.5; }
.floating-msg.assistant { background: rgba(255,255,255,0.06); color: rgba(255,255,255,0.7); align-self: flex-start; border-bottom-left-radius: 4px; }
.floating-msg.user { background: rgba(99,102,241,0.2); color: rgba(255,255,255,0.8); align-self: flex-end; border-bottom-right-radius: 4px; }
.floating-msg.typing { display: flex; gap: 3px; align-items: center; background: rgba(255,255,255,0.06); align-self: flex-start; border-bottom-left-radius: 4px; padding: 10px 14px; }
.typing-dot { width: 5px; height: 5px; border-radius: 50%; background: rgba(255,255,255,0.4); animation: typing-bounce 1.4s infinite ease-in-out both; }
.typing-dot:nth-child(1) { animation-delay: -0.32s; }
.typing-dot:nth-child(2) { animation-delay: -0.16s; }
@keyframes typing-bounce { 0%,80%,100% { transform: scale(0.6); opacity: 0.3; } 40% { transform: scale(1); opacity: 0.8; } }

/* ── Hotkey Guide ──────────────────────────────────────────── */
.hotkey-guide { display: flex; gap: 8px; flex-wrap: wrap; }
.hotkey-item { display: flex; align-items: center; gap: 6px; padding: 6px 10px; background: rgba(0,0,0,0.2); border-radius: 8px; font-size: 12px; color: rgba(255,255,255,0.5); }
.hotkey-item kbd { padding: 2px 6px; background: rgba(255,255,255,0.08); border-radius: 4px; font-family: var(--font-mono, monospace); font-size: 11px; color: rgba(255,255,255,0.7); border: 1px solid rgba(255,255,255,0.1); }

/* ── Screenshot Preview ────────────────────────────────────── */
.screenshot-preview-demo { display: flex; gap: 12px; margin-bottom: 14px; }
.screenshot-img-placeholder {
  width: 140px; flex-shrink: 0; background: rgba(0,0,0,0.3);
  border: 1px solid rgba(255,255,255,0.08); border-radius: 10px;
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 8px; padding: 20px;
}
.screenshot-icon { color: rgba(255,255,255,0.2); }
.screenshot-hint { font-size: 12px; color: rgba(255,255,255,0.3); font-weight: 600; }
.screenshot-result-demo {
  flex: 1; background: rgba(0,0,0,0.3); border: 1px solid rgba(255,255,255,0.08);
  border-radius: 10px; overflow: hidden; font-family: var(--font-mono, monospace); font-size: 11px; line-height: 1.7;
}
.result-line { display: flex; padding: 1px 8px; }
.result-line.code { color: rgba(255,255,255,0.5); }
.result-line.highlight { background: rgba(99,102,241,0.1); color: rgba(255,255,255,0.8); }
.line-num { width: 20px; flex-shrink: 0; color: rgba(255,255,255,0.15); text-align: right; margin-right: 12px; user-select: none; }
.line-code { white-space: pre; }

/* ── Step Tip ──────────────────────────────────────────────── */
.step-tip { display: flex; align-items: flex-start; gap: 8px; padding: 10px 14px; background: rgba(99,102,241,0.06); border: 1px solid rgba(99,102,241,0.12); border-radius: 10px; font-size: 12px; line-height: 1.5; color: rgba(255,255,255,0.45); }

/* ── Footer ────────────────────────────────────────────────── */
.content-footer { display: flex; align-items: center; justify-content: space-between; padding: 14px 24px; border-top: 1px solid rgba(255,255,255,0.06); flex-shrink: 0; }
.footer-left, .footer-right { display: flex; align-items: center; gap: 8px; }
.btn-nav { display: flex; align-items: center; gap: 6px; padding: 8px 16px; border-radius: 9px; border: none; font-size: 13px; font-weight: 600; cursor: pointer; transition: all 0.2s ease; font-family: inherit; }
.btn-prev { background: rgba(255,255,255,0.06); color: rgba(255,255,255,0.6); }
.btn-prev:hover { background: rgba(255,255,255,0.1); color: rgba(255,255,255,0.8); }
.btn-primary { background: linear-gradient(135deg, #6366f1, #8b5cf6); color: white; }
.btn-primary:hover { box-shadow: 0 4px 20px rgba(99,102,241,0.4); transform: translateY(-1px); }
.btn-complete { background: linear-gradient(135deg, #10b981, #059669); }
.btn-complete:hover { box-shadow: 0 4px 20px rgba(16,185,129,0.4); }

/* ── Utility Buttons ────────────────────────────────────────── */
.btn-small { display: inline-flex; align-items: center; gap: 4px; padding: 4px 10px; background: rgba(255,255,255,0.06); border: 1px solid rgba(255,255,255,0.08); border-radius: 6px; color: rgba(255,255,255,0.5); font-size: 11px; cursor: pointer; transition: all 0.2s; font-family: inherit; }
.btn-small:hover { background: rgba(255,255,255,0.1); }
.btn-accent-small { background: rgba(99,102,241,0.15); border-color: rgba(99,102,241,0.25); color: #818cf8; }
.btn-accent-small:hover { background: rgba(99,102,241,0.25); }

/* ── Transitions ───────────────────────────────────────────── */
.tutorial-fade-enter-active { transition: all 0.35s cubic-bezier(0.16, 1, 0.3, 1); }
.tutorial-fade-leave-active { transition: all 0.25s ease-in; }
.tutorial-fade-enter-from, .tutorial-fade-leave-to { opacity: 0; }
.tutorial-fade-enter-from .tutorial-modal, .tutorial-fade-leave-to .tutorial-modal { transform: scale(0.92) translateY(20px); opacity: 0; }

.step-slide-enter-active { transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1); }
.step-slide-leave-active { transition: all 0.2s ease-in; }
.step-slide-enter-from { opacity: 0; transform: translateX(30px); }
.step-slide-leave-to { opacity: 0; transform: translateX(-20px); }

.quick-fade-enter-active { transition: all 0.25s ease; }
.quick-fade-leave-active { transition: all 0.15s ease; }
.quick-fade-enter-from, .quick-fade-leave-to { opacity: 0; transform: translateY(-4px); }

/* ── Scrollbar ─────────────────────────────────────────────── */
.content-body::-webkit-scrollbar, .sidebar-nav::-webkit-scrollbar { width: 4px; }
.content-body::-webkit-scrollbar-thumb, .sidebar-nav::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.08); border-radius: 2px; }
.content-body::-webkit-scrollbar-track, .sidebar-nav::-webkit-scrollbar-track { background: transparent; }
</style>
