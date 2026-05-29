<template>
  <div id="main-interface" class="main-interface" :class="{ visible: ui.mainVisible }">
    <!-- Left Panel - History -->
    <div class="left-panel" id="history-list" :class="{ collapsed: isHistoryCollapsed }">
      <div class="panel-header">
        <span v-if="!isHistoryCollapsed" class="panel-title">历史记录</span>
        <button class="toggle-btn" @click="isHistoryCollapsed = !isHistoryCollapsed"
          :title="isHistoryCollapsed ? '展开' : '收起'">
          <Icon :name="isHistoryCollapsed ? 'panel-right' : 'panel-left'" :size="14" />
        </button>
      </div>
      <div class="history-content" v-show="!isHistoryCollapsed">
        <div v-if="solution.history.length === 0" class="history-empty">
          <Icon name="file" :size="20" class="empty-icon" />
          <span class="empty-text">暂无记录</span>
        </div>
        <HistoryItem v-for="(h, idx) in solution.history" :key="idx"
          :summary="solution.getSummary(h)"
          :time="h.time"
          :isActive="idx === solution.activeHistoryIndex"
          :isFirst="idx === 0"
          :previewHtml="solution.renderMarkdown(solution.getSummary(h))"
          :roundsCount="solution.getRoundsCount(h)"
          @select="solution.selectHistory(idx)"
          @delete="solution.deleteHistory(idx)"
          @export-image="solution.exportImage(idx)" />
      </div>
    </div>

    <!-- Right Panel - Content -->
    <div class="right-panel">
      <!-- Empty state -->
      <EmptyState v-if="solution.history.length === 0 && !solution.isLoading && !solution.errorState.show"
        :shortcut="settingsStore.solveShortcut" :sendShortcut="settingsStore.sendShortcut" />

      <!-- Error state -->
      <ErrorView v-else-if="solution.errorState.show"
        :errorState="solution.errorState"
        :solveShortcut="settingsStore.solveShortcut" />

      <!-- Loading state (initial) -->
      <LoadingView v-else-if="solution.isLoading && !solution.isThinking && solution.currentRounds.length <= 1 && !hasAnyContent" />

      <!-- Content area -->
      <div v-else id="content" class="content-area md-body">
        <template v-for="(round, idx) in solution.currentRounds" :key="idx">
          <!-- Round divider (only for multi-round, shown between rounds) -->
          <RoundDivider v-if="idx > 0" :round="idx + 1" />

          <div class="chat-round">
            <!-- Thinking chain -->
            <ThinkingBlock
              v-if="round.thinking"
              :thinking="round.thinking"
              :isActive="solution.isThinking && idx === solution.currentRounds.length - 1"
              :isStalled="solution.isThinkingStalled && idx === solution.currentRounds.length - 1"
              :statusText="(solution.isThinking && idx === solution.currentRounds.length - 1) ? solution.thinkingStatusText : (round.thinkingStatus || 'Reasoning Process')"
            />

            <!-- AI response -->
            <div class="ai-response"
              v-html="(idx === solution.currentRounds.length - 1 && solution.streamingHtml) ? solution.streamingHtml : solution.renderMarkdown(round.aiResponse)">
            </div>

            <!-- Streaming cursor -->
            <span v-if="idx === solution.currentRounds.length - 1 && solution.streamingHtml && !solution.isLoading" class="stream-cursor">▍</span>

            <!-- Inline error -->
            <div v-if="round.error" class="inline-error">
              <Icon :name="round.error.icon || 'x-circle'" :size="18" class="inline-error-icon" />
              <div class="inline-error-content">
                <div class="inline-error-title">{{ round.error.title }}</div>
                <div class="inline-error-desc">{{ round.error.desc }}</div>
              </div>
            </div>

            <!-- Action bar (only for completed rounds with content) -->
            <ActionBar
              v-if="round.aiResponse && !solution.isLoading && !(idx === solution.currentRounds.length - 1 && solution.streamingHtml)"
              :content="round.aiResponse"
              :model="settingsStore.settings.model"
              @export="solution.exportImage(solution.activeHistoryIndex)"
            />
          </div>
        </template>

        <!-- Neural pill loader (loading, no thinking yet) -->
        <div v-if="solution.isLoading && !solution.isThinking" class="neural-pill-loader">
          <div class="pill-track">
            <span class="pill-bar"></span>
            <span class="pill-bar"></span>
            <span class="pill-bar"></span>
          </div>
          <span class="pill-text">AI is thinking...</span>
        </div>

        <!-- Append loading (additional round) -->
        <div v-if="solution.isAppending && !solution.isThinking" class="append-loading">
          <div class="ai-orb">
            <div class="ai-orb-inner"></div>
          </div>
          <span class="append-text">AI 正在回复</span>
          <div class="wave-dots"><span></span><span></span><span></span></div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useSolutionStore } from '../stores/solution'
import { useSettingsStore } from '../stores/settings'
import { useUIStore } from '../stores/ui'
import Icon from './Icon.vue'
import HistoryItem from './HistoryItem.vue'
import EmptyState from './EmptyState.vue'
import ErrorView from './ErrorView.vue'
import LoadingView from './LoadingView.vue'
import ThinkingBlock from './ThinkingBlock.vue'
import ActionBar from './ActionBar.vue'
import RoundDivider from './RoundDivider.vue'

const solution = useSolutionStore()
const settingsStore = useSettingsStore()
const ui = useUIStore()

const isHistoryCollapsed = ref(false)

// Check if there's any content in the current rounds
const hasAnyContent = computed(() => {
  const rounds = solution.currentRounds
  if (!rounds || rounds.length === 0) return false
  return rounds.some(r => r.aiResponse || r.thinking)
})
</script>

