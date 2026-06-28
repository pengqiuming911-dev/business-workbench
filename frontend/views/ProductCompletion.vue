<template>
  <div class="product-completion-page">
    <div class="tab-bar">
      <button class="btn tab-btn" :class="{ active: activeTab === 'all' }" @click="activeTab = 'all'">全量</button>
      <button class="btn tab-btn" :class="{ active: activeTab === 'calendar' }" @click="activeTab = 'calendar'; loadCalendarData()">观察日历</button>
      <button class="btn tab-btn" :class="{ active: activeTab === 'today' }" @click="activeTab = 'today'; loadTodayData()">今日观察</button>
      <button class="btn tab-btn" :class="{ active: activeTab === 'copy' }" @click="activeTab = 'copy'; loadTodayData()">喜报文案</button>
    </div>

    <div v-if="activeTab === 'all'">
      <p class="text-body" style="margin-bottom: 24px;">展示存续产品（持有中）的派息与敲出观察情况。数据来源为航班服务交易总表 · 产品表。</p>

      <div class="compact-toolbar">
        <input v-model="searchText" type="text" class="input" placeholder="按产品名称或航班编号搜索..." style="max-width: 280px;" />
        <button class="btn btn-primary" :disabled="refreshing" @click="refreshPrices">
          {{ refreshing ? '刷新中...' : '刷新标的价格' }}
        </button>
        <button class="btn btn-secondary" :disabled="generating" @click="generateObservations">
          {{ generating ? '生成中...' : '生成观察记录' }}
        </button>
        <span v-if="lastUpdated" class="text-label" style="margin-left: 8px;">最后更新: {{ lastUpdated }}</span>
        <span v-if="errorMsg" class="error-msg" style="margin-left: 8px;">{{ errorMsg }}</span>
        <span v-if="successMsg" class="success-msg" style="margin-left: 8px;">{{ successMsg }}</span>
      </div>

      <div v-if="filteredProducts.length">
        <PanelCard title="存续产品观察概览">
          <template #header-actions>
            <FullscreenToggle target=".product-completion-page .all-table-section" />
          </template>
          <div class="table-section all-table-section">
            <div class="table-wrap">
            <table class="overview-table">
              <thead>
                <tr>
                  <th class="col-left sticky-col sticky-col-1">航班编号</th>
                  <th class="col-left sticky-col sticky-col-2">产品名称</th>
                  <th class="col-left">私募管理人</th>
                  <th class="col-left">持有状态</th>
                  <th class="col-left">代码</th>
                  <th class="col-left">结构类型</th>
                  <th class="col-left">入场日</th>
                  <th class="col-left">存续月</th>
                  <th class="col-left">锁定期(月)</th>
                  <th class="col-left">下个观察日</th>
                  <th class="col-left">入场价</th>
                  <th class="col-left">当前点位</th>
                  <th class="col-left">降敲</th>
                  <th class="col-left">敲出价</th>
                  <th class="col-left">派息线</th>
                  <th class="col-left">月票息（税费后）</th>
                  <th class="col-left">第一段票息（税费后）</th>
                  <th class="col-left">第二段票息（税费后）</th>
                  <th class="col-left">第三段票息（税费后）</th>
                  <th class="col-center th-sub">是否敲出<span>当前点位预测</span></th>
                  <th class="col-center th-sub">是否派息<span>当前点位预测</span></th>
                </tr>
              </thead>
              <tbody>
                <template v-for="p in filteredProducts" :key="p.id">
                  <tr class="data-row" @click="toggleExpand(p.id)">
                    <td class="col-left sticky-col sticky-col-1">
                      <span class="chevron" :class="{ open: expandedId === p.id }">›</span>
                      {{ p.id }}
                    </td>
                    <td class="col-left sticky-col sticky-col-2">{{ p.name }}</td>
                    <td class="col-left">{{ p.manager }}</td>
                    <td class="col-left"><span class="status-dot status-active">{{ p.holding_status }}</span></td>
                    <td class="col-left code-cell">{{ p.code }}</td>
                    <td class="col-left">{{ p.structure_type || '--' }}</td>
                    <td class="col-left">{{ p.issue_date || '--' }}</td>
                    <td class="col-left">{{ computeMonthsSince(p) }}</td>
                    <td class="col-left">{{ p.lock_months || '--' }}</td>
                    <td class="col-left next-date">{{ p.next_observation_date || '--' }}</td>
                    <td class="col-left">{{ formatPrice(p.entry_price, p) }}</td>
                    <td class="col-left">{{ formatPrice(latestObs(p)?.underlying_price, p) }}</td>
                    <td class="col-left">{{ p.monthly_decrease ?? '--' }}</td>
                    <td class="col-left">{{ formatPrice(latestObs(p)?.knockout_price, p) }}</td>
                    <td class="col-left">{{ formatPrice(latestObs(p)?.dividend_line, p) }}</td>
                    <td class="col-left">{{ p.monthly_coupon ?? '--' }}</td>
                    <td class="col-left">{{ p.coupon_1st ?? '--' }}</td>
                    <td class="col-left">{{ p.coupon_2nd ?? '--' }}</td>
                    <td class="col-left">{{ p.coupon_3rd ?? '--' }}</td>
                    <td class="col-center" :class="knockoutClass(latestObs(p)?.is_knocked_out)">
                      {{ latestObs(p)?.is_knocked_out || '--' }}
                    </td>
                    <td class="col-center" :class="dividendClass(!p.monthly_coupon ? '不观察' : latestObs(p)?.is_dividend)">
                      {{ !p.monthly_coupon ? '不观察' : (latestObs(p)?.is_dividend || '--') }}
                    </td>
                  </tr>
                  <tr v-if="expandedId === p.id && p.observations.length" class="detail-row">
                    <td colspan="21" class="detail-cell">
                      <div class="detail-label">历史观察日明细</div>
                      <table class="detail-table">
                        <thead>
                          <tr>
                            <th>观察日</th>
                            <th class="num">标的价格</th>
                            <th class="num">敲出价</th>
                            <th class="num">派息线</th>
                            <th>是否敲出</th>
                            <th>是否派息</th>
                          </tr>
                        </thead>
                        <tbody>
                          <tr v-for="(obs, obsIdx) in p.observations" :key="obs.date">
                            <td>{{ obs.date }}</td>
                            <td class="num">{{ obsIdx === p.observations.length - 1 ? '-' : formatPrice(obs.underlying_price, p) }}</td>
                            <td class="num">{{ formatPrice(obs.knockout_price, p) }}</td>
                            <td class="num">{{ formatPrice(obs.dividend_line, p) }}</td>
                            <td :class="knockoutClass(obs.is_knocked_out)">{{ obs.is_knocked_out }}</td>
                            <td :class="dividendClass(obs.is_dividend)">{{ obs.is_dividend }}</td>
                          </tr>
                        </tbody>
                      </table>
                    </td>
                  </tr>
                  <tr v-if="expandedId === p.id && !p.observations.length" class="detail-row">
                    <td colspan="21" class="detail-cell">
                      <div class="detail-empty">暂无观察日记录</div>
                    </td>
                  </tr>
                </template>
              </tbody>
            </table>
            </div>
          <p class="table-summary">共 {{ filteredProducts.length }} 个存续产品</p>
          </div>
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
                  <div v-if="product.is_knockout_observable && product.knockout_price != null"
                       class="cal-detail-block cal-detail-knockout-spot">
                    <div class="cal-detail-row">
                      <span class="cal-detail-label">敲出</span>
                      <strong>{{ fmtCalPrice(product.knockout_price) }}</strong>
                    </div>
                  </div>
                  <div v-if="product.has_dividend_observation && product.dividend_line != null" class="cal-detail-row cal-detail-dividend">
                    <span class="cal-detail-label">派息</span>
                    <strong>{{ fmtCalPrice(product.dividend_line) }}</strong>
                  </div>
                  <div v-if="product.spot_price != null" class="cal-detail-row cal-spot-row">
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
          <div class="table-section today-table-section">
            <div class="table-toolbar">
              <FullscreenToggle target=".product-completion-page .today-table-section" />
            </div>
            <div class="table-wrap">
            <table class="overview-table">
              <thead>
                <tr>
                  <th class="col-left sticky-col">航班编号</th>
                  <th class="col-left">产品名称</th>
                  <th class="col-left">私募管理人</th>
                  <th class="col-left">代码</th>
                  <th class="num">入场价</th>
                  <th class="num">存续月</th>
                  <th class="num">标的价格</th>
                  <th class="num">敲出价</th>
                  <th class="num">派息线</th>
                  <th class="col-center">是否敲出</th>
                  <th class="col-center">是否派息</th>
                  <th class="col-center">喜报文案</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="p in todayProducts" :key="p.id" class="data-row">
                  <td class="col-left sticky-col">{{ p.id }}</td>
                  <td class="col-left">{{ p.name }}</td>
                  <td class="col-left">{{ p.manager }}</td>
                  <td class="col-left code-cell">{{ p.code }}</td>
                  <td class="num">{{ formatPrice(p.entry_price, p) }}</td>
                  <td class="num">{{ computeMonthsSince(p) }}</td>
                  <td class="num">{{ formatPrice(todayObs(p)?.underlying_price, p) }}</td>
                  <td class="num">{{ formatPrice(todayObs(p)?.knockout_price, p) }}</td>
                  <td class="num">{{ formatPrice(todayObs(p)?.dividend_line, p) }}</td>
                  <td class="col-center" :class="knockoutClass(todayObs(p)?.is_knocked_out)">
                    {{ todayObs(p)?.is_knocked_out || '--' }}
                  </td>
                  <td class="col-center" :class="dividendClass(todayObs(p)?.is_dividend)">
                    {{ todayObs(p)?.is_dividend || '--' }}
                  </td>
                  <td class="col-center">
                    <button
                      class="btn btn-sm btn-secondary"
                      :disabled="!eventType(p)"
                      @click.stop="openCopywriter(p)"
                    >
                      生成文案
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
            </div>
          <p class="table-summary">今日共 {{ todayProducts.length }} 个产品需观察</p>
          </div>
        </PanelCard>
      </div>
      <div v-else-if="todayLoaded" class="empty-state">
        <p>今日无产品需要观察派息/敲出。</p>
      </div>
    </div>

    <div v-if="activeTab === 'copy'" class="copywriter-page">
      <div class="copywriter-intro">
        <div>
          <h2 class="text-section">喜报文案</h2>
          <p class="text-body">选择今日已触发敲出或派息的产品，生成可直接发送的通知文案和喜报内容。</p>
        </div>
        <button class="btn btn-secondary" :disabled="todayLoading" @click="loadTodayData">
          {{ todayLoading ? '刷新中...' : '刷新今日观察' }}
        </button>
      </div>

      <div v-if="todayLoading" class="loading-state"><p>加载中...</p></div>
      <div v-else class="copywriter-layout">
        <PanelCard title="今日触发产品">
          <div v-if="copyCandidates.length" class="copy-product-list">
            <button
              v-for="p in copyCandidates"
              :key="p.id"
              class="copy-product-item"
              :class="{ active: selectedCopyProduct?.id === p.id }"
              @click="selectCopyProduct(p)"
            >
              <span class="copy-product-main">
                <strong>{{ p.name || p.id }}</strong>
                <span>{{ p.code || '--' }}</span>
              </span>
              <span class="badge" :class="eventType(p) === 'knockout' ? 'badge-red' : 'badge-green'">
                {{ eventType(p) === 'knockout' ? '敲出' : '派息' }}
              </span>
            </button>
          </div>
          <div v-else class="copy-empty">今日暂无已触发敲出或派息的产品。</div>
        </PanelCard>

        <PanelCard title="文案参数">
          <div v-if="selectedCopyProduct" class="copy-settings">
            <div class="form-row">
              <label>文案类型</label>
              <select v-model="copyEventType" class="input">
                <option value="knockout" :disabled="todayObs(selectedCopyProduct)?.is_knocked_out !== '是'">敲出</option>
                <option value="dividend" :disabled="todayObs(selectedCopyProduct)?.is_dividend !== '是'">派息</option>
              </select>
            </div>
            <div class="form-row">
              <label>T+天数</label>
              <input v-model.number="copyTDays" type="number" min="0" class="input" />
            </div>
            <div class="form-row">
              <label>到账日期</label>
              <input v-model="copyArrivalDate" type="date" class="input" />
            </div>
            <div class="form-row">
              <label>到账说明</label>
              <input v-model="copyArrivalNote" type="text" class="input" placeholder="例如：本周五 / 下周一" />
            </div>
          </div>
          <div v-else class="copy-empty">请先选择一个触发产品。</div>
        </PanelCard>

        <PanelCard title="通知文案">
          <template #header-actions>
            <button class="btn btn-sm btn-secondary" :disabled="!notificationCopy" @click="copyText(notificationCopy)">复制</button>
          </template>
          <textarea class="textarea copy-textarea" readonly :value="notificationCopy"></textarea>
        </PanelCard>

        <PanelCard title="喜报内容">
          <template #header-actions>
            <button class="btn btn-sm btn-secondary" :disabled="!posterCopy" @click="copyText(posterCopy)">复制</button>
          </template>
          <textarea class="textarea copy-textarea" readonly :value="posterCopy"></textarea>
        </PanelCard>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import PanelCard from '../components/PanelCard.vue'
