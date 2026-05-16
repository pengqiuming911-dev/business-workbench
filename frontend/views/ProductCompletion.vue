<template>
  <SubPageLayout title="产品完结与发行">
    <div class="section">
      <p class="desc">指定开始/结束日期区间（精确到日），统计完结产品与发行产品的数量与金额。</p>

      <div class="panel">
        <h3 class="panel-title">参数设置</h3>

        <div class="form-row">
          <label>数据来源</label>
          <div class="file-source">
            <span class="file-badge">📊 航班服务交易总表 · 产品表</span>
            <span class="file-from">本地数据库</span>
          </div>
        </div>

        <div class="form-row">
          <label>开始日期</label>
          <input v-model="startDate" type="date" class="input" />
        </div>
        <div class="form-row">
          <label>结束日期</label>
          <input v-model="endDate" type="date" class="input" />
        </div>
        <button class="btn btn-primary" :disabled="loading" @click="run">
          {{ loading ? '查询中...' : '生成报告' }}
        </button>
        <span v-if="errorMsg" class="error">{{ errorMsg }}</span>
      </div>

      <!-- 每月完结与发行产品概览 -->
      <div v-if="tableData.length" class="report-panel">
        <h3 class="section-title">每月完结与发行产品概览</h3>
        <table class="overview-table">
          <thead>
            <tr>
              <th class="col-left">年月</th>
              <th class="col-right">完结数量</th>
              <th class="col-right">完结金额（万元）</th>
              <th class="col-right">发行数量</th>
              <th class="col-right">发行金额（万元）</th>
              <th class="col-right">净增金额（万元）</th>
            </tr>
          </thead>
          <tbody>
            <template v-for="row in tableData" :key="row.month">
              <tr class="data-row" @click="row._showCompleted = !row._showCompleted; row._showIssued = !row._showIssued">
                <td class="col-left month-cell">
                  <span class="chevron" :class="{ open: row._showCompleted }">›</span>
                  {{ row.month }}
                </td>
                <td class="col-right">
                  <span class="count-badge completed">{{ row.completedCount }}</span>
                </td>
                <td class="col-right amt-cell">{{ fmt(row.completedAmount) }}</td>
                <td class="col-right">
                  <span class="count-badge issued">{{ row.issuedCount }}</span>
                </td>
                <td class="col-right amt-cell">{{ fmt(row.issuedAmount) }}</td>
                <td class="col-right" :class="row.netAmount >= 0 ? 'positive' : 'negative'">
                  {{ row.netAmount >= 0 ? '+' : '' }}{{ fmt(row.netAmount) }}
                </td>
              </tr>
              <tr v-if="row._showCompleted || row._showIssued" class="detail-row">
                <td colspan="6" class="detail-cell">
                  <div class="detail-grid">
                    <div v-if="row.completedCount" class="detail-block">
                      <div class="detail-label completed-label">完结产品</div>
                      <div class="name-tags">
                        <span v-for="name in row.completedNames" :key="name" class="name-tag completed-tag">{{ name }}</span>
                      </div>
                    </div>
                    <div v-if="row.issuedCount" class="detail-block">
                      <div class="detail-label issued-label">发行产品</div>
                      <div class="name-tags">
                        <span v-for="name in row.issuedNames" :key="name" class="name-tag issued-tag">{{ name }}</span>
                      </div>
                    </div>
                  </div>
                </td>
              </tr>
            </template>
          </tbody>
          <tfoot>
            <tr class="total-row">
              <td class="col-left">合计</td>
              <td class="col-right">{{ totals.completedCount }}</td>
              <td class="col-right">{{ fmt(totals.completedAmount) }}</td>
              <td class="col-right">{{ totals.issuedCount }}</td>
              <td class="col-right">{{ fmt(totals.issuedAmount) }}</td>
              <td class="col-right" :class="totals.netAmount >= 0 ? 'positive' : 'negative'">
                {{ totals.netAmount >= 0 ? '+' : '' }}{{ fmt(totals.netAmount) }}
              </td>
            </tr>
          </tfoot>
        </table>
      </div>
    </div>
  </SubPageLayout>
</template>

<script setup>
import { ref, computed } from 'vue'
import SubPageLayout from '../components/SubPageLayout.vue'

const startDate = ref('')
const endDate = ref('')
const loading = ref(false)
const errorMsg = ref('')
const tableData = ref([])

async function run() {
  errorMsg.value = ''
  tableData.value = []

  if (!startDate.value || !endDate.value) {
    errorMsg.value = '请选择开始和结束日期'
    return
  }

  loading.value = true
  try {
    const res = await fetch(`/api/db/products?start=${startDate.value}&end=${endDate.value}`)
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '查询失败')
    compute(data.rows)
  } catch (err) {
    errorMsg.value = err.message
  } finally {
    loading.value = false
  }
}

