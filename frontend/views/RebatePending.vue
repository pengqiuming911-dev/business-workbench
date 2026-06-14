<template>
  <div class="rebate-pending-page">
    <div class="page-header">
      <h1 class="text-page-title">待返费分析</h1>
      <p class="text-body">分析待返费订单，管理返费状态</p>
    </div>

    <!-- Filters -->
    <div class="filter-bar">
      <div class="filter-group">
        <label>客户姓名</label>
        <input
          v-model="filters.customerName"
          type="text"
          class="input input-sm"
          placeholder="请输入客户姓名"
        />
      </div>
      <div class="filter-group">
        <label>返还人</label>
        <input
          v-model="filters.rebateTarget"
          type="text"
          class="input input-sm"
          placeholder="请输入返还人"
        />
      </div>
      <div class="filter-group">
        <label>是否可返</label>
        <select v-model="filters.isReturnable" class="input input-sm">
          <option value="">全部</option>
          <option value="是">是</option>
          <option value="否">否</option>
        </select>
      </div>
      <div class="filter-group">
        <label>未返</label>
        <div class="multi-select" ref="unreturndDropdownRef">
          <button
            class="multi-select-trigger input input-sm"
            @click="toggleDropdown('unreturned')"
          >
            {{ unreturnedLabel }}
            <span class="caret">&#9662;</span>
          </button>
          <div v-show="openDropdown === 'unreturned'" class="multi-select-dropdown">
            <label class="multi-option" @click.prevent="toggleAllMulti(filters.unreturnedCategories, feeCategories)">
              <input type="checkbox" :checked="allChecked(filters.unreturnedCategories, feeCategories)" readonly />
              全选
            </label>
            <label v-for="cat in feeCategories" :key="'ur-' + cat" class="multi-option">
              <input type="checkbox" :value="cat" v-model="filters.unreturnedCategories" />
              {{ cat }}
            </label>
          </div>
        </div>
      </div>
      <div class="filter-group">
        <label>本次拟返</label>
        <div class="multi-select" ref="planDropdownRef">
          <button
            class="multi-select-trigger input input-sm"
            @click="toggleDropdown('plan')"
          >
            {{ planLabel }}
            <span class="caret">&#9662;</span>
          </button>
          <div v-show="openDropdown === 'plan'" class="multi-select-dropdown">
            <label class="multi-option" @click.prevent="toggleAllMulti(filters.planCategories, feeCategories)">
              <input type="checkbox" :checked="allChecked(filters.planCategories, feeCategories)" readonly />
              全选
            </label>
            <label v-for="cat in feeCategories" :key="'pl-' + cat" class="multi-option">
              <input type="checkbox" :value="cat" v-model="filters.planCategories" />
              {{ cat }}
            </label>
          </div>
        </div>
      </div>
      <div class="filter-actions">
        <button class="btn btn-primary btn-sm" @click="fetchData">
          <Search :size="14" />
          查询
        </button>
        <button class="btn btn-secondary btn-sm" @click="resetFilters">重置</button>
      </div>
    </div>

    <!-- Advanced filters -->
    <div class="advanced-toggle" @click="showAdvanced = !showAdvanced">
      <span class="chevron" :class="{ open: showAdvanced }">&#8250;</span>
      高级筛选
    </div>

    <div v-show="showAdvanced" class="filter-bar advanced-bar">
      <div class="filter-group">
        <label>订单号</label>
        <input
          v-model="filters.orderId"
          type="text"
          class="input input-sm"
          placeholder="请输入订单号"
        />
      </div>
      <div class="filter-group">
        <label>航班编号</label>
        <input
          v-model="filters.flightId"
          type="text"
          class="input input-sm"
          placeholder="请输入航班编号"
        />
      </div>
      <div class="filter-group">
        <label>航班名称</label>
        <input
          v-model="filters.productName"
          type="text"
          class="input input-sm"
          placeholder="请输入航班名称"
        />
      </div>
      <div class="filter-group">
        <label>应返</label>
        <div class="multi-select" ref="shouldReturnDropdownRef">
          <button
            class="multi-select-trigger input input-sm"
            @click="toggleDropdown('shouldReturn')"
          >
            {{ shouldReturnLabel }}
            <span class="caret">&#9662;</span>
          </button>
          <div v-show="openDropdown === 'shouldReturn'" class="multi-select-dropdown">
            <label class="multi-option" @click.prevent="toggleAllMulti(filters.shouldReturnCategories, feeCategories)">
              <input type="checkbox" :checked="allChecked(filters.shouldReturnCategories, feeCategories)" readonly />
              全选
            </label>
            <label v-for="cat in feeCategories" :key="'sr-' + cat" class="multi-option">
              <input type="checkbox" :value="cat" v-model="filters.shouldReturnCategories" />
              {{ cat }}
            </label>
          </div>
        </div>
      </div>
    </div>

    <!-- Action bar -->
    <div class="action-bar">
      <button class="btn btn-secondary btn-sm" @click="downloadCSV">
        <Download :size="14" />
        下载
      </button>
      <button class="btn btn-secondary btn-sm" @click="showBatchPanel = !showBatchPanel">
        <CheckSquare :size="14" />
        批量勾选
      </button>
    </div>

    <!-- Batch panel -->
    <div v-if="showBatchPanel" class="batch-panel">
      <span class="batch-label">批量勾选本次拟返：</span>
      <button
        v-for="cat in feeCategories"
        :key="'batch-' + cat"
        class="btn btn-sm"
        :class="batchChecked[cat] ? 'btn-primary' : 'btn-secondary'"
        @click="toggleBatchCategory(cat)"
      >
        {{ cat }}
      </button>
      <button class="btn btn-sm btn-primary" style="margin-left: 12px;" @click="applyBatch">
        应用
      </button>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="loading-state">加载中...</div>

    <!-- Empty -->
    <div v-else-if="!loading && items.length === 0 && loaded" class="empty-state">
      暂无待返费数据
    </div>

    <!-- Data table -->
    <div v-else-if="items.length > 0" class="table-wrap">
      <table class="data-table rebate-table">
        <colgroup>
          <col style="min-width: 120px" /><!-- 订单号 -->
          <col style="min-width: 100px" /><!-- 航班编号 -->
          <col style="min-width: 140px" /><!-- 航班名称 -->
          <col style="min-width: 90px" /><!-- 客户姓名 -->
          <col style="min-width: 90px" /><!-- 返还人 -->
          <col style="min-width: 100px" /><!-- 本金 -->
          <col span="3" style="min-width: 100px" /><!-- 应收 x3 -->
          <col span="3" style="min-width: 90px" /><!-- 返还比例 x3 -->
          <col span="3" style="min-width: 90px" /><!-- 扣税比例 x3 -->
          <col span="3" style="min-width: 100px" /><!-- 应返 x3 -->
          <col span="3" style="min-width: 100px" /><!-- 已返 x3 -->
          <col span="3" style="min-width: 100px" /><!-- 未返 x3 -->
          <col style="min-width: 80px" /><!-- 是否可返 -->
          <col span="4" style="min-width: 100px" /><!-- 本次拟返 x3 + 合计 -->
          <col style="min-width: 120px" /><!-- 操作 -->
        </colgroup>
        <thead>
          <tr class="header-group-row">
            <th rowspan="2" class="sticky-col">订单号</th>
            <th rowspan="2">航班编号</th>
            <th rowspan="2">航班名称</th>
            <th rowspan="2">客户姓名</th>
            <th rowspan="2">返还人</th>
            <th rowspan="2" class="num-col">本金</th>
            <th colspan="3" class="group-header group-receivable">应收</th>
            <th colspan="3" class="group-header group-ratio">返还比例</th>
            <th colspan="3" class="group-header group-tax">扣税比例</th>
            <th colspan="3" class="group-header group-should">应返</th>
            <th colspan="3" class="group-header group-returned">已返</th>
            <th colspan="3" class="group-header group-unreturned">未返</th>
            <th rowspan="2">是否可返</th>
            <th colspan="4" class="group-header group-plan">本次拟返</th>
            <th rowspan="2">操作</th>
          </tr>
          <tr class="header-sub-row">
            <!-- 应收 -->
            <th class="num-col sub-receivable">申购费</th>
            <th class="num-col sub-receivable">管理费实收</th>
            <th class="num-col sub-receivable">业绩报酬应收</th>
            <!-- 返还比例 -->
            <th class="num-col sub-ratio">申购费</th>
            <th class="num-col sub-ratio">管理费</th>
            <th class="num-col sub-ratio">业绩报酬</th>
            <!-- 扣税比例 -->
            <th class="num-col sub-tax">申购费</th>
            <th class="num-col sub-tax">管理费</th>
            <th class="num-col sub-tax">业绩报酬</th>
            <!-- 应返 -->
            <th class="num-col sub-should">申购费</th>
            <th class="num-col sub-should">管理费</th>
            <th class="num-col sub-should">业绩报酬</th>
            <!-- 已返 -->
            <th class="num-col sub-returned">申购费</th>
            <th class="num-col sub-returned">管理费</th>
            <th class="num-col sub-returned">业绩报酬</th>
            <!-- 未返 -->
            <th class="num-col sub-unreturned">申购费</th>
            <th class="num-col sub-unreturned">管理费</th>
            <th class="num-col sub-unreturned">业绩报酬</th>
            <!-- 本次拟返 -->
            <th class="sub-plan">申购费</th>
            <th class="sub-plan">管理费</th>
            <th class="sub-plan">业绩报酬</th>
            <th class="num-col sub-plan">合计</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(item, idx) in items" :key="item.order_id" :class="{ 'row-alt': idx % 2 === 1 }">
            <td class="sticky-col">{{ item.order_id }}</td>
            <td>{{ item.flight_id || '--' }}</td>
            <td class="name-cell" :title="item.product_name">{{ truncate(item.product_name, 12) }}</td>
            <td>{{ item.customer_name || '--' }}</td>
            <td>{{ item.rebate_target || '--' }}</td>
            <td class="num-col">{{ fmtNum(item.principal) }}</td>
            <!-- 应收 -->
            <td class="num-col">{{ fmtNum(calcSubscribeFee(item)) }}</td>
            <td class="num-col">{{ fmtNum(calcManagementFeeReceived(item)) }}</td>
            <td class="num-col">{{ fmtNum(calcPerformanceFeeReceivable(item)) }}</td>
            <!-- 返还比例 -->
            <td class="num-col">{{ fmtPct(item.subscribe_fee_ratio) }}</td>
            <td class="num-col">{{ fmtPct(item.management_fee_ratio) }}</td>
            <td class="num-col">{{ fmtPct(item.performance_fee_ratio) }}</td>
            <!-- 扣税比例 -->
            <td class="num-col">{{ fmtPct(item.tax_subscribe_ratio) }}</td>
            <td class="num-col">{{ fmtPct(item.tax_management_ratio) }}</td>
            <td class="num-col">{{ fmtPct(item.tax_performance_ratio) }}</td>
            <!-- 应返 -->
            <td class="num-col">{{ fmtNum(item.expected_subscribe ?? calcShouldReturn(item, 'subscribe')) }}</td>
            <td class="num-col">{{ fmtNum(item.expected_management ?? calcShouldReturn(item, 'management')) }}</td>
            <td class="num-col">{{ fmtNum(item.expected_performance ?? calcShouldReturn(item, 'performance')) }}</td>
            <!-- 已返 -->
            <td class="num-col">{{ fmtNum(item.returned_subscribe) }}</td>
            <td class="num-col">{{ fmtNum(item.returned_management) }}</td>
            <td class="num-col">{{ fmtNum(item.returned_performance) }}</td>
            <!-- 未返 -->
            <td class="num-col">{{ fmtNum(item.outstanding_subscribe ?? calcUnreturned(item, 'subscribe')) }}</td>
            <td class="num-col">{{ fmtNum(item.outstanding_management ?? calcUnreturned(item, 'management')) }}</td>
            <td class="num-col">{{ fmtNum(item.outstanding_performance ?? calcUnreturned(item, 'performance')) }}</td>
            <!-- 是否可返 -->
            <td class="returnable-cell">
              <button
                class="returnable-btn"
                :class="returnableClass(item.is_returnable)"
                @click="toggleReturnable(item)"
              >
                {{ item.is_returnable || '暂不可返' }}
              </button>
            </td>
            <!-- 本次拟返 checkboxes -->
            <td class="plan-cell">
              <label class="plan-check">
                <input
                  type="checkbox"
                  :checked="!!item.plan_subscribe"
                  @change="togglePlan(item, 'plan_subscribe', $event)"
                />
              </label>
            </td>
            <td class="plan-cell">
              <label class="plan-check">
                <input
                  type="checkbox"
                  :checked="!!item.plan_management"
                  @change="togglePlan(item, 'plan_management', $event)"
                />
              </label>
            </td>
            <td class="plan-cell">
              <label class="plan-check">
                <input
                  type="checkbox"
                  :checked="!!item.plan_performance"
                  @change="togglePlan(item, 'plan_performance', $event)"
                />
              </label>
            </td>
            <td class="num-col plan-total">{{ fmtNum(calcPlanTotal(item)) }}</td>
            <td class="action-cell">
              <button
                v-if="(item.outstanding_subscribe ?? calcUnreturned(item, 'subscribe')) > 0"
                class="btn btn-sm btn-action"
                @click="requestMarkReturned(item, '申购费', item.outstanding_subscribe ?? calcUnreturned(item, 'subscribe'))"
              >返申购费</button>
              <button
                v-if="(item.outstanding_management ?? calcUnreturned(item, 'management')) > 0"
                class="btn btn-sm btn-action"
                @click="requestMarkReturned(item, '管理费', item.outstanding_management ?? calcUnreturned(item, 'management'))"
              >返管理费</button>
              <button
                v-if="(item.outstanding_performance ?? calcUnreturned(item, 'performance')) > 0"
                class="btn btn-sm btn-action"
                @click="requestMarkReturned(item, '业绩报酬', item.outstanding_performance ?? calcUnreturned(item, 'performance'))"
              >返业绩报酬</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Confirm dialog -->
    <Teleport to="body">
      <div v-if="confirmDialog.visible" class="dialog-overlay" @click.self="confirmDialog.visible = false">
        <div class="dialog-box">
          <h3 class="dialog-title">确认返还</h3>
          <p class="dialog-body">{{ confirmDialog.message }}</p>
          <div class="dialog-actions">
            <button class="btn btn-secondary btn-sm" @click="confirmDialog.visible = false">取消</button>
            <button class="btn btn-primary btn-sm" @click="confirmMarkReturned">确认</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { Search, Download, CheckSquare } from '@lucide/vue'

