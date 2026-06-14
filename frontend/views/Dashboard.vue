<template>
  <WorkbenchLayout>
    <section class="home-page">
      <header class="home-header">
        <div>
          <div class="breadcrumbs">
            <span>业务工作台</span>
            <span>/</span>
            <strong>总览</strong>
          </div>
          <h1>航班服务业务中台</h1>
          <p>集中处理数据同步、产品观察、报告输出和客户分析。</p>
        </div>

        <div class="header-actions">
          <RouterLink class="secondary-action" to="/product-completion">
            <CalendarDays :size="17" />
            今日观察
          </RouterLink>
          <RouterLink class="primary-action" to="/data-preparation">
            <RefreshCw :size="17" />
            同步数据
          </RouterLink>
        </div>
      </header>

      <section class="metric-grid" aria-label="业务指标">
        <article v-for="item in summaryCards" :key="item.label" class="metric-card">
          <span class="metric-icon">
            <component :is="item.icon" :size="20" />
          </span>
          <div class="metric-body">
            <span class="metric-label">{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
          </div>
        </article>
      </section>

      <section class="workbench-grid">
        <article class="workflow-panel">
          <header class="panel-heading">
            <div>
              <h2>今日流程</h2>
              <p>按顺序完成核心业务动作</p>
            </div>
            <span>{{ shortDate }}</span>
          </header>

          <div class="pipeline" aria-label="业务流程">
            <RouterLink
              v-for="(step, index) in pipeline"
              :key="step.label"
              :to="step.path"
              class="pipeline-step"
            >
              <span class="step-index">{{ index + 1 }}</span>
              <span class="step-icon">
                <component :is="step.icon" :size="19" />
              </span>
              <div>
                <strong>{{ step.label }}</strong>
                <em>{{ step.desc }}</em>
              </div>
            </RouterLink>
          </div>
        </article>

        <article class="source-panel">
          <header class="panel-heading">
            <div>
              <h2>数据健康</h2>
              <p>主页依赖的数据源状态</p>
            </div>
            <RouterLink to="/data-preparation">管理</RouterLink>
          </header>

          <div class="source-list">
            <div class="source-row">
              <span class="source-dot" :class="{ ready: syncStatus.row_count }"></span>
              <div>
                <strong>交易总表</strong>
                <em>产品 / 交易 / 客户 / 渠道</em>
              </div>
              <span>{{ syncStatus.synced_at ? formatTime(syncStatus.synced_at) : '未同步' }}</span>
            </div>
            <div class="source-row">
              <span class="source-dot" :class="{ ready: coinvestStatus.row_count }"></span>
              <div>
                <strong>合投用户表</strong>
                <em>画像 / 专户 / 行业标签</em>
              </div>
              <span>{{ coinvestStatus.synced_at ? formatTime(coinvestStatus.synced_at) : '未同步' }}</span>
            </div>
          </div>
        </article>
      </section>

      <section class="module-panel">
        <header class="panel-heading">
          <div>
            <h2>常用入口</h2>
            <p>直接进入高频分析和输出页面</p>
          </div>
          <span>6 项</span>
        </header>

        <div class="module-grid">
          <RouterLink
            v-for="item in workflowItems"
            :key="item.path"
            :to="item.path"
            class="module-tile"
          >
            <span>
              <component :is="item.icon" :size="21" />
            </span>
            <div>
              <strong>{{ item.title }}</strong>
              <em>{{ item.desc }}</em>
            </div>
          </RouterLink>
        </div>
      </section>
    </section>
  </WorkbenchLayout>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import {
  Award,
  BarChart3,
  CalendarDays,
  Database,
  FileText,
  PieChart,
  RefreshCw,
  Search,
  TrendingDown,
  UserRound,
  Users,
} from '@lucide/vue'

import WorkbenchLayout from '../components/WorkbenchLayout.vue'

const shortDate = new Date().toLocaleDateString('zh-CN', {
  month: 'long',
  day: 'numeric',
  weekday: 'short',
})

const stats = ref({
  totalProducts: 0,
  activeProducts: 0,
  totalCustomers: 0,
  totalChannels: 0,
})
const syncStatus = ref({})
const coinvestStatus = ref({})

const pipeline = [
  { label: '同步数据', desc: '刷新飞书交易与合投数据', path: '/data-preparation', icon: Database },
  { label: '观察日历', desc: '处理敲出、派息和观察节点', path: '/product-completion', icon: Award },
  { label: '产品报告', desc: '生成存续产品运行材料', path: '/product-report', icon: FileText },
  { label: '客户画像', desc: '筛选专户与行业标签', path: '/user-profile', icon: UserRound },
  { label: '复盘分析', desc: '查看渠道、流失和存续表现', path: '/channel-analysis', icon: PieChart },
]

const workflowItems = [
  { path: '/data-preparation', title: '数据准备', desc: '飞书同步', icon: RefreshCw },
  { path: '/product-completion', title: '观察日历', desc: '敲出 / 派息', icon: Award },
  { path: '/product-report', title: '产品报告', desc: '运行材料', icon: FileText },
  { path: '/ongoing-product', title: '存续分析', desc: '规模与人次', icon: BarChart3 },
  { path: '/user-profile', title: '用户画像', desc: '条件筛选', icon: Search },
  { path: '/customer-churn', title: '流失识别', desc: '未复购客户', icon: TrendingDown },
]

const summaryCards = computed(() => [
  { label: '产品', value: stats.value.totalProducts.toLocaleString('zh-CN'), icon: FileText },
  { label: '存续', value: stats.value.activeProducts.toLocaleString('zh-CN'), icon: BarChart3 },
  { label: '客户', value: stats.value.totalCustomers.toLocaleString('zh-CN'), icon: Users },
  { label: '渠道', value: stats.value.totalChannels.toLocaleString('zh-CN'), icon: PieChart },
  { label: '同步', value: syncStatus.value.row_count ? '就绪' : '待同步', icon: Database },
])

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

async function loadSyncStatus() {
  try {
    const res1 = await fetch('/api/db/sync-status')
    if (res1.ok) syncStatus.value = await res1.json()
    const res2 = await fetch('/api/db/sync-coinvest-status')
    if (res2.ok) coinvestStatus.value = await res2.json()
  } catch {}
}

onMounted(async () => {
  await Promise.all([loadStats(), loadSyncStatus()])
})
</script>

<style scoped>
.home-page {
  padding-top: 18px;
}

.home-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 24px;
}

