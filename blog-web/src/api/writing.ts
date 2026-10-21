import { sseFetch } from './sseFetch'

export function streamWriting(
  topic: string,
  requirements: string,
  token: string,
  signal: AbortSignal,
  onDelta: (content: string) => void,
) {
  return sseFetch('/api/ai/writing/stream', { topic, requirements }, {
    signal,
    headers: { Authorization: token },
    onEvent({ event, data }) {
      const payload = data as Record<string, any>
      if (event === 'delta') onDelta(String(payload.content || ''))
      if (event === 'error') throw new Error(payload.error?.message || '写作服务暂时不可用')
    },
  })
}

export async function saveWritingDraft(input: {
  article_id?: number
  title: string
  content: string
  confirmed: true
}, token: string) {
  const response = await fetch('/api/ai/drafts/save', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: token },
    body: JSON.stringify(input),
  })
  const body = await response.json().catch(() => ({}))
  if (!response.ok) throw new Error(body.error || '保存草稿失败')
  return body.data as { id: number; title: string; version: number; status: 'draft' }
}
