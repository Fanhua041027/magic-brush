<template>
  <div class="resume-import">
    <div v-if="!resumePath" class="empty-state">
      <div class="upload-card" @click="$emit('select-resume')">
        <div class="upload-visual">
          <div class="upload-icon-wrap">
            <Icon name="file" :size="28" class="upload-icon" />
          </div>
          <div class="upload-glow"></div>
        </div>
        <div class="upload-text">
          <h3 class="upload-title">导入 PDF 简历</h3>
          <p class="upload-desc">AI 会在解题时参考你的背景信息，提供更贴近场景的回答。</p>
        </div>
        <div class="upload-action">
          <span class="upload-btn">
            <Icon name="download" :size="14" />
            <span>选择文件</span>
          </span>
          <span class="upload-hint">支持 .pdf 格式</span>
        </div>
      </div>
    </div>

    <div v-else class="resume-content">
      <div class="file-bar">
        <div class="file-info">
          <div class="file-icon-box">
            <Icon name="file" :size="16" class="pdf-icon" />
          </div>
          <div class="file-meta">
            <span class="file-name">{{ fileName }}</span>
            <span class="file-badge">PDF</span>
          </div>
        </div>
        <div class="file-actions">
          <button class="action-btn" @click="$emit('select-resume')" title="更换文件">
            <Icon name="refresh" :size="14" />
          </button>
          <button class="action-btn danger" @click="$emit('clear-resume')" title="清除简历">
            <Icon name="trash" :size="14" />
          </button>
        </div>
      </div>

      <div v-if="isParsing" class="status-card">
        <div class="parsing-visual">
          <div class="parse-spinner"></div>
        </div>
        <div class="status-text">
          <span class="status-title">正在解析简历</span>
          <span class="status-desc">当前会直接调用你配置的模型接口，将 PDF 解析为 Markdown，请稍候。</span>
        </div>
      </div>

      <div v-else-if="!localContent && !isEditing" class="status-card">
        <div class="error-visual">
          <Icon name="alert-triangle" :size="24" class="error-icon" />
        </div>
        <div class="status-text">
          <span class="status-title">暂未生成解析结果</span>
          <span class="status-desc">你可以重新解析，也可以直接手动粘贴或编辑 Markdown 内容。</span>
        </div>
        <button class="retry-btn" @click="$emit('parse-resume')">
          <Icon name="refresh" :size="13" />
          <span>重新解析</span>
        </button>
      </div>

      <div v-else class="md-panel">
        <div class="md-toolbar">
          <div class="toolbar-tabs">
            <button class="tab-btn" :class="{ active: !isEditing }" @click="isEditing = false">预览</button>
            <button class="tab-btn" :class="{ active: isEditing }" @click="isEditing = true">编辑</button>
          </div>
          <div class="toolbar-right">
            <button class="toolbar-icon-btn" @click="$emit('parse-resume')" :disabled="isParsing" title="重新解析">
              <Icon name="refresh" :size="14" :spinning="isParsing" />
            </button>
          </div>
        </div>

        <div v-if="isEditing" class="editor-wrap">
          <textarea
            v-model="localContent"
            @input="updateContent"
            class="md-editor"
            placeholder="在此输入或粘贴 Markdown 格式的简历内容..."
          ></textarea>
        </div>

        <div v-else class="md-preview md-body" v-html="renderedContent"></div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { marked } from 'marked'
import Icon from './Icon.vue'

const props = defineProps({
  resumePath: { type: String, default: '' },
  rawContent: { type: String, default: '' },
  isParsing: { type: Boolean, default: false },
})

const emit = defineEmits(['select-resume', 'clear-resume', 'parse-resume', 'update:rawContent'])

const isEditing = ref(false)
const localContent = ref(props.rawContent)

const fileName = computed(() => {
  if (!props.resumePath) return ''
  return props.resumePath.split(/[\\/]/).pop() || 'resume.pdf'
})

const renderedContent = computed(() => {
  if (!localContent.value) return ''
  return marked.parse(localContent.value)
})

watch(() => props.rawContent, (newVal) => {
  if (newVal !== localContent.value) localContent.value = newVal
})

function updateContent() {
  emit('update:rawContent', localContent.value)
}
</script>

<style scoped>
.resume-import {
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--text-primary);
}

.empty-state {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--sp-5);
}

.upload-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--sp-5);
  padding: var(--sp-8) var(--sp-10);
  border: 2px dashed var(--border-default);
  border-radius: var(--radius-xl);
  cursor: pointer;
  transition: all var(--duration-base) var(--ease-smooth);
  text-align: center;
  max-width: 340px;
}

.upload-card:hover {
  border-color: var(--accent-border);
  background: var(--accent-muted);
}

.upload-visual {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}

.upload-icon-wrap {
  width: 56px;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--accent-muted);
  border: 1px solid var(--accent-border);
  border-radius: var(--radius-lg);
  color: var(--accent);
  position: relative;
  z-index: 1;
}

.upload-glow {
  position: absolute;
  width: 80px;
  height: 80px;
  background: radial-gradient(circle, var(--accent-glow) 0%, transparent 70%);
  border-radius: 50%;
  filter: blur(16px);
  animation: breathe 3s ease-in-out infinite;
}

