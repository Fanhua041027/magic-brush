<template>
  <Transition name="dock-slide">
    <div v-if="screenshots.length > 0" class="screenshot-dock" style="--wails-draggable: no-drag">
      <div class="dock-label">
        <Icon name="camera" :size="14" />
        <span class="dock-count">{{ screenshots.length }}/3</span>
      </div>
      <div class="dock-divider" />
      <div class="dock-thumbs">
        <div v-for="(img, i) in screenshots" :key="i" class="thumb-card">
          <img :src="img" class="thumb-img" />
        </div>
      </div>
      <div class="dock-divider" />
      <div class="dock-hints">
        <div class="dock-hint">
          <kbd class="dock-kbd">{{ deleteShortcut }}</kbd>
          <span>删除</span>
        </div>
        <div class="dock-hint">
          <kbd class="dock-kbd">{{ sendShortcut }}</kbd>
          <span>发送</span>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import Icon from './Icon.vue'
import { useSettingsStore } from '../stores/settings'
import { onMany } from '../services/events'

const settingsStore = useSettingsStore()
const sendShortcut = settingsStore.sendShortcut
const deleteShortcut = settingsStore.deleteShortcut

const screenshots = ref([])

let disposeEvents = null

onMounted(() => {
  disposeEvents = onMany({
    'screenshot-taken': (base64) => {
      screenshots.value.push(base64)
    },
    'screenshot-removed': (index) => {
      screenshots.value.splice(index, 1)
    },
    'screenshots-cleared': () => {
      screenshots.value = []
    },
    'start-solving': () => {
      screenshots.value = []
    },
  })
})

onUnmounted(() => {
  disposeEvents?.()
  disposeEvents = null
})

defineExpose({ screenshots })
</script>

<style scoped>
.screenshot-dock {
  position: fixed;
  bottom: 33px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 100;
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  padding: var(--sp-3) var(--sp-4);
  background: var(--surface-elevated);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  backdrop-filter: blur(24px);
  flex-shrink: 0;
  pointer-events: auto;
}

.dock-label {
  display: flex;
  align-items: center;
  gap: var(--sp-1-5);
  color: var(--text-secondary);
  font-size: var(--text-sm);
  font-weight: var(--weight-medium);
  white-space: nowrap;
}

.dock-count {
  color: var(--accent);
  font-weight: var(--weight-semibold);
}

.dock-divider {
  width: 1px;
  height: 24px;
  background: var(--border-default);
  flex-shrink: 0;
}

.dock-thumbs {
  display: flex;
  gap: var(--sp-2);
  flex: 1;
  min-width: 0;
}

.thumb-card {
  position: relative;
  width: 56px;
  height: 40px;
  border-radius: var(--radius-sm);
  overflow: hidden;
  border: 1.5px solid var(--border-default);
  flex-shrink: 0;
}

.thumb-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: calc(var(--radius-sm) - 1px);
}

.dock-hints {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  flex-shrink: 0;
}

.dock-hint {
  display: flex;
  align-items: center;
  gap: var(--sp-1-5);
  color: var(--text-tertiary);
  font-size: var(--text-sm);
  white-space: nowrap;
}

.dock-kbd {
  font-size: var(--text-xs);
  font-family: var(--font-mono);
  background: var(--surface-card);
  color: var(--text-secondary);
  padding: 1px 5px;
  border-radius: var(--radius-xs);
  border: 1px solid var(--border-default);
}

/* Slide-up animation */
.dock-slide-enter-active {
  transition: all var(--duration-slow) var(--ease-spring);
}
.dock-slide-leave-active {
  transition: all var(--duration-base) var(--ease-smooth);
}
.dock-slide-enter-from {
  transform: translateX(-50%) translateY(100%);
  opacity: 0;
}
.dock-slide-leave-to {
  transform: translateX(-50%) translateY(100%);
  opacity: 0;
}
</style>
