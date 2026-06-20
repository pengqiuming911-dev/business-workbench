<template>
  <div class="product-completion-page">
    <h1 class="text-page-title">派息/敲出观察</h1>

    <div class="tab-bar">
      <button class="btn tab-btn" :class="{ active: activeTab === 'all' }" @click="activeTab = 'all'">全量</button>
      <button class="btn tab-btn" :class="{ active: activeTab === 'calendar' }" @click="activeTab = 'calendar'; loadCalendarData()">观察日历</button>
      <button class="btn tab-btn" :class="{ active: activeTab === 'today' }" @click="activeTab = 'today'; loadTodayData()">今日观察</button>
      <button class="btn tab-btn" :class="{ active: activeTab === 'posters' }" @click="activeTab = 'posters'; loadPosters()">🎉 喜报</button>
    </div>

    <div v-if="activeTab === 'all'">
      <p class="text-body" style="margin-bottom: 24px;">展示存续产品（持有中）的派息与敲出观察情况。数据来源为航班服务交易总表 · 产品表。</p>

      <PanelCard title="操作">
        <div class="form-row">
          <label>数据来源</label>
          <div class="file-source">
            <span class="text-card-title">📊 航班服务交易总表 · 产品表</span>
            <span class="badge badge-blue">本地数据库</span>
          </div>
        </div>
        <div class="form-row">
          <label>搜索</label>
          <input v-model="searchText" type="text" class="input" placeholder="按产品名称或航班编号搜索..." />
        </div>
        <button class="btn btn-primary" :disabled="refreshing" @click="refreshPrices">
          {{ refreshing ? '刷新中...' : '刷新标的价格' }}
        </button>
        <button class="btn btn-secondary" :disabled="generating" @click="generateObservations">
          {{ generating ? '生成中...' : '生成观察记录' }}
        </button>
        <span v-if="lastUpdated" class="text-label" style="margin-left: 16px;">最后更新: {{ lastUpdated }}</span>
        <span v-if="errorMsg" class="error-msg" style="margin-left: 12px;">{{ errorMsg }}</span>
        <span v-if="successMsg" class="success-msg" style="margin-left: 12px;">{{ successMsg }}</span>
      </PanelCard>

      <div v-if="filteredProducts.length">
        <PanelCard title="存续产品观察概览">
          <div class="table-wrap">
            <table class="overview-table">
              <thead>
                <tr>
                  <th class="col-left sticky-col">航班编号</th>
                  <th class="col-left">产品名称</th>
                  <th class="col-left">私募管理人</th>
                  <th class="col-left">持有状态</th>
                  <th class="col-left">代码</th>
                  <th class="col-right">入场价</th>
                  <th class="col-left">入场日</th>
                  <th class="col-right">存续月</th>
                  <th class="col-right">锁定期(月)</th>
                  <th class="col-left">最近观察日</th>
                  <th class="col-left">下个观察日</th>
                  <th class="col-right">标的价格</th>
                  <th class="col-right">降敲</th>
                  <th class="col-right">敲出价</th>
                  <th class="col-right">派息线</th>
                  <th class="col-center">是否敲出</th>
                  <th class="col-center">是否派息</th>
                </tr>
              </thead>
              <tbody>
                <template v-for="p in filteredProducts" :key="p.id">
                  <tr class="data-row" @click="toggleExpand(p.id)">
                    <td class="col-left sticky-col">
                      <span class="chevron" :class="{ open: expandedId === p.id }">›</span>
                      {{ p.id }}
                    </td>
                    <td class="col-left">{{ p.name }}</td>
                    <td class="col-left">{{ p.manager }}</td>
                    <td class="col-left"><span class="badge badge-green">{{ p.holding_status }}</span></td>
                    <td class="col-left code-cell">{{ p.code }}</td>
                    <td class="col-right">{{ formatPrice(p.entry_price, p) }}</td>
                    <td class="col-left">{{ p.issue_date || '--' }}</td>
                    <td class="col-right">{{ computeMonthsSince(p) }}</td>
                    <td class="col-right">{{ p.lock_months || '--' }}</td>
                    <td class="col-left">{{ latestObs(p)?.date || '--' }}</td>
                    <td class="col-left next-date">{{ p.next_observation_date || '--' }}</td>
                    <td class="col-right">{{ formatPrice(latestObs(p)?.underlying_price, p) }}</td>
                    <td class="col-right">{{ p.monthly_decrease ?? '--' }}</td>
                    <td class="col-right">{{ formatPrice(latestObs(p)?.knockout_price, p) }}</td>
                    <td class="col-right">{{ formatPrice(latestObs(p)?.dividend_line, p) }}</td>
                    <td class="col-center" :class="knockoutClass(latestObs(p)?.is_knocked_out)">
                      {{ latestObs(p)?.is_knocked_out || '--' }}
                    </td>
                    <td class="col-center" :class="dividendClass(latestObs(p)?.is_dividend)">
                      {{ latestObs(p)?.is_dividend || '--' }}
                    </td>
                  </tr>
                  <tr v-if="expandedId === p.id && p.observations.length" class="detail-row">
                    <td colspan="17" class="detail-cell">
                      <div class="detail-label">历史观察日明细</div>
                      <table class="detail-table">
                        <thead>
                          <tr>
                            <th>观察日</th>
                            <th>标的价格</th>
                            <th>敲出价</th>
                            <th>派息线</th>
                            <th>是否敲出</th>
                            <th>是否派息</th>
                          </tr>
                        </thead>
                        <tbody>
                          <tr v-for="obs in p.observations" :key="obs.date">
                            <td>{{ obs.date }}</td>
                            <td>{{ formatPrice(obs.underlying_price, p) }}</td>
                            <td>{{ formatPrice(obs.knockout_price, p) }}</td>
                            <td>{{ formatPrice(obs.dividend_line, p) }}</td>
                            <td :class="knockoutClass(obs.is_knocked_out)">{{ obs.is_knocked_out }}</td>
                            <td :class="dividendClass(obs.is_dividend)">{{ obs.is_dividend }}</td>
                          </tr>
                        </tbody>
                      </table>
                    </td>
                  </tr>
                  <tr v-if="expandedId === p.id && !p.observations.length" class="detail-row">
                    <td colspan="17" class="detail-cell">
                      <div class="detail-empty">暂无观察日记录</div>
                    </td>
                  </tr>
                </template>
              </tbody>
            </table>
          </div>
          <p class="table-summary">共 {{ filteredProducts.length }} 个存续产品</p>
        </PanelCard>
      </div>
      <div v-else-if="loaded && !filteredProducts.length" class="empty-state">
        <p>暂无存续产品数据，请先在「数据准备」页面同步飞书数据。</p>
      </div>
    </div>

    <div v-if="activeTab === 'calendar'" class="calendar-section">
      <div class="calendar-toolbar">
        <div class="calendar-month-picker">
          <label>月份</label>
          <input v-model="calendarMonth" type="month" class="input month-input" @change="loadCalendarData" />
          <label style="margin-left: 16px;">状态</label>
          <select v-model="calendarStatus" class="input month-input" @change="loadCalendarData">
            <option value="ongoing">存续</option>
            <option value="completed">已完结</option>
          </select>
          <span v-if="calendarError" class="error-msg" style="margin-left: 12px;">{{ calendarError }}</span>
        </div>
        <div class="calendar-summary">本月共 {{ calendarProductCount }} 个产品观察安排</div>
      </div>

      <div v-if="calendarLoading" class="loading-state"><p>加载中...</p></div>
      <div v-else class="calendar-wrap">
        <div class="calendar-weekdays">
          <div v-for="day in weekDays" :key="day" class="calendar-weekday">{{ day }}</div>
        </div>
        <div class="calendar-grid">
          <div
            v-for="cell in calendarCells"
            :key="cell.key"
            class="calendar-cell"
            :class="{ muted: !cell.inMonth, today: cell.date === todayDate }"
          >
            <div class="calendar-day" :class="{ 'has-items': cell.products.length }">{{ cell.day || '' }}</div>
            <div v-if="cell.products.length" class="calendar-products">
              <div
                v-for="product in cell.products"
                :key="product.id"
                class="cal-card"
                :title="product.name"
              >
                <div class="cal-card-name">{{ product.name || product.id }}</div>
                <div class="cal-card-details">
                  <div v-if="product.is_knockout_observable && product.knockout_price != null" class="cal-detail-row cal-detail-knockout">
                    <span class="cal-detail-label">敲出</span>
                    <strong>{{ fmtCalPrice(product.knockout_price) }}</strong>
                  </div>
                  <div v-if="product.has_dividend_observation && product.dividend_line != null" class="cal-detail-row cal-detail-dividend">
                    <span class="cal-detail-label">派息</span>
                    <strong>{{ fmtCalPrice(product.dividend_line) }}</strong>
                  </div>
                  <div v-if="product.spot_price != null" class="cal-detail-row cal-detail-spot">
                    <span class="cal-detail-label">{{ calendarStatus === 'completed' ? '当日' : '今日' }}</span>
                    <strong>{{ fmtCalPrice(product.spot_price) }}</strong>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-if="activeTab === 'today'">
      <p class="text-body" style="margin-bottom: 24px;">展示今日需要观察派息或敲出的存续产品。今日日期: {{ todayDate }}</p>
      <div v-if="todayLoading" class="loading-state"><p>加载中...</p></div>
      <div v-else-if="todayProducts.length">
        <PanelCard title="今日观察（{{ todayDate }}）">
          <div class="table-wrap">
            <table class="overview-table">
              <thead>
                <tr>
                  <th class="col-left sticky-col">航班编号</th>
                  <th class="col-left">产品名称</th>
                  <th class="col-left">私募管理人</th>
                  <th class="col-left">代码</th>
                  <th class="col-right">入场价</th>
                  <th class="col-right">存续月</th>
                  <th class="col-right">标的价格</th>
                  <th class="col-right">敲出价</th>
                  <th class="col-right">派息线</th>
                  <th class="col-center">是否敲出</th>
                  <th class="col-center">是否派息</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="p in todayProducts" :key="p.id" class="data-row">
                  <td class="col-left sticky-col">{{ p.id }}</td>
                  <td class="col-left">{{ p.name }}</td>
                  <td class="col-left">{{ p.manager }}</td>
                  <td class="col-left code-cell">{{ p.code }}</td>
                  <td class="col-right">{{ formatPrice(p.entry_price, p) }}</td>
                  <td class="col-right">{{ computeMonthsSince(p) }}</td>
                  <td class="col-right">{{ formatPrice(todayObs(p)?.underlying_price, p) }}</td>
                  <td class="col-right">{{ formatPrice(todayObs(p)?.knockout_price, p) }}</td>
                  <td class="col-right">{{ formatPrice(todayObs(p)?.dividend_line, p) }}</td>
                  <td class="col-center" :class="knockoutClass(todayObs(p)?.is_knocked_out)">
                    {{ todayObs(p)?.is_knocked_out || '--' }}
                  </td>
                  <td class="col-center" :class="dividendClass(todayObs(p)?.is_dividend)">
                    {{ todayObs(p)?.is_dividend || '--' }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <p class="table-summary">今日共 {{ todayProducts.length }} 个产品需观察</p>
        </PanelCard>
      </div>
      <div v-else-if="todayLoaded" class="empty-state">
        <p>今日无产品需要观察派息/敲出。</p>
      </div>
    </div>

    <div v-if="activeTab === 'posters'">
      <p class="text-body" style="margin-bottom: 24px;">自动生成敲出/派息喜报海报。可选择日期查询或生成对应日期的喜报。</p>

      <PanelCard title="喜报操作">
        <div class="form-row">
          <label>筛选日期</label>
          <input v-model="filterDate" type="date" class="input" style="width: 160px; flex: none;" @change="loadPosters" />
        </div>
        <div class="form-row">
          <button class="btn btn-primary" :disabled="posterGenerating" @click="generatePosters">
            {{ posterGenerating ? '生成中...' : '生成喜报' }}
          </button>
          <button class="btn btn-secondary" @click="resetFilterDate">重置为今日</button>
          <span v-if="posterMsg" class="success-msg" style="margin-left: 12px;">{{ posterMsg }}</span>
          <span v-if="posterError" class="error-msg" style="margin-left: 12px;">{{ posterError }}</span>
        </div>
      </PanelCard>

      <div v-if="posterLoading" class="loading-state"><p>加载中...</p></div>
      <div v-else-if="posterList.length === 0 && posterLoaded" class="empty-state">
        <p>{{ filterDate }} 暂无喜报。可点击"生成喜报"为该日期生成。</p>
      </div>
      <div v-else-if="posterList.length > 0">
        <PanelCard title="喜报（{{ filterDate }}）· 共 {{ posterList.length }} 张">
          <div class="poster-grid">
            <div v-for="p in posterList" :key="p.id" class="poster-card">
              <div class="poster-card-header">
                <span class="poster-type-badge" :class="p.poster_type">
                  {{ p.poster_type === 'knockout' ? '敲出喜报' : '派息喜报' }}
                </span>
                <span class="poster-product">{{ p.product_name }}</span>
              </div>
              <PosterTemplate
                :poster-type="p.poster_type"
                :data="p"
                @generated="onPosterGenerated"
              />
            </div>
          </div>
        </PanelCard>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import PanelCard from '../components/PanelCard.vue'
import PosterTemplate from '../components/PosterTemplate.vue'

const activeTab = ref('all')
const searchText = ref('')
const products = ref([])
const lastUpdated = ref(null)
const loaded = ref(false)
const refreshing = ref(false)
const generating = ref(false)
const errorMsg = ref('')
const successMsg = ref('')
const expandedId = ref(null)

const todayDate = ref(new Date().toISOString().slice(0, 10))
const todayProducts = ref([])
const todayLoading = ref(false)
const todayLoaded = ref(false)

const calendarMonth = ref(new Date().toISOString().slice(0, 7))
const calendarStatus = ref('ongoing')
const calendarItems = ref([])
const calendarLoading = ref(false)
const calendarLoaded = ref(false)
const calendarError = ref('')
const weekDays = ['一', '二', '三', '四', '五', '六', '日']

const posterList = ref([])
const posterLoading = ref(false)
const posterLoaded = ref(false)
const posterGenerating = ref(false)
const posterMsg = ref('')
const posterError = ref('')
const filterDate = ref(new Date().toISOString().slice(0, 10))

onMounted(() => loadData())

async function loadData() {
  errorMsg.value = ''
  try {
    const res = await fetch('/api/observations')
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '加载失败')
    products.value = data.products || []
    lastUpdated.value = data.lastUpdated
  } catch (err) {
    errorMsg.value = err.message
  } finally {
    loaded.value = true
  }
}