<style scoped>
/* ========================================
   SolveView Layout
   ======================================== */
.main-interface {
  flex: 1;
  display: flex;
  gap: var(--sp-3);
  min-height: 0;
  padding: 0 var(--sp-2) var(--sp-2);
  pointer-events: auto;
  opacity: 0;
  transform: translateY(6px);
  transition: opacity 0.35s var(--ease-out), transform 0.35s var(--ease-out);
}
.main-interface.visible {
  opacity: 1;
  transform: translateY(0);
}

/* ---- Left Panel ---- */
.left-panel {
  width: 200px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  background: rgba(
    var(--app-bg-r, 12),
    var(--app-bg-g, 12),
    var(--app-bg-b, 16),
    var(--app-panel-a, 0.50)
  );
  backdrop-filter: blur(16px);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-subtle);
  overflow: hidden;
  transition: width 0.25s var(--ease-out), opacity 0.25s ease, background-color var(--duration-base) ease;
}
.left-panel.collapsed {
  width: 44px;
}
.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--sp-3);
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}
.panel-title {
  font-size: var(--text-sm);
  font-weight: var(--weight-bold);
  color: var(--text-primary);
  white-space: nowrap;
}
.toggle-btn {
  width: 26px;
  height: 26px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  border-radius: var(--radius-sm);
  color: var(--text-muted);
  cursor: pointer;
  transition: all var(--duration-fast) ease;
}
.toggle-btn:hover {
  background: var(--surface-card-hover);
  color: var(--text-primary);
}
.history-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--sp-2);
  display: flex;
  flex-direction: column;
  gap: var(--sp-2);
}
.history-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--sp-8) var(--sp-4);
  color: var(--text-muted);
  gap: var(--sp-2);
}
.empty-icon { opacity: 0.5; }
.empty-text { font-size: var(--text-xs); }

/* ---- Right Panel ---- */
.right-panel {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  border-radius: var(--radius-lg);
  overflow: hidden;
}

/* ---- Content Area ---- */
.content-area {
  flex: 1;
  overflow-y: auto;
  padding: var(--sp-5);
}

/* Text shadow for high transparency readability */
.ai-response,
.md-body {
  text-shadow: 0 0 10px rgba(var(--app-bg-r, 12), var(--app-bg-g, 12), var(--app-bg-b, 16), var(--text-shadow-a, 0));
}

/* ---- Chat Round ---- */
.chat-round {
  margin-bottom: var(--sp-2);
}

/* AI response */
.ai-response {
  line-height: var(--leading-loose);
  color: var(--text-primary);
  font-size: var(--text-base);
}

/* Streaming cursor */
.stream-cursor {
  color: var(--accent);
  font-weight: var(--weight-bold);
  animation: cursorBlink 0.8s step-end infinite;
  font-size: var(--text-lg);
  line-height: 1;
  vertical-align: text-bottom;
}
@keyframes cursorBlink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

/* ---- Inline error ---- */
.inline-error {
  display: flex;
  gap: var(--sp-3);
  align-items: flex-start;
  padding: var(--sp-4);
  background: var(--error-bg);
  border: 1px solid var(--error-border);
  border-radius: var(--radius-md);
  margin-top: var(--sp-3);
}
.inline-error-icon {
  color: var(--color-error);
  flex-shrink: 0;
  margin-top: 1px;
}
.inline-error-title {
  font-size: var(--text-sm);
  font-weight: var(--weight-bold);
  color: var(--color-error);
  margin-bottom: var(--sp-1);
}
.inline-error-desc {
  font-size: var(--text-xs);
  color: var(--text-secondary);
  line-height: var(--leading-normal);
}

/* ---- Neural Pill Loader ---- */
.neural-pill-loader {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  padding: var(--sp-3) var(--sp-5);
  margin: var(--sp-4) 0;
  background: var(--surface-card);
  border-radius: var(--radius-full);
  border: 1px solid var(--border-subtle);
  width: fit-content;
}
.pill-track {
  display: flex;
  gap: 3px;
}
.pill-bar {
  width: 3px;
  height: 14px;
  background: var(--accent);
  border-radius: 2px;
  animation: pillWave 0.8s ease-in-out infinite;
}
.pill-bar:nth-child(2) { animation-delay: 0.1s; }
.pill-bar:nth-child(3) { animation-delay: 0.2s; }
@keyframes pillWave {
  0%,100% { transform: scaleY(0.5); opacity: 0.4; }
  50% { transform: scaleY(1); opacity: 1; }
}
.pill-text {
  font-size: var(--text-sm);
  color: var(--text-secondary);
  font-weight: var(--weight-semibold);
}

/* ---- Append Loading ---- */
.append-loading {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  padding: var(--sp-4) 0;
  margin: var(--sp-3) 0;
}
.ai-orb {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--accent-muted);
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--accent-border);
}
.ai-orb-inner {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--accent);
  animation: orbPulse 1.5s ease-in-out infinite;
}
@keyframes orbPulse {
  0%,100% { transform: scale(0.8); opacity: 0.5; }
  50% { transform: scale(1.1); opacity: 1; }
}
.append-text {
  font-size: var(--text-sm);
  color: var(--text-secondary);
  font-weight: var(--weight-semibold);
}
.wave-dots {
  display: flex;
  gap: 3px;
}
.wave-dots span {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: var(--accent);
  animation: waveDot 1s ease-in-out infinite;
}
.wave-dots span:nth-child(2) { animation-delay: 0.15s; }
.wave-dots span:nth-child(3) { animation-delay: 0.3s; }
@keyframes waveDot {
  0%,60%,100% { transform: translateY(0); }
  30% { transform: translateY(-4px); }
}
</style>
