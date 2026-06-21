<template>
  <div class="dashboard-page">
    <!-- Hero -->
    <section class="hero-section">
      <div class="hero-content">
        <h2 class="hero-title">业务工作台</h2>
        <p class="hero-desc">集中处理数据同步、产品观察、报告输出和客户分析。</p>
        <div class="hero-actions">
          <RouterLink class="btn btn-primary" to="/data-preparation">同步数据</RouterLink>
          <RouterLink class="btn btn-outline" to="/product-completion">今日观察</RouterLink>
          <button class="btn btn-outline" @click="openDrawer">智能助手</button>
        </div>
      </div>
      <div class="hero-indices">
        <article
          v-for="idx in indexItems"
          :key="idx.code"
          class="index-card"
        >
          <div class="index-header">
            <span class="index-name">{{ idx.name }}</span>
            <span class="index-code">{{ idx.code }}</span>
          </div>
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
          <div class="index-chart" :ref="el => setChartRef(el, idx.code)"></div>
        </article>
      </div>
    </section>

    <!-- Stats Row -->
    <section class="stats-row">
      <article class="stat-card" v-for="item in statCards" :key="item.label">
        <span class="stat-label">{{ item.label }}</span>
        <strong class="stat-value">{{ item.value }}</strong>
        <span class="stat-note">{{ item.note }}</span>
      </article>
    </section>

    <!-- Content Grid -->
    <section class="content-grid">
      <div class="main-col">
        <div class="card-head">
          <h3>常用入口</h3>
          <span class="card-link">{{ modules.length }} 项</span>
        </div>
        <div class="module-grid">
          <RouterLink
            v-for="item in modules"
            :key="item.path"
            :to="item.path"
            class="module-item"
          >
            <strong>{{ item.title }}</strong>
            <span>{{ item.desc }}</span>
          </RouterLink>
        </div>
      </div>

      <div class="side-col">
        <div class="side-card">
          <div class="card-head">
            <h3>同步状态</h3>
            <RouterLink to="/data-preparation" class="card-link">管理</RouterLink>
          </div>
          <div class="sync-info">
            <div class="sync-row">
              <span class="sync-dot" :class="{ ready: syncStatus.row_count }"></span>
              <strong>交易总表</strong>
              <span>{{ syncStatus.synced_at ? formatTime(syncStatus.synced_at) : '未同步' }}</span>
            </div>
            <div class="sync-row">
              <span class="sync-dot" :class="{ ready: docStatus.doc_count }"></span>
              <strong>物料文档</strong>
              <span>{{ docStatus.synced_at ? formatTime(docStatus.synced_at) : '未同步' }}</span>
            </div>
            <div class="sync-row">
              <span class="sync-dot" :class="{ ready: rebateStatus.row_count }"></span>
              <strong>返费明细</strong>
              <span>{{ rebateStatus.synced_at ? formatTime(rebateStatus.synced_at) : '未同步' }}</span>
            </div>
          </div>
        </div>

        <div class="side-card">
          <div class="card-head">
            <h3>今日重点</h3>
          </div>
          <div class="quick-links">
            <RouterLink to="/product-completion" class="quick-link">观察日历</RouterLink>
            <RouterLink to="/product-report" class="quick-link">销售物料</RouterLink>
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
  { code: '513180.SH', name: '恒科ETF', value: null, changePct: null },
  { code: '000905.SH', name: '中证500', value: null, changePct: null },
  { code: '000300.SH', name: '沪深300', value: null, changePct: null },
  { code: '000001.SH', name: '上证指数', value: null, changePct: null },
  { code: '399006.SZ', name: '创业板指', value: null, changePct: null },
])

const statCards = computed(() => [
  { label: '产品总数', value: stats.value.totalProducts.toLocaleString('zh-CN'), note: '全部产品' },
  { label: '存续产品', value: stats.value.activeProducts.toLocaleString('zh-CN'), note: '持有中' },
  { label: '客户数量', value: stats.value.totalCustomers.toLocaleString('zh-CN'), note: '已登记' },
])

