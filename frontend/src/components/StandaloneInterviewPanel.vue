<template>
  <div class="standalone-root">
    <!-- 独立面试面板：三栏布局 -->
    <header class="sa-topbar">
      <div class="sa-left">
        <span class="sa-brand">AI 辅助面试</span>
      </div>
      <div class="sa-right">
        <button class="sa-btn" @click="chatStore.startNewConversation" title="创建新对话">
          <Icon name="plus" :size="14" />
        </button>
        <button class="sa-btn" @click="showHistory = !showHistory" title="历史对话">
          <Icon name="clock" :size="14" />
        </button>
        <button class="sa-btn sa-close" @click="closeWindow" title="关闭窗口">
          <Icon name="x" :size="16" />
        </button>
      </div>
    </header>

    <!-- 历史面板 -->
    <Transition name="slide-fade">
      <div v-if="showHistory" class="sa-history-panel">
        <div class="sa-hp-header"><span>历史对话</span></div>
        <div class="sa-hp-body">
          <div v-if="chatStore.savedConversations.length === 0" class="sa-hp-empty">暂无历史对话</div>
          <div v-for="conv in chatStore.savedConversations" :key="conv.id" class="sa-hp-item"
            :class="{ active: conv.id === chatStore.activeConversationId }"
            @click="selectHistory(conv.id)">
            <div class="sa-hp-title">{{ conv.title }}</div>
            <div class="sa-hp-meta">{{ conv.messageCount }} 条</div>
          </div>
        </div>
      </div>
    </Transition>

    <!-- 三栏内容 -->
    <main class="sa-main">
      <section class="sa-col sa-col-transcript">
        <div class="sa-col-header">对话转录</div>
        <div class="sa-col-body" ref="transcriptRef">
          <div v-if="transcripts.length === 0" class="sa-col-empty">
            <p>面试对话将实时显示</p>
          </div>
          <div v-for="(t, i) in transcripts" :key="i" class="sa-bubble" :class="t.speaker">
            <div class="sa-bubble-label">{{ t.speaker === 'interviewer' ? '面试官' : '我' }}</div>
            <div class="sa-bubble-text">{{ t.content }}</div>
          </div>
        </div>
      </section>

      <section class="sa-col sa-col-agent">
        <div class="sa-col-header">
          <span>AI 智能体</span>
          <span v-if="chatStore.isLoading" class="sa-thinking">思考中...</span>
        </div>
        <div class="sa-col-body" ref="agentRef">
          <div v-if="chatStore.messages.length === 0" class="sa-col-empty">
            <p>AI 建议将显示在此处</p>
          </div>
          <div v-for="(msg, i) in chatStore.messages" :key="i" class="sa-agent-msg" :class="msg.role">
            <div v-if="msg.role === 'user'" class="sa-user-msg">{{ msg.content }}</div>
            <div v-else class="sa-ai-block">
              <div class="sa-ai-header">
                <span class="sa-ai-tag"><Icon name="sparkles" :size="10" /> AI 回答</span>
                <span v-if="msg._streaming" class="sa-streaming">生成中...</span>
              </div>
              <div class="sa-ai-content" v-html="renderMarkdown(msg.content)"></div>
            </div>
          </div>
        </div>
      </section>

      <section class="sa-col sa-col-input">
        <div class="sa-quick-actions">
          <button class="sa-qa" @click="inputText = '请解释这道题的思路'">💡 解题思路</button>
          <button class="sa-qa" @click="inputText = '请帮我优化这段代码'">💻 代码优化</button>
          <button class="sa-qa" @click="inputText = '用 STAR 法则回答'">⭐ STAR 回答</button>
        </div>
        <div class="sa-input-area">
          <textarea v-model="inputText" class="sa-input" placeholder="输入你的问题..." rows="3"
            @keydown.enter.exact="sendMessage"></textarea>
        </div>
        <div class="sa-controls">
          <button v-if="chatStore.isLoading" class="sa-stop-btn" @click="stopThinking">
            <Icon name="square" :size="12" /> 停止
          </button>
          <button class="sa-send-btn" :disabled="!inputText.trim() || chatStore.isLoading" @click="sendMessage">
            <Icon name="send" :size="14" /> 发送
          </button>
        </div>
      </section>
    </main>
  </div>
</template>