.breadcrumbs {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
  color: var(--ink-soft);
  font-size: 12.5px;
  font-weight: 600;
  letter-spacing: 0.02em;
}

.breadcrumbs strong {
  color: var(--ink);
  font-weight: 700;
}

.home-header h1 {
  color: var(--ink-strong);
  font-size: 34px;
  font-weight: 700;
  line-height: 1.15;
  letter-spacing: -0.025em;
}

.home-header p {
  max-width: 640px;
  margin-top: 8px;
  color: var(--ink-soft);
  font-size: 14.5px;
  line-height: 1.65;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 0 0 auto;
}

.primary-action,
.secondary-action {
  min-height: 42px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 0 18px;
  border-radius: var(--radius);
  font-size: 13.5px;
  font-weight: 600;
  letter-spacing: 0.01em;
  transition: background 200ms cubic-bezier(0.4, 0, 0.2, 1), border-color 200ms ease, transform 200ms cubic-bezier(0.4, 0, 0.2, 1), box-shadow 200ms ease;
}

.primary-action {
  color: #fff;
  background: var(--brand);
  border: 1px solid var(--brand);
  box-shadow: 0 1px 2px rgba(37, 99, 235, 0.2), 0 4px 12px rgba(37, 99, 235, 0.1);
}

.secondary-action {
  color: var(--ink);
  border: 1px solid var(--border-soft);
  background: #fff;
  box-shadow: var(--shadow-sm);
}

.primary-action:hover,
.secondary-action:hover {
  transform: translateY(-1px);
}

.primary-action:hover {
  box-shadow: 0 2px 4px rgba(37, 99, 235, 0.2), 0 8px 20px rgba(37, 99, 235, 0.14);
}

.secondary-action:hover {
  box-shadow: var(--shadow-md);
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 14px;
  margin-bottom: 18px;
}

.metric-card {
  min-height: 102px;
  display: grid;
  grid-template-columns: 42px minmax(0, 1fr);
  align-items: start;
  column-gap: 14px;
  padding: 20px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  background: var(--bg-card);
  box-shadow: var(--shadow-sm);
  transition: box-shadow 200ms ease, border-color 200ms ease;
}

.metric-card:hover {
  box-shadow: var(--shadow-md);
  border-color: var(--border);
}

.metric-icon,
.module-tile > span,
.step-icon {
  width: 42px;
  height: 42px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  border-radius: 10px;
  color: var(--brand);
  background: var(--brand-soft);
}