import FullscreenToggle from '../components/FullscreenToggle.vue'

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
const selectedCopyProduct = ref(null)
const copyEventType = ref('knockout')
const copyTDays = ref(4)
const copyArrivalDate = ref('')
const copyArrivalNote = ref('')

const calendarMonth = ref(new Date().toISOString().slice(0, 7))
const calendarStatus = ref('ongoing')
const calendarItems = ref([])
const calendarLoading = ref(false)
const calendarLoaded = ref(false)
const calendarError = ref('')
const weekDays = ['一', '二', '三', '四', '五', '六', '日']

onMounted(() => loadData())

watch(copyTDays, value => {
  if (!selectedCopyProduct.value) return
  copyArrivalDate.value = addBusinessDays(todayDate.value, value)
  copyArrivalNote.value = weekdayNote(copyArrivalDate.value)
})

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
    todayProducts.value = (data.products || []).filter(p => !p.name || !p.name.includes('【专】'))
    todayDate.value = data.today || todayDate.value
    if (!selectedCopyProduct.value && copyCandidates.value.length) {
      selectCopyProduct(copyCandidates.value[0])
    }
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

const filteredProducts = computed(() => {
  let list = products.value.filter(p => !p.name || !p.name.includes('【专】'))
  if (searchText.value) {
    const q = searchText.value.toLowerCase()
    list = list.filter(p =>
      (p.name && p.name.toLowerCase().includes(q)) || p.id.toLowerCase().includes(q)
    )
  }
  return list
})