async function loadTodayData() {
  todayLoading.value = true
  try {
    const res = await fetch('/api/observations/today')
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '加载失败')
    todayProducts.value = data.products || []
    todayDate.value = data.today || todayDate.value
  } catch (err) {
    errorMsg.value = err.message
  } finally {
    todayLoading.value = false
    todayLoaded.value = true
  }
}

async function loadCalendarData() {
  calendarLoading.value = true
  calendarError.value = ''
  try {
    const res = await fetch(`/api/observations/calendar?month=${calendarMonth.value}&status=${calendarStatus.value}`)
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '加载失败')
    calendarItems.value = data.calendar || []
    calendarMonth.value = data.month || calendarMonth.value
  } catch (err) {
    calendarError.value = err.message
  } finally {
    calendarLoading.value = false
    calendarLoaded.value = true
  }
}

async function refreshPrices() {
  refreshing.value = true
  errorMsg.value = ''
  successMsg.value = ''
  try {
    const res = await fetch('/api/observations/refresh-prices', { method: 'POST' })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '刷新失败')
    successMsg.value = `价格刷新完成：${data.refreshed} 个成功${data.failed ? '，' + data.failed + ' 个失败' : ''}`
    await loadData()
  } catch (err) {
    errorMsg.value = err.message
  } finally {
    refreshing.value = false
  }
}

