<template>
  <div class="rebate-analysis-page">
    <div class="page-header">
      <h1 class="text-page-title">返费分析</h1>
      <p class="text-body">在同一页面切换查看待返费和已返费数据。</p>
    </div>

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

.rebate-analysis-page {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.rebate-analysis-page > .page-header {
  flex-shrink: 0;
}

.rebate-analysis-page > .tab-bar {
  flex-shrink: 0;
}

:deep(.rebate-pending-page),
:deep(.rebate-completed-page) {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
</style>
