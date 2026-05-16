<template>
  <SubPageLayout title="数据准备">
    <div class="section">
      <p class="desc">连接飞书账号后，点击「同步数据」将「航班服务交易总表」（含产品表、交易表、客户表、渠道表、直客来源表）写入本地数据库，供各功能页面使用。数据同步后无需再次连接，直接使用即可。</p>

      <!-- 飞书授权状态 -->
      <div class="panel auth-panel">
        <h3 class="panel-title">飞书账号连接</h3>
        <div v-if="authorized" class="auth-row">
          <span class="auth-badge auth-ok">已连接</span>
          <button class="btn btn-outline" @click="logout">断开连接</button>
        </div>
        <div v-else class="auth-row">
          <span class="auth-badge auth-none">未连接</span>
          <button class="btn btn-primary" :disabled="authLoading" @click="loginFeishu">
            {{ authLoading ? '跳转中...' : '连接飞书账号' }}
          </button>
        </div>
      </div>

      <!-- 数据同步 -->
      <div class="panel">
        <h3 class="panel-title">数据同步</h3>
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
            <span v-if="syncStatus.synced_at && !syncing" class="status-badge status-ok">已就绪</span>
            <span v-else-if="syncing" class="status-badge status-loading">同步中...</span>
            <span v-else class="status-badge status-none">未同步</span>
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
          <span v-if="!authorized" class="hint">请先连接飞书账号</span>
          <span v-if="syncError" class="error-msg">{{ syncError }}</span>
          <span v-if="syncSuccess" class="success-msg">{{ syncSuccess }}</span>
        </div>
      </div>

      <!-- 合投用户表 -->
      <div class="panel">
        <h3 class="panel-title">合投用户表</h3>
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
            <span v-if="coinvestStatus.synced_at && !coinvestSyncing" class="status-badge status-ok">已就绪</span>
            <span v-else-if="coinvestSyncing" class="status-badge status-loading">同步中...</span>
            <span v-else class="status-badge status-none">未同步</span>
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
          <span v-if="!authorized" class="hint">请先连接飞书账号</span>
          <span v-if="coinvestError" class="error-msg">{{ coinvestError }}</span>
          <span v-if="coinvestSuccess" class="success-msg">{{ coinvestSuccess }}</span>
        </div>
      </div>
    </div>

    <div v-if="result" class="result">{{ result }}</div>
  </SubPageLayout>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import SubPageLayout from '../components/SubPageLayout.vue'

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
.desc { color: #6B5C4E; font-size: 14px; line-height: 1.8; margin-bottom: 24px; }
.panel { background: #fff; border-radius: 12px; padding: 24px; margin-bottom: 20px; border: 1px solid #E8DDD0; }
.panel-title { font-size: 16px; font-weight: 600; margin-bottom: 16px; }
.btn { padding: 8px 20px; border-radius: 6px; font-size: 13px; cursor: pointer; border: none; }
.btn:disabled { opacity: 0.6; cursor: not-allowed; }
.btn-primary { background: #D97757; color: #fff; }
.btn-outline { background: #fff; color: #D97757; border: 1px solid #D97757; }
.result { margin-top: 16px; padding: 14px 18px; background: #FDF3E7; border-radius: 8px; color: #D97757; font-size: 14px; }

.auth-panel .panel-title { margin-bottom: 12px; }
.auth-row { display: flex; align-items: center; gap: 12px; }
.auth-badge { font-size: 12px; font-weight: 600; padding: 3px 10px; border-radius: 20px; }
.auth-ok { background: #E6F4EA; color: #2E7D32; }
.auth-none { background: #FDF3E7; color: #D97757; }

.source-row { display: flex; align-items: center; gap: 14px; margin-bottom: 20px; }
.source-icon { font-size: 28px; flex-shrink: 0; }
.source-info { flex: 1; display: flex; flex-direction: column; gap: 4px; }
.source-name { font-size: 14px; font-weight: 600; color: #1A1109; }
.source-desc { font-size: 12px; color: #A8967E; }
.source-status { flex-shrink: 0; }
.status-badge { font-size: 12px; font-weight: 600; padding: 3px 10px; border-radius: 20px; }
.status-ok { background: #E6F4EA; color: #2E7D32; }
.status-loading { background: #FDF3E7; color: #D97757; }
.status-none { background: #F5F0E8; color: #A8967E; }

.sync-actions { display: flex; align-items: center; gap: 12px; }
.hint { font-size: 12px; color: #A8967E; }
.error-msg { font-size: 13px; color: #C62828; }
.success-msg { font-size: 13px; color: #2E7D32; }
</style>
