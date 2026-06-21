<template>
  <div class="product-report-page">
    <div class="page-header">
      <h1 class="text-page-title">销售物料</h1>
    </div>

    <PanelCard title="物料展示">
      <div class="month-search">
        <span class="text-label">数据源：</span>
        <select v-model="selectedMonth" class="input" style="min-width: 200px; flex: 1;" @change="loadProducts">
          <option value="">-- 请选择月份 --</option>
          <option v-for="month in availableMonths" :key="month" :value="month">
            {{ month }}
          </option>
        </select>
        <span v-if="lastSyncTime" class="text-label">最后同步: {{ lastSyncTime }}</span>
      </div>

      <div v-if="error" class="error-msg">{{ error }}</div>

      <div v-if="loading" class="loading-state">正在加载产品结构...</div>
      <div v-else-if="products.length > 0" class="product-cards">
        <div v-for="product in products" :key="product.doc_token" class="product-card">
          <div class="card-header">
            <span class="card-icon">📋</span>
            <span class="card-title">{{ getDisplayName(product.doc_name) }}</span>
          </div>
          <div class="card-body">
            <template v-if="product.structured">
              <div v-for="(value, key) in product.structured" :key="key" class="info-row">
                <span class="info-key">{{ key }}</span>
                <span class="info-val" :class="{ multiline: key === '降敲' }">{{ value }}</span>
              </div>
            </template>
            <template v-else>
              <div class="raw-content">{{ product.raw_content || '无内容' }}</div>
            </template>
          </div>
        </div>
      </div>
      <div v-else-if="selectedMonth && !loading" class="empty-state">
        该月份暂无产品结构数据
      </div>
      <div v-else-if="!selectedMonth" class="empty-state">
        请选择月份查看产品结构，数据请在「数据准备」页面同步
      </div>
    </PanelCard>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import PanelCard from '../components/PanelCard.vue'

const selectedMonth = ref('')
const products = ref([])
const loading = ref(false)
const error = ref('')
const lastSyncTime = ref('')
const availableMonths = ref([])

function getDisplayName(docName) {
  return docName.replace(/^销售物料[：:\s]*/, '')
}

function extractMonth(doc) {
  const path = doc.parent_path || ''
  const match = path.match(/(\d{4})[年\s]*(\d{1,2})[月\s]*/)
  if (match) {
    const month = parseInt(match[2])
    return `${match[1]}年${month}月`
  }
  const nameMatch = doc.doc_name.match(/(\d{4})[年\s]*(\d{1,2})[月\s]*/)
  if (nameMatch) {
    const month = parseInt(nameMatch[2])
    return `${nameMatch[1]}年${month}月`
  }
  return '其他'
}

async function loadProducts() {
  if (!selectedMonth.value) {
    products.value = []
    return
  }

  loading.value = true
  error.value = ''

  try {
    const res = await fetch(`/api/drive/product-docs?month=${encodeURIComponent(selectedMonth.value)}`)
    if (!res.ok) throw new Error('加载失败')
    const data = await res.json()
    const filtered = data.filter(d => d.doc_name.includes('物料'))
    products.value = filtered
  } catch (e) {
    error.value = e.message || '加载产品失败'
    products.value = []
  } finally {
    loading.value = false
  }
}

async function refreshSyncStatus() {
  try {
    const res = await fetch('/api/drive/product-docs/sync-status')
    if (res.ok) {
      const data = await res.json()
      if (data.synced_at) {
        lastSyncTime.value = new Date(data.synced_at).toLocaleString('zh-CN')
      }
    }
  } catch (e) {
    console.error('获取同步状态失败:', e)
  }
}

async function loadAvailableMonths() {
  try {
    const res = await fetch('/api/drive/product-docs')
    if (res.ok) {
      const data = await res.json()
      const monthsSet = new Set()
      data.forEach(doc => {
        const month = extractMonth(doc)
        if (month) monthsSet.add(month)
      })

      availableMonths.value = Array.from(monthsSet).sort((a, b) => {
        const aMatch = a.match(/(\d{4})年(\d+)月/)
        const bMatch = b.match(/(\d{4})年(\d+)月/)
        if (aMatch && bMatch) {
          const aKey = parseInt(aMatch[1]) * 100 + parseInt(aMatch[2])
          const bKey = parseInt(bMatch[1]) * 100 + parseInt(bMatch[2])
          return bKey - aKey
        }
        return a.localeCompare(b)
      })
    }
  } catch (e) {
    console.error('加载月份列表失败:', e)
  }
}

onMounted(async () => {
  await refreshSyncStatus()
  await loadAvailableMonths()
})
</script>

<style scoped>
:deep(.workbench-main) {
  max-width: none;
}

.product-report-page {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.month-search {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--border-soft);
  flex-wrap: wrap;
}

.product-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.product-card {
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  overflow: hidden;
  transition: box-shadow 0.2s, border-color 0.2s;
}

.product-card:hover {
  border-color: var(--brand);
  box-shadow: var(--shadow-md);
}

.card-header {
  background: var(--brand);
  color: #fff;
  padding: 9px 12px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.card-icon { font-size: 15px; }

.card-title {
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-body { padding: 4px 0; }

.info-row {
  display: grid;
  grid-template-columns: 70px 1fr;
  align-items: start;
  padding: 5px 12px;
  gap: 6px;
  border-bottom: 1px solid var(--border-soft);
}

.info-row:last-child { border-bottom: none; }

.info-key {
  font-size: 11px;
  color: var(--ink-soft);
  white-space: nowrap;
  padding-top: 2px;
}

.info-val {
  font-size: 12px;
  color: var(--ink-strong);
  line-height: 1.55;
  font-family: var(--font-mono);
}

.info-val.multiline {
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.raw-content {
  font-size: 12px;
  color: var(--ink-soft);
  white-space: pre-wrap;
  max-height: 150px;
  overflow: auto;
  padding: 12px;
}

@media (max-width: 1400px) {
  .product-cards { grid-template-columns: repeat(3, 1fr); }
}
@media (max-width: 1000px) {
  .product-cards { grid-template-columns: repeat(2, 1fr); }
}
@media (max-width: 640px) {
  .product-cards { grid-template-columns: 1fr; }
}
</style>
