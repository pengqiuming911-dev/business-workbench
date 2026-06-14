<template>
  <WorkbenchLayout>
    <h1 class="text-page-title">数据准备</h1>
    <p class="text-body">连接飞书账号后，将航班服务交易总表和合投用户表同步到本地数据库，供后续业务页面使用。</p>

    <PanelCard title="飞书账号连接">
      <div v-if="authorized" class="auth-row">
        <span class="badge badge-green">已连接</span>
        <button class="btn btn-outline" @click="logout">断开连接</button>
      </div>
      <div v-else class="auth-row">
        <span class="badge badge-red">未连接</span>
        <button class="btn btn-primary" :disabled="authLoading" @click="loginFeishu">
          {{ authLoading ? '跳转中...' : '连接飞书账号' }}
        </button>
      </div>
    </PanelCard>

    <PanelCard title="数据同步">
      <p class="text-body">连接飞书账号后，点击「同步数据」将「航班服务交易总表」（含产品表、交易表、客户表、渠道表、直客来源表）写入本地数据库，供各功能页面使用。数据同步后无需再次连接，直接使用即可。</p>
      <div class="source-row">
        <span class="source-icon">📊</span>
        <div class="source-info">
          <span class="source-name">航班服务交易总表</span>
          <span class="source-desc">
            <template v-if="syncStatus.synced_at">
              上次同步：{{ formatTime(syncStatus.synced_at) }}，共 {{ syncStatus.row_count }} 条
            </template>
            <template v-else>尚未同步</template>
          </span>
        </div>
        <div class="source-status">
          <span v-if="syncStatus.synced_at && !syncing" class="badge badge-green">已就绪</span>
          <span v-else-if="syncing" class="badge">同步中...</span>
          <span v-else class="badge badge-red">未同步</span>
        </div>
      </div>

      <div class="sync-actions">
        <button
          class="btn btn-primary"
          :disabled="!authorized || syncing"
          @click="syncData"
        >
          {{ syncing ? '同步中...' : '同步数据' }}
        </button>
        <span v-if="!authorized" class="text-muted">请先连接飞书账号</span>
        <span v-if="syncError" class="error-msg">{{ syncError }}</span>
        <span v-if="syncSuccess" class="success-msg">{{ syncSuccess }}</span>
      </div>
    </PanelCard>

    <PanelCard title="合投用户表">
      <div class="source-row">
        <span class="source-icon">👥</span>
        <div class="source-info">
          <span class="source-name">合投用户表</span>
          <span class="source-desc">
            <template v-if="coinvestStatus.synced_at">
              上次同步：{{ formatTime(coinvestStatus.synced_at) }}，共 {{ coinvestStatus.row_count }} 条
            </template>
            <template v-else>尚未同步</template>
          </span>
        </div>
        <div class="source-status">
          <span v-if="coinvestStatus.synced_at && !coinvestSyncing" class="badge badge-green">已就绪</span>
          <span v-else-if="coinvestSyncing" class="badge">同步中...</span>
          <span v-else class="badge badge-red">未同步</span>
        </div>
      </div>

      <div class="sync-actions">
        <button
          class="btn btn-primary"
          :disabled="!authorized || coinvestSyncing"
          @click="syncCoInvest"
        >
          {{ coinvestSyncing ? '同步中...' : '同步数据' }}
        </button>
        <span v-if="!authorized" class="text-muted">请先连接飞书账号</span>
        <span v-if="coinvestError" class="error-msg">{{ coinvestError }}</span>
        <span v-if="coinvestSuccess" class="success-msg">{{ coinvestSuccess }}</span>
      </div>
    </PanelCard>

    <div v-if="result" class="panel-card">{{ result }}</div>
  </WorkbenchLayout>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import WorkbenchLayout from '../components/WorkbenchLayout.vue'
import PanelCard from '../components/PanelCard.vue'

const result = ref('')
const authorized = ref(false)
const authLoading = ref(false)
const syncing = ref(false)
const syncError = ref('')
const syncSuccess = ref('')
const syncStatus = ref({ synced_at: null, row_count: 0 })
const coinvestSyncing = ref(false)
const coinvestError = ref('')
const coinvestSuccess = ref('')
const coinvestStatus = ref({ synced_at: null, row_count: 0 })

onMounted(async () => {
  await checkAuth()
  await loadSyncStatus()
  await loadCoInvestStatus()
  const params = new URLSearchParams(window.location.search)
  if (params.get('auth') === 'success') {
    authorized.value = true
    window.history.replaceState({}, '', window.location.pathname)
  } else if (params.get('auth') === 'error') {
    result.value = '飞书授权失败：' + (params.get('msg') || '未知错误')
    window.history.replaceState({}, '', window.location.pathname)
  }
})

async function checkAuth() {
  try {
    const res = await fetch('/api/auth/status')
    const data = await res.json()
    authorized.value = data.authorized
  } catch {
    authorized.value = false
  }
}

async function loadSyncStatus() {
  try {
    const res = await fetch('/api/db/sync-status')
    syncStatus.value = await res.json()
  } catch {}
}

async function loadCoInvestStatus() {
  try {
    const res = await fetch('/api/db/sync-coinvest-status')
    coinvestStatus.value = await res.json()
  } catch {}
}

async function loginFeishu() {
  authLoading.value = true
  try {
    const res = await fetch('/api/auth/login')
    const { url } = await res.json()
    window.location.href = url
  } catch {
    result.value = '获取授权链接失败，请确认后端服务已启动。'
    authLoading.value = false
  }
}

async function logout() {
  await fetch('/api/auth/logout', { method: 'DELETE' })
  authorized.value = false
}

async function syncData() {
  syncing.value = true
  syncError.value = ''
  syncSuccess.value = ''
  try {
    const res = await fetch('/api/db/sync', { method: 'POST' })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '同步失败')
    syncSuccess.value = `同步成功，共写入 ${data.rowCount} 条数据`
    await loadSyncStatus()
  } catch (e) {
    syncError.value = e.message
  } finally {
    syncing.value = false
  }
}

async function syncCoInvest() {
  coinvestSyncing.value = true
  coinvestError.value = ''
  coinvestSuccess.value = ''
  try {
    const res = await fetch('/api/db/sync-coinvest', { method: 'POST' })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '同步失败')
    coinvestSuccess.value = `同步成功，共写入 ${data.rowCount} 条数据`
    await loadCoInvestStatus()
  } catch (e) {
    coinvestError.value = e.message
  } finally {
    coinvestSyncing.value = false
  }
}

function formatTime(iso) {
  const d = new Date(iso)
  return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')} ${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`
}
</script>

<style scoped>
.auth-row { display: flex; align-items: center; gap: 14px; }
.source-row { display: flex; align-items: center; gap: 16px; margin-bottom: 24px; }
.source-icon { font-size: 28px; flex-shrink: 0; }
.source-info { flex: 1; display: flex; flex-direction: column; gap: 5px; }
.source-name { font-size: 14.5px; font-weight: 600; color: var(--ink-strong); letter-spacing: -0.005em; }
.source-desc { font-size: 12.5px; color: var(--ink-soft); line-height: 1.5; }
.source-status { flex-shrink: 0; }
.sync-actions { display: flex; align-items: center; gap: 14px; }
</style>
