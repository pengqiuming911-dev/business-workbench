<template>
  <div class="panel-card stat-card">
    <p class="stat-label">{{ title }}</p>
    <p class="stat-value">{{ displayValue }}</p>
    <div v-if="trend != null" class="stat-trend" :class="trendClass">
      <TrendingUp v-if="trend > 0" :size="14" />
      <TrendingDown v-else-if="trend < 0" :size="14" />
      <span class="trend-value">{{ trend > 0 ? '+' : '' }}{{ formatTrend(trend) }}%</span>
    </div>
    <div v-else class="stat-trend stat-trend-neutral">
      <span class="trend-dash">—</span>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { TrendingUp, TrendingDown } from '@lucide/vue'

const props = defineProps({
  title: { type: String, required: true },
  value: { type: [String, Number], required: true },
  trend: { type: Number, default: null },
})

const displayValue = computed(() => {
  if (typeof props.value === 'number') return props.value.toLocaleString('zh-CN')
  return props.value
})

const trendClass = computed(() => {
  if (props.trend > 0) return 'trend-up'
  if (props.trend < 0) return 'trend-down'
  return 'trend-neutral'
})

function formatTrend(v) {
  return Math.abs(v).toFixed(1)
}
</script>

<style scoped>
/* Depends on .panel-card defined in global main.css */
.stat-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.stat-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--ink-soft);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.stat-value {
  font-size: 28px;
  font-weight: 800;
  color: var(--ink-strong);
  line-height: 1.1;
  font-family: var(--font-mono);
}

.stat-trend {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  font-weight: 600;
  margin-top: 4px;
}

.trend-up { color: var(--success); }
.trend-down { color: var(--danger); }
.trend-neutral { color: var(--ink-faint); }

.trend-dash { letter-spacing: 0.1em; color: var(--ink-faint); }
</style>