<template>
  <div class="round-divider">
    <div class="divider-line"></div>
    <span class="divider-label">第 {{ round }} 轮</span>
    <div class="divider-line"></div>
  </div>
</template>

<script setup>
defineProps({
  round: { type: Number, required: true }
})
</script>

<style scoped>
.round-divider {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  margin: var(--sp-6) 0;
  user-select: none;
}
.divider-line {
  flex: 1;
  height: 1px;
  background: linear-gradient(90deg, transparent, var(--border-default), transparent);
}
.divider-label {
  font-size: var(--text-xs);
  font-weight: var(--weight-semibold);
  color: var(--text-muted);
  white-space: nowrap;
  letter-spacing: 0.5px;
}
</style>
