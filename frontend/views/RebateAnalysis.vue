<template>
  <div class="rebate-analysis-page">
    <h1 class="text-page-title">返费分析</h1>
    <p class="text-body" style="margin-bottom: 20px;">在同一页面切换查看待返费和已返费数据。</p>

    <div class="tab-bar">
      <button
        class="tab-btn"
        :class="{ active: activeTab === 'pending' }"
        @click="activeTab = 'pending'"
      >待返费分析</button>
      <button
        class="tab-btn"
        :class="{ active: activeTab === 'completed' }"
        @click="activeTab = 'completed'"
      >已返费分析</button>
    </div>

    <RebatePending v-if="activeTab === 'pending'" embedded />
    <RebateCompleted v-else embedded />
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import RebatePending from './RebatePending.vue'
import RebateCompleted from './RebateCompleted.vue'

const route = useRoute()
const router = useRouter()

function normalizeTab(value) {
  return value === 'completed' ? 'completed' : 'pending'
}

const activeTab = ref(normalizeTab(route.query.tab))

watch(
  () => route.query.tab,
  (value) => {
    activeTab.value = normalizeTab(value)
  },
)

watch(activeTab, (value) => {
  const next = normalizeTab(value)
  if (route.query.tab === next) return
  router.replace({
    query: {
      ...route.query,
      tab: next,
    },
  })
})
</script>

<style scoped>
:deep(.workbench-main) {
  max-width: none;
}

.tab-bar {
  display: flex;
  gap: 4px;
  margin-bottom: 28px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  padding: 5px;
  width: fit-content;
  box-shadow: var(--shadow-sm);
}

.tab-btn {
  min-height: 40px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 22px;
  border: none;
  border-radius: 10px;
  background: transparent;
  color: var(--ink-soft);
  font-size: 16px;
  font-weight: 600;
  letter-spacing: 0.01em;
  cursor: pointer;
  transition: background 200ms ease, color 200ms ease;
}

.tab-btn:hover {
  background: var(--bg-hover);
  color: var(--ink-strong);
}

.tab-btn.active {
  background: var(--brand);
  color: #fff;
}
</style>