// --- Constants ---
const feeCategories = ['申购费', '管理费', '业绩报酬']

// --- Reactive state ---
const loading = ref(false)
const loaded = ref(false)
const items = ref([])
const showAdvanced = ref(false)
const showBatchPanel = ref(false)
const openDropdown = ref('')

const filters = ref({
  customerName: '',
  rebateTarget: '',
  isReturnable: '',
  unreturnedCategories: [],
  planCategories: [],
  orderId: '',
  flightId: '',
  productName: '',
  shouldReturnCategories: [],
})

const batchChecked = ref({
  '申购费': false,
  '管理费': false,
  '业绩报酬': false,
})

const confirmDialog = ref({
  visible: false,
  message: '',
  item: null,
  category: '',
  amount: 0,
})

// --- Dropdown refs ---
const unreturndDropdownRef = ref(null)
const planDropdownRef = ref(null)
const shouldReturnDropdownRef = ref(null)

// --- Computed labels ---
const unreturnedLabel = computed(() => multiLabel(filters.value.unreturnedCategories, '未返'))
const planLabel = computed(() => multiLabel(filters.value.planCategories, '本次拟返'))
const shouldReturnLabel = computed(() => multiLabel(filters.value.shouldReturnCategories, '应返'))

function multiLabel(selected, fallback) {
  if (!selected || selected.length === 0) return `全部`
  if (selected.length === feeCategories.length) return '全部'
  return selected.join(', ')
}

