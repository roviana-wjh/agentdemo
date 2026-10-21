<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { createComment, listComments, type Comment } from '../api/blog'
import { useChatStore } from '../stores/chat'

const props = defineProps<{ articleId: number }>()
const { state } = useChatStore()
const comments = ref<Comment[]>([])
const content = ref('')
const loading = ref(false)
const sending = ref(false)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try { comments.value = (await listComments(props.articleId)).comments || [] }
  catch (reason) { error.value = (reason as Error).message }
  finally { loading.value = false }
}

async function submit() {
  const value = content.value.trim()
  if (!value || !state.token || sending.value) return
  sending.value = true
  error.value = ''
  try { await createComment(props.articleId, value, state.token); content.value = ''; await load() }
  catch (reason) { error.value = (reason as Error).message }
  finally { sending.value = false }
}

onMounted(load)
</script>

<template>
  <section class="comments-section">
    <div class="comments-heading"><div><span class="kicker">READER DISCUSSION</span><h2>评论 · {{ comments.length }}</h2></div><button class="quiet-button" @click="load">↻ 刷新</button></div>
    <form class="comment-form" @submit.prevent="submit"><textarea v-model="content" rows="3" :placeholder="state.token ? '写下你的想法…' : '请先在页面顶部登录'" :disabled="!state.token || sending" /><div><span v-if="error" class="writing-error">{{ error }}</span><button :disabled="!state.token || !content.trim() || sending">{{ sending ? '发送中…' : state.token ? '发表评论' : '登录后评论' }}</button></div></form>
    <div v-if="loading" class="comment-state">正在读取评论…</div><div v-else-if="!comments.length" class="comment-state">还没有评论，来写下第一条吧。</div>
    <div v-else class="comment-list"><article v-for="item in comments" :key="item.id" class="comment-card"><div class="comment-avatar">{{ (item.user?.name || '读').slice(0, 1) }}</div><div><header><strong>{{ item.user?.name || '读者' }}</strong><time>{{ new Date(item.created_at).toLocaleString('zh-CN') }}</time></header><p>{{ item.content }}</p></div></article></div>
  </section>
</template>
