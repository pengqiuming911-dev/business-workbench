<template>
  <WorkbenchLayout wide>
    <div class="agent-layout">
      <div class="conversation-sidebar" :class="{ collapsed: sidebarCollapsed }">
        <div class="sidebar-header">
          <button class="btn btn-primary btn-sm new-chat-btn" @click="newConversation">
            <Plus :size="16" /> 新对话
          </button>
          <button class="btn btn-outline btn-sm toggle-btn" @click="sidebarCollapsed = !sidebarCollapsed">
            <component :is="sidebarCollapsed ? PanelRightOpen : PanelRightClose" :size="16" />
          </button>
        </div>
        <div v-if="!sidebarCollapsed" class="conversation-list">
          <div
            v-for="conv in conversations"
            :key="conv.id"
            class="conversation-item"
            :class="{ active: currentConversationId === conv.id }"
            @click="selectConversation(conv.id)"
          >
            <MessageSquare :size="15" />
            <span class="conv-title">{{ conv.title }}</span>
            <button class="conv-delete" @click.stop="deleteConversation(conv.id)" title="删除">
              <Trash2 :size="13" />
            </button>
          </div>
          <div v-if="conversations.length === 0" class="empty-conversations">
            暂无对话记录
          </div>
        </div>
      </div>

      <div class="chat-main">
        <div ref="messagesContainer" class="messages-area">
          <div v-if="messages.length === 0" class="welcome-state">
            <Bot :size="48" :stroke-width="1.5" class="welcome-icon" />
            <h2 class="text-section">智能业务助手</h2>
            <p class="text-body" style="color:var(--ink-soft);max-width:400px;text-align:center">
              我可以帮你查询产品信息、分析观察结果、解读产品文档，或回答业务相关问题。试试问我：
            </p>
            <div class="suggestion-chips">
              <button v-for="s in suggestions" :key="s" class="suggestion-chip" @click="sendSuggestion(s)">
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

        <div class="input-area">
          <form class="input-form" @submit.prevent="sendMessage">
            <textarea
              ref="inputRef"
              v-model="inputText"
              class="input chat-input"
              rows="1"
              placeholder="输入你的问题..."
              @keydown="handleKeydown"
              @input="autoResize"
            ></textarea>
            <button type="submit" class="btn btn-primary send-btn" :disabled="!inputText.trim() || isLoading">
              <SendHorizontal :size="18" />
            </button>
          </form>
        </div>
      </div>
    </div>
  </WorkbenchLayout>
</template>

<script setup>
import { ref, onMounted, nextTick } from 'vue'
import WorkbenchLayout from '../components/WorkbenchLayout.vue'
import ChatMessage from '../components/ChatMessage.vue'
import {
  Bot, Plus, MessageSquare, Trash2, SendHorizontal,
  PanelRightOpen, PanelRightClose,
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
  '沪深300最新价格是多少？',
  '什么是敲出？什么是派息？',
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
      messages.value = msgs.map(m => ({
        ...m,
        tool_calls: null,
        tool_calls_display: m.tool_calls ? (typeof m.tool_calls === 'string' ? JSON.parse(m.tool_calls) : m.tool_calls) : null,
        streaming: false,
      }))
    }
  } catch {}
  scrollToBottom()
}

async function newConversation() {
  currentConversationId.value = null
  messages.value = []
  inputText.value = ''
  inputRef.value?.focus()
}

