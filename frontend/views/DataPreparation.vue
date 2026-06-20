<template>
  <div class="data-preparation-page">
    <div class="page-header">
      <h1 class="text-page-title">数据准备</h1>
      <p class="text-body">连接飞书账号后，同步数据到本地数据库，供后续业务页面使用。</p>
    </div>

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
      <p class="text-body">点击「同步数据」将以下三项数据全部同步到本地：航班服务交易总表、物料文档、返费明细。</p>

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
          <span v-if="syncStatus.synced_at && !syncing && !syncingTransactions" class="badge badge-green">已就绪</span>
          <span v-else-if="syncing || syncingTransactions" class="badge">同步中...</span>
          <span v-else class="badge badge-red">未同步</span>
          <button
            class="btn btn-sm btn-outline source-sync-btn"
            :disabled="!authorized || syncing || syncingTransactions"
            @click="syncTransactions"
          >
            {{ syncingTransactions ? '同步中' : '同步' }}
          </button>
          <span v-if="txMsg" class="source-msg" :class="{ 'msg-error': txMsgIsError }">{{ txMsg }}</span>
        </div>
      </div>

      <div class="source-row">
        <span class="source-icon">📄</span>
        <div class="source-info">
          <span class="source-name">物料文档</span>
          <span class="source-desc">
            <template v-if="docStatus.synced_at">
              上次同步：{{ formatTime(docStatus.synced_at) }}，共 {{ docStatus.doc_count || 0 }} 份
            </template>
            <template v-else>尚未同步</template>
          </span>
        </div>
        <div class="source-status">
          <span v-if="docStatus.synced_at && !syncing && !syncingDocs" class="badge badge-green">已就绪</span>
          <span v-else-if="syncing || syncingDocs" class="badge">同步中...</span>
          <span v-else class="badge badge-red">未同步</span>
          <button
            class="btn btn-sm btn-outline source-sync-btn"
            :disabled="!authorized || syncing || syncingDocs"
            @click="syncDocs"
          >
            {{ syncingDocs ? '同步中' : '同步' }}
          </button>
          <span v-if="docMsg" class="source-msg" :class="{ 'msg-error': docMsgIsError }">{{ docMsg }}</span>
        </div>
      </div>

      <div class="source-row">
        <span class="source-icon">💰</span>
        <div class="source-info">
          <span class="source-name">返费明细</span>
          <span class="source-desc">
            <template v-if="rebateStatus.synced_at">
              上次同步：{{ formatTime(rebateStatus.synced_at) }}
              <span v-if="rebateStatus.sheet_name" class="source-sheet">（{{ rebateStatus.sheet_name }}）</span>
              ，共 {{ rebateStatus.row_count || 0 }} 条
            </template>
            <template v-else>尚未同步</template>
          </span>
        </div>
        <div class="source-status">
          <span v-if="rebateStatus.synced_at && !syncing && !syncingRebate" class="badge badge-green">已就绪</span>
          <span v-else-if="syncing || syncingRebate" class="badge">同步中...</span>
          <span v-else class="badge badge-red">未同步</span>
          <button
            class="btn btn-sm btn-outline source-sync-btn"
            :disabled="!authorized || syncing || syncingRebate"
            @click="syncRebate"
          >
            {{ syncingRebate ? '同步中' : '同步' }}
          </button>
          <span v-if="rebateMsg" class="source-msg" :class="{ 'msg-error': rebateMsgIsError }">{{ rebateMsg }}</span>
        </div>
      </div>

      <div class="sync-actions">
        <button
          class="btn btn-primary"
          :disabled="!authorized || syncing"
          @click="syncAll"
        >
          {{ syncing ? syncProgress || '同步中...' : '同步数据' }}
        </button>
        <span v-if="!authorized" class="text-muted">请先连接飞书账号</span>
        <span v-if="syncError" class="error-msg">{{ syncError }}</span>
        <span v-if="syncSuccess" class="success-msg">{{ syncSuccess }}</span>
      </div>
    </PanelCard>

    <div v-if="result" class="panel-card">{{ result }}</div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import PanelCard from '../components/PanelCard.vue'

const REBATE_FOLDER_TOKEN = 'HINVfSPnPl266ad6xVschyK4nnb'
const REBATE_STATUS_KEY = 'rebate_detail_sync'
const REBATE_DATE_RE = /(?:^|[\D])(\d{2})(\d{2})(\d{2})(?=[\D]|$)/

const result = ref('')
const authorized = ref(false)
const authLoading = ref(false)
const syncing = ref(false)
const syncProgress = ref('')
const syncError = ref('')
const syncSuccess = ref('')
const syncStatus = ref({ synced_at: null, row_count: 0 })
const docStatus = ref({ synced_at: null, doc_count: 0 })
const rebateStatus = ref({ synced_at: null, row_count: 0, sheet_name: '' })

// 单项同步状态
const syncingTransactions = ref(false)
const syncingDocs = ref(false)
const syncingRebate = ref(false)
const txMsg = ref(''); const txMsgIsError = ref(false)
const docMsg = ref(''); const docMsgIsError = ref(false)
const rebateMsg = ref(''); const rebateMsgIsError = ref(false)