function allChecked(selected, options) {
  return options.every(o => selected.includes(o))
}

function toggleAllMulti(selected, options) {
  if (allChecked(selected, options)) {
    selected.splice(0, selected.length)
  } else {
    selected.splice(0, selected.length, ...options)
  }
}

function toggleDropdown(name) {
  openDropdown.value = openDropdown.value === name ? '' : name
}

// Close dropdowns on outside click
function handleOutsideClick(e) {
  const refs = [unreturndDropdownRef.value, planDropdownRef.value, shouldReturnDropdownRef.value]
  const clickedInside = refs.some(r => r && r.contains(e.target))
  if (!clickedInside) {
    openDropdown.value = ''
  }
}

onMounted(() => {
  document.addEventListener('click', handleOutsideClick, true)
  fetchData()
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleOutsideClick, true)
})

// --- Formatting ---
function fmtNum(val) {
  if (val == null || val === '') return '--'
  const n = Number(val)
  if (isNaN(n)) return '--'
  return n.toFixed(2)
}

function fmtPct(val) {
  if (val == null || val === '') return '--'
  const n = Number(val)
  if (isNaN(n)) return '--'
  return (n * 100).toFixed(2) + '%'
}

function truncate(str, len) {
  if (!str) return '--'
  return str.length > len ? str.slice(0, len) + '...' : str
}

