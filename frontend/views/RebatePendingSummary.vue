<template>
  <div class="rebate-pending-summary-page">
    <div v-if="!embedded" class="page-header">
      <h1 class="text-page-title">待返费明细</h1>
      <p class="text-body">按返还人汇总状态为待返的订单明细</p>
    </div>

    <div class="filter-bar">
      <div class="filter-group">
        <label>返还人</label>
        <input v-model="filters.rebateTarget" type="text" class="input input-sm input-compact" placeholder="返还人" />
      </div>
      <div class="filter-group">
        <label>订单号</label>
        <input v-model="filters.orderId" type="text" class="input input-sm" placeholder="订单号" />
      </div>
      <div class="filter-group">
        <label>客户姓名</label>
        <input v-model="filters.customerName" type="text" class="input input-sm input-compact" placeholder="客户姓名" />
      </div>
      <div class="filter-group">
        <label>航班编号</label>
        <input v-model="filters.flightId" type="text" class="input input-sm input-compact" placeholder="航班编号" />
      </div>
      <div class="filter-group">
        <label>航班名称</label>
        <input v-model="filters.productName" type="text" class="input input-sm input-compact" placeholder="航班名称" />
      </div>
      <div class="filter-actions">
        <button class="btn btn-primary btn-sm" @click="applyFilters">
          <Search :size="14" />
          查询
        </button>
        <button class="btn btn-secondary btn-sm" @click="resetFilters">重置</button>
        <FullscreenToggle target=".rebate-pending-summary-page .table-section" />
      </div>
    </div>

    <div class="action-bar">
      <span class="text-label">
        汇总 {{ groupedRows.length }} 人 / 待返订单 {{ items.length }} 条
      </span>
      <button class="btn btn-secondary btn-sm" @click="downloadCSV">
        <Download :size="14" />
        下载
      </button>
    </div>

    <div v-if="loading" class="loading-state">加载中...</div>
    <div v-else-if="!loading && items.length === 0 && loaded" class="empty-state">暂无待返费数据</div>

    <div v-else-if="items.length > 0" class="table-section">
      <div class="table-wrap">
        <table class="data-table pending-summary-table">
          <colgroup>
            <col style="min-width: 150px" />
            <col span="3" style="min-width: 110px" />
            <col span="3" style="min-width: 110px" />
            <col span="3" style="min-width: 110px" />
            <col span="3" style="min-width: 110px" />
            <col style="min-width: 150px" />
            <col style="min-width: 130px" />
            <col style="min-width: 130px" />
          </colgroup>
          <thead>
            <tr class="header-group-row">
              <th rowspan="2" class="sticky-col">返还人</th>
              <th colspan="3" class="group-header group-receivable">应收</th>
              <th colspan="3" class="group-header group-should">应返</th>
              <th colspan="3" class="group-header group-returned">已返</th>
              <th colspan="3" class="group-header group-unreturned">未返</th>
              <th class="group-header group-plan">本次拟返</th>
              <th class="group-header group-review">审核</th>
              <th class="group-header group-payment">打款</th>
            </tr>
            <tr class="header-sub-row">
              <th class="num sub-receivable">申购费</th>
              <th class="num sub-receivable">管理费实收</th>
              <th class="num sub-receivable">业绩报酬应收</th>
              <th class="num sub-should">申购费</th>
              <th class="num sub-should">管理费</th>
              <th class="num sub-should">业绩报酬</th>
              <th class="num sub-returned">申购费</th>
              <th class="num sub-returned">管理费</th>
              <th class="num sub-returned">业绩报酬</th>
              <th class="num sub-unreturned">申购费</th>
              <th class="num sub-unreturned">管理费</th>
              <th class="num sub-unreturned">业绩报酬</th>
              <th class="sub-plan">
                <button class="flow-btn" type="button" :disabled="flowLoading" @click="sendReview">
                  发送审核
                </button>
              </th>
              <th class="sub-review">
                <button class="flow-btn" type="button" :disabled="flowLoading" @click="sendPayment">
                  发送打款
                </button>
              </th>
              <th class="sub-payment">
                <button class="flow-btn" type="button" :disabled="flowLoading" @click="completePayment">
                  支付完成
                </button>
              </th>
            </tr>
          </thead>
          <tbody>
            <template v-for="group in pagedGroups" :key="group.rebate_target">
              <tr class="summary-row">
                <td class="sticky-col group-title-cell">
                  <button class="expand-btn" type="button" @click="toggleExpanded(group.rebate_target)">
                    <ChevronRight v-if="!expanded[group.rebate_target]" :size="14" />
                    <ChevronDown v-else :size="14" />
                  </button>
                  <span>{{ group.rebate_target }}</span>
                  <span class="order-count">{{ group.items.length }} 单</span>
                </td>
                <td>{{ fmtNum(group.receivable_subscribe) }}</td>
                <td>{{ fmtNum(group.receivable_management) }}</td>
                <td>{{ fmtNum(group.receivable_performance) }}</td>
                <td>{{ fmtNum(group.expected_subscribe) }}</td>
                <td>{{ fmtNum(group.expected_management) }}</td>
                <td>{{ fmtNum(group.expected_performance) }}</td>
                <td>{{ fmtNum(group.returned_subscribe) }}</td>
                <td>{{ fmtNum(group.returned_management) }}</td>
                <td>{{ fmtNum(group.returned_performance) }}</td>
                <td>{{ fmtNum(group.outstanding_subscribe) }}</td>
                <td>{{ fmtNum(group.outstanding_management) }}</td>
                <td>{{ fmtNum(group.outstanding_performance) }}</td>
                <td class="plan-cell">
                  <label class="plan-check">
                    <input
                      type="checkbox"
                      :checked="isGroupPlanned(group)"
                      @change="toggleGroupPlan(group, $event)"
                    />
                    <span>{{ isGroupPlanned(group) ? fmtNum(group.plan_total) : '0.00' }}</span>
                  </label>
                </td>
                <td class="plan-cell">
                  <label class="state-check">
                    <input type="checkbox" :checked="isGroupReviewed(group)" disabled />
                  </label>
                </td>
                <td class="plan-cell">
                  <label class="state-check">
                    <input type="checkbox" :checked="isGroupPaymentSent(group)" disabled />
                  </label>
                </td>
              </tr>
              <tr v-if="expanded[group.rebate_target]" class="detail-holder-row">
                <td colspan="16" class="detail-holder-cell">
                  <table class="detail-table">
                    <thead>
                      <tr>
                        <th>订单号</th>
                        <th>航班编号</th>
                        <th>航班名称</th>
                        <th>客户姓名</th>
                        <th class="num">未返-申购费</th>
                        <th class="num">未返-管理费</th>
                        <th class="num">未返-业绩报酬</th>
                        <th class="num">未返合计</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="item in group.items" :key="item.order_id">
                        <td>{{ item.order_id }}</td>
                        <td>{{ item.flight_id || '--' }}</td>
                        <td class="name-cell" :title="item.product_name">{{ truncate(item.product_name, 18) }}</td>
                        <td>{{ item.customer_name || '--' }}</td>
                        <td>{{ fmtNum(outstanding(item, 'subscribe')) }}</td>
                        <td>{{ fmtNum(outstanding(item, 'management')) }}</td>
                        <td>{{ fmtNum(outstanding(item, 'performance')) }}</td>
                        <td>{{ fmtNum(itemOutstandingTotal(item)) }}</td>
                      </tr>
                    </tbody>
                  </table>
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>

      <div class="pagination">
        <span class="text-label">共 {{ groupedRows.length }} 人 · 第 {{ page }} / {{ totalPages }} 页</span>
        <div class="pagination-controls">
          <button class="btn btn-secondary btn-sm" :disabled="page <= 1" @click="gotoPage(page - 1)">上一页</button>
          <button class="btn btn-secondary btn-sm" :disabled="page >= totalPages" @click="gotoPage(page + 1)">下一页</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ChevronDown, ChevronRight, Download, Search } from '@lucide/vue'
