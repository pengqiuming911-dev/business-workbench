<template>
  <WorkbenchLayout wide>
    <div class="agent-shell">
      <aside class="conversation-sidebar" :class="{ collapsed: sidebarCollapsed }">
        <div class="sidebar-top">
          <div v-if="!sidebarCollapsed" class="sidebar-title">智能体会话</div>
          <button class="icon-btn" type="button" @click="sidebarCollapsed = !sidebarCollapsed">
            <component :is="sidebarCollapsed ? PanelRightOpen : PanelRightClose" :size="16" />
          </button>
        </div>

        <button
          v-if="!sidebarCollapsed"
          class="new-chat-btn"
          type="button"
          @click="newConversation"
        >
          <Plus :size="16" />
          <span>新建对话</span>
        </button>

        <div v-if="!sidebarCollapsed" class="conversation-list">
          <button
            v-for="conv in conversations"
            :key="conv.id"
            type="button"
            class="conversation-item"
            :class="{ active: currentConversationId === conv.id }"
            @click="selectConversation(conv.id)"
          >
            <div class="conversation-main">
              <MessageSquare :size="15" />
              <span class="conv-title">{{ conv.title }}</span>
            </div>
            <span class="conv-delete" @click.stop="deleteConversation(conv.id)">
              <Trash2 :size="13" />
            </span>
          </button>

          <div v-if="conversations.length === 0" class="empty-conversations">
            暂无对话记录
          </div>
        </div>
      </aside>

      <section class="chat-stage">
        <div ref="messagesContainer" class="messages-area">
          <div v-if="messages.length === 0" class="welcome-state">
            <h2 class="welcome-title">开始一段更清晰的业务对话</h2>
            <p class="welcome-copy">
              你可以查询产品信息、观察结果、产品文档，或者直接让助手帮你做业务分析。
            </p>

            <div class="suggestion-grid">
              <button v-for="s in suggestions" :key="s" class="suggestion-card" @click="sendSuggestion(s)">
                {{ s }}
              </button>
            </div>
          </div>

          <template v-for="msg in messages" :key="msg.id || msg._tempId">
            <ChatMessage
              :role="msg.role"
              :content="msg.content"
              :streaming="msg.streaming"
              :tool-calls="msg.tool_calls_display"
            />
          </template>

          <div v-if="isLoading && !streaming" class="loading-indicator">
            <div class="dot-pulse"></div>
          </div>
        </div>

        <footer class="composer-wrap">
          <div class="composer-panel">
            <form class="input-form" @submit.prevent="sendMessage">
              <textarea
                ref="inputRef"
                v-model="inputText"
                class="chat-input"
                rows="1"
                placeholder="给智能体发送消息"
                @keydown="handleKeydown"
                @input="autoResize"
              ></textarea>
              <button type="submit" class="send-btn" :disabled="!inputText.trim() || isLoading">
                <SendHorizontal :size="18" />
              </button>
            </form>
            <div class="composer-hint">Enter 发送，Shift + Enter 换行</div>
          </div>
        </footer>
      </section>
    </div>
  </WorkbenchLayout>
</template>

<script setup>
import { nextTick, onMounted, ref } from 'vue'
import WorkbenchLayout from '../components/WorkbenchLayout.vue'
import ChatMessage from '../components/ChatMessage.vue'
import {
  MessageSquare,
  PanelRightClose,
  PanelRightOpen,
  Plus,
  SendHorizontal,
  Trash2,
} from '@lucide/vue'

const conversations = ref([])
const currentConversationId = ref(null)
const messages = ref([])
const inputText = ref('')
const isLoading = ref(false)
const streaming = ref(false)
const sidebarCollapsed = ref(false)
const messagesContainer = ref(null)
const inputRef = ref(null)

const suggestions = [
  '目前有多少存续产品？',
  '今天有哪些产品需要观察？',
  '帮我总结最近的产品观察结果',
  '中证1000 相关产品有哪些？',
]

