<template>
  <div class="action-bar">
    <div class="action-left">
      <!-- Copy -->
      <button class="action-btn" @click="handleCopy" :title="copied ? '已复制' : '复制全文'">
        <Icon :name="copied ? 'check' : 'copy'" :size="14" />
        <span>{{ copied ? '已复制' : '复制' }}</span>
      </button>
      <!-- Export -->
      <button class="action-btn" @click="$emit('export')" title="导出为图片">
        <Icon name="image" :size="14" />
        <span>导出</span>
      </button>
    </div>
    <div class="action-right">
      <span class="meta-text" v-if="model">{{ model }}</span>
      <span class="meta-dot" v-if="model && duration">·</span>
      <span class="meta-text" v-if="duration">{{ duration }}</span>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import Icon from './Icon.vue'

const props = defineProps({
  content: { type: String, default: '' },
  model: { type: String, default: '' },
  duration: { type: String, default: '' },
})

defineEmits(['export'])

const copied = ref(false)

function handleCopy() {
  if (!props.content) return
  navigator.clipboard.writeText(props.content).then(() => {
    copied.value = true
    setTimeout(() => { copied.value = false }, 2000)
  }).catch(() => {
    // Fallback
    const ta = document.createElement('textarea')
    ta.value = props.content
    document.body.appendChild(ta)
    ta.select()
    document.execCommand('copy')
    document.body.removeChild(ta)
    copied.value = true
    setTimeout(() => { copied.value = false }, 2000)
  })
}
</script>

<style scoped>
.action-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--sp-2) 0;
  margin-top: var(--sp-3);
  border-top: 1px solid var(--border-subtle);
}
.action-left {
  display: flex;
  gap: var(--sp-1);
}
.action-btn {
  display: flex;
  align-items: center;
  gap: var(--sp-1);
  padding: var(--sp-1) var(--sp-2);
  border: none;
  background: transparent;
  border-radius: var(--radius-sm);
  color: var(--text-muted);
  font-size: var(--text-xs);
  font-weight: var(--weight-medium);
  cursor: pointer;
  transition: all var(--duration-fast) ease;
}
.action-btn:hover {
  background: var(--surface-card);
  color: var(--text-secondary);
}
.action-right {
  display: flex;
  align-items: center;
  gap: var(--sp-1-5);
}
.meta-text {
  font-size: var(--text-xs);
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-weight: var(--weight-medium);
}
.meta-dot {
  color: var(--text-muted);
  opacity: 0.4;
  font-size: 10px;
}
</style>
