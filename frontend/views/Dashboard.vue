<template>
  <WorkbenchLayout>
    <header class="page-header">
      <h1 class="text-page-title">Dashboard</h1>
      <p class="text-body dashboard-date">{{ today }}</p>
    </header>

    <!-- Stat row -->
    <div class="stat-row">
      <StatCard title="产品总数" :value="stats.totalProducts" :trend="null" />
      <StatCard title="存续产品" :value="stats.activeProducts" :trend="null" />
      <StatCard title="客户总数" :value="stats.totalCustomers" :trend="null" />
      <StatCard title="渠道总数" :value="stats.totalChannels" :trend="null" />
    </div>

    <!-- Charts row -->
    <div class="chart-row">
      <PanelCard title="月度成交趋势">
        <div ref="lineChartRef" class="chart-container"></div>
      </PanelCard>
      <PanelCard title="渠道成交占比">
        <div ref="pieChartRef" class="chart-container"></div>
      </PanelCard>
    </div>

    <!-- Feeds row -->
    <div class="feed-row">
      <PanelCard title="近期观察日">
        <div v-if="obsLoading" class="loading-state">加载中...</div>
        <div v-else-if="observations.length === 0" class="empty-state">暂无今日观察</div>
        <ul v-else class="obs-list">
          <li v-for="p in observations" :key="p.id" class="obs-item">
            <strong class="obs-name">{{ p.name }}</strong>
            <span class="obs-manager">{{ p.manager }}</span>
            <span v-if="p.code" class="obs-code">{{ p.code }}</span>
          </li>
        </ul>
      </PanelCard>

      <PanelCard title="数据同步状态">
        <div class="sync-list">
          <div class="sync-row">
            <span class="sync-label">交易总表</span>
            <span class="sync-time">{{ syncStatus.synced_at ? formatTime(syncStatus.synced_at) : '未同步' }}</span>
            <span class="sync-count" v-if="syncStatus.row_count">{{ syncStatus.row_count }} 条</span>
          </div>
          <div class="sync-row">
            <span class="sync-label">合投用户表</span>
            <span class="sync-time">{{ coinvestStatus.synced_at ? formatTime(coinvestStatus.synced_at) : '未同步' }}</span>
            <span class="sync-count" v-if="coinvestStatus.row_count">{{ coinvestStatus.row_count }} 条</span>
          </div>
        </div>
      </PanelCard>
    </div>

    <!-- Quick entry grid -->
    <div class="quick-grid">
      <RouterLink
        v-for="m in modules"
        :key="m.path"
        :to="m.path"
        class="quick-card panel-card"
      >
        <component :is="m.icon" :size="20" :stroke-width="1.8" class="quick-icon" />
        <strong>{{ m.title }}</strong>
        <em>{{ m.desc }}</em>
      </RouterLink>
    </div>
  </WorkbenchLayout>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { RouterLink } from 'vue-router'
import {
  Database, UserRound, UserX, FileText,
  Eye, BarChart3, PieChart, Users
} from '@lucide/vue'

import WorkbenchLayout from '../components/WorkbenchLayout.vue'
import StatCard from '../components/StatCard.vue'
import PanelCard from '../components/PanelCard.vue'