async function loadConversations() {
  try {
    const res = await fetch('/api/agent/conversations')
    if (res.ok) conversations.value = await res.json()
  } catch {}
}

async function selectConversation(id) {
  currentConversationId.value = id
  try {
    const res = await fetch(`/api/agent/conversations/${id}/messages`)
    if (res.ok) {
      const msgs = await res.json()
      messages.value = msgs.map((m) => {
        let display = null
        if (m.tool_calls) {
          const raw = typeof m.tool_calls === 'string' ? JSON.parse(m.tool_calls) : m.tool_calls
          if (Array.isArray(raw)) {
            display = raw.map((tc) => ({
              name: tc.function?.name || tc.name || tc,
              status: 'done',
            }))
          }
        }
        return {
          ...m,
          tool_calls: null,
          tool_calls_display: display,
          streaming: false,
        }
      })
    }
  } catch {}
  scrollToBottom()
}

function newConversation() {
  currentConversationId.value = null
  messages.value = []
  inputText.value = ''
  inputRef.value?.focus()
}

async function deleteConversation(id) {
  try {
    await fetch(`/api/agent/conversations/${id}`, { method: 'DELETE' })
    conversations.value = conversations.value.filter((c) => c.id !== id)
    if (currentConversationId.value === id) {
      currentConversationId.value = null
      messages.value = []
    }
  } catch {}
}

function sendSuggestion(text) {
  inputText.value = text
  sendMessage()
}

async function sendMessage() {
  const text = inputText.value.trim()
  if (!text || isLoading.value) return

  inputText.value = ''
  isLoading.value = true
  streaming.value = true

  const tempId = `temp-${Date.now()}`
  messages.value.push({ _tempId: tempId, role: 'user', content: text, streaming: false })
  scrollToBottom()

  const assistantMsgId = `assistant-${Date.now()}`
  messages.value.push({
    _tempId: assistantMsgId,
    role: 'assistant',
    content: '',
    reasoning: '',
    streaming: true,
    tool_calls_display: [],
    hasContent: false,
  })

  try {
    const res = await fetch('/api/agent/chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        conversation_id: currentConversationId.value,
        message: text,
      }),
    })

    if (!res.ok) {
      const err = await res.json()
      throw new Error(err.error || 'Request failed')
    }

    const reader = res.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop()

      for (const line of lines) {
        if (!line.startsWith('data: ')) continue
        const data = line.slice(6).trim()
        if (!data) continue

        let event
        try {
          event = JSON.parse(data)
        } catch {
          continue
        }

        if (event.type === 'conversation_id') {
          currentConversationId.value = event.conversation_id
        } else if (event.type === 'reasoning_delta') {
          const msg = messages.value.find((m) => m._tempId === assistantMsgId)
          if (msg) msg.reasoning = (msg.reasoning || '') + event.text
          scrollToBottom()
        } else if (event.type === 'delta') {
          const msg = messages.value.find((m) => m._tempId === assistantMsgId)
          if (msg) {
            if (!msg.hasContent) {
              msg.content = ''
              msg.hasContent = true
            }
            msg.content += event.text
          }
          scrollToBottom()
        } else if (event.type === 'tool_call') {
          const msg = messages.value.find((m) => m._tempId === assistantMsgId)
          if (msg) {
            if (!msg.tool_calls_display) msg.tool_calls_display = []
            msg.tool_calls_display.push({ name: event.name, status: 'calling' })
          }
          scrollToBottom()
        } else if (event.type === 'tool_done') {
          const msg = messages.value.find((m) => m._tempId === assistantMsgId)
          if (msg && msg.tool_calls_display) {
            const tc = [...msg.tool_calls_display].reverse().find((t) => t.name === event.name && t.status === 'calling')
            if (tc) tc.status = 'done'
          }
        } else if (event.type === 'done') {
          streaming.value = false
          const msg = messages.value.find((m) => m._tempId === assistantMsgId)
          if (msg) {
            msg.streaming = false
            if (!msg.hasContent && !msg.content.trim()) {
              const hasTools = msg.tool_calls_display && msg.tool_calls_display.length > 0
              if (!hasTools) {
                msg.content = '已完成，但模型没有返回可展示的正文。'
              }
            }
          }
        } else if (event.type === 'error') {
          const msg = messages.value.find((m) => m._tempId === assistantMsgId)
          if (msg) {
            msg.content = `错误：${event.error}`
            msg.streaming = false
          }
          streaming.value = false
        }
      }
    }
  } catch (err) {
    const msg = messages.value.find((m) => m._tempId === assistantMsgId)
    if (msg) {
      msg.content = `请求失败：${err.message}`
      msg.streaming = false
      msg.hasContent = true
    }
  } finally {
    isLoading.value = false
    streaming.value = false
    const msg = messages.value.find((m) => m._tempId === assistantMsgId)
    if (msg) msg.streaming = false
    await loadConversations()
  }
}

