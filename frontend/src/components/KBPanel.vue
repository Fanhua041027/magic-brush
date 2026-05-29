<template>
  <div class="kb-panel" :class="{ collapsed }">
    <div class="kb-header" @click="collapsed = !collapsed">
      <svg class="kb-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20" />
        <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z" />
      </svg>
      <span class="kb-title">知识库</span>
      <span v-if="sectionCount > 0" class="kb-count">{{ sectionCount }} 篇</span>
      <svg class="chevron" :class="{ open: !collapsed }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polyline points="6 9 12 15 18 9" />
      </svg>
    </div>

    <div v-if="!collapsed" class="kb-body">
      <div v-if="!kbReady" class="kb-empty">
        <p>未加载知识库</p>
        <p class="kb-hint">在设置中配置知识库路径</p>
      </div>

      <template v-else>
        <div class="kb-search-box">
          <input
            v-model="query"
            class="kb-input"
            placeholder="搜索知识库..."
            @keydown.enter="doSearch"
          />
          <button class="kb-search-btn" @click="doSearch">搜索</button>
        </div>

        <div v-if="isSearching" class="kb-searching">搜索中...</div>

        <div v-else-if="searchResults.length > 0" class="kb-results">
          <div
            v-for="(item, i) in searchResults"
            :key="i"
            class="kb-result-item"
            @click="$emit('select-result', item)"
            :title="item.content"
          >
            <div class="kr-source">{{ item.source }}</div>
            <div class="kr-header">{{ item.header }}</div>
            <div class="kr-score">{{ (item.score * 100).toFixed(0) }}%</div>
          </div>
        </div>

        <div v-else-if="query && !isSearching" class="kb-no-results">
          无匹配结果
        </div>
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../services/api'

defineEmits(['select-result'])

const collapsed = ref(true)
const kbReady = ref(false)
const sectionCount = ref(0)
const query = ref('')
const searchResults = ref([])
const isSearching = ref(false)

onMounted(async () => {
  try {
    const status = await api.getKBStatus()
    kbReady.value = status.ready || false
    sectionCount.value = status.section_count || 0
  } catch (e) {
    kbReady.value = false
  }
})

async function doSearch() {
  if (!query.value.trim()) return
  isSearching.value = true
  try {
    const result = await api.searchKB(query.value)
    searchResults.value = result.results || []
  } catch (e) {
    console.error('KB search error:', e)
    searchResults.value = []
  } finally {
    isSearching.value = false
  }
}
</script>

<style scoped>
.kb-panel {
  border-top: 1px solid var(--border-default);
  background: var(--surface-base);
}
.kb-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  cursor: pointer;
  user-select: none;
  transition: background var(--duration-fast) ease;
}
.kb-header:hover {
  background: var(--surface-card-hover);
}
.kb-icon {
  width: 16px;
  height: 16px;
  color: var(--text-muted);
}
.kb-title {
  flex: 1;
  font-size: var(--text-xs);
  font-weight: var(--weight-semibold);
  color: var(--text-secondary);
}
.kb-count {
  font-size: 10px;
  color: var(--text-muted);
  background: var(--surface-card);
  padding: 1px 6px;
  border-radius: 8px;
}
.chevron {
  width: 14px;
  height: 14px;
  color: var(--text-muted);
  transition: transform var(--duration-fast) ease;
}
.chevron.open {
  transform: rotate(180deg);
}
.kb-body {
  padding: 0 12px 8px;
}
.kb-empty {
  text-align: center;
  padding: 12px 0;
  color: var(--text-muted);
  font-size: var(--text-xs);
}
.kb-hint {
  font-size: 10px;
  margin-top: 4px;
}
.kb-search-box {
  display: flex;
  gap: 6px;
  margin-bottom: 8px;
}
.kb-input {
  flex: 1;
  padding: 4px 8px;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  background: var(--surface-card);
  color: var(--text-primary);
  font-size: var(--text-xs);
  outline: none;
}
.kb-input:focus {
  border-color: var(--accent);
}
.kb-search-btn {
  padding: 4px 10px;
  border: none;
  border-radius: var(--radius-sm);
  background: var(--accent);
  color: white;
  font-size: var(--text-xs);
  cursor: pointer;
}
.kb-search-btn:hover {
  background: var(--accent-hover);
}
.kb-searching,
.kb-no-results {
  text-align: center;
  color: var(--text-muted);
  font-size: 10px;
  padding: 8px 0;
}
.kb-results {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.kb-result-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background var(--duration-fast) ease;
}
.kb-result-item:hover {
  background: var(--surface-card-hover);
}
.kr-source {
  font-size: 9px;
  color: var(--accent);
  background: var(--accent-bg);
  padding: 0 4px;
  border-radius: 3px;
  white-space: nowrap;
}
.kr-header {
  flex: 1;
  font-size: var(--text-xs);
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.kr-score {
  font-size: 9px;
  color: var(--text-muted);
}
</style>
