<template>
  <div class="history-item" :class="{ active: isActive }" @click="$emit('select')">
    <div class="history-header">
      <span class="history-tag">{{ isFirst ? '当前问题' : '历史问题' }}</span>
      <span v-if="roundsCount > 1" class="rounds-badge">{{ roundsCount }} 轮</span>
      <div class="menu-trigger" @click.stop="toggleMenu" ref="menuTriggerRef">
        <Icon name="more-vertical" :size="14" class="dots-icon" />
      </div>
    </div>
    <div class="history-preview" v-html="previewHtml"></div>
    <div class="history-time">{{ time }}</div>
  </div>

  <Teleport to="body">
    <Transition name="menu-fade">
      <div v-if="menuOpen" class="history-menu" :style="menuStyle" @click.stop>
        <div class="menu-item" @click="handleExportImage">
          <Icon name="image" :size="15" />
          <span>导出为图片</span>
        </div>
        <div class="menu-divider"></div>
        <div class="menu-item danger" @click="handleDelete">
          <Icon name="trash" :size="15" />
          <span>删除此会话</span>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import Icon from './Icon.vue'

const props = defineProps({
  summary: { type: String, default: '' },
  time: { type: String, default: '' },
  isActive: { type: Boolean, default: false },
  isFirst: { type: Boolean, default: false },
  previewHtml: { type: String, default: '' },
  roundsCount: { type: Number, default: 1 }
})

const emit = defineEmits(['select', 'delete', 'export-image'])

const menuOpen = ref(false)
const menuTriggerRef = ref(null)
const menuStyle = reactive({ top: '0px', left: '0px' })

function toggleMenu() {
  if (!menuOpen.value && menuTriggerRef.value) {
    const rect = menuTriggerRef.value.getBoundingClientRect()
    menuStyle.top = `${rect.top}px`
    menuStyle.left = `${rect.right + 8}px`
  }
  menuOpen.value = !menuOpen.value
}

function closeMenu() { menuOpen.value = false }
function handleDelete() { emit('delete'); closeMenu() }
function handleExportImage() { emit('export-image'); closeMenu() }

function handleClickOutside(event) {
  if (menuTriggerRef.value && !menuTriggerRef.value.contains(event.target)) closeMenu()
}

onMounted(() => { document.addEventListener('click', handleClickOutside) })
onUnmounted(() => { document.removeEventListener('click', handleClickOutside) })
</script>

<style scoped>
.history-item {
  background: transparent;
  border-radius: var(--radius-md);
  padding: var(--sp-3);
  cursor: pointer;
  transition: all var(--duration-fast) ease;
  border: 1px solid var(--border-subtle);
  position: relative;
  overflow: visible;
  max-height: 120px;
}
.history-item::before {
  content: '';
  position: absolute;
  left: 0; top: 50%;
  transform: translateY(-50%);
  width: 3px; height: 0;
  background: var(--accent);
  border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
  transition: height var(--duration-fast) ease;
}
.history-item:hover {
  background: var(--surface-card);
  border-color: var(--border-default);
  transform: translateX(2px);
}
.history-item:hover::before { height: 40%; }
.history-item.active {
  background: var(--accent-muted);
  border-color: var(--accent-border);
}
.history-item.active::before { height: 60%; }

.history-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--sp-2);
}
.history-tag {
  font-size: var(--text-xs);
  padding: 2px var(--sp-2);
  border-radius: var(--radius-sm);
  background: var(--surface-card-hover);
  color: var(--text-muted);
  font-weight: var(--weight-semibold);
  letter-spacing: 0.3px;
}
.history-item.active .history-tag {
  background: var(--accent-muted);
  color: var(--accent);
}
.rounds-badge {
  font-size: 10px;
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  background: var(--accent-muted);
  color: var(--accent);
  font-weight: var(--weight-semibold);
  margin-left: 4px;
}
.history-preview {
  font-size: var(--text-sm);
  color: var(--text-primary);
  line-height: var(--leading-normal);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.history-time {
  font-size: var(--text-xs);
  color: var(--text-muted);
  margin-top: var(--sp-2);
  text-align: right;
  font-family: var(--font-mono);
}

/* Menu trigger */
.menu-trigger { position: relative; }
.dots-icon {
  color: var(--text-muted);
  cursor: pointer;
  padding: var(--sp-1);
  border-radius: var(--radius-sm);
  transition: all var(--duration-fast) ease;
}
.dots-icon:hover {
  color: var(--text-primary);
}

/* Context menu */
.history-menu {
  position: fixed;
  background: var(--surface-popover);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  z-index: var(--z-tooltip);
  min-width: 160px;
  padding: var(--sp-1);
  backdrop-filter: blur(16px);
}
.menu-item {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  padding: var(--sp-2) var(--sp-3);
  font-size: var(--text-sm);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--duration-fast) ease;
  border-radius: var(--radius-sm);
}
.menu-item:hover {
  background: var(--surface-card-hover);
  color: var(--text-primary);
}
.menu-item.danger { color: var(--color-error); }
.menu-item.danger:hover { background: var(--error-bg); }
.menu-divider {
  height: 1px;
  background: var(--border-subtle);
  margin: var(--sp-1) 0;
}

/* Menu transition */
.menu-fade-enter-active, .menu-fade-leave-active {
  transition: all 0.2s var(--ease-out);
}
.menu-fade-enter-from, .menu-fade-leave-to {
  opacity: 0;
  transform: translateY(-4px) scale(0.96);
}
</style>
