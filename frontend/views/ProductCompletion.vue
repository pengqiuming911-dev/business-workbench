<template>
  <SubPageLayout
    title="产品派息/敲出观察"
    description="展示存续产品的派息与敲出观察情况，并生成对应日期的喜报海报。"
    wide
  >
    <div class="section">
      <div class="tab-bar">
        <button class="tab-btn" :class="{ active: activeTab === 'all' }" @click="activeTab = 'all'">全量</button>
        <button class="tab-btn" :class="{ active: activeTab === 'calendar' }" @click="activeTab = 'calendar'; loadCalendarData()">观察日历</button>
        <button class="tab-btn" :class="{ active: activeTab === 'today' }" @click="activeTab = 'today'; loadTodayData()">今日观察</button>
        <button class="tab-btn" :class="{ active: activeTab === 'posters' }" @click="activeTab = 'posters'; loadPosters()">🎉 喜报</button>
      </div>

      <div v-if="activeTab === 'all'">
        <p class="desc">展示存续产品（持有中）的派息与敲出观察情况。数据来源为航班服务交易总表 · 产品表。</p>

        <div class="panel">
          <h3 class="panel-title">操作</h3>
          <div class="form-row">
            <label>数据来源</label>
            <div class="file-source">
              <span class="file-badge">📊 航班服务交易总表 · 产品表</span>
              <span class="file-from">本地数据库</span>
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
          <span v-if="lastUpdated" class="update-time">最后更新: {{ lastUpdated }}</span>
          <span v-if="errorMsg" class="error">{{ errorMsg }}</span>
          <span v-if="successMsg" class="success">{{ successMsg }}</span>
        </div>

        <div v-if="filteredProducts.length" class="report-panel">
          <h3 class="section-title">存续产品观察概览</h3>
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
                    <td class="col-left"><span class="status-badge">{{ p.holding_status }}</span></td>
                    <td class="col-left code-cell">{{ p.code }}</td>
                    <td class="col-right">{{ formatPrice(p.entry_price, p) }}</td>
                    <td class="col-left">{{ p.issue_date || '--' }}</td>
                    <td class="col-right">{{ computeMonthsSince(p) }}</td>
                    <td class="col-right">{{ p.lock_months || '--' }}</td>
                    <td class="col-left">{{ latestObs(p)?.date || '--' }}</td>
                    <td class="col-left next-date">{{ p.next_observation_date || '--' }}</td>
                    <td class="col-right">{{ formatPrice(latestObs(p)?.underlying_price, p) }}</td>
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
                    <td colspan="16" class="detail-cell">
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
                    <td colspan="16" class="detail-cell">
                      <div class="detail-empty">暂无观察日记录</div>
                    </td>
                  </tr>
                </template>
              </tbody>
            </table>
          </div>
          <p class="table-summary">共 {{ filteredProducts.length }} 个存续产品</p>
        </div>
        <div v-else-if="loaded && !filteredProducts.length" class="empty-state">
          <p>暂无存续产品数据，请先在「数据准备」页面同步飞书数据。</p>
        </div>
      </div>

      <div v-if="activeTab === 'calendar'">
        <p class="desc">按月份查看存续产品观察日，并在对应日期展示需要观察的产品名称。</p>

        <div class="panel">
          <h3 class="panel-title">日历筛选</h3>
          <div class="form-row">
            <label>月份</label>
            <input v-model="calendarMonth" type="month" class="input month-input" @change="loadCalendarData" />
            <span v-if="calendarError" class="error">{{ calendarError }}</span>
          </div>
        </div>

        <div v-if="calendarLoading" class="empty-state"><p>加载中...</p></div>
        <div v-else class="report-panel calendar-panel">
          <h3 class="section-title">观察日历（{{ calendarTitle }}）</h3>
          <div class="calendar-weekdays">
            <div v-for="day in weekDays" :key="day" class="calendar-weekday">{{ day }}</div>
          </div>
          <div class="calendar-grid">
            <div
              v-for="cell in calendarCells"
              :key="cell.key"
              class="calendar-cell"
              :class="{ muted: !cell.inMonth, 'has-products': cell.products.length }"
            >
              <div class="calendar-day">{{ cell.day || '' }}</div>
              <div v-if="cell.products.length" class="calendar-products">
                <div v-for="product in cell.products" :key="product.id" class="calendar-product" :title="product.name">
                  {{ product.name || product.id }}
                </div>
              </div>
            </div>
          </div>
          <p class="table-summary">本月共 {{ calendarProductCount }} 个产品观察安排</p>
        </div>
      </div>

      <div v-if="activeTab === 'today'">
        <p class="desc">展示今日需要观察派息或敲出的存续产品。今日日期: {{ todayDate }}</p>
        <div v-if="todayLoading" class="empty-state"><p>加载中...</p></div>
        <div v-else-if="todayProducts.length" class="report-panel">
          <h3 class="section-title">今日观察（{{ todayDate }}）</h3>
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
        </div>
        <div v-else-if="todayLoaded" class="empty-state">
          <p>今日无产品需要观察派息/敲出。</p>
        </div>
      </div>

      <div v-if="activeTab === 'posters'">
        <p class="desc">自动生成敲出/派息喜报海报。可选择日期查询或生成对应日期的喜报。</p>

        <div class="panel">
          <h3 class="panel-title">喜报操作</h3>
          <div class="form-row">
            <label>筛选日期</label>
            <input v-model="filterDate" type="date" class="input" style="width: 160px; flex: none;" @change="loadPosters" />
          </div>
          <div class="form-row">
            <button class="btn btn-primary" :disabled="posterGenerating" @click="generatePosters">
              {{ posterGenerating ? '生成中...' : '生成喜报' }}
            </button>
            <button class="btn btn-secondary" @click="resetFilterDate">重置为今日</button>
            <span v-if="posterMsg" class="success">{{ posterMsg }}</span>
            <span v-if="posterError" class="error">{{ posterError }}</span>
          </div>
        </div>

        <div v-if="posterLoading" class="empty-state"><p>加载中...</p></div>
        <div v-else-if="posterList.length === 0 && posterLoaded" class="empty-state">
          <p>{{ filterDate }} 暂无喜报。可点击"生成喜报"为该日期生成。</p>
        </div>
        <div v-else-if="posterList.length > 0" class="report-panel">
          <h3 class="section-title">喜报（{{ filterDate }}）· 共 {{ posterList.length }} 张</h3>
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
        </div>
      </div>
    </div>
  </SubPageLayout>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import SubPageLayout from '../components/SubPageLayout.vue'
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
    const res = await fetch(`/api/observations/calendar?month=${calendarMonth.value}`)
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

