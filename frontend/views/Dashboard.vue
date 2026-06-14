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
      <div class="hero-chart">
        <img :src="klineChart" alt="趋势图" />
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
          <span class="card-link">6 项</span>
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
              <span class="sync-dot" :class="{ ready: coinvestStatus.row_count }"></span>
              <strong>合投用户表</strong>
              <span>{{ coinvestStatus.synced_at ? formatTime(coinvestStatus.synced_at) : '未同步' }}</span>
            </div>
          </div>
        </div>

        <div class="side-card">
          <div class="card-head">
            <h3>今日重点</h3>
          </div>
          <div class="quick-links">
            <RouterLink to="/product-completion" class="quick-link">观察日历</RouterLink>
            <RouterLink to="/product-report" class="quick-link">产品报告</RouterLink>
            <RouterLink to="/customer-churn" class="quick-link">流失识别</RouterLink>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import klineChart from '../assets/kline-chart.svg'

const stats = ref({
  totalProducts: 0,
  activeProducts: 0,
  totalCustomers: 0,
  totalChannels: 0,
})
const syncStatus = ref({})
const coinvestStatus = ref({})

const statCards = computed(() => [
  { label: '产品总数', value: stats.value.totalProducts.toLocaleString('zh-CN'), note: '全部产品' },
  { label: '存续产品', value: stats.value.activeProducts.toLocaleString('zh-CN'), note: '持有中' },
  { label: '客户数量', value: stats.value.totalCustomers.toLocaleString('zh-CN'), note: '已登记' },
  { label: '渠道数量', value: stats.value.totalChannels.toLocaleString('zh-CN'), note: '活跃渠道' },
])

const modules = [
  { path: '/data-preparation', title: '数据准备', desc: '飞书同步' },
  { path: '/product-completion', title: '观察日历', desc: '敲出 / 派息' },
  { path: '/product-report', title: '产品报告', desc: '运行材料' },
  { path: '/ongoing-product', title: '产品分析', desc: '规模与人次' },
  { path: '/user-profile', title: '用户画像', desc: '条件筛选' },
  { path: '/customer-churn', title: '流失识别', desc: '未复购客户' },
]

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
.dashboard-page {
  padding-top: 8px;
}

/* ─── Hero ─── */
.hero-section {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 32px;
  align-items: center;
  margin-bottom: 28px;
  padding: 32px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: 14px;
  box-shadow: var(--shadow-sm);
}

.hero-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.hero-title {
  color: var(--ink-strong);
  font-size: 28px;
  font-weight: 750;
  line-height: 1.15;
}

.hero-desc {
  color: var(--ink-soft);
  font-size: 16px;
  line-height: 1.6;
  max-width: 400px;
}

.hero-actions {
  display: flex;
  gap: 10px;
  margin-top: 8px;
}

.hero-chart {
  display: flex;
  justify-content: center;
  align-items: center;
}

.hero-chart img {
  width: 100%;
  max-width: 420px;
  height: auto;
}

/* ─── Stats ─── */
.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 20px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: 14px;
  box-shadow: var(--shadow-sm);
}

.stat-label {
  font-size: 15px;
  font-weight: 600;
  color: var(--ink-soft);
}

.stat-value {
  font-size: 28px;
  font-weight: 750;
  color: var(--ink-strong);
  line-height: 1.1;
}

.stat-note {
  font-size: 14px;
  color: var(--ink-faint);
}

/* ─── Content Grid ─── */
.content-grid {
  display: grid;
  grid-template-columns: 1fr 340px;
  gap: 20px;
}

.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.card-head h3 {
  font-size: 18px;
  font-weight: 700;
  color: var(--ink-strong);
}

.card-link {
  font-size: 15px;
  color: var(--ink-soft);
  font-weight: 600;
}

.module-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
}

.module-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 20px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: 14px;
  box-shadow: var(--shadow-sm);
  transition: transform 150ms ease, box-shadow 150ms ease;
}

.module-item:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.module-item strong {
  font-size: 15px;
  font-weight: 700;
  color: var(--ink-strong);
}

.module-item span {
  font-size: 14px;
  color: var(--ink-soft);
}

/* ─── Side Column ─── */
.side-col {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.side-card {
  padding: 20px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: 14px;
  box-shadow: var(--shadow-sm);
}

.sync-info {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.sync-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.sync-row strong {
  font-size: 15px;
  font-weight: 600;
  color: var(--ink-strong);
  flex: 1;
}

.sync-row span {
  font-size: 14px;
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
  gap: 8px;
}

.quick-link {
  font-size: 15px;
  font-weight: 600;
  color: var(--brand);
  padding: 8px 0;
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
@media (max-width: 960px) {
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

@media (max-width: 640px) {
  .stats-row {
    grid-template-columns: 1fr;
  }

  .module-grid {
    grid-template-columns: 1fr;
  }

  .hero-section {
    padding: 20px;
  }
}
</style>
