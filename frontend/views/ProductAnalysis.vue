<template>
  <div>
    <div class="filter-bar">
      <div class="filter-group">
        <label>申购日期</label>
        <input v-model="filters.issueDateStart" type="date" class="input input-sm" />
        <span class="filter-sep">至</span>
        <input v-model="filters.issueDateEnd" type="date" class="input input-sm" />
      </div>
      <div class="filter-group">
        <label>完结日期</label>
        <input v-model="filters.completeDateStart" type="date" class="input input-sm" />
        <span class="filter-sep">至</span>
        <input v-model="filters.completeDateEnd" type="date" class="input input-sm" />
      </div>
      <div class="filter-group">
        <label>持有状态</label>
        <select v-model="filters.holdingStatus" class="input input-sm">
          <option value="">全部</option>
          <option v-for="opt in filterOptions.holdingStatuses" :key="opt" :value="opt">
            {{ normalizeHoldingStatus(opt) }}
          </option>
        </select>
      </div>
      <div class="filter-group">
        <label>管理人</label>
        <input
          v-model="filters.manager"
          list="manager-options"
          type="text"
          class="input input-sm input-narrow"
          placeholder="模糊匹配"
        />
        <datalist id="manager-options">
          <option v-for="opt in filterOptions.managers" :key="opt" :value="opt">{{ opt }}</option>
        </datalist>
      </div>
      <div class="filter-actions">
        <button class="btn btn-primary btn-sm" @click="fetchData">查询</button>
        <button class="btn btn-secondary btn-sm" @click="resetFilters">重置</button>
      </div>
    </div>

    <div class="advanced-toggle" @click="showAdvanced = !showAdvanced">
      <span class="chevron" :class="{ open: showAdvanced }">▸</span>
      高级筛选
    </div>

    <div v-show="showAdvanced" class="filter-bar advanced-bar">
      <div class="filter-group">
        <label>挂钩标的</label>
        <select v-model="filters.code" class="input input-sm">
          <option value="">全部</option>
          <option v-for="opt in filterOptions.codes" :key="opt" :value="opt">
            {{ stripParentheses(opt) }}
          </option>
        </select>
      </div>
      <div class="filter-group">
        <label>结构类型</label>
        <select v-model="filters.structureType" class="input input-sm">
          <option value="">全部</option>
          <option v-for="opt in filterOptions.structureTypes" :key="opt" :value="opt">{{ opt }}</option>
        </select>
      </div>
      <div class="filter-group">
        <label>锁定期</label>
        <select v-model="filters.lockMonths" class="input input-sm">
          <option value="">全部</option>
          <option v-for="opt in filterOptions.lockMonths" :key="opt" :value="opt">{{ opt }}</option>
        </select>
      </div>
      <div class="filter-group">
        <label>保证金比例</label>
        <select v-model="filters.marginRatio" class="input input-sm">
          <option value="">全部</option>
          <option v-for="opt in filterOptions.marginRatios" :key="opt" :value="opt">{{ opt }}</option>
        </select>
      </div>
    </div>

    <div class="data-source-badge">
      <span class="text-label">数据源：航班服务交易总表 / 产品表</span>
      <span class="badge badge-blue">本地数据库</span>
    </div>

    <div v-if="loading" class="loading-state">正在加载数据...</div>
    <div v-else-if="items.length > 0">
      <div class="table-wrap">
        <table class="data-table product-table">
          <colgroup>
            <col style="width: 120px" /><!-- 航班编号(sticky) -->
            <col style="width: 110px" /><!-- 管理人(sticky) -->
            <col style="width: 170px" /><!-- 产品名称(sticky) -->
          </colgroup>
          <thead>
            <tr>
              <th class="sticky-col sticky-col-1">航班编号</th>
              <th class="sticky-col sticky-col-2">管理人</th>
              <th class="sticky-col sticky-col-3 name-cell" :title="''">产品名称</th>
              <th>持有状态</th>
              <th>申购日期</th>
              <th class="num">存续时间（月）</th>
              <th>结构类型</th>
              <th>挂钩标的</th>
              <th class="num">锁定期</th>
              <th class="num">保证金比例</th>
              <th class="num">敲入</th>
              <th class="num">首月敲出</th>
              <th class="num">入场价</th>
              <th>是否敲入</th>
              <th class="num">每月递减</th>
              <th>托管券商</th>
              <th>交易对手</th>
              <th>期限</th>
              <th>完结日期</th>
              <th class="num">降落伞</th>
              <th class="num">派息障碍</th>
              <th class="num">月票息（税费后）</th>
              <th class="num">第一段票息（税费后）</th>
              <th class="num">第二段票息（税费后）</th>
              <th class="num">第三段票息（税费后）</th>
              <th class="num">绝对收益率</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in items" :key="item.id">
              <td class="sticky-col sticky-col-1">{{ item.id }}</td>
              <td class="sticky-col sticky-col-2">{{ item.manager || '--' }}</td>
              <td class="sticky-col sticky-col-3 name-cell" :title="item.name">{{ truncateName(item.name) }}</td>
              <td class="status-cell">
                <span class="badge" :class="isActiveStatus(item.holding_status) ? 'badge-green' : 'badge-amber'">
                  {{ normalizeHoldingStatus(item.holding_status) }}
                </span>
              </td>
              <td>{{ item.issue_date || '--' }}</td>
              <td class="num">{{ formatDuration(item) }}</td>
              <td>{{ item.structure_type || '--' }}</td>
              <td>{{ stripParentheses(item.code) }}</td>
              <td class="num">{{ item.lock_months ?? '--' }}</td>
              <td class="num">{{ item.margin_ratio ?? '--' }}</td>
              <td class="num">{{ item.knock_in ?? '--' }}</td>
              <td class="num">{{ formatRatio2(item.first_knockout_ratio) }}</td>
              <td class="num">{{ item.entry_price ?? '--' }}</td>
              <td>{{ item.knocked_in ?? '--' }}</td>
              <td class="num">{{ item.monthly_decrease ?? '--' }}</td>
              <td>{{ item.custodian || '--' }}</td>
              <td>{{ item.counterparty || '--' }}</td>
              <td>{{ item.term || '--' }}</td>
              <td>{{ item.complete_date || '--' }}</td>
              <td class="num">{{ item.parachute ?? '--' }}</td>
              <td class="num">{{ item.dividend_barrier ?? '--' }}</td>
              <td class="num">{{ item.monthly_coupon ?? '--' }}</td>
              <td class="num">{{ item.coupon_1st ?? '--' }}</td>
              <td class="num">{{ item.coupon_2nd ?? '--' }}</td>
              <td class="num">{{ item.coupon_3rd ?? '--' }}</td>
              <td class="num">{{ item.absolute_return ?? '--' }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="pagination">
        <span class="text-label">共 {{ total }} 条</span>
        <div class="pagination-controls">
          <button class="btn btn-secondary btn-sm" :disabled="currentPage <= 1" @click="goPage(currentPage - 1)">上一页</button>
          <span class="page-info">{{ currentPage }} / {{ totalPages }}</span>
          <button class="btn btn-secondary btn-sm" :disabled="currentPage >= totalPages" @click="goPage(currentPage + 1)">下一页</button>
        </div>
      </div>
    </div>
    <div v-else-if="loaded" class="empty-state">暂无产品数据</div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'

const loading = ref(false)
const loaded = ref(false)
const items = ref([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = 50
const showAdvanced = ref(false)

const filters = ref({
  issueDateStart: '',
  issueDateEnd: '',
  holdingStatus: '',
  manager: '',
  completeDateStart: '',
  completeDateEnd: '',
  code: '',
  structureType: '',
  lockMonths: '',
  marginRatio: '',
})

const filterOptions = ref({
  holdingStatuses: [],
  managers: [],
  codes: [],
  structureTypes: [],
  lockMonths: [],
  marginRatios: [],
})

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

function truncateName(val) {
  if (!val) return '--'
  return val.length > 10 ? `${val.slice(0, 10)}...` : val
}

function isCompletedStatus(val) {
  if (typeof val !== 'string') return false
  return ['完结', '已完结', '瀹岀粨', '宸插畬缁'].some(keyword => val.includes(keyword))
}

function isActiveStatus(val) {
  return !isCompletedStatus(val)
}

function normalizeHoldingStatus(val) {
  if (!val) return '--'
  if (isCompletedStatus(val)) return '已完结'
  return val
}

function stripParentheses(val) {
  if (!val) return '--'
  return val.replace(/[（(].*?[)）]/g, '').trim() || val
}

function formatRatio2(val) {
  if (val === null || val === undefined || val === '') return '--'
  const n = Number(val)
  if (isNaN(n)) return '--'
  return n.toFixed(2)
}

function formatDuration(item) {
  if (isCompletedStatus(item.holding_status)) {
    return item.duration_months ?? '--'
  }
  if (item.duration_months != null) return item.duration_months
  if (item.duration_days != null) return (item.duration_days / 30).toFixed(1)
  return '--'
}

async function loadFilterOptions() {
  try {
    const res = await fetch('/api/holding/filter-options')
    if (!res.ok) return
    const data = await res.json()
    filterOptions.value = {
      holdingStatuses: data.holding_statuses || [],
      managers: data.managers || [],
      codes: data.codes || [],
      structureTypes: data.structure_types || [],
      lockMonths: data.lock_months || [],
      marginRatios: data.margin_ratios || [],
    }
  } catch {}
}

async function fetchData() {
  loading.value = true
  try {
    const params = new URLSearchParams()
    const f = filters.value
    if (f.issueDateStart) params.set('issue_date_start', f.issueDateStart)
    if (f.issueDateEnd) params.set('issue_date_end', f.issueDateEnd)
    if (f.holdingStatus) params.set('holding_status', f.holdingStatus)
    if (f.manager) params.set('manager', f.manager)
    if (f.completeDateStart) params.set('complete_date_start', f.completeDateStart)
    if (f.completeDateEnd) params.set('complete_date_end', f.completeDateEnd)
    if (f.code) params.set('code', f.code)
    if (f.structureType) params.set('structure_type', f.structureType)
    if (f.lockMonths) params.set('lock_months', f.lockMonths)
    if (f.marginRatio) params.set('margin_ratio', f.marginRatio)
    params.set('page', currentPage.value)
    params.set('page_size', pageSize)

    const res = await fetch(`/api/holding/products?${params}`)
    if (!res.ok) throw new Error('加载失败')
    const data = await res.json()
    items.value = data.items || []
    total.value = data.total || items.value.length
  } catch {
    items.value = []
    total.value = 0
  } finally {
    loading.value = false
    loaded.value = true
  }
}

function resetFilters() {
  filters.value = {
    issueDateStart: '',
    issueDateEnd: '',
    holdingStatus: '',
    manager: '',
    completeDateStart: '',
    completeDateEnd: '',
    code: '',
    structureType: '',
    lockMonths: '',
    marginRatio: '',
  }
  currentPage.value = 1
  fetchData()
}

function goPage(page) {
  currentPage.value = page
  fetchData()
}

onMounted(async () => {
  await loadFilterOptions()
  await fetchData()
})
</script>

<style scoped>
.filter-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: flex-end;
  margin-bottom: 18px;
  padding: 16px 20px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  box-shadow: var(--shadow-sm);
}

.filter-group {
  display: flex;
  align-items: center;
  gap: 6px;
}

.filter-group label {
  color: var(--ink-soft);
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
  letter-spacing: 0.01em;
}

.input-sm {
  height: 32px;
  min-height: 32px;
  padding: 0 10px;
  font-size: 13px;
  width: auto;
  min-width: 120px;
}

.filter-sep {
  color: var(--ink-faint);
  font-size: 13px;
}

.filter-actions {
  display: flex;
  gap: 8px;
  margin-left: auto;
}

.advanced-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--brand);
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  margin-bottom: 18px;
  user-select: none;
  transition: color 180ms ease;
}

