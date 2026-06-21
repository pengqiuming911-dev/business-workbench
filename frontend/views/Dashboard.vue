<template>
  <div class="dashboard-page">
    <section class="hero-section">
      <div class="hero-content">
        <div class="hero-kicker">金融衍生品运营工作台</div>
        <h2 class="hero-title">衍选运营平台</h2>
        <p class="hero-desc">
          面向结构化产品运营的日常工作台。优先呈现今日需要处理的同步状态、观察任务和核心规模指标。
        </p>
        <div class="hero-actions">
          <RouterLink class="btn btn-primary" to="/data-preparation">同步数据</RouterLink>
          <RouterLink class="btn btn-outline" to="/product-completion">今日观察</RouterLink>
          <button class="btn btn-outline" @click="openDrawer">打开智能体</button>
        </div>
      </div>

      <div class="hero-indices">
        <article v-for="idx in indexItems" :key="idx.code" class="index-card">
          <div class="index-header">
            <div>
              <span class="index-name">{{ idx.name }}</span>
              <span class="index-code">{{ idx.code }}</span>
            </div>
            <span class="index-badge">市场</span>
          </div>
          <div class="index-main">
            <strong class="index-value">{{ idx.value ?? '--' }}</strong>
            <span
              class="index-change"
              :class="{
                up: idx.changePct > 0,
                down: idx.changePct < 0,
              }"
            >
              {{ idx.changePct !== null ? (idx.changePct > 0 ? '+' : '') + idx.changePct + '%' : '--' }}
            </span>
          </div>
          <div class="index-chart" :ref="el => setChartRef(el, idx.code)"></div>
        </article>
      </div>
    </section>

    <section class="stats-row">
      <article class="stat-card stat-card-primary" v-for="item in statCards" :key="item.label">
        <div class="stat-head">
          <span class="stat-label">{{ item.label }}</span>
          <span class="stat-trend">{{ item.trend }}</span>
        </div>
        <strong class="stat-value">{{ item.value }}</strong>
        <span class="stat-note">{{ item.note }}</span>
      </article>
    </section>

    <section class="content-grid">
      <div class="main-col">
        <div class="module-panel">
          <div class="card-head">
            <h3>常用入口</h3>
            <span class="card-link">{{ modules.length }} 项</span>
          </div>
          <div class="module-grid">
            <RouterLink v-for="item in modules" :key="item.path" :to="item.path" class="module-item">
              <span class="module-meta">{{ item.meta }}</span>
              <strong>{{ item.title }}</strong>
              <span>{{ item.desc }}</span>
            </RouterLink>
          </div>
        </div>
      </div>

      <div class="side-col">
        <div class="side-card sync-card">
          <div class="card-head">
            <h3>同步状态</h3>
            <RouterLink to="/data-preparation" class="card-link">管理</RouterLink>
          </div>
          <div class="sync-info">
            <div class="sync-row">
              <div class="sync-main">
                <span class="sync-dot" :class="{ ready: syncStatus.row_count }"></span>
                <strong>交易总表</strong>
              </div>
              <span class="sync-status">{{ syncStatus.row_count ? '已同步' : '未连接' }}</span>
            </div>
            <span class="sync-time">{{ syncStatus.synced_at ? formatTime(syncStatus.synced_at) : '尚未同步' }}</span>

            <div class="sync-row">
              <div class="sync-main">
                <span class="sync-dot" :class="{ ready: docStatus.doc_count }"></span>
                <strong>销售物料</strong>
              </div>
              <span class="sync-status">{{ docStatus.doc_count ? '已同步' : '未连接' }}</span>
            </div>
            <span class="sync-time">{{ docStatus.synced_at ? formatTime(docStatus.synced_at) : '尚未同步' }}</span>

            <div class="sync-row">
              <div class="sync-main">
                <span class="sync-dot" :class="{ ready: rebateStatus.row_count }"></span>
                <strong>返费明细</strong>
              </div>
              <span class="sync-status">{{ rebateStatus.row_count ? '已同步' : '未连接' }}</span>
            </div>
            <span class="sync-time">{{ rebateStatus.synced_at ? formatTime(rebateStatus.synced_at) : '尚未同步' }}</span>
          </div>
        </div>

        <div class="side-card focus-card">
          <div class="card-head">
            <h3>今日重点</h3>
          </div>
          <div class="quick-links">
            <RouterLink to="/product-completion" class="quick-link">
              <strong>观察日历</strong>
              <span>检查派息 / 敲出产品</span>
            </RouterLink>
            <RouterLink to="/product-report" class="quick-link">
              <strong>销售物料</strong>
              <span>查看月度物料与空状态</span>
            </RouterLink>
            <RouterLink to="/rebate-analysis" class="quick-link">
              <strong>返费复核</strong>
              <span>快速进入待返 / 已返分析</span>
            </RouterLink>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import * as echarts from 'echarts'

