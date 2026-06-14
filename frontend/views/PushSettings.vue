<template>
  <div class="push-settings-page">
    <h1 class="text-page-title">飞书推送设置</h1>
    <p class="text-body" style="margin-bottom:24px">配置今日观察的飞书群机器人推送。设置推送时间后，系统将每天定时把今日观察内容推送到飞书群。</p>

    <PanelCard title="推送配置">
      <div class="form-row">
        <label>启用推送</label>
        <label class="toggle-label">
          <input type="checkbox" v-model="form.enabled" />
          <span>{{ form.enabled ? '已启用' : '已关闭' }}</span>
        </label>
      </div>

      <div class="form-row">
        <label>Webhook URL</label>
        <input
          v-model="form.webhook_url"
          type="text"
          class="input"
          placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/xxx"
        />
        <span class="text-label" style="margin-top:4px;display:block">
          在飞书群 → 设置 → 群机器人 → 添加自定义机器人，复制 Webhook 地址
        </span>
      </div>

      <div class="form-row">
        <label>推送时间</label>
        <div class="time-picker">
          <select v-model.number="form.cron_hour" class="input time-select">
            <option v-for="h in 24" :key="h - 1" :value="h - 1">{{ String(h - 1).padStart(2, '0') }} 时</option>
          </select>
          <select v-model.number="form.cron_minute" class="input time-select">
            <option v-for="m in minuteOptions" :key="m" :value="m">{{ String(m).padStart(2, '0') }} 分</option>
          </select>
        </div>
      </div>

      <div class="form-row" style="display:flex;gap:12px;flex-wrap:wrap">
        <button class="btn btn-primary" :disabled="saving" @click="saveConfig">
          {{ saving ? '保存中...' : '保存设置' }}
        </button>
        <button class="btn btn-secondary" :disabled="testing" @click="testPush">
          {{ testing ? '发送中...' : '发送测试消息' }}
        </button>
      </div>

      <p v-if="message" class="text-label" :class="{ 'error-msg': isError, 'success-msg': !isError }" style="margin-top:12px">
        {{ message }}
      </p>
    </PanelCard>

    <PanelCard title="推送状态" style="margin-top:20px">
      <div class="form-row">
        <label>上次推送时间</label>
        <span class="text-body">{{ lastPushTime || '从未推送' }}</span>
      </div>
      <div class="form-row">
        <label>上次推送结果</label>
        <span class="text-body">{{ lastPushResult || '--' }}</span>
      </div>
    </PanelCard>
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

onMounted(loadConfig)
</script>

<style scoped>
.toggle-label {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  font-size: 14px;
}

.time-picker {
  display: flex;
  gap: 8px;
}

.time-select {
  width: 120px;
}
</style>