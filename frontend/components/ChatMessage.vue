<template>
  <div class="chat-message" :class="[`role-${role}`, { streaming }]">
    <div class="message-avatar">
      <component :is="role === 'user' ? User : Bot" :size="17" :stroke-width="2" />
    </div>
    <div class="message-column">
      <div v-if="toolCalls && toolCalls.length" class="tool-calls">
        <div
          v-for="(tc, i) in toolCalls"
          :key="i"
          class="tool-call-badge"
          :class="{ 'is-calling': tc.status === 'calling' }"
        >
          <span v-if="tc.status === 'calling'" class="call-spinner"></span>
          <span v-else class="call-check">✓</span>
          {{ tc.name || tc.function?.name || tc }}
        </div>
      </div>

      <div v-if="hasContent || streaming" class="message-card">
        <div class="message-content" v-html="renderedContent"></div>
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
  streaming: { type: Boolean, default: false },
  toolCalls: { type: Array, default: null },
})

marked.setOptions({
  breaks: true,
  gfm: true,
})

const hasContent = computed(() => !!props.content)

const renderedContent = computed(() => {
  if (!props.content) return ''
  return DOMPurify.sanitize(marked.parse(props.content))
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

.message-content :deep(p) {
  margin: 0 0 10px 0;
}

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
  margin-bottom: 10px;
}

.tool-call-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 11px;
  border-radius: 999px;
  background: rgba(100, 116, 139, 0.08);
  color: #5b6b80;
  font-size: 12px;
  font-weight: 600;
}

.tool-call-badge.is-calling {
  background: rgba(47, 108, 246, 0.09);
  color: #2f6cf6;
}

.call-spinner {
  width: 11px;
  height: 11px;
  border-radius: 50%;
  border: 1.5px solid rgba(47, 108, 246, 0.25);
  border-top-color: #2f6cf6;
  animation: spin 0.8s linear infinite;
}

.call-check {
  color: #22a87a;
  font-size: 12px;
  line-height: 1;
}

@keyframes spin {
  to { transform: rotate(360deg); }
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