// --- Calculations ---
function calcSubscribeFee(item) {
  return (item.principal || 0) * (item.subscribe_fee_ratio || 0)
}

function calcManagementFeeReceived(item) {
  return (item.principal || 0) * (item.management_fee_ratio || 0)
}

function calcPerformanceFeeReceivable(item) {
  return (item.principal || 0) * (item.performance_fee_ratio || 0)
}

function calcShouldReturn(item, type) {
  const principal = item.principal || 0
  if (type === 'subscribe') {
    const fee = principal * (item.subscribe_fee_ratio || 0)
    const tax = item.tax_subscribe_ratio || 0
    return fee * (1 - tax)
  }
  if (type === 'management') {
    const fee = principal * (item.management_fee_ratio || 0)
    const tax = item.tax_management_ratio || 0
    return fee * (1 - tax)
  }
  if (type === 'performance') {
    const fee = principal * (item.performance_fee_ratio || 0)
    const tax = item.tax_performance_ratio || 0
    return fee * (1 - tax)
  }
  return 0
}

function calcUnreturned(item, type) {
  const should = calcShouldReturn(item, type)
  if (type === 'subscribe') return should - (item.returned_subscribe || 0)
  if (type === 'management') return should - (item.returned_management || 0)
  if (type === 'performance') return should - (item.returned_performance || 0)
  return 0
}

