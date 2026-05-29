<template>
  <div class="resize-handle" @mousedown="startResize" title="拖动调整窗口大小">
    <svg class="resize-icon" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path d="M22 22L12 22M22 22L22 12M22 22L12 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
    </svg>
  </div>
</template>

<script setup>
import { WindowGetSize, WindowSetSize } from '../../wailsjs/runtime/runtime'
import { api } from '../services/api'

const MIN_WIDTH = 840
const MIN_HEIGHT = 700
const MAX_WIDTH = 1024
const MAX_HEIGHT = 768

let startX = 0
let startY = 0
let startWidth = 0
let startHeight = 0

async function startResize(e) {
  e.preventDefault()
  const size = await WindowGetSize()
  startWidth = size.w
  startHeight = size.h
  startX = e.screenX
  startY = e.screenY
  document.addEventListener('mousemove', onResize)
  document.addEventListener('mouseup', stopResize)
  document.body.style.cursor = 'nwse-resize'
  document.body.style.userSelect = 'none'
}

function onResize(e) {
  const deltaX = e.screenX - startX
  const deltaY = e.screenY - startY
  const newWidth = Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, startWidth + deltaX))
  const newHeight = Math.min(MAX_HEIGHT, Math.max(MIN_HEIGHT, startHeight + deltaY))
  WindowSetSize(newWidth, newHeight)
}

async function stopResize() {
  document.removeEventListener('mousemove', onResize)
  document.removeEventListener('mouseup', stopResize)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
  try {
    const size = await WindowGetSize()
    const settings = await api.getSettings()
    settings.windowWidth = size.w
    settings.windowHeight = size.h
    await api.updateSettings(JSON.stringify(settings))
  } catch (e) {
    console.error('保存窗口尺寸失败:', e)
  }
}
</script>

<style scoped>
.resize-handle {
  position: fixed;
  bottom: 4px;
  right: 4px;
  width: 20px;
  height: 20px;
  cursor: nwse-resize;
  z-index: 99999;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  transition: all var(--duration-fast) ease;
  pointer-events: auto;
  opacity: 0.4;
}

.resize-handle:hover {
  opacity: 1;
  background: var(--surface-card-hover);
}

.resize-handle:active {
  opacity: 1;
  background: var(--surface-card-hover);
}

.resize-icon {
  width: 14px;
  height: 14px;
  color: var(--text-tertiary);
  transition: color var(--duration-fast) ease;
}

.resize-handle:hover .resize-icon {
  color: var(--text-secondary);
}
</style>