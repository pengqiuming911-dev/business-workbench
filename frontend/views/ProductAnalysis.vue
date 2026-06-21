<template>
  <div class="product-analysis-page">
    <div class="filter-bar primary-filter-bar">
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
        <button class="btn btn-primary btn-sm" @click="applyFilters">查询</button>
        <button class="btn btn-secondary btn-sm" @click="resetFilters">重置</button>
        <FullscreenToggle target=".product-analysis-page .table-section" />
      </div>
    </div>

    <button class="advanced-toggle" type="button" @click="showAdvanced = !showAdvanced">
      <span class="chevron" :class="{ open: showAdvanced }">▸</span>
      高级筛选
      <span class="advanced-note">{{ showAdvanced ? '收起' : '展开更多条件' }}</span>
    </button>

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

    <div v-if="activeFilterChips.length > 0" class="filter-chip-bar">
      <span class="chip-bar-label">当前筛选</span>
      <button
        v-for="chip in activeFilterChips"
        :key="chip.key"
        type="button"
        class="filter-chip"
        @click="clearFilter(chip.key)"
      >
        <span>{{ chip.label }}</span>
        <span class="chip-remove">×</span>
      </button>
      <button type="button" class="clear-all" @click="resetFilters">清空全部</button>
    </div>

    <div class="analysis-toolbar">
      <div class="summary-card">
        <span class="summary-label">筛选结果</span>
        <strong class="summary-value">{{ total }}</strong>
        <span class="summary-note">当前命中产品</span>
      </div>
      <div class="summary-card">
        <span class="summary-label">存续产品</span>
        <strong class="summary-value">{{ activeCount }}</strong>
        <span class="summary-note">持有状态为存续 / 未完结</span>
      </div>
      <div class="summary-card">
        <span class="summary-label">已完结产品</span>
        <strong class="summary-value">{{ completedCount }}</strong>
        <span class="summary-note">用于观察完结占比</span>
      </div>
      <div class="data-source-badge">
        <span class="text-label">数据源：交易总表 / 产品表</span>
        <span class="badge badge-blue">本地数据库</span>
      </div>
    </div>

    <div v-if="loading" class="loading-state">正在加载产品数据...</div>
    <div v-else-if="items.length > 0" class="table-section">
      <div class="table-wrap">
        <table class="data-table product-table">
          <colgroup>
            <col style="width: 120px" />
            <col style="width: 110px" />
            <col style="width: 188px" />
          </colgroup>
          <thead>
            <tr>
              <th class="sticky-col sticky-col-1">航班编号</th>
              <th class="sticky-col sticky-col-2">管理人</th>
              <th class="sticky-col sticky-col-3 name-cell">产品名称</th>
              <th>持有状态</th>
              <th>申购日期</th>
              <th class="num">存续时长(月)</th>
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
              <th class="num">月票息(税后)</th>
              <th class="num">第一段票息(税后)</th>
              <th class="num">第二段票息(税后)</th>
              <th class="num">第三段票息(税后)</th>
              <th class="num">绝对收益率</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in items" :key="item.id">
              <td class="sticky-col sticky-col-1">{{ item.id }}</td>
              <td class="sticky-col sticky-col-2">{{ item.manager || '--' }}</td>
              <td class="sticky-col sticky-col-3 name-cell" :title="item.name">{{ truncateName(item.name) }}</td>
              <td>
                <span class="state-tag" :class="isActiveStatus(item.holding_status) ? 'state-tag-active' : 'state-tag-completed'">
                  {{ normalizeHoldingStatus(item.holding_status) }}
                </span>
              </td>
              <td>{{ item.issue_date || '--' }}</td>
              <td class="num">{{ formatDuration(item) }}</td>
              <td>
                <span class="structure-tag">{{ item.structure_type || '--' }}</span>
              </td>
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
    <div v-else-if="loaded" class="empty-state">暂无匹配的产品数据</div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import FullscreenToggle from '../components/FullscreenToggle.vue'