function calcPlanTotal(item) {
  let total = 0
  if (item.plan_subscribe) total += Math.max(0, item.outstanding_subscribe ?? calcUnreturned(item, 'subscribe'))
  if (item.plan_management) total += Math.max(0, item.outstanding_management ?? calcUnreturned(item, 'management'))
  if (item.plan_performance) total += Math.max(0, item.outstanding_performance ?? calcUnreturned(item, 'performance'))
  return total
}

// --- Returnable toggle ---
function returnableClass(val) {
  if (val === '是') return 'returnable-yes'
  if (val === '否') return 'returnable-no'
  return 'returnable-empty'
}

async function toggleReturnable(item) {
  const cycle = ['', '是', '否']
  const idx = cycle.indexOf(item.is_returnable)
  const next = cycle[(idx + 1) % cycle.length]
  item.is_returnable = next
  try {
    await fetch('/api/rebate/pending/status', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        order_id: item.order_id,
        is_returnable: item.is_returnable,
        plan_subscribe: item.plan_subscribe,
        plan_management: item.plan_management,
        plan_performance: item.plan_performance,
      }),
    })
  } catch {
    // Revert on failure
    item.is_returnable = cycle[idx]
  }
}

// --- Plan toggle ---
async function togglePlan(item, field, event) {
  const checked = event.target.checked
  item[field] = checked ? 1 : 0
  try {
    await fetch('/api/rebate/pending/status', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        order_id: item.order_id,
        is_returnable: item.is_returnable,
        plan_subscribe: item.plan_subscribe,
        plan_management: item.plan_management,
        plan_performance: item.plan_performance,
      }),
    })
  } catch {
    // Revert on failure
    item[field] = checked ? 0 : 1
  }
}

// --- Batch operations ---
function toggleBatchCategory(cat) {
  batchChecked.value[cat] = !batchChecked.value[cat]
}

