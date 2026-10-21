<script setup lang="ts">
import { computed, ref } from 'vue'
import { Check, CircleStop, FilePenLine, Save, Sparkles, X } from 'lucide-vue-next'
import { streamWriting, saveWritingDraft } from '../api/writing'
import { useChatStore } from '../stores/chat'
import StreamMarkdown from './StreamMarkdown.vue'

defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: [] }>()
const { state } = useChatStore()
const topic = ref('')
const requirements = ref('')
const content = ref('')
const draftID = ref<number>()
const generating = ref(false)
const saving = ref(false)
const editing = ref(false)
const confirmSave = ref(false)
const error = ref('')
const success = ref('')
let controller: AbortController | null = null

const title = computed(() => {
  const match = content.value.match(/^#\s+(.+)$/m)
  return match?.[1]?.trim() || topic.value.trim()
})

async function generate() {
  if (!topic.value.trim() || generating.value || !state.token) return
  content.value = ''
  success.value = ''
  error.value = ''
  generating.value = true
  controller = new AbortController()
  try {
    await streamWriting(topic.value, requirements.value, state.token, controller.signal, (delta) => {
      content.value += delta
    })
  } catch (reason) {
    if ((reason as Error).name !== 'AbortError') error.value = (reason as Error).message
  } finally {
    generating.value = false
    controller = null
  }
}

function stop() {
  controller?.abort()
  generating.value = false
}

async function save() {
  if (!confirmSave.value || !content.value.trim()) return
  saving.value = true
  error.value = ''
  success.value = ''
  try {
    const result = await saveWritingDraft({
      article_id: draftID.value,
      title: title.value,
      content: content.value,
      confirmed: true,
    }, state.token)
    draftID.value = result.id
    success.value = `草稿已保存（版本 ${result.version}），尚未发布。`
    confirmSave.value = false
  } catch (reason) {
    error.value = (reason as Error).message
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div v-if="open" class="writing-backdrop" @click.self="emit('close')">
    <section class="writing-panel">
      <header class="writing-header">
        <div><span class="kicker">AI WRITING STUDIO</span><h2>写一篇新文章</h2></div>
        <button title="关闭" @click="emit('close')"><X :size="20" /></button>
      </header>

      <div class="writing-body">
        <aside class="writing-brief">
          <label>文章主题<textarea v-model="topic" rows="3" placeholder="例如：如何设计可靠的 RAG 系统" /></label>
          <label>补充要求<textarea v-model="requirements" rows="5" placeholder="目标读者、语气、篇幅、必须覆盖的观点…" /></label>
          <button v-if="!generating" class="generate-button" :disabled="!topic.trim() || !state.token" @click="generate"><Sparkles :size="17" /> 生成大纲与文章</button>
          <button v-else class="generate-button stop" @click="stop"><CircleStop :size="17" /> 停止生成</button>
          <p class="writing-note">AI 会先检索站内主题文章，再生成大纲和 Markdown。生成内容不会自动保存或发布。</p>
        </aside>

        <main class="writing-editor">
          <div class="editor-toolbar">
            <span>{{ draftID ? `草稿 #${draftID}` : '未保存草稿' }}</span>
            <button :class="{ active: !editing }" @click="editing = false">预览</button>
            <button :class="{ active: editing }" @click="editing = true"><FilePenLine :size="14" /> 编辑</button>
          </div>
          <textarea v-if="editing" v-model="content" class="markdown-editor" placeholder="生成内容会出现在这里，也可以手动编辑。" />
          <div v-else class="writing-preview">
            <StreamMarkdown v-if="content" :content="content" :streaming="generating" />
            <div v-else class="empty-writing"><Sparkles :size="26" /><p>填写主题，让 AI 帮你完成第一版草稿。</p></div>
          </div>
        </main>
      </div>

      <footer class="writing-footer">
        <p v-if="error" class="writing-error">{{ error }}</p>
        <p v-if="success" class="writing-success"><Check :size="15" /> {{ success }}</p>
        <label class="confirm-draft"><input v-model="confirmSave" type="checkbox" /> 我确认将当前内容保存为草稿（不会发布）</label>
        <button class="save-draft-button" :disabled="!confirmSave || !content.trim() || saving || generating" @click="save"><Save :size="17" /> {{ saving ? '保存中…' : draftID ? '更新草稿' : '保存草稿' }}</button>
      </footer>
    </section>
  </div>
</template>