async function generateObservations() {
  generating.value = true
  errorMsg.value = ''
  successMsg.value = ''
  try {
    const res = await fetch('/api/observations/generate', { method: 'POST' })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '生成失败')
    const recalculated = data.recalculatedExisting ?? data.skippedExisting ?? 0
    successMsg.value = `生成完成：新增 ${data.generated} 条${recalculated ? '，重算 ' + recalculated + ' 条' : ''}`
    await loadData()
  } catch (err) {
    errorMsg.value = err.message
  } finally {
    generating.value = false
  }
}

function toggleExpand(id) {
  expandedId.value = expandedId.value === id ? null : id
}

async function loadPosters() {
  posterLoading.value = true
  posterError.value = ''
  try {
    const res = await fetch(`/api/posters/today?date=${filterDate.value}`)
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '加载失败')
    posterList.value = data.posters || []
    todayDate.value = data.today || todayDate.value
  } catch (err) {
    posterError.value = err.message
  } finally {
    posterLoading.value = false
    posterLoaded.value = true
  }
}

async function generatePosters() {
  posterGenerating.value = true
  posterMsg.value = ''
  posterError.value = ''
  try {
    const res = await fetch('/api/posters/generate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ date: filterDate.value }),
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '生成失败')
    posterMsg.value = `生成完成：敲出 ${data.knockout} 张，派息 ${data.dividend} 张`
    await loadPosters()
  } catch (err) {
    posterError.value = err.message
  } finally {
    posterGenerating.value = false
  }
}