const stats = ref({
  totalProducts: 0,
  activeProducts: 0,
  totalCustomers: 0,
})
const syncStatus = ref({})
const docStatus = ref({})
const rebateStatus = ref({})

const indexItems = ref([
  { code: '000852.SH', name: '中证1000', value: null, changePct: null },
  { code: '513180.SH', name: '恒生科技ETF', value: null, changePct: null },
  { code: '000905.SH', name: '中证500', value: null, changePct: null },
  { code: '000300.SH', name: '沪深300', value: null, changePct: null },
  { code: '000001.SH', name: '上证指数', value: null, changePct: null },
  { code: '399006.SZ', name: '创业板指', value: null, changePct: null },
])

const statCards = computed(() => [
  { label: '产品总数', value: stats.value.totalProducts.toLocaleString('zh-CN'), note: '全部产品', trend: '规模' },
  { label: '运行中', value: stats.value.activeProducts.toLocaleString('zh-CN'), note: '用于今日观察与跟踪', trend: '运行中' },
])

const modules = [
  { path: '/product-completion', title: '派息 / 敲出', desc: '观察日历与记录生成', meta: '观察' },
  { path: '/product-report', title: '销售物料', desc: '按月份查看产品物料', meta: '物料' },
  { path: '/holding-analysis', title: '产品&持仓', desc: '产品和客户持仓分析', meta: '分析' },
  { path: '/rebate-analysis', title: '返费', desc: '待返与已返记录复核', meta: '结算' },
  { path: '/data-preparation', title: '数据准备', desc: '飞书同步与数据接入', meta: '数据' },
]

const chartRefs = {}
const chartInstances = new Map()
let indicesRefreshTimer = null

function setChartRef(el, code) {
  if (el) chartRefs[code] = el
}

function openDrawer() {
  window.dispatchEvent(new CustomEvent('toggle-agent-drawer'))
}

function formatTime(iso) {
  const d = new Date(iso)
  return d.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

async function loadStats() {
  try {
    const res = await fetch('/api/dashboard/stats')
    if (res.ok) stats.value = await res.json()
  } catch {}
}

async function loadSyncStatuses() {
  try {
    const res1 = await fetch('/api/db/sync-status')
    if (res1.ok) syncStatus.value = await res1.json()
    const res2 = await fetch('/api/drive/product-docs/sync-status')
    if (res2.ok) docStatus.value = await res2.json()
    const res3 = await fetch('/api/db/rebate-detail-status')
    if (res3.ok) rebateStatus.value = await res3.json()
  } catch {}
}

async function loadIndices() {
  try {
    const res = await fetch('/api/dashboard/indices')
    if (!res.ok) return
    const payload = await res.json()
    const results = new Map((payload.indices || []).map(item => [item.code, item]))
    for (const idx of indexItems.value) {
      const result = results.get(idx.code)
      if (!result) continue
      if (result.name) idx.name = result.name
      if (result.value != null) idx.value = result.value
      if (result.change_pct != null) idx.changePct = +result.change_pct.toFixed(2)
    }
  } catch {}
}

async function loadKlines() {
  try {
    const res = await fetch('/api/dashboard/klines')
    if (!res.ok) return
    const payload = await res.json()
    const results = new Map((payload.klines || []).map(item => [item.code, item]))
    for (const idx of indexItems.value) {
      const result = results.get(idx.code)
      if (!result?.klines?.length) continue
      const el = chartRefs[idx.code]
      if (!el) continue
      chartInstances.get(idx.code)?.dispose()
      const chart = echarts.init(el)
      chartInstances.set(idx.code, chart)
      const closes = result.klines.map(point => point.close)
      const isUp = closes[closes.length - 1] >= closes[0]
      const color = isUp ? '#b91c1c' : '#047857'
      chart.setOption({
        grid: { top: 2, bottom: 2, left: 2, right: 2 },
        xAxis: { type: 'category', show: false, boundaryGap: false },
        yAxis: { type: 'value', show: false, min: 'dataMin', max: 'dataMax' },
        series: [{
          type: 'line',
          data: closes,
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 1.75, color },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: color + '28' },
              { offset: 1, color: color + '04' },
            ]),
          },
        }],
      })
    }
  } catch {}
}

onMounted(async () => {
  await Promise.all([loadStats(), loadSyncStatuses(), loadIndices()])
  loadKlines()
  indicesRefreshTimer = window.setInterval(loadIndices, 30000)
})

onUnmounted(() => {
  if (indicesRefreshTimer) window.clearInterval(indicesRefreshTimer)
  chartInstances.forEach(chart => chart.dispose())
  chartInstances.clear()
})
</script>

<style scoped>
.dashboard-page {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.hero-section {
  display: grid;
  grid-template-columns: minmax(320px, 0.78fr) minmax(0, 1.4fr);
  gap: 24px;
  align-items: stretch;
  margin-bottom: 22px;
  padding: 28px;
  background:
    radial-gradient(circle at top left, rgba(44, 92, 224, 0.08), transparent 34%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.96), rgba(255, 255, 255, 0.9));
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
}

