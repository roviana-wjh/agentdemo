import { sseFetch } from './sseFetch'

export type InsightKind = 'summary' | 'comments' | 'analytics'

const paths: Record<InsightKind, string> = {
  summary: '/api/ai/articles/summary/stream',
  comments: '/api/ai/comments/analyze/stream',
  analytics: '/api/ai/analytics/stream',
}

export function streamInsight(
  kind: InsightKind,
  input: Record<string, unknown>,
  token: string,
  signal: AbortSignal,
  onDelta: (content: string) => void,
) {
  return sseFetch(paths[kind], input, {
    signal,
    headers: { Authorization: token },
    onEvent({ event, data }) {
      const payload = data as Record<string, any>
      if (event === 'delta') onDelta(String(payload.content || ''))
      if (event === 'error') throw new Error(payload.error?.message || '分析服务暂时不可用')
    },
  })
}