@keyframes breathe {
  0%, 100% { transform: scale(1); opacity: 0.4; }
  50% { transform: scale(1.15); opacity: 0.7; }
}

.upload-title {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: var(--weight-bold);
  color: var(--text-primary);
}

.upload-desc {
  margin: var(--sp-1) 0 0;
  font-size: var(--text-xs);
  color: var(--text-muted);
  line-height: var(--leading-relaxed);
  max-width: 240px;
}

.upload-action {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--sp-2);
}

.upload-btn {
  display: inline-flex;
  align-items: center;
  gap: var(--sp-1-5);
  padding: var(--sp-2) var(--sp-5);
  background: var(--accent-gradient);
  border-radius: var(--radius-md);
  color: white;
  font-size: var(--text-sm);
  font-weight: var(--weight-semibold);
  transition: all var(--duration-fast) ease;
}

.upload-card:hover .upload-btn { box-shadow: var(--shadow-accent); }

.upload-hint {
  font-size: var(--text-xs);
  color: var(--text-muted);
}

.resume-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
  min-height: 0;
}

.file-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--sp-2-5) var(--sp-3);
  background: var(--surface-card);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.file-info {
  display: flex;
  align-items: center;
  gap: var(--sp-2-5);
}

.file-icon-box {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--accent-muted);
  border-radius: var(--radius-sm);
  color: var(--accent);
}

.file-meta {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
}

.file-name {
  font-size: var(--text-sm);
  font-weight: var(--weight-semibold);
  color: var(--text-primary);
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-badge {
  font-size: 10px;
  font-weight: var(--weight-bold);
  padding: 1px 5px;
  border-radius: var(--radius-xs);
  background: var(--accent-muted);
  color: var(--accent);
  letter-spacing: 0.5px;
}

.file-actions {
  display: flex;
  gap: var(--sp-1);
}

.action-btn {
  width: 28px;
  height: 28px;
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

.action-btn:hover {
  background: var(--surface-card-hover);
  color: var(--text-primary);
}

.action-btn.danger:hover {
  background: var(--error-bg);
  color: var(--color-error);
}

.status-card {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--sp-4);
  padding: var(--sp-8);
  text-align: center;
  background: var(--surface-card);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-subtle);
}

.parse-spinner {
  width: 36px;
  height: 36px;
  border: 2.5px solid var(--border-default);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin { to { transform: rotate(360deg); } }

.error-visual {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--warning-bg);
  border: 1px solid var(--warning-border);
  border-radius: var(--radius-lg);
}

.error-icon { color: var(--color-warning); }

.status-text {
  display: flex;
  flex-direction: column;
  gap: var(--sp-1);
}

.status-title {
  font-size: var(--text-base);
  font-weight: var(--weight-semibold);
  color: var(--text-primary);
}

.status-desc {
  font-size: var(--text-xs);
  color: var(--text-muted);
  line-height: var(--leading-relaxed);
}

.retry-btn {
  display: inline-flex;
  align-items: center;
  gap: var(--sp-1-5);
  padding: var(--sp-2) var(--sp-4);
  background: transparent;
  border: 1px solid var(--accent-border);
  border-radius: var(--radius-md);
  color: var(--accent);
  font-size: var(--text-sm);
  font-weight: var(--weight-medium);
  cursor: pointer;
  transition: all var(--duration-fast) ease;
}

.retry-btn:hover {
  background: var(--accent-muted);
  border-color: var(--accent);
}

.md-panel {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--surface-card);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-subtle);
  overflow: hidden;
}

.md-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--sp-1-5) var(--sp-3);
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.toolbar-tabs {
  display: flex;
  gap: 2px;
  background: var(--surface-input);
  padding: 2px;
  border-radius: var(--radius-sm);
}

.tab-btn {
  padding: var(--sp-1) var(--sp-3);
  background: transparent;
  border: none;
  border-radius: var(--radius-xs);
  color: var(--text-muted);
  font-size: var(--text-xs);
  font-weight: var(--weight-medium);
  cursor: pointer;
  transition: all var(--duration-fast) ease;
}

.tab-btn:hover { color: var(--text-secondary); }

.tab-btn.active {
  background: var(--surface-elevated);
  color: var(--accent);
  font-weight: var(--weight-semibold);
  box-shadow: var(--shadow-sm);
}

.toolbar-right {
  display: flex;
  gap: var(--sp-1);
}

.toolbar-icon-btn {
  width: 28px;
  height: 28px;
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

.toolbar-icon-btn:hover:not(:disabled) {
  background: var(--surface-card-hover);
  color: var(--accent);
}

.toolbar-icon-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.editor-wrap {
  flex: 1;
  min-height: 0;
}

.md-editor {
  width: 100%;
  height: 100%;
  padding: var(--sp-4);
  background: transparent;
  border: none;
  color: var(--text-primary);
  font-family: var(--font-mono);
  font-size: var(--text-sm);
  line-height: var(--leading-relaxed);
  resize: none;
  outline: none;
  box-sizing: border-box;
}

.md-editor::placeholder { color: var(--text-muted); }

.md-preview {
  flex: 1;
  padding: var(--sp-4);
  overflow-y: auto;
  font-size: var(--text-sm);
  line-height: var(--leading-relaxed);
}
</style>
