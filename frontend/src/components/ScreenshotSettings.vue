<template>
  <div class="screenshot-settings">
    <!-- macOS 截图权限提示 -->
    <div v-if="isMacOS && !hasPermission" class="permission-banner">
      <div class="banner-body">
        <span class="banner-icon">⚠️</span>
        <div class="banner-text">
          <strong>需要截图权限</strong>
          <p>请授权截图权限以正常使用截图功能，否则只能截取桌面壁纸。</p>
        </div>
      </div>
      <button v-if="!settingsOpened" class="btn-action warn" @click="requestPermission" :disabled="requestingPermission">
        {{ requestingPermission ? '正在请求...' : '授权截图权限' }}
      </button>
      <button v-else class="btn-action success" @click="refreshPermission" :disabled="requestingPermission">
        {{ requestingPermission ? '正在检查...' : '刷新权限状态' }}
      </button>
    </div>

    <!-- 截图预览区 -->
    <div class="ss-group">
      <div class="label-row">
        <label>截图预览</label>
        <button class="btn-refresh-preview" @click="updatePreview" :disabled="loading">
          <svg :class="{ spinning: loading }" width="14" height="14" viewBox="0 0 24 24" fill="none">
            <path d="M21 12C21 16.97 16.97 21 12 21C7.03 21 3 16.97 3 12C3 7.03 7.03 3 12 3C15.3 3 18.19 4.78 19.75 7.43" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
            <path d="M21 3V8H16" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
          <span>刷新预览</span>
        </button>
      </div>
      <div class="preview-box" @click="previewImage && (showLightbox = true)">
        <div v-if="loading" class="preview-placeholder">
          <div class="preview-spinner"></div>
          <span>截图中...</span>
        </div>
        <img v-else-if="previewImage" :src="previewImage" class="preview-img" title="点击放大" />
        <div v-else class="preview-placeholder">
          <span>点击刷新查看预览</span>
        </div>
        <span v-if="imageSize && !loading" class="size-tag">{{ imageSize }}</span>
      </div>
    </div>

    <!-- 截图模式 -->
    <div class="ss-group">
      <div class="label-row">
        <label>截图模式</label>
        <span class="hint">选择截图捕获区域</span>
      </div>
      <div class="mode-pills">
        <button class="pill" :class="{ active: screenshotMode === 'window' }" @click="setMode('window')">
          <span class="pill-icon">🔲</span>
          <span class="pill-text">窗口区域</span>
        </button>
        <button class="pill" :class="{ active: screenshotMode === 'fullscreen' }" @click="setMode('fullscreen')">
          <span class="pill-icon">🖥️</span>
          <span class="pill-text">全屏截图</span>
        </button>
      </div>
    </div>

    <!-- 不压缩开关 -->
    <div class="ss-group">
      <div class="setting-card">
        <div class="setting-row">
          <div class="setting-info">
            <span class="setting-title">不压缩图片</span>
            <span class="setting-desc">直接上传原始 PNG 截图，体积最大但保留所有细节</span>
          </div>
          <label class="switch">
            <input type="checkbox" v-model="noCompression" @change="updatePreview" />
            <span class="slider round"></span>
          </label>
        </div>
      </div>
    </div>

    <!-- 压缩参数 -->
    <div class="compression-params" :class="{ disabled: noCompression }">
      <div class="ss-group">
        <div class="label-row">
          <label>压缩质量</label>
          <span class="value-badge">{{ quality }}</span>
        </div>
        <input type="range" v-model.number="quality" min="1" max="90" step="1" @change="updatePreview" :disabled="noCompression" />
        <div class="range-hints">
          <span>低质量 · 小体积</span>
          <span>高质量 · 大体积</span>
        </div>
      </div>

      <div class="ss-group">
        <div class="label-row">
          <label>锐化程度</label>
          <span class="value-badge">{{ sharpen.toFixed(1) }}</span>
        </div>
        <input type="range" v-model.number="sharpen" min="0" max="5" step="0.1" @change="updatePreview" :disabled="noCompression" />
        <div class="range-hints">
          <span>无锐化</span>
          <span>强锐化</span>
        </div>
      </div>

      <div class="ss-group">
        <div class="setting-card compact">
          <div class="setting-row">
            <div class="setting-info">
              <span class="setting-title">灰度模式</span>
              <span class="setting-desc">移除颜色信息，显著减小体积</span>
            </div>
            <label class="switch">
              <input type="checkbox" v-model="isGrayscale" @change="updatePreview" :disabled="noCompression" />
              <span class="slider round"></span>
            </label>
          </div>
        </div>
      </div>
    </div>

    <!-- Lightbox -->
    <Teleport to="body">
      <Transition name="lightbox">
        <div v-if="showLightbox" class="lightbox" @click="showLightbox = false">
          <img :src="previewImage" class="lightbox-img" />
          <span class="lightbox-hint">点击任意处关闭</span>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import { api } from '../services/api'

