# 墨屿 AI 博客前端

Task 10 的 Vue 3 + Vite MVP。包含博客首页、文章详情、登录和 AI 助手。

## 无需安装（CDN 模式）

推荐直接由 Blog 服务托管。默认目录结构下启动 `blog-api` 后访问：

```text
http://127.0.0.1:8080/web/
```

也可以用任意静态服务器打开项目根目录，例如：

```bash
python -m http.server 5173
```

然后访问 `http://127.0.0.1:5173`。Vue、Marked 和 DOMPurify 由 jsDelivr CDN 加载。

## 标准工程模式

```bash
npm install
npm run dev
```

默认将 `/api` 代理到 `http://127.0.0.1:8080`。可通过 `BLOG_API_URL` 修改 Blog API 地址。

AI 请求链路：浏览器 → `POST /api/ai/assistant/stream` → Blog API → Agent。统一入口会先识别 `knowledge_rag` 或 `blog_tool` 意图，再转发到对应能力；旧的 `/api/ai/chat/stream` 仍保留兼容旧客户端。

已包含 Task 11/12 展示能力：AI Markdown 写作与确认保存草稿、文章一键总结、评论情感/主题分析，以及热门文章/标签/阅读评论统计。所有写入都需要登录并由用户显式确认；分析能力只读取已发布内容。

Blog 与 Agent 启动前需要配置同一个 `BLOG_INTERNAL_TOKEN`。统一入口默认使用 Agent 的 `/v1/blog/assistant/stream`，也可将 Blog 环境变量 `AGENT_UNIFIED_CHAT_STREAM_PATH` 设置为 `/v1/unified/chat/stream` 使用图版 Flow；旧网关路径仍可通过 `AGENT_CHAT_STREAM_PATH` 在 `/v1/rag/chat/stream` 与 `/v1/blog/chat/stream` 之间切换。
