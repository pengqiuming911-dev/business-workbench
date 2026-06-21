<template>
  <div class="push-settings-page">
    <div class="page-header">
      <h1 class="text-page-title">飞书推送设置</h1>
      <p class="text-body">
        配置观察提醒的发送方式、发送时间和回执状态。页面收窄为配置版心，避免长字段横向拉满。
      </p>
    </div>

    <div class="push-layout">
      <PanelCard title="推送配置">
        <div class="settings-form">
          <div class="settings-row settings-row-compact">
            <label>启用推送</label>
            <label class="toggle-card">
              <input type="checkbox" v-model="form.enabled" />
              <span class="toggle-indicator" :class="{ on: form.enabled }"></span>
              <span class="toggle-copy">{{ form.enabled ? '已启用，按计划发送' : '已关闭，不会自动推送' }}</span>
            </label>
          </div>

          <div class="settings-row settings-row-stack">
            <label>Webhook URL</label>
            <div class="webhook-field">
              <input
                v-model="form.webhook_url"
                type="text"
                class="input webhook-input"
                placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/xxx"
              />
              <button class="btn btn-secondary btn-sm" type="button" :disabled="!form.webhook_url" @click="copyWebhook">
                复制
              </button>
            </div>
            <span class="text-label helper-text">
              在飞书群设置中添加自定义机器人，并复制 Webhook 地址到这里。
            </span>
          </div>

          <div class="settings-row settings-row-stack">
            <label>推送时间</label>
            <div class="time-picker">
              <select v-model.number="form.cron_hour" class="input time-select">
                <option v-for="h in 24" :key="h - 1" :value="h - 1">{{ String(h - 1).padStart(2, '0') }} 时</option>
              </select>
              <select v-model.number="form.cron_minute" class="input time-select">
                <option v-for="m in minuteOptions" :key="m" :value="m">{{ String(m).padStart(2, '0') }} 分</option>
              </select>
            </div>
            <span class="text-label helper-text">建议设置在开盘前后或盘后复核时段。</span>
          </div>

          <div class="settings-actions">
            <button class="btn btn-primary" :disabled="saving" @click="saveConfig">
              {{ saving ? '保存中...' : '保存设置' }}
            </button>
            <button class="btn btn-secondary" :disabled="testing" @click="testPush">
              {{ testing ? '发送中...' : '发送测试消息' }}
            </button>
          </div>

          <p v-if="message" class="feedback-line" :class="{ 'error-msg': isError, 'success-msg': !isError }">
            {{ message }}
          </p>
        </div>
      </PanelCard>

      <div class="status-column">
        <PanelCard title="状态回执">
          <div class="status-stack">
            <div class="status-item">
              <span class="status-label">当前状态</span>
              <strong class="status-value">{{ form.enabled ? '自动推送开启' : '自动推送关闭' }}</strong>
            </div>
            <div class="status-item">
              <span class="status-label">上次推送时间</span>
              <strong class="status-value">{{ lastPushTime || '从未推送' }}</strong>
            </div>
            <div class="status-item">
              <span class="status-label">上次推送结果</span>
              <strong class="status-value">{{ lastPushResult || '--' }}</strong>
            </div>
          </div>
        </PanelCard>

        <PanelCard title="使用建议">
          <div class="tips-list">
            <div class="tip-item">
              <strong>先保存，再测试</strong>
              <span>避免测试仍命中旧的 Webhook 或旧时间配置。</span>
            </div>
            <div class="tip-item">
              <strong>优先固定时间段</strong>
              <span>建议每日固定推送时刻，减少运营习惯漂移。</span>
            </div>
            <div class="tip-item">
              <strong>长地址保持可复制</strong>
              <span>Webhook 只做单行展示，避免整屏横向延展。</span>
            </div>
          </div>
        </PanelCard>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import PanelCard from '../components/PanelCard.vue'

const form = ref({
  webhook_url: '',
  cron_hour: 9,
  cron_minute: 0,
  enabled: false,
})

const saving = ref(false)
const testing = ref(false)
const message = ref('')
const isError = ref(false)
const lastPushTime = ref('')
const lastPushResult = ref('')

const minuteOptions = [0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55]