import FullscreenToggle from '../components/FullscreenToggle.vue'

defineProps({
  embedded: { type: Boolean, default: false },
})

const loading = ref(false)
const loaded = ref(false)
const flowLoading = ref(false)
const items = ref([])
const expanded = ref({})
const page = ref(1)
const pageSize = ref(20)

const filters = ref({
  rebateTarget: '',
  orderId: '',
  customerName: '',
  flightId: '',
  productName: '',
})

const groupedRows = computed(() => {
  const groups = new Map()
  for (const item of items.value) {
    const key = item.rebate_target || '未填写返还人'
    if (!groups.has(key)) {
      groups.set(key, {
        rebate_target: key,
        items: [],
        receivable_subscribe: 0,
        receivable_management: 0,
        receivable_performance: 0,
        expected_subscribe: 0,
        expected_management: 0,
        expected_performance: 0,
        returned_subscribe: 0,
        returned_management: 0,
        returned_performance: 0,
        outstanding_subscribe: 0,
        outstanding_management: 0,
        outstanding_performance: 0,
        plan_total: 0,
      })
    }
    const group = groups.get(key)
    group.items.push(item)
    group.receivable_subscribe += receivable(item, 'subscribe')
    group.receivable_management += receivable(item, 'management')
    group.receivable_performance += receivable(item, 'performance')
    group.expected_subscribe += expected(item, 'subscribe')
    group.expected_management += expected(item, 'management')
    group.expected_performance += expected(item, 'performance')
    group.returned_subscribe += returned(item, 'subscribe')
    group.returned_management += returned(item, 'management')
    group.returned_performance += returned(item, 'performance')
    group.outstanding_subscribe += outstanding(item, 'subscribe')
    group.outstanding_management += outstanding(item, 'management')
    group.outstanding_performance += outstanding(item, 'performance')
    group.plan_total += Math.max(0, outstanding(item, 'subscribe')) + Math.max(0, outstanding(item, 'management')) + Math.max(0, outstanding(item, 'performance'))
  }
  return Array.from(groups.values()).sort((a, b) => b.plan_total - a.plan_total)
})

