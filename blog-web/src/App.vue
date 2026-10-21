<script setup lang="ts">
import { ref } from 'vue'
import { BookOpen, LogIn, LogOut, Menu, Search, Sparkles, X } from 'lucide-vue-next'
import AIChatPanel from './components/AIChatPanel.vue'
import AIWritingPanel from './components/AIWritingPanel.vue'
import AnalyticsPanel from './components/AnalyticsPanel.vue'
import DraftManager from './components/DraftManager.vue'
import HomeView from './views/HomeView.vue'
import ArticleView from './views/ArticleView.vue'
import { login, logout, type Article } from './api/blog'
import { useChatStore } from './stores/chat'

const { state: chatState, setToken, clearToken } = useChatStore()
const selectedArticle = ref<Article | null>(null)
const loginOpen = ref(false)
const mobileOpen = ref(false)
const username = ref('')
const password = ref('')
const loginError = ref('')
const loggingIn = ref(false)
const writingOpen = ref(false)
const analyticsOpen = ref(false)
const draftsOpen = ref(false)

async function submitLogin() {
  loggingIn.value = true
  loginError.value = ''
  try {
    const result = await login(username.value, password.value)
    setToken(result.token!)
    loginOpen.value = false
    username.value = ''
    password.value = ''
  } catch (error) {
    loginError.value = (error as Error).message
  } finally {
    loggingIn.value = false
  }
}

async function submitLogout() {
  const token = chatState.token
  try { await logout(token) } finally {
    clearToken()
    chatState.open = false
    writingOpen.value = false
    analyticsOpen.value = false
    draftsOpen.value = false
  }
}
</script>

<template>
  <div class="site-shell">
    <header class="site-header">
      <button class="brand" @click="selectedArticle = null">
        <span class="brand-mark">墨</span>
        <span><strong>墨屿</strong><small>INK ISLE</small></span>
      </button>
      <nav :class="{ open: mobileOpen }">
        <button class="active" @click="selectedArticle = null; mobileOpen = false">文章</button>
        <a href="#latest" @click="mobileOpen = false">随笔</a>
        <a href="#about" @click="mobileOpen = false">关于</a>
      </nav>
      <div class="header-actions">
        <button class="round-button" title="搜索"><Search :size="18" /></button>
        <button v-if="!chatState.token" class="login-button" @click="loginOpen = true"><LogIn :size="16" /> 登录</button>
        <details v-else class="tool-menu">
          <summary><BookOpen :size="16" /> 创作中心</summary>
          <div>
            <button @click="writingOpen = true">AI 写作</button>
            <button @click="draftsOpen = true">草稿箱 / 发布</button>
            <button @click="analyticsOpen = true">数据洞察</button>
          </div>
        </details>
        <button v-if="chatState.token" class="login-button connected" @click="chatState.open = true"><Sparkles :size="16" /> AI 已连接</button>
        <button v-if="chatState.token" class="logout-button" title="退出登录" @click="submitLogout"><LogOut :size="16" /> 退出</button>
        <button class="mobile-menu" @click="mobileOpen = !mobileOpen"><Menu :size="20" /></button>
      </div>
    </header>

    <HomeView v-if="!selectedArticle" @select="selectedArticle = $event" />
    <ArticleView v-else :article="selectedArticle" @back="selectedArticle = null" />

    <footer id="about" class="site-footer">
      <div><span class="brand-mark small">墨</span><strong>在持续写作中，保持诚实与清醒。</strong></div>
      <p>© 2026 墨屿 · Built with Go, Vue & curiosity.</p>
    </footer>

    <AIChatPanel />
    <AIWritingPanel :open="writingOpen" @close="writingOpen = false" />
    <AnalyticsPanel :open="analyticsOpen" @close="analyticsOpen = false" />
    <DraftManager :open="draftsOpen" @close="draftsOpen = false" />

    <div v-if="loginOpen" class="modal-backdrop" @click.self="loginOpen = false">
      <form class="login-modal" @submit.prevent="submitLogin">
        <button type="button" class="modal-close" @click="loginOpen = false"><X :size="19" /></button>
        <div class="login-icon"><BookOpen :size="22" /></div>
        <span class="kicker">WELCOME BACK</span>
        <h2>登录墨屿</h2>
        <p>首次登录会自动创建你的读者账号。</p>
        <label>用户名<input v-model="username" autocomplete="username" required /></label>
        <label>密码<input v-model="password" type="password" autocomplete="current-password" required /></label>
        <p v-if="loginError" class="form-error">{{ loginError }}</p>
        <button class="modal-submit" :disabled="loggingIn">{{ loggingIn ? '正在登录…' : '登录并继续' }}</button>
      </form>
    </div>
  </div>
</template>
