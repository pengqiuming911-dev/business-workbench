const { TOOL_DEFINITIONS, executeTool } = require('./agentTools')
const { searchDocs, buildDocContext } = require('./documentRetriever')
const dbModule = require('../db')

const DEEPSEEK_API_URL = process.env.DEEPSEEK_API_URL || 'https://api.deepseek.com'
const DEEPSEEK_MODEL = process.env.DEEPSEEK_MODEL || 'deepseek-chat'
const DEEPSEEK_API_KEY = process.env.DEEPSEEK_API_KEY || ''
const MAX_TOOL_ROUNDS = 5

const SYSTEM_PROMPT = `你是一个专业的金融结构化产品业务助手，服务于"航班服务"业务工作台系统。

你的职责：
1. 回答关于结构化金融产品的专业知识问题（如敲出、派息、观察日、降落伞等概念）
2. 查询具体产品的状态、观察结果、标的价格等业务数据
3. 帮助用户理解如何使用业务工作台系统
4. 提供产品分析和对比的辅助决策建议

注意事项：
- 回答要准确，基于实际的系统数据和文档
- 对于金额、比率等数据，保留原始精度
- 如果不确定，明确告知用户并建议使用系统中的其他功能
- 使用中文回答`

async function streamChat({ conversationId, userMessage, abortSignal, onDelta, onToolCall, onDone, onError }) {
  try {
    const messages = await buildMessageHistory(conversationId, userMessage)

    let round = 0
    while (round < MAX_TOOL_ROUNDS) {
      round++
      if (abortSignal?.aborted) {
        throw new Error('Client disconnected')
      }
      const result = await callDeepSeek(messages, abortSignal, onDelta, onToolCall)

      if (result.finishReason === 'tool_calls') {
        if (result.toolCalls.length === 0) {
          throw new Error('Model returned finish_reason=tool_calls but no valid tool calls were parsed')
        }
        const assistantMsg = { role: 'assistant', content: result.content || '', tool_calls: result.toolCalls }
        messages.push(assistantMsg)
        dbModule.addAgentMessage({ conversation_id: conversationId, role: 'assistant', content: result.content || '', tool_calls: result.toolCalls })

        for (const tc of result.toolCalls) {
          let args
          try {
            args = JSON.parse(tc.function.arguments || '{}')
          } catch {
            const errMsg = `Failed to parse tool arguments: ${tc.function.arguments}`
            messages.push({ role: 'tool', tool_call_id: tc.id, content: JSON.stringify({ error: errMsg }) })
            dbModule.addAgentMessage({ conversation_id: conversationId, role: 'tool', content: JSON.stringify({ error: errMsg }), tool_call_id: tc.id })
            continue
          }
          if (onToolCall) onToolCall(tc.function.name, args)
          const toolResult = await executeTool(tc.function.name, args)
          messages.push({
            role: 'tool',
            tool_call_id: tc.id,
            content: JSON.stringify(toolResult),
          })
          dbModule.addAgentMessage({ conversation_id: conversationId, role: 'tool', content: JSON.stringify(toolResult), tool_call_id: tc.id })
        }
        continue
      }

      dbModule.addAgentMessage({
        conversation_id: conversationId,
        role: 'assistant',
        content: result.content,
        tool_calls: result.toolCalls.length > 0 ? result.toolCalls : null,
      })

      if (onDone) onDone(result.usage)
      return
    }

    throw new Error('Max tool call rounds exceeded')
  } catch (err) {
    if (onError) onError(err)
  }
}

async function buildMessageHistory(conversationId, userMessage) {
  const dbMessages = dbModule.getAgentMessages(conversationId)

  const historyMessages = dbMessages.map(m => {
    const msg = { role: m.role, content: m.content }
    if (m.tool_calls) {
      try { msg.tool_calls = JSON.parse(m.tool_calls) } catch {}
    }
    if (m.tool_call_id) msg.tool_call_id = m.tool_call_id
    return msg
  })

  const relevantDocs = searchDocs(userMessage)
  const docContext = buildDocContext(relevantDocs)
  const systemContent = SYSTEM_PROMPT + docContext

  return [
    { role: 'system', content: systemContent },
    ...historyMessages,
    { role: 'user', content: userMessage },
  ]
}

async function callDeepSeek(messages, abortSignal, onDelta, onToolCall) {
  const body = {
    model: DEEPSEEK_MODEL,
    messages,
    tools: TOOL_DEFINITIONS,
    stream: true,
    stream_options: { include_usage: true },
  }

  const response = await fetch(`${DEEPSEEK_API_URL}/chat/completions`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${DEEPSEEK_API_KEY}`,
    },
    body: JSON.stringify(body),
    signal: abortSignal,
  })

  if (!response.ok) {
    const text = await response.text()
    throw new Error(`DeepSeek API error ${response.status}: ${text}`)
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()

  let content = ''
  let toolCalls = []
  let finishReason = null
  let usage = null
  let sseBuffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) {
      if (sseBuffer.trim()) {
        const lastLine = sseBuffer.trim()
        if (lastLine.startsWith('data: ') && lastLine.slice(6) !== '[DONE]') {
          try {
            const parsed = JSON.parse(lastLine.slice(6))
            if (parsed.usage) usage = parsed.usage
          } catch {}
        }
      }
      break
    }

    sseBuffer += decoder.decode(value, { stream: true })
    const lines = sseBuffer.split('\n')
    sseBuffer = lines.pop()

    for (const line of lines) {
      const trimmed = line.trim()
      if (!trimmed || !trimmed.startsWith('data: ')) continue
      const data = trimmed.slice(6)
      if (data === '[DONE]') {
        sseBuffer = ''
        continue
      }

      let parsed
      try { parsed = JSON.parse(data) } catch { continue }

      if (parsed.usage) {
        usage = parsed.usage
      }

      const choice = parsed.choices?.[0]
      if (!choice) continue

      const delta = choice.delta || {}

      if (delta.content) {
        content += delta.content
        if (onDelta) onDelta(delta.content)
      }

      if (delta.tool_calls) {
        for (const tc of delta.tool_calls) {
          const idx = tc.index
          if (!toolCalls[idx]) {
            toolCalls[idx] = {
              id: tc.id || '',
              type: 'function',
              function: { name: '', arguments: '' },
            }
          }
          if (tc.id) toolCalls[idx].id = tc.id
          if (tc.function?.name) toolCalls[idx].function.name += tc.function.name
          if (tc.function?.arguments) toolCalls[idx].function.arguments += tc.function.arguments
        }
      }

      if (choice.finish_reason) {
        finishReason = choice.finish_reason
      }
    }
  }

  toolCalls = toolCalls.filter(Boolean)
  return { content, toolCalls, finishReason, usage }
}

module.exports = { streamChat }
