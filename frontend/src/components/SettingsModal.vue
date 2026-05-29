<template>
  <div v-if="ui.showSettings" class="modal" id="settings-modal" style="display: flex">
    <div class="modal-content">
      <div class="modal-warning-banner">
        <Icon name="alert-triangle" :size="13" class="banner-icon" />
        <span>当前窗口已获得焦点，关闭设置后将自动恢复低打扰模式</span>
      </div>

      <div class="modal-header">
        <div class="tabs">
          <div class="tab" :class="{ active: ui.activeTab === 'general' }" @click="ui.activeTab = 'general'">常规</div>
          <div class="tab" :class="{ active: ui.activeTab === 'model' }" @click="ui.activeTab = 'model'">模型</div>
          <div class="tab" :class="{ active: ui.activeTab === 'screenshot' }" @click="ui.activeTab = 'screenshot'">截图</div>
          <div class="tab" :class="{ active: ui.activeTab === 'resume' }" @click="ui.activeTab = 'resume'">简历</div>
          <div class="tab" :class="{ active: ui.activeTab === 'api' }" @click="ui.activeTab = 'api'">API</div>
        </div>
        <button class="close-btn" @click="settingsStore.closeSettings">
          <Icon name="x" :size="16" />
        </button>
      </div>

      <div class="modal-body">
        <div v-show="ui.activeTab === 'general'" class="tab-pane">
          <div class="form-group">
            <label>快捷键配置 {{ settingsStore.isMacOS ? '(macOS 使用固定快捷键)' : '(点击录制)' }}</label>
            <div class="shortcut-list">
              <div class="shortcut-item" v-for="key in settingsStore.shortcutActions" :key="key.action">
                <span>{{ key.label }}</span>
                <button
                  class="btn-record"
                  :class="{ recording: settingsStore.recordingAction === key.action, disabled: settingsStore.isMacOS }"
                  @click="!settingsStore.isMacOS && settingsStore.recordKey(key.action)"
                >
                  {{
                    settingsStore.recordingAction === key.action
                      ? settingsStore.recordingText
                      : (settingsStore.tempShortcuts[key.action]?.keyName || (settingsStore.isMacOS ? key.macDefault : key.default))
                  }}
                </button>
              </div>
            </div>
          </div>

          <div class="form-group">
            <label for="opacity-slider">窗口透明度 <span>{{ Math.round(settingsStore.tempSettings.transparency * 100) }}%</span></label>
            <input
              id="opacity-slider"
              v-model.number="settingsStore.tempSettings.transparency"
              type="range"
              min="0.0"
              max="1.0"
              step="0.05"
            />
          </div>
        </div>

        <div v-show="ui.activeTab === 'model'" class="tab-pane model-tab">
          <div class="form-group model-select-group">
            <div class="model-header">
              <label>模型选择</label>
              <div class="model-actions">
                <button
                  class="btn-icon"
                  @click="settingsStore.refreshModels"
                  :disabled="ui.isLoadingModels || !settingsStore.tempSettings.apiKey"
                  title="刷新模型列表"
                >
                  <svg class="action-icon" :class="{ spin: ui.isLoadingModels }" viewBox="0 0 16 16" fill="none">
                    <path d="M14 8a6 6 0 01-10.24 4.24" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
                    <path d="M2 8a6 6 0 0110.24-4.24" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
                    <path d="M14 3v5h-5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                    <path d="M2 13V8h5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                  </svg>
                </button>
                <button
                  class="btn-icon"
                  @click="settingsStore.testConnection"
                  :disabled="ui.isTestingConnection || !settingsStore.tempSettings.model"
                  title="测试模型连通性"
                >
                  <svg v-if="ui.isTestingConnection" class="action-icon spin" viewBox="0 0 16 16" fill="none">
                    <circle cx="8" cy="8" r="6" stroke="currentColor" stroke-width="1.5" stroke-dasharray="28 10" stroke-linecap="round" />
                  </svg>
                  <svg v-else class="action-icon" viewBox="0 0 16 16" fill="none">
                    <path d="M4 3l9 5-9 5V3z" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round" />
                  </svg>
                </button>
              </div>
            </div>

            <ModelSelect v-model="settingsStore.tempSettings.model" :models="ui.availableModels" :loading="ui.isLoadingModels" />

            <div v-if="ui.connectionStatus" class="connection-status" :class="ui.connectionStatus.type">
              <span class="cs-icon">{{ ui.connectionStatus.icon }}</span>
              <span class="cs-text">{{ ui.connectionStatus.message }}</span>
            </div>

            <p v-if="!settingsStore.tempSettings.apiKey" class="hint-text warning-hint">请先填写 API Key</p>
          </div>

          <div class="form-group domain-group">
            <DomainSelector v-model="settingsStore.tempSettings.domainId" :categories="settingsStore.domainCategories" />
          </div>
        </div>

        <div v-show="ui.activeTab === 'screenshot'" class="tab-pane">
          <ScreenshotSettings v-model="screenshotConfig" />
        </div>

        <div v-if="ui.activeTab === 'resume'" class="tab-pane">
          <ResumeImport
            :resumePath="settingsStore.tempSettings.resumePath"
            :rawContent="settingsStore.resumeRawContent"
            :isParsing="settingsStore.isResumeParsing"
            @update:rawContent="val => settingsStore.resumeRawContent = val"
            @select-resume="settingsStore.selectResume"
            @clear-resume="settingsStore.clearResume"
            @parse-resume="settingsStore.parseResume"
          />
        </div>

        <div v-show="ui.activeTab === 'api'" class="tab-pane api-tab-pane">
          <ProviderSelect
            v-model:apiKey="settingsStore.tempSettings.apiKey"
            v-model:baseURL="settingsStore.tempSettings.baseURL"
          />
        </div>
      </div>

      <div class="modal-footer">
        <button class="btn-primary" @click="settingsStore.saveSettings">保存</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useUIStore } from '../stores/ui'
