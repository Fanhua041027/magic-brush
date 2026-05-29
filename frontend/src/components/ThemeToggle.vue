<template>
  <button class="theme-toggle" @click="toggleTheme" :title="currentTheme === 'dark' ? '切换到浅色模式' : '切换到深色模式'">
    <Icon :name="currentTheme === 'dark' ? 'sun' : 'moon'" :size="15" />
  </button>
</template>

<script setup>
import { currentTheme, toggleTheme } from '../services/theme'
import Icon from './Icon.vue'
</script>

<style scoped>
.theme-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: var(--radius-sm);
  border: none;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition: all var(--duration-fast) ease;
}
.theme-toggle:hover {
  background: var(--surface-card-hover);
  color: var(--text-primary);
}
</style>
