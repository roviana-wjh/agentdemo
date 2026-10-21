/* No-build CDN entry. The TypeScript/.vue source remains the production build path. */
const { createApp, computed, nextTick, onMounted, reactive, ref, watch } = Vue
const apiBase = String(window.BLOG_API_URL || '').replace(/\/$/, '')

const demoArticles = [
  { ID: 101, title: '用 Go 构建一个会记忆的 AI Agent', content: '从流式协议、会话记忆到工具调用，一步步搭建可靠的 Agent 服务。', view_count: 1284, published_at: '2026-07-18T10:00:00Z' },
  { ID: 102, title: 'RAG 不只是向量搜索：混合检索实践', content: 'BM25、向量召回、RRF 与 Rerank 如何一起工作。', view_count: 968, published_at: '2026-07-15T10:00:00Z' },
  { ID: 103, title: '把复杂系统，写成简单的故事', content: '技术写作的结构、节奏，以及如何让读者真正理解一个系统。', view_count: 726, published_at: '2026-07-11T10:00:00Z' },
]

async function requestJSON(path, options) {
  const response = await fetch(apiBase + path, options)
  const body = await response.json().catch(() => ({}))
  if (!response.ok) throw new Error(body.msg || body.error || `请求失败（${response.status}）`)
  return body
}

async function sseFetch(path, body, { signal, token, onEvent }) {
  const response = await fetch(apiBase + path, {
    method: 'POST', signal,
    headers: { Accept: 'text/event-stream', 'Content-Type': 'application/json', ...(token ? { Authorization: token } : {}) },
    body: JSON.stringify(body),
  })
  if (!response.ok || !response.body) throw new Error(await response.text().catch(() => '') || `AI 服务请求失败（${response.status}）`)
  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  while (true) {
    const { done, value } = await reader.read()
    buffer += decoder.decode(value || new Uint8Array(), { stream: !done })
    const blocks = buffer.split(/\r?\n\r?\n/)
    buffer = blocks.pop() || ''
    blocks.forEach((block) => {
      let event = 'message'; const lines = []
      block.split(/\r?\n/).forEach((line) => {
        if (line.startsWith('event:')) event = line.slice(6).trim()
        if (line.startsWith('data:')) lines.push(line.slice(5).trimStart())
      })
      if (!lines.length) return
      let data = lines.join('\n')
      try { data = JSON.parse(data) } catch (_) { /* diagnostic text */ }
      onEvent(event, data)
    })
    if (done) break
  }
}

const chat = reactive({
  open: false,
  token: localStorage.getItem('blog_token') || '',
  sessionId: crypto.randomUUID(),
  messages: [], streaming: false, error: '', lastQuestion: '', controller: null,
})

function message(role, content) {
  return { id: crypto.randomUUID(), role, content, citations: [], tools: [] }
}

async function sendChat(question) {
  question = question.trim()
  if (!question || chat.streaming) return
  if (!chat.token) { chat.error = '请先登录，再使用 AI 助手。'; return }
  chat.error = ''; chat.lastQuestion = question
  chat.messages.push(message('user', question))
  const answer = message('assistant', '')
  chat.messages.push(answer); chat.streaming = true; chat.controller = new AbortController()
  try {
    await sseFetch('/api/ai/assistant/stream', { question, session_id: chat.sessionId }, {
      signal: chat.controller.signal, token: chat.token,
      onEvent(event, raw) {
        const payload = raw && typeof raw === 'object' ? raw : { content: String(raw ?? '') }
        if (event === 'delta') answer.content += payload.content || ''
        if (event === 'citation') answer.citations.push(payload.data || payload)
        if (event === 'tool') answer.tools.push(payload.data || payload)
        if (event === 'route') {
          const route = payload.data || payload
          if (route.intent) answer.tools.push({ name: `路由：${route.intent}`, status: '已分类' })
        }
        if (event === 'error') chat.error = payload.error?.message || payload.message || 'AI 服务暂时不可用'
      },
    })
  } catch (error) {
    if (error.name !== 'AbortError') chat.error = error.message
  } finally {
    chat.streaming = false; chat.controller = null
    if (!answer.content && !answer.citations.length && !answer.tools.length) chat.messages = chat.messages.filter((item) => item.id !== answer.id)
  }
}

