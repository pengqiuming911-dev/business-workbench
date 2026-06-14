<template>
  <div class="chat-message" :class="[`role-${role}`, { streaming }]">
    <div class="message-avatar">
      <component :is="role === 'user' ? User : Bot" :size="17" :stroke-width="2" />
    </div>
    <div class="message-column">
      <div class="message-header">
        {{ role === 'user' ? '你' : '智能助手' }}
      </div>

      <div v-if="reasoning" class="reasoning-block">
        <div class="reasoning-label">思考过程</div>
        <div class="reasoning-content" v-html="renderedReasoning"></div>
      </div>

      <div class="message-card">
        <div class="message-content" v-html="renderedContent"></div>
      </div>

      <div v-if="toolCalls && toolCalls.length" class="tool-calls">
        <div v-for="(tc, i) in toolCalls" :key="i" class="tool-call-badge">
          调用工具：{{ tc.function?.name || tc }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Bot, User } from '@lucide/vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

const props = defineProps({
  role: { type: String, required: true },
  content: { type: String, default: '' },
  reasoning: { type: String, default: '' },
  streaming: { type: Boolean, default: false },
  toolCalls: { type: Array, default: null },
})

marked.setOptions({
  breaks: true,
  gfm: true,
})

const renderedContent = computed(() => {
  if (!props.content) return ''
  return DOMPurify.sanitize(marked.parse(props.content))
})

const renderedReasoning = computed(() => {
  if (!props.reasoning) return ''
  return DOMPurify.sanitize(marked.parse(props.reasoning))
})
</script>

<style scoped>
.chat-message {
  display: flex;
  gap: 14px;
  padding: 14px 0;
}

.message-avatar {
  width: 34px;
  height: 34px;
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.9);
  border: 1px solid rgba(15, 23, 42, 0.08);
  color: #617086;
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.04);
}

.role-user .message-avatar {
  color: #2f6cf6;
  background: rgba(243, 248, 255, 0.98);
  border-color: rgba(47, 108, 246, 0.14);
}

.message-column {
  flex: 1;
  min-width: 0;
}

.message-header {
  margin-bottom: 8px;
  color: #506078;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.02em;
}

.message-card {
  border: 1px solid rgba(15, 23, 42, 0.07);
  border-radius: 22px;
  background: rgba(255, 255, 255, 0.9);
  padding: 18px 20px;
  box-shadow: 0 18px 36px rgba(15, 23, 42, 0.04);
}

.role-user .message-card {
  background: linear-gradient(180deg, rgba(245, 249, 255, 0.98), rgba(241, 246, 255, 0.98));
  border-color: rgba(47, 108, 246, 0.12);
}

.message-content {
  font-size: 15px;
  line-height: 1.85;
  color: #1f2a3d;
  word-break: break-word;
}

.reasoning-block {
  margin-bottom: 10px;
  padding: 12px 14px;
  border-radius: 16px;
  border: 1px solid rgba(15, 23, 42, 0.06);
  background: rgba(248, 250, 252, 0.94);
}

.reasoning-label {
  margin-bottom: 6px;
  color: #8d98a8;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.reasoning-content {
  color: #66758b;
  font-size: 13px;
  line-height: 1.75;
}

.reasoning-content :deep(p),
.message-content :deep(p) {
  margin: 0 0 10px 0;
}

.reasoning-content :deep(p:last-child),
.message-content :deep(p:last-child) {
  margin-bottom: 0;
}

.message-content :deep(ul),
.message-content :deep(ol) {
  padding-left: 20px;
  margin: 8px 0;
}

.message-content :deep(code) {
  background: rgba(241, 245, 249, 0.9);
  padding: 2px 7px;
  border-radius: 8px;
  font-family: var(--font-mono);
  font-size: 13px;
}

.message-content :deep(pre) {
  margin: 10px 0;
  padding: 14px 16px;
  overflow-x: auto;
  border-radius: 16px;
  background: #f6f8fb;
  border: 1px solid rgba(15, 23, 42, 0.06);
}

.message-content :deep(pre code) {
  background: transparent;
  padding: 0;
}

.message-content :deep(table) {
  width: 100%;
  margin: 10px 0;
  border-collapse: collapse;
  font-size: 13px;
}

.message-content :deep(th),
.message-content :deep(td) {
  padding: 8px 10px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  text-align: left;
}

.message-content :deep(th) {
  background: #f7f9fc;
  font-weight: 700;
}

.tool-calls {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}

.tool-call-badge {
  padding: 5px 10px;
  border-radius: 999px;
  background: rgba(47, 108, 246, 0.08);
  color: #3165d4;
  font-size: 12px;
  font-weight: 600;
}

.streaming .message-content::after {
  content: '▋';
  color: #2f6cf6;
  animation: blink 1s step-end infinite;
}

@keyframes blink {
  50% {
    opacity: 0;
  }
}
</style>
