import { computed, reactive } from 'vue'
import { streamAIChat, type Citation, type ToolEvent } from '../api/ai'

export interface ChatMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  citations: Citation[]
  tools: ToolEvent[]
  createdAt: number
}

const savedToken = localStorage.getItem('blog_token') || ''
const state = reactive({
  open: false,
  token: savedToken,
  sessionId: crypto.randomUUID(),
  messages: [] as ChatMessage[],
  streaming: false,
  error: '',
  lastQuestion: '',
  controller: null as AbortController | null,
})

export function useChatStore() {
  const canChat = computed(() => Boolean(state.token))

  function setToken(token: string) {
    state.token = token
    localStorage.setItem('blog_token', token)
  }

  function clearToken() {
    state.token = ''
    localStorage.removeItem('blog_token')
  }

  async function send(question: string) {
    const cleanQuestion = question.trim()
    if (!cleanQuestion || state.streaming) return
    if (!state.token) {
      state.error = '请先登录，再使用 AI 助手。'
      return
    }
    state.error = ''
    state.lastQuestion = cleanQuestion
    state.messages.push(createMessage('user', cleanQuestion))
    const answer = createMessage('assistant', '')
    state.messages.push(answer)
    state.streaming = true
    state.controller = new AbortController()
    try {
      await streamAIChat(cleanQuestion, state.sessionId, state.token, state.controller.signal, {
        onDelta: (content) => { answer.content += content },
        onCitation: (citation) => { answer.citations.push(citation) },
        onTool: (tool) => { answer.tools.push(tool) },
        onRoute: (route) => {
          if (route.intent) answer.tools.push({ name: `路由：${route.intent}`, status: '已分类' })
        },
        onError: (message) => { state.error = message },
      })
    } catch (error) {
      if ((error as Error).name !== 'AbortError') state.error = (error as Error).message
    } finally {
      state.streaming = false
      state.controller = null
      if (!answer.content && !answer.citations.length && !answer.tools.length) {
        state.messages = state.messages.filter((message) => message.id !== answer.id)
      }
    }
  }

  function stop() {
    state.controller?.abort()
    state.streaming = false
  }

  function clear() {
    stop()
    state.messages = []
    state.error = ''
    state.lastQuestion = ''
    state.sessionId = crypto.randomUUID()
  }

  function retry() {
    if (state.lastQuestion && !state.streaming) {
      const trailingUser = state.messages.at(-1)?.role === 'user'
      if (trailingUser) state.messages.pop()
      return send(state.lastQuestion)
    }
  }

  return { state, canChat, setToken, clearToken, send, stop, clear, retry }
}

function createMessage(role: ChatMessage['role'], content: string): ChatMessage {
  return {
    id: crypto.randomUUID(),
    role,
    content,
    citations: [],
    tools: [],
    createdAt: Date.now(),
  }
}