.advanced-toggle:hover {
  color: var(--brand-hover);
}

.advanced-bar {
  margin-top: -8px;
}

.chevron {
  font-size: 16px;
  transition: transform 0.2s;
  display: inline-block;
}

.chevron.open {
  transform: rotate(90deg);
}

.data-source-badge {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 18px;
}

.product-table {
  min-width: 3200px;
  font-size: 13px;
  border-collapse: separate;
  border-spacing: 0;
}

.product-table thead {
  position: sticky;
  top: 0;
  z-index: 4;
}

.product-table th {
  padding: 10px 12px;
  font-size: 11px;
  font-weight: 800;
  text-transform: none;
  letter-spacing: 0.04em;
  white-space: nowrap;
  color: var(--ink-soft);
  background: var(--bg-card);
  border-bottom: 2px solid var(--border);
  text-align: left;
}

.product-table th.num {
  text-align: right;
}

.product-table td {
  padding: 7px 12px;
  white-space: nowrap;
  border-bottom: 1px solid #f0f2f5;
  color: var(--ink-strong);
  font-size: 13px;
}

.product-table td.num {
  text-align: right;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 12.5px;
  letter-spacing: -0.01em;
}

.name-cell {
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
  cursor: default;
  font-weight: 600;
}

.status-cell {
  padding-left: 8px !important;
  padding-right: 8px !important;
  white-space: nowrap;
  width: 1%;
}