import { useSettingsStore } from '../stores/settings'
import ResumeImport from './ResumeImport.vue'
import ProviderSelect from './ProviderSelect.vue'
import ModelSelect from './ModelSelect.vue'
import DomainSelector from './DomainSelector.vue'
import ScreenshotSettings from './ScreenshotSettings.vue'
import Icon from './Icon.vue'

const ui = useUIStore()
const settingsStore = useSettingsStore()

const screenshotConfig = computed({
  get: () => ({
    compressionQuality: settingsStore.tempSettings.compressionQuality,
    sharpening: settingsStore.tempSettings.sharpening,
    grayscale: settingsStore.tempSettings.grayscale,
    noCompression: settingsStore.tempSettings.noCompression,
    screenshotMode: settingsStore.tempSettings.screenshotMode,
  }),
  set: (val) => {
    settingsStore.tempSettings.compressionQuality = val.compressionQuality
    settingsStore.tempSettings.sharpening = val.sharpening
    settingsStore.tempSettings.grayscale = val.grayscale
    settingsStore.tempSettings.noCompression = val.noCompression
    settingsStore.tempSettings.screenshotMode = val.screenshotMode
  }
})
</script>

<style scoped>
.modal {
  position: fixed;
  inset: 0;
  z-index: 3000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.55);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  pointer-events: auto;
}

.modal-content {
  width: 520px;
  max-width: 92vw;
  height: 580px;
  max-height: 85vh;
  background: var(--surface-popover);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-xl);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  animation: modalIn 0.25s var(--ease-out);
}

@keyframes modalIn {
  from { opacity: 0; transform: scale(0.96) translateY(8px); }
  to { opacity: 1; transform: scale(1) translateY(0); }
}

.modal-warning-banner {
  background: var(--warning-bg);
  border: 1px solid var(--warning-border);
  border-radius: var(--radius-full);
  padding: 6px 16px;
  color: var(--color-warning);
  font-size: var(--text-xs);
  font-weight: var(--weight-medium);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--sp-1-5);
  margin: 12px auto 4px auto;
  width: fit-content;
}

.banner-icon { flex-shrink: 0; }

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--sp-3) var(--sp-5);
  border-bottom: 1px solid var(--border-subtle);
}

.tabs {
  display: flex;
  gap: var(--sp-1);
}

.tab {
  padding: var(--sp-2) var(--sp-4);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--text-muted);
  cursor: pointer;
  transition: all var(--duration-fast) ease;
}

.tab:hover {
  color: var(--text-primary);
  background: var(--surface-card);
}

.tab.active {
  color: var(--accent);
  background: var(--accent-muted);
}

