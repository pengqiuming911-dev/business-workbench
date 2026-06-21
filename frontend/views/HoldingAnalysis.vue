<template>
  <div class="holding-analysis-page">
    <div class="page-header">
      <h1 class="text-page-title">产品&持仓</h1>
      <p class="text-body">数据自动同步自航班服务交易总表</p>
    </div>

    <div class="tab-bar">
      <button
        class="tab-btn"
        :class="{ active: activeTab === 'product' }"
        @click="activeTab = 'product'"
      >产品分析</button>
      <button
        class="tab-btn"
        :class="{ active: activeTab === 'customer' }"
        @click="activeTab = 'customer'"
      >客户持有分析</button>
    </div>

    <ProductAnalysis v-if="activeTab === 'product'" />
    <CustomerHolding v-else />
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import ProductAnalysis from './ProductAnalysis.vue'
import CustomerHolding from './CustomerHolding.vue'

const route = useRoute()
const router = useRouter()

function normalizeTab(value) {
  return value === 'customer' ? 'customer' : 'product'
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

.holding-analysis-page {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.holding-analysis-page > .page-header {
  flex-shrink: 0;
}

.holding-analysis-page > .tab-bar {
  flex-shrink: 0;
}

:deep(.product-analysis-page),
:deep(.customer-holding-page) {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
</style>
