<template>
  <div class="rebate-completed-page">
    <div v-if="!embedded" class="page-header">
      <h1 class="text-page-title">已返费分析</h1>
      <p class="text-body">查看和管理已完成返费的订单记录</p>
    </div>

    <!-- Filters -->
    <div class="filter-bar">
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
        <label>费用类别</label>
        <select v-model="filters.expenseCategory" class="input input-sm input-select-compact">
          <option value="">全部</option>
          <option value="申购费">申购费</option>
          <option value="管理费">管理费</option>
          <option value="业绩报酬">业绩报酬</option>
        </select>
      </div>
      <div class="filter-group">
        <label>来源</label>
        <select v-model="filters.source" class="input input-sm input-select-compact">
          <option value="">全部</option>
          <option value="manual">手工录入</option>
          <option value="upload">批量上传</option>
          <option value="auto_sync">自动同步</option>
        </select>
      </div>
      <div class="filter-group">
        <label>航班编号</label>
        <input
          v-model="filters.flightId"
          type="text"
          class="input input-sm input-compact"
          placeholder="请输入航班编号"
        />
      </div>
      <div class="filter-group">
        <label>客户姓名</label>
        <input
          v-model="filters.customerName"
          type="text"
          class="input input-sm input-compact"
          placeholder="请输入客户姓名"
        />
      </div>
      <div class="filter-group">
        <label>返还人</label>
        <input
          v-model="filters.rebateTarget"
          type="text"
          class="input input-sm input-compact"
          placeholder="请输入返还人"
        />
      </div>
      <div class="filter-actions">
        <button class="btn btn-primary btn-sm" @click="fetchData">
          <Search :size="14" />
          查询
        </button>
        <button class="btn btn-secondary btn-sm" @click="resetFilters">重置</button>
        <FullscreenToggle target=".rebate-completed-page .table-section" />
      </div>
    </div>

    <!-- Action bar -->
    <div class="action-bar">
      <button class="btn btn-primary btn-sm" @click="openAddModal">
        <Plus :size="14" />
        新增记录
      </button>
      <button class="btn btn-secondary btn-sm" @click="triggerFileUpload">
        <Upload :size="14" />
        批量上传
      </button>
      <button class="btn btn-secondary btn-sm" @click="downloadCSV">
        <Download :size="14" />
        下载
      </button>
      <button class="btn btn-secondary btn-sm" @click="downloadTemplate">
        <Download :size="14" />
        下载模板
      </button>
      <input
        ref="fileInputRef"
        type="file"
        accept=".xlsx,.csv"
        class="file-input-hidden"
        @change="handleFileSelect"
      />
    </div>

    <!-- Loading -->
    <div v-if="loading" class="loading-state">加载中...</div>

    <!-- Empty -->
    <div v-else-if="!loading && items.length === 0 && loaded" class="empty-state">
      暂无已返费数据
    </div>

    <!-- Data table -->
    <div v-else-if="items.length > 0" class="table-section">
      <div class="table-wrap">
      <div v-if="validationSummary.count > 0" class="validation-banner">
        检测到 {{ validationSummary.count }} 条已返记录存在校验冲突，共 {{ validationSummary.issueCount }} 项，请优先处理。
      </div>
      <table class="data-table completed-table">
        <colgroup>
          <col style="min-width: 120px" /><!-- 订单号 -->
          <col style="min-width: 100px" /><!-- 航班编号 -->
          <col style="min-width: 140px" /><!-- 产品名称 -->
          <col style="min-width: 90px" /><!-- 客户姓名 -->
          <col style="min-width: 90px" /><!-- 渠道/直客 -->
          <col style="min-width: 100px" /><!-- 本金 -->
          <col style="min-width: 90px" /><!-- 保证金比例 -->
          <col style="min-width: 90px" /><!-- 业务类型 -->
          <col style="min-width: 100px" /><!-- 认购日 -->
          <col style="min-width: 90px" /><!-- 订单状态 -->
          <col style="min-width: 90px" /><!-- 返还人 -->
          <col span="3" style="min-width: 90px" /><!-- 渠道返还比例 x3 -->
          <col span="6" style="min-width: 90px" /><!-- 支出流水明细 x6 -->
          <col style="min-width: 80px" /><!-- 来源 -->
          <col style="min-width: 180px" /><!-- 校验 -->
          <col style="min-width: 70px" /><!-- 操作 -->
        </colgroup>
        <thead>
          <tr class="header-group-row">
            <th rowspan="2" class="sticky-col">订单号</th>
            <th rowspan="2">航班编号</th>
            <th rowspan="2">产品名称</th>
            <th rowspan="2">客户姓名</th>
            <th rowspan="2">渠道/直客</th>
            <th rowspan="2" class="num">本金</th>
            <th rowspan="2" class="num">保证金比例</th>
            <th rowspan="2">业务类型</th>
            <th rowspan="2">认购日</th>
            <th rowspan="2">订单状态</th>
            <th rowspan="2">返还人</th>
            <th colspan="3" class="group-header group-channel-ratio">渠道返还比例</th>
            <th colspan="6" class="group-header group-expense">支出流水明细</th>
            <th rowspan="2">来源</th>
            <th rowspan="2">校验</th>
            <th rowspan="2">操作</th>
          </tr>
          <tr class="header-sub-row">
            <!-- 渠道返还比例 -->
            <th class="num sub-channel-ratio">申购费</th>
            <th class="num sub-channel-ratio">管理费</th>
            <th class="num sub-channel-ratio">业绩报酬</th>
            <!-- 支出流水明细 -->
            <th class="sub-expense">类别</th>
            <th class="num sub-expense">金额</th>
            <th class="sub-expense">支付时间</th>
            <th class="sub-expense">年</th>
            <th class="sub-expense">月</th>
            <th class="sub-expense">日</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(item, idx) in items" :key="item.id" :class="{ 'row-alt': idx % 2 === 1 }">
            <td class="sticky-col">{{ item.order_id }}</td>
            <td>{{ item.flight_id || '--' }}</td>
            <td class="name-cell" :title="item.product_name">{{ truncate(item.product_name, 12) }}</td>
            <td>{{ item.customer_name || '--' }}</td>
            <td>{{ item.channel_or_direct || '--' }}</td>
            <td class="num">{{ fmtNum(item.principal) }}</td>
            <td class="num">{{ fmtPct(item.margin_ratio) }}</td>
            <td>{{ item.business_type || '--' }}</td>
            <td>{{ item.subscribe_date || '--' }}</td>
            <td>{{ item.order_status || '--' }}</td>
            <td>{{ item.rebate_target || '--' }}</td>
            <!-- 渠道返还比例 -->
            <td class="num">{{ fmtPct(item.channel_subscribe_ratio) }}</td>
            <td class="num">{{ fmtPct(item.channel_management_ratio) }}</td>
            <td class="num">{{ fmtPct(item.channel_performance_ratio) }}</td>
            <!-- 支出流水明细 -->
            <td>{{ item.expense_category || '--' }}</td>
            <td class="num">{{ fmtNum(item.expense_amount) }}</td>
            <td>{{ item.payment_time || '--' }}</td>
            <td>{{ item.payment_year || '--' }}</td>
            <td>{{ item.payment_month || '--' }}</td>
            <td>{{ item.payment_day || '--' }}</td>
            <!-- 来源 -->
            <td>
              <span class="source-pill" :class="sourceClass(item.source)">
                {{ sourceLabel(item.source) }}
              </span>
            </td>
            <td>
              <span v-if="item.validation_conflicts?.length" class="validation-pill validation-error" :title="conflictMessage(item.validation_conflicts)">
                {{ item.validation_conflicts.length }}项冲突
              </span>
              <span
                v-else-if="ignoredConflictCount(item) > 0"
                class="validation-pill validation-muted"
                :title="ignoredConflictMessage(item)"
              >
                已忽略{{ ignoredConflictCount(item) }}项
              </span>
              <span v-else class="validation-pill validation-ok">通过</span>
            </td>
            <!-- 操作 -->
            <td>
              <button class="btn-icon btn-icon-danger" title="删除" @click="confirmDelete(item)">
                <Trash2 :size="14" />
              </button>
            </td>
          </tr>
        </tbody>
      </table>
      </div>
    </div>

    <!-- Add record modal -->
    <Teleport to="body">
      <div v-if="addModal.visible" class="modal-overlay" @click.self="addModal.visible = false">
        <div class="modal-card">
          <div class="modal-header">
            <h3 class="modal-title">新增已返费记录</h3>
            <button class="modal-close" @click="addModal.visible = false">
              <X :size="18" />
            </button>
          </div>
          <div class="modal-body">
            <div class="form-grid">
              <div class="form-field">
                <label>订单号</label>
                <input v-model="addForm.order_id" type="text" class="input" placeholder="请输入订单号" />
              </div>
              <div class="form-field">
                <label>航班编号</label>
                <input v-model="addForm.flight_id" type="text" class="input" placeholder="请输入航班编号" />
              </div>
              <div class="form-field">
                <label>产品名称</label>
                <input v-model="addForm.product_name" type="text" class="input" placeholder="请输入产品名称" />
              </div>
              <div class="form-field">
                <label>客户姓名</label>
                <input v-model="addForm.customer_name" type="text" class="input" placeholder="请输入客户姓名" />
              </div>
              <div class="form-field">
                <label>渠道/直客</label>
                <input v-model="addForm.channel_or_direct" type="text" class="input" placeholder="请输入渠道/直客" />
              </div>
              <div class="form-field">
                <label>本金</label>
                <input v-model.number="addForm.principal" type="number" class="input" placeholder="请输入本金" />
              </div>
              <div class="form-field">
                <label>保证金比例</label>
                <input v-model.number="addForm.margin_ratio" type="number" step="0.01" class="input" placeholder="例如 0.3" />
              </div>
              <div class="form-field">
                <label>业务类型</label>
                <input v-model="addForm.business_type" type="text" class="input" placeholder="请输入业务类型" />
              </div>
              <div class="form-field">
                <label>认购日</label>
                <input v-model="addForm.subscribe_date" type="date" class="input" />
              </div>
              <div class="form-field">
                <label>订单状态</label>
                <input v-model="addForm.order_status" type="text" class="input" placeholder="请输入订单状态" />
              </div>
              <div class="form-field">
                <label>返还人</label>
                <input v-model="addForm.rebate_target" type="text" class="input" placeholder="请输入返还人" />
              </div>
              <div class="form-field">&nbsp;</div>
            </div>

            <div class="form-section-title">渠道返还比例</div>
            <div class="form-grid form-grid-3">
              <div class="form-field">
                <label>申购费</label>
                <input v-model.number="addForm.channel_subscribe_ratio" type="number" step="0.01" class="input" placeholder="例如 0.5" />
              </div>
              <div class="form-field">
                <label>管理费</label>
                <input v-model.number="addForm.channel_management_ratio" type="number" step="0.01" class="input" placeholder="例如 0.3" />
              </div>
              <div class="form-field">
                <label>业绩报酬</label>
                <input v-model.number="addForm.channel_performance_ratio" type="number" step="0.01" class="input" placeholder="例如 0.2" />
              </div>
            </div>

            <div class="form-section-title">支出流水明细</div>
            <div class="form-grid">
              <div class="form-field">
                <label>类别</label>
                <select v-model="addForm.expense_category" class="input">
                  <option value="">请选择</option>
                  <option value="申购费">申购费</option>
                  <option value="管理费">管理费</option>
                  <option value="业绩报酬">业绩报酬</option>
                </select>
              </div>
              <div class="form-field">
                <label>金额</label>
                <input v-model.number="addForm.expense_amount" type="number" step="0.01" class="input" placeholder="请输入金额" />
              </div>
              <div class="form-field form-field-full">
                <label>支付时间</label>
                <input v-model="addForm.payment_time" type="date" class="input" @change="onPaymentTimeChange" />
              </div>
              <div class="form-field">
                <label>年</label>
                <input v-model="addForm.payment_year" type="text" class="input" readonly />
              </div>
              <div class="form-field">
                <label>月</label>
                <input v-model="addForm.payment_month" type="text" class="input" readonly />
              </div>
              <div class="form-field">
                <label>日</label>
                <input v-model="addForm.payment_day" type="text" class="input" readonly />
              </div>
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary btn-sm" @click="addModal.visible = false">取消</button>
            <button class="btn btn-primary btn-sm" :disabled="addModal.submitting" @click="submitAddForm">
              {{ addModal.submitting ? '提交中...' : '提交' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Batch upload preview modal -->
    <Teleport to="body">
      <div v-if="uploadModal.visible" class="modal-overlay" @click.self="uploadModal.visible = false">
        <div class="modal-card modal-card-lg">
          <div class="modal-header">
            <h3 class="modal-title">批量上传预览</h3>
            <button class="modal-close" @click="cancelUpload">
              <X :size="18" />
            </button>
          </div>
          <div class="modal-body">
            <p class="upload-info">
              已解析 <strong>{{ uploadModal.records.length }}</strong> 条记录，请确认后上传。
            </p>
            <div class="table-wrap upload-preview-table">
              <table class="data-table">
                <thead>
                  <tr>
                    <th>订单号</th>
                    <th>航班编号</th>
                    <th>产品名称</th>
                    <th>客户姓名</th>
                    <th>渠道/直客</th>
                    <th>本金</th>
                    <th>费用类别</th>
                    <th>金额</th>
                    <th>支付时间</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(row, i) in uploadModal.records.slice(0, 50)" :key="i">
                    <td>{{ row.order_id || '--' }}</td>
                    <td>{{ row.flight_id || '--' }}</td>
                    <td>{{ row.product_name || '--' }}</td>
                    <td>{{ row.customer_name || '--' }}</td>
                    <td>{{ row.channel_or_direct || '--' }}</td>
                    <td>{{ row.principal ?? '--' }}</td>
                    <td>{{ row.expense_category || '--' }}</td>
                    <td>{{ row.expense_amount ?? '--' }}</td>
                    <td>{{ row.payment_time || '--' }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <p v-if="uploadModal.records.length > 50" class="upload-truncated">
              仅显示前 50 条记录，共 {{ uploadModal.records.length }} 条。
            </p>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary btn-sm" @click="cancelUpload">取消</button>
            <button class="btn btn-primary btn-sm" :disabled="uploadModal.submitting" @click="submitBatchUpload">
              {{ uploadModal.submitting ? '上传中...' : '确认上传' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Delete confirm dialog -->
    <Teleport to="body">
      <div v-if="deleteDialog.visible" class="modal-overlay" @click.self="deleteDialog.visible = false">
        <div class="dialog-box">
          <h3 class="dialog-title">确认删除</h3>
          <p class="dialog-body">确认删除订单 <strong>{{ deleteDialog.item?.order_id }}</strong> 的记录？此操作不可撤销。</p>
          <div class="dialog-actions">
            <button class="btn btn-secondary btn-sm" @click="deleteDialog.visible = false">取消</button>
            <button class="btn btn-danger btn-sm" @click="executeDelete">删除</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { Search, Plus, Upload, Download, Trash2, X } from '@lucide/vue'
import FullscreenToggle from '../components/FullscreenToggle.vue'

defineProps({
  embedded: { type: Boolean, default: false },
})

// --- Reactive state ---
const loading = ref(false)
const loaded = ref(false)
const items = ref([])
const fileInputRef = ref(null)

const filters = reactive({
  orderId: '',
  flightId: '',
  customerName: '',
  rebateTarget: '',
  expenseCategory: '',
  source: '',
})

// --- Add modal ---
const addModal = reactive({
  visible: false,
  submitting: false,
})

const autofillState = reactive({
  applying: false,
})

const validationSummary = reactive({
  count: 0,
  issueCount: 0,
})

function createEmptyForm() {
  return {
    order_id: '',
    flight_id: '',
    product_name: '',
    customer_name: '',
    channel_or_direct: '',
    principal: null,
    margin_ratio: null,
    business_type: '',
    subscribe_date: '',
    order_status: '',
    rebate_target: '',
    channel_subscribe_ratio: null,
    channel_management_ratio: null,
    channel_performance_ratio: null,
    expense_category: '',
    expense_amount: null,
    payment_time: '',
    payment_year: '',
    payment_month: '',
    payment_day: '',
    ignored_conflicts: [],
  }
}

const addForm = reactive(createEmptyForm())

// --- Upload modal ---
const uploadModal = reactive({
  visible: false,
  submitting: false,
  records: [],
})

// --- Delete dialog ---
const deleteDialog = reactive({
  visible: false,
  item: null,
})

// --- Lifecycle ---
onMounted(() => {
  fetchData()
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

// --- Source display ---
function sourceLabel(source) {
  const map = {
    manual: '手工录入',
    upload: '批量上传',
    auto_sync: '自动同步',
    auto_pending: '自动同步',
  }
  return map[source] || source || '--'
}

function sourceClass(source) {
  if (source === 'manual') return 'source-manual'
  if (source === 'upload') return 'source-upload'
  if (source === 'auto_sync' || source === 'auto_pending') return 'source-auto'
  return ''
}

// --- Data fetching ---
async function fetchData() {
  loading.value = true
  try {
    const params = new URLSearchParams()
    if (filters.orderId) params.set('order_id', filters.orderId)
    if (filters.flightId) params.set('flight_id', filters.flightId)
    if (filters.customerName) params.set('customer_name', filters.customerName)
    if (filters.rebateTarget) params.set('rebate_target', filters.rebateTarget)
    if (filters.expenseCategory) params.set('expense_category', filters.expenseCategory)
    if (filters.source) params.set('source', filters.source)

    const qs = params.toString()
    const url = qs ? `/api/rebate/completed?${qs}` : '/api/rebate/completed'
    const res = await fetch(url)
    if (!res.ok) throw new Error('加载失败')
    const data = await res.json()
    items.value = await validateLoadedItems(data.items || [])
  } catch {
    items.value = []
    validationSummary.count = 0
    validationSummary.issueCount = 0
  } finally {
    loading.value = false
    loaded.value = true
  }
}

function resetFilters() {
  filters.orderId = ''
  filters.flightId = ''
  filters.customerName = ''
  filters.rebateTarget = ''
  filters.expenseCategory = ''
  filters.source = ''
  fetchData()
}

// --- Add record ---
function openAddModal() {
  Object.assign(addForm, createEmptyForm())
  addModal.submitting = false
  addModal.visible = true
}

function onPaymentTimeChange() {
  if (addForm.payment_time) {
    const parts = addForm.payment_time.split('-')
    addForm.payment_year = parts[0] || ''
    addForm.payment_month = parts[1] || ''
    addForm.payment_day = parts[2] || ''
  } else {
    addForm.payment_year = ''
    addForm.payment_month = ''
    addForm.payment_day = ''
  }
}

function toAssistPayload(form) {
  return {
    order_id: form.order_id || '',
    flight_id: form.flight_id || '',
    product_name: form.product_name || '',
    customer_name: form.customer_name || '',
    channel_or_direct: form.channel_or_direct || '',
    principal: form.principal === '' || form.principal == null ? null : Number(form.principal),
    margin_ratio: form.margin_ratio === '' || form.margin_ratio == null ? null : Number(form.margin_ratio),
    subscribe_date: form.subscribe_date || '',
    order_status: form.order_status || '',
    rebate_target: form.rebate_target || '',
    channel_subscribe_ratio: form.channel_subscribe_ratio === '' || form.channel_subscribe_ratio == null ? null : Number(form.channel_subscribe_ratio),
    channel_management_ratio: form.channel_management_ratio === '' || form.channel_management_ratio == null ? null : Number(form.channel_management_ratio),
    channel_performance_ratio: form.channel_performance_ratio === '' || form.channel_performance_ratio == null ? null : Number(form.channel_performance_ratio),
    ignored_conflicts: Array.isArray(form.ignored_conflicts) ? [...new Set(form.ignored_conflicts.filter(Boolean))] : [],
  }
}

function applyAutofillToForm(form, autofill) {
  for (const [key, value] of Object.entries(autofill || {})) {
    if (form[key] == null || form[key] === '') {
      form[key] = value
    }
  }
}

const CONFLICT_FIELD_LABELS = {
  order_id: '订单号',
  flight_id: '航班编号',
  product_name: '产品名称',
  customer_name: '客户姓名',
  rebate_target: '返还人',
  channel_or_direct: '渠道/直客',
  subscribe_date: '认购日',
  order_status: '订单状态',
  principal: '本金',
  margin_ratio: '保证金比例',
  product_ref: '产品表匹配',
  'product_name(product_table)': '产品名称',
}

function conflictFieldLabel(field) {
  return CONFLICT_FIELD_LABELS[field] || field
}

function conflictExpectedLabel(field) {
  if (field === 'product_name(product_table)' || field === 'margin_ratio' || field === 'product_ref') {
    return '产品表'
  }
  return '交易表'
}

function conflictMessage(conflicts) {
  return conflicts
    .map(item => `${conflictFieldLabel(item.field)}: 当前=${item.current}，${conflictExpectedLabel(item.field)}=${item.expected}`)
    .join('\n')
}

function ignoredConflictCount(item) {
  return Array.isArray(item?.ignored_conflicts) ? item.ignored_conflicts.length : 0
}

function ignoredConflictMessage(item) {
  const entries = Array.isArray(item?.ignored_conflicts) ? item.ignored_conflicts : []
  if (!entries.length) return '无已忽略冲突'
  return entries
    .map((entry) => {
      const [field, ...rest] = String(entry).split('|')
      const expected = rest.join('|')
      return expected ? `${conflictFieldLabel(field)}: ${conflictExpectedLabel(field)}=${expected}` : conflictFieldLabel(field)
    })
    .join('\n')
}

async function assistCompletedRecordLegacy(form, { interactive = false } = {}) {
  const payload = toAssistPayload(form)
  const meaningfulKeys = ['order_id', 'flight_id', 'product_name', 'customer_name', 'principal', 'rebate_target']
  if (!meaningfulKeys.some(key => payload[key] !== '' && payload[key] != null)) {
    return { allowed: true, conflicts: [] }
  }

  try {
    autofillState.applying = true
    const res = await fetch('/api/rebate/completed/assist', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '自动识别失败')
    applyAutofillToForm(form, data.autofill)
    if (data.conflicts?.length) {
      if (!interactive) return { allowed: false, conflicts: data.conflicts }
      return window.confirm(`检测到与交易表不一致的字段：\n${conflictMessage(data.conflicts)}\n\n点击“确定”忽略并继续，点击“取消”返回修改。`)
    }
    return true
  } catch (error) {
    if (interactive) {
      alert(error.message || '自动识别失败')
    }
    return !interactive
  } finally {
    autofillState.applying = false
  }
}

function mergeIgnoredConflictSignatures(form, conflicts) {
  if (!Array.isArray(form.ignored_conflicts)) {
    form.ignored_conflicts = []
  }
  const signatures = (conflicts || []).map(item => item?.signature).filter(Boolean)
  if (!signatures.length) return
  form.ignored_conflicts = [...new Set([...form.ignored_conflicts, ...signatures])]
}

async function assistCompletedRecord(form, { interactive = false } = {}) {
  const payload = toAssistPayload(form)
  const meaningfulKeys = ['order_id', 'flight_id', 'product_name', 'customer_name', 'principal', 'rebate_target']
  if (!meaningfulKeys.some(key => payload[key] !== '' && payload[key] != null)) {
    return { allowed: true, conflicts: [] }
  }

  try {
    autofillState.applying = true
    const res = await fetch('/api/rebate/completed/assist', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '自动识别失败')
    applyAutofillToForm(form, data.autofill)
    if (data.conflicts?.length) {
      if (!interactive) return { allowed: false, conflicts: data.conflicts }
      const confirmed = window.confirm(`检测到与交易表不一致的字段：\n${conflictMessage(data.conflicts)}\n\n点击“确定”忽略并继续，点击“取消”返回修改。`)
      if (confirmed) {
        mergeIgnoredConflictSignatures(form, data.conflicts)
      }
      return { allowed: confirmed, conflicts: data.conflicts }
    }
    return { allowed: true, conflicts: [] }
  } catch (error) {
    if (interactive) {
      alert(error.message || '自动识别失败')
    }
    return { allowed: !interactive, conflicts: [] }
  } finally {
    autofillState.applying = false
  }
}

async function validateLoadedItems(rows) {
  const validated = []
  let conflictCount = 0
  let issueCount = 0
  for (const row of rows) {
    try {
      const res = await fetch('/api/rebate/completed/assist', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(toAssistPayload(row)),
      })
      const data = await res.json()
      const conflicts = res.ok ? (data.conflicts || []) : []
      if (conflicts.length > 0) conflictCount += 1
      issueCount += conflicts.length
      validated.push({
        ...row,
        validation_conflicts: conflicts,
      })
    } catch {
      validated.push({
        ...row,
        validation_conflicts: [],
      })
    }
  }
  validationSummary.count = conflictCount
  validationSummary.issueCount = issueCount
  return validated
}

async function submitAddForm() {
  if (!addForm.order_id) {
    alert('请填写订单号')
    return
  }
  addModal.submitting = true
  try {
    const assistResult = await assistCompletedRecord(addForm, { interactive: true })
    if (!assistResult.allowed) return
    const body = { ...addForm }
    // Clean null numeric fields
    for (const key of ['principal', 'margin_ratio', 'channel_subscribe_ratio', 'channel_management_ratio', 'channel_performance_ratio', 'expense_amount']) {
      if (body[key] === null || body[key] === '') body[key] = 0
    }
    body.source = 'manual'

    const res = await fetch('/api/rebate/completed', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
    if (!res.ok) throw new Error('提交失败')
    addModal.visible = false
    await fetchData()
  } catch {
    alert('提交失败，请重试')
  } finally {
    addModal.submitting = false
  }
}

// --- Batch upload ---
function triggerFileUpload() {
  if (fileInputRef.value) {
    fileInputRef.value.value = ''
    fileInputRef.value.click()
  }
}

// Column header mapping: Chinese headers to field names
const COLUMN_MAP = {
  '订单号': 'order_id',
  '航班编号': 'flight_id',
  '产品名称': 'product_name',
  '客户姓名': 'customer_name',
  '渠道/直客': 'channel_or_direct',
  '渠道': 'channel_or_direct',
  '直客': 'channel_or_direct',
  '本金': 'principal',
  '保证金比例': 'margin_ratio',
  '业务类型': 'business_type',
  '认购日': 'subscribe_date',
  '订单状态': 'order_status',
  '返还人': 'rebate_target',
  '渠道返还比例-申购费': 'channel_subscribe_ratio',
  '申购费返还比例': 'channel_subscribe_ratio',
  '渠道返还比例-管理费': 'channel_management_ratio',
  '管理费返还比例': 'channel_management_ratio',
  '渠道返还比例-业绩报酬': 'channel_performance_ratio',
  '业绩报酬返还比例': 'channel_performance_ratio',
  '费用类别': 'expense_category',
  '类别': 'expense_category',
  '金额': 'expense_amount',
  '支出金额': 'expense_amount',
  '支付时间': 'payment_time',
  '年': 'payment_year',
  '月': 'payment_month',
  '日': 'payment_day',
  '来源': 'source',
}

function resolveColumnField(trimmedKey) {
  if (COLUMN_MAP[trimmedKey]) return COLUMN_MAP[trimmedKey]

  if (trimmedKey === '产品表对应类别') return 'expense_category'
  if (trimmedKey === '支付年') return 'payment_year'
  if (trimmedKey === '支付月') return 'payment_month'
  if (trimmedKey === '支付日') return 'payment_day'

  if (/^(渠道)?返还比例-?申购费$/.test(trimmedKey) || trimmedKey === '申购费比例') {
    return 'channel_subscribe_ratio'
  }
  if (/^(渠道)?返还比例-?管理费$/.test(trimmedKey) || trimmedKey === '管理费比例') {
    return 'channel_management_ratio'
  }
  if (/^(渠道)?返还比例-?业绩报酬$/.test(trimmedKey) || trimmedKey === '业绩报酬比例') {
    return 'channel_performance_ratio'
  }
  if (trimmedKey === '支出金额（元）' || trimmedKey === '金额（元）') {
    return 'expense_amount'
  }
  if (trimmedKey === '支付日期' || trimmedKey === '支付年月日') {
    return 'payment_time'
  }

  return trimmedKey
}

function normalizeHeaderText(value) {
  return String(value ?? '').replace(/\s+/g, ' ').trim()
}

function buildRowsFromSheetMatrix(matrix) {
  if (!Array.isArray(matrix) || matrix.length === 0) return []

  const firstRow = Array.isArray(matrix[0]) ? matrix[0] : []
  const secondRow = Array.isArray(matrix[1]) ? matrix[1] : []
  const secondHeaderMarkers = new Set([
    '申购费比例',
    '管理费比例',
    '业绩报酬比例',
    '产品表对应类别',
    '支出金额',
    '支付时间',
    '支付年',
    '支付月',
    '支付日',
  ])
  const secondHeaderHits = secondRow
    .map(normalizeHeaderText)
    .filter(text => secondHeaderMarkers.has(text))
    .length

  if (secondHeaderHits < 3) {
    const headers = firstRow.map(cell => normalizeHeaderText(cell))
    return matrix
      .slice(1)
      .map((row) => {
        const record = {}
        headers.forEach((header, index) => {
          if (!header) return
          record[header] = row?.[index] ?? ''
        })
        return record
      })
      .filter(row => Object.values(row).some(value => String(value ?? '').trim() !== ''))
  }

  const groupHeaders = firstRow.map(cell => normalizeHeaderText(cell))
  const subHeaders = secondRow.map(cell => normalizeHeaderText(cell))
  const headers = groupHeaders.map((groupHeader, index) => {
    const subHeader = subHeaders[index]
    if (subHeader) {
      const field = resolveColumnField(subHeader)
      if (field !== subHeader || COLUMN_MAP[subHeader]) return subHeader
      const combined = `${groupHeader}-${subHeader}`
      const combinedField = resolveColumnField(combined)
      if (combinedField !== combined || COLUMN_MAP[combined]) return combined
    }
    return groupHeader
  })

  return matrix
    .slice(2)
    .map((row) => {
      const record = {}
      headers.forEach((header, index) => {
        if (!header) return
        record[header] = row?.[index] ?? ''
      })
      return record
    })
    .filter(row => Object.values(row).some(value => String(value ?? '').trim() !== ''))
}

async function parseExcelFile(file) {
  const XLSX = await import('xlsx')
  const data = await file.arrayBuffer()
  const wb = XLSX.read(data, { type: 'array' })
  const ws = wb.Sheets[wb.SheetNames[0]]
  const matrix = XLSX.utils.sheet_to_json(ws, { header: 1, defval: '', raw: false })
  return buildRowsFromSheetMatrix(matrix)
}

const NUMERIC_FIELDS = new Set([
  'principal',
  'margin_ratio',
  'channel_subscribe_ratio',
  'channel_management_ratio',
  'channel_performance_ratio',
  'expense_amount',
])

const RATIO_FIELDS = new Set([
  'margin_ratio',
  'channel_subscribe_ratio',
  'channel_management_ratio',
  'channel_performance_ratio',
])

const DATE_FIELDS = new Set(['subscribe_date', 'payment_time'])

function pad2(value) {
  return String(value).padStart(2, '0')
}

function excelSerialToDateString(value) {
  const serial = Number(value)
  if (!Number.isFinite(serial)) return ''
  const utcDays = Math.floor(serial - 25569)
  const utcValue = utcDays * 86400
  const dateInfo = new Date(utcValue * 1000)
  if (Number.isNaN(dateInfo.getTime())) return ''
  return `${dateInfo.getUTCFullYear()}-${pad2(dateInfo.getUTCMonth() + 1)}-${pad2(dateInfo.getUTCDate())}`
}

function normalizeNumericValue(fieldName, value) {
  if (value == null) return null
  if (typeof value === 'number') return Number.isFinite(value) ? value : null
  const text = String(value).trim().replace(/,/g, '')
  if (!text) return null
  const isPercent = text.endsWith('%')
  const parsed = Number(isPercent ? text.slice(0, -1) : text)
  if (!Number.isFinite(parsed)) return null
  if (isPercent && RATIO_FIELDS.has(fieldName)) return parsed / 100
  return parsed
}

function normalizeDateValue(value) {
  if (value == null) return ''
  if (typeof value === 'number') return excelSerialToDateString(value)

  const text = String(value).trim()
  if (!text) return ''
  if (/^\d{5,6}(\.\d+)?$/.test(text)) {
    return excelSerialToDateString(text)
  }

  const normalized = text.replace(/[./]/g, '-')
  const match = normalized.match(/^(\d{4})-(\d{1,2})-(\d{1,2})$/)
  if (match) {
    return `${match[1]}-${pad2(match[2])}-${pad2(match[3])}`
  }

  const parsed = new Date(normalized)
  if (Number.isNaN(parsed.getTime())) return text
  return `${parsed.getFullYear()}-${pad2(parsed.getMonth() + 1)}-${pad2(parsed.getDate())}`
}

function normalizeRowValue(fieldName, value) {
  if (NUMERIC_FIELDS.has(fieldName)) return normalizeNumericValue(fieldName, value)
  if (DATE_FIELDS.has(fieldName)) return normalizeDateValue(value)
  if (fieldName === 'payment_year') {
    const text = String(value ?? '').trim().replace(/年$/, '')
    return text || ''
  }
  if (fieldName === 'payment_month' || fieldName === 'payment_day') {
    const text = String(value ?? '').trim().replace(/[月日]$/, '')
    return text ? pad2(text) : ''
  }
  return typeof value === 'string' ? value.trim() : value
}

function mapRowFields(rawRow) {
  const mapped = {}
  for (const [key, value] of Object.entries(rawRow)) {
    const trimmedKey = String(key).trim()
    const fieldName = resolveColumnField(trimmedKey)
    mapped[fieldName] = normalizeRowValue(fieldName, value)
  }
  // Auto-fill year/month/day from payment_time if present
  if (mapped.payment_time && !mapped.payment_year) {
    const dateStr = String(mapped.payment_time)
    const parts = dateStr.split('-')
    if (parts.length === 3) {
      mapped.payment_year = parts[0]
      mapped.payment_month = parts[1]
      mapped.payment_day = parts[2]
    }
  }
  return mapped
}

async function handleFileSelect(event) {
  const file = event.target.files?.[0]
  if (!file) return

  try {
    const rawRows = await parseExcelFile(file)
    if (!rawRows || rawRows.length === 0) {
      alert('文件中未找到数据')
      return
    }
    uploadModal.records = rawRows.map(mapRowFields)
    uploadModal.submitting = false
    uploadModal.visible = true
  } catch {
    alert('文件解析失败，请检查文件格式')
  }
}

async function submitBatchUploadLegacy() {
  if (uploadModal.records.length === 0) return
  uploadModal.submitting = true
  try {
    const records = []
    for (const row of uploadModal.records) {
      const cloned = { ...row }
      const allowed = await assistCompletedRecord(cloned, { interactive: false })
      if (!allowed) {
        const ignore = window.confirm(`上传记录与交易表存在冲突：\n订单号=${cloned.order_id || '--'}\n点击“确定”忽略并继续上传，点击“取消”停止上传。`)
        if (!ignore) return
      }
      records.push({
        ...cloned,
        source: 'upload',
      })
    }
    const res = await fetch('/api/rebate/completed/batch', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ records }),
    })
    if (!res.ok) throw new Error('上传失败')
    uploadModal.visible = false
    uploadModal.records = []
    await fetchData()
  } catch {
    alert('批量上传失败，请重试')
  } finally {
    uploadModal.submitting = false
  }
}

async function submitBatchUpload() {
  if (uploadModal.records.length === 0) return
  uploadModal.submitting = true
  try {
    const records = []
    for (const row of uploadModal.records) {
      const cloned = {
        ...row,
        ignored_conflicts: Array.isArray(row.ignored_conflicts) ? [...row.ignored_conflicts] : [],
      }
      const assistResult = await assistCompletedRecord(cloned, { interactive: false })
      if (!assistResult.allowed) {
        const ignore = window.confirm(`上传记录与交易表存在冲突：\n订单号：${cloned.order_id || '--'}\n${conflictMessage(assistResult.conflicts)}\n\n点击“确定”忽略并继续上传，点击“取消”停止上传。`)
        if (!ignore) return
        mergeIgnoredConflictSignatures(cloned, assistResult.conflicts)
      }
      records.push({
        ...cloned,
        source: 'upload',
      })
    }
    const res = await fetch('/api/rebate/completed/batch', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ records }),
    })
    if (!res.ok) throw new Error('批量上传失败')
    uploadModal.visible = false
    uploadModal.records = []
    await fetchData()
  } catch {
    alert('批量上传失败，请重试')
  } finally {
    uploadModal.submitting = false
  }
}

function cancelUpload() {
  uploadModal.visible = false
  uploadModal.records = []
}

async function downloadTemplate() {
  const XLSX = await import('xlsx')
  const rows = [
    ['订单号', '航班编号', '产品名称', '客户姓名', '渠道/直客', '本金', '保证金比例', '业务类型', '认购日', '订单状态', '返还人', '渠道返还比例', '', '', '支出流水明细', '', '', '', '', ''],
    ['', '', '', '', '', '', '', '', '', '', '', '申购费比例', '管理费比例', '业绩报酬比例', '产品表对应类别', '支出金额', '支付时间', '支付年', '支付月', '支付日'],
  ]

  const ws = XLSX.utils.aoa_to_sheet(rows)
  ws['!merges'] = [
    { s: { r: 0, c: 0 }, e: { r: 1, c: 0 } },
    { s: { r: 0, c: 1 }, e: { r: 1, c: 1 } },
    { s: { r: 0, c: 2 }, e: { r: 1, c: 2 } },
    { s: { r: 0, c: 3 }, e: { r: 1, c: 3 } },
    { s: { r: 0, c: 4 }, e: { r: 1, c: 4 } },
    { s: { r: 0, c: 5 }, e: { r: 1, c: 5 } },
    { s: { r: 0, c: 6 }, e: { r: 1, c: 6 } },
    { s: { r: 0, c: 7 }, e: { r: 1, c: 7 } },
    { s: { r: 0, c: 8 }, e: { r: 1, c: 8 } },
    { s: { r: 0, c: 9 }, e: { r: 1, c: 9 } },
    { s: { r: 0, c: 10 }, e: { r: 1, c: 10 } },
    { s: { r: 0, c: 11 }, e: { r: 0, c: 13 } },
    { s: { r: 0, c: 14 }, e: { r: 0, c: 19 } },
  ]
  ws['!cols'] = [
    { wch: 20 }, { wch: 14 }, { wch: 22 }, { wch: 14 }, { wch: 12 },
    { wch: 14 }, { wch: 12 }, { wch: 12 }, { wch: 12 }, { wch: 12 },
    { wch: 14 }, { wch: 12 }, { wch: 12 }, { wch: 14 }, { wch: 14 },
    { wch: 12 }, { wch: 14 }, { wch: 10 }, { wch: 10 }, { wch: 10 },
  ]

  const wb = XLSX.utils.book_new()
  XLSX.utils.book_append_sheet(wb, ws, '已返费模板')
  XLSX.writeFile(wb, `已返费明细导入模板_${new Date().toISOString().slice(0, 10)}.xlsx`)
}

// --- Delete ---
function confirmDelete(item) {
  deleteDialog.item = item
  deleteDialog.visible = true
}

async function executeDelete() {
  const item = deleteDialog.item
  if (!item) return
  deleteDialog.visible = false
  try {
    const res = await fetch(`/api/rebate/completed/${item.id}`, {
      method: 'DELETE',
    })
    if (!res.ok) throw new Error('删除失败')
    await fetchData()
  } catch {
    alert('删除失败，请重试')
  } finally {
    deleteDialog.item = null
  }
}

// --- CSV download ---
function downloadCSV() {
  if (items.value.length === 0) return

  const headers = [
    '订单号', '航班编号', '产品名称', '客户姓名', '渠道/直客', '本金', '保证金比例',
    '业务类型', '认购日', '订单状态', '返还人',
    '渠道返还比例-申购费', '渠道返还比例-管理费', '渠道返还比例-业绩报酬',
    '费用类别', '金额', '支付时间', '年', '月', '日',
    '来源',
  ]

  const rows = items.value.map(item => [
    item.order_id,
    item.flight_id,
    item.product_name,
    item.customer_name,
    item.channel_or_direct,
    fmtNum(item.principal),
    fmtPct(item.margin_ratio),
    item.business_type,
    item.subscribe_date,
    item.order_status,
    item.rebate_target,
    fmtPct(item.channel_subscribe_ratio),
    fmtPct(item.channel_management_ratio),
    fmtPct(item.channel_performance_ratio),
    item.expense_category,
    fmtNum(item.expense_amount),
    item.payment_time,
    item.payment_year,
    item.payment_month,
    item.payment_day,
    sourceLabel(item.source),
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
  link.download = `已返费分析_${new Date().toISOString().slice(0, 10)}.csv`
  link.click()
  URL.revokeObjectURL(url)
}
</script>

<style scoped>
:deep(.workbench-main) {
  max-width: none;
}

.rebate-completed-page {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  font-size: 12px;
}

.rebate-completed-page > .page-header {
  flex-shrink: 0;
}

.rebate-completed-page > .filter-bar {
  flex-shrink: 0;
  margin-bottom: 8px;
}

.rebate-completed-page > .action-bar {
  flex-shrink: 0;
}

.input-compact {
  min-width: 80px !important;
  width: 100px !important;
}

.input-select-compact {
  min-width: 116px;
}

.validation-banner {
  margin-bottom: 12px;
  padding: 10px 14px;
  border: 1px solid rgba(239, 68, 68, 0.18);
  border-radius: var(--radius);
  background: var(--danger-soft);
  color: #9f1239;
  font-size: 13px;
  font-weight: 700;
}

/* --- Action bar --- */
.action-bar {
  display: flex;
  gap: 10px;
  margin-bottom: 8px;
}

.table-bottom-actions {
  display: flex;
  justify-content: flex-end;
  padding: 8px 0;
}

.file-input-hidden {
  position: absolute;
  width: 0;
  height: 0;
  opacity: 0;
  pointer-events: none;
}

/* --- Scrollable table container --- */
.rebate-completed-page > .table-section {
  flex: 1;
  min-height: 0;
}

.rebate-completed-page > .table-section > .table-wrap {
  overflow-x: auto;
  flex: 1;
  min-height: 0;
  max-height: none;
}

/* --- Table --- */
.completed-table {
  min-width: 2800px;
  font-family: var(--font-sans);
  font-size: 15px;
  border-collapse: separate;
  border-spacing: 0;
  background: var(--bg-card);
}

.completed-table thead {
  position: sticky;
  top: 0;
  z-index: 10;
}

.completed-table .header-group-row th {
  padding: 8px 12px;
  font-size: 12px;
  font-weight: 700;
  text-align: left;
  white-space: nowrap;
  color: var(--ink-strong);
  border-bottom: 1px solid var(--border-soft);
  background: #fffaf4;
  letter-spacing: 0;
  line-height: 1.4;
}

.completed-table .header-group-row th[rowspan="2"] {
  vertical-align: middle;
  background: #fffaf4;
}

.completed-table .header-sub-row th {
  padding: 6px 12px;
  font-size: 11px;
  font-weight: 600;
  text-align: left;
  white-space: nowrap;
  color: var(--ink-soft);
  border-bottom: 1px solid var(--border-soft);
  letter-spacing: 0;
  background: #fffaf4;
}

/* Group header colors — 渠道返还比例: purple */
.completed-table .group-channel-ratio {
  background: #f1edfb !important;
  color: #6b5b95 !important;
  text-align: center !important;
}
.completed-table .sub-channel-ratio {
  background: #f1edfb !important;
  color: #6b5b95 !important;
}

/* Group header colors — 支出流水明细: light blue */
.completed-table .group-expense {
  background: #eef4ff !important;
  color: #2563a8 !important;
  text-align: center !important;
}
.completed-table .sub-expense {
  background: #eef4ff !important;
  color: #2563a8 !important;
}

.completed-table td {
  padding: 5px 10px;
  white-space: nowrap;
  border-bottom: 1px solid var(--border-soft);
  color: var(--ink-strong);
  font-size: 12px;
  line-height: 1.4;
  background: var(--bg-card);
  text-align: left !important;
}

/* Sticky first column — body cells */
.completed-table tbody .sticky-col {
  position: sticky;
  left: 0;
  z-index: 5;
  background: var(--bg-card);
}

.row-alt .sticky-col {
  background: #fcfcfd;
}

.row-alt td {
  background: #fcfcfd;
}

.completed-table tbody tr:hover .sticky-col {
  background: #f7f8fa;
}

.completed-table tbody tr:hover td {
  background: #f7f8fa;
}

/* Sticky first column — header cells need higher z-index */
.completed-table thead .sticky-col {
  position: sticky;
  left: 0;
  z-index: 20 !important;
  text-align: left;
  background: #fffaf4;
}

@media (max-width: 1440px) {
  .filter-actions {
    width: 100%;
    margin-left: 0;
    justify-content: flex-end;
  }
}

.name-cell {
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
  cursor: default;
}

/* --- Source pill --- */
.source-pill {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 2px 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 700;
  line-height: 1;
}

.source-manual {
  color: #215bd7;
  background: var(--brand-soft);
}

.source-upload {
  color: #087c58;
  background: var(--success-soft);
}

.source-auto {
  color: #8a5a00;
  background: var(--warning-soft);
}

.validation-pill {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 2px 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 700;
  line-height: 1;
}

.validation-ok {
  color: #087c58;
  background: var(--success-soft);
}

.validation-error {
  color: #b42318;
  background: var(--danger-soft);
  cursor: help;
}

.validation-muted {
  color: #8a5a00;
  background: var(--warning-soft);
}

/* --- Icon button --- */
.btn-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--ink-faint);
  cursor: pointer;
  transition: background 150ms ease, color 150ms ease;
}

.btn-icon:hover {
  background: var(--bg-hover);
  color: var(--ink);
}

.btn-icon-danger:hover {
  background: var(--danger-soft);
  color: var(--danger);
}

/* --- Modal --- */
.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.35);
  backdrop-filter: blur(2px);
}

