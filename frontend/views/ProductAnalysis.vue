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
        <label>产品状态</label>
        <select v-model="filters.holdingStatus" class="input input-sm">
          <option value="">全部</option>
          <option v-for="opt in filterOptions.holdingStatuses" :key="opt" :value="opt">{{ opt }}</option>
        </select>
      </div>
      <div class="filter-group">
        <label>管理人</label>
        <select v-model="filters.manager" class="input input-sm">
          <option value="">全部</option>
          <option v-for="opt in filterOptions.managers" :key="opt" :value="opt">{{ opt }}</option>
        </select>
      </div>
      <div class="filter-group">
        <label>完结时间</label>
        <input v-model="filters.completeDateStart" type="date" class="input input-sm" />
        <span class="filter-sep">至</span>
        <input v-model="filters.completeDateEnd" type="date" class="input input-sm" />
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
        <label>标的</label>
        <select v-model="filters.code" class="input input-sm">
          <option value="">全部</option>
          <option v-for="opt in filterOptions.codes" :key="opt" :value="opt">{{ opt }}</option>
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
      <span class="text-label">📊 航班服务交易总表 · 产品表</span>
      <span class="badge badge-blue">本地数据库</span>
    </div>

    <div v-if="loading" class="loading-state">正在加载数据...</div>
    <div v-else-if="items.length > 0">
      <div class="table-wrap">
        <table class="data-table product-table">
          <thead>
            <tr>
              <th class="sticky-col">航班编号</th>
              <th>管理人</th>
              <th>产品名称</th>
              <th>持有状态</th>
              <th>申购日期</th>
              <th>存续时间（月）</th>
              <th>结构类型</th>
              <th>标的</th>
              <th>锁定期</th>
              <th>保证金比例</th>
              <th>敲入</th>
              <th>首月敲出</th>
              <th>入场价</th>
              <th>是否敲入</th>
              <th>每月递减</th>
              <th>托管券商</th>
              <th>交易对手</th>
              <th>期限</th>
              <th>完结时间</th>
              <th>降落伞</th>
              <th>派息障碍</th>
              <th>月票息（税费后）</th>
              <th>第一段票息（税费后）</th>
              <th>第二段票息（税费后）</th>
              <th>第三段票息（税费后）</th>
              <th>绝对收益率</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in items" :key="item.id">
              <td class="sticky-col">{{ item.id }}</td>
              <td>{{ item.manager }}</td>
              <td>{{ item.name }}</td>
              <td>
                <span class="badge" :class="item.holding_status === '存续' ? 'badge-green' : 'badge-amber'">
                  {{ item.holding_status }}
                </span>
              </td>
              <td>{{ item.issue_date || '--' }}</td>
              <td>{{ formatDuration(item) }}</td>
              <td>{{ item.structure_type || '--' }}</td>
              <td>{{ stripParentheses(item.code) }}</td>
              <td>{{ item.lock_months ?? '--' }}</td>
              <td>{{ item.margin_ratio ?? '--' }}</td>
              <td>{{ item.knock_in ?? '--' }}</td>
              <td>{{ item.first_knockout_ratio ?? '--' }}</td>
              <td>{{ item.entry_price ?? '--' }}</td>
              <td>{{ item.knocked_in ?? '--' }}</td>
              <td>{{ item.monthly_decrease ?? '--' }}</td>
              <td>{{ item.custodian || '--' }}</td>
              <td>{{ item.counterparty || '--' }}</td>
              <td>{{ item.term || '--' }}</td>
              <td>{{ item.complete_date || '--' }}</td>
              <td>{{ item.parachute ?? '--' }}</td>
              <td>{{ item.dividend_barrier ?? '--' }}</td>
              <td>{{ item.monthly_coupon ?? '--' }}</td>
              <td>{{ item.coupon_1st ?? '--' }}</td>
              <td>{{ item.coupon_2nd ?? '--' }}</td>
              <td>{{ item.coupon_3rd ?? '--' }}</td>
              <td>{{ item.absolute_return ?? '--' }}</td>
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
import { ref, computed, onMounted } from 'vue'

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

function stripParentheses(val) {
  if (!val) return '--'
  return val.replace(/[（(].*?[）)]/g, '').trim() || val
}

function formatDuration(item) {
  if (item.holding_status === '完结') {
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
      codes: (data.codes || []).map(c => c.replace(/[（(].*?[）)]/g, '').trim()),
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

.data-source-badge {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 18px;
}

.product-table {
  min-width: 3200px;
  font-size: 12.5px;
}

.product-table th {
  padding: 11px 14px;
  font-size: 11px;
  font-weight: 700;
  text-transform: none;
  letter-spacing: 0.03em;
  white-space: nowrap;
  background: rgba(241, 245, 249, 0.5);
  border-bottom: 1px solid var(--border-soft);
}

.product-table td {
  padding: 11px 14px;
  white-space: nowrap;
  border-bottom: 1px solid var(--border-soft);
  color: var(--ink-strong);
  font-size: 12.5px;
}

.product-table tr:hover td {
  background: rgba(241, 245, 249, 0.5);
}

.sticky-col {
  position: sticky;
  left: 0;
  z-index: 2;
  background: var(--bg-card);
}

.product-table tr:hover .sticky-col {
  background: rgba(241, 245, 249, 0.5);
}

.product-table th.sticky-col {
  z-index: 3;
  background: rgba(241, 245, 249, 0.5);
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