function applyBatch() {
  const fieldMap = {
    '申购费': 'plan_subscribe',
    '管理费': 'plan_management',
    '业绩报酬': 'plan_performance',
  }
  for (const item of items.value) {
    for (const cat of feeCategories) {
      const field = fieldMap[cat]
      item[field] = batchChecked.value[cat] ? 1 : 0
    }
  }
  // Fire status updates for all items
  const promises = items.value.map(item =>
    fetch('/api/rebate/pending/status', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        order_id: item.order_id,
        is_returnable: item.is_returnable,
        plan_subscribe: item.plan_subscribe,
        plan_management: item.plan_management,
        plan_performance: item.plan_performance,
      }),
    }).catch(() => {})
  )
  Promise.all(promises)
  showBatchPanel.value = false
}

// --- Mark returned ---
function requestMarkReturned(item, category, amount) {
  confirmDialog.value = {
    visible: true,
    message: `确认将 ${item.order_id} 的 ${category} ${fmtNum(amount)} 标记为已返？`,
    item,
    category,
    amount,
  }
}

async function confirmMarkReturned() {
  const { item, category, amount } = confirmDialog.value
  confirmDialog.value.visible = false
  try {
    await fetch('/api/rebate/pending/mark-returned', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        order_id: item.order_id,
        category,
        amount,
      }),
    })
    await fetchData()
  } catch {
    // silently fail
  }
}

// --- Data fetching ---
async function fetchData() {
  loading.value = true
  try {
    const params = new URLSearchParams()
    const f = filters.value
    if (f.customerName) params.set('customer_name', encodeURIComponent(f.customerName))
    if (f.rebateTarget) params.set('rebate_target', encodeURIComponent(f.rebateTarget))
    if (f.isReturnable) params.set('is_returnable', encodeURIComponent(f.isReturnable))
    if (f.orderId) params.set('order_id', encodeURIComponent(f.orderId))
    if (f.flightId) params.set('flight_id', encodeURIComponent(f.flightId))
    if (f.productName) params.set('product_name', encodeURIComponent(f.productName))
    if (f.unreturnedCategories.length > 0 && f.unreturnedCategories.length < feeCategories.length) {
      params.set('unreturned_categories', f.unreturnedCategories.join(','))
    }
    if (f.planCategories.length > 0 && f.planCategories.length < feeCategories.length) {
      params.set('plan_categories', f.planCategories.join(','))
    }
    if (f.shouldReturnCategories.length > 0 && f.shouldReturnCategories.length < feeCategories.length) {
      params.set('should_return_categories', f.shouldReturnCategories.join(','))
    }

    const res = await fetch(`/api/rebate/pending?${params}`)
    if (!res.ok) throw new Error('加载失败')
    const data = await res.json()
    items.value = data.items || []
  } catch {
    items.value = []
  } finally {
    loading.value = false
    loaded.value = true
  }
}

function resetFilters() {
  filters.value = {
    customerName: '',
    rebateTarget: '',
    isReturnable: '',
    unreturnedCategories: [],
    planCategories: [],
    orderId: '',
    flightId: '',
    productName: '',
    shouldReturnCategories: [],
  }
  fetchData()
}

