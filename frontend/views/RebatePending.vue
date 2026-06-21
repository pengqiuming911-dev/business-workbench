<template>
  <div class="rebate-pending-page">
    <div v-if="!embedded" class="page-header">
      <h1 class="text-page-title">待返费分析</h1>
      <p class="text-body">分析待返费订单，管理返费状态</p>
    </div>

    <!-- Filters -->
    <div class="filter-bar primary-filter-bar" :class="{ 'show-all': showAdvanced }">
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
        <label>客户姓名</label>
        <input
          v-model="filters.customerName"
          type="text"
          class="input input-sm input-compact"
          placeholder="客户姓名"
        />
      </div>
      <div class="filter-group">
        <label>返还人</label>
        <input
          v-model="filters.rebateTarget"
          type="text"
          class="input input-sm input-compact"
          placeholder="返还人"
        />
      </div>
      <div class="filter-group">
        <label>本次拟返</label>
        <div class="multi-select" ref="planDropdownRef">
          <button
            class="multi-select-trigger input input-sm input-select-compact"
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
      <div class="filter-group">
        <label>是否可返</label>
        <select v-model="filters.isReturnable" class="input input-sm input-select-compact">
          <option value="">全部</option>
          <option value="待返">待返</option>
          <option value="暂不可返">暂不可返</option>
        </select>
      </div>
      <div class="filter-group">
        <label>未返</label>
        <div class="multi-select" ref="unreturndDropdownRef">
          <button
            class="multi-select-trigger input input-sm input-select-compact"
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
        <label>航班编号</label>
        <input v-model="filters.flightId" type="text" class="input input-sm input-compact" placeholder="航班编号" />
      </div>
      <div class="filter-group">
        <label>航班名称</label>
        <input v-model="filters.productName" type="text" class="input input-sm input-compact" placeholder="航班名称" />
      </div>
      <div class="filter-group">
        <label>校验类别</label>
        <select v-model="filters.checkCategory" class="input input-sm input-select-compact">
          <option value="">全部</option>
          <option value="subscribe">申购费</option>
          <option value="management">管理费</option>
          <option value="performance">业绩报酬</option>
        </select>
      </div>
      <div class="filter-group">
        <label>校验结果</label>
        <select v-model="filters.checkResult" class="input input-sm input-select-compact">
          <option value="">全部</option>
          <option value="-">-</option>
          <option value="T">T</option>
          <option value="F">F</option>
        </select>
      </div>
      <div class="filter-group">
        <label>应返</label>
        <div class="multi-select" ref="shouldReturnDropdownRef">
          <button
            class="multi-select-trigger input input-sm input-select-compact"
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
      <div class="filter-actions">
        <button class="btn btn-primary btn-sm" @click="applyFilters">
          <Search :size="14" />
          查询
        </button>
        <button class="btn btn-secondary btn-sm" @click="resetFilters">重置</button>
        <FullscreenToggle target=".rebate-pending-page .table-section" />
      </div>
    </div>

    <button class="advanced-toggle" type="button" @click="showAdvanced = !showAdvanced">
      <span class="chevron" :class="{ open: showAdvanced }">▸</span>
      高级筛选
      <span class="advanced-note">{{ showAdvanced ? '收起' : '展开更多条件' }}</span>
    </button>

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

    <!-- Action bar -->
    <div class="action-bar">
      <div class="action-bar-meta">
        <span class="text-label">筛选后 {{ filteredItems.length }} 条 / 全部 {{ items.length }} 条</span>
      </div>
      <div class="action-bar-actions">
      <button class="btn btn-secondary btn-sm" @click="downloadCSV">
        <Download :size="14" />
        下载
      </button>
      <button class="btn btn-secondary btn-sm" @click="showBatchPanel = !showBatchPanel">
        <CheckSquare :size="14" />
        批量勾选
      </button>
      </div>
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
    <div v-else-if="items.length > 0" class="table-section">
      <div class="table-wrap">
      <table class="data-table rebate-table">
        <colgroup>
          <col style="min-width: 120px" /><!-- 订单号 -->
          <col style="min-width: 100px" /><!-- 航班编号 -->
          <col style="min-width: 140px" /><!-- 航班名称 -->
          <col style="min-width: 90px" /><!-- 客户姓名 -->
          <col style="min-width: 90px" /><!-- 返还人 -->
          <col style="min-width: 100px" /><!-- 本金 -->
          <col style="min-width: 80px" /><!-- 申购费率 -->
          <col span="3" style="min-width: 100px" /><!-- 应收 x3 -->
          <col span="3" style="min-width: 90px" /><!-- 返还比例 x3 -->
          <col span="3" style="min-width: 90px" /><!-- 扣税比例 x3 -->
          <col span="3" style="min-width: 100px" /><!-- 应返 x3 -->
          <col span="3" style="min-width: 100px" /><!-- 已返 x3 -->
          <col span="3" style="min-width: 100px" /><!-- 未返 x3 -->
          <col style="min-width: 80px" /><!-- 是否可返 -->
          <col span="3" style="min-width: 74px" /><!-- 校验 x3 -->
          <col span="4" style="min-width: 72px" /><!-- 本次拟返 x3 + 合计 -->
          <col style="min-width: 120px" /><!-- 操作 -->
        </colgroup>
        <thead>
          <tr class="header-group-row">
            <th rowspan="2" class="sticky-col">订单号</th>
            <th rowspan="2">航班编号</th>
            <th rowspan="2">航班名称</th>
            <th rowspan="2">客户姓名</th>
            <th rowspan="2">返还人</th>
            <th rowspan="2" class="num">本金</th>
            <th rowspan="2" class="num">申购费率</th>
            <th colspan="3" class="group-header group-receivable">应收</th>
            <th colspan="3" class="group-header group-ratio">返还比例</th>
            <th colspan="3" class="group-header group-tax">扣税比例</th>
            <th colspan="3" class="group-header group-should">应返</th>
            <th colspan="3" class="group-header group-returned">已返</th>
            <th colspan="3" class="group-header group-unreturned">未返</th>
            <th rowspan="2">是否可返</th>
            <th colspan="3" class="group-header group-check">校验</th>
            <th colspan="4" class="group-header group-plan">本次拟返</th>
            <th rowspan="2">操作</th>
          </tr>
          <tr class="header-sub-row">
            <!-- 应收 -->
            <th class="num sub-receivable">申购费</th>
            <th class="num sub-receivable">管理费实收</th>
            <th class="num sub-receivable">业绩报酬应收</th>
            <!-- 返还比例 -->
            <th class="num sub-ratio">申购费</th>
            <th class="num sub-ratio">管理费</th>
            <th class="num sub-ratio">业绩报酬</th>
            <!-- 扣税比例 -->
            <th class="num sub-tax">申购费</th>
            <th class="num sub-tax">管理费</th>
            <th class="num sub-tax">业绩报酬</th>
            <!-- 应返 -->
            <th class="num sub-should">申购费</th>
            <th class="num sub-should">管理费</th>
            <th class="num sub-should">业绩报酬</th>
            <!-- 已返 -->
            <th class="num sub-returned">申购费</th>
            <th class="num sub-returned">管理费</th>
            <th class="num sub-returned">业绩报酬</th>
            <!-- 未返 -->
            <th class="num sub-unreturned">申购费</th>
            <th class="num sub-unreturned">管理费</th>
            <th class="num sub-unreturned">业绩报酬</th>
            <!-- 校验 -->
            <th class="sub-check">申购费</th>
            <th class="sub-check">管理费</th>
            <th class="sub-check">业绩报酬</th>
            <!-- 本次拟返 -->
            <th class="sub-plan">申购费</th>
            <th class="sub-plan">管理费</th>
            <th class="sub-plan">业绩报酬</th>
            <th class="num sub-plan">合计</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(item, idx) in pagedItems" :key="item.order_id" :class="{ 'row-alt': idx % 2 === 1 }">
            <td class="sticky-col">{{ item.order_id }}</td>
            <td>{{ item.flight_id || '--' }}</td>
            <td class="name-cell" :title="item.product_name">{{ truncate(item.product_name, 12) }}</td>
            <td>{{ item.customer_name || '--' }}</td>
            <td>{{ item.rebate_target || '--' }}</td>
            <td>{{ fmtNum(item.principal) }}</td>
            <td>{{ fmtPct(item.subscribe_fee_rate) }}</td>
            <!-- 应收 -->
            <td>{{ fmtNum(calcSubscribeFee(item)) }}</td>
            <td>{{ fmtNum(calcManagementFeeReceived(item)) }}</td>
            <td>{{ fmtNum(calcPerformanceFeeReceivable(item)) }}</td>
            <!-- 返还比例 -->
            <td>{{ fmtPct(item.subscribe_fee_ratio) }}</td>
            <td>{{ fmtPct(item.management_fee_ratio) }}</td>
            <td>{{ fmtPct(item.performance_fee_ratio) }}</td>
            <!-- 扣税比例 -->
            <td>{{ fmtPct(item.tax_subscribe_ratio) }}</td>
            <td>{{ fmtPct(item.tax_management_ratio) }}</td>
            <td>{{ fmtPct(item.tax_performance_ratio) }}</td>
            <!-- 应返 -->
            <td>{{ fmtNum(item.expected_subscribe ?? calcShouldReturn(item, 'subscribe')) }}</td>
            <td>{{ fmtNum(item.expected_management ?? calcShouldReturn(item, 'management')) }}</td>
            <td>{{ fmtNum(item.expected_performance ?? calcShouldReturn(item, 'performance')) }}</td>
            <!-- 已返 -->
            <td>{{ fmtNum(item.returned_subscribe) }}</td>
            <td>{{ fmtNum(item.returned_management) }}</td>
            <td>{{ fmtNum(item.returned_performance) }}</td>
            <!-- 未返 -->
            <td>{{ fmtNum(item.outstanding_subscribe ?? calcUnreturned(item, 'subscribe')) }}</td>
            <td>{{ fmtNum(item.outstanding_management ?? calcUnreturned(item, 'management')) }}</td>
            <td>{{ fmtNum(item.outstanding_performance ?? calcUnreturned(item, 'performance')) }}</td>
            <!-- 是否可返 -->
            <td class="returnable-cell">
              <span
                class="returnable-pill"
                :class="returnableClass(item.is_returnable)"
              >
                {{ item.is_returnable || '暂不可返' }}
              </span>
            </td>
            <!-- 校验 -->
            <td class="check-cell">
              <span class="check-pill" :class="checkClass(item.check_subscribe)">{{ item.check_subscribe || '--' }}</span>
            </td>
            <td class="check-cell">
              <span class="check-pill" :class="checkClass(item.check_management)">{{ item.check_management || '--' }}</span>
            </td>
            <td class="check-cell">
              <span class="check-pill" :class="checkClass(item.check_performance)">{{ item.check_performance || '--' }}</span>
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
            <td class="plan-total">{{ fmtNum(calcPlanTotal(item)) }}</td>
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

      <!-- 分页 -->
      <div class="pagination">
        <span class="text-label">共 {{ filteredItems.length }} 条（筛选后） / {{ items.length }} 条（全部） · 第 {{ page }} / {{ totalPages }} 页</span>
        <div class="pagination-controls">
          <button class="btn btn-secondary btn-sm" :disabled="page <= 1" @click="gotoPage(page - 1)">上一页</button>
          <button class="btn btn-secondary btn-sm" :disabled="page >= totalPages" @click="gotoPage(page + 1)">下一页</button>
        </div>
      </div>
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
import FullscreenToggle from '../components/FullscreenToggle.vue'