.modal-card {
  width: 640px;
  max-width: 90vw;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
}

.modal-card-lg {
  width: 900px;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 24px 0;
}

.modal-title {
  font-size: 17px;
  font-weight: 750;
  color: var(--ink-strong);
}

.modal-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--ink-faint);
  cursor: pointer;
  transition: background 150ms ease, color 150ms ease;
}

.modal-close:hover {
  background: var(--bg-hover);
  color: var(--ink);
}

.modal-body {
  padding: 20px 24px;
  overflow-y: auto;
  flex: 1;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 16px 24px;
  border-top: 1px solid var(--border-soft);
}

/* --- Form grid --- */
.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px 20px;
  margin-bottom: 20px;
}

.form-grid-3 {
  grid-template-columns: 1fr 1fr 1fr;
}

.form-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-field label {
  color: var(--ink-soft);
  font-size: 13px;
  font-weight: 700;
  white-space: nowrap;
}

.form-field .input {
  height: 38px;
  font-size: 13px;
}

.form-field-full {
  grid-column: 1 / -1;
}

.form-section-title {
  font-size: 14px;
  font-weight: 750;
  color: var(--ink-strong);
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border-soft);
}

/* --- Upload preview --- */
.upload-info {
  font-size: 14px;
  color: var(--ink);
  margin-bottom: 16px;
}

.upload-preview-table {
  max-height: 400px;
  overflow-y: auto;
}

.upload-truncated {
  font-size: 13px;
  color: var(--ink-faint);
  margin-top: 12px;
  text-align: center;
}

/* --- Delete dialog --- */
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

/* --- Danger button --- */
.btn-danger {
  color: #fff;
  background: var(--danger);
  border-color: var(--danger);
  box-shadow: var(--shadow-sm);
}

.btn-danger:hover {
  background: #dc2626;
  border-color: #dc2626;
}
</style>
