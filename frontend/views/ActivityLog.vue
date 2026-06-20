<template>
  <div class="activity-log-page">
    <div class="page-header">
      <h1 class="text-page-title">操作日志</h1>
      <div class="filter-bar">
        <button
          v-for="f in filters"
          :key="f.value"
          class="btn"
          :class="activeFilter === f.value ? 'btn-primary' : 'btn-outline'"
          @click="setFilter(f.value)"
        >{{ f.label }}</button>
      </div>
    </div>

    <div class="panel-card">
      <div v-if="loading" class="loading-state">加载中...</div>
      <ActivityTimeline v-else :logs="logs" />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import ActivityTimeline from '../components/ActivityTimeline.vue'

const logs = ref([])
const loading = ref(true)
const activeFilter = ref('')

const filters = [
  { label: '全部', value: '' },
  { label: '同步', value: 'sync' },
  { label: '查询', value: 'query' },
  { label: '导出', value: 'export' },
]

async function load() {
  loading.value = true
  try {
    const url = activeFilter.value
      ? `/api/activity-logs?type=${activeFilter.value}`
      : '/api/activity-logs'
    const res = await fetch(url)
    const data = await res.json()
    logs.value = data.logs || []
  } catch {
    logs.value = []
  } finally {
    loading.value = false
  }
}

function setFilter(v) {
  activeFilter.value = v
  load()
}

onMounted(load)
</script>

<style scoped>
.page-header {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
  gap: 16px;
}

.filter-bar {
  display: flex;
  gap: 6px;
  padding: 0;
  background: none;
  border: none;
  border-radius: 0;
  box-shadow: none;
  margin-bottom: 0;
}
</style>