<script setup lang="ts">
import { ref, watch } from 'vue'
import { BarChart3, CircleStop, Sparkles, X } from 'lucide-vue-next'
import { streamInsight } from '../api/insights'
import { useChatStore } from '../stores/chat'
import StreamMarkdown from './StreamMarkdown.vue'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: [] }>()
const { state } = useChatStore()
const question = ref('概括当前博客的热门文章、热门标签和读者互动情况')
const content = ref('')
const loading = ref(false)
const error = ref('')
let controller: AbortController | null = null

async function analyze() {
  if (!state.token || loading.value) return
  content.value = ''
  error.value = ''
  loading.value = true
  controller = new AbortController()
  try {
    await streamInsight('analytics', { question: question.value }, state.token, controller.signal, (delta) => { content.value += delta })
  } catch (reason) {
    if ((reason as Error).name !== 'AbortError') error.value = (reason as Error).message
  } finally { loading.value = false; controller = null }
}

function stop() { controller?.abort(); loading.value = false }
watch(() => props.open, (open) => { if (open && !content.value) analyze() })
</script>

<template>
  <div v-if="open" class="analytics-backdrop" @click.self="emit('close')">
    <section class="analytics-panel">
      <header><div class="analytics-icon"><BarChart3 :size="21" /></div><div><span class="kicker">BLOG INTELLIGENCE</span><h2>博客数据洞察</h2></div><button @click="emit('close')"><X :size="20" /></button></header>
      <div class="analytics-query"><input v-model="question" placeholder="用自然语言询问博客统计…" /><button v-if="!loading" @click="analyze"><Sparkles :size="16" /> 分析</button><button v-else class="stop" @click="stop"><CircleStop :size="16" /> 停止</button></div>
      <div class="analytics-report"><StreamMarkdown v-if="content" :content="content" :streaming="loading" /><div v-else-if="loading" class="empty-writing">正在读取文章、标签与评论统计…</div><p v-if="error" class="writing-error">{{ error }}</p></div>
    </section>
  </div>
</template>
