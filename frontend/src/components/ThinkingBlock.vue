<template>
  <div class="thinking-block" :class="{ active: isActive, collapsed: !expanded, stalled: isStalled }">
    <!-- Header -->
    <div class="thinking-header" @click="expanded = !expanded">
      <div class="header-left">
        <!-- Animated indicator when active -->
        <div class="thinking-indicator" v-if="isActive">
          <span class="pulse-dot"></span>
          <span class="pulse-dot"></span>
          <span class="pulse-dot"></span>
        </div>
        <Icon v-else :name="expanded ? 'chevron-down' : 'chevron-right'" :size="14" class="chevron-icon" />
        <span class="thinking-label">{{ statusText }}</span>
      </div>
      <div class="header-right">
        <span v-if="duration" class="thinking-duration">{{ duration }}</span>
      </div>
    </div>

    <!-- Collapsed preview -->
    <div class="thinking-preview" v-if="!expanded && thinking">
      <span class="preview-text">{{ previewText }}</span>
    </div>

    <!-- Expanded body -->
    <Transition name="expand">
      <div class="thinking-body" v-show="expanded">
        <div class="thinking-content">
          <pre>{{ thinking }}</pre>
        </div>
        <!-- Stall indicator -->
        <div class="stall-bar" v-if="isStalled && isActive"></div>
      </div>
    </Transition>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import Icon from './Icon.vue'

const props = defineProps({
  thinking: { type: String, default: '' },
  isActive: { type: Boolean, default: false },
  isStalled: { type: Boolean, default: false },
  statusText: { type: String, default: 'Reasoning Process' },
  initialExpanded: { type: Boolean, default: true },
})

const expanded = ref(props.initialExpanded)

// Auto-expand when thinking starts
watch(() => props.isActive, (val) => {
  if (val) expanded.value = true
})

// Auto-collapse when thinking finishes
watch(() => props.isActive, (newVal, oldVal) => {
  if (oldVal && !newVal) {
    expanded.value = false
  }
})

const duration = computed(() => {
  // Duration is handled by parent; this is just a display slot
  return null
})

const previewText = computed(() => {
  if (!props.thinking) return ''
  const lines = props.thinking.split('\n').filter(l => l.trim())
  if (lines.length === 0) return 'Starting...'
  const lastLine = lines[lines.length - 1]
  return lastLine.length > 100 ? lastLine.substring(0, 100) + '...' : lastLine
})
</script>

<style scoped>
.thinking-block {
  border-left: 2px solid var(--border-default);
  margin-bottom: var(--sp-4);
  transition: all var(--duration-base) var(--ease-smooth);
}
.thinking-block.active {
  border-left-color: var(--accent);
}
.thinking-block.stalled {
  opacity: 0.75;
}

/* Header */
.thinking-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--sp-2) var(--sp-3);
  cursor: pointer;
  user-select: none;
  border-radius: var(--radius-sm);
  transition: background var(--duration-fast) ease;
}
.thinking-header:hover {
  background: var(--surface-card);
}
.header-left {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
}
.chevron-icon {
  color: var(--text-muted);
  transition: color var(--duration-fast) ease;
}
.thinking-label {
  font-size: var(--text-xs);
  font-weight: var(--weight-semibold);
  color: var(--text-muted);
  letter-spacing: 0.3px;
}
.thinking-block.active .thinking-label {
  color: var(--accent);
}
.thinking-duration {
  font-size: var(--text-xs);
  font-family: var(--font-mono);
  color: var(--text-muted);
}

/* Pulse dots indicator */
.thinking-indicator {
  display: flex;
  gap: 3px;
  align-items: center;
}
.pulse-dot {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: var(--accent);
  animation: pulseBounce 0.8s ease-in-out infinite;
}
.pulse-dot:nth-child(2) { animation-delay: 0.1s; }
.pulse-dot:nth-child(3) { animation-delay: 0.2s; }
@keyframes pulseBounce {
  0%, 100% { transform: scale(0.8); opacity: 0.4; }
  50% { transform: scale(1.2); opacity: 1; }
}

/* Preview (collapsed) */
.thinking-preview {
  padding: 0 var(--sp-3) var(--sp-2);
}
.preview-text {
  font-size: var(--text-xs);
  color: var(--text-muted);
  font-style: italic;
  opacity: 0.75;
  line-height: var(--leading-normal);
}

/* Body (expanded) */
.thinking-body {
  position: relative;
}
.thinking-content {
  padding: var(--sp-2) var(--sp-3);
  max-height: 300px;
  overflow-y: auto;
}
.thinking-content pre {
  font-family: var(--font-mono);
  font-size: 11px;
  line-height: var(--leading-relaxed);
  color: var(--text-muted);
  white-space: pre-wrap;
  word-break: break-word;
  margin: 0;
  opacity: 0.85;
}

/* Stall indicator bar */
.stall-bar {
  height: 2px;
  background: linear-gradient(90deg, transparent, var(--accent), transparent);
  animation: stallPulse 1.5s ease-in-out infinite;
}
@keyframes stallPulse {
  0%, 100% { opacity: 0.3; transform: scaleX(0.5); }
  50% { opacity: 1; transform: scaleX(1); }
}

/* Expand transition */
.expand-enter-active { transition: all 0.25s var(--ease-out); }
.expand-leave-active { transition: all 0.2s ease-in; }
.expand-enter-from, .expand-leave-to {
  opacity: 0;
  max-height: 0;
}
</style>
