<template>
  <div>
    <div class="filter-bar">
      <div class="filter-group">
        <label>客户</label>
        <input v-model="filters.customerName" type="text" class="input input-sm" placeholder="输入客户名" />
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
      <div class="filter-group">
        <label>返还对象</label>
        <select v-model="filters.rebateTarget" class="input input-sm">
          <option value="">全部</option>
          <option v-for="opt in filterOptions.rebateTargets" :key="opt" :value="opt">{{ opt }}</option>
        </select>
      </div>
      <div class="filter-group">
        <label>存续状态</label>
        <select v-model="filters.holdingStatus" class="input input-sm">
          <option value="">全部</option>
          <option v-for="opt in filterOptions.holdingStatuses" :key="opt" :value="opt">{{ opt }}</option>
        </select>
      </div>
      <div class="filter-actions">
        <button class="btn btn-primary btn-sm" @click="fetchData">查询</button>
        <button class="btn btn-secondary btn-sm" @click="resetFilters">重置</button>
      </div>
    </div>

    <div class="advanced-toggle" @click="showAdvanced = !showAdvanced">
      <span class="chevron" :class="{ open: showAdvanced }">›</span>
      高级筛选
    </div>

    <div v-show="showAdvanced" class="filter-bar advanced-bar">
      <div class="filter-group">
        <label>产品名字</label>
        <input v-model="filters.productName" type="text" class="input input-sm" placeholder="模糊搜索" />
      </div>
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
    </div>

    <div v-if="loading" class="loading-state">正在加载数据...</div>
    <div v-else-if="items.length > 0">
      <div class="table-wrap">
        <table class="data-table tx-table">
          <thead>
            <tr>
              <th>产品名字</th>
              <th>姓名</th>
              <th>实际申购人</th>
              <th>金额/万</th>
              <th>申购费返还比例</th>
              <th>管理费返还比例</th>
              <th>业绩报酬返还比例</th>
              <th>返还对象</th>
              <th>申购日期</th>
              <th>存续状态</th>
              <th>完结日期</th>
              <th>挂钩标的</th>
              <th>结构类型</th>
              <th>锁定期</th>
              <th>观察日</th>
              <th>入场价</th>
              <th>首月敲出</th>
              <th>每月降敲</th>
              <th>敲出价</th>
              <th>今日点位</th>
              <th>敲出线以上/下</th>
              <th>降落伞</th>
              <th>派息障碍（如有）</th>
              <th>月票息（税费后）</th>
              <th>第一段票息（税费后）</th>
              <th>第二段票息（税费后）</th>
              <th>第三段票息（税费后）</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(item, idx) in items" :key="idx">
              <td>{{ item.product_name || '--' }}</td>
              <td>{{ item.customer_name || '--' }}</td>
              <td>{{ item.actual_buyer || '--' }}</td>
              <td>{{ item.amount ?? '--' }}</td>
              <td>{{ item.subscribe_fee_ratio ?? '--' }}</td>
              <td>{{ item.management_fee_ratio ?? '--' }}</td>
              <td>{{ item.performance_fee_ratio ?? '--' }}</td>
              <td>{{ item.rebate_target || '--' }}</td>
              <td>{{ item.flight_date || '--' }}</td>
              <td>
                <span class="badge" :class="item.holding_status === '存续' ? 'badge-green' : 'badge-amber'">
                  {{ item.holding_status || '--' }}
                </span>
              </td>
              <td>{{ item.complete_date || '--' }}</td>
              <td>{{ item.underlying || '--' }}</td>
              <td>{{ item.structure_type || '--' }}</td>
              <td>{{ item.lock_period || '--' }}</td>
              <td class="obs-cell">
                <span>{{ item.observation_day || '--' }}</span>
                <span v-if="item.observation_type" class="obs-type">{{ item.observation_type }}</span>
              </td>
              <td>{{ item.entry_price ?? '--' }}</td>
              <td>{{ item.first_knockout_ratio ?? '--' }}</td>
              <td>{{ item.monthly_decrease ?? '--' }}</td>
              <td>{{ item.knockout_price ?? '--' }}</td>
              <td>
                <span>{{ item.today_price ?? '--' }}</span>
                <button
                  v-if="item.underlying"
                  class="refresh-btn"
                  title="刷新价格"
                  @click="refreshPrice(item)"
                >↻</button>
              </td>
              <td :class="knockoutPositionClass(item.knockout_position)">
                {{ item.knockout_position || '--' }}
              </td>
              <td>{{ item.parachute ?? '--' }}</td>
              <td>{{ item.dividend_barrier ?? '--' }}</td>
              <td>{{ item.monthly_coupon ?? '--' }}</td>
              <td>{{ item.coupon_1st ?? '--' }}</td>
              <td>{{ item.coupon_2nd ?? '--' }}</td>
              <td>{{ item.coupon_3rd ?? '--' }}</td>
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
import { ref, computed, onMounted } from 'vue'

const loading = ref(false)
const loaded = ref(false)
const items = ref([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = 50
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

function knockoutPositionClass(val) {
  if (!val) return ''
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
    }
  } catch {}
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
  gap: 16px;
  align-items: flex-end;
  margin-bottom: 18px;
  padding: 22px 26px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  box-shadow: var(--shadow-sm);
}

.filter-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-group label {
  color: var(--ink-soft);
  font-size: 12.5px;
  font-weight: 600;
  white-space: nowrap;
  letter-spacing: 0.01em;
}

.input-sm {
  height: 36px;
  min-height: 36px;
  padding: 0 12px;
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

.checkbox-label {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
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
  font-size: 13px;
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

.tx-table {
  min-width: 3400px;
  font-size: 12.5px;
}

.tx-table th {
  padding: 11px 14px;
  font-size: 11px;
  font-weight: 700;
  text-transform: none;
  letter-spacing: 0.03em;
  white-space: nowrap;
  background: rgba(241, 245, 249, 0.5);
  border-bottom: 1px solid var(--border-soft);
}

.tx-table td {
  padding: 11px 14px;
  white-space: nowrap;
  border-bottom: 1px solid var(--border-soft);
  color: var(--ink-strong);
  font-size: 12.5px;
}

.tx-table tr:hover td {
  background: rgba(241, 245, 249, 0.5);
}

.obs-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.obs-type {
  font-size: 10px;
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

.pos-above {
  color: var(--danger);
  font-weight: 700;
}

.pos-below {
  color: var(--success);
  font-weight: 700;
}

.pos-done {
  color: var(--ink-faint);
  font-weight: 600;
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
  font-size: 13px;
  color: var(--ink-soft);
  font-weight: 600;
}
</style>
