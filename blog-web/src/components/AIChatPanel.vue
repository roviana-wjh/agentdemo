<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import {
  Bot, CircleStop, LoaderCircle, MessageCircle, RotateCcw,
  Send, Sparkles, Trash2, Wrench, X,
} from 'lucide-vue-next'
import { useChatStore } from '../stores/chat'
import CitationList from './CitationList.vue'
import StreamMarkdown from './StreamMarkdown.vue'

const { state, canChat, send, stop, clear, retry } = useChatStore()
const input = ref('')
const scrollArea = ref<HTMLElement | null>(null)

async function submit() {
  const value = input.value
  input.value = ''
  await send(value)
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    submit()
  }
}

watch(() => state.messages.map((message) => message.content).join('|'), async () => {
  await nextTick()
  scrollArea.value?.scrollTo({ top: scrollArea.value.scrollHeight, behavior: 'smooth' })
})
</script>

<template>
  <button v-if="!state.open" class="ai-fab" title="打开 AI 助手" @click="state.open = true">
    <Sparkles :size="20" />
    <span>问问墨屿</span>
  </button>

  <aside v-else class="chat-panel" aria-label="AI 博客助手">
    <header class="chat-header">
      <div class="assistant-avatar"><Bot :size="20" /></div>
      <div>
        <strong>墨屿 AI 助手</strong>
        <span><i :class="canChat ? 'online' : ''" />{{ canChat ? '已连接博客知识库' : '登录后可用' }}</span>
      </div>
      <button title="清空会话" @click="clear"><Trash2 :size="17" /></button>
      <button title="关闭" @click="state.open = false"><X :size="19" /></button>
    </header>

    <div ref="scrollArea" class="chat-scroll">
      <section v-if="!state.messages.length" class="chat-welcome">
        <div class="welcome-mark"><MessageCircle :size="25" /></div>
        <h2>从博客里，找到答案</h2>
        <p>我可以检索文章、追溯引用，也能告诉你最近最受欢迎的内容。</p>
        <div class="suggestion-list">
          <button @click="send('最近最热门的文章是什么？')">最近最热门的文章是什么？</button>
          <button @click="send('帮我找一篇关于 Go 的文章')">帮我找一篇关于 Go 的文章</button>
          <button @click="send('最近的读者评论在讨论什么？')">最近评论在讨论什么？</button>
        </div>
      </section>

      <article v-for="message in state.messages" :key="message.id" class="chat-message" :class="message.role">
        <div v-if="message.role === 'assistant'" class="message-avatar"><Bot :size="15" /></div>
        <div class="message-content">
          <StreamMarkdown
            v-if="message.role === 'assistant'"
            :content="message.content"
            :streaming="state.streaming && message === state.messages.at(-1)"
          />
          <p v-else>{{ message.content }}</p>

          <div v-if="message.tools.length" class="tool-events">
            <div v-for="(tool, index) in message.tools" :key="index" class="tool-event">
              <Wrench :size="13" />
              <span>{{ tool.name || '博客工具' }}</span>
              <small>{{ tool.status || '已完成' }}</small>
            </div>
          </div>
          <CitationList :citations="message.citations" />
        </div>
      </article>

      <div v-if="state.streaming && !state.messages.at(-1)?.content" class="thinking-row">
        <LoaderCircle :size="16" class="spin" /> 正在查阅博客…
      </div>

      <div v-if="state.error" class="chat-error">
        <span>{{ state.error }}</span>
        <button @click="retry"><RotateCcw :size="14" /> 重试</button>
      </div>
    </div>

    <footer class="chat-composer">
      <textarea
        v-model="input"
        rows="1"
        :placeholder="canChat ? '问一篇文章，或聊聊博客里的观点…' : '请先在页面顶部登录'"
        :disabled="!canChat || state.streaming"
        @keydown="handleKeydown"
      />
      <button v-if="state.streaming" class="stop-button" title="停止生成" @click="stop">
        <CircleStop :size="19" />
      </button>
      <button v-else class="send-button" :disabled="!input.trim() || !canChat" title="发送" @click="submit">
        <Send :size="18" />
      </button>
      <small>Enter 发送 · Shift + Enter 换行</small>
    </footer>
  </aside>
</template>