/* 斑马纹 */
.product-table tbody tr:nth-child(even) td {
  background: #fafbfd;
}

.product-table tr:hover td {
  background: #eef2f7;
}

/* sticky 列 */
.sticky-col {
  position: sticky;
  z-index: 2;
  background: var(--bg-card);
}
.sticky-col-1 { left: 0; }
.sticky-col-2 { left: 120px; box-shadow: -4px 0 0 0 var(--bg-card); }
.sticky-col-3 { left: 230px; box-shadow: -4px 0 0 0 var(--bg-card); }

.product-table tbody tr:nth-child(even) .sticky-col {
  background: #fafbfd;
}
.product-table tbody tr:nth-child(even) .sticky-col-2,
.product-table tbody tr:nth-child(even) .sticky-col-3 {
  box-shadow: -4px 0 0 0 #fafbfd;
}

.product-table tr:hover .sticky-col {
  background: #eef2f7;
}
.product-table tr:hover .sticky-col-2,
.product-table tr:hover .sticky-col-3 {
  box-shadow: -4px 0 0 0 #eef2f7;
}

.product-table th.sticky-col {
  z-index: 5;
  background: var(--bg-card);
}
.product-table th.sticky-col-2,
.product-table th.sticky-col-3 {
  box-shadow: -4px 0 0 0 var(--bg-card);
}

.input-narrow {
  min-width: 90px;
  width: 110px;
}

.pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 18px;
  padding: 0 4px;
}

.pagination-controls {
  display: flex;
  align-items: center;
  gap: 12px;
}

.page-info {
  font-size: 15px;
  color: var(--ink-soft);
  font-weight: 600;
}
</style>
