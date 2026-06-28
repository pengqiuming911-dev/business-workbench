<template>
  <WorkbenchLayout wide>
    <div class="poster-copy-page">
      <div class="page-header poster-header">
        <div>
          <h1 class="text-page-title">喜报文案</h1>
          <p class="text-body">按观察日期查看产品，并用当前保存模板实时生成每个产品自己的喜报文案。</p>
        </div>
        <div class="mode-pill" :class="{ history: mode === 'date' }">
          {{ mode === 'date' ? '历史日期' : '最新观察' }}
        </div>
      </div>

      <PanelCard title="观察日期">
        <div class="toolbar">
          <div class="date-field">
            <label for="poster-date">日历日期</label>
            <input
              id="poster-date"
              v-model="selectedDate"
              type="date"
              class="input"
              @change="loadByDate"
            />
          </div>
          <button class="btn btn-secondary" type="button" :disabled="loading" @click="loadLatest">
            最新
          </button>
          <button class="btn btn-primary" type="button" :disabled="loading" @click="reload">
            {{ loading ? '加载中...' : '刷新' }}
          </button>
          <span v-if="date" class="query-date">当前日期：{{ date }}</span>
        </div>
        <div v-if="error" class="error-msg">{{ error }}</div>
      </PanelCard>

      <div v-if="loading" class="loading-state">正在加载喜报文案...</div>
      <div v-else-if="empty" class="empty-panel">
        <div class="empty-title">无观察产品</div>
        <div class="empty-copy">{{ date ? `${date} 没有需要观察的产品` : '暂无历史观察记录' }}</div>
      </div>
      <div v-else class="copy-list">
        <section v-for="item in products" :key="item.product.id" class="copy-card">
          <div class="copy-card-header">
            <div>
              <h2>{{ item.product.name }}</h2>
              <div class="copy-meta">
                <span>{{ item.product.code || '未填写标的' }}</span>
                <span>观察日 {{ item.observation.observation_date }}</span>
                <span>更新 {{ formatTime(item.observation.updated_at) }}</span>
              </div>
            </div>
            <div class="status-tags">
              <span class="badge" :class="item.observation.is_knocked_out === '是' ? 'badge-red' : 'badge-blue'">
                敲出：{{ item.observation.is_knocked_out || '-' }}
              </span>
              <span class="badge" :class="item.observation.is_dividend === '是' ? 'badge-green' : 'badge-blue'">
                派息：{{ item.observation.is_dividend || '-' }}
              </span>
            </div>
          </div>

          <div class="copy-grid">
            <div class="copy-summary">
              <div class="summary-row">
                <span>年化收益</span>
                <strong>{{ item.artifact.annualized_return }}%</strong>
              </div>
              <div class="summary-row">
                <span>本月派息</span>
                <strong>{{ item.artifact.monthly_coupon }}%</strong>
              </div>
              <div class="summary-row">
                <span>累计派息</span>
                <strong>{{ item.artifact.cumulative_dividend_rate }}%</strong>
              </div>
              <div class="summary-row">
                <span>敲出界限</span>
                <strong>{{ item.artifact.knockout_value || '-' }}</strong>
              </div>
            </div>
            <div class="poster-preview">
              <DividendReportTemplate :fields="item.artifact" @downloaded="d => archivePoster(item.artifact, d)" />
            </div>
          </div>
        </section>
      </div>
    </div>
  </WorkbenchLayout>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import WorkbenchLayout from '../components/WorkbenchLayout.vue'
import PanelCard from '../components/PanelCard.vue'
import DividendReportTemplate from '../components/DividendReportTemplate.vue'
import { archivePoster } from '../utils/posterArchive.js'

const selectedDate = ref('')
const date = ref('')
const mode = ref('latest')
const products = ref([])
const loading = ref(false)
const empty = ref(false)
const error = ref('')

async function fetchPosterCopy(params = '') {
  loading.value = true
  error.value = ''
  try {
    const res = await fetch(`/api/poster-copy${params}`)
    if (!res.ok) {
      const data = await res.json().catch(() => ({}))
      throw new Error(data.error || '加载失败')
    }
    const data = await res.json()
    mode.value = data.mode || 'latest'
    date.value = data.date || ''
    products.value = data.products || []
    empty.value = Boolean(data.empty)
  } catch (e) {
    error.value = e.message || '加载失败'
    products.value = []
    empty.value = true
  } finally {
    loading.value = false
  }
}

function loadLatest() {
  selectedDate.value = ''
  fetchPosterCopy()
}

function loadByDate() {
  if (!selectedDate.value) {
    loadLatest()
    return
  }
  fetchPosterCopy(`?date=${encodeURIComponent(selectedDate.value)}`)
}

function reload() {
  if (selectedDate.value) loadByDate()
  else loadLatest()
}

function formatTime(value) {
  if (!value) return '-'
  const time = new Date(value)
  if (Number.isNaN(time.getTime())) return value
  return time.toLocaleString('zh-CN')
}

onMounted(loadLatest)
</script>

<style scoped>
:deep(.workbench-main) {
  max-width: none;
}

.poster-copy-page {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.poster-header {
  flex-direction: row;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.mode-pill {
  min-height: 32px;
  display: inline-flex;
  align-items: center;
  padding: 0 12px;
  border-radius: 999px;
  color: var(--brand);
  background: var(--brand-soft);
  font-size: 12px;
  font-weight: 800;
  white-space: nowrap;
}

.mode-pill.history {
  color: var(--success);
  background: var(--success-soft);
}

.toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: 12px;
}

.date-field {
  display: grid;
  gap: 6px;
  min-width: 220px;
}

.date-field label {
  color: var(--ink-soft);
  font-size: 12px;
  font-weight: 700;
}

.query-date {
  min-height: 36px;
  display: inline-flex;
  align-items: center;
  color: var(--ink-soft);
  font-size: 13px;
  font-weight: 700;
}

.empty-panel {
  margin-top: 16px;
  padding: 56px 24px;
  text-align: center;
  border: 1px dashed var(--border);
  border-radius: var(--radius-lg);
  background: rgba(255, 255, 255, 0.72);
}

.empty-title {
  color: var(--ink-strong);
  font-size: 18px;
  font-weight: 800;
}

.empty-copy {
  margin-top: 8px;
  color: var(--ink-soft);
  font-size: 14px;
}

.copy-list {
  display: grid;
  gap: 18px;
  margin-top: 16px;
}

.copy-card {
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  background: var(--bg-card);
  box-shadow: var(--shadow-sm);
  overflow: hidden;
}

.copy-card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 18px 20px;
  border-bottom: 1px solid var(--border-soft);
  background: #f8fafc;
}

.copy-card-header h2 {
  margin: 0;
  color: var(--ink-strong);
  font-size: 17px;
  line-height: 1.35;
}

.copy-meta,
.status-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.copy-meta {
  margin-top: 8px;
  color: var(--ink-soft);
  font-size: 12px;
  font-weight: 600;
}

.copy-grid {
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr);
  gap: 18px;
  padding: 20px;
}

.copy-summary {
  display: grid;
  align-content: start;
  gap: 10px;
}

.summary-row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 12px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  background: #fff;
  color: var(--ink-soft);
  font-size: 13px;
}

.summary-row strong {
  color: var(--ink-strong);
  font-family: var(--font-mono);
}

.poster-preview {
  min-width: 0;
  overflow-x: auto;
  padding-bottom: 6px;
}

@media (max-width: 1180px) {
  .copy-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 720px) {
  .poster-header,
  .copy-card-header {
    flex-direction: column;
  }

  .date-field,
  .toolbar .btn {
    width: 100%;
  }
}
</style>