const StreamMarkdown = {
  props: ['content', 'streaming'],
  setup(props) {
    const html = computed(() => DOMPurify.sanitize(marked.parse(props.content || '', { breaks: true, gfm: true })))
    return { html }
  },
  template: '<div class="markdown-body" :class="{ streaming }" v-html="html"></div>',
}

const CitationList = {
  props: ['citations'],
  template: `
    <section v-if="citations.length" class="citation-section">
      <p class="message-label">▣ 参考来源</p>
      <div class="citation-grid">
        <a v-for="item in citations" :key="item.index + '-' + item.chunk_id" class="citation-card" :href="item.url || '#'">
          <span class="citation-index">{{ item.index }}</span>
          <span><strong>{{ item.title || ('文章 #' + (item.article_id || item.index)) }}</strong><small>{{ item.content_excerpt || '查看引用文章' }}</small></span>
          <b>↗</b>
        </a>
      </div>
    </section>`,
}

const AIChatPanel = {
  components: { StreamMarkdown, CitationList },
  setup() {
    const input = ref(''); const scrollArea = ref(null)
    const canChat = computed(() => Boolean(chat.token))
    const submit = () => { const value = input.value; input.value = ''; sendChat(value) }
    const keydown = (event) => { if (event.key === 'Enter' && !event.shiftKey) { event.preventDefault(); submit() } }
    const stop = () => { chat.controller?.abort(); chat.streaming = false }
    const clear = () => { stop(); chat.messages = []; chat.error = ''; chat.sessionId = crypto.randomUUID() }
    const retry = () => sendChat(chat.lastQuestion)
    watch(() => chat.messages.map((item) => item.content).join('|'), async () => { await nextTick(); scrollArea.value?.scrollTo({ top: scrollArea.value.scrollHeight, behavior: 'smooth' }) })
    return { chat, input, canChat, submit, keydown, stop, clear, retry, sendChat, scrollArea }
  },
  template: `
    <button v-if="!chat.open" class="ai-fab" @click="chat.open = true"><b>✦</b><span>问问墨屿</span></button>
    <aside v-else class="chat-panel">
      <header class="chat-header">
        <div class="assistant-avatar">✦</div><div><strong>墨屿 AI 助手</strong><span><i :class="canChat ? 'online' : ''"></i>{{ canChat ? '已连接博客知识库' : '登录后可用' }}</span></div>
        <button title="清空会话" @click="clear">⌫</button><button title="关闭" @click="chat.open = false">✕</button>
      </header>
      <div ref="scrollArea" class="chat-scroll">
        <section v-if="!chat.messages.length" class="chat-welcome">
          <div class="welcome-mark">✦</div><h2>从博客里，找到答案</h2><p>我可以检索文章、追溯引用，也能告诉你最近最受欢迎的内容。</p>
          <div class="suggestion-list"><button @click="sendChat('最近最热门的文章是什么？')">最近最热门的文章是什么？</button><button @click="sendChat('帮我找一篇关于 Go 的文章')">帮我找一篇关于 Go 的文章</button><button @click="sendChat('最近的读者评论在讨论什么？')">最近评论在讨论什么？</button></div>
        </section>
        <article v-for="item in chat.messages" :key="item.id" class="chat-message" :class="item.role">
          <div v-if="item.role === 'assistant'" class="message-avatar">✦</div>
          <div class="message-content"><StreamMarkdown v-if="item.role === 'assistant'" :content="item.content" :streaming="chat.streaming && item === chat.messages.at(-1)"/><p v-else>{{ item.content }}</p>
            <div v-if="item.tools.length" class="tool-events"><div v-for="(tool, i) in item.tools" :key="i" class="tool-event">⌘ <span>{{ tool.name || '博客工具' }}</span><small>{{ tool.status || '已完成' }}</small></div></div>
            <CitationList :citations="item.citations"/>
          </div>
        </article>
        <div v-if="chat.streaming && !chat.messages.at(-1)?.content" class="thinking-row"><span class="spin">◌</span> 正在查阅博客…</div>
        <div v-if="chat.error" class="chat-error"><span>{{ chat.error }}</span><button @click="retry">↻ 重试</button></div>
      </div>
      <footer class="chat-composer"><textarea v-model="input" rows="1" :placeholder="canChat ? '问一篇文章，或聊聊博客里的观点…' : '请先在页面顶部登录'" :disabled="!canChat || chat.streaming" @keydown="keydown"></textarea><button v-if="chat.streaming" class="stop-button" @click="stop">■</button><button v-else class="send-button" :disabled="!input.trim() || !canChat" @click="submit">➤</button><small>Enter 发送 · Shift + Enter 换行</small></footer>
    </aside>`,
}