onMounted(async () => {
  await checkAuth()
  await loadSyncStatus()
  await loadDocStatus()
  await loadRebateStatus()
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

async function loadDocStatus() {
  try {
    const res = await fetch('/api/drive/product-docs/sync-status')
    if (res.ok) docStatus.value = await res.json()
  } catch {}
}

function loadRebateStatusFromStorage() {
  try {
    const raw = localStorage.getItem(REBATE_STATUS_KEY)
    if (raw) rebateStatus.value = JSON.parse(raw)
  } catch {}
}

function saveRebateStatusToStorage(status) {
  rebateStatus.value = status
  localStorage.setItem(REBATE_STATUS_KEY, JSON.stringify(status))
}

async function loadRebateStatus() {
  try {
    const res = await fetch('/api/db/rebate-detail-status')
    if (res.ok) {
      const data = await res.json()
      if (data.synced_at) {
        rebateStatus.value = data
        saveRebateStatusToStorage(data)
        return
      }
    }
  } catch {}
  loadRebateStatusFromStorage()
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

async function syncRebateDetailClientSide() {
  syncProgress.value = '同步中：读取返费文件夹...'
  const filesRes = await fetch(`/api/drive/files?folder_token=${REBATE_FOLDER_TOKEN}`)
  if (!filesRes.ok) throw new Error('读取返费文件夹失败')
  const filesData = await filesRes.json()
  const files = filesData.files || []

  const rebateFiles = []
  for (const f of files) {
    if (f.type === 'folder' || f.type === 'shortcut' || !f.token) continue
    if (!f.name.includes('返款明细') && !f.name.includes('返费明细')) continue
    const m = f.name.match(REBATE_DATE_RE)
    if (m) {
      const dateKey = '20' + m[1] + m[2] + m[3]
      rebateFiles.push({ token: f.token, name: f.name, date: dateKey })
    }
  }
  if (rebateFiles.length === 0) throw new Error('返费文件夹下未找到返费明细表')
  rebateFiles.sort((a, b) => a.date.localeCompare(b.date))
  const latest = rebateFiles[rebateFiles.length - 1]

  syncProgress.value = `同步中：导出 ${latest.name}（可能需要15秒）...`
  const exportRes = await fetch(`/api/drive/export-sheet?sheet_token=${latest.token}`)
  if (!exportRes.ok) throw new Error('导出返费明细表失败')
  const blob = await exportRes.blob()
  const contentType = exportRes.headers.get('content-type') || ''
  if (contentType.includes('application/json')) {
    const errData = await blob.text()
    throw new Error('导出返费明细表失败：' + errData)
  }

  syncProgress.value = '同步中：解析数据...'
  const XLSX = await import('xlsx')
  const arrayBuffer = await blob.arrayBuffer()
  const workbook = XLSX.read(arrayBuffer, { type: 'array' })
  const sheetName = workbook.SheetNames[0]
  const sheet = workbook.Sheets[sheetName]
  const matrix = XLSX.utils.sheet_to_json(sheet, { header: 1, defval: '', raw: false })
  const headers = matrix[0] || []
  const rows = matrix.slice(1)
    .map(row => {
      const record = {}
      headers.forEach((h, i) => { if (h) record[h] = row[i] ?? '' })
      return record
    })
    .filter(r => Object.values(r).some(v => String(v ?? '').trim() !== ''))

  const now = new Date().toISOString()
  saveRebateStatusToStorage({ synced_at: now, row_count: rows.length, sheet_name: latest.name })
  return { row_count: rows.length, sheet_name: latest.name, rows }
}

async function syncAll() {
  syncing.value = true
  syncError.value = ''
  syncSuccess.value = ''
  try {
    const res = await fetch('/api/db/sync-all', { method: 'POST' })
    if (res.ok) {
      const data = await res.json()
      const parts = []
      if (data.transactionCount !== undefined) parts.push(`交易表 ${data.transactionCount} 条`)
      if (data.docCount !== undefined) parts.push(`物料 ${data.docCount} 份`)
      if (data.rebateRowCount !== undefined) {
        parts.push(`返费明细 ${data.rebateRowCount} 条`)
      } else if (data.rebateError) {
        try {
          const rebateInfo = await syncRebateDetailClientSide()
          parts.push(`返费明细 ${rebateInfo.row_count} 条（${rebateInfo.sheet_name}）`)
        } catch (rebateErr) {
          syncError.value = '返费同步失败：' + rebateErr.message
        }
      }
      syncSuccess.value = '同步成功：' + parts.join('、')
      await Promise.all([loadSyncStatus(), loadDocStatus(), loadRebateStatus()])
      return
    }

    const parts = []

    syncProgress.value = '同步中：交易总表...'
    const syncRes = await fetch('/api/db/sync', { method: 'POST' })
    if (syncRes.ok) {
      const syncData = await syncRes.json()
      if (syncData.rowCount !== undefined) parts.push(`交易表 ${syncData.rowCount} 条`)
    }
    await loadSyncStatus()

    syncProgress.value = '同步中：物料文档...'
    const docRes = await fetch('/api/drive/sync-product-docs', { method: 'POST' })
    if (docRes.ok) {
      const docData = await docRes.json()
      if (docData.doc_count !== undefined) parts.push(`物料 ${docData.doc_count} 份`)
    }
    await loadDocStatus()

    syncProgress.value = '同步中：返费明细...'
    try {
      const rebateRes = await fetch('/api/db/sync-rebate-detail', { method: 'POST' })
      if (rebateRes.ok) {
        const rebateData = await rebateRes.json()
        if (rebateData.row_count !== undefined) parts.push(`返费明细 ${rebateData.row_count} 条`)
      } else {
        const rebateInfo = await syncRebateDetailClientSide()
        parts.push(`返费明细 ${rebateInfo.row_count} 条（${rebateInfo.sheet_name}）`)
      }
    } catch (rebateErr) {
      syncError.value = '返费同步失败：' + rebateErr.message
    }

    syncProgress.value = ''
    if (parts.length > 0) {
      syncSuccess.value = '同步成功：' + parts.join('、')
    } else if (!syncError.value) {
      syncSuccess.value = '同步完成（部分数据可能未更新）'
    }
  } catch (e) {
    syncError.value = e.message
  } finally {
    syncing.value = false
    syncProgress.value = ''
  }
}

async function syncTransactions() {
  if (!authorized.value || syncingTransactions.value || syncing.value) return
  syncingTransactions.value = true
  txMsg.value = ''
  try {
    const res = await fetch('/api/db/sync', { method: 'POST' })
    if (res.ok) {
      const data = await res.json()
      txMsg.value = `同步成功：交易表 ${data.rowCount ?? ''} 条`
      txMsgIsError.value = false
      await loadSyncStatus()
    } else {
      const e = await res.json().catch(() => ({}))
      txMsg.value = '同步失败：' + (e.error || res.status)
      txMsgIsError.value = true
    }
  } catch (e) {
    txMsg.value = '同步失败：' + e.message
    txMsgIsError.value = true
  } finally {
    syncingTransactions.value = false
  }
}

async function syncDocs() {
  if (!authorized.value || syncingDocs.value || syncing.value) return
  syncingDocs.value = true
  docMsg.value = ''
  try {
    const res = await fetch('/api/drive/sync-product-docs', { method: 'POST' })
    if (res.ok) {
      const data = await res.json()
      docMsg.value = `同步成功：物料 ${data.doc_count ?? ''} 份`
      docMsgIsError.value = false
      await loadDocStatus()
    } else {
      const e = await res.json().catch(() => ({}))
      docMsg.value = '同步失败：' + (e.error || res.status)
      docMsgIsError.value = true
    }
  } catch (e) {
    docMsg.value = '同步失败：' + e.message
    docMsgIsError.value = true
  } finally {
    syncingDocs.value = false
  }
}

async function syncRebate() {
  if (!authorized.value || syncingRebate.value || syncing.value) return
  syncingRebate.value = true
  rebateMsg.value = ''
  try {
    const res = await fetch('/api/db/sync-rebate-detail', { method: 'POST' })
    if (res.ok) {
      const data = await res.json()
      rebateMsg.value = `同步成功：返费明细 ${data.row_count ?? ''} 条${data.sheet_name ? '（' + data.sheet_name + '）' : ''}`
      rebateMsgIsError.value = false
      await loadRebateStatus()
      return
    }
    // 后端导出失败时回退到客户端解析
    const info = await syncRebateDetailClientSide()
    rebateMsg.value = `同步成功：返费明细 ${info.row_count} 条（${info.sheet_name}）`
    rebateMsgIsError.value = false
    await loadRebateStatus()
  } catch (e) {
    rebateMsg.value = '同步失败：' + e.message
    rebateMsgIsError.value = true
  } finally {
    syncingRebate.value = false
  }
}

function formatTime(iso) {
  const d = new Date(iso)
  return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')} ${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`
}
</script>

<style scoped>
.auth-row { display: flex; align-items: center; gap: 12px; }
.source-row { display: flex; align-items: center; gap: 14px; margin-bottom: 20px; }
.source-icon { font-size: 28px; flex-shrink: 0; }
.source-info { flex: 1; display: flex; flex-direction: column; gap: 4px; }
.source-name { font-size: 14px; font-weight: 700; color: var(--ink-strong); }
.source-desc { font-size: 12px; color: var(--ink-soft); }
.source-sheet { color: var(--ink-faint); font-style: italic; }
.source-status { flex-shrink: 0; display: flex; flex-direction: column; align-items: flex-end; gap: 4px; }
.source-sync-btn { font-size: 12px; padding: 2px 12px; }
.source-msg { font-size: 12px; color: var(--brand); max-width: 220px; text-align: right; }
.source-msg.msg-error { color: #e5484d; }
.sync-actions { display: flex; align-items: center; gap: 12px; margin-top: 8px; flex-wrap: wrap; }
</style>