function resetFilterDate() {
  filterDate.value = new Date().toISOString().slice(0, 10)
  loadPosters()
}

function onPosterGenerated(canvas) {
  console.log('喜报图片已生成')
}

const filteredProducts = computed(() => {
  if (!searchText.value) return products.value
  const q = searchText.value.toLowerCase()
  return products.value.filter(p =>
    (p.name && p.name.toLowerCase().includes(q)) || p.id.toLowerCase().includes(q)
  )
})

const calendarMap = computed(() => {
  const map = new Map()
  for (const item of calendarItems.value) {
    map.set(item.date, item.products || [])
  }
  return map
})

const calendarProductCount = computed(() => (
  calendarItems.value.reduce((sum, item) => sum + (item.products?.length || 0), 0)
))

const calendarCells = computed(() => {
  const [year, month] = calendarMonth.value.split('-').map(Number)
  if (!year || !month) return []

  const firstDay = new Date(year, month - 1, 1)
  const daysInMonth = new Date(year, month, 0).getDate()
  const leadingBlanks = (firstDay.getDay() + 6) % 7
  const totalCells = Math.ceil((leadingBlanks + daysInMonth) / 7) * 7
  const cells = []

  for (let i = 0; i < totalCells; i++) {
    const day = i - leadingBlanks + 1
    if (day < 1 || day > daysInMonth) {
      cells.push({ key: `blank-${i}`, day: '', date: null, inMonth: false, products: [] })
      continue
    }

    const date = `${calendarMonth.value}-${String(day).padStart(2, '0')}`
    cells.push({
      key: date,
      day,
      date,
      inMonth: true,
      products: calendarMap.value.get(date) || [],
    })
  }

  return cells
})

