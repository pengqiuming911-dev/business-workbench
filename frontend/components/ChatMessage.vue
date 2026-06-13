<template>
  <div class="chat-message" :class="[`role-${role}`, { streaming }]">
    <div class="message-avatar">
      <component :is="role === 'user' ? User : Bot" :size="18" :stroke-width="2" />
    </div>
    <div class="message-body">
      <div class="message-header">
        {{ role === 'user' ? '你' : '智能助手' }}
      </div>
      <div class="message-content" v-html="renderedContent"></div>
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
import { User, Bot } from '@lucide/vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

const props = defineProps({
  role: { type: String, required: true },
  content: { type: String, default: '' },
  streaming: { type: Boolean, default: false },
  toolCalls: { type: Array, default: null },
})

marked.setOptions({
  breaks: true,
  gfm: true,
})

const renderedContent = computed(() => {
  if (!props.content) return ''
  const raw = marked.parse(props.content)
  return DOMPurify.sanitize(raw)
})
</script>

<style scoped>
.chat-message {
  display: flex;
  gap: 12px;
  padding: 16px 0;
}

.message-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  background: var(--bg-hover);
  color: var(--ink-soft);
}

.role-user .message-avatar {
  background: var(--brand-soft);
  color: var(--brand);
}

.role-assistant .message-avatar {
  background: var(--surface-muted);
  color: var(--ink);
}

.message-body {
  flex: 1;
  min-width: 0;
}

.message-header {
  font-size: 13px;
  font-weight: 600;
  color: var(--ink-strong);
  margin-bottom: 4px;
}

.message-content {
  font-size: 14.5px;
  line-height: 1.7;
  color: var(--ink);
  word-break: break-word;
}

.message-content :deep(p) {
  margin: 0 0 8px 0;
}

.message-content :deep(p:last-child) {
  margin-bottom: 0;
}

.message-content :deep(code) {
  background: var(--bg-hover);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: var(--font-mono);
  font-size: 13px;
}

.message-content :deep(pre) {
  background: var(--bg-hover);
  padding: 12px 16px;
  border-radius: 8px;
  overflow-x: auto;
  margin: 8px 0;
}

.message-content :deep(pre code) {
  background: none;
  padding: 0;
}

.message-content :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin: 8px 0;
  font-size: 13px;
}

.message-content :deep(th),
.message-content :deep(td) {
  border: 1px solid var(--border);
  padding: 6px 12px;
  text-align: left;
}

.message-content :deep(th) {
  background: var(--bg-hover);
  font-weight: 600;
}

.tool-calls {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
}

.tool-call-badge {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 4px;
  background: var(--brand-soft);
  color: var(--brand);
  font-weight: 500;
}

.streaming .message-content::after {
  content: '▍';
  animation: blink 1s step-end infinite;
  color: var(--brand);
}

@keyframes blink {
  50% { opacity: 0; }
}
</style>
