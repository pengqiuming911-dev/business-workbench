<template>
  <SubPageLayout title="产品运行报告">
    <div class="section">
      <p class="desc">指定开始/结束年月至月份区间，统计每月完结产品数量与金额、按结构类型与标的的完结与交易分布、交易人次与人数及人均金额、不同标的的新客/续做/增购人数与金额。</p>

      <div class="panel">
        <h3 class="panel-title">参数设置</h3>
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
      </div>

      <div class="report-panel">
        <h3 class="panel-title">报告区域</h3>

        <!-- 数据源和同步控制 -->
        <div class="month-search">
          <span class="source-label">数据源：</span>
          <select v-model="selectedMonth" class="month-select" @change="loadProducts">
            <option value="">-- 请选择月份 --</option>
            <option v-for="month in availableMonths" :key="month" :value="month">
              {{ month }}
            </option>
          </select>
          <button class="btn btn-secondary" @click="syncFromFeishu" :disabled="syncing">
            {{ syncing ? '同步中...' : '从飞书同步' }}
          </button>
          <span v-if="lastSyncTime" class="sync-info">最后同步: {{ lastSyncTime }}</span>
        </div>

        <div v-if="error" class="error-msg">{{ error }}</div>

        <!-- 产品卡片列表 -->
        <div v-if="loading" class="loading-msg">正在加载产品结构...</div>
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
        <div v-else-if="selectedMonth && !loading" class="placeholder">
          该月份暂无产品结构数据
        </div>
        <div v-else-if="!selectedMonth" class="placeholder">
          请选择月份查看产品结构，或点击「从飞书同步」更新数据
        </div>
      </div>
    </div>
  </SubPageLayout>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import SubPageLayout from '../components/SubPageLayout.vue'

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

// 去掉"销售物料："前缀，显示更简洁的产品名
function getDisplayName(docName) {
  return docName.replace(/^销售物料[：:\s]*/, '')
}

// 提取月份名称
function extractMonth(doc) {
  const path = doc.parent_path || ''
  // 匹配各种格式: "2026年4月", "2026 年 4 月", "2026年 4月" 等
  const match = path.match(/(\d{4})[年\s]*(\d{1,2})[月\s]*/)
  if (match) {
    const month = parseInt(match[2])
    return `${match[1]}年${month}月`
  }
  // 尝试从文档名提取
  const nameMatch = doc.doc_name.match(/(\d{4})[年\s]*(\d{1,2})[月\s]*/)
  if (nameMatch) {
    const month = parseInt(nameMatch[2])
    return `${nameMatch[1]}年${month}月`
  }
  return '其他'
}

// 从SQLite加载产品文档
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
    // 只保留名称包含"物料"的文档
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

// 从飞书同步产品文档
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

    // 显示同步成功信息
    alert(`同步成功！共 ${result.doc_count} 个文档，${result.folder_count} 个文件夹`)

    // 刷新同步状态
    await refreshSyncStatus()

    // 重新加载产品列表
    await loadAvailableMonths()

    // 显示有多少月份可用
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

// 获取同步状态
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

// 加载所有可用的月份
async function loadAvailableMonths() {
  try {
    const res = await fetch('/api/drive/product-docs')
    if (res.ok) {
      const data = await res.json()
      console.log('数据库中的所有文档数量:', data.length)
      console.log('文档名示例:', data.slice(0, 5).map(d => ({ name: d.doc_name, path: d.parent_path })))

      // 提取所有月份
      const monthsSet = new Set()
      data.forEach(doc => {
        const month = extractMonth(doc)
        console.log('提取月份:', doc.doc_name, '->', month, '| path:', doc.parent_path)
        if (month) monthsSet.add(month)
      })

      // 排序月份（最新的在前）
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
/* 覆盖 SubPageLayout 的 max-width，让内容撑满屏宽 */
:deep(.content) { max-width: none; padding: 24px 28px; }

.desc { color: #6B5C4E; font-size: 14px; line-height: 1.8; margin-bottom: 24px; }
.panel { background: #fff; border-radius: 12px; padding: 24px; margin-bottom: 20px; border: 1px solid #E8DDD0; }
.panel-title { font-size: 16px; font-weight: 600; margin-bottom: 16px; }
.form-row { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
.form-row label { font-size: 13px; white-space: nowrap; width: 90px; }
.input { flex: 1; border: 1px solid #E8DDD0; border-radius: 6px; padding: 8px 12px; font-size: 13px; outline: none; background: #fff; }
.input:focus { border-color: #D97757; }
.btn { padding: 8px 20px; border-radius: 6px; font-size: 13px; cursor: pointer; border: none; transition: all 0.15s; }
.btn-primary { background: #D97757; color: #fff; }
.btn-primary:hover { background: #C5684A; }
.btn-secondary { background: #fff; color: #6B5C4E; border: 1px solid #D6C9BB; padding: 6px 14px; font-size: 12px; }
.btn-secondary:hover:not(:disabled) { background: #EFE9DF; border-color: #D97757; color: #D97757; }
.btn-secondary:disabled { opacity: 0.6; cursor: not-allowed; }
.report-panel { background: #fff; border-radius: 12px; padding: 24px; border: 1px solid #E8DDD0; }
.placeholder { color: #aaa; font-size: 14px; padding: 40px 0; text-align: center; }
.loading-msg { color: #8C7B6E; font-size: 14px; padding: 40px 0; text-align: center; }
.error-msg { color: #D97757; font-size: 13px; padding: 12px 16px; background: #FDF0E8; border-radius: 6px; margin-bottom: 16px; }

/* 月份搜索 */
.month-search { display: flex; align-items: center; gap: 12px; margin-bottom: 20px; padding-bottom: 16px; border-bottom: 1px solid #E8DDD0; flex-wrap: wrap; }
.source-label { font-size: 13px; font-weight: 500; color: #6B5C4E; }
.month-select { flex: 1; min-width: 200px; border: 1px solid #E8DDD0; border-radius: 6px; padding: 8px 12px; font-size: 13px; background: #fff; color: #1A1109; outline: none; cursor: pointer; }
.month-select:focus { border-color: #D97757; }
.sync-info { font-size: 12px; color: #8C7B6E; white-space: nowrap; }

/* 产品卡片列表 - 4列自适应，固定宽度 */
.product-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.product-card {
  background: #fff;
  border: 1px solid #E8DDD0;
  border-radius: 12px;
  overflow: hidden;
  transition: box-shadow 0.2s, border-color 0.2s;
}

.product-card:hover {
  border-color: #D97757;
  box-shadow: 0 4px 16px rgba(217, 119, 87, 0.12);
}

.card-header {
  background: linear-gradient(135deg, #D97757 0%, #C5684A 100%);
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
  border-bottom: 1px solid #F5F0E8;
}

.info-row:last-child { border-bottom: none; }

.info-key {
  font-size: 11px;
  color: #8C7B6E;
  white-space: nowrap;
  padding-top: 2px;
}

.info-val {
  font-size: 12px;
  color: #1A1109;
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
  color: #8C7B6E;
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
