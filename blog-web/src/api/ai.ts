import { sseFetch } from './sseFetch'

export interface Citation {
  index: number
  chunk_id?: string
  article_id?: number
  title?: string
  url?: string
  content_excerpt?: string
  score?: number
}

export interface ToolEvent {
  name?: string
  status?: string
  input?: unknown
  output?: unknown
  [key: string]: unknown
}

export interface AIRoute {
  intent?: string
  reason?: string
}

export interface AIStreamHandlers {
  onStart?: () => void
  onDelta: (content: string) => void
  onCitation: (citation: Citation) => void
  onTool: (tool: ToolEvent) => void
  onRoute?: (route: AIRoute) => void
  onDone?: () => void
  onError?: (message: string) => void
}

export function streamAIChat(
  question: string,
  sessionId: string,
  token: string,
  signal: AbortSignal,
  handlers: AIStreamHandlers,
) {
  return sseFetch('/api/ai/assistant/stream', { question, session_id: sessionId }, {
    signal,
    headers: token ? { Authorization: token } : {},
    onEvent({ event, data }) {
      const payload = (data && typeof data === 'object' ? data : {}) as Record<string, any>
      if (event === 'start') handlers.onStart?.()
      if (event === 'delta') handlers.onDelta(String(payload.content || ''))
      if (event === 'citation') handlers.onCitation((payload.data || payload) as Citation)
      if (event === 'tool') handlers.onTool((payload.data || payload) as ToolEvent)
      if (event === 'route') handlers.onRoute?.((payload.data || payload) as AIRoute)
      if (event === 'done') handlers.onDone?.()
      if (event === 'error') {
        const message = payload.error?.message || payload.message || 'AI 服务暂时不可用'
        handlers.onError?.(message)
      }
    },
  })
}