const totalPages = computed(() => Math.max(1, Math.ceil(groupedRows.value.length / pageSize.value)))
const pagedGroups = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return groupedRows.value.slice(start, start + pageSize.value)
})

onMounted(fetchData)

function num(v) {
  const n = Number(v)
  return Number.isFinite(n) ? n : 0
}

function fmtNum(val) {
  if (val == null || val === '') return '--'
  const n = Number(val)
  if (!Number.isFinite(n)) return '--'
  return n.toFixed(2)
}

function truncate(str, len) {
  if (!str) return '--'
  return str.length > len ? str.slice(0, len) + '...' : str
}

function receivable(item, type) {
  if (type === 'subscribe') return num(item.principal) * num(item.subscribe_fee_rate)
  if (type === 'management') return num(item.management_fee_received)
  if (type === 'performance') return num(item.performance_fee_receivable)
  return 0
}

function calcExpected(item, type) {
  if (type === 'subscribe') {
    return num(item.principal) * num(item.subscribe_fee_rate) * num(item.subscribe_fee_ratio) * (1 - num(item.tax_subscribe_ratio))
  }
  if (type === 'management') {
    return num(item.management_fee_received) * num(item.management_fee_ratio) * (1 - num(item.tax_management_ratio))
  }
  if (type === 'performance') {
    return num(item.performance_fee_receivable) * num(item.performance_fee_ratio) * (1 - num(item.tax_performance_ratio))
  }
  return 0
}

function expected(item, type) {
  const field = {
    subscribe: 'expected_subscribe',
    management: 'expected_management',
    performance: 'expected_performance',
  }[type]
  return item[field] == null ? calcExpected(item, type) : num(item[field])
}

function returned(item, type) {
  const field = {
    subscribe: 'returned_subscribe',
    management: 'returned_management',
    performance: 'returned_performance',
  }[type]
  return num(item[field])
}

function outstanding(item, type) {
  const field = {
    subscribe: 'outstanding_subscribe',
    management: 'outstanding_management',
    performance: 'outstanding_performance',
  }[type]
  return item[field] == null ? expected(item, type) - returned(item, type) : num(item[field])
}

function itemOutstandingTotal(item) {
  return outstanding(item, 'subscribe') + outstanding(item, 'management') + outstanding(item, 'performance')
}

function normalizeItem(item) {
  const principal =
    item.principal != null && item.principal !== 0
      ? item.principal
      : item.subscribe_amount != null && item.subscribe_amount !== 0
        ? item.subscribe_amount
        : item.amount != null
          ? item.amount
          : 0
  return { ...item, principal }
}

async function fetchData() {
  loading.value = true
  page.value = 1
  try {
    const params = new URLSearchParams()
    const f = filters.value
    params.set('is_returnable', '待返')
    if (f.rebateTarget) params.set('rebate_target', f.rebateTarget)
    if (f.orderId) params.set('order_id', f.orderId)
    if (f.customerName) params.set('customer_name', f.customerName)
    if (f.flightId) params.set('flight_id', f.flightId)
    if (f.productName) params.set('product_name', f.productName)
    const res = await fetch(`/api/rebate/pending?${params}`)
    if (!res.ok) throw new Error('加载失败')
    const data = await res.json()
    items.value = (data.items || []).map(normalizeItem)
  } catch {
    items.value = []
  } finally {
    loading.value = false
    loaded.value = true
  }
}

function applyFilters() {
  fetchData()
}

