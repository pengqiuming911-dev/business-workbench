<template>
  <div class="timeline">
    <div v-if="logs.length === 0" class="empty-state">暂无操作记录</div>
    <div v-for="log in logs" :key="log.id" class="timeline-entry">
      <div class="timeline-dot" :class="'dot-' + log.type"></div>
      <div class="timeline-content">
        <div class="timeline-header">
          <span class="timeline-time">{{ formatTime(log.createdAt) }}</span>
          <span class="badge" :class="badgeClass(log.type)">{{ typeLabel(log.type) }}</span>
        </div>
        <p class="timeline-action">{{ log.action }}</p>
        <p v-if="log.detail" class="timeline-detail">{{ log.detail }}</p>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  logs: { type: Array, default: () => [] }
})

function typeLabel(t) {
  return { sync: '同步', query: '查询', export: '导出' }[t] || t
}

function badgeClass(t) {
  return { sync: 'badge-blue', query: 'badge-green', export: 'badge-amber' }[t] || 'badge-blue'
}

function formatTime(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  return d.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}
</script>

<style scoped>
.timeline { position: relative; padding-left: 28px; }
.timeline-entry { position: relative; padding-bottom: 20px; }
.timeline-entry:last-child { padding-bottom: 0; }

.timeline-dot {
  position: absolute;
  left: -28px;
  top: 5px;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--brand);
  box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.12);
}
.timeline-dot.dot-sync { background: var(--brand); box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.12); }
.timeline-dot.dot-query { background: var(--success); box-shadow: 0 0 0 3px rgba(15, 159, 110, 0.12); }
.timeline-dot.dot-export { background: var(--warning); box-shadow: 0 0 0 3px rgba(198, 120, 17, 0.12); }

.timeline-entry:not(:last-child)::before {
  content: '';
  position: absolute;
  left: -24px;
  top: 18px;
  bottom: -4px;
  width: 1.5px;
  background: var(--border-soft);
}

.timeline-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 6px;
}

.timeline-time {
  font-size: 11.5px;
  color: var(--ink-faint);
  font-family: var(--font-mono);
  font-weight: 500;
}

.timeline-action {
  font-size: 14px;
  color: var(--ink-strong);
  font-weight: 600;
  letter-spacing: -0.005em;
}

.timeline-detail {
  font-size: 12.5px;
  color: var(--ink-soft);
  margin-top: 3px;
  line-height: 1.5;
}
</style>