function latestObs(product) {
  if (!product.observations || !product.observations.length) return null
  return product.observations[product.observations.length - 1]
}

function todayObs(product) {
  if (!product.observations || !product.observations.length) return null
  return product.observations.find(o => o.date === todayDate.value) || product.observations[product.observations.length - 1]
}

function computeMonthsSince(product) {
  if (!product.issue_date) return '--'
  const entry = new Date(product.issue_date)
  const now = new Date()
  return (now.getFullYear() - entry.getFullYear()) * 12 + (now.getMonth() - entry.getMonth())
}

function isETF(product) {
  if (!product) return false
  return (product.name && product.name.includes('恒科ETF')) || (product.code && product.code.includes('恒科ETF'))
}

function formatPrice(val, product) {
  if (val === null || val === undefined) return '--'
  const decimals = isETF(product) ? 3 : 2
  return Number(val).toLocaleString('zh-CN', { minimumFractionDigits: decimals, maximumFractionDigits: decimals })
}

function knockoutClass(status) {
  if (status === '是') return 'result-yes-knockout'
  if (status === '否') return 'result-no'
  if (status === '不观察') return 'result-na'
  return ''
}

function dividendClass(status) {
  if (status === '是') return 'result-yes-dividend'
  if (status === '否') return 'result-no'
  return ''
}