const loading = ref(false)
const loaded = ref(false)
const items = ref([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = 20
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
const activeCount = computed(() => items.value.filter(item => isActiveStatus(item.holding_status)).length)
const completedCount = computed(() => items.value.length - activeCount.value)

const activeFilterChips = computed(() => {
  const f = filters.value
  const chips = []
  if (f.issueDateStart || f.issueDateEnd) {
    chips.push({
      key: 'issueDate',
      label: `申购日期 ${f.issueDateStart || '开始'} - ${f.issueDateEnd || '结束'}`,
    })
  }
  if (f.completeDateStart || f.completeDateEnd) {
    chips.push({
      key: 'completeDate',
      label: `完结日期 ${f.completeDateStart || '开始'} - ${f.completeDateEnd || '结束'}`,
    })
  }
  if (f.holdingStatus) {
    chips.push({ key: 'holdingStatus', label: `持有状态 ${normalizeHoldingStatus(f.holdingStatus)}` })
  }
  if (f.manager) {
    chips.push({ key: 'manager', label: `管理人 ${f.manager}` })
  }
  if (f.code) {
    chips.push({ key: 'code', label: `挂钩标的 ${stripParentheses(f.code)}` })
  }
  if (f.structureType) {
    chips.push({ key: 'structureType', label: `结构类型 ${f.structureType}` })
  }
  if (f.lockMonths) {
    chips.push({ key: 'lockMonths', label: `锁定期 ${f.lockMonths}` })
  }
  if (f.marginRatio) {
    chips.push({ key: 'marginRatio', label: `保证金比例 ${f.marginRatio}` })
  }
  return chips
})

function defaultFilters() {
  return {
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
}

function truncateName(val) {
  if (!val) return '--'
  return val.length > 12 ? `${val.slice(0, 12)}...` : val
}

function isCompletedStatus(val) {
  if (typeof val !== 'string') return false
  return ['完结', '已完结', '敲出', '到期'].some(keyword => val.includes(keyword))
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

function clearFilter(key) {
  if (key === 'issueDate') {
    filters.value.issueDateStart = ''
    filters.value.issueDateEnd = ''
  } else if (key === 'completeDate') {
    filters.value.completeDateStart = ''
    filters.value.completeDateEnd = ''
  } else {
    filters.value[key] = ''
  }
  applyFilters()
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

function applyFilters() {
  currentPage.value = 1
  fetchData()
}

function resetFilters() {
  filters.value = defaultFilters()
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
:deep(.workbench-main) {
  max-width: none;
}

.product-analysis-page {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.primary-filter-bar {
  margin-bottom: 14px;
}

.advanced-toggle {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  width: fit-content;
  padding: 0;
  margin-bottom: 14px;
  border: none;
  background: transparent;
  color: var(--brand);
  font-size: 14px;
  font-weight: 700;
}

.advanced-note {
  color: var(--ink-faint);
  font-weight: 600;
}

.advanced-bar {
  margin-top: -4px;
}

.chevron {
  font-size: 16px;
  transition: transform 0.2s ease;
}

.chevron.open {
  transform: rotate(90deg);
}

.filter-chip-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  padding: 14px 18px;
  background: rgba(255, 255, 255, 0.86);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
}

.chip-bar-label {
  color: var(--ink-soft);
  font-size: 12px;
  font-weight: 700;
}

.filter-chip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 30px;
  padding: 0 10px;
  border: 1px solid rgba(31, 58, 138, 0.12);
  border-radius: 999px;
  background: var(--brand-soft);
  color: var(--brand);
  font-size: 12px;
  font-weight: 700;
}

.chip-remove {
  font-size: 14px;
  line-height: 1;
}

.clear-all {
  border: none;
  background: transparent;
  color: var(--ink-soft);
  font-size: 12px;
  font-weight: 700;
}

.analysis-toolbar {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 200px)) minmax(220px, 1fr);
  gap: 14px;
  align-items: stretch;
  margin-bottom: 18px;
}

.summary-card {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 16px 18px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
}

.summary-label {
  color: var(--ink-soft);
  font-size: 12px;
  font-weight: 700;
}

.summary-value {
  color: var(--ink-strong);
  font-size: 28px;
  line-height: 1.1;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
}

.summary-note {
  color: var(--ink-faint);
  font-size: 12px;
}

.data-source-badge {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  padding: 0 4px;
}

.table-section {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.table-wrap {
  flex: 1;
  min-height: 0;
}

.product-table {
  min-width: 3200px;
}

.product-table thead {
  position: sticky;
  top: 0;
  z-index: 4;
}

.product-table th {
  color: var(--ink-soft);
  background: #f8fafc;
}

.name-cell {
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
  cursor: default;
  font-weight: 600;
}

.state-tag,
.structure-tag {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 0 10px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 800;
  line-height: 1;
}

.state-tag-active {
  color: var(--success);
  background: var(--success-soft);
}

.state-tag-completed {
  color: var(--ink-soft);
  background: #edf2f7;
}

.structure-tag {
  color: var(--brand);
  background: var(--brand-soft);
}

.sticky-col {
  position: sticky;
  z-index: 2;
  background: #fff;
}

.sticky-col-1 {
  left: 0;
}

.sticky-col-2 {
  left: 120px;
  box-shadow: -4px 0 0 0 #fff;
}

.sticky-col-3 {
  left: 230px;
  box-shadow: -4px 0 0 0 #fff;
}

.product-table tbody tr:nth-child(even) .sticky-col {
  background: var(--bg-zebra);
}

.product-table tbody tr:nth-child(even) .sticky-col-2,
.product-table tbody tr:nth-child(even) .sticky-col-3 {
  box-shadow: -4px 0 0 0 var(--bg-zebra);
}

.product-table tr:hover .sticky-col {
  background: #eef4fb;
}

.product-table tr:hover .sticky-col-2,
.product-table tr:hover .sticky-col-3 {
  box-shadow: -4px 0 0 0 #eef4fb;
}

.product-table th.sticky-col {
  z-index: 5;
  background: #f8fafc;
}

.product-table th.sticky-col-2,
.product-table th.sticky-col-3 {
  box-shadow: -4px 0 0 0 #f8fafc;
}

@media (max-width: 1440px) {
  .analysis-toolbar {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .data-source-badge {
    justify-content: flex-start;
    padding: 10px 0 0;
  }
}

@media (max-width: 1024px) {
  .filter-actions {
    width: 100%;
    margin-left: 0;
    justify-content: flex-end;
  }
}

@media (max-width: 720px) {
  .analysis-toolbar {
    grid-template-columns: 1fr;
  }
}
</style>