function resetFilters() {
  filters.value = {
    rebateTarget: '',
    orderId: '',
    customerName: '',
    flightId: '',
    productName: '',
  }
  fetchData()
}

function gotoPage(p) {
  page.value = Math.min(Math.max(1, p), totalPages.value)
}

function toggleExpanded(key) {
  expanded.value = {
    ...expanded.value,
    [key]: !expanded.value[key],
  }
}

function isItemPlanned(item) {
  return !!item.plan_subscribe && !!item.plan_management && !!item.plan_performance
}

function isGroupPlanned(group) {
  return group.items.length > 0 && group.items.every(isItemPlanned)
}

function isItemReviewed(item) {
  return !!item.review_sent
}

function isGroupReviewed(group) {
  return group.items.length > 0 && group.items.every(isItemReviewed)
}

function isItemPaymentSent(item) {
  return !!item.payment_sent
}

function isGroupPaymentSent(group) {
  return group.items.length > 0 && group.items.every(isItemPaymentSent)
}

function selectedPlannedItems() {
  return items.value.filter(isItemPlanned)
}

function selectedReviewedItems() {
  return items.value.filter(item => isItemPlanned(item) && isItemReviewed(item))
}

function selectedPaymentItems() {
  return items.value.filter(item => isItemPlanned(item) && isItemReviewed(item) && isItemPaymentSent(item))
}

function flowPayload(sourceItems) {
  return {
    rebate_target: sourceItems.length === 1 ? sourceItems[0].rebate_target : '',
    items: sourceItems.map(item => ({
      order_id: item.order_id,
      flight_id: item.flight_id || '',
      product_name: item.product_name || '',
      customer_name: item.customer_name || '',
      rebate_target: item.rebate_target || '',
      outstanding_subscribe: Math.max(0, outstanding(item, 'subscribe')),
      outstanding_management: Math.max(0, outstanding(item, 'management')),
      outstanding_performance: Math.max(0, outstanding(item, 'performance')),
    })),
  }
}

async function postFlow(url, sourceItems) {
  if (sourceItems.length === 0 || flowLoading.value) return null
  flowLoading.value = true
  try {
    const res = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(flowPayload(sourceItems)),
    })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(data.error || '操作失败')
    return data
  } finally {
    flowLoading.value = false
  }
}

async function sendReview() {
  const targets = selectedPlannedItems()
  if (targets.length === 0) return
  try {
    await postFlow('/api/rebate/pending/send-review', targets)
    for (const item of targets) item.review_sent = 1
  } catch (e) {
    alert(e.message || '发送审核失败')
  }
}

async function sendPayment() {
  const targets = selectedReviewedItems()
  if (targets.length === 0) return
  try {
    await postFlow('/api/rebate/pending/send-payment', targets)
    for (const item of targets) item.payment_sent = 1
  } catch (e) {
    alert(e.message || '发送打款失败')
  }
}

async function completePayment() {
  const targets = selectedPaymentItems()
  if (targets.length === 0) return
  try {
    await postFlow('/api/rebate/pending/complete-payment', targets)
    await fetchData()
  } catch (e) {
    alert(e.message || '支付完成失败')
  }
}