defineProps({
  embedded: { type: Boolean, default: false },
})

// --- Constants ---
const feeCategories = ['申购费', '管理费', '业绩报酬']

// --- Reactive state ---
const loading = ref(false)
const loaded = ref(false)
const items = ref([])

// --- 分页：items 仍持有全部筛选结果(供导出/批量使用)，表格只渲染当前页 ---
const page = ref(1)
const pageSize = ref(20)
const totalPages = computed(() => Math.max(1, Math.ceil(filteredItems.value.length / pageSize.value)))
const filteredItems = computed(() => {
  const f = filters.value
  if (!f.checkCategory || !f.checkResult) return items.value
  return items.value.filter(item => {
    let checkValue = ''
    if (f.checkCategory === 'subscribe') checkValue = item.check_subscribe
    else if (f.checkCategory === 'management') checkValue = item.check_management
    else if (f.checkCategory === 'performance') checkValue = item.check_performance
    return checkValue === f.checkResult
  })
})
const pagedItems = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return filteredItems.value.slice(start, start + pageSize.value)
})
function gotoPage(p) {
  const n = Math.min(Math.max(1, p), totalPages.value)
  page.value = n
}
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
  checkCategory: '',
  checkResult: '',
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
const activeFilterChips = computed(() => {
  const f = filters.value
  const chips = []
  if (f.orderId) chips.push({ key: 'orderId', label: `订单号 ${f.orderId}` })
  if (f.customerName) chips.push({ key: 'customerName', label: `客户 ${f.customerName}` })
  if (f.rebateTarget) chips.push({ key: 'rebateTarget', label: `返费对象 ${f.rebateTarget}` })
  if (f.isReturnable) chips.push({ key: 'isReturnable', label: `是否可返 ${f.isReturnable}` })
  if (f.planCategories.length > 0 && f.planCategories.length < feeCategories.length) {
    chips.push({ key: 'planCategories', label: `本次拟返 ${f.planCategories.join(', ')}` })
  }
  if (f.unreturnedCategories.length > 0 && f.unreturnedCategories.length < feeCategories.length) {
    chips.push({ key: 'unreturnedCategories', label: `未返 ${f.unreturnedCategories.join(', ')}` })
  }
  if (f.flightId) chips.push({ key: 'flightId', label: `航班编号 ${f.flightId}` })
  if (f.productName) chips.push({ key: 'productName', label: `航班名称 ${f.productName}` })
  if (f.checkCategory) chips.push({ key: 'checkCategory', label: `校验类别 ${f.checkCategory}` })
  if (f.checkResult) chips.push({ key: 'checkResult', label: `校验结果 ${f.checkResult}` })
  if (f.shouldReturnCategories.length > 0 && f.shouldReturnCategories.length < feeCategories.length) {
    chips.push({ key: 'shouldReturnCategories', label: `应返 ${f.shouldReturnCategories.join(', ')}` })
  }
  return chips
})

