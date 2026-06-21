<template>
  <div class="customer-holding-page">
    <div class="filter-bar">
      <div class="filter-group">
        <label>客户</label>
        <input v-model="filters.customerName" type="text" class="input input-sm input-narrow" placeholder="客户关键词" />
        <label class="checkbox-label">
          <input v-model="filters.matchName" type="checkbox" />
          姓名
        </label>
        <label class="checkbox-label">
          <input v-model="filters.matchBuyer" type="checkbox" />
          实际申购人
        </label>
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
        <label>返佣对象</label>
        <select v-model="filters.rebateTarget" class="input input-sm">
          <option value="">全部</option>
          <option v-for="opt in filterOptions.rebateTargets" :key="opt" :value="opt">{{ opt }}</option>
        </select>
      </div>
      <div class="filter-group">
        <label>产品名称</label>
        <input v-model="filters.productName" type="text" class="input input-sm input-narrow" placeholder="模糊搜索" />
      </div>
      <div class="filter-actions">
        <button class="btn btn-primary btn-sm" @click="fetchData">查询</button>
        <button class="btn btn-secondary btn-sm" @click="resetFilters">重置</button>
        <FullscreenToggle target=".customer-holding-page .table-section" />
      </div>
    </div>

    <div class="advanced-toggle" @click="showAdvanced = !showAdvanced">
      <span class="chevron" :class="{ open: showAdvanced }">▸</span>
      高级筛选
    </div>

    <div v-show="showAdvanced" class="filter-bar advanced-bar">
      <div class="filter-group">
        <label>申购日期</label>
        <input v-model="filters.flightDateStart" type="date" class="input input-sm" />
        <span class="filter-sep">至</span>
        <input v-model="filters.flightDateEnd" type="date" class="input input-sm" />
      </div>
      <div class="filter-group">
        <label>完结日期</label>
        <input v-model="filters.completeDateStart" type="date" class="input input-sm" />
        <span class="filter-sep">至</span>
        <input v-model="filters.completeDateEnd" type="date" class="input input-sm" />
      </div>
      <div class="filter-group">
        <label>观察日</label>
        <input v-model="filters.obsDateStart" type="date" class="input input-sm" />
        <span class="filter-sep">至</span>
        <input v-model="filters.obsDateEnd" type="date" class="input input-sm" />
        <label class="checkbox-label">
          <input v-model="filters.obsDividend" type="checkbox" />
          派息
        </label>
        <label class="checkbox-label">
          <input v-model="filters.obsKnockout" type="checkbox" />
          敲出
        </label>
      </div>
    </div>

    <div class="update-hint">今日点位每日 15:05 自动更新，也支持手动刷新。</div>

    <div v-if="loading" class="loading-state">正在加载数据...</div>
    <div v-else-if="items.length > 0" class="table-section">
      <div class="table-wrap">
        <table class="data-table tx-table">
          <colgroup>
            <col style="width: 180px" /><!-- 产品名称(sticky) -->
            <col style="width: 90px" /><!-- 姓名(sticky) -->
          </colgroup>
          <thead>
            <tr>
              <th class="sticky-col sticky-col-1">产品名称</th>
              <th class="sticky-col sticky-col-2">姓名</th>
              <th>实际申购人</th>
              <th class="num">金额 / 万</th>
              <th class="num">申购费返还比例</th>
              <th class="num">管理费返还比例</th>
              <th class="num">业绩报酬返还比例</th>
              <th>返佣对象</th>
              <th>申购日期</th>
              <th>持有状态</th>
              <th>完结日期</th>
              <th>挂钩标的</th>
              <th>结构类型</th>
              <th>锁定期</th>
              <th>观察日</th>
              <th class="num">入场价</th>
              <th class="num">首月敲出</th>
              <th class="num">每月递减</th>
              <th class="num">敲出价</th>
              <th>今日点位</th>
              <th>敲出线以上 / 以下</th>
              <th>降落伞</th>
              <th>派息障碍（如有）</th>
              <th class="num">月票息（税费后）</th>
              <th class="num">第一段票息（税费后）</th>
              <th class="num">第二段票息（税费后）</th>
              <th class="num">第三段票息（税费后）</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(item, idx) in items" :key="idx">
              <td class="sticky-col sticky-col-1 name-cell" :title="item.product_name">{{ item.product_name || '--' }}</td>
              <td class="sticky-col sticky-col-2">{{ item.customer_name || '--' }}</td>
              <td>{{ item.actual_buyer || '--' }}</td>
              <td class="num">{{ item.amount ?? '--' }}</td>
              <td class="num">{{ item.subscribe_fee_ratio ?? '--' }}</td>
              <td class="num">{{ item.management_fee_ratio ?? '--' }}</td>
              <td class="num">{{ item.performance_fee_ratio ?? '--' }}</td>
              <td>{{ item.rebate_target || '--' }}</td>
              <td>{{ item.flight_date || '--' }}</td>
              <td>
                <span class="status-dot" :class="isActiveStatus(item.holding_status) ? 'status-active' : 'status-inactive'">
                  {{ normalizeHoldingStatus(item.holding_status) }}
                </span>
              </td>
              <td>{{ item.complete_date || '--' }}</td>
              <td>{{ item.underlying || '--' }}</td>
              <td>{{ item.structure_type || '--' }}</td>
              <td>{{ item.lock_period || '--' }}</td>
              <td class="obs-cell">
                <span>{{ displayObservationDay(item) }}</span>
                <span v-if="displayObservationType(item)" class="obs-type">{{ displayObservationType(item) }}</span>
              </td>
              <td class="num">{{ item.entry_price ?? '--' }}</td>
              <td class="num">{{ item.first_knockout_ratio != null ? Number(item.first_knockout_ratio).toFixed(2) : '--' }}</td>
              <td class="num">{{ item.monthly_decrease ?? '--' }}</td>
              <td class="num">{{ item.knockout_price ?? '--' }}</td>
              <td>
                <span>{{ displayTodayPrice(item) }}</span>
                <button
                  v-if="item.underlying"
                  class="refresh-btn"
                  title="刷新今日点位"
                  @click="refreshPrice(item)"
                >↻</button>
              </td>
              <td :class="knockoutPositionClass(displayKnockoutPosition(item))">
                {{ displayKnockoutPosition(item) }}
              </td>
              <td>{{ item.parachute ?? '--' }}</td>
              <td>{{ item.dividend_barrier ?? '--' }}</td>
              <td class="num">{{ item.monthly_coupon ?? '--' }}</td>
              <td class="num">{{ item.coupon_1st ?? '--' }}</td>
              <td class="num">{{ item.coupon_2nd ?? '--' }}</td>
              <td class="num">{{ item.coupon_3rd ?? '--' }}</td>
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
    <div v-else-if="loaded" class="empty-state">暂无交易数据</div>
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
  customerName: '',
  matchName: true,
  matchBuyer: true,
  obsDateStart: '',
  obsDateEnd: '',
  obsDividend: false,
  obsKnockout: false,
  rebateTarget: '',
  holdingStatus: '',
  productName: '',
  flightDateStart: '',
  flightDateEnd: '',
  completeDateStart: '',
  completeDateEnd: '',
})