function compute(rows) {
  const monthMap = {}
  const start = startDate.value
  const end = endDate.value

  function ensureMonth(ym) {
    if (!monthMap[ym]) {
      monthMap[ym] = {
        completedIds: new Set(), completedNames: new Set(), completedAmt: 0,
        issuedIds: new Set(), issuedNames: new Set(), issuedAmt: 0,
      }
    }
    return monthMap[ym]
  }

  for (const r of rows) {
    const id = r.id
    const name = r.name || id
    const amt = r.subscribe_amount / 10000

    // 子产品（is_main=0）不计入数量和金额
    if (!r.is_main) continue

    if (r.complete_date && r.complete_date >= start && r.complete_date <= end) {
      const ym = r.complete_date.slice(0, 7)
      const m = ensureMonth(ym)
      m.completedIds.add(id)
      m.completedNames.add(name)
      m.completedAmt += amt
    }
    if (r.issue_date && r.issue_date >= start && r.issue_date <= end) {
      const ym = r.issue_date.slice(0, 7)
      const m = ensureMonth(ym)
      m.issuedIds.add(id)
      m.issuedNames.add(name)
      m.issuedAmt += amt
    }
  }

  tableData.value = Object.entries(monthMap)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([ym, d]) => ({
      month: ym,
      completedCount: d.completedIds.size,
      completedAmount: d.completedAmt,
      completedNames: [...d.completedNames],
      issuedCount: d.issuedIds.size,
      issuedAmount: d.issuedAmt,
      issuedNames: [...d.issuedNames],
      netAmount: d.issuedAmt - d.completedAmt,
      _showCompleted: false,
      _showIssued: false,
    }))

  if (tableData.value.length === 0) {
    errorMsg.value = '所选日期范围内无数据'
  }
}

const totals = computed(() => ({
  completedCount: tableData.value.reduce((s, r) => s + r.completedCount, 0),
  completedAmount: tableData.value.reduce((s, r) => s + r.completedAmount, 0),
  issuedCount: tableData.value.reduce((s, r) => s + r.issuedCount, 0),
  issuedAmount: tableData.value.reduce((s, r) => s + r.issuedAmount, 0),
  netAmount: tableData.value.reduce((s, r) => s + r.netAmount, 0),
}))

function fmt(val) {
  return val.toLocaleString('zh-CN', { minimumFractionDigits: 1, maximumFractionDigits: 1 })
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

.btn { padding: 8px 20px; border-radius: 6px; font-size: 13px; cursor: pointer; border: none; font-weight: 500; }
.btn-primary { background: #C62828; color: #fff; }
.btn-primary:hover:not(:disabled) { background: #B71C1C; }
.btn-primary:disabled { background: #EF9A9A; cursor: not-allowed; }

.error { margin-left: 12px; color: #C62828; font-size: 13px; }

.report-panel {
  background: #fff;
  border-radius: 12px;
  padding: 28px 28px 8px;
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

.overview-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.overview-table th {
  padding: 10px 16px;
  border-bottom: 1px solid #E8DDD0;
  color: #8B7355;
  font-weight: 600;
  background: #FAF7F4;
  font-size: 12px;
  letter-spacing: 0.02em;
}

.data-row {
  cursor: pointer;
  transition: background 0.15s;
}
.data-row:hover { background: #FAF7F4; }

.overview-table td {
  padding: 13px 16px;
  border-bottom: 1px solid #F0EAE0;
  color: #1A1109;
}

.col-left { text-align: left; }
.col-right { text-align: right; }

.month-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
}

.chevron {
  font-size: 16px;
  color: #A8967E;
  transition: transform 0.2s;
  display: inline-block;
  line-height: 1;
}
.chevron.open { transform: rotate(90deg); }

.amt-cell { color: #3D3028; font-variant-numeric: tabular-nums; }

.count-badge {
  display: inline-block;
  min-width: 24px;
  padding: 2px 8px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
  text-align: center;
}
.count-badge.completed { background: #FEF3E2; color: #C25E1A; }
.count-badge.issued { background: #E8F4EC; color: #2E7D45; }

.positive { color: #2E7D45; font-weight: 500; }
.negative { color: #C62828; font-weight: 500; }

/* 展开明细行 */
.detail-row td { padding: 0; border-bottom: 1px solid #F0EAE0; }
.detail-cell { background: #FAFAF8; }

.detail-grid {
  display: flex;
  gap: 0;
  padding: 12px 16px 16px 32px;
}

.detail-block { flex: 1; padding-right: 24px; }
.detail-block:last-child { padding-right: 0; }

.detail-label {
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.04em;
  margin-bottom: 8px;
}
.completed-label { color: #C25E1A; }
.issued-label { color: #2E7D45; }

.name-tags { display: flex; flex-wrap: wrap; gap: 6px; }

.name-tag {
  font-size: 12px;
  padding: 3px 10px;
  border-radius: 4px;
  line-height: 1.5;
}
.completed-tag { background: #FEF3E2; color: #7A3A0A; border: 1px solid #F5D9B0; }
.issued-tag { background: #E8F4EC; color: #1A5C30; border: 1px solid #B8DEC4; }

/* 合计行 */
.total-row td {
  font-weight: 700;
  padding: 14px 16px;
  border-top: 2px solid #E8DDD0;
  border-bottom: none;
  color: #1A1109;
  background: #FAF7F4;
}
</style>
