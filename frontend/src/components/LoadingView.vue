<template>
  <div class="loading-container">
    <div class="loader-content">
      <div class="orb-container">
        <div class="orb"></div>
        <div class="ring ring-1"></div>
        <div class="ring ring-2"></div>
      </div>
      <div class="loading-text">
        <span class="text">深度思考中</span>
        <span class="timer">{{ formattedTime }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'

const startTime = ref(Date.now())
const currentTime = ref(Date.now())
let timerInterval = null

const formattedTime = computed(() => {
  const diff = currentTime.value - startTime.value
  const seconds = Math.floor(diff / 1000)
  const ms = Math.floor((diff % 1000) / 10)
  return `${seconds}.${ms.toString().padStart(2, '0')}s`
})

onMounted(() => {
  timerInterval = setInterval(() => { currentTime.value = Date.now() }, 30)
})

onUnmounted(() => { if (timerInterval) clearInterval(timerInterval) })
</script>

<style scoped>
.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  width: 100%;
  background: transparent;
  animation: containerFadeIn 0.4s var(--ease-out);
}

.loader-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--sp-6);
}

.orb-container {
  position: relative;
  width: 80px;
  height: 80px;
  display: flex;
  justify-content: center;
  align-items: center;
}

.orb {
  width: 32px;
  height: 32px;
  background: var(--accent-gradient);
  border-radius: 50%;
  box-shadow: 0 0 24px var(--accent-glow), 0 0 48px var(--accent-muted);
  animation: orbBreathe 3s ease-in-out infinite;
  z-index: 2;
}

.ring {
  position: absolute;
  border-radius: 50%;
  border: 1.5px solid transparent;
  border-top-color: var(--accent);
  border-right-color: var(--accent-muted);
}

.ring-1 {
  width: 52px;
  height: 52px;
  animation: ringSpin 2.5s linear infinite;
}

.ring-2 {
  width: 74px;
  height: 74px;
  border-top-color: var(--accent-border);
  border-left-color: var(--accent);
  animation: ringSpin 3.5s linear infinite reverse;
}

.loading-text {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
}

.text {
  font-size: var(--text-base);
  font-weight: var(--weight-semibold);
  color: var(--text-primary);
  letter-spacing: 0.5px;
}

.timer {
  font-family: var(--font-mono);
  font-size: var(--text-sm);
  font-weight: var(--weight-semibold);
  color: var(--accent);
  background: var(--accent-muted);
  padding: var(--sp-1) var(--sp-3);
  border-radius: var(--radius-sm);
  border: 1px solid var(--accent-border);
}

@keyframes containerFadeIn {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}

@keyframes orbBreathe {
  0%, 100% {
    transform: scale(0.92);
    box-shadow: 0 0 20px var(--accent-glow), 0 0 40px var(--accent-muted);
  }
  50% {
    transform: scale(1.08);
    box-shadow: 0 0 32px var(--accent-glow), 0 0 64px var(--accent-muted);
  }
}

@keyframes ringSpin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}
</style>
