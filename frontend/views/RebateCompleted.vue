<template>
  <div class="rebate-completed-page">
    <div class="page-header">
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
        <label>航班编号</label>
        <input
          v-model="filters.flightId"
          type="text"
          class="input input-sm"
          placeholder="请输入航班编号"
        />
      </div>
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
        <label>费用类别</label>
        <select v-model="filters.expenseCategory" class="input input-sm">
          <option value="">全部</option>
          <option value="申购费">申购费</option>
          <option value="管理费">管理费</option>
          <option value="业绩报酬">业绩报酬</option>
        </select>
      </div>
      <div class="filter-group">
        <label>来源</label>
        <select v-model="filters.source" class="input input-sm">
          <option value="">全部</option>
          <option value="manual">手工录入</option>
          <option value="upload">批量上传</option>
          <option value="auto_sync">自动同步</option>
        </select>
      </div>
      <div class="filter-actions">
        <button class="btn btn-primary btn-sm" @click="fetchData">
          <Search :size="14" />
          查询
        </button>
        <button class="btn btn-secondary btn-sm" @click="resetFilters">重置</button>
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
    <div v-else-if="items.length > 0" class="table-wrap">
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
          <col style="min-width: 70px" /><!-- 操作 -->
        </colgroup>
        <thead>
          <tr class="header-group-row">
            <th rowspan="2" class="sticky-col">订单号</th>
            <th rowspan="2">航班编号</th>
            <th rowspan="2">产品名称</th>
            <th rowspan="2">客户姓名</th>
            <th rowspan="2">渠道/直客</th>
            <th rowspan="2" class="num-col">本金</th>
            <th rowspan="2" class="num-col">保证金比例</th>
            <th rowspan="2">业务类型</th>
            <th rowspan="2">认购日</th>
            <th rowspan="2">订单状态</th>
            <th rowspan="2">返还人</th>
            <th colspan="3" class="group-header group-channel-ratio">渠道返还比例</th>
            <th colspan="6" class="group-header group-expense">支出流水明细</th>
            <th rowspan="2">来源</th>
            <th rowspan="2">操作</th>
          </tr>
          <tr class="header-sub-row">
            <!-- 渠道返还比例 -->
            <th class="num-col sub-channel-ratio">申购费</th>
            <th class="num-col sub-channel-ratio">管理费</th>
            <th class="num-col sub-channel-ratio">业绩报酬</th>
            <!-- 支出流水明细 -->
            <th class="sub-expense">类别</th>
            <th class="num-col sub-expense">金额</th>
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
            <td class="num-col">{{ fmtNum(item.principal) }}</td>
            <td class="num-col">{{ fmtPct(item.margin_ratio) }}</td>
            <td>{{ item.business_type || '--' }}</td>
            <td>{{ item.subscribe_date || '--' }}</td>
            <td>{{ item.order_status || '--' }}</td>
            <td>{{ item.rebate_target || '--' }}</td>
            <!-- 渠道返还比例 -->
            <td class="num-col">{{ fmtPct(item.channel_subscribe_ratio) }}</td>
            <td class="num-col">{{ fmtPct(item.channel_management_ratio) }}</td>
            <td class="num-col">{{ fmtPct(item.channel_performance_ratio) }}</td>
            <!-- 支出流水明细 -->
            <td>{{ item.expense_category || '--' }}</td>
            <td class="num-col">{{ fmtNum(item.expense_amount) }}</td>
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
    items.value = data.items || []
  } catch {
    items.value = []
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

async function submitAddForm() {
  if (!addForm.order_id) {
    alert('请填写订单号')
    return
  }
  addModal.submitting = true
  try {
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

async function parseExcelFile(file) {
  const XLSX = await import('xlsx')
  const data = await file.arrayBuffer()
  const wb = XLSX.read(data, { type: 'array' })
  const ws = wb.Sheets[wb.SheetNames[0]]
  return XLSX.utils.sheet_to_json(ws)
}

function mapRowFields(rawRow) {
  const mapped = {}
  for (const [key, value] of Object.entries(rawRow)) {
    const trimmedKey = String(key).trim()
    const fieldName = COLUMN_MAP[trimmedKey] || trimmedKey
    mapped[fieldName] = value
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

async function submitBatchUpload() {
  if (uploadModal.records.length === 0) return
  uploadModal.submitting = true
  try {
    const records = uploadModal.records.map(r => ({
      ...r,
      source: 'upload',
    }))
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

function cancelUpload() {
  uploadModal.visible = false
  uploadModal.records = []
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

/* --- Action bar --- */
.action-bar {
  display: flex;
  gap: 10px;
  margin-bottom: 16px;
}

.file-input-hidden {
  position: absolute;
  width: 0;
  height: 0;
  opacity: 0;
  pointer-events: none;
}

/* --- Table --- */
.completed-table {
  min-width: 2800px;
  font-size: 14px;
}

.completed-table thead {
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
.group-channel-ratio { background: #f0fdf4 !important; }
.group-expense       { background: #eef2ff !important; }

.sub-channel-ratio   { background: #f7fef9 !important; }
.sub-expense         { background: #f5f7ff !important; }

.completed-table td {
  padding: 8px 10px;
  white-space: nowrap;
  border-bottom: 1px solid var(--border-soft);
  color: var(--ink-strong);
  font-size: 14px;
}

.completed-table .num-col {
  text-align: right;
  font-variant-numeric: tabular-nums;
}

.completed-table thead .num-col {
  text-align: center;
}

.row-alt td {
  background: var(--bg-page);
}

.completed-table tr:hover td {
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

.completed-table tr:hover .sticky-col {
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