function handleKeydown(e) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    sendMessage()
  }
}

function autoResize() {
  const el = inputRef.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = `${Math.min(el.scrollHeight, 180)}px`
}

function scrollToBottom() {
  nextTick(() => {
    const container = messagesContainer.value
    if (container) container.scrollTop = container.scrollHeight
  })
}

onMounted(() => {
  loadConversations()
  inputRef.value?.focus()
})
</script>

<style scoped>
:deep(.workbench-shell) {
  height: auto;
  flex: 1;
  min-height: 0;
}

:deep(.workbench-content) {
  flex: 1;
  min-height: 0;
}

:deep(.workbench-main) {
  flex: 1;
  min-height: 0;
}

.agent-shell {
  display: grid;
  grid-template-columns: 300px minmax(0, 1fr);
  gap: 20px;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.conversation-sidebar {
  display: flex;
  flex-direction: column;
  border: 1px solid rgba(15, 23, 42, 0.07);
  border-radius: 24px;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.96), rgba(247, 249, 252, 0.96));
  box-shadow: 0 24px 60px rgba(15, 23, 42, 0.06);
  overflow: hidden;
  transition: width 180ms ease, opacity 180ms ease;
}

.conversation-sidebar.collapsed {
  width: 72px;
}

.sidebar-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 18px 18px 12px;
}

.sidebar-title {
  font-size: 14px;
  font-weight: 700;
  color: #3d4a5d;
  letter-spacing: 0.01em;
}

.icon-btn {
  width: 36px;
  height: 36px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.84);
  color: #536176;
}