async function toggleGroupPlan(group, event) {
  const checked = event.target.checked
  for (const item of group.items) {
    item.plan_subscribe = checked ? 1 : 0
    item.plan_management = checked ? 1 : 0
    item.plan_performance = checked ? 1 : 0
  }
  await Promise.all(
    group.items.map(item =>
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
  )
}

function downloadCSV() {
  if (groupedRows.value.length === 0) return
  const headers = [
    '返还人',
    '应收-申购费', '应收-管理费实收', '应收-业绩报酬应收',
    '应返-申购费', '应返-管理费', '应返-业绩报酬',
    '已返-申购费', '已返-管理费', '已返-业绩报酬',
    '未返-申购费', '未返-管理费', '未返-业绩报酬',
    '本次拟返合计',
    '订单数',
  ]
  const rows = groupedRows.value.map(group => [
    group.rebate_target,
    fmtNum(group.receivable_subscribe),
    fmtNum(group.receivable_management),
    fmtNum(group.receivable_performance),
    fmtNum(group.expected_subscribe),
    fmtNum(group.expected_management),
    fmtNum(group.expected_performance),
    fmtNum(group.returned_subscribe),
    fmtNum(group.returned_management),
    fmtNum(group.returned_performance),
    fmtNum(group.outstanding_subscribe),
    fmtNum(group.outstanding_management),
    fmtNum(group.outstanding_performance),
    isGroupPlanned(group) ? fmtNum(group.plan_total) : '0.00',
    group.items.length,
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
  link.download = `待返费明细_${new Date().toISOString().slice(0, 10)}.csv`
  link.click()
  URL.revokeObjectURL(url)
}
</script>

<style scoped>
:deep(.workbench-main) {
  max-width: none;
}

.rebate-pending-summary-page {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  font-size: 12px;
}

.rebate-pending-summary-page > .page-header,
.rebate-pending-summary-page > .filter-bar,
.rebate-pending-summary-page > .action-bar {
  flex-shrink: 0;
}

.filter-bar {
  margin-bottom: 8px;
}

.input-compact {
  min-width: 102px;
}

.action-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 8px;
}

.table-wrap {
  overflow-x: auto;
}

.pending-summary-table {
  min-width: 1680px;
  border-collapse: separate;
  border-spacing: 0;
  background: var(--bg-card);
  font-size: 12px;
}

.pending-summary-table thead {
  position: sticky;
  top: 0;
  z-index: 10;
}

.header-group-row th,
.header-sub-row th {
  white-space: nowrap;
  border-bottom: 1px solid var(--border-soft);
  background: #f8fafc;
}

.header-group-row th {
  padding: 8px 12px;
  font-size: 12px;
  font-weight: 700;
  color: var(--ink-strong);
  text-align: center;
}

.header-group-row th[rowspan="2"] {
  text-align: left;
}

.header-sub-row th {
  padding: 6px 12px;
  font-size: 11px;
  font-weight: 600;
  color: var(--ink-soft);
  text-align: left;
}

.group-receivable,
.sub-receivable {
  background: #f8fbff !important;
  color: #36527c !important;
}

.group-should,
.sub-should {
  background: #f4fbf7 !important;
  color: #2f6b58 !important;
}

.group-returned,
.sub-returned {
  background: #f7f8fc !important;
  color: #5a6685 !important;
}

.group-unreturned,
.sub-unreturned {
  background: #fffaf4 !important;
  color: #9a6340 !important;
}

.group-plan {
  background: #f3f7ff !important;
  color: #315599 !important;
}

.group-review,
.sub-review {
  background: #f4fbf7 !important;
  color: #2f6b58 !important;
}

.group-payment,
.sub-payment {
  background: #fffaf4 !important;
  color: #9a6340 !important;
}

.pending-summary-table td {
  padding: 6px 10px;
  white-space: nowrap;
  border-bottom: 1px solid var(--border-soft);
  color: var(--ink-strong);
  background: var(--bg-card);
}

.summary-row td {
  font-weight: 700;
  background: #fff !important;
}

.summary-row:hover td {
  background: #f7f8fa !important;
}

.sticky-col {
  position: sticky !important;
  left: 0 !important;
  z-index: 5 !important;
  background: inherit !important;
  box-shadow: 2px 0 4px rgba(0, 0, 0, 0.06);
}

thead .sticky-col {
  z-index: 20 !important;
  background: #f8fafc !important;
}

.group-title-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.expand-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  background: var(--bg-card);
  color: var(--ink-soft);
  cursor: pointer;
}

.order-count {
  color: var(--ink-faint);
  font-weight: 600;
}

.plan-cell {
  text-align: left;
}

.plan-check {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 120px;
  font-weight: 700;
}

.plan-check input {
  width: 16px;
  height: 16px;
  accent-color: var(--brand);
}

.state-check {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 44px;
}

.state-check input {
  width: 16px;
  height: 16px;
  accent-color: var(--brand);
}

.flow-btn {
  min-width: 88px;
  height: 26px;
  padding: 0 10px;
  border: 1px solid rgba(31, 58, 138, 0.16);
  border-radius: var(--radius);
  background: var(--bg-card);
  color: var(--brand);
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
}

.flow-btn:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.detail-holder-cell {
  padding: 8px 10px 12px 42px !important;
  background: #fbfcfe !important;
}

.detail-table {
  width: 100%;
  border-collapse: separate;
  border-spacing: 0;
  border: 1px solid var(--border-soft);
  background: #fff;
}

.detail-table th,
.detail-table td {
  padding: 6px 10px;
  border-bottom: 1px solid var(--border-soft);
  font-size: 12px;
  white-space: nowrap;
}

.detail-table th {
  background: #f8fafc;
  color: var(--ink-soft);
  font-weight: 700;
  text-align: left;
}

.detail-table tr:last-child td {
  border-bottom: none;
}

.name-cell {
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.num {
  text-align: right;
}

.pagination {
  flex-shrink: 0;
}

@media (max-width: 1440px) {
  .action-bar {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