const props = defineProps(['modelValue'])
const emit = defineEmits(['update:modelValue'])

// ---- 状态 ----
const quality = ref(80)
const sharpen = ref(0)
const previewImage = ref('')
const imageSize = ref('')
const loading = ref(false)
const isGrayscale = ref(true)
const noCompression = ref(false)
const showLightbox = ref(false)
const screenshotMode = ref('fullscreen')

// ---- macOS 权限 ----
const isMacOS = ref(false)
const hasPermission = ref(true)
const requestingPermission = ref(false)
const settingsOpened = ref(false)

function detectPlatform() {
  const ua = (navigator.userAgent || '').toLowerCase()
  isMacOS.value = ua.includes('mac')
}

async function checkPermission() {
  if (!isMacOS.value) { hasPermission.value = true; return }
  try {
    hasPermission.value = await api.checkScreenCapturePermission()
  } catch {
    hasPermission.value = true
  }
}

async function requestPermission() {
  requestingPermission.value = true
  try {
    await api.requestScreenCapturePermission()
    await api.setAlwaysOnTop(false)
    await api.openScreenCaptureSettings()
    settingsOpened.value = true
  } catch (e) {
    console.error('请求截图权限失败:', e)
  } finally {
    requestingPermission.value = false
  }
}

async function refreshPermission() {
  requestingPermission.value = true
  try {
    await checkPermission()
    await api.setAlwaysOnTop(true)
    settingsOpened.value = false
    if (hasPermission.value) updatePreview()
  } catch {
    await api.setAlwaysOnTop(true)
    settingsOpened.value = false
  } finally {
    requestingPermission.value = false
  }
}

// ---- 模式切换 ----
function setMode(mode) {
  screenshotMode.value = mode
  updatePreview()
}

// ---- 双向绑定 ----
watch(() => props.modelValue, (val) => {
  if (!val) return
  quality.value = val.compressionQuality ?? 80
  sharpen.value = val.sharpening ?? 0
  isGrayscale.value = val.grayscale ?? true
  noCompression.value = val.noCompression ?? false
  screenshotMode.value = val.screenshotMode ?? 'fullscreen'
}, { immediate: true, deep: true })

watch([quality, sharpen, isGrayscale, noCompression, screenshotMode], () => {
  emit('update:modelValue', {
    ...props.modelValue,
    compressionQuality: quality.value,
    sharpening: sharpen.value,
    grayscale: isGrayscale.value,
    noCompression: noCompression.value,
    screenshotMode: screenshotMode.value,
  })
})

// ---- 预览 ----
async function updatePreview() {
  loading.value = true
  try {
    const result = await api.getScreenshotPreview(
      quality.value, sharpen.value, isGrayscale.value,
      noCompression.value, screenshotMode.value
    )
    previewImage.value = result.base64
    imageSize.value = result.size
  } catch (e) {
    console.error('预览截图失败:', e)
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  detectPlatform()
  await checkPermission()
  updatePreview()
})
</script>

<style scoped>
.screenshot-settings {
  display: flex;
  flex-direction: column;
  gap: var(--sp-4);
}

/* ---- Permission Banner ---- */
.permission-banner {
  background: var(--warning-bg);
  border: 1px solid var(--warning-border);
  border-radius: var(--radius-md);
  padding: var(--sp-4);
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
}
.banner-body {
  display: flex;
  align-items: flex-start;
  gap: var(--sp-3);
}
.banner-icon { font-size: 20px; flex-shrink: 0; }
.banner-text { flex: 1; }
.banner-text strong {
  display: block;
  font-size: var(--text-sm);
  color: var(--color-warning);
  margin-bottom: 2px;
}
.banner-text p {
  font-size: var(--text-xs);
  color: var(--text-secondary);
  margin: 0;
  line-height: 1.5;
}
.btn-action {
  padding: var(--sp-2) var(--sp-4);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 600;
  cursor: pointer;
  border: none;
  transition: all var(--duration-fast) ease;
}
.btn-action:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-action.warn { background: var(--color-warning); color: var(--text-inverse); }
.btn-action.warn:hover:not(:disabled) { filter: brightness(1.1); }
.btn-action.success { background: var(--color-success); color: var(--text-inverse); }
.btn-action.success:hover:not(:disabled) { filter: brightness(1.1); }