async function loadConfig() {
  try {
    const res = await fetch('/api/push-config')
    const data = await res.json()
    form.value = {
      webhook_url: data.webhook_url || '',
      cron_hour: data.cron_hour ?? 9,
      cron_minute: data.cron_minute ?? 0,
      enabled: !!data.enabled,
    }
    lastPushTime.value = data.last_push_time || ''
    lastPushResult.value = data.last_push_result || ''
  } catch (err) {
    message.value = '加载配置失败: ' + err.message
    isError.value = true
  }
}

async function saveConfig() {
  saving.value = true
  message.value = ''
  try {
    const res = await fetch('/api/push-config', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(form.value),
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '保存失败')
    message.value = '保存成功'
    isError.value = false
  } catch (err) {
    message.value = '保存失败: ' + err.message
    isError.value = true
  } finally {
    saving.value = false
  }
}

async function testPush() {
  testing.value = true
  message.value = ''
  try {
    const res = await fetch('/api/push/test', { method: 'POST' })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '推送失败')
    if (data.sent) {
      message.value = `测试推送成功，已发送 ${data.count} 个产品`
      isError.value = false
    } else {
      message.value = `未发送: ${data.reason}`
      isError.value = false
    }
    await loadConfig()
  } catch (err) {
    message.value = '推送失败: ' + err.message
    isError.value = true
  } finally {
    testing.value = false
  }
}

async function copyWebhook() {
  try {
    await navigator.clipboard.writeText(form.value.webhook_url)
    message.value = 'Webhook 已复制'
    isError.value = false
  } catch {
    message.value = '复制失败，请手动复制'
    isError.value = true
  }
}

onMounted(loadConfig)
</script>

<style scoped>
.push-settings-page {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.push-layout {
  display: grid;
  grid-template-columns: minmax(0, 760px) 320px;
  gap: 18px;
  align-items: start;
}

.settings-form {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.settings-row {
  display: flex;
  gap: 12px;
}

.settings-row > label:first-child {
  width: 96px;
  flex-shrink: 0;
  color: var(--ink-soft);
  font-size: 13px;
  font-weight: 700;
  padding-top: 10px;
}

.settings-row-compact {
  align-items: center;
}

.settings-row-compact > label:first-child {
  padding-top: 0;
}

.settings-row-stack {
  align-items: flex-start;
}

.toggle-card {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  min-height: 44px;
  padding: 0 14px;
  border: 1px solid var(--border-soft);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.9);
  cursor: pointer;
}

.toggle-card input {
  display: none;
}

.toggle-indicator {
  width: 34px;
  height: 20px;
  border-radius: 999px;
  background: #cbd5e1;
  position: relative;
  transition: background 150ms ease;
}

.toggle-indicator::after {
  content: '';
  position: absolute;
  top: 2px;
  left: 2px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: #fff;
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.18);
  transition: transform 150ms ease;
}

.toggle-indicator.on {
  background: var(--brand);
}

.toggle-indicator.on::after {
  transform: translateX(14px);
}

.toggle-copy {
  color: var(--ink);
  font-size: 14px;
  font-weight: 600;
}

.webhook-field {
  display: flex;
  gap: 10px;
  width: 100%;
  max-width: 560px;
}

.webhook-input {
  flex: 1;
  min-width: 0;
}

.helper-text {
  display: block;
  margin-top: 6px;
}

.time-picker {
  display: flex;
  gap: 10px;
}

.time-select {
  width: 120px;
}

.settings-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.feedback-line {
  margin-top: -4px;
}

.status-column {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.status-stack,
.tips-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.status-item,
.tip-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 14px 0;
  border-bottom: 1px solid var(--border-soft);
}

.status-item:last-child,
.tip-item:last-child {
  border-bottom: none;
  padding-bottom: 0;
}

.status-label {
  color: var(--ink-soft);
  font-size: 12px;
  font-weight: 700;
}

.status-value,
.tip-item strong {
  color: var(--ink-strong);
  font-size: 14px;
  font-weight: 800;
}

.tip-item span {
  color: var(--ink-soft);
  font-size: 12px;
  line-height: 1.6;
}

@media (max-width: 1120px) {
  .push-layout {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 720px) {
  .settings-row {
    flex-direction: column;
  }

  .settings-row > label:first-child {
    width: auto;
    padding-top: 0;
  }

  .webhook-field {
    max-width: none;
  }
}
</style>