const calendarMap = computed(() => {
  const map = new Map()
  for (const item of calendarItems.value) {
    const filtered = (item.products || []).filter(p => !p.name || !p.name.includes('【专】'))
    map.set(item.date, filtered)
  }
  return map
})

const calendarProductCount = computed(() => (
  Array.from(calendarMap.value.values()).reduce((sum, products) => sum + products.length, 0)
))

const copyCandidates = computed(() => (
  todayProducts.value.filter(p => eventType(p))
))

const notificationCopy = computed(() => {
  if (!selectedCopyProduct.value) return ''
  return buildNotificationCopy(selectedCopyProduct.value, copyEventType.value)
})

const posterCopy = computed(() => {
  if (!selectedCopyProduct.value) return ''
  return buildPosterCopy(selectedCopyProduct.value, copyEventType.value)
})

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

function eventType(product) {
  const obs = todayObs(product)
  if (obs?.is_knocked_out === '是') return 'knockout'
  if (obs?.is_dividend === '是') return 'dividend'
  return ''
}

function openCopywriter(product) {
  selectCopyProduct(product)
  activeTab.value = 'copy'
}

function selectCopyProduct(product) {
  selectedCopyProduct.value = product
  copyEventType.value = eventType(product) || 'knockout'
  copyTDays.value = copyEventType.value === 'knockout' ? 4 : 3
  copyArrivalDate.value = addBusinessDays(todayDate.value, copyTDays.value)
  copyArrivalNote.value = weekdayNote(copyArrivalDate.value)
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

function formatPlainPrice(val, product) {
  if (val === null || val === undefined) return '--'
  const decimals = isETF(product) ? 3 : 2
  return Number(val).toLocaleString('zh-CN', { minimumFractionDigits: decimals, maximumFractionDigits: decimals, useGrouping: false })
}

function formatPercent(value, digits = 0) {
  if (value === null || value === undefined || Number.isNaN(Number(value))) return '--'
  const ratio = Math.abs(Number(value)) > 2 ? Number(value) / 100 : Number(value)
  return `${(ratio * 100).toFixed(digits)}%`
}

function underlyingName(product) {
  const code = product?.code || ''
  return code.split(/[（(]/)[0].trim() || '--'
}

function productName(product) {
  return product?.name || product?.id || '--'
}

function maskedProductName(product) {
  const name = productName(product)
  const match = name.match(/^(.).*?(\d+号(?:[（(][^)）]+[)）])?)/)
  if (match) return `${match[1]}*${match[2]}`
  return name.length > 1 ? `${name[0]}*${name.slice(-1)}` : name
}

function underlyingIndexName(product) {
  const name = underlyingName(product)
  if (!name || name === '--') return '--'
  if (name.includes('指数') || name.includes('ETF')) return name
  return `${name}指数`
}

function formatChineseDate(dateStr, withYear = true) {
  const date = parseLocalDate(dateStr)
  if (!date) return dateStr || '--'
  const year = withYear ? `${date.getFullYear()}年` : ''
  return `${year}${date.getMonth() + 1}月${date.getDate()}日`
}

function arrivalText() {
  const dateText = formatChineseDate(copyArrivalDate.value, false)
  return copyArrivalNote.value ? `${dateText}（${copyArrivalNote.value}）` : dateText
}

function knockoutLinePercent(product, obs) {
  if (product.entry_price && obs?.knockout_price) {
    return formatPercent(Number(obs.knockout_price) / Number(product.entry_price), 0)
  }
  return formatPercent(product.first_knockout_ratio, 0)
}

function currentKnockoutPercent(product, obs, digits = 0) {
  if (product.entry_price && obs?.knockout_price) {
    return formatPercent(Number(obs.knockout_price) / Number(product.entry_price), digits)
  }
  return formatPercent(product.first_knockout_ratio, digits)
}

function barrierPercent(raw, digits = 0) {
  if (raw === null || raw === undefined || raw === '') return '--'
  const text = String(raw)
  const match = text.match(/-?\d+(?:\.\d+)?/)
  if (!match) return '--'
  return formatPercent(Number(match[0]), digits)
}

function eventMonths(product, obs) {
  const months = Number(obs?.months_since_entry)
  if (Number.isFinite(months) && months > 0) return Math.round(months)
  const issue = parseLocalDate(product.issue_date)
  const observe = parseLocalDate(obs?.date || todayDate.value)
  if (!issue || !observe) return 0
  return Math.max(0, (observe.getFullYear() - issue.getFullYear()) * 12 + observe.getMonth() - issue.getMonth())
}

function annualizedRate(product) {
  if (product.monthly_coupon) return normalizeRatio(product.monthly_coupon) * 12
  const months = Number(product.duration_months)
  if (months > 0 && months <= 12 && product.coupon_1st) return normalizeRatio(product.coupon_1st)
  if (months > 12 && product.coupon_2nd) return normalizeRatio(product.coupon_2nd)
  if (product.coupon_1st) return normalizeRatio(product.coupon_1st)
  if (product.coupon_2nd) return normalizeRatio(product.coupon_2nd)
  return 0
}

function normalizeRatio(value) {
  const number = Number(value)
  if (!Number.isFinite(number)) return 0
  return Math.abs(number) > 2 ? number / 100 : number
}

function formatRateNumber(rate, digits = 2) {
  return (rate * 100).toFixed(digits)
}

function dividendCount(product, obs) {
  return Math.max(0, eventMonths(product, obs))
}

function priceCompareText(obs, targetPrice, product) {
  const close = Number(obs?.underlying_price)
  const target = Number(targetPrice)
  if (!Number.isFinite(close) || !Number.isFinite(target)) return '达到观察条件'
  return close >= target
    ? `大于${copyEventType.value === 'knockout' ? '敲出价' : '派息价'}`
    : `低于${copyEventType.value === 'knockout' ? '敲出价' : '派息价'}`
}

function buildNotificationCopy(product, type) {
  const obs = todayObs(product)
  if (!obs) return ''
  const name = productName(product)
  const closePrice = formatPlainPrice(obs.underlying_price, product)
  if (type === 'knockout') {
    const koPrice = formatPlainPrice(obs.knockout_price, product)
    return [
      '各位投资人大家好：',
      '',
      `【${name}】于${formatChineseDate(product.issue_date)}进场`,
      '',
      `✅️挂钩标的：【${underlyingName(product)}】`,
      `✅️入场价：【${formatPlainPrice(product.entry_price, product)}】`,
      `✅️敲出观察线：【${knockoutLinePercent(product, obs)}】，对应敲出线【${koPrice}】`,
      `✅️今天收盘价：【${closePrice}】${priceCompareText(obs, obs.knockout_price, product)}，触发敲出事件。`,
      '',
      `资金将于T+【${copyTDays.value}】交易日，也就是${arrivalText()}到账。`,
      '',
      '恭喜各位投资人！[庆祝][庆祝][庆祝]',
    ].join('\n')
  }
  const dividendLine = formatPlainPrice(obs.dividend_line, product)
  return [
    '各位投资者大家好！',
    '',
    `【${name}】于${formatChineseDate(product.issue_date)}进场`,
    `✅️挂钩标的：【${underlyingName(product)}】`,
    `✅️派息观察线：【${formatPercent(product.dividend_barrier, 0)}】，对应派息线【${dividendLine}】`,
    `✅️今天收盘价：【${closePrice}】${priceCompareText(obs, obs.dividend_line, product)}，触发派息分红事件。`,
    '',
    `分红将于T+【${copyTDays.value}】日，也就是${arrivalText()}到账。`,
    '',
    '恭喜各位投资人！[庆祝][庆祝]',
  ].join('\n')
}

function buildPosterCopy(product, type) {
  const obs = todayObs(product)
  if (!obs) return ''
  const maskedName = maskedProductName(product)
  const observeDate = formatChineseDate(obs.date || todayDate.value)
  const entryDate = formatChineseDate(product.issue_date)
  if (type === 'knockout') {
    const months = eventMonths(product, obs)
    const annualized = annualizedRate(product)
    const absoluteReturn = annualized / 12 * months
    return `帮我把这个图片里的“致*4号”改为“${maskedName}”；把“2026年3月2日”改成“${observeDate}”；把“敲出绝对收益12.03%”中的“12.03”改为“${formatRateNumber(absoluteReturn)}”，把“年化收益48.10%”中的“48.10”改为“${formatRateNumber(annualized)}”；把“存续时间3月”中的“3”改成“${months}”，把“中证1000指数”改成“${underlyingIndexName(product)}”；把“下跌保护界限：70%”中的“70”改成“${barrierPercent(product.parachute).replace('%', '')}”；把“止盈界限:101%”中的“101”改成“${currentKnockoutPercent(product, obs).replace('%', '')}”；把“2025年11月 28日”改成“${entryDate}”`
  }
  const monthlyCoupon = normalizeRatio(product.monthly_coupon)
  const count = dividendCount(product, obs)
  const cumulative = monthlyCoupon * count
  return `这张图帮我把“鹿*8号（三期）”改为“${maskedName}”；把“2026年4月30日”改成“${observeDate}”；把15.96%改成“${formatRateNumber(monthlyCoupon * 12)}%”；把“分红3次”改成“分红${count}次”；把“3.99%”改成“${formatRateNumber(cumulative)}%”；把“1.33%”改为“${formatRateNumber(monthlyCoupon)}%”；把“恒科ETF指数”改成“${underlyingIndexName(product)}”；把“80%”改成“${formatPercent(product.dividend_barrier, 0)}”；把“102%”改成“${currentKnockoutPercent(product, obs)}”；把“75%”改成“${barrierPercent(product.parachute)}”；把“2026年1月30日”改成“${entryDate}”`
}

async function copyText(text) {
  if (!text) return
  await navigator.clipboard.writeText(text)
  successMsg.value = '文案已复制'
}

function parseLocalDate(dateStr) {
  if (!dateStr) return null
  const [year, month, day] = dateStr.split('-').map(Number)
  if (!year || !month || !day) return null
  return new Date(year, month - 1, day)
}

function toDateInput(date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

function addBusinessDays(dateStr, days) {
  const date = parseLocalDate(dateStr) || new Date()
  let remaining = Number(days) || 0
  while (remaining > 0) {
    date.setDate(date.getDate() + 1)
    const day = date.getDay()
    if (day !== 0 && day !== 6) remaining -= 1
  }
  return toDateInput(date)
}

function weekdayNote(dateStr) {
  const date = parseLocalDate(dateStr)
  if (!date) return ''
  return `周${['日', '一', '二', '三', '四', '五', '六'][date.getDay()]}`
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
  if (status === '不观察') return 'result-na'
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

.product-completion-page {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.compact-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 16px;
}

.product-completion-page > .page-header {
  flex-shrink: 0;
}

.product-completion-page > .tab-bar {
  flex-shrink: 0;
}

.panel-card .table-wrap {
  flex: 1;
  min-height: 0;
  max-height: 65vh;
}

.table-toolbar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 12px;
}

.table-section.is-fullscreen .table-wrap {
  max-height: none;
}

.table-section.is-fullscreen .table-summary {
  margin: 0;
  padding: 10px 16px;
  text-align: left;
  background: #fff;
  border-top: 1px solid var(--border-soft);
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
  font-size: 13px;
  min-width: 1300px;
}

.overview-table th {
  padding: 10px 12px;
  border-bottom: 2px solid var(--border);
  color: var(--ink-strong);
  font-weight: 800;
  font-size: 11px;
  letter-spacing: 0.04em;
  white-space: nowrap;
  position: sticky;
  top: 0;
  z-index: 5;
  background: #fef9ee;
}

.overview-table th.th-sub {
  white-space: normal;
  line-height: 1.2;
  vertical-align: bottom;
  min-width: 80px;
  text-align: center;
}

.overview-table th.th-sub span {
  display: block;
  font-size: 9px;
  font-weight: 600;
  color: var(--ink-faint);
  letter-spacing: 0;
  margin-top: 2px;
}

.data-row {
  cursor: pointer;
  transition: background 0.15s;
}
.data-row:hover { background: var(--surface-muted); }

.overview-table td {
  padding: 7px 12px;
  border-bottom: 1px solid #f0f2f5;
  color: var(--ink-strong);
  white-space: nowrap;
}

.overview-table tbody tr:nth-child(even) td {
  background: var(--bg-zebra);
}

.overview-table tr:hover td {
  background: #eef2f7;
}

.col-left { text-align: left; }
.col-center { text-align: left; }

.sticky-col {
  position: sticky;
  background: var(--bg-card);
  z-index: 3;
}
.sticky-col-1 { left: 0; }
.sticky-col-2 { left: 120px; box-shadow: 4px 0 8px -4px rgba(0,0,0,0.12); }
.overview-table tbody tr:nth-child(even) .sticky-col {
  background: var(--bg-zebra);
}
.data-row:hover .sticky-col { background: #eef2f7; }
.overview-table th.sticky-col { z-index: 6; background: #fef9ee; }

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
  padding: 8px 12px 4px;
}

.detail-empty {
  font-size: 12px;
  color: var(--ink-soft);
  padding: 8px 12px;
}

.detail-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 11px;
  margin: 0 12px 8px;
}

.detail-table th {
  padding: 3px 10px;
  border-bottom: 1px solid var(--border-soft);
  color: var(--ink-soft);
  font-weight: 800;
  font-size: 10px;
  letter-spacing: 0.04em;
  background: transparent;
  text-align: left;
}

.detail-table th.num {
  text-align: right;
}

.detail-table td {
  padding: 2px 10px;
  border-bottom: 1px solid var(--border-soft);
  color: var(--ink);
}

.detail-table td.num {
  text-align: right;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 11px;
}

.detail-table tbody tr:nth-child(even) td {
  background: rgba(0, 0, 0, 0.015);
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
  padding: 8px 10px;
  border-radius: 6px;
  border: 1px solid #e2e8f0;
  font-size: 11px;
  line-height: 1.45;
  background: #fff;
  transition: border-color 150ms ease, box-shadow 150ms ease;
}

.cal-card:hover {
  border-color: #c7d6e8;
  box-shadow: 0 2px 8px rgba(37, 99, 235, 0.06);
}

.cal-card-name {
  font-weight: 700;
  color: var(--ink-strong);
  font-size: 12px;
  line-height: 1.4;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  padding-bottom: 5px;
}

.cal-card-details {
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding-top: 5px;
  border-top: 1px solid #f0f3f7;
}

.cal-card-details:empty {
  display: none;
}

.cal-detail-row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
  min-height: 20px;
  padding: 2px 7px 2px 9px;
  border-left: 3px solid transparent;
  border-radius: 0 5px 5px 0;
  font-size: 11px;
}

.cal-detail-block {
  border-left: 3px solid transparent;
  border-radius: 0 5px 5px 0;
  padding: 2px 0;
}

.cal-detail-block .cal-detail-row {
  border-left: none;
  border-radius: 0;
  padding: 2px 7px 2px 6px;
}

.cal-detail-label {
  white-space: nowrap;
  font-weight: 600;
  min-width: 2em;
  display: inline-block;
}

.cal-detail-row strong {
  font-weight: 700;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.cal-detail-knockout-spot {
  border-left-color: #2563a8;
  background: #eef4ff;
  color: #2563a8;
}

.cal-detail-knockout-spot .cal-detail-row strong {
  color: #1a4f8a;
}

.cal-spot-row {
  border-left: 3px solid #7c6baa;
  background: #f1edfb;
  color: #6b5b95;
  border-radius: 0 5px 5px 0;
  padding: 2px 7px 2px 6px;
}

.cal-spot-row .cal-detail-label {
  color: #6b5b95;
}

.cal-spot-row strong {
  color: #4c3f73;
}


.cal-detail-dividend {
  border-left-color: #0d9668;
  background: #eafaf3;
  color: #0d9668;
}

.cal-detail-dividend strong {
  color: #0a7a54;
}

.copywriter-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.copywriter-intro {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 16px 18px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
}

.copywriter-layout {
  display: grid;
  grid-template-columns: minmax(260px, 0.8fr) minmax(260px, 0.8fr);
  gap: 16px;
  align-items: start;
}

.copywriter-layout .panel-card:nth-child(n + 3) {
  grid-column: span 1;
}

.copy-product-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.copy-product-item {
  width: 100%;
  min-height: 58px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  color: var(--ink);
  background: #fff;
  text-align: left;
  transition: background 120ms ease, border-color 120ms ease, box-shadow 120ms ease;
}

.copy-product-item:hover,
.copy-product-item.active {
  border-color: rgba(44, 92, 224, 0.28);
  background: #f8fbff;
  box-shadow: var(--shadow-sm);
}

.copy-product-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.copy-product-main strong,
.copy-product-main span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.copy-product-main strong {
  color: var(--ink-strong);
  font-size: 13px;
}

.copy-product-main span {
  color: var(--ink-soft);
  font-size: 12px;
}

.copy-settings {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.copy-textarea {
  min-height: 320px;
  line-height: 1.75;
  white-space: pre-wrap;
}

.copy-empty {
  padding: 28px 12px;
  color: var(--ink-soft);
  font-size: 13px;
  text-align: center;
}

@media (max-width: 980px) {
  .copywriter-layout {
    grid-template-columns: 1fr;
  }

  .copywriter-layout .panel-card:nth-child(n + 3) {
    grid-column: auto;
  }
}

@media (max-width: 720px) {
  .copywriter-intro {
    flex-direction: column;
  }

  .copywriter-intro .btn {
    width: 100%;
  }
}
</style>
