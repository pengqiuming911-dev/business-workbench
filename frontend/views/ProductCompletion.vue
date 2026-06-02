<template>
  <SubPageLayout title="产品派息/敲出观察">
    <div class="section">
      <p class="desc">展示存续产品（持有中）的派息与敲出观察情况。数据来源为航班服务交易总表 · 产品表。</p>

      <div class="panel">
        <h3 class="panel-title">操作</h3>
        <div class="form-row">
          <label>数据来源</label>
          <div class="file-source">
            <span class="file-badge">📊 航班服务交易总表 · 产品表</span>
            <span class="file-from">本地数据库</span>
          </div>
        </div>
        <div class="form-row">
          <label>搜索</label>
          <input v-model="searchText" type="text" class="input" placeholder="按产品名称或航班编号搜索..." />
        </div>
        <button class="btn btn-primary" :disabled="refreshing" @click="refreshPrices">
          {{ refreshing ? '刷新中...' : '刷新标的价格' }}
        </button>
        <button class="btn btn-secondary" :disabled="generating" @click="generateObservations">
          {{ generating ? '生成中...' : '生成观察记录' }}
        </button>
        <span v-if="lastUpdated" class="update-time">最后更新: {{ lastUpdated }}</span>
        <span v-if="errorMsg" class="error">{{ errorMsg }}</span>
        <span v-if="successMsg" class="success">{{ successMsg }}</span>
      </div>

      <div v-if="filteredProducts.length" class="report-panel">
        <h3 class="section-title">存续产品观察概览</h3>
        <div class="table-wrap">
          <table class="overview-table">
            <thead>
              <tr>
                <th class="col-left sticky-col">航班编号</th>
                <th class="col-left">产品名称</th>
                <th class="col-left">私募管理人</th>
                <th class="col-left">持有状态</th>
                <th class="col-left">代码</th>
                <th class="col-right">入场价</th>
                <th class="col-left">入场日</th>
                <th class="col-right">存续月</th>
                <th class="col-right">锁定期(月)</th>
                <th class="col-left">最近观察日</th>
                <th class="col-right">标的价格</th>
                <th class="col-right">敲出价</th>
                <th class="col-right">派息线</th>
                <th class="col-center">是否敲出</th>
                <th class="col-center">是否派息</th>
              </tr>
            </thead>
            <tbody>
              <template v-for="p in filteredProducts" :key="p.id">
                <tr class="data-row" @click="toggleExpand(p.id)">
                  <td class="col-left sticky-col">
                    <span class="chevron" :class="{ open: expandedId === p.id }">›</span>
                    {{ p.id }}
                  </td>
                  <td class="col-left">{{ p.name }}</td>
                  <td class="col-left">{{ p.manager }}</td>
                  <td class="col-left">
                    <span class="status-badge">{{ p.holding_status }}</span>
                  </td>
                  <td class="col-left code-cell">{{ p.code }}</td>
                  <td class="col-right">{{ formatPrice(p.entry_price) }}</td>
                  <td class="col-left">{{ p.issue_date || '--' }}</td>
                  <td class="col-right">{{ computeMonthsSince(p) }}</td>
                  <td class="col-right">{{ p.lock_months || '--' }}</td>
                  <td class="col-left">{{ latestObs(p)?.date || '--' }}</td>
                  <td class="col-right">{{ formatPrice(latestObs(p)?.underlying_price) }}</td>
                  <td class="col-right">{{ formatPrice(latestObs(p)?.knockout_price) }}</td>
                  <td class="col-right">{{ formatPrice(latestObs(p)?.dividend_line) }}</td>
                  <td class="col-center" :class="knockoutClass(latestObs(p)?.is_knocked_out)">
                    {{ latestObs(p)?.is_knocked_out || '--' }}
                  </td>
                  <td class="col-center" :class="dividendClass(latestObs(p)?.is_dividend)">
                    {{ latestObs(p)?.is_dividend || '--' }}
                  </td>
                </tr>
                <tr v-if="expandedId === p.id && p.observations.length" class="detail-row">
                  <td colspan="15" class="detail-cell">
                    <div class="detail-label">历史观察日明细</div>
                    <table class="detail-table">
                      <thead>
                        <tr>
                          <th>观察日</th>
                          <th>标的价格</th>
                          <th>敲出价</th>
                          <th>派息线</th>
                          <th>是否敲出</th>
                          <th>是否派息</th>
                        </tr>
                      </thead>
                      <tbody>
                        <tr v-for="obs in p.observations" :key="obs.date">
                          <td>{{ obs.date }}</td>
                          <td>{{ formatPrice(obs.underlying_price) }}</td>
                          <td>{{ formatPrice(obs.knockout_price) }}</td>
                          <td>{{ formatPrice(obs.dividend_line) }}</td>
                          <td :class="knockoutClass(obs.is_knocked_out)">{{ obs.is_knocked_out }}</td>
                          <td :class="dividendClass(obs.is_dividend)">{{ obs.is_dividend }}</td>
                        </tr>
                      </tbody>
                    </table>
                  </td>
                </tr>
                <tr v-if="expandedId === p.id && !p.observations.length" class="detail-row">
                  <td colspan="15" class="detail-cell">
                    <div class="detail-empty">暂无观察日记录</div>
                  </td>
                </tr>
              </template>
            </tbody>
          </table>
        </div>
        <p class="table-summary">共 {{ filteredProducts.length }} 个存续产品</p>
      </div>

      <div v-else-if="loaded && !filteredProducts.length" class="empty-state">
        <p>暂无存续产品数据，请先在「数据准备」页面同步飞书数据。</p>
      </div>
    </div>
  </SubPageLayout>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import SubPageLayout from '../components/SubPageLayout.vue'

