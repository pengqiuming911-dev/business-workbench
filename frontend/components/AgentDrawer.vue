<template>
  <transition name="drawer-slide">
    <aside v-if="open" class="agent-drawer">
      <div class="drawer-header">
        <h3 class="drawer-title">衍选智能体</h3>
        <button class="drawer-close" @click="emit('close')">
          <X :size="18" :stroke-width="2" />
        </button>
      </div>
      <div class="drawer-body">
        <div class="chat-messages" ref="messagesRef">
          <!-- Welcome -->
          <div v-if="messages.length === 0" class="welcome">
            <div class="welcome-icon"><Sparkles :size="28" :stroke-width="1.8" /></div>
            <p class="welcome-text">你好！我是衍选智能体，有什么可以帮你的吗？</p>
            <div class="suggestions">
              <button v-for="s in suggestions" :key="s" class="suggestion-btn" @click="sendText(s)">{{ s }}</button>
            </div>
          </div>

          <!-- Messages -->
          <template v-for="(msg, i) in messages" :key="i">
            <div class="chat-msg" :class="msg.role">
              <!-- Tool call indicator -->
              <div v-if="msg.toolName" class="tool-indicator">
                <Zap :size="12" :stroke-width="2.5" />
                <span>{{ msg.toolName }}</span>
                <Loader2 v-if="msg.toolLoading" :size="12" :stroke-width="2.5" class="spin" />
              </div>
              <div v-if="msg.content" class="msg-bubble" v-html="renderMarkdown(msg.content)"></div>
            </div>
          </template>

          <!-- Streaming indicator -->
          <div v-if="streaming && !currentDelta" class="chat-msg assistant">
            <div class="msg-bubble typing">
              <span class="dot"></span><span class="dot"></span><span class="dot"></span>
            </div>
          </div>
        </div>

        <div class="chat-input-wrap">
          <input
            v-model="inputText"
            class="chat-input"
            placeholder="输入消息..."
            @keydown.enter="send"
          />
          <button class="send-btn" :disabled="!inputText.trim() || streaming" @click="send">
            <ArrowUp :size="16" :stroke-width="2.5" />
          </button>
        </div>
      </div>
    </aside>
  </transition>
</template>

<script setup>
import { ref, nextTick, watch } from 'vue'
import { X, Sparkles, Zap, Loader2, ArrowUp } from '@lucide/vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

defineProps({ open: { type: Boolean, default: false } })
const emit = defineEmits(['close'])

const inputText = ref('')
const messages = ref([])
const streaming = ref(false)
const currentDelta = ref('')
const conversationId = ref(null)
const messagesRef = ref(null)

const suggestions = [
  '今日有哪些产品需要观察？',
  '帮我查看存续产品概况',
  '最近有哪些敲出的产品？',
]

function renderMarkdown(text) {
  return DOMPurify.sanitize(marked.parse(text || ''))
}

function scrollToBottom() {
  nextTick(() => {
    if (messagesRef.value) {
      messagesRef.value.scrollTop = messagesRef.value.scrollHeight
    }
  })
}

watch(messages, scrollToBottom, { deep: true })
watch(currentDelta, scrollToBottom)

function sendText(text) {
  inputText.value = text
  send()
}

async function send() {
  const text = inputText.value.trim()
  if (!text || streaming.value) return

  messages.value.push({ role: 'user', content: text })
  inputText.value = ''
  streaming.value = true
  currentDelta.value = ''
  scrollToBottom()

  try {
    const res = await fetch('/api/agent/chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        conversation_id: conversationId.value || 0,
        message: text,
      }),
    })

    if (!res.ok) {
      const err = await res.json().catch(() => ({}))
      throw new Error(err.error || '请求失败')
    }

    const reader = res.body.getReader()
    const decoder = new TextDecoder()
    let assistantMsg = { role: 'assistant', content: '' }
    messages.value.push(assistantMsg)

    let buffer = ''
    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n\n')
      buffer = lines.pop() || ''

      for (const block of lines) {
        const line = block.trim()
        if (!line.startsWith('data: ')) continue
        const raw = line.slice(6)
        try {
          const event = JSON.parse(raw)
          switch (event.type) {
            case 'conversation_id':
              conversationId.value = event.conversation_id
              break
            case 'delta':
              assistantMsg.content += event.text
              currentDelta.value = assistantMsg.content
              break
            case 'reasoning_delta':
              // silently consume reasoning
              break
            case 'tool_call':
              messages.value.push({ role: 'assistant', toolName: event.name, toolLoading: true, content: '' })
              scrollToBottom()
              break
            case 'tool_done': {
              const tc = [...messages.value].reverse().find(m => m.toolName === event.name && m.toolLoading)
              if (tc) tc.toolLoading = false
              break
            }
            case 'error':
              assistantMsg.content += `\n\n⚠️ ${event.error}`
              break
            case 'done':
              break
          }
        } catch {}
      }
    }

    if (!assistantMsg.content) {
      assistantMsg.content = '（无回复内容）'
    }
  } catch (err) {
    messages.value.push({ role: 'assistant', content: '抱歉，发生了错误：' + err.message })
  } finally {
    streaming.value = false
    currentDelta.value = ''
    scrollToBottom()
  }
}
</script>

