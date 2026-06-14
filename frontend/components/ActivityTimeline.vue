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
.timeline { position: relative; padding-left: 24px; }
.timeline-entry { position: relative; padding-bottom: 16px; }
.timeline-entry:last-child { padding-bottom: 0; }

.timeline-dot {
  position: absolute;
  left: -24px;
  top: 4px;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--brand);
}
.timeline-dot.dot-sync { background: var(--brand); }
.timeline-dot.dot-query { background: var(--success); }
.timeline-dot.dot-export { background: var(--warning); }

.timeline-entry:not(:last-child)::before {
  content: '';
  position: absolute;
  left: -20px;
  top: 16px;
  bottom: -4px;
  width: 1px;
  background: var(--border-soft);
}

.timeline-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.timeline-time {
  font-size: 11px;
  color: var(--ink-faint);
  font-family: var(--font-mono);
}

.timeline-action {
  font-size: 14px;
  color: var(--ink-strong);
  font-weight: 500;
}

.timeline-detail {
  font-size: 12px;
  color: var(--ink-soft);
  margin-top: 2px;
}
</style>