function fmtCalPrice(val) {
  if (val == null) return '--'
  const value = Number(val)
  const decimals = Math.abs(value) < 10 ? 3 : 2
  return value.toLocaleString('zh-CN', {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  })
}
</script>

<style scoped>
:deep(.workbench-main) {
  max-width: none;
}

.tab-bar {
  display: flex;
  gap: 4px;
  margin-bottom: 24px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  padding: 4px;
  width: fit-content;
}

.tab-btn {
  border: none;
  background: transparent;
  color: var(--ink-soft);
}

.tab-btn:hover {
  background: var(--surface-muted);
  color: var(--ink);
}

.tab-btn.active {
  background: var(--brand);
  color: #fff;
}

.file-source { flex: 1; display: flex; align-items: center; gap: 10px; }

.month-input { width: 180px; flex: none; }

.table-summary {
  font-size: 12px;
  color: var(--ink-soft);
  text-align: right;
  padding-top: 8px;
}

.overview-table {
  width: 100%;
  /* border-collapse: separate 让 sticky 列/表头背景能正确盖住横向滚动内容，
     避免 collapse 下表头与首列透出相邻列（与 holding/rebate 表一致） */
  border-collapse: separate;
  border-spacing: 0;
  font-size: 12px;
  min-width: 1400px;
}

.overview-table th {
  padding: 10px 12px;
  border-bottom: 1px solid var(--border-soft);
  color: var(--ink-soft);
  font-weight: 600;
  background: var(--surface-muted);
  font-size: 11px;
  letter-spacing: 0.02em;
  white-space: nowrap;
  position: sticky;
  top: 0;
  z-index: 1;
}

.data-row {
  cursor: pointer;
  transition: background 0.15s;
}
.data-row:hover { background: var(--surface-muted); }

.overview-table td {
  padding: 11px 12px;
  border-bottom: 1px solid var(--border-soft);
  color: var(--ink-strong);
  white-space: nowrap;
}

.col-left { text-align: left; }
.col-right { text-align: left; }
.col-center { text-align: left; }

.sticky-col {
  position: sticky;
  left: 0;
  background: var(--bg-card);
  z-index: 2;
}
.data-row:hover .sticky-col { background: var(--surface-muted); }
.overview-table th.sticky-col { z-index: 3; background: var(--surface-muted); }

.chevron {
  font-size: 14px;
  color: var(--ink-soft);
  transition: transform 0.2s;
  display: inline-block;
  line-height: 1;
  margin-right: 4px;
}
.chevron.open { transform: rotate(90deg); }

.code-cell { font-family: var(--font-mono); font-size: 11px; color: var(--ink-soft); }
.next-date { color: var(--danger); font-weight: 600; }

.result-yes-knockout {
  color: var(--danger);
  font-weight: 600;
  background: var(--danger-soft);
  border-radius: 4px;
  padding: 2px 6px;
}

.result-yes-dividend {
  color: var(--success);
  font-weight: 600;
  background: var(--success-soft);
  border-radius: 4px;
  padding: 2px 6px;
}

.result-no { color: var(--ink-soft); }

.result-na {
  color: var(--ink-soft);
  font-style: italic;
  font-size: 11px;
}

.detail-row td {
  padding: 0;
  border-bottom: 1px solid var(--border-soft);
}

.detail-cell { background: var(--surface-muted); }

.detail-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--ink-soft);
  letter-spacing: 0.04em;
  padding: 12px 16px 8px;
}

.detail-empty {
  font-size: 12px;
  color: var(--ink-soft);
  padding: 12px 16px;
}

.detail-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 11px;
  margin: 0 16px 12px;
}

.detail-table th {
  padding: 6px 12px;
  border-bottom: 1px solid var(--border-soft);
  color: var(--ink-soft);
  font-weight: 600;
  background: transparent;
  text-align: left;
}

.detail-table td {
  padding: 6px 12px;
  border-bottom: 1px solid var(--border-soft);
  color: var(--ink);
}

.calendar-section {
  margin-top: 0;
}