/* ---- Preview ---- */
.preview-box {
  position: relative;
  height: 180px;
  background: var(--surface-input);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  cursor: pointer;
  transition: border-color var(--duration-fast) ease;
}
.preview-box:hover { border-color: var(--border-hover); }
.preview-img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
}
.preview-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--sp-2);
  color: var(--text-muted);
  font-size: var(--text-xs);
}
.preview-spinner {
  width: 20px;
  height: 20px;
  border: 2px solid var(--border-default);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
.size-tag {
  position: absolute;
  bottom: var(--sp-2);
  right: var(--sp-2);
  background: var(--accent-muted);
  color: var(--accent);
  padding: 2px 8px;
  border-radius: var(--radius-full);
  font-size: 10px;
  font-weight: 600;
  border: 1px solid var(--accent-border);
}

/* ---- Refresh Button ---- */
.btn-refresh-preview {
  display: flex;
  align-items: center;
  gap: var(--sp-1);
  padding: 2px var(--sp-2);
  font-size: var(--text-xs);
  color: var(--text-secondary);
  background: var(--surface-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-xs);
  cursor: pointer;
  transition: all var(--duration-fast) ease;
}
.btn-refresh-preview:hover { color: var(--accent); border-color: var(--accent-border); }
.btn-refresh-preview:disabled { opacity: 0.5; cursor: not-allowed; }
.spinning { animation: spin 0.8s linear infinite; }

/* ---- Mode Pills ---- */
.mode-pills {
  display: flex;
  gap: var(--sp-2);
}
.pill {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--sp-2);
  padding: var(--sp-2) var(--sp-3);
  background: var(--surface-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  font-size: var(--text-sm);
  cursor: pointer;
  transition: all var(--duration-fast) ease;
}
.pill:hover { background: var(--surface-card-hover); color: var(--text-primary); }
.pill.active {
  background: var(--accent-muted);
  border-color: var(--accent-border);
  color: var(--accent);
}
.pill-icon { font-size: 16px; }

/* ---- Setting Card ---- */
.setting-card {
  background: var(--surface-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: var(--sp-4);
}
.setting-card.compact { padding: var(--sp-3); }
.setting-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.setting-info { flex: 1; }
.setting-title {
  display: block;
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--text-primary);
}
.setting-desc {
  display: block;
  font-size: var(--text-xs);
  color: var(--text-muted);
  margin-top: 2px;
}

/* ---- Compression Params ---- */
.compression-params {
  display: flex;
  flex-direction: column;
  gap: var(--sp-4);
  transition: opacity var(--duration-base) ease;
}
.compression-params.disabled { opacity: 0.4; pointer-events: none; }

/* ---- Label Row ---- */
.label-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--sp-2);
}
.label-row label {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}
.hint {
  font-size: var(--text-xs);
  color: var(--text-muted);
}
.value-badge {
  background: var(--accent-muted);
  color: var(--accent);
  padding: 1px 8px;
  border-radius: var(--radius-full);
  font-size: var(--text-xs);
  font-weight: 700;
  font-family: var(--font-mono);
  border: 1px solid var(--accent-border);
}

/* ---- Range Hints ---- */
.range-hints {
  display: flex;
  justify-content: space-between;
  font-size: 10px;
  color: var(--text-muted);
  margin-top: var(--sp-1);
}

/* ---- Switch ---- */
.switch {
  position: relative;
  display: inline-block;
  width: 40px;
  height: 22px;
  flex-shrink: 0;
}
.switch input { display: none; }
.slider {
  position: absolute;
  cursor: pointer;
  inset: 0;
  background: var(--surface-input);
  border: 1px solid var(--border-default);
  transition: all var(--duration-base) ease;
}
.slider.round { border-radius: var(--radius-full); }
.slider::before {
  content: '';
  position: absolute;
  height: 16px;
  width: 16px;
  left: 2px;
  bottom: 2px;
  background: var(--text-secondary);
  transition: all var(--duration-base) ease;
  border-radius: 50%;
}
.switch input:checked + .slider {
  background: var(--accent);
  border-color: var(--accent);
}
.switch input:checked + .slider::before {
  transform: translateX(18px);
  background: var(--text-inverse);
}

/* ---- Range Slider ---- */
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
input[type="range"]:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

/* ---- Lightbox ---- */
.lightbox {
  position: fixed;
  inset: 0;
  z-index: 100000;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.85);
  backdrop-filter: blur(10px);
  cursor: zoom-out;
  pointer-events: auto;
}
.lightbox-img {
  max-width: 90%;
  max-height: 85%;
  object-fit: contain;
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-xl);
}
.lightbox-hint {
  color: var(--text-muted);
  font-size: var(--text-sm);
  margin-top: var(--sp-4);
}
.lightbox-enter-active,
.lightbox-leave-active { transition: opacity 0.2s ease; }
.lightbox-enter-from,
.lightbox-leave-to { opacity: 0; }

/* ---- Animations ---- */
@keyframes spin { to { transform: rotate(360deg); } }

/* ---- Scoped form group ---- */
.ss-group { margin: 0; }
</style>