const filterOptions = ref({
  rebateTargets: [],
  holdingStatuses: [],
})

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

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

function normalizeObservationType(val) {
  if (!val) return ''
  if (val.includes('派息') || val.includes('敲出')) return val
  if (val.includes('娲炬伅') && val.includes('鏁插嚭')) return '派息 / 敲出'
  if (val.includes('娲炬伅')) return '派息'
  if (val.includes('鏁插嚭')) return '敲出'
  return val
}

function normalizeKnockoutPosition(val) {
  if (!val) return '--'
  if (val.includes('以上') || val.includes('浠ヤ笂')) return '以上'
  if (val.includes('以下') || val.includes('浠ヤ笅')) return '以下'
  if (isCompletedStatus(val)) return '已完结'
  return val
}

function displayObservationDay(item) {
  if (isCompletedStatus(item.holding_status)) return '已完结'
  return item.observation_day || '--'
}

function displayObservationType(item) {
  if (isCompletedStatus(item.holding_status)) return ''
  return normalizeObservationType(item.observation_type || '')
}

function displayTodayPrice(item) {
  return item.today_price ?? '--'
}

function displayKnockoutPosition(item) {
  if (isCompletedStatus(item.holding_status)) return '已完结'
  return normalizeKnockoutPosition(item.knockout_position || '--')
}

function knockoutPositionClass(val) {
  if (val === '以上') return 'pos-above'
  if (val === '以下') return 'pos-below'
  if (val === '已完结') return 'pos-done'
  return ''
}

async function loadFilterOptions() {
  try {
    const res = await fetch('/api/holding/filter-options')
    if (!res.ok) return
    const data = await res.json()
    filterOptions.value = {
      rebateTargets: data.rebate_targets || [],
      holdingStatuses: data.holding_statuses || [],
    }
  } catch {}
}

