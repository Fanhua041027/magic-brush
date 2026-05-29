<template>
  <div class="domain-selector">
    <label class="domain-label">角色设定 / 场景</label>

    <div class="category-tabs" v-if="categories && categories.length > 0">
      <button v-for="cat in categories" :key="cat.id" class="cat-btn"
        :class="{ active: currentCategory === cat.id }" @click="currentCategory = cat.id">
        {{ cat.label }}
      </button>
    </div>

    <div v-if="!categories || categories.length === 0" class="loading-state">
      加载配置中...
    </div>

    <div class="domain-list" v-else>
      <button v-for="item in currentItems" :key="item.id" class="domain-item"
        :class="{ active: modelValue === item.id }" @click="selectDomain(item.id)">
        <span class="item-icon">{{ item.icon }}</span>
        <div class="item-body">
          <span class="item-title">{{ item.label }}</span>
          <span class="item-desc">{{ item.description }}</span>
        </div>
        <span class="item-check" v-if="modelValue === item.id">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
        </span>
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  categories: { type: Array, default: () => [] }
})

const emit = defineEmits(['update:modelValue'])
const currentCategory = ref('')

watch(() => [props.categories, props.modelValue], () => {
  if (!currentCategory.value && props.categories.length > 0) {
    const foundCat = props.categories.find(cat => cat.items.some(item => item.id === props.modelValue))
    if (foundCat) currentCategory.value = foundCat.id
    else currentCategory.value = props.categories[0].id
  }
}, { immediate: true })

const currentItems = computed(() => {
  if (!props.categories) return []
  const cat = props.categories.find(c => c.id === currentCategory.value)
  return cat ? cat.items : []
})

function selectDomain(id) { emit('update:modelValue', id) }
</script>

<style scoped>
.domain-selector {
  margin-top: var(--sp-2);
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
}

.domain-label {
  font-size: var(--text-sm);
  font-weight: var(--weight-semibold);
  color: var(--text-secondary);
}

/* ---- Category Tabs — pill style ---- */
.category-tabs {
  display: flex;
  align-items: center;
  gap: 4px;
  overflow-x: auto;
  overflow-y: hidden;
  padding: 4px 6px;
  background: var(--surface-input);
  border-radius: var(--radius-md);
  scrollbar-width: none;
  min-height: 40px;
}
.category-tabs::-webkit-scrollbar { height: 0; }

.cat-btn {
  font-size: var(--text-xs);
  padding: 0 var(--sp-5);
  height: 32px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  color: var(--text-muted);
  white-space: nowrap;
  transition: all var(--duration-fast) ease;
  font-weight: var(--weight-medium);
  background: transparent;
  border: none;
  flex-shrink: 0;
  line-height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;
}
.cat-btn:hover {
  color: var(--text-primary);
  background: var(--surface-card-hover);
}
.cat-btn.active {
  background: var(--surface-elevated);
  color: var(--accent);
  font-weight: var(--weight-semibold);
  box-shadow: var(--shadow-sm);
}

/* ---- Domain list — compact rows ---- */
.domain-list {
  display: flex;
  flex-direction: column;
  gap: var(--sp-1-5);
  max-height: 220px;
  overflow-y: auto;
  padding-right: 2px;
  scrollbar-width: thin;
}
.domain-list::-webkit-scrollbar { width: 3px; }
.domain-list::-webkit-scrollbar-thumb { background: var(--scrollbar-thumb); border-radius: 4px; }

.domain-item {
  display: flex;
  align-items: center;
  gap: var(--sp-2-5);
  padding: var(--sp-2-5) var(--sp-3);
  background: var(--surface-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--duration-fast) ease;
  text-align: left;
  min-height: 44px;
}
.domain-item:hover {
  background: var(--surface-card-hover);
  border-color: var(--border-default);
}
.domain-item.active {
  background: var(--accent-muted);
  border-color: var(--accent-border);
}

.item-icon {
  font-size: 18px;
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--surface-card);
  border-radius: var(--radius-sm);
}
.domain-item.active .item-icon {
  background: var(--accent-muted);
}

.item-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.item-title {
  font-size: var(--text-sm);
  font-weight: var(--weight-semibold);
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.3;
}
.domain-item.active .item-title { color: var(--accent); }

.item-desc {
  font-size: 11px;
  color: var(--text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.3;
}

.item-check {
  flex-shrink: 0;
  color: var(--accent);
  display: flex;
  align-items: center;
}

.loading-state {
  padding: var(--sp-5);
  text-align: center;
  color: var(--text-muted);
  font-size: var(--text-xs);
}
</style>