.metric-body {
  min-width: 0;
  padding-top: 1px;
}

.metric-label {
  display: block;
  color: var(--ink-soft);
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.03em;
  text-transform: uppercase;
}

.metric-card strong {
  display: block;
  margin-top: 6px;
  color: var(--ink-strong);
  font-size: 26px;
  font-weight: 700;
  line-height: 1.05;
  letter-spacing: -0.02em;
  overflow-wrap: anywhere;
}

.workbench-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.35fr) minmax(340px, 0.65fr);
  gap: 16px;
  margin-bottom: 18px;
}

.workflow-panel,
.module-panel,
.source-panel {
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  background: var(--bg-card);
  box-shadow: var(--shadow-sm);
  padding: 24px 26px;
}

.panel-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 20px;
}

.panel-heading h2 {
  color: var(--ink-strong);
  font-size: 18px;
  font-weight: 700;
  letter-spacing: -0.015em;
}

.panel-heading p {
  margin-top: 4px;
  color: var(--ink-soft);
  font-size: 13px;
  line-height: 1.5;
}

.panel-heading span,
.panel-heading a {
  color: var(--ink-soft);
  font-size: 12.5px;
  font-weight: 600;
}

.pipeline {
  display: grid;
  grid-template-columns: 1fr;
  gap: 10px;
}

.pipeline-step {
  min-height: 74px;
  display: grid;
  grid-template-columns: 28px 42px 1fr;
  align-items: center;
  gap: 14px;
  padding: 14px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  background: #fff;
  transition: transform 200ms cubic-bezier(0.4, 0, 0.2, 1), border-color 200ms ease, background 200ms ease, box-shadow 200ms ease;
}

.pipeline-step:hover {
  border-color: rgba(37, 99, 235, 0.25);
  background: #fafbff;
  box-shadow: 0 2px 8px rgba(37, 99, 235, 0.06);
}

.step-index {
  color: var(--ink-faint);
  font-family: var(--font-mono);
  font-size: 12px;
  font-weight: 700;
}

.pipeline-step strong,
.module-tile strong {
  display: block;
  color: var(--ink-strong);
  font-size: 14.5px;
  font-weight: 600;
  letter-spacing: -0.005em;
}

.pipeline-step em,
.module-tile em {
  display: block;
  margin-top: 4px;
  color: var(--ink-soft);
  font-size: 12.5px;
  font-style: normal;
  line-height: 1.4;
}

.module-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.module-tile {
  min-height: 88px;
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 18px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  color: var(--ink);
  background: #fff;
  transition: transform 200ms cubic-bezier(0.4, 0, 0.2, 1), background 200ms ease, border-color 200ms ease, box-shadow 200ms ease;
}

.module-tile:hover {
  background: #fafbff;
  border-color: rgba(37, 99, 235, 0.2);
  box-shadow: 0 2px 8px rgba(37, 99, 235, 0.05);
}

.module-tile:hover {
  transform: translateY(-1px);
}

.source-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.source-row {
  min-height: 76px;
  display: grid;
  grid-template-columns: 12px 1fr auto;
  align-items: center;
  gap: 14px;
  padding: 16px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  background: #fff;
}

.source-dot {
  width: 10px;
  height: 10px;
  border-radius: 999px;
  background: #cbd5e1;
  box-shadow: 0 0 0 3px rgba(203, 213, 225, 0.2);
}

.source-dot.ready {
  background: var(--success);
  box-shadow: 0 0 0 3px rgba(15, 159, 110, 0.15);
}

.source-row div {
  display: flex;
  flex-direction: column;
  gap: 5px;
  min-width: 0;
}

.source-row strong {
  color: var(--ink-strong);
  font-size: 14.5px;
  font-weight: 600;
}

.source-row em,
.source-row > span:last-child {
  color: var(--ink-soft);
  font-size: 12px;
  font-style: normal;
  font-weight: 500;
}

@media (max-width: 1280px) {
  .workbench-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 920px) {
  .metric-grid,
  .module-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .metric-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .home-page {
    padding-top: 20px;
  }

  .home-header,
  .header-actions {
    align-items: stretch;
    flex-direction: column;
  }

  .home-header h1 {
    font-size: 27px;
  }

  .metric-grid,
  .module-grid {
    grid-template-columns: 1fr;
  }

  .pipeline-step {
    grid-template-columns: 24px 42px 1fr;
  }

  .workbench-grid {
    grid-template-columns: 1fr;
  }
}
</style>