<style scoped>
.agent-drawer {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  width: 400px;
  z-index: 180;
  background: #fff;
  border-left: 1px solid var(--border-soft);
  display: flex;
  flex-direction: column;
  box-shadow: -4px 0 24px rgba(0, 0, 0, 0.06);
}

.drawer-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 18px 20px;
  border-bottom: 1px solid var(--border-soft);
}

.drawer-title {
  font-size: 17px;
  font-weight: 700;
  color: var(--ink-strong);
}

.drawer-close {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--ink-soft);
  cursor: pointer;
  transition: background 150ms ease;
}

.drawer-close:hover {
  background: var(--border-soft);
}

.drawer-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

/* Welcome */
.welcome {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  padding: 32px 12px 16px;
  text-align: center;
}

.welcome-icon {
  width: 52px;
  height: 52px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: linear-gradient(135deg, #ecfdf5, #d1fae5);
  color: #10b981;
}

.welcome-text {
  font-size: 15px;
  font-weight: 600;
  color: var(--ink);
  line-height: 1.5;
}

.suggestions {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
  margin-top: 4px;
}

.suggestion-btn {
  width: 100%;
  padding: 10px 14px;
  border: 1px solid var(--border-soft);
  border-radius: 10px;
  background: #fff;
  color: var(--ink);
  font-size: 13px;
  font-weight: 600;
  text-align: left;
  cursor: pointer;
  transition: background 150ms ease, border-color 150ms ease;
}

.suggestion-btn:hover {
  background: #f0fdf8;
  border-color: #10b981;
}

/* Messages */
.chat-msg.user {
  align-self: flex-end;
}

.chat-msg.assistant {
  align-self: flex-start;
}

.msg-bubble {
  max-width: 300px;
  padding: 10px 14px;
  border-radius: 14px;
  font-size: 14px;
  line-height: 1.6;
  color: var(--ink-strong);
}

.user .msg-bubble {
  background: #10b981;
  color: #fff;
  border-bottom-right-radius: 4px;
}

.assistant .msg-bubble {
  background: var(--bg-page, #f8f9fa);
  border-bottom-left-radius: 4px;
}

.msg-bubble.typing {
  display: flex;
  gap: 4px;
  padding: 12px 18px;
}

.dot {
  width: 6px;
  height: 6px;
  background: var(--ink-faint);
  border-radius: 50%;
  animation: bounce 1.4s infinite ease-in-out both;
}

.dot:nth-child(1) { animation-delay: -0.32s; }
.dot:nth-child(2) { animation-delay: -0.16s; }

@keyframes bounce {
  0%, 80%, 100% { transform: scale(0); }
  40% { transform: scale(1); }
}

/* Tool indicator */
.tool-indicator {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 100px;
  background: #fffbeb;
  color: #b45309;
  font-size: 12px;
  font-weight: 600;
  margin-bottom: 4px;
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Input */
.chat-input-wrap {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 20px;
  border-top: 1px solid var(--border-soft);
}

.chat-input {
  flex: 1;
  height: 40px;
  padding: 0 14px;
  border: 1px solid var(--border-soft);
  border-radius: 100px;
  font-size: 14px;
  outline: none;
  transition: border-color 150ms ease;
}

.chat-input:focus {
  border-color: #10b981;
}

.send-btn {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 50%;
  background: #10b981;
  color: #fff;
  cursor: pointer;
  transition: opacity 150ms ease;
}

.send-btn:disabled {
  opacity: 0.4;
  cursor: default;
}

/* Transition */
.drawer-slide-enter-active,
.drawer-slide-leave-active {
  transition: transform 250ms ease;
}

.drawer-slide-enter-from,
.drawer-slide-leave-to {
  transform: translateX(100%);
}

@media (max-width: 640px) {
  .agent-drawer {
    width: 100%;
  }
}
</style>