.close-btn {
  color: var(--text-muted);
  cursor: pointer;
  width: 30px;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  border: none;
  background: transparent;
  transition: all var(--duration-fast) ease;
}

.close-btn:hover {
  background: var(--surface-card-hover);
  color: var(--text-primary);
}

.modal-body {
  flex: 1;
  overflow: hidden;
  padding: 0;
  min-height: 0;
  position: relative;
}

.tab-pane {
  height: 100%;
  overflow-y: auto;
  padding: var(--sp-5);
  box-sizing: border-box;
}

.api-tab-pane {
  overflow-y: auto;
  display: block;
}

.modal-footer {
  padding: var(--sp-4) var(--sp-5);
  border-top: 1px solid var(--border-subtle);
  display: flex;
  justify-content: flex-end;
}

.btn-primary {
  padding: var(--sp-2) var(--sp-6);
  border-radius: var(--radius-md);
  background: var(--accent);
  color: var(--text-inverse);
  font-size: var(--text-sm);
  font-weight: 700;
  border: none;
  cursor: pointer;
  transition: all var(--duration-fast) ease;
}

.btn-primary:hover {
  background: var(--accent-hover);
  transform: translateY(-1px);
}

.form-group { margin-bottom: var(--sp-5); }

.form-group label {
  display: block;
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: var(--sp-2);
}

.model-tab {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.model-select-group { flex-shrink: 0; }

.domain-group {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

:deep(.domain-selector) {
  height: 100%;
  display: flex;
  flex-direction: column;
}

:deep(.domain-grid) {
  flex: 1;
  height: auto !important;
}

.model-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--sp-2);
}

.model-actions {
  display: flex;
  gap: var(--sp-2);
}

.btn-icon {
  width: 30px;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--surface-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--duration-fast) ease;
}

.btn-icon:hover {
  background: var(--surface-card-hover);
  color: var(--text-primary);
  border-color: var(--border-hover);
}

.btn-icon:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.action-icon {
  width: 14px;
  height: 14px;
}

.action-icon.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

.connection-status {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  padding: var(--sp-2) var(--sp-3);
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
  margin-top: var(--sp-2);
}

.connection-status.success {
  background: var(--success-bg);
  border: 1px solid var(--success-border);
  color: var(--color-success);
}

.connection-status.error {
  background: var(--error-bg);
  border: 1px solid var(--error-border);
  color: var(--color-error);
}

.cs-icon {
  font-size: 14px;
  font-weight: 700;
}

.cs-text { font-weight: 600; }

.hint-text {
  font-size: var(--text-xs);
  color: var(--text-muted);
  margin-top: var(--sp-2);
}

.warning-hint { color: var(--color-warning); }

.shortcut-list {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--sp-2);
}

.shortcut-item:last-child:nth-child(odd) { grid-column: 1 / -1; }

.shortcut-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--sp-2) var(--sp-3);
  background: var(--surface-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  color: var(--text-primary);
  gap: var(--sp-2);
}

.btn-record {
  padding: var(--sp-1) var(--sp-3);
  font-size: var(--text-xs);
  font-family: var(--font-mono);
  font-weight: 600;
  background: var(--surface-input);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-xs);
  color: var(--text-secondary);
  cursor: pointer;
  min-width: 70px;
  text-align: center;
  transition: all var(--duration-fast) ease;
}

.btn-record:hover:not(.disabled) {
  border-color: var(--accent-border);
  color: var(--accent);
}

.btn-record.recording {
  border-color: var(--accent);
  background: var(--accent-muted);
  color: var(--accent);
  animation: pulseRecord 1s ease-in-out infinite;
}

.btn-record.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

@keyframes pulseRecord {
  0%, 100% { box-shadow: 0 0 0 0 var(--accent-glow); }
  50% { box-shadow: 0 0 0 4px var(--accent-glow); }
}

input[type="range"] {
  width: 100%;
  height: 4px;
  appearance: none;
  -webkit-appearance: none;
  background: var(--border-default);
  border-radius: var(--radius-full);
  outline: none;
  cursor: pointer;
}

input[type="range"]::-webkit-slider-thumb {
  -webkit-appearance: none;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: var(--accent);
  border: 2px solid var(--surface-elevated);
  box-shadow: var(--shadow-sm);
  cursor: pointer;
}
</style>
