<template>
  <div class="holding-analysis-page">
    <h1 class="text-page-title">持有产品分析</h1>
    <p class="text-body" style="margin-bottom: 20px;">数据自动同步自航班服务交易总表</p>

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
