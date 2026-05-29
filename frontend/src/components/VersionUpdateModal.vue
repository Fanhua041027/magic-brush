<template>
  <Transition name="overlay-fade">
    <div v-if="show" class="version-overlay">
      <div class="version-card">
        <!-- 顶部装饰条 -->
        <div class="card-accent-bar"></div>

        <!-- 图标 -->
        <div class="version-visual">
          <div class="icon-bg">
            <div class="icon-ring"></div>
            <div class="icon-box">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <circle cx="12" cy="12" r="10"/>
                <line x1="12" y1="8" x2="12" y2="12"/>
                <line x1="12" y1="16" x2="12.01" y2="16"/>
              </svg>
            </div>
          </div>
        </div>

        <!-- 内容 -->
        <h2 class="v-title">{{ title || '版本不可用' }}</h2>
        <p class="v-message">{{ message || '当前版本已停止服务，请更新到最新版本后继续使用。' }}</p>

        <!-- 按钮 -->
        <button class="v-quit-btn" @click.stop="handleQuit">
          <span>我知道了</span>
        </button>
      </div>
    </div>
  </Transition>
</template>

<script setup>
defineProps({
  show: Boolean,
  title: String,
  message: String
})

const emit = defineEmits(['quit'])

const handleQuit = () => {
  console.log('[VersionModal] Quit button clicked')
  emit('quit')
}
</script>

<style scoped>
.version-overlay {
  position: fixed;
  inset: 0;
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 11000;
  /* 完全不透明的深色背景 */
  background: #0f0f17;
  pointer-events: auto;
}

.version-card {
  width: 400px;
  max-width: 90vw;
  background: #1a1a28;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 20px;
  padding: 0 40px 40px;
  text-align: center;
  position: relative;
  overflow: hidden;
  box-shadow:
    0 0 0 1px rgba(255, 255, 255, 0.04),
    0 20px 60px rgba(0, 0, 0, 0.5),
    0 0 120px rgba(99, 102, 241, 0.08);
}

/* 顶部强调条 */
.card-accent-bar {
  width: 100%;
  height: 3px;
  background: linear-gradient(90deg, transparent 0%, var(--accent, #6366f1) 50%, transparent 100%);
  margin-bottom: 36px;
}

/* 图标 */
.version-visual {
  display: flex;
  justify-content: center;
  margin-bottom: 28px;
}
.icon-bg {
  position: relative;
  width: 72px;
  height: 72px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.icon-ring {
  position: absolute;
  inset: 0;
  border: 1.5px solid rgba(239, 68, 68, 0.25);
  border-radius: 50%;
  animation: ringBreath 3s ease-in-out infinite;
}
.icon-box {
  width: 52px;
  height: 52px;
  background: rgba(239, 68, 68, 0.12);
  border: 1px solid rgba(239, 68, 68, 0.20);
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #f87171;
  z-index: 1;
}
.icon-box svg {
  width: 26px;
  height: 26px;
}

/* 标题 */
.v-title {
  font-size: 22px;
  font-weight: 700;
  color: #f0f0f5;
  margin: 0 0 12px;
  letter-spacing: 0.5px;
}

/* 描述 */
.v-message {
  font-size: 14px;
  line-height: 1.7;
  color: #9294a0;
  margin: 0 0 32px;
  padding: 0 8px;
}

/* 按钮 */
.v-quit-btn {
  width: 100%;
  height: 46px;
  background: var(--accent, #6366f1);
  border: none;
  border-radius: 12px;
  color: #fff;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
  letter-spacing: 0.5px;
}
.v-quit-btn:hover {
  filter: brightness(1.15);
  transform: translateY(-1px);
  box-shadow: 0 6px 24px rgba(99, 102, 241, 0.35);
}
.v-quit-btn:active {
  transform: translateY(0);
}

/* 过渡动画 */
.overlay-fade-enter-active { transition: opacity 0.35s ease; }
.overlay-fade-leave-active { transition: opacity 0.25s ease; }
.overlay-fade-enter-from, .overlay-fade-leave-to { opacity: 0; }

.overlay-fade-enter-active .version-card {
  animation: cardIn 0.4s cubic-bezier(0.16, 1, 0.3, 1);
}
@keyframes cardIn {
  from { opacity: 0; transform: scale(0.92) translateY(16px); }
  to { opacity: 1; transform: scale(1) translateY(0); }
}
@keyframes ringBreath {
  0%, 100% { transform: scale(1); opacity: 0.5; }
  50% { transform: scale(1.12); opacity: 0.8; }
}
</style>