// --- CSV download ---
function downloadCSV() {
  if (items.value.length === 0) return

  const headers = [
    '订单号', '航班编号', '航班名称', '客户姓名', '返还人', '本金',
    '应收-申购费', '应收-管理费实收', '应收-业绩报酬应收',
    '返还比例-申购费', '返还比例-管理费', '返还比例-业绩报酬',
    '扣税比例-申购费', '扣税比例-管理费', '扣税比例-业绩报酬',
    '应返-申购费', '应返-管理费', '应返-业绩报酬',
    '已返-申购费', '已返-管理费', '已返-业绩报酬',
    '未返-申购费', '未返-管理费', '未返-业绩报酬',
    '是否可返',
    '本次拟返-申购费', '本次拟返-管理费', '本次拟返-业绩报酬', '本次拟返-合计',
  ]

  const rows = items.value.map(item => [
    item.order_id,
    item.flight_id,
    item.product_name,
    item.customer_name,
    item.rebate_target,
    fmtNum(item.principal),
    fmtNum(calcSubscribeFee(item)),
    fmtNum(calcManagementFeeReceived(item)),
    fmtNum(calcPerformanceFeeReceivable(item)),
    fmtPct(item.subscribe_fee_ratio),
    fmtPct(item.management_fee_ratio),
    fmtPct(item.performance_fee_ratio),
    fmtPct(item.tax_subscribe_ratio),
    fmtPct(item.tax_management_ratio),
    fmtPct(item.tax_performance_ratio),
    fmtNum(item.expected_subscribe ?? calcShouldReturn(item, 'subscribe')),
    fmtNum(item.expected_management ?? calcShouldReturn(item, 'management')),
    fmtNum(item.expected_performance ?? calcShouldReturn(item, 'performance')),
    fmtNum(item.returned_subscribe),
    fmtNum(item.returned_management),
    fmtNum(item.returned_performance),
    fmtNum(item.outstanding_subscribe ?? calcUnreturned(item, 'subscribe')),
    fmtNum(item.outstanding_management ?? calcUnreturned(item, 'management')),
    fmtNum(item.outstanding_performance ?? calcUnreturned(item, 'performance')),
    item.is_returnable || '',
    item.plan_subscribe ? fmtNum(item.outstanding_subscribe ?? calcUnreturned(item, 'subscribe')) : '0.00',
    item.plan_management ? fmtNum(item.outstanding_management ?? calcUnreturned(item, 'management')) : '0.00',
    item.plan_performance ? fmtNum(item.outstanding_performance ?? calcUnreturned(item, 'performance')) : '0.00',
    fmtNum(calcPlanTotal(item)),
  ])

  const BOM = '﻿'
  const csvContent = BOM + [
    headers.join(','),
    ...rows.map(r => r.map(c => `"${String(c ?? '').replace(/"/g, '""')}"`).join(',')),
  ].join('\n')

  const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `待返费分析_${new Date().toISOString().slice(0, 10)}.csv`
  link.click()
  URL.revokeObjectURL(url)
}
</script>

<style scoped>
:deep(.workbench-main) {
  max-width: none;
}

/* --- Filter bar --- */
.filter-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  align-items: flex-end;
  margin-bottom: 16px;
  padding: 20px 24px;
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
  font-size: 13px;
  font-weight: 700;
  white-space: nowrap;
}

.input-sm {
  height: 34px;
  min-height: 34px;
  padding: 0 12px;
  font-size: 13px;
  width: auto;
  min-width: 120px;
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
  font-weight: 700;
  cursor: pointer;
  margin-bottom: 16px;
  user-select: none;
  transition: color 180ms ease;
}

.advanced-toggle:hover {
  color: var(--brand-strong);
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

/* --- Multi-select dropdown --- */
.multi-select {
  position: relative;
}

.multi-select-trigger {
  display: inline-flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-width: 140px;
  text-align: left;
  cursor: pointer;
  background: #fff;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  color: var(--ink);
  font-size: 13px;
  white-space: nowrap;
}

.multi-select-trigger:focus {
  border-color: rgba(38, 119, 255, 0.34);
  box-shadow: 0 0 0 4px rgba(38, 119, 255, 0.1);
}

.caret {
  font-size: 10px;
  color: var(--ink-faint);
}

.multi-select-dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  z-index: 20;
  margin-top: 4px;
  min-width: 160px;
  padding: 6px 0;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  box-shadow: var(--shadow-md);
}

.multi-option {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 14px;
  font-size: 13px;
  color: var(--ink);
  cursor: pointer;
  transition: background 120ms ease;
}

.multi-option:hover {
  background: var(--bg-hover);
}

.multi-option input[type="checkbox"] {
  accent-color: var(--brand);
  width: 14px;
  height: 14px;
  cursor: pointer;
}

/* --- Action bar --- */
.action-bar {
  display: flex;
  gap: 10px;
  margin-bottom: 16px;
}

/* --- Batch panel --- */
.batch-panel {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
  padding: 14px 20px;
  background: var(--brand-soft);
  border: 1px solid rgba(41, 98, 255, 0.15);
  border-radius: var(--radius);
}

.batch-label {
  font-size: 13px;
  font-weight: 700;
  color: var(--ink);
  margin-right: 4px;
}

/* --- Table --- */
.rebate-table {
  min-width: 3600px;
  font-size: 14px;
}

.rebate-table thead {
  position: sticky;
  top: 0;
  z-index: 4;
}

