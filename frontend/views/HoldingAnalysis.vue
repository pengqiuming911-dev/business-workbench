<template>
  <WorkbenchLayout>
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
  </WorkbenchLayout>
</template>

<script setup>
import { ref } from 'vue'
import WorkbenchLayout from '../components/WorkbenchLayout.vue'
import ProductAnalysis from './ProductAnalysis.vue'
import CustomerHolding from './CustomerHolding.vue'

const activeTab = ref('product')
</script>

<style scoped>
:deep(.workbench-main) {
  max-width: none;
}

.tab-bar {
  display: flex;
  gap: 4px;
  margin-bottom: 24px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  padding: 4px;
  width: fit-content;
}

.tab-btn {
  min-height: 40px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 20px;
  border: none;
  border-radius: var(--radius);
  background: transparent;
  color: var(--ink-soft);
  font-size: 14px;
  font-weight: 750;
  cursor: pointer;
  transition: background 150ms ease, color 150ms ease;
}

.tab-btn:hover {
  background: var(--bg-hover);
  color: var(--ink);
}

.tab-btn.active {
  background: var(--brand);
  color: #fff;
}
</style>