const searchText = ref('')
const products = ref([])
const lastUpdated = ref(null)
const loaded = ref(false)
const refreshing = ref(false)
const generating = ref(false)
const errorMsg = ref('')
const successMsg = ref('')
const expandedId = ref(null)

onMounted(() => loadData())

async function loadData() {
  errorMsg.value = ''
  try {
    const res = await fetch('/api/observations')
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '加载失败')
    products.value = data.products || []
    lastUpdated.value = data.lastUpdated
  } catch (err) {
    errorMsg.value = err.message
  } finally {
    loaded.value = true
  }
}

async function refreshPrices() {
  refreshing.value = true
  errorMsg.value = ''
  successMsg.value = ''
  try {
    const res = await fetch('/api/observations/refresh-prices', { method: 'POST' })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '刷新失败')
    successMsg.value = `价格刷新完成：${data.refreshed} 个成功${data.failed ? '，' + data.failed + ' 个失败' : ''}`
    await loadData()
  } catch (err) {
    errorMsg.value = err.message
  } finally {
    refreshing.value = false
  }
}

async function generateObservations() {
  generating.value = true
  errorMsg.value = ''
  successMsg.value = ''
  try {
    const res = await fetch('/api/observations/generate', { method: 'POST' })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '生成失败')
    successMsg.value = `生成完成：新增 ${data.generated} 条，更新 ${data.updated} 条`
    await loadData()
  } catch (err) {
    errorMsg.value = err.message
  } finally {
    generating.value = false
  }
}

function toggleExpand(id) {
  expandedId.value = expandedId.value === id ? null : id
}

const filteredProducts = computed(() => {
  if (!searchText.value) return products.value
  const q = searchText.value.toLowerCase()
  return products.value.filter(p =>
    (p.name && p.name.toLowerCase().includes(q)) || p.id.toLowerCase().includes(q)
  )
})

function latestObs(product) {
  if (!product.observations || !product.observations.length) return null
  return product.observations[product.observations.length - 1]
}

function computeMonthsSince(product) {
  if (!product.issue_date) return '--'
  const entry = new Date(product.issue_date)
  const now = new Date()
  return (now.getFullYear() - entry.getFullYear()) * 12 + (now.getMonth() - entry.getMonth())
}