const modules = [
  { path: '/data-preparation', title: '数据准备', desc: '飞书同步' },
  { path: '/product-completion', title: '观察日历', desc: '敲出 / 派息' },
  { path: '/product-report', title: '销售物料', desc: '产品物料' },
  { path: '/holding-analysis', title: '产品&持仓', desc: '产品与客户持有' },
  { path: '/rebate-analysis', title: '返费', desc: '待返费与已返费' },
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
      const color = isUp ? '#dc2626' : '#16a34a'
      chart.setOption({
        grid: { top: 2, bottom: 2, left: 2, right: 2 },
        xAxis: { type: 'category', show: false, boundaryGap: false },
        yAxis: { type: 'value', show: false, min: 'dataMin', max: 'dataMax' },
        series: [{
          type: 'line',
          data: closes,
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 1.5, color },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: color + '30' },
              { offset: 1, color: color + '05' },
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
  padding-top: 8px;
}

/* ─── Hero ─── */
.hero-section {
  display: grid;
  grid-template-columns: minmax(280px, 0.7fr) minmax(0, 1.5fr);
  gap: 22px;
  align-items: center;
  margin-bottom: 22px;
  padding: 22px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  box-shadow: var(--shadow-sm);
}

.hero-content {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.hero-title {
  color: var(--ink-strong);
  font-size: 24px;
  font-weight: 750;
  line-height: 1.15;
}

.hero-desc {
  color: var(--ink-soft);
  font-size: 14px;
  line-height: 1.6;
  max-width: 400px;
}

.hero-actions {
  display: flex;
  gap: 8px;
  margin-top: 4px;
}

.hero-indices {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  padding: 6px;
  background: var(--bg-hover, #f9fafb);
  border-radius: var(--radius);
}

.hero-indices .index-card {
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 12px 14px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  box-shadow: var(--shadow-sm);
}

.index-header {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.index-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--ink-soft);
}

.index-code {
  font-size: 11px;
  color: var(--ink-faint);
  font-family: var(--font-mono);
}

.index-value {
  font-size: 17px;
  font-weight: 750;
  color: var(--ink-strong);
  line-height: 1.2;
  font-family: var(--font-mono);
}

.index-change {
  font-size: 12px;
  font-weight: 600;
  color: var(--ink-faint);
}

.index-chart {
  width: 100%;
  height: 48px;
  margin-top: 4px;
}

.index-change.up {
  color: #dc2626;
}

.index-change.down {
  color: #16a34a;
}

/* ─── Stats ─── */
.stats-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 16px;
}

.stat-card {
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 16px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  box-shadow: var(--shadow-sm);
}

.stat-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--ink-soft);
}

.stat-value {
  font-size: 26px;
  font-weight: 750;
  color: var(--ink-strong);
  line-height: 1.1;
  font-family: var(--font-mono);
}

.stat-note {
  font-size: 12px;
  color: var(--ink-faint);
}

/* ─── Content Grid ─── */
.content-grid {
  display: grid;
  grid-template-columns: 1fr 320px;
  gap: 18px;
}

.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.card-head h3 {
  font-size: 16px;
  font-weight: 700;
  color: var(--ink-strong);
}

.card-link {
  font-size: 13px;
  color: var(--ink-soft);
  font-weight: 600;
}

.module-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
}

.module-item {
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 16px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  box-shadow: var(--shadow-sm);
  transition: border-color 150ms ease, box-shadow 150ms ease;
}

.module-item:hover {
  border-color: var(--border);
  box-shadow: var(--shadow-md);
}

.module-item strong {
  font-size: 14px;
  font-weight: 700;
  color: var(--ink-strong);
}

.module-item span {
  font-size: 12px;
  color: var(--ink-soft);
}

/* ─── Side Column ─── */
.side-col {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.side-card {
  padding: 18px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  box-shadow: var(--shadow-sm);
}

.sync-info {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.sync-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.sync-row strong {
  font-size: 14px;
  font-weight: 600;
  color: var(--ink-strong);
  flex: 1;
}

.sync-row span {
  font-size: 13px;
  color: var(--ink-soft);
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
  gap: 6px;
}

.quick-link {
  font-size: 14px;
  font-weight: 600;
  color: var(--brand);
  padding: 6px 0;
  border-bottom: 1px solid var(--border-soft);
  transition: color 150ms ease;
}

.quick-link:last-child {
  border-bottom: none;
}

.quick-link:hover {
  color: var(--brand-strong);
}

/* ─── Responsive ─── */
@media (max-width: 1180px) {
  .hero-section {
    grid-template-columns: 1fr;
  }

  .hero-chart {
    display: none;
  }

  .stats-row {
    grid-template-columns: repeat(2, 1fr);
  }

  .content-grid {
    grid-template-columns: 1fr;
  }

  .module-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 760px) {
  .hero-indices {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .stats-row {
    grid-template-columns: 1fr;
  }

  .module-grid {
    grid-template-columns: 1fr;
  }

  .hero-section {
    padding: 18px;
  }

  .hero-indices {
    grid-template-columns: 1fr;
  }
}
</style>