async function deleteConversation(id) {
  try {
    await fetch(`/api/agent/conversations/${id}`, { method: 'DELETE' })
    conversations.value = conversations.value.filter(c => c.id !== id)
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
  if (!text || isLoading) return

  inputText.value = ''
  isLoading.value = true
  streaming.value = true

  const tempId = `temp-${Date.now()}`
  messages.value.push({ _tempId: tempId, role: 'user', content: text, streaming: false })
  scrollToBottom()

  const assistantMsgId = `assistant-${Date.now()}`
  messages.value.push({ _tempId: assistantMsgId, role: 'assistant', content: '', streaming: true, tool_calls_display: [] })

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
        try { event = JSON.parse(data) } catch { continue }

        if (event.type === 'conversation_id') {
          currentConversationId.value = event.conversation_id
        } else if (event.type === 'delta') {
          const msg = messages.value.find(m => m._tempId === assistantMsgId)
          if (msg) msg.content += event.text
          scrollToBottom()
        } else if (event.type === 'tool_call') {
          const msg = messages.value.find(m => m._tempId === assistantMsgId)
          if (msg) {
            if (!msg.tool_calls_display) msg.tool_calls_display = []
            msg.tool_calls_display.push({ function: { name: event.name } })
          }
        } else if (event.type === 'done') {
          streaming.value = false
          const msg = messages.value.find(m => m._tempId === assistantMsgId)
          if (msg) msg.streaming = false
        } else if (event.type === 'error') {
          const msg = messages.value.find(m => m._tempId === assistantMsgId)
          if (msg) {
            msg.content = `⚠ 错误：${event.error}`
            msg.streaming = false
          }
          streaming.value = false
        }
      }
    }
  } catch (err) {
    const msg = messages.value.find(m => m._tempId === assistantMsgId)
    if (msg) {
      msg.content = `⚠ 请求失败：${err.message}`
      msg.streaming = false
    }
  } finally {
    isLoading.value = false
    streaming.value = false
    const msg = messages.value.find(m => m._tempId === assistantMsgId)
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
  el.style.height = Math.min(el.scrollHeight, 150) + 'px'
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
.agent-layout {
  display: flex;
  height: calc(100vh - 80px);
  gap: 0;
}

.conversation-sidebar {
  width: 260px;
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  transition: width 200ms ease;
}

.conversation-sidebar.collapsed {
  width: 48px;
}

.sidebar-header {
  display: flex;
  gap: 6px;
  padding: 12px;
  border-bottom: 1px solid var(--border-soft);
}

.new-chat-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.toggle-btn {
  padding: 6px;
}

.conversation-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.conversation-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 13px;
  color: var(--ink);
  transition: background 120ms;
}

.conversation-item:hover {
  background: var(--bg-hover);
}

.conversation-item.active {
  background: var(--bg-active);
}

.conv-title {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.conv-delete {
  opacity: 0;
  background: none;
  border: none;
  color: var(--ink-faint);
  cursor: pointer;
  padding: 2px;
  display: flex;
}

.conversation-item:hover .conv-delete {
  opacity: 1;
}

.empty-conversations {
  text-align: center;
  padding: 24px 12px;
  color: var(--ink-faint);
  font-size: 13px;
}

.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.messages-area {
  flex: 1;
  overflow-y: auto;
  padding: 16px 32px;
}

.welcome-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  gap: 12px;
}

.welcome-icon {
  color: var(--brand);
  margin-bottom: 8px;
}

.suggestion-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: center;
  margin-top: 16px;
}

.suggestion-chip {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 20px;
  padding: 8px 16px;
  font-size: 13px;
  color: var(--ink);
  cursor: pointer;
  transition: all 120ms;
}

.suggestion-chip:hover {
  background: var(--brand-soft);
  border-color: var(--brand);
  color: var(--brand);
}

.input-area {
  padding: 12px 32px 16px;
  border-top: 1px solid var(--border-soft);
}

.input-form {
  display: flex;
  gap: 8px;
  align-items: flex-end;
}

.chat-input {
  flex: 1;
  resize: none;
  min-height: 40px;
  max-height: 150px;
  line-height: 1.5;
  font-family: var(--font-sans);
}

.send-btn {
  height: 40px;
  width: 40px;
  padding: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.loading-indicator {
  padding: 16px 0;
  display: flex;
  justify-content: center;
}

.dot-pulse {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--brand);
  animation: pulse 1.2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { transform: scale(0.8); opacity: 0.4; }
  50% { transform: scale(1.2); opacity: 1; }
}

@media (max-width: 860px) {
  .conversation-sidebar {
    display: none;
  }

  .messages-area {
    padding: 12px 16px;
  }

  .input-area {
    padding: 8px 16px 12px;
  }
}
</style>
