<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ArrowRight, Clock3, Eye, RefreshCw } from 'lucide-vue-next'
import { listArticles, type Article } from '../api/blog'

const emit = defineEmits<{ select: [article: Article] }>()
const articles = ref<Article[]>([])
const loading = ref(true)
const error = ref('')

const demoArticles: Article[] = [
  { ID: 101, title: '用 Go 构建一个会记忆的 AI Agent', content: '从流式协议、会话记忆到工具调用，一步步搭建可靠的 Agent 服务。', view_count: 1284, published_at: '2026-07-18T10:00:00Z' },
  { ID: 102, title: 'RAG 不只是向量搜索：混合检索实践', content: 'BM25、向量召回、RRF 与 Rerank 如何一起工作。', view_count: 968, published_at: '2026-07-15T10:00:00Z' },
  { ID: 103, title: '把复杂系统，写成简单的故事', content: '技术写作的结构、节奏，以及如何让读者真正理解一个系统。', view_count: 726, published_at: '2026-07-11T10:00:00Z' },
]

async function load() {
  loading.value = true
  error.value = ''
  try {
    const result = await listArticles()
    articles.value = result.list || []
  } catch (requestError) {
    articles.value = demoArticles
    error.value = `博客服务暂未连接，当前展示预览内容。${(requestError as Error).message}`
  } finally {
    loading.value = false
  }
}

onMounted(load)

function excerpt(content = '') {
  return content.replace(/[#>*_`\[\]]/g, '').slice(0, 116) || '这篇文章正在等待你阅读。'
}

function formatDate(value?: string) {
  if (!value) return '最近更新'
  return new Intl.DateTimeFormat('zh-CN', { month: 'long', day: 'numeric' }).format(new Date(value))
}
</script>

<template>
  <main>
    <section class="hero-section">
      <div class="hero-copy">
        <span class="kicker">THOUGHTS · CODE · INTELLIGENCE</span>
        <h1>在技术的深处，<br><em>寻找清晰。</em></h1>
        <p>关于工程、人工智能与创造的长期笔记。写下那些值得反复思考，也值得被认真讨论的事。</p>
        <a href="#latest" class="text-link">开始阅读 <ArrowRight :size="17" /></a>
      </div>
      <div class="hero-orbit" aria-hidden="true">
        <div class="orbit orbit-one" />
        <div class="orbit orbit-two" />
        <div class="orb-core">墨</div>
        <span class="orbit-note note-one">BUILD</span>
        <span class="orbit-note note-two">THINK</span>
      </div>
    </section>

    <section id="latest" class="latest-section">
      <div class="section-title-row">
        <div>
          <span class="kicker">LATEST WRITING</span>
          <h2>最近文章</h2>
        </div>
        <button class="quiet-button" title="刷新" @click="load"><RefreshCw :size="16" /> 刷新</button>
      </div>

      <p v-if="error" class="preview-notice">{{ error }}</p>
      <div v-if="loading" class="article-skeletons">
        <div v-for="item in 3" :key="item" class="article-card skeleton" />
      </div>
      <div v-else class="article-grid">
        <article v-for="(article, index) in articles" :key="article.ID" class="article-card" @click="emit('select', article)">
          <div class="article-number">0{{ index + 1 }}</div>
          <div class="article-meta">
            <span><Clock3 :size="13" /> {{ formatDate(article.published_at || article.CreatedAt) }}</span>
            <span><Eye :size="13" /> {{ article.view_count || 0 }}</span>
          </div>
          <h3>{{ article.title }}</h3>
          <p>{{ excerpt(article.content) }}</p>
          <button>阅读全文 <ArrowRight :size="15" /></button>
        </article>
      </div>
    </section>
  </main>
</template>