function formatPrice(val) {
  if (val === null || val === undefined) return '--'
  return Number(val).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

function knockoutClass(status) {
  if (status === '是') return 'result-yes-knockout'
  if (status === '否') return 'result-no'
  return ''
}

function dividendClass(status) {
  if (status === '是') return 'result-yes-dividend'
  if (status === '否') return 'result-no'
  return ''
}
</script>

<style scoped>
.desc { color: #6B5C4E; font-size: 14px; line-height: 1.8; margin-bottom: 24px; }

.panel {
  background: #fff;
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 20px;
  border: 1px solid #E8DDD0;
}

.panel-title { font-size: 15px; font-weight: 600; color: #1A1109; margin-bottom: 16px; }

.form-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.form-row > label:first-child {
  font-size: 13px;
  color: #6B5C4E;
  white-space: nowrap;
  width: 70px;
  flex-shrink: 0;
}

.input {
  flex: 1;
  border: 1px solid #E8DDD0;
  border-radius: 6px;
  padding: 8px 12px;
  font-size: 13px;
  outline: none;
  background: #fff;
  color: #1A1109;
}

.input:focus { border-color: #8B7355; }

.file-source { flex: 1; display: flex; align-items: center; gap: 10px; }
.file-badge { font-size: 13px; color: #1A1109; font-weight: 500; }
.file-from { font-size: 12px; color: #A8967E; background: #F5F0E8; padding: 2px 8px; border-radius: 10px; }

.btn { padding: 8px 20px; border-radius: 6px; font-size: 13px; cursor: pointer; border: none; font-weight: 500; margin-right: 8px; }
.btn-primary { background: #C62828; color: #fff; }
.btn-primary:hover:not(:disabled) { background: #B71C1C; }
.btn-primary:disabled { background: #EF9A9A; cursor: not-allowed; }
.btn-secondary { background: #8B7355; color: #fff; }
.btn-secondary:hover:not(:disabled) { background: #7A6348; }
.btn-secondary:disabled { background: #C4B5A5; cursor: not-allowed; }

.update-time { margin-left: 16px; color: #8B7355; font-size: 12px; }
.error { margin-left: 12px; color: #C62828; font-size: 13px; }
.success { margin-left: 12px; color: #2E7D45; font-size: 13px; }

.report-panel {
  background: #fff;
  border-radius: 12px;
  padding: 28px;
  border: 1px solid #E8DDD0;
}

.section-title {
  font-size: 15px;
  font-weight: 700;
  color: #1A1109;
  margin-bottom: 20px;
  padding-bottom: 12px;
  border-bottom: 2px solid #F0EAE0;
}

.table-wrap {
  overflow-x: auto;
  margin-bottom: 12px;
}

.overview-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
  min-width: 1400px;
}

.overview-table th {
  padding: 10px 12px;
  border-bottom: 1px solid #E8DDD0;
  color: #8B7355;
  font-weight: 600;
  background: #FAF7F4;
  font-size: 11px;
  letter-spacing: 0.02em;
  white-space: nowrap;
  position: sticky;
  top: 0;
  z-index: 1;
}

.data-row {
  cursor: pointer;
  transition: background 0.15s;
}
.data-row:hover { background: #FAF7F4; }

.overview-table td {
  padding: 11px 12px;
  border-bottom: 1px solid #F0EAE0;
  color: #1A1109;
  white-space: nowrap;
}

.col-left { text-align: left; }
.col-right { text-align: right; }
.col-center { text-align: center; }

.sticky-col {
  position: sticky;
  left: 0;
  background: #fff;
  z-index: 2;
}
.data-row:hover .sticky-col { background: #FAF7F4; }
.overview-table th.sticky-col { z-index: 3; background: #FAF7F4; }

.chevron {
  font-size: 14px;
  color: #A8967E;
  transition: transform 0.2s;
  display: inline-block;
  line-height: 1;
  margin-right: 4px;
}
.chevron.open { transform: rotate(90deg); }

.code-cell { font-family: monospace; font-size: 11px; color: #6B5C4E; }

.status-badge {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 10px;
  background: #E8F4EC;
  color: #2E7D45;
}

.result-yes-knockout {
  color: #C62828;
  font-weight: 600;
  background: #FEF3E2;
  border-radius: 4px;
  padding: 2px 6px;
}

.result-yes-dividend {
  color: #2E7D45;
  font-weight: 600;
  background: #E8F4EC;
  border-radius: 4px;
  padding: 2px 6px;
}

.result-no {
  color: #8B7355;
}

.detail-row td {
  padding: 0;
  border-bottom: 1px solid #F0EAE0;
}

.detail-cell {
  background: #FAFAF8;
}

.detail-label {
  font-size: 11px;
  font-weight: 600;
  color: #8B7355;
  letter-spacing: 0.04em;
  padding: 12px 16px 8px;
}

.detail-empty {
  font-size: 12px;
  color: #A8967E;
  padding: 12px 16px;
}

.detail-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 11px;
  margin: 0 16px 12px;
}

.detail-table th {
  padding: 6px 12px;
  border-bottom: 1px solid #E8DDD0;
  color: #8B7355;
  font-weight: 600;
  background: transparent;
  text-align: left;
}

.detail-table td {
  padding: 6px 12px;
  border-bottom: 1px solid #F0EAE0;
  color: #3D3028;
}

.table-summary {
  font-size: 12px;
  color: #8B7355;
  text-align: right;
  padding-top: 8px;
}

.empty-state {
  text-align: center;
  padding: 48px 24px;
  color: #A8967E;
  font-size: 14px;
  background: #fff;
  border-radius: 12px;
  border: 1px solid #E8DDD0;
}
</style>