const calendarTitle = computed(() => {
  const [year, month] = calendarMonth.value.split('-')
  return `${year}年${Number(month)}月`
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
</script>

<style scoped>
.desc { color: #6B5C4E; font-size: 14px; line-height: 1.8; margin-bottom: 24px; }

.tab-bar {
  display: flex;
  gap: 2px;
  margin-bottom: 20px;
  background: #fff;
  border: 1px solid #E8DDD0;
  border-radius: 8px;
  padding: 4px;
  width: fit-content;
}

.tab-btn {
  padding: 8px 20px;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  background: transparent;
  color: #6B5C4E;
  transition: all 0.15s;
}

.tab-btn:hover { background: #F5F0E8; }

.tab-btn.active {
  background: #D97757;
  color: #fff;
}

.panel {
  background: #fff;
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 20px;
  border: 1px solid #E8DDD0;
}

.panel-title { font-size: 15px; font-weight: 600; color: #1A1109; margin-bottom: 16px; }

.form-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.form-row > label:first-child {
  font-size: 13px;
  color: #6B5C4E;
  white-space: nowrap;
  width: 70px;
  flex-shrink: 0;
}

.input {
  flex: 1;
  border: 1px solid #E8DDD0;
  border-radius: 6px;
  padding: 8px 12px;
  font-size: 13px;
  outline: none;
  background: #fff;
  color: #1A1109;
}

.input:focus { border-color: #8B7355; }

.file-source { flex: 1; display: flex; align-items: center; gap: 10px; }
.file-badge { font-size: 13px; color: #1A1109; font-weight: 500; }
.file-from { font-size: 12px; color: #A8967E; background: #F5F0E8; padding: 2px 8px; border-radius: 10px; }
.month-input { width: 180px; flex: none; }

.btn { padding: 8px 20px; border-radius: 6px; font-size: 13px; cursor: pointer; border: none; font-weight: 500; margin-right: 8px; }
.btn-primary { background: #C62828; color: #fff; }
.btn-primary:hover:not(:disabled) { background: #B71C1C; }
.btn-primary:disabled { background: #EF9A9A; cursor: not-allowed; }
.btn-secondary { background: #8B7355; color: #fff; }
.btn-secondary:hover:not(:disabled) { background: #7A6348; }
.btn-secondary:disabled { background: #C4B5A5; cursor: not-allowed; }

.update-time { margin-left: 16px; color: #8B7355; font-size: 12px; }
.error { margin-left: 12px; color: #C62828; font-size: 13px; }
.success { margin-left: 12px; color: #2E7D45; font-size: 13px; }

.report-panel {
  background: #fff;
  border-radius: 12px;
  padding: 28px;
  border: 1px solid #E8DDD0;
}

.section-title {
  font-size: 15px;
  font-weight: 700;
  color: #1A1109;
  margin-bottom: 20px;
  padding-bottom: 12px;
  border-bottom: 2px solid #F0EAE0;
}

.table-wrap {
  overflow-x: auto;
  margin-bottom: 12px;
}

.overview-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
  min-width: 1400px;
}

.overview-table th {
  padding: 10px 12px;
  border-bottom: 1px solid #E8DDD0;
  color: #8B7355;
  font-weight: 600;
  background: #FAF7F4;
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
.data-row:hover { background: #FAF7F4; }

.overview-table td {
  padding: 11px 12px;
  border-bottom: 1px solid #F0EAE0;
  color: #1A1109;
  white-space: nowrap;
}

.col-left { text-align: left; }
.col-right { text-align: right; }
.col-center { text-align: center; }

.sticky-col {
  position: sticky;
  left: 0;
  background: #fff;
  z-index: 2;
}
.data-row:hover .sticky-col { background: #FAF7F4; }
.overview-table th.sticky-col { z-index: 3; background: #FAF7F4; }

.chevron {
  font-size: 14px;
  color: #A8967E;
  transition: transform 0.2s;
  display: inline-block;
  line-height: 1;
  margin-right: 4px;
}
.chevron.open { transform: rotate(90deg); }

.code-cell { font-family: monospace; font-size: 11px; color: #6B5C4E; }
.next-date { color: #C62828; font-weight: 600; }

.status-badge {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 10px;
  background: #E8F4EC;
  color: #2E7D45;
}

.result-yes-knockout {
  color: #C62828;
  font-weight: 600;
  background: #FEF3E2;
  border-radius: 4px;
  padding: 2px 6px;
}

.result-yes-dividend {
  color: #2E7D45;
  font-weight: 600;
  background: #E8F4EC;
  border-radius: 4px;
  padding: 2px 6px;
}

.result-no {
  color: #8B7355;
}

.result-na {
  color: #A8967E;
  font-style: italic;
  font-size: 11px;
}

.detail-row td {
  padding: 0;
  border-bottom: 1px solid #F0EAE0;
}

.detail-cell {
  background: #FAFAF8;
}

.detail-label {
  font-size: 11px;
  font-weight: 600;
  color: #8B7355;
  letter-spacing: 0.04em;
  padding: 12px 16px 8px;
}

.detail-empty {
  font-size: 12px;
  color: #A8967E;
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
  border-bottom: 1px solid #E8DDD0;
  color: #8B7355;
  font-weight: 600;
  background: transparent;
  text-align: left;
}

.detail-table td {
  padding: 6px 12px;
  border-bottom: 1px solid #F0EAE0;
  color: #3D3028;
}

.table-summary {
  font-size: 12px;
  color: #8B7355;
  text-align: right;
  padding-top: 8px;
}

.empty-state {
  text-align: center;
  padding: 48px 24px;
  color: #A8967E;
  font-size: 14px;
  background: #fff;
  border-radius: 12px;
  border: 1px solid #E8DDD0;
}

/* Workbench theme overrides */
.desc { color: var(--ink-soft); }

.tab-bar, .panel, .report-panel, .empty-state { border-color: var(--border-soft); border-radius: var(--radius); background: var(--surface); }
.tab-btn, .form-row > label:first-child, .file-from, .update-time, .table-summary,
.detail-label, .detail-empty, .code-cell, .result-no, .result-na { color: var(--ink-soft); }
.tab-btn:hover, .data-row:hover { background: var(--surface-muted); }
.tab-btn.active, .btn-primary { background: var(--brand); color: #fff; }
.btn-primary:hover:not(:disabled) { background: var(--brand-hover); }
.btn-secondary { border: 1px solid var(--border); color: var(--ink); background: #fff; }
.btn-secondary:hover:not(:disabled) { border-color: var(--brand); color: var(--brand); background: var(--brand-soft); }
.panel-title, .section-title, .file-badge, .overview-table td { color: var(--ink-strong); }
.input { border-color: var(--border); border-radius: var(--radius); color: var(--ink); }
.input:focus { border-color: var(--brand); box-shadow: 0 0 0 3px var(--brand-soft); }
.file-from, .overview-table th, .overview-table th.sticky-col, .detail-cell, .data-row:hover .sticky-col { background: var(--surface-muted); }
.sticky-col { background: var(--surface); }
.overview-table th, .overview-table td, .detail-row td, .detail-table th, .detail-table td { border-color: var(--border-soft); }
.status-badge, .result-yes-dividend { color: var(--success); background: var(--success-soft); }
.result-yes-knockout { color: var(--danger); background: var(--danger-soft); }

.calendar-panel { padding: 24px; }

.calendar-weekdays {
  display: grid;
  grid-template-columns: repeat(7, minmax(120px, 1fr));
  border: 1px solid var(--border-soft);
  border-bottom: none;
  background: var(--surface-muted);
}

.calendar-weekday {
  padding: 10px 12px;
  color: var(--ink-soft);
  font-size: 12px;
  font-weight: 700;
  text-align: center;
  border-right: 1px solid var(--border-soft);
}

.calendar-weekday:last-child { border-right: none; }

.calendar-grid {
  display: grid;
  grid-template-columns: repeat(7, minmax(120px, 1fr));
  border-left: 1px solid var(--border-soft);
  border-top: 1px solid var(--border-soft);
  overflow-x: auto;
}

.calendar-cell {
  min-height: 118px;
  padding: 10px;
  border-right: 1px solid var(--border-soft);
  border-bottom: 1px solid var(--border-soft);
  background: var(--surface);
}

.calendar-cell.muted { background: var(--surface-muted); }
.calendar-cell.has-products { background: var(--brand-soft); }

.calendar-day {
  height: 20px;
  color: var(--ink-strong);
  font-size: 12px;
  font-weight: 800;
  margin-bottom: 8px;
}

.calendar-products {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.calendar-product {
  padding: 4px 6px;
  border-radius: var(--radius);
  background: var(--brand-soft);
  color: var(--brand);
  font-size: 11px;
  line-height: 1.35;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.poster-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
  justify-content: center;
}

.poster-card {
  background: var(--surface);
  border-radius: var(--radius);
  padding: 20px;
  border: 1px solid var(--border-soft);
  box-shadow: none;
  transition: box-shadow 0.2s;
}

.poster-card:hover {
  box-shadow: var(--shadow-soft);
}

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
