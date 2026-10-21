<script setup lang="ts">
import { BookOpen, ExternalLink } from 'lucide-vue-next'
import type { Citation } from '../api/ai'

defineProps<{ citations: Citation[] }>()
</script>

<template>
  <section v-if="citations.length" class="citation-section">
    <p class="message-label"><BookOpen :size="14" /> 参考来源</p>
    <div class="citation-grid">
      <a
        v-for="citation in citations"
        :key="`${citation.index}-${citation.chunk_id}`"
        class="citation-card"
        :href="citation.url || (citation.article_id ? `#article-${citation.article_id}` : '#')"
      >
        <span class="citation-index">{{ citation.index }}</span>
        <span>
          <strong>{{ citation.title || `文章 #${citation.article_id || citation.index}` }}</strong>
          <small>{{ citation.content_excerpt || '查看引用文章' }}</small>
        </span>
        <ExternalLink :size="14" />
      </a>
    </div>
  </section>
</template>
