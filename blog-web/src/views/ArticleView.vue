<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ArrowLeft, CalendarDays, Eye } from 'lucide-vue-next'
import { getArticle, type Article } from '../api/blog'
import StreamMarkdown from '../components/StreamMarkdown.vue'
import ArticleInsights from '../components/ArticleInsights.vue'
import CommentsSection from '../components/CommentsSection.vue'

const props = defineProps<{ article: Article }>()
defineEmits<{ back: [] }>()
const detail = ref(props.article)

onMounted(async () => {
  try {
    detail.value = (await getArticle(props.article.ID)).article
  } catch {
    // Keep the list payload as an offline preview.
  }
})
</script>

<template>
  <main class="article-view">
    <button class="back-button" @click="$emit('back')"><ArrowLeft :size="17" /> 返回文章列表</button>
    <article>
      <header class="article-header">
        <span class="kicker">ARTICLE · {{ detail.ID }}</span>
        <h1>{{ detail.title }}</h1>
        <div class="article-detail-meta">
          <span><CalendarDays :size="15" /> {{ detail.published_at?.slice(0, 10) || '最近发布' }}</span>
          <span><Eye :size="15" /> {{ detail.view_count || 0 }} 次阅读</span>
        </div>
      </header>
      <div class="article-divider"><span /></div>
      <StreamMarkdown :content="detail.content" />
    </article>
    <ArticleInsights :article-id="detail.ID" />
    <CommentsSection :article-id="detail.ID" />
  </main>
</template>