.hero-content {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 14px;
}

.hero-kicker {
  color: var(--brand);
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.hero-title {
  color: var(--ink-strong);
  font-size: 32px;
  font-weight: 800;
  line-height: 1.15;
  letter-spacing: -0.02em;
  max-width: 560px;
}

.hero-desc {
  color: var(--ink-soft);
  font-size: 14px;
  line-height: 1.75;
  max-width: 480px;
}

.hero-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 4px;
}

.hero-indices {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  align-self: stretch;
}

.index-card {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 8px;
  min-height: 132px;
  padding: 14px 16px;
  background: rgba(255, 255, 255, 0.96);
  border: 1px solid rgba(226, 232, 240, 0.9);
  border-radius: 14px;
  box-shadow: var(--shadow-sm);
}

.index-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
}

.index-name {
  display: block;
  font-size: 13px;
  font-weight: 700;
  color: var(--ink-strong);
}

.index-code {
  display: block;
  margin-top: 2px;
  font-size: 11px;
  color: var(--ink-faint);
  font-family: var(--font-mono);
}

.index-badge {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 0 8px;
  border-radius: 999px;
  background: var(--brand-soft);
  color: var(--brand);
  font-size: 10px;
  font-weight: 800;
}

.index-main {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
}

.index-value {
  font-size: 22px;
  font-weight: 800;
  color: var(--ink-strong);
  line-height: 1.1;
  font-family: var(--font-mono);
}

.index-change {
  font-size: 12px;
  font-weight: 700;
  color: var(--ink-faint);
}

.index-chart {
  width: 100%;
  height: 60px;
}

.index-change.up {
  color: var(--danger);
}

.index-change.down {
  color: var(--success);
}

.stats-row {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 18px;
}

.stat-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 18px 18px 16px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
}

.stat-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.stat-label {
  font-size: 13px;
  font-weight: 700;
  color: var(--ink-soft);
}

.stat-trend {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 0 8px;
  border-radius: 999px;
  background: #edf2f7;
  color: var(--ink-soft);
  font-size: 11px;
  font-weight: 800;
}

.stat-value {
  font-size: 30px;
  font-weight: 800;
  color: var(--ink-strong);
  line-height: 1.1;
  font-family: var(--font-mono);
}

.stat-note {
  font-size: 12px;
  color: var(--ink-faint);
}

.content-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 340px;
  gap: 18px;
}

.module-panel,
.side-card {
  padding: 20px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
}

.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 14px;
}

.card-head h3 {
  font-size: 16px;
  font-weight: 800;
  color: var(--ink-strong);
}

.card-link {
  font-size: 13px;
  color: var(--ink-soft);
  font-weight: 700;
}

.module-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.module-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-height: 120px;
  padding: 18px;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.98), rgba(248, 250, 252, 0.94));
  border: 1px solid var(--border-soft);
  border-radius: 14px;
  box-shadow: var(--shadow-sm);
  transition: transform 140ms ease, border-color 140ms ease, box-shadow 140ms ease;
}

.module-item:hover {
  transform: translateY(-1px);
  border-color: rgba(31, 58, 138, 0.12);
  box-shadow: var(--shadow-md);
}

.module-meta {
  color: var(--brand);
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.module-item strong {
  font-size: 15px;
  font-weight: 800;
  color: var(--ink-strong);
}

.module-item span:last-child {
  font-size: 12px;
  color: var(--ink-soft);
  line-height: 1.6;
}

.side-col {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.sync-info {
  display: grid;
  gap: 10px;
}

.sync-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.sync-main {
  display: flex;
  align-items: center;
  gap: 10px;
}

.sync-main strong {
  font-size: 14px;
  font-weight: 700;
  color: var(--ink-strong);
}

.sync-status,
.sync-time {
  font-size: 12px;
  color: var(--ink-soft);
}

.sync-time {
  padding-left: 18px;
}

.sync-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--ink-faint);
  flex-shrink: 0;
}

.sync-dot.ready {
  background: var(--success);
}

.quick-links {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.quick-link {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 14px 0;
  border-bottom: 1px solid var(--border-soft);
}

.quick-link:last-child {
  border-bottom: none;
}

.quick-link strong {
  font-size: 14px;
  font-weight: 800;
  color: var(--ink-strong);
}

.quick-link span {
  font-size: 12px;
  color: var(--ink-soft);
}

@media (max-width: 1180px) {
  .hero-section {
    grid-template-columns: 1fr;
  }

  .content-grid {
    grid-template-columns: 1fr;
  }

  .module-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 760px) {
  .hero-indices {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .hero-section {
    padding: 20px;
  }

  .hero-title {
    font-size: 26px;
  }

  .stats-row,
  .module-grid,
  .hero-indices {
    grid-template-columns: 1fr;
  }
}
</style>
