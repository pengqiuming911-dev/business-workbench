<template>
  <div class="product-report-page">
    <h1 class="text-page-title">产品运行报告</h1>
    <p class="text-body" style="margin-bottom: 24px;">指定开始/结束年月，统计每月完结产品数量与金额、按结构类型与标的的完结与交易分布、交易人次与人数及人均金额、不同标的的新客/续做/增购人数与金额。</p>

    <PanelCard title="参数设置">
      <div class="form-row">
        <label>开始年月</label>
        <input v-model="startMonth" type="month" class="input" />
      </div>
      <div class="form-row">
        <label>结束年月</label>
        <input v-model="endMonth" type="month" class="input" />
      </div>
      <div class="form-row">
        <label>数据文件</label>
        <input v-model="filePath" type="text" placeholder="请输入交易表_修正后.xlsx 路径" class="input" />
      </div>
      <button class="btn btn-primary" @click="run">生成产品运行报告</button>
    </PanelCard>

    <PanelCard title="报告区域">
      <div class="month-search">
        <span class="text-label">数据源：</span>
        <select v-model="selectedMonth" class="input" style="min-width: 200px; flex: 1;" @change="loadProducts">
          <option value="">-- 请选择月份 --</option>
          <option v-for="month in availableMonths" :key="month" :value="month">
            {{ month }}
          </option>
        </select>
        <button class="btn btn-secondary" @click="syncFromFeishu" :disabled="syncing">
          {{ syncing ? '同步中...' : '从飞书同步' }}
        </button>
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
        请选择月份查看产品结构，或点击「从飞书同步」更新数据
      </div>
    </PanelCard>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import PanelCard from '../components/PanelCard.vue'

const startMonth = ref('')
const endMonth = ref('')
const filePath = ref('')
const selectedMonth = ref('')
const products = ref([])
const loading = ref(false)
const syncing = ref(false)
const error = ref('')
const lastSyncTime = ref('')
const availableMonths = ref([])

const MONTH_PATTERN = /(\d{4})[年\s]*(\d{1,2})[月]?/

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
    console.log('查询月份:', selectedMonth.value, '返回文档数:', data.length)
    const filtered = data.filter(d => d.doc_name.includes('物料'))
    console.log('过滤后物料文档:', filtered.map(d => ({ name: d.doc_name, structured_json: d.structure_json ? '有内容' : '空', raw: d.raw_content?.slice(0, 50) })))
    products.value = filtered
  } catch (e) {
    error.value = e.message || '加载产品失败'
    products.value = []
  } finally {
    loading.value = false
  }
}

async function syncFromFeishu() {
  syncing.value = true
  error.value = ''

  try {
    const res = await fetch('/api/drive/sync-product-docs', { method: 'POST' })
    const result = await res.json()
    console.log('同步结果:', result)

    if (!res.ok) {
      throw new Error(result.error || '同步失败')
    }

    alert(`同步成功！共 ${result.doc_count} 个文档，${result.folder_count} 个文件夹`)

    await refreshSyncStatus()

    await loadAvailableMonths()

    if (availableMonths.value.length > 0) {
      console.log('可用月份:', availableMonths.value)
    } else {
      console.log('没有找到任何月份数据，请检查同步日志')
    }

    if (selectedMonth.value) {
      await loadProducts()
    }
  } catch (e) {
    error.value = e.message || '同步失败'
  } finally {
    syncing.value = false
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
      console.log('数据库中的所有文档数量:', data.length)
      console.log('文档名示例:', data.slice(0, 5).map(d => ({ name: d.doc_name, path: d.parent_path })))

      const monthsSet = new Set()
      data.forEach(doc => {
        const month = extractMonth(doc)
        console.log('提取月份:', doc.doc_name, '->', month, '| path:', doc.parent_path)
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

      console.log('最终月份列表:', JSON.stringify(availableMonths.value))
    }
  } catch (e) {
    console.error('加载月份列表失败:', e)
  }
}

function run() { alert('产品运行报告功能尚未接入后端。') }

onMounted(async () => {
  await refreshSyncStatus()
  await loadAvailableMonths()
})
</script>

<style scoped>
:deep(.workbench-main) {
  max-width: none;
}

.month-search {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
  padding-bottom: 16px;
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
  box-shadow: var(--shadow-soft);
}

.card-header {
  background: var(--brand);
  color: #fff;
  padding: 11px 14px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.card-icon { font-size: 17px; }

.card-title {
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-body { padding: 5px 0; }

.info-row {
  display: grid;
  grid-template-columns: 76px 1fr;
  align-items: start;
  padding: 6px 14px;
  gap: 8px;
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
  padding: 14px;
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