<script setup>
import { ref, nextTick, onMounted, onUnmounted, watch } from 'vue'
import Icon from './Icon.vue'
import { useChatStore } from '../stores/chat'
import { renderMarkdownWithLatex } from '../utils/markdown-latex'
import { CancelRunningTask } from '../../wailsjs/go/app/App'
import { Quit, EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

const chatStore = useChatStore()
const inputText = ref('')
const transcriptRef = ref(null)
const agentRef = ref(null)
const showHistory = ref(false)
const transcripts = ref([])
const streamingText = ref('')

function renderMarkdown(text) { return text ? renderMarkdownWithLatex(text) : '' }

function closeWindow() { Quit() }

function selectHistory(id) {
  chatStore.loadConversation(id)
  showHistory.value = false
}

function addTranscript(speaker, content) {
  transcripts.value.push({
    speaker, content,
    time: new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
    timestamp: Date.now(),
  })
}

async function sendMessage() {
  const text = inputText.value.trim()
  if (!text || chatStore.isLoading) return
  inputText.value = ''
  await chatStore.sendMessage(text)
}

async function stopThinking() {
  try { await CancelRunningTask(); chatStore.isLoading = false }
  catch (e) { console.error(e) }
}

onMounted(() => {
  EventsOn('chat-stream-chunk', chunk => chatStore.handleStreamChunk(chunk))
  EventsOn('chat-stream-done', () => chatStore.handleStreamDone())
  EventsOn('chat-stream-error', error => chatStore.handleStreamError(error))
})

onUnmounted(() => {
  EventsOff('chat-stream-chunk')
  EventsOff('chat-stream-done')
  EventsOff('chat-stream-error')
})

watch(() => chatStore.messages.length, () => {
  nextTick(() => { if (agentRef.value) agentRef.value.scrollTop = agentRef.value.scrollHeight })
})
watch(() => transcripts.value.length, () => {
  nextTick(() => { if (transcriptRef.value) transcriptRef.value.scrollTop = transcriptRef.value.scrollHeight })
})
</script>

<style scoped>
.standalone-root {
  width: 100vw;
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: rgba(20, 22, 30, 0.92);
  overflow: hidden;
}

.sa-topbar {
  display: flex; align-items: center; justify-content: space-between;
  padding: 8px 14px; height: 38px; flex-shrink: 0;
  background: rgba(15, 17, 23, 0.8);
  border-bottom: 1px solid rgba(255,255,255,0.06);
}
.sa-left { display: flex; align-items: center; gap: 8px; }
.sa-brand { font-size: 14px; font-weight: 700; color: rgba(255,255,255,0.9); }
.sa-right { display: flex; align-items: center; gap: 4px; }
.sa-btn {
  width: 28px; height: 28px; border: none; border-radius: 6px;
  background: transparent; color: rgba(255,255,255,0.35);
  cursor: pointer; display: flex; align-items: center; justify-content: center;
}
.sa-btn:hover { background: rgba(255,255,255,0.06); color: rgba(255,255,255,0.6); }
.sa-close:hover { background: rgba(239,68,68,0.1); color: #ef4444; }

/* 历史面板 */
.sa-history-panel {
  position: fixed; top: 46px; right: 10px;
  width: 220px; max-height: 300px;
  background: rgba(24,26,35,0.96);
  border: 1px solid rgba(255,255,255,0.08);
  border-radius: 10px; box-shadow: 0 12px 40px rgba(0,0,0,0.5);
  z-index: 100; overflow: hidden;
}
.sa-hp-header { padding: 8px 12px; font-size: 12px; font-weight: 600; color: rgba(255,255,255,0.5); border-bottom: 1px solid rgba(255,255,255,0.06); }
.sa-hp-body { max-height: 260px; overflow-y: auto; padding: 4px; }
.sa-hp-empty { padding: 20px; text-align: center; font-size: 12px; color: rgba(255,255,255,0.2); }
.sa-hp-item { padding: 8px 10px; border-radius: 6px; cursor: pointer; }
.sa-hp-item:hover { background: rgba(255,255,255,0.04); }
.sa-hp-item.active { background: rgba(99,102,241,0.08); }
.sa-hp-title { font-size: 12px; font-weight: 600; color: rgba(255,255,255,0.7); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sa-hp-meta { font-size: 10px; color: rgba(255,255,255,0.2); }

/* 三栏 */
.sa-main {
  flex: 1; display: grid;
  grid-template-columns: 1fr 1.6fr 1fr;
  gap: 1px; overflow: hidden;
}
.sa-col {
  display: flex; flex-direction: column;
  min-width: 0; min-height: 0; overflow: hidden;
  background: rgba(0,0,0,0.15);
}
.sa-col-header {
  display: flex; align-items: center; gap: 6px;
  padding: 8px 12px; font-size: 11px; font-weight: 700;
  color: rgba(255,255,255,0.35); text-transform: uppercase;
  letter-spacing: 0.5px;
  border-bottom: 1px solid rgba(255,255,255,0.04); flex-shrink: 0;
}
.sa-thinking {
  margin-left: auto; font-size: 10px; text-transform: none;
  color: #818cf8; background: rgba(99,102,241,0.1);
  padding: 2px 8px; border-radius: 5px; font-weight: 600;
}
.sa-col-body {
  flex: 1; overflow-y: auto; padding: 8px;
  scrollbar-width: thin;
}
.sa-col-body::-webkit-scrollbar { width: 3px; }
.sa-col-body::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.06); border-radius: 2px; }
.sa-col-empty { display: flex; align-items: center; justify-content: center; height: 100%; color: rgba(255,255,255,0.15); font-size: 13px; }

/* 转录气泡 */
.sa-bubble { padding: 6px 8px; margin-bottom: 4px; border-radius: 6px; border-left: 2px solid transparent; }
.sa-bubble.interviewer { border-left-color: rgba(255,255,255,0.15); }
.sa-bubble.me { border-left-color: rgba(99,102,241,0.4); background: rgba(99,102,241,0.04); }
.sa-bubble-label { font-size: 10px; color: rgba(255,255,255,0.25); margin-bottom: 2px; }
.sa-bubble-text { font-size: 13px; color: rgba(255,255,255,0.7); line-height: 1.5; }

/* AI 消息 */
.sa-agent-msg { margin-bottom: 8px; }
.sa-user-msg { text-align: right; padding: 6px 10px; background: rgba(99,102,241,0.08); border-radius: 6px; font-size: 13px; color: rgba(255,255,255,0.8); }
.sa-ai-block { border: 1px solid rgba(139,92,246,0.12); border-radius: 6px; overflow: hidden; }
.sa-ai-header { display: flex; justify-content: space-between; padding: 4px 8px; background: rgba(139,92,246,0.06); border-bottom: 1px solid rgba(139,92,246,0.08); }
.sa-ai-tag { display: flex; align-items: center; gap: 4px; font-size: 10px; font-weight: 700; color: #a78bfa; }
.sa-streaming { font-size: 10px; color: rgba(255,255,255,0.25); animation: pulse 1s infinite; }
.sa-ai-content { padding: 8px 10px; font-size: 13px; line-height: 1.7; color: rgba(255,255,255,0.85); word-break: break-word; }

/* 输入区 */
.sa-quick-actions { display: flex; flex-wrap: wrap; gap: 4px; padding: 6px 8px; border-bottom: 1px solid rgba(255,255,255,0.04); }
.sa-qa { padding: 3px 8px; background: rgba(255,255,255,0.04); border: 1px solid rgba(255,255,255,0.06); border-radius: 5px; color: rgba(255,255,255,0.45); font-size: 11px; cursor: pointer; }
.sa-qa:hover { background: rgba(255,255,255,0.07); color: rgba(255,255,255,0.65); }

.sa-input-area { flex: 1; padding: 8px; display: flex; flex-direction: column; }
.sa-input { flex: 1; padding: 8px 10px; background: rgba(0,0,0,0.35); border: 1px solid rgba(255,255,255,0.06); border-radius: 8px; color: rgba(255,255,255,0.8); font-size: 13px; font-family: inherit; outline: none; resize: none; }

.sa-controls { display: flex; align-items: center; gap: 6px; padding: 6px 8px; border-top: 1px solid rgba(255,255,255,0.04); }
.sa-stop-btn { display: flex; align-items: center; gap: 4px; padding: 4px 10px; background: rgba(239,68,68,0.12); border: 1px solid rgba(239,68,68,0.2); border-radius: 6px; color: #ef4444; font-size: 11px; font-weight: 600; cursor: pointer; }
.sa-send-btn { margin-left: auto; display: flex; align-items: center; gap: 4px; padding: 5px 12px; background: linear-gradient(135deg,#6366f1,#8b5cf6); border: none; border-radius: 6px; color: white; font-size: 12px; font-weight: 600; cursor: pointer; }
.sa-send-btn:disabled { opacity: 0.4; cursor: not-allowed; }

.slide-fade-enter-active { transition: all 0.2s ease; }
.slide-fade-leave-active { transition: all 0.15s ease; }
.slide-fade-enter-from, .slide-fade-leave-to { opacity: 0; transform: translateY(-4px); }

@keyframes pulse { 0%,100% { opacity: 1; } 50% { opacity: 0.3; } }
</style>