.header-group-row th {
  padding: 8px 10px;
  font-size: 13px;
  font-weight: 800;
  text-align: center;
  white-space: nowrap;
  color: var(--ink-soft);
  border-bottom: 1px solid var(--border);
  background: #f1f5f9;
  letter-spacing: 0.02em;
}

.header-group-row th[rowspan="2"] {
  vertical-align: middle;
  background: #f1f5f9;
}

.header-sub-row th {
  padding: 6px 10px;
  font-size: 12px;
  font-weight: 700;
  text-align: center;
  white-space: nowrap;
  color: var(--ink-soft);
  border-bottom: 2px solid var(--border);
  letter-spacing: 0.01em;
}

/* Group header colors */
.group-receivable { background: #eef2ff !important; }
.group-ratio      { background: #f0fdf4 !important; }
.group-tax        { background: #fefce8 !important; }
.group-should     { background: #eff6ff !important; }
.group-returned   { background: #f0fdf4 !important; }
.group-unreturned { background: #fef2f2 !important; }
.group-plan       { background: #faf5ff !important; }

.sub-receivable   { background: #f5f7ff !important; }
.sub-ratio        { background: #f7fef9 !important; }
.sub-tax          { background: #fffef5 !important; }
.sub-should       { background: #f5f9ff !important; }
.sub-returned     { background: #f7fef9 !important; }
.sub-unreturned   { background: #fef8f8 !important; }
.sub-plan         { background: #fdf8ff !important; }

.rebate-table td {
  padding: 8px 10px;
  white-space: nowrap;
  border-bottom: 1px solid var(--border-soft);
  color: var(--ink-strong);
  font-size: 14px;
}

.rebate-table .num-col {
  text-align: right;
  font-variant-numeric: tabular-nums;
}

.rebate-table thead .num-col {
  text-align: center;
}

.row-alt td {
  background: var(--bg-page);
}

.rebate-table tr:hover td {
  background: rgba(41, 98, 255, 0.04);
}

.sticky-col {
  position: sticky;
  left: 0;
  z-index: 2;
  background: var(--bg-card);
}

.row-alt .sticky-col {
  background: var(--bg-page);
}

.rebate-table tr:hover .sticky-col {
  background: rgba(41, 98, 255, 0.04);
}

.header-group-row .sticky-col {
  z-index: 5;
  text-align: left;
}

.name-cell {
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
  cursor: default;
}

/* --- Returnable button --- */
.returnable-cell {
  text-align: center;
}

.returnable-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 64px;
  height: 26px;
  padding: 0 10px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  transition: all 150ms ease;
  background: var(--bg-card);
  color: var(--ink-faint);
}

.returnable-yes {
  color: #087c58;
  background: var(--success-soft);
  border-color: rgba(16, 185, 129, 0.2);
}

.returnable-no {
  color: #b43227;
  background: var(--danger-soft);
  border-color: rgba(239, 68, 68, 0.2);
}

.returnable-empty {
  color: var(--ink-faint);
  font-style: italic;
  background: var(--bg-card);
}

.returnable-btn:hover {
  transform: translateY(-1px);
  box-shadow: var(--shadow-sm);
}

/* --- Plan checkboxes --- */
.plan-cell {
  text-align: center;
}

.plan-check {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.plan-check input[type="checkbox"] {
  width: 16px;
  height: 16px;
  accent-color: var(--brand);
  cursor: pointer;
}

.plan-total {
  font-weight: 700;
  color: var(--brand) !important;
}

.action-cell {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: center;
  justify-content: center;
}

.btn-action {
  font-size: 11px;
  padding: 2px 6px;
  height: auto;
  line-height: 1.4;
  white-space: nowrap;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-sm);
  color: var(--brand);
  cursor: pointer;
  transition: all 150ms ease;
}

.btn-action:hover {
  background: var(--brand-soft);
  border-color: var(--brand);
  color: var(--brand-strong);
}

/* --- Confirm dialog --- */
.dialog-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.35);
  backdrop-filter: blur(2px);
}

.dialog-box {
  width: 400px;
  max-width: 90vw;
  padding: 28px;
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
}

.dialog-title {
  font-size: 17px;
  font-weight: 750;
  color: var(--ink-strong);
  margin-bottom: 12px;
}

.dialog-body {
  font-size: 14px;
  color: var(--ink);
  line-height: 1.6;
  margin-bottom: 24px;
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
