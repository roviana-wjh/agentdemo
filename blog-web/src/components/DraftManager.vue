<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { FileText, LoaderCircle, RefreshCw, Send, ShieldCheck, X } from 'lucide-vue-next'
import {
  listDrafts,
  publishArticle,
  reviewArticle,
  type Article,
  type ArticleReviewResult,
  type ReviewStatus,
} from '../api/blog'
import { useChatStore } from '../stores/chat'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: []; published: [] }>()
const { state } = useChatStore()
const drafts = ref<Article[]>([])
const loading = ref(false)
const publishingID = ref<number>()
const reviewingID = ref<number>()
const error = ref('')
const success = ref('')
const reviewResults = reactive<Record<number, ArticleReviewResult>>({})
const hasDrafts = computed(() => drafts.value.length > 0)

const statusLabels: Record<ReviewStatus, string> = {
  not_reviewed: '未审核',
  approved: '审核通过',
  needs_revision: '需要修改',
  rejected: '审核拒绝',
  failed: '审核失败',
}

function reviewStatus(draft: Article): ReviewStatus {
  return reviewResults[draft.ID]?.status || draft.review_status || 'not_reviewed'
}

async function load() {
  if (!state.token) return
  loading.value = true
  error.value = ''
  try {
    drafts.value = (await listDrafts(state.token)).list || []
  } catch (reason) {
    error.value = (reason as Error).message
  } finally {
    loading.value = false
  }
}

async function review(draft: Article) {
  reviewingID.value = draft.ID
  error.value = ''
  success.value = ''
  try {
    const response = await reviewArticle(draft.ID, state.token)
    reviewResults[draft.ID] = response.data
    draft.review_status = response.data.status
    draft.reviewed_at = response.reviewed_at
    success.value = response.data.status === 'approved'
      ? `《${draft.title}》已通过 AI 审核，可以发布。`
      : `《${draft.title}》审核完成，请根据审核建议修改。`
  } catch (reason) {
    error.value = (reason as Error).message
  } finally {
    reviewingID.value = undefined
  }
}

async function publish(draft: Article) {
  if (reviewStatus(draft) !== 'approved') {
    error.value = '文章必须先通过 AI 审核才能发布。'
    return
  }
  const confirmed = window.confirm(`确定发布《${draft.title}》吗？发布后将进入知识库索引。`)
  if (!confirmed) return
  publishingID.value = draft.ID
  error.value = ''
  success.value = ''
  try {
    const result = await publishArticle(draft.ID, state.token)
    drafts.value = drafts.value.filter((item) => item.ID !== draft.ID)
    delete reviewResults[draft.ID]
    success.value = result.index_status === 'failed'
      ? `文章已发布，但 AI 索引失败：${result.index_error || '可稍后重新索引'}`
      : '文章已发布并提交知识库索引。'
    emit('published')
  } catch (reason) {
    error.value = (reason as Error).message
  } finally {
    publishingID.value = undefined
  }
}

watch(() => props.open, (open) => { if (open) load() })
</script>

<template>
  <div v-if="open" class="draft-backdrop" @click.self="emit('close')">
    <section class="draft-panel">
      <header>
        <div><span class="kicker">DRAFT WORKSPACE</span><h2>我的草稿</h2></div>
        <button title="刷新" @click="load"><RefreshCw :size="17" /></button>
        <button title="关闭" @click="emit('close')"><X :size="20" /></button>
      </header>
      <p v-if="error" class="writing-error draft-notice">{{ error }}</p>
      <p v-if="success" class="writing-success draft-notice">{{ success }}</p>
      <div v-if="loading" class="draft-state"><LoaderCircle class="spin" :size="20" /> 正在读取草稿…</div>
      <div v-else-if="!hasDrafts" class="draft-state"><FileText :size="28" /><p>还没有草稿，可以先用 AI 写作生成一篇。</p></div>
      <div v-else class="draft-list">
        <article v-for="draft in drafts" :key="draft.ID" class="draft-card">
          <div class="draft-copy">
            <div class="draft-meta">
              <span>草稿 #{{ draft.ID }} · 版本 {{ draft.version || 1 }}</span>
              <span class="review-badge" :data-status="reviewStatus(draft)">{{ statusLabels[reviewStatus(draft)] }}</span>
            </div>
            <h3>{{ draft.title || '未命名草稿' }}</h3>
            <p>{{ draft.content?.replace(/[#>*_`]/g, '').slice(0, 130) || '暂无正文' }}</p>

            <div v-if="reviewResults[draft.ID]" class="review-result">
              <strong>风险等级：{{ reviewResults[draft.ID].risk_level }} · 评分 {{ reviewResults[draft.ID].score }}</strong>
              <ul v-if="reviewResults[draft.ID].reasons.length">
                <li v-for="reason in reviewResults[draft.ID].reasons" :key="reason">{{ reason }}</li>
              </ul>
              <div v-if="reviewResults[draft.ID].suggestions.length" class="review-suggestions">
                <strong>修改建议</strong>
                <ul><li v-for="suggestion in reviewResults[draft.ID].suggestions" :key="suggestion">{{ suggestion }}</li></ul>
              </div>
            </div>
          </div>

          <div class="draft-actions">
            <button class="review-button" :disabled="reviewingID === draft.ID || publishingID === draft.ID" @click="review(draft)">
              <ShieldCheck :size="15" /> {{ reviewingID === draft.ID ? '审核中…' : 'AI 审核' }}
            </button>
            <button
              :disabled="reviewStatus(draft) !== 'approved' || publishingID === draft.ID || reviewingID === draft.ID"
              @click="publish(draft)"
            >
              <Send :size="15" /> {{ publishingID === draft.ID ? '发布中…' : '确认发布' }}
            </button>
          </div>
        </article>
      </div>
    </section>
  </div>
</template>
