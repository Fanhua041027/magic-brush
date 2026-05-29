/**
 * 主题管理服务
 * 主题通过后端 config 持久化，localStorage 作为启动快速缓存（防闪烁）
 * 默认主题：light
 */
import { ref, watch } from 'vue'

const THEME_KEY = 'q-solver-theme'

// 启动时先从 localStorage 快速加载（避免白→黑闪烁），后续由 settings store 同步后端值
export const currentTheme = ref(loadThemeFromCache())

function loadThemeFromCache() {
  if (typeof localStorage !== 'undefined') {
    return localStorage.getItem(THEME_KEY) || 'light'
  }
  return 'light'
}

export function toggleTheme() {
  currentTheme.value = currentTheme.value === 'dark' ? 'light' : 'dark'
}

export function setTheme(theme) {
  if (theme && (theme === 'dark' || theme === 'light')) {
    currentTheme.value = theme
  }
}

export function applyTheme(theme) {
  document.documentElement.setAttribute('data-theme', theme)
  // 同步写入 localStorage 作为快速缓存
  if (typeof localStorage !== 'undefined') {
    localStorage.setItem(THEME_KEY, theme)
  }
  // 触发自定义事件，通知其他模块
  window.dispatchEvent(new CustomEvent('theme-changed', { detail: { theme } }))
}

// 自动应用
watch(currentTheme, (val) => applyTheme(val), { immediate: true })
