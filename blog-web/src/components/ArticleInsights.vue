<script setup lang="ts">
import { ref } from 'vue'
import { BrainCircuit, CircleStop, MessageSquareText, RefreshCw } from 'lucide-vue-next'
import { streamInsight, type InsightKind } from '../api/insights'
import { useChatStore } from '../stores/chat'
import StreamMarkdown from './StreamMarkdown.vue'

const props = defineProps<{ articleId: number }>()
const { state } = useChatStore()
const active = ref<InsightKind | ''>('')
const content = ref('')
const loading = ref(false)
const error = ref('')
let controller: AbortController | null = null

async function run(kind: InsightKind) {
  if (!state.token || loading.value) return
  active.value = kind
  content.value = ''
  error.value = ''
  loading.value = true
  controller = new AbortController()
  try {
    await streamInsight(kind, { article_id: props.articleId, limit: 50 }, state.token, controller.signal, (delta) => {
      content.value += delta
    })
  } catch (reason) {
    if ((reason as Error).name !== 'AbortError') error.value = (reason as Error).message
  } finally {
    loading.value = false
    controller = null
  }
}

function stop() { controller?.abort(); loading.value = false }
</script>

<template>
  <section class="article-insights">
    <div class="insight-actions">
      <div><span class="kicker">AI READING NOTES</span><h2>读懂这篇文章</h2></div>
      <button :disabled="!state.token || loading" @click="run('summary')"><BrainCircuit :size="16" /> 一键总结</button>
      <button :disabled="!state.token || loading" @click="run('comments')"><MessageSquareText :size="16" /> 评论分析</button>
      <button v-if="loading" class="cancel-insight" @click="stop"><CircleStop :size="16" /> 停止</button>
    </div>
    <p v-if="!state.token" class="insight-hint">登录后可生成文章总结和评论洞察。</p>
    <div v-if="content || loading || error" class="insight-result">
      <StreamMarkdown :content="content" :streaming="loading" />
      <p v-if="error" class="writing-error">{{ error }} <button @click="run(active || 'summary')"><RefreshCw :size="13" /> 重试</button></p>
    </div>
  </section>
</template>