import * as echarts from 'echarts/core'
import { LineChart, PieChart as EPieChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'

echarts.use([LineChart, EPieChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer])

const today = new Date().toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric', weekday: 'long' })

const stats = ref({ totalProducts: 0, activeProducts: 0, totalCustomers: 0, totalChannels: 0 })
const obsLoading = ref(true)
const observations = ref([])
const syncStatus = ref({})
const coinvestStatus = ref({})

const lineChartRef = ref(null)
const pieChartRef = ref(null)
let lineChart = null
let pieChart = null

const modules = [
  { path: '/data-preparation', title: '数据准备', desc: '同步飞书数据', icon: Database },
  { path: '/user-profile', title: '用户画像', desc: '查询用户特征', icon: UserRound },
  { path: '/customer-churn', title: '客户流失', desc: '未复购客户', icon: UserX },
  { path: '/product-report', title: '产品报告', desc: '运行材料', icon: FileText },
  { path: '/product-completion', title: '观察', desc: '跟踪观察日', icon: Eye },
  { path: '/ongoing-product', title: '存续分析', desc: '持有产品', icon: BarChart3 },
  { path: '/channel-analysis', title: '渠道分析', desc: '渠道表现', icon: PieChart },
  { path: '/nominal-buyer', title: '名义购买人', desc: '管理人匹配', icon: Users },
]

function formatTime(iso) {
  const d = new Date(iso)
  return d.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

async function loadStats() {
  try {
    const res = await fetch('/api/dashboard/stats')
    if (res.ok) stats.value = await res.json()
  } catch {}
}

async function loadObservations() {
  obsLoading.value = true
  try {
    const res = await fetch('/api/observations/today')
    const data = await res.json()
    observations.value = (data.products || []).slice(0, 8)
  } catch {
    observations.value = []
  } finally {
    obsLoading.value = false
  }
}

async function loadSyncStatus() {
  try {
    const res1 = await fetch('/api/db/sync-status')
    if (res1.ok) syncStatus.value = await res1.json()
    const res2 = await fetch('/api/db/sync-coinvest-status')
    if (res2.ok) coinvestStatus.value = await res2.json()
  } catch {}
}

function renderCharts(chartData) {
  if (lineChartRef.value && !lineChart) {
    lineChart = echarts.init(lineChartRef.value)
    lineChart.setOption({
      tooltip: { trigger: 'axis' },
      grid: { left: 40, right: 20, top: 20, bottom: 30 },
      xAxis: { type: 'category', data: chartData.monthlyTrend.map(d => d.month), axisLabel: { fontSize: 11 } },
      yAxis: [
        { type: 'value', name: '金额', axisLabel: { fontSize: 11, formatter: v => (v / 10000).toFixed(0) + '万' } },
        { type: 'value', name: '笔数', axisLabel: { fontSize: 11 } },
      ],
      series: [
        { name: '金额', type: 'line', smooth: true, data: chartData.monthlyTrend.map(d => d.amount), itemStyle: { color: '#2677ff' } },
        { name: '笔数', type: 'line', smooth: true, yAxisIndex: 1, data: chartData.monthlyTrend.map(d => d.count), itemStyle: { color: '#6c5ce7' } },
      ],
    })
  }

  if (pieChartRef.value && !pieChart) {
    pieChart = echarts.init(pieChartRef.value)
    pieChart.setOption({
      tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
      series: [{
        type: 'pie',
        radius: ['40%', '70%'],
        center: ['50%', '50%'],
        label: { fontSize: 11 },
        data: chartData.channelDistribution.map(d => ({ name: d.channel, value: d.amount })),
      }],
    })
  }
}

async function loadCharts() {
  try {
    const res = await fetch('/api/dashboard/charts')
    if (res.ok) {
      const data = await res.json()
      renderCharts(data)
    }
  } catch {}
}

function handleResize() {
  lineChart?.resize()
  pieChart?.resize()
}

onMounted(async () => {
  await Promise.all([loadStats(), loadObservations(), loadSyncStatus(), loadCharts()])
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  lineChart?.dispose()
  pieChart?.dispose()
})
</script>

<style scoped>
.page-header {
  display: flex;
  align-items: baseline;
  gap: 12px;
  margin-bottom: 24px;
}

.dashboard-date {
  color: var(--ink-soft);
}

.stat-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.chart-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin-bottom: 24px;
}

.chart-container {
  height: 280px;
  width: 100%;
}

.feed-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin-bottom: 24px;
}

.obs-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.obs-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 0;
  border-bottom: 1px solid var(--border-soft);
}
.obs-item:last-child { border-bottom: none; }

.obs-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--ink-strong);
}

.obs-manager {
  font-size: 12px;
  color: var(--ink-soft);
}

.obs-code {
  font-size: 11px;
  color: var(--ink-faint);
  font-family: var(--font-mono);
}

.sync-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.sync-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 0;
  border-bottom: 1px solid var(--border-soft);
}
.sync-row:last-child { border-bottom: none; }

.sync-label {
  font-weight: 600;
  font-size: 13px;
  color: var(--ink-strong);
  flex: 1;
}

.sync-time, .sync-count {
  font-size: 12px;
  color: var(--ink-soft);
  font-family: var(--font-mono);
}

.quick-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 12px;
}

.quick-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 16px;
  cursor: pointer;
  text-decoration: none;
}

.quick-card strong {
  font-size: 14px;
  color: var(--ink-strong);
}

.quick-card em {
  font-size: 12px;
  color: var(--ink-soft);
  font-style: normal;
}

.quick-icon {
  color: var(--brand);
}

@media (max-width: 720px) {
  .stat-row { grid-template-columns: repeat(2, 1fr); }
  .chart-row, .feed-row { grid-template-columns: 1fr; }
}
</style>