function defaultFilters() {
  return {
    customerName: '',
    rebateTarget: '',
    isReturnable: '',
    unreturnedCategories: [],
    planCategories: [],
    orderId: '',
    flightId: '',
    productName: '',
    shouldReturnCategories: [],
    checkCategory: '',
    checkResult: '',
  }
}

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
  if (val === '-' || val === '暂不可返' || val === '不可返') return '-'
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
  return (item.principal || 0) * (item.subscribe_fee_rate || 0)
}

function calcManagementFeeReceived(item) {
  return item.management_fee_received || 0
}

function calcPerformanceFeeReceivable(item) {
  return item.performance_fee_receivable || 0
}

function taxNum(v) {
  const n = Number(v)
  return isNaN(n) ? 0 : n
}

function calcShouldReturn(item, type) {
  if (type === 'subscribe') {
    // 应返申购费 = 本金 × 申购费率 × 申购费返还比例 × (1 − 申购费扣税)
    return (item.principal || 0) * (item.subscribe_fee_rate || 0) * (item.subscribe_fee_ratio || 0) * (1 - taxNum(item.tax_subscribe_ratio))
  }
  if (type === 'management') {
    // 应返管理费 = 管理费实收 × 管理费返还比例 × (1 − 管理费扣税)
    return (item.management_fee_received || 0) * (item.management_fee_ratio || 0) * (1 - taxNum(item.tax_management_ratio))
  }
  if (type === 'performance') {
    // 应返业绩报酬 = 业绩报酬应收 × 业绩报酬返还比例 × (1 − 业绩报酬扣税)
    return (item.performance_fee_receivable || 0) * (item.performance_fee_ratio || 0) * (1 - taxNum(item.tax_performance_ratio))
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

function returnableClass(val) {
  if (val === '待返') return 'returnable-ready'
  if (val === '暂不可返') return 'returnable-waiting'
  return 'returnable-empty'
}

function checkClass(val) {
  if (val === 'T') return 'check-pass'
  if (val === 'F') return 'check-fail'
  return 'check-empty'
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
  page.value = 1
  try {
    const params = new URLSearchParams()
    const f = filters.value
    if (f.customerName) params.set('customer_name', f.customerName)
    if (f.rebateTarget) params.set('rebate_target', f.rebateTarget)
    if (f.isReturnable) params.set('is_returnable', f.isReturnable)
    if (f.orderId) params.set('order_id', f.orderId)
    if (f.flightId) params.set('flight_id', f.flightId)
    if (f.productName) params.set('product_name', f.productName)
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
    items.value = (data.items || []).map(item => {
      const principal =
        item.principal != null && item.principal !== 0
          ? item.principal
          : item.subscribe_amount != null && item.subscribe_amount !== 0
            ? item.subscribe_amount
            : item.amount != null
              ? item.amount
              : 0
      return { ...item, principal }
    })
  } catch {
    items.value = []
  } finally {
    loading.value = false
    loaded.value = true
  }
}

function applyFilters() {
  page.value = 1
  fetchData()
}

function clearFilter(key) {
  if (Array.isArray(filters.value[key])) {
    filters.value[key] = []
  } else {
    filters.value[key] = ''
  }
  applyFilters()
}

function resetFilters() {
  filters.value = defaultFilters()
  applyFilters()
}

// --- CSV download ---
function downloadCSV() {
  if (items.value.length === 0) return

  const headers = [
    '订单号', '航班编号', '航班名称', '客户姓名', '返还人', '本金', '申购费率',
    '应收-申购费', '应收-管理费实收', '应收-业绩报酬应收',
    '返还比例-申购费', '返还比例-管理费', '返还比例-业绩报酬',
    '扣税比例-申购费', '扣税比例-管理费', '扣税比例-业绩报酬',
    '应返-申购费', '应返-管理费', '应返-业绩报酬',
    '已返-申购费', '已返-管理费', '已返-业绩报酬',
    '未返-申购费', '未返-管理费', '未返-业绩报酬',
    '是否可返',
    '校验-申购费', '校验-管理费', '校验-业绩报酬',
    '本次拟返-申购费', '本次拟返-管理费', '本次拟返-业绩报酬', '本次拟返-合计',
  ]

  const rows = items.value.map(item => [
    item.order_id,
    item.flight_id,
    item.product_name,
    item.customer_name,
    item.rebate_target,
    fmtNum(item.principal),
    fmtPct(item.subscribe_fee_rate),
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
    item.check_subscribe || '--',
    item.check_management || '--',
    item.check_performance || '--',
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

.rebate-pending-page {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  font-size: 12px;
}

.rebate-pending-page > .page-header {
  flex-shrink: 0;
}

.rebate-pending-page > .filter-bar {
  flex-shrink: 0;
  margin-bottom: 6px;
}

.primary-filter-bar:not(.show-all) .filter-group:nth-child(n + 6):nth-child(-n + 10) {
  display: none;
}

.rebate-pending-page > .action-bar {
  flex-shrink: 0;
  margin-bottom: 6px;
}

.rebate-pending-page > .batch-panel {
  flex-shrink: 0;
}

.table-wrap {
  overflow-x: auto;
}

.table-section > .pagination {
  flex-shrink: 0;
}

.input-compact {
  min-width: 102px;
}

.input-narrow {
  min-width: 90px;
  width: 110px;
}

.input-select-compact {
  min-width: 112px;
}

.advanced-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  width: fit-content;
  border: none;
  background: transparent;
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

.advanced-note {
  color: var(--ink-faint);
  font-weight: 600;
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

.filter-chip-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 14px;
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

/* --- Multi-select dropdown --- */
.multi-select {
  position: relative;
}

.multi-select-trigger {
  display: inline-flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-width: 120px;
  text-align: left;
  cursor: pointer;
  background: var(--bg-card);
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
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 6px;
}

.action-bar-actions {
  display: flex;
  gap: 10px;
}

/* --- Batch panel --- */
.batch-panel {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
  padding: 14px 20px;
  background: rgba(255, 255, 255, 0.92);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
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
  font-family: var(--font-sans);
  font-size: 15px;
  /* border-collapse: separate 让 sticky 列背景能正确盖住横向滚动的内容，
     避免 collapse 下 sticky 列透出相邻列文字 */
  border-collapse: separate;
  border-spacing: 0;
  background: var(--bg-card);
}

.rebate-table thead {
  position: sticky;
  top: 0;
  z-index: 10;
}

.header-group-row th {
  padding: 8px 12px;
  font-size: 12px;
  font-weight: 700;
  text-align: center;
  white-space: nowrap;
  color: var(--ink-strong);
  border-bottom: 1px solid var(--border-soft);
  background: #f8fafc;
  letter-spacing: 0;
  line-height: 1.4;
}

.header-group-row th[rowspan="2"] {
  vertical-align: middle;
  background: #f8fafc;
  text-align: left;
}

.header-sub-row th {
  padding: 6px 12px;
  font-size: 11px;
  font-weight: 600;
  text-align: left;
  white-space: nowrap;
  color: var(--ink-soft);
  border-bottom: 1px solid var(--border-soft);
  letter-spacing: 0;
  background: #f8fafc;
}

/* Group header colors */
.group-receivable {
  background: #f8fbff !important;
  color: #36527c !important;
}

.group-ratio {
  background: #f8fafc !important;
  color: #556274 !important;
}

.group-tax {
  background: #fff7f7 !important;
  color: #a14a4a !important;
}

.group-should {
  background: #f4fbf7 !important;
  color: #2f6b58 !important;
}

.group-returned {
  background: #f7f8fc !important;
  color: #5a6685 !important;
}

.group-unreturned {
  background: #fffaf4 !important;
  color: #9a6340 !important;
}

.group-check {
  background: #f8fafc !important;
  color: #475569 !important;
}

.group-plan {
  background: #f3f7ff !important;
  color: #315599 !important;
}

/* Sub-header colors matching groups */
.sub-receivable {
  background: #f8fbff !important;
  color: #36527c !important;
}

.sub-ratio {
  background: #f8fafc !important;
  color: #556274 !important;
}

.sub-tax {
  background: #fff7f7 !important;
  color: #a14a4a !important;
}

.sub-should {
  background: #f4fbf7 !important;
  color: #2f6b58 !important;
}

.sub-returned {
  background: #f7f8fc !important;
  color: #5a6685 !important;
}

.sub-unreturned {
  background: #fffaf4 !important;
  color: #9a6340 !important;
}

.sub-check {
  background: #f0f4f8 !important;
  color: #475569 !important;
}

.sub-plan {
  background: #f3f7ff !important;
  color: #315599 !important;
}

/* ── 数据行单元格：禁止任何列颜色，仅保持白底 ── */
.rebate-table tbody td {
  background: var(--bg-card) !important;
}

.rebate-table tbody tr.row-alt td {
  background: #fcfcfd !important;
}

.rebate-table tbody tr:hover td {
  background: #f7f8fa !important;
}

.rebate-table td {
  padding: 5px 10px;
  white-space: nowrap;
  border-bottom: 1px solid var(--border-soft);
  color: var(--ink-strong);
  font-size: 12px;
  line-height: 1.4;
  background: var(--bg-card);
}

/* 斑马行 & hover — 已由 tbody td 规则管理 */

/* ── 订单号列固定（水平 + 垂直滚动） ── */
.header-group-row .sticky-col {
  position: sticky !important;
  left: 0 !important;
  z-index: 20 !important;
  text-align: left;
  background: #f8fafc;
  box-shadow: 2px 0 4px rgba(0, 0, 0, 0.06);
}

.header-sub-row .sticky-col {
  position: sticky !important;
  left: 0 !important;
  z-index: 20 !important;
  background: #f8fafc;
  box-shadow: 2px 0 4px rgba(0, 0, 0, 0.06);
}

.rebate-table tbody .sticky-col {
  position: sticky !important;
  left: 0 !important;
  z-index: 5 !important;
  background: var(--bg-card) !important;
  box-shadow: 2px 0 4px rgba(0, 0, 0, 0.06);
}

.rebate-table tr.row-alt .sticky-col {
  background: #fcfcfd !important;
}

.rebate-table tr:hover .sticky-col {
  background: #f7f8fa !important;
}

@media (max-width: 1440px) {
  .filter-actions {
    width: 100%;
    margin-left: 0;
    justify-content: flex-end;
  }

  .action-bar {
    flex-direction: column;
    align-items: flex-start;
  }
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

.returnable-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 68px;
  min-height: 22px;
  padding: 2px 8px;
  border: 1px solid var(--border-soft);
  border-radius: 4px;
  font-size: 11px;
  font-weight: 800;
  background: var(--bg-card);
  color: var(--ink-faint);
}

.returnable-ready {
  color: #087c58;
  background: var(--success-soft);
  border-color: rgba(16, 185, 129, 0.2);
}

.returnable-waiting {
  color: #8a5a00;
  background: var(--warning-soft);
  border-color: rgba(245, 158, 11, 0.24);
}

.returnable-empty {
  color: var(--ink-faint);
  background: var(--bg-card);
}

.check-cell {
  text-align: center;
}

.check-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 26px;
  min-height: 24px;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 700;
}

.check-pass {
  color: #087c58;
  background: var(--success-soft);
}

.check-fail {
  color: #b42318;
  background: var(--danger-soft);
}

.check-empty {
  color: var(--ink-faint);
  background: #f8fafc;
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