const AIWritingPanel = {
  components: { StreamMarkdown },
  props: ['open'],
  emits: ['close'],
  setup() {
    const topic = ref(''); const requirements = ref(''); const content = ref(''); const draftID = ref(null)
    const generating = ref(false); const saving = ref(false); const editing = ref(false); const confirmed = ref(false)
    const error = ref(''); const success = ref(''); let controller = null
    const title = computed(() => content.value.match(/^#\s+(.+)$/m)?.[1]?.trim() || topic.value.trim())
    async function generate() {
      if (!topic.value.trim() || !chat.token) return
      content.value = ''; error.value = ''; success.value = ''; generating.value = true; controller = new AbortController()
      try {
        await sseFetch('/api/ai/writing/stream', { topic: topic.value, requirements: requirements.value }, {
          signal: controller.signal, token: chat.token,
          onEvent(event, raw) {
            const payload = typeof raw === 'object' ? raw : { content: String(raw) }
            if (event === 'delta') content.value += payload.content || ''
            if (event === 'error') error.value = payload.error?.message || '写作服务暂时不可用'
          },
        })
      } catch (reason) { if (reason.name !== 'AbortError') error.value = reason.message }
      finally { generating.value = false; controller = null }
    }
    function stop() { controller?.abort(); generating.value = false }
    async function save() {
      if (!confirmed.value || !content.value.trim()) return
      saving.value = true; error.value = ''; success.value = ''
      try {
        const response = await requestJSON('/api/ai/drafts/save', {
          method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: chat.token },
          body: JSON.stringify({ article_id: draftID.value || undefined, title: title.value, content: content.value, confirmed: true }),
        })
        draftID.value = response.data.id; success.value = `草稿已保存（版本 ${response.data.version}），尚未发布。`; confirmed.value = false
      } catch (reason) { error.value = reason.message }
      finally { saving.value = false }
    }
    return { chat, topic, requirements, content, draftID, generating, saving, editing, confirmed, error, success, title, generate, stop, save }
  },
  template: `
    <div v-if="open" class="writing-backdrop" @click.self="$emit('close')"><section class="writing-panel">
      <header class="writing-header"><div><span class="kicker">AI WRITING STUDIO</span><h2>写一篇新文章</h2></div><button @click="$emit('close')">✕</button></header>
      <div class="writing-body"><aside class="writing-brief"><label>文章主题<textarea v-model="topic" rows="3" placeholder="例如：如何设计可靠的 RAG 系统"></textarea></label><label>补充要求<textarea v-model="requirements" rows="5" placeholder="目标读者、语气、篇幅、必须覆盖的观点…"></textarea></label><button v-if="!generating" class="generate-button" :disabled="!topic.trim() || !chat.token" @click="generate">✦ 生成大纲与文章</button><button v-else class="generate-button stop" @click="stop">■ 停止生成</button><p class="writing-note">AI 会先检索站内主题文章，再生成大纲和 Markdown。生成内容不会自动保存或发布。</p></aside>
      <main class="writing-editor"><div class="editor-toolbar"><span>{{ draftID ? ('草稿 #' + draftID) : '未保存草稿' }}</span><button :class="{active:!editing}" @click="editing=false">预览</button><button :class="{active:editing}" @click="editing=true">✎ 编辑</button></div><textarea v-if="editing" v-model="content" class="markdown-editor" placeholder="生成内容会出现在这里，也可以手动编辑。"></textarea><div v-else class="writing-preview"><StreamMarkdown v-if="content" :content="content" :streaming="generating"/><div v-else class="empty-writing"><b>✦</b><p>填写主题，让 AI 帮你完成第一版草稿。</p></div></div></main></div>
      <footer class="writing-footer"><p v-if="error" class="writing-error">{{error}}</p><p v-if="success" class="writing-success">✓ {{success}}</p><label class="confirm-draft"><input v-model="confirmed" type="checkbox"> 我确认将当前内容保存为草稿（不会发布）</label><button class="save-draft-button" :disabled="!confirmed || !content.trim() || saving || generating" @click="save">▣ {{ saving ? '保存中…' : draftID ? '更新草稿' : '保存草稿' }}</button></footer>
    </section></div>`,
}

const DraftManager = {
  props: ['open'],
  emits: ['close', 'published'],
  setup(props, { emit }) {
    const drafts = ref([]); const loading = ref(false); const publishingID = ref(null)
    const error = ref(''); const success = ref('')
    async function load() {
      if (!chat.token) return
      loading.value = true; error.value = ''; success.value = ''
      try { drafts.value = (await requestJSON('/api/article/drafts', { headers: { Authorization: chat.token } })).list || [] }
      catch (reason) { error.value = reason.message }
      finally { loading.value = false }
    }
    async function publish(draft) {
      if (!window.confirm(`确定发布《${draft.title || '未命名草稿'}》吗？发布后将进入知识库索引。`)) return
      publishingID.value = draft.ID; error.value = ''; success.value = ''
      try {
        const result = await requestJSON('/api/article/publish', {
          method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: chat.token },
          body: JSON.stringify({ id: draft.ID }),
        })
        drafts.value = drafts.value.filter((item) => item.ID !== draft.ID)
        success.value = result.index_status === 'failed'
          ? `文章已发布，但 AI 索引失败：${result.index_error || '可稍后重新索引'}`
          : '文章已发布并提交索引。'
        emit('published')
      } catch (reason) { error.value = reason.message }
      finally { publishingID.value = null }
    }
    const draftExcerpt = (value = '') => value.replace(/[#>*_`]/g, '').slice(0, 130) || '暂无正文'
    watch(() => props.open, (open) => { if (open) load() })
    return { drafts, loading, publishingID, error, success, load, publish, draftExcerpt }
  },
  template: `
    <div v-if="open" class="draft-backdrop" @click.self="$emit('close')"><section class="draft-panel">
      <header><div><span class="kicker">DRAFT WORKSPACE</span><h2>我的草稿</h2></div><button title="刷新" @click="load">↻</button><button title="关闭" @click="$emit('close')">✕</button></header>
      <p v-if="error" class="writing-error draft-notice">{{ error }}</p><p v-if="success" class="writing-success draft-notice">{{ success }}</p>
      <div v-if="loading" class="draft-state"><span class="spin">◌</span> 正在读取草稿…</div>
      <div v-else-if="!drafts.length" class="draft-state"><b>▤</b><p>还没有草稿。可以先用 AI 写作生成一篇。</p></div>
      <div v-else class="draft-list"><article v-for="draft in drafts" :key="draft.ID" class="draft-card"><div><span>草稿 #{{ draft.ID }} · 版本 {{ draft.version || 1 }}</span><h3>{{ draft.title || '未命名草稿' }}</h3><p>{{ draftExcerpt(draft.content) }}</p></div><button :disabled="publishingID === draft.ID" @click="publish(draft)">➤ {{ publishingID === draft.ID ? '发布中…' : '确认发布' }}</button></article></div>
    </section></div>`,
}

const App = {
  components: { AIChatPanel, AIWritingPanel, DraftManager, StreamMarkdown },
  setup() {
    const articles = ref([]); const loading = ref(true); const notice = ref(''); const selected = ref(null)
    const loginOpen = ref(false); const writingOpen = ref(false); const draftsOpen = ref(false); const analyticsOpen = ref(false); const username = ref(''); const password = ref(''); const loginError = ref(''); const loggingIn = ref(false)
    const insightContent = ref(''); const insightLoading = ref(false); const insightError = ref('')
    const analyticsContent = ref(''); const analyticsLoading = ref(false); const analyticsQuestion = ref('概括当前博客的热门文章、热门标签和读者互动情况')
    const comments = ref([]); const commentsLoading = ref(false); const commentContent = ref(''); const commentSending = ref(false); const commentError = ref('')
    let insightController = null
    async function load() {
      loading.value = true; notice.value = ''
      try { articles.value = (await requestJSON('/api/article/list?page=1&pageSize=12')).list || [] }
      catch (error) { articles.value = demoArticles; notice.value = `博客服务暂未连接，当前展示预览内容。${error.message}` }
      finally { loading.value = false }
    }
    async function selectArticle(item) {
      selected.value = item
      try { selected.value = (await requestJSON(`/api/article/${item.ID}?id=${item.ID}`)).article }
      catch (_) { /* offline preview */ }
      await loadComments(selected.value.ID)
      window.scrollTo({ top: 0, behavior: 'smooth' })
    }
    async function loadComments(articleID = selected.value?.ID) {
      if (!articleID) return
      commentsLoading.value = true; commentError.value = ''
      try { comments.value = (await requestJSON(`/api/comment/list?article_id=${articleID}`)).comments || [] }
      catch (reason) { commentError.value = reason.message }
      finally { commentsLoading.value = false }
    }
    async function submitComment() {
      const content = commentContent.value.trim()
      if (!content || !selected.value || commentSending.value) return
      if (!chat.token) { loginOpen.value = true; return }
      commentSending.value = true; commentError.value = ''
      try {
        await requestJSON('/api/comment/create', { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: chat.token }, body: JSON.stringify({ article_id: selected.value.ID, content }) })
        commentContent.value = ''; await loadComments(selected.value.ID)
      } catch (reason) { commentError.value = reason.message }
      finally { commentSending.value = false }
    }
    async function submitLogin() {
      loggingIn.value = true; loginError.value = ''
      try {
        const result = await requestJSON('/api/login', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ username: username.value, password: password.value }) })
        if (result.state !== 'success' || !result.token) throw new Error(result.msg || '登录失败')
        chat.token = result.token; localStorage.setItem('blog_token', result.token); loginOpen.value = false
      } catch (error) { loginError.value = error.message }
      finally { loggingIn.value = false }
    }
	async function logoutUser() {
	  try { await requestJSON('/api/logout', { method: 'POST', headers: { Authorization: chat.token } }) } catch (_) { /* clear local session even if server is unavailable */ }
	  chat.token = ''; localStorage.removeItem('blog_token'); chat.open = false
	  writingOpen.value = false; draftsOpen.value = false; analyticsOpen.value = false
	}
    async function runInsight(kind) {
      if (!chat.token || insightLoading.value || !selected.value) return
      insightContent.value = ''; insightError.value = ''; insightLoading.value = true; insightController = new AbortController()
      const path = kind === 'summary' ? '/api/ai/articles/summary/stream' : '/api/ai/comments/analyze/stream'
      try {
        await sseFetch(path, { article_id: selected.value.ID, limit: 50 }, { signal: insightController.signal, token: chat.token, onEvent(event, raw) { const payload = typeof raw === 'object' ? raw : { content: String(raw) }; if (event === 'delta') insightContent.value += payload.content || ''; if (event === 'error') insightError.value = payload.error?.message || '分析失败' } })
      } catch (reason) { if (reason.name !== 'AbortError') insightError.value = reason.message }
      finally { insightLoading.value = false; insightController = null }
    }
    async function runAnalytics() {
      if (!chat.token || analyticsLoading.value) return
      analyticsContent.value = ''; analyticsLoading.value = true; insightController = new AbortController()
      try {
        await sseFetch('/api/ai/analytics/stream', { question: analyticsQuestion.value }, { signal: insightController.signal, token: chat.token, onEvent(event, raw) { const payload = typeof raw === 'object' ? raw : { content: String(raw) }; if (event === 'delta') analyticsContent.value += payload.content || '' } })
      } catch (reason) { insightError.value = reason.message }
      finally { analyticsLoading.value = false; insightController = null }
    }
    const excerpt = (value = '') => value.replace(/[#>*_`\[\]]/g, '').slice(0, 116) || '这篇文章正在等待你阅读。'
    const date = (value) => value ? new Intl.DateTimeFormat('zh-CN', { month: 'long', day: 'numeric' }).format(new Date(value)) : '最近更新'
    onMounted(load)
    return { articles, loading, notice, selected, loginOpen, writingOpen, draftsOpen, analyticsOpen, username, password, loginError, loggingIn, insightContent, insightLoading, insightError, analyticsContent, analyticsLoading, analyticsQuestion, comments, commentsLoading, commentContent, commentSending, commentError, load, selectArticle, loadComments, submitComment, submitLogin, runInsight, runAnalytics, logoutUser, excerpt, date, chat }
  },
  template: `
    <div class="site-shell">
      <header class="site-header"><button class="brand" @click="selected = null"><span class="brand-mark">墨</span><span><strong>墨屿</strong><small>INK ISLE</small></span></button><nav><button class="active" @click="selected = null">文章</button><a href="#latest">随笔</a><a href="#about">关于</a></nav><div class="header-actions"><button class="round-button">⌕</button><button v-if="!chat.token" class="login-button" @click="loginOpen = true">⇥ 登录</button><details v-if="chat.token" class="tool-menu"><summary>▣ 创作中心</summary><div><button @click="$event.currentTarget.closest('details').open=false;writingOpen=true">✦ AI 写作</button><button @click="$event.currentTarget.closest('details').open=false;draftsOpen=true">▤ 草稿箱 / 发布</button><button @click="$event.currentTarget.closest('details').open=false;analyticsOpen=true;runAnalytics()">▥ 数据洞察</button></div></details><button v-if="chat.token" class="login-button connected" @click="chat.open = true">✦ AI 已连接</button><button v-if="chat.token" class="logout-button" @click="logoutUser">⇥ 退出</button></div></header>
      <main v-if="!selected">
        <section class="hero-section"><div class="hero-copy"><span class="kicker">THOUGHTS · CODE · INTELLIGENCE</span><h1>在技术的深处，<br><em>寻找清晰。</em></h1><p>关于工程、人工智能与创造的长期笔记。写下那些值得反复思考，也值得被认真讨论的事。</p><a href="#latest" class="text-link">开始阅读 →</a></div><div class="hero-orbit"><div class="orbit orbit-one"></div><div class="orbit orbit-two"></div><div class="orb-core">墨</div><span class="orbit-note note-one">BUILD</span><span class="orbit-note note-two">THINK</span></div></section>
        <section id="latest" class="latest-section"><div class="section-title-row"><div><span class="kicker">LATEST WRITING</span><h2>最近文章</h2></div><button class="quiet-button" @click="load">↻ 刷新</button></div><p v-if="notice" class="preview-notice">{{ notice }}</p><div v-if="loading" class="article-skeletons"><div v-for="n in 3" :key="n" class="article-card skeleton"></div></div><div v-else class="article-grid"><article v-for="(item, i) in articles" :key="item.ID" class="article-card" @click="selectArticle(item)"><div class="article-number">0{{ i + 1 }}</div><div class="article-meta"><span>◷ {{ date(item.published_at || item.CreatedAt) }}</span><span>◉ {{ item.view_count || 0 }}</span></div><h3>{{ item.title }}</h3><p>{{ excerpt(item.content) }}</p><button>阅读全文 →</button></article></div></section>
      </main>
      <main v-else class="article-view"><button class="back-button" @click="selected = null">← 返回文章列表</button><article><header class="article-header"><span class="kicker">ARTICLE · {{ selected.ID }}</span><h1>{{ selected.title }}</h1><div class="article-detail-meta"><span>▣ {{ selected.published_at?.slice(0,10) || '最近发布' }}</span><span>◉ {{ selected.view_count || 0 }} 次阅读</span></div></header><div class="article-divider"><span></span></div><StreamMarkdown :content="selected.content"/></article><section class="article-insights"><div class="insight-actions"><div><span class="kicker">AI READING NOTES</span><h2>读懂这篇文章</h2></div><button :disabled="!chat.token || insightLoading" @click="runInsight('summary')">✦ 一键总结</button><button :disabled="!chat.token || insightLoading" @click="runInsight('comments')">◫ 评论分析</button></div><p v-if="!chat.token" class="insight-hint">登录后可生成文章总结和评论洞察。</p><div v-if="insightContent || insightLoading || insightError" class="insight-result"><StreamMarkdown :content="insightContent" :streaming="insightLoading"/><p v-if="insightError" class="writing-error">{{insightError}}</p></div></section><section class="comments-section"><div class="comments-heading"><div><span class="kicker">READER DISCUSSION</span><h2>评论 · {{ comments.length }}</h2></div><button class="quiet-button" @click="loadComments()">↻ 刷新</button></div><form class="comment-form" @submit.prevent="submitComment"><textarea v-model="commentContent" rows="3" :placeholder="chat.token ? '写下你的想法…' : '登录后参与评论'" :disabled="!chat.token || commentSending"></textarea><div><span v-if="commentError" class="writing-error">{{ commentError }}</span><button v-if="!chat.token" type="button" @click="loginOpen=true">登录后评论</button><button v-else :disabled="!commentContent.trim() || commentSending">{{ commentSending ? '发送中…' : '发表评论' }}</button></div></form><div v-if="commentsLoading" class="comment-state">正在读取评论…</div><div v-else-if="!comments.length" class="comment-state">还没有评论，来写下第一条吧。</div><div v-else class="comment-list"><article v-for="item in comments" :key="item.id" class="comment-card"><div class="comment-avatar">{{ (item.user?.name || '读').slice(0,1) }}</div><div><header><strong>{{ item.user?.name || '读者' }}</strong><time>{{ item.created_at ? new Date(item.created_at).toLocaleString('zh-CN') : '' }}</time></header><p>{{ item.content }}</p></div></article></div></section></main>
      <footer id="about" class="site-footer"><div><span class="brand-mark small">墨</span><strong>在持续写作中，保持诚实与清醒。</strong></div><p>© 2026 墨屿 · Built with Go, Vue & curiosity.</p></footer>
      <AIChatPanel/>
      <AIWritingPanel :open="writingOpen" @close="writingOpen=false"/>
      <DraftManager :open="draftsOpen" @close="draftsOpen=false" @published="load"/>
      <div v-if="analyticsOpen" class="analytics-backdrop" @click.self="analyticsOpen=false"><section class="analytics-panel"><header><div class="analytics-icon">▥</div><div><span class="kicker">BLOG INTELLIGENCE</span><h2>博客数据洞察</h2></div><button @click="analyticsOpen=false">✕</button></header><div class="analytics-query"><input v-model="analyticsQuestion" placeholder="用自然语言询问博客统计…"><button @click="runAnalytics">✦ 分析</button></div><div class="analytics-report"><StreamMarkdown v-if="analyticsContent" :content="analyticsContent" :streaming="analyticsLoading"/><div v-else class="empty-writing">正在读取文章、标签与评论统计…</div></div></section></div>
      <div v-if="loginOpen" class="modal-backdrop" @click.self="loginOpen = false"><form class="login-modal" @submit.prevent="submitLogin"><button type="button" class="modal-close" @click="loginOpen = false">✕</button><div class="login-icon">墨</div><span class="kicker">WELCOME BACK</span><h2>登录墨屿</h2><p>首次登录会自动创建你的读者账号。</p><label>用户名<input v-model="username" required></label><label>密码<input v-model="password" type="password" required></label><p v-if="loginError" class="form-error">{{ loginError }}</p><button class="modal-submit" :disabled="loggingIn">{{ loggingIn ? '正在登录…' : '登录并继续' }}</button></form></div>
    </div>`,
}

createApp(App).mount('#app')