async function fetchData() {
  loading.value = true
  try {
    const params = new URLSearchParams()
    const f = filters.value
    if (f.customerName) params.set('customer_name', f.customerName)
    params.set('match_name', f.matchName)
    params.set('match_buyer', f.matchBuyer)
    if (f.rebateTarget) params.set('rebate_target', f.rebateTarget)
    if (f.holdingStatus) params.set('holding_status', f.holdingStatus)
    if (f.obsDateStart) params.set('obs_date_start', f.obsDateStart)
    if (f.obsDateEnd) params.set('obs_date_end', f.obsDateEnd)
    params.set('observe_dividend', f.obsDividend)
    params.set('observe_knockout', f.obsKnockout)
    if (f.completeDateStart) params.set('complete_date_start', f.completeDateStart)
    if (f.completeDateEnd) params.set('complete_date_end', f.completeDateEnd)
    if (f.productName) params.set('product_name', f.productName)
    if (f.flightDateStart) params.set('flight_date_start', f.flightDateStart)
    if (f.flightDateEnd) params.set('flight_date_end', f.flightDateEnd)
    params.set('page', currentPage.value)
    params.set('page_size', pageSize)

    const res = await fetch(`/api/holding/transactions?${params}`)
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
    customerName: '',
    matchName: true,
    matchBuyer: true,
    obsDateStart: '',
    obsDateEnd: '',
    obsDividend: false,
    obsKnockout: false,
    rebateTarget: '',
    holdingStatus: '',
    productName: '',
    flightDateStart: '',
    flightDateEnd: '',
    completeDateStart: '',
    completeDateEnd: '',
  }
  currentPage.value = 1
  fetchData()
}

function goPage(page) {
  currentPage.value = page
  fetchData()
}

async function refreshPrice(item) {
  if (!item.underlying) return
  try {
    const res = await fetch(`/api/holding/refresh-price?code=${encodeURIComponent(item.underlying)}`, { method: 'POST' })
    if (!res.ok) return
    const data = await res.json()
    if (data.price != null) {
      item.today_price = data.price
      if (!isCompletedStatus(item.holding_status) && item.knockout_price != null) {
        item.knockout_position = Number(data.price) >= Number(item.knockout_price) ? '以上' : '以下'
      }
    }
  } catch {}
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

.customer-holding-page {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.customer-holding-page > .filter-bar {
  flex-shrink: 0;
}

.customer-holding-page > .advanced-toggle {
  flex-shrink: 0;
}

.customer-holding-page > .advanced-bar {
  flex-shrink: 0;
}

.customer-holding-page > .update-hint {
  flex-shrink: 0;
}

.table-section > .pagination {
  flex-shrink: 0;
}

.checkbox-label {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 14px;
  font-weight: 600;
  color: var(--ink);
  cursor: pointer;
  min-width: auto;
}

.checkbox-label input[type="checkbox"] {
  width: 16px;
  height: 16px;
  accent-color: var(--brand);
  cursor: pointer;
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

.update-hint {
  margin-bottom: 18px;
  color: var(--ink-soft);
  font-size: 14px;
  font-weight: 600;
}

.tx-table {
  min-width: 3400px;
}

.tx-table thead {
  position: sticky;
  top: 0;
  z-index: 4;
}

.tx-table th {
  background: #fef9ee;
  color: var(--ink-strong);
}

.name-cell {
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* sticky 列定位（base .sticky-col 规则已在 main.css 中定义） */
.sticky-col-1 { left: 0; }
.sticky-col-2 { left: 180px; box-shadow: -4px 0 0 0 var(--bg-card); }

.tx-table tr:hover .sticky-col-2 {
  box-shadow: -4px 0 0 0 #eef2f7;
}

.tx-table th.sticky-col {
  background: #fef9ee;
}

.tx-table th.sticky-col-2 {
  box-shadow: -4px 0 0 0 #fef9ee;
}

.obs-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.obs-type {
  font-size: 12px;
  color: var(--ink-soft);
  font-weight: 600;
}

.refresh-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  margin-left: 6px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--brand);
  font-size: 14px;
  cursor: pointer;
  transition: background 180ms ease, border-color 180ms ease;
}

.refresh-btn:hover {
  background: var(--brand-soft);
  border-color: var(--brand);
}

.pos-above { color: var(--success); }
.pos-below { color: var(--danger); }
.pos-done { color: var(--ink-soft); }
</style>
