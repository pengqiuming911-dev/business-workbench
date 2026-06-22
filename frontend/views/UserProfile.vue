<template>
  <div class="account-page">
    <div class="page-header">
      <h1 class="text-page-title">当前账号</h1>
      <p class="text-body">展示当前连接的飞书登录信息，便于确认同步和推送所使用的账号。</p>
    </div>

    <PanelCard title="飞书账号信息">
      <div v-if="loading" class="account-state">
        <span class="badge">读取中...</span>
      </div>

      <div v-else-if="!authorized" class="account-state account-state-stack">
        <span class="badge badge-red">未连接飞书</span>
        <p class="text-body">当前还没有可用的飞书登录态，请先前往数据准备页面完成授权。</p>
        <RouterLink to="/data-preparation" class="btn btn-primary">前往数据准备</RouterLink>
      </div>

      <div v-else class="account-panel">
        <div class="account-hero">
          <img
            v-if="user?.avatar_url"
            :src="user.avatar_url"
            :alt="displayName"
            class="account-avatar"
          />
          <div v-else class="account-avatar account-avatar-fallback">
            <UserRound :size="32" :stroke-width="2" />
          </div>

          <div class="account-headline">
            <div class="account-title-row">
              <h2 class="account-name">{{ displayName }}</h2>
              <span class="badge badge-green">已连接</span>
            </div>
            <p class="account-subtitle">
              {{ user?.email || user?.en_name || '当前飞书登录账号' }}
            </p>
            <p v-if="userError" class="account-warning">
              用户详情读取异常：{{ userError }}
            </p>
          </div>
        </div>

        <div class="account-grid">
          <div class="info-item">
            <span class="info-label">姓名</span>
            <span class="info-value">{{ user?.name || '-' }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">英文名</span>
            <span class="info-value">{{ user?.en_name || '-' }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">邮箱</span>
            <span class="info-value">{{ user?.email || '-' }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">Open ID</span>
            <span class="info-value info-code">{{ user?.open_id || '-' }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">User ID</span>
            <span class="info-value info-code">{{ user?.user_id || '-' }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">Union ID</span>
            <span class="info-value info-code">{{ user?.union_id || '-' }}</span>
          </div>
        </div>

        <div class="account-actions">
          <RouterLink to="/data-preparation" class="btn btn-outline">前往数据准备</RouterLink>
          <button class="btn btn-outline" :disabled="logoutLoading" @click="logout">
            {{ logoutLoading ? '断开中...' : '断开连接' }}
          </button>
        </div>
      </div>
    </PanelCard>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { UserRound } from '@lucide/vue'
import PanelCard from '../components/PanelCard.vue'

const loading = ref(true)
const logoutLoading = ref(false)
const authorized = ref(false)
const user = ref(null)
const userError = ref('')

const displayName = computed(() => user.value?.name || user.value?.en_name || '当前账号')

onMounted(() => {
  loadAccount()
})

async function loadAccount() {
  loading.value = true
  userError.value = ''
  try {
    const res = await fetch('/api/auth/status')
    const data = await res.json()
    authorized.value = Boolean(data.authorized)
    user.value = data.user || null
    userError.value = data.user_error || ''
    window.dispatchEvent(new Event('auth-status-changed'))
  } catch (error) {
    authorized.value = false
    user.value = null
    userError.value = error instanceof Error ? error.message : '读取失败'
  } finally {
    loading.value = false
  }
}

async function logout() {
  logoutLoading.value = true
  try {
    await fetch('/api/auth/logout', { method: 'DELETE' })
    authorized.value = false
    user.value = null
    window.dispatchEvent(new Event('auth-status-changed'))
  } finally {
    logoutLoading.value = false
  }
}
</script>

<style scoped>
.account-page {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.account-state {
  display: flex;
  align-items: center;
  gap: 12px;
}

.account-state-stack {
  align-items: flex-start;
  flex-direction: column;
}

.account-panel {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.account-hero {
  display: flex;
  align-items: center;
  gap: 18px;
  padding-bottom: 20px;
  border-bottom: 1px solid var(--border-soft);
}

.account-avatar {
  width: 72px;
  height: 72px;
  border-radius: 20px;
  object-fit: cover;
  flex: 0 0 auto;
  box-shadow: var(--shadow-sm);
}

.account-avatar-fallback {
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, rgba(31, 58, 138, 0.14), rgba(44, 92, 224, 0.12));
  color: var(--brand);
}

.account-headline {
  min-width: 0;
}

.account-title-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 6px;
}

.account-name {
  margin: 0;
  font-size: 26px;
  line-height: 1.1;
  color: var(--ink-strong);
}

.account-subtitle {
  margin: 0;
  color: var(--ink-soft);
}

.account-warning {
  margin: 10px 0 0;
  color: var(--danger);
  font-size: 13px;
}

.account-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 16px 18px;
  border-radius: 14px;
  background: rgba(247, 248, 250, 0.95);
  border: 1px solid rgba(226, 232, 240, 0.9);
}

.info-label {
  font-size: 12px;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--ink-faint);
}

.info-value {
  color: var(--ink-strong);
  font-weight: 600;
  word-break: break-word;
}

.info-code {
  font-family: "IBM Plex Mono", "Roboto Mono", monospace;
  font-size: 12px;
}

.account-actions {
  display: flex;
  gap: 12px;
}

@media (max-width: 860px) {
  .account-hero {
    align-items: flex-start;
    flex-direction: column;
  }

  .account-grid {
    grid-template-columns: 1fr;
  }

  .account-actions {
    flex-direction: column;
  }
}
</style>