.calendar-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  padding: 14px 18px;
  margin-bottom: 16px;
  box-shadow: 0 8px 24px rgba(37, 99, 235, 0.04);
}

.calendar-month-picker {
  display: flex;
  align-items: center;
  gap: 12px;
}

.calendar-month-picker label {
  font-size: 13px;
  font-weight: 600;
  color: var(--ink-soft);
}

.calendar-summary {
  font-size: 13px;
  font-weight: 600;
  color: var(--ink-soft);
  padding: 6px 0;
}

.calendar-wrap {
  background: var(--bg-card);
  border: 1px solid #dfe8f3;
  border-radius: var(--radius);
  overflow: hidden;
  box-shadow: 0 12px 32px rgba(37, 99, 235, 0.05);
}

.calendar-weekdays {
  display: grid;
  grid-template-columns: repeat(7, minmax(140px, 1fr));
  background: #f5f9ff;
  border-bottom: 1px solid #dfe8f3;
}

.calendar-weekday {
  padding: 12px 8px;
  color: #52657a;
  font-size: 12px;
  font-weight: 700;
  text-align: center;
  letter-spacing: 0.05em;
}

.calendar-grid {
  display: grid;
  grid-template-columns: repeat(7, minmax(140px, 1fr));
  overflow-x: auto;
}

.calendar-cell {
  min-height: 132px;
  padding: 10px 10px 8px;
  border-right: 1px solid #edf2f7;
  border-bottom: 1px solid #edf2f7;
  background: #fff;
}

.calendar-cell:nth-child(7n) {
  border-right: none;
}

.calendar-cell.muted {
  background: #fbfcfe;
}

.calendar-cell.today {
  background: #f1f7ff;
}

.calendar-day {
  font-size: 13px;
  font-weight: 700;
  color: var(--ink-faint);
  margin-bottom: 8px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.calendar-day.has-items {
  color: var(--ink-strong);
}

.calendar-day.has-items::after {
  content: '';
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: #60a5fa;
  flex-shrink: 0;
}

.calendar-products {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.cal-card {
  padding: 9px 10px;
  border-radius: 8px;
  border: 1px solid #dfe8f3;
  font-size: 11px;
  line-height: 1.45;
  background: #fff;
  transition: border-color 150ms ease, box-shadow 150ms ease;
}

.cal-card:hover {
  border-color: #bfd4ec;
  box-shadow: 0 8px 20px rgba(37, 99, 235, 0.08);
}

.cal-card-name {
  font-weight: 700;
  color: var(--ink-strong);
  font-size: 12px;
  line-height: 1.5;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  padding-bottom: 6px;
}

.cal-card-details {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding-top: 6px;
  border-top: 1px solid #edf2f7;
}

.cal-card-details:empty {
  display: none;
}

.cal-detail-row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
  min-height: 24px;
  padding: 3px 7px;
  border-radius: 6px;
  font-size: 11px;
}

.cal-detail-label {
  white-space: nowrap;
  font-weight: 700;
}

.cal-detail-row strong {
  font-weight: 700;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.cal-detail-knockout {
  color: #2563a8;
  background: #edf6ff;
}

.cal-detail-knockout strong {
  color: #1d4f8a;
}

.cal-detail-dividend {
  color: #16806a;
  background: #edfbf7;
}

.cal-detail-dividend strong {
  color: #116451;
}

.cal-detail-spot {
  color: #6b5b95;
  background: #f3f0fb;
}

.cal-detail-spot strong {
  color: #4c3f73;
}

.poster-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
  justify-content: center;
}

.poster-card {
  background: var(--bg-card);
  border-radius: var(--radius);
  padding: 20px;
  border: 1px solid var(--border-soft);
  box-shadow: none;
  transition: box-shadow 0.2s;
}

.poster-card:hover { box-shadow: var(--shadow-soft); }

.poster-card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.poster-type-badge {
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 700;
}

.poster-type-badge.knockout {
  background: var(--danger-soft);
  color: var(--danger);
}

.poster-type-badge.dividend {
  background: var(--success-soft);
  color: var(--success);
}

.poster-product {
  font-size: 13px;
  color: var(--ink-soft);
  font-weight: 700;
}
</style>
