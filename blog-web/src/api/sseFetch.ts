export type SSEEventName = 'start' | 'delta' | 'citation' | 'tool' | 'route' | 'done' | 'error'

export interface SSEMessage<T = unknown> {
  event: SSEEventName | string
  data: T
}

interface StreamOptions {
  signal?: AbortSignal
  headers?: HeadersInit
  onEvent: (message: SSEMessage) => void
}

export async function sseFetch(url: string, body: unknown, options: StreamOptions) {
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      Accept: 'text/event-stream',
      'Content-Type': 'application/json',
      ...options.headers,
    },
    body: JSON.stringify(body),
    signal: options.signal,
  })

  if (!response.ok || !response.body) {
    const detail = await response.text().catch(() => '')
    throw new Error(detail || `AI 服务请求失败（${response.status}）`)
  }
  if (!response.headers.get('content-type')?.includes('text/event-stream')) {
    throw new Error('AI 服务返回了非流式响应')
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  while (true) {
    const { done, value } = await reader.read()
    buffer += decoder.decode(value || new Uint8Array(), { stream: !done })
    const blocks = buffer.split(/\r?\n\r?\n/)
    buffer = blocks.pop() || ''
    for (const block of blocks) parseBlock(block, options.onEvent)
    if (done) break
  }
  if (buffer.trim()) parseBlock(buffer, options.onEvent)
}

function parseBlock(block: string, onEvent: StreamOptions['onEvent']) {
  let event = 'message'
  const dataLines: string[] = []
  for (const line of block.split(/\r?\n/)) {
    if (line.startsWith(':')) continue
    if (line.startsWith('event:')) event = line.slice(6).trim()
    if (line.startsWith('data:')) dataLines.push(line.slice(5).trimStart())
  }
  if (!dataLines.length) return
  const raw = dataLines.join('\n')
  let data: unknown = raw
  try {
    data = JSON.parse(raw)
  } catch {
    // Preserve non-JSON upstream errors for diagnostics.
  }
  onEvent({ event, data })
}
