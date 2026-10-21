export interface Article {
  ID: number
  title: string
  content: string
  image?: string
  author_id?: number
  view_count?: number
  published_at?: string
  CreatedAt?: string
	version?: number
	review_status?: ReviewStatus
	reviewed_at?: string
}

export type ReviewStatus = 'not_reviewed' | 'approved' | 'needs_revision' | 'rejected' | 'failed'

export interface ReviewIssue {
	type: string
	message: string
	severity: string
	keyword?: string
}

export interface ArticleReviewResult {
	article_id: number
	version: number
	content_hash: string
	status: ReviewStatus
	risk_level: 'low' | 'medium' | 'high' | 'unknown'
	score: number
	reasons: string[]
	suggestions: string[]
	rule_hits: ReviewIssue[]
}

export interface Comment {
  id: number
  content: string
  article_id: number
  created_at: string
  user?: { name?: string }
}

export async function listArticles(page = 1, pageSize = 12) {
  return request<{ total: number; list: Article[] }>(`/api/article/list?page=${page}&pageSize=${pageSize}`)
}

export async function getArticle(id: number) {
  return request<{ article: Article }>(`/api/article/${id}?id=${id}`)
}

export async function login(username: string, password: string) {
  const response = await request<{ state: string; msg: string; token?: string }>('/api/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  })
  if (response.state !== 'success' || !response.token) throw new Error(response.msg || '登录失败')
  return response
}

export async function logout(token: string) {
  const response = await fetch('/api/logout', {
    method: 'POST',
    headers: token ? { Authorization: token } : {},
  })
  return response.json().catch(() => ({}))
}

export async function listDrafts(token: string) {
  return request<{ list: Article[] }>('/api/article/drafts', {
    headers: { Authorization: token },
  })
}

export async function publishArticle(id: number, token: string) {
  return request<{ msg: string; id: number; index_status?: string; index_error?: string }>('/api/article/publish', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: token },
    body: JSON.stringify({ id }),
  })
}

export async function reviewArticle(id: number, token: string) {
	return request<{ data: ArticleReviewResult; reviewed_at: string }>(`/api/article/${id}/review`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json', Authorization: token },
	})
}

export async function listComments(articleID: number) {
  return request<{ comments: Comment[] }>(`/api/comment/list?article_id=${articleID}`)
}

export async function createComment(articleID: number, content: string, token: string) {
  return request<{ msg: string }>('/api/comment/create', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: token },
    body: JSON.stringify({ article_id: articleID, content }),
  })
}

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const response = await fetch(url, options)
  const body = await response.json().catch(() => ({}))
  if (!response.ok) throw new Error(body.msg || body.error || `请求失败（${response.status}）`)
  return body as T
}