.new-chat-btn {
  margin: 0 18px 16px;
  height: 42px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: none;
  border-radius: 14px;
  background: linear-gradient(135deg, #3f7cff, #2f6cf6);
  color: #fff;
  font-size: 14px;
  font-weight: 700;
  box-shadow: 0 12px 30px rgba(47, 108, 246, 0.22);
}

.conversation-list {
  flex: 1;
  overflow-y: auto;
  padding: 0 12px 14px;
}

.conversation-item {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 12px 12px;
  border: none;
  border-radius: 16px;
  background: transparent;
  color: #5c697d;
  text-align: left;
  transition: background 140ms ease, color 140ms ease, transform 140ms ease;
}

.conversation-item:hover {
  background: rgba(255, 255, 255, 0.88);
  color: #202a39;
  transform: translateY(-1px);
}

.conversation-item.active {
  background: linear-gradient(135deg, rgba(63, 124, 255, 0.12), rgba(90, 149, 255, 0.08));
  color: #1f3f86;
}

.conversation-main {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.conv-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 600;
}

.conv-delete {
  opacity: 0;
  color: #8e98a9;
  display: inline-flex;
  align-items: center;
}

.conversation-item:hover .conv-delete {
  opacity: 1;
}

.empty-conversations {
  padding: 28px 12px;
  text-align: center;
  color: #98a3b3;
  font-size: 13px;
}

.chat-stage {
  display: flex;
  flex-direction: column;
  min-width: 0;
  border: 1px solid rgba(15, 23, 42, 0.06);
  border-radius: 28px;
  background:
    radial-gradient(circle at top, rgba(255, 255, 255, 0.92), rgba(245, 247, 250, 0.95) 58%, rgba(241, 244, 248, 0.98));
  box-shadow: 0 28px 64px rgba(15, 23, 42, 0.06);
  overflow: hidden;
}

.messages-area {
  flex: 1;
  overflow-y: auto;
  padding: 10px 30px 12px;
}

.welcome-state {
  min-height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 52px 20px 64px;
  text-align: center;
}

.welcome-title {
  font-size: clamp(32px, 4vw, 48px);
  line-height: 1.06;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: #161f2d;
  max-width: 820px;
}

.welcome-copy {
  margin-top: 16px;
  max-width: 620px;
  color: #6d7b90;
  font-size: 16px;
  line-height: 1.75;
}

.suggestion-grid {
  width: 100%;
  max-width: 760px;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  margin-top: 28px;
}

.suggestion-card {
  min-height: 74px;
  padding: 18px 18px;
  text-align: left;
  border: 1px solid rgba(15, 23, 42, 0.07);
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.9);
  color: #2a3546;
  font-size: 14px;
  line-height: 1.55;
  font-weight: 600;
  transition: transform 160ms ease, box-shadow 160ms ease, border-color 160ms ease;
}

.suggestion-card:hover {
  transform: translateY(-2px);
  border-color: rgba(47, 108, 246, 0.18);
  box-shadow: 0 18px 36px rgba(15, 23, 42, 0.05);
}

.composer-wrap {
  padding: 14px 24px 24px;
}

.composer-panel {
  max-width: 940px;
  margin: 0 auto;
  padding: 14px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.95);
  box-shadow: 0 20px 44px rgba(15, 23, 42, 0.06);
}

.input-form {
  display: flex;
  gap: 12px;
  align-items: flex-end;
}

.chat-input {
  width: 100%;
  min-height: 52px;
  max-height: 180px;
  padding: 14px 16px;
  border: none;
  outline: none;
  resize: none;
  background: transparent;
  color: #1f2937;
  font-size: 15px;
  line-height: 1.7;
  font-family: var(--font-sans);
}

.chat-input::placeholder {
  color: #9aa4b4;
}

.send-btn {
  width: 48px;
  height: 48px;
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 16px;
  background: linear-gradient(135deg, #3f7cff, #2f6cf6);
  color: #fff;
  box-shadow: 0 14px 28px rgba(47, 108, 246, 0.25);
}

.send-btn:disabled {
  opacity: 0.44;
  box-shadow: none;
}

.composer-hint {
  padding: 8px 6px 2px;
  color: #9aa4b4;
  font-size: 12px;
}

.loading-indicator {
  padding: 20px 0;
  display: flex;
  justify-content: center;
}

.dot-pulse {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #2f6cf6;
  animation: pulse 1.2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% {
    transform: scale(0.8);
    opacity: 0.4;
  }
  50% {
    transform: scale(1.2);
    opacity: 1;
  }
}

@media (max-width: 1080px) {
  .agent-shell {
    grid-template-columns: 1fr;
  }

  .conversation-sidebar {
    display: none;
  }
}

@media (max-width: 720px) {
  .chat-topbar {
    padding: 20px 18px 8px;
  }

  .messages-area {
    padding: 8px 16px 10px;
  }

  .welcome-title {
    font-size: 30px;
  }

  .welcome-copy {
    font-size: 14px;
  }

  .suggestion-grid {
    grid-template-columns: 1fr;
  }

  .composer-wrap {
    padding: 10px 12px 14px;
  }
}
</style>
