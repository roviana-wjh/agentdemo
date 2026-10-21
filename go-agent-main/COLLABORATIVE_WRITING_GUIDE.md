# 多Agent协作写作系统 - 使用指南

## 🎉 Phase 1 完成！

恭喜！你已经实现了一个**真正的多Agent协作系统**！

---

## 🏗️ 系统架构

```
用户输入主题
    ↓
🔍 研究员Agent
   - 搜索相关博客文章
   - 提取关键发现
   - 生成研究摘要
    ↓
🏗️ 架构师Agent
   - 设计文章大纲
   - 规划章节结构
   - 分配字数
    ↓
✍️ 作家Agent
   - 根据大纲撰写
   - 引用研究结果
   - 生成完整文章
    ↓
🔎 审核Agent
   - 评估质量（100分制）
   - 检查风险
   - 提出改进建议
    ↓
   [条件分支]
  /           \
通过          不通过
 ↓             ↓
             重写（最多3次）
 ↓             ↓
💎 编辑Agent ←┘
   - 润色优化
   - 修正语法
   - 提升可读性
    ↓
✅ 输出成品文章
```

---

## 🚀 快速开始

### 1. 启动服务

```bash
cd go-agent-main
go-agent.exe
```

服务启动后会看到：
```
CollaborativeWriting initialized (Multi-Agent System)
```

### 2. 调用API

**请求示例：**

```bash
curl -X POST http://localhost:8081/v1/collaborative/writing \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "如何使用Docker部署Go应用",
    "requirements": "面向中级开发者，包含实战案例和代码示例",
    "author_id": 1,
    "target_words": 2000,
    "difficulty": "intermediate",
    "tone": "technical"
  }'
```

**响应示例：**

```json
{
  "success": true,
  "content": "# 如何使用Docker部署Go应用\n\n## 1. Docker基础概念\n\n...",
  "word_count": 2150,
  "review_score": 87.5,
  "iterations": 1,
  "total_time": "2m15s",
  "stage_timings": {
    "research": "25.3s",
    "architect": "12.1s",
    "write": "58.7s",
    "review": "18.5s",
    "edit": "20.8s"
  }
}
```

---

## 📋 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `topic` | string | ✅ | 文章主题 |
| `requirements` | string | ❌ | 额外要求（目标读者、风格等） |
| `author_id` | uint | ✅ | 作者ID |
| `target_words` | int | ❌ | 目标字数（默认2000） |
| `difficulty` | string | ❌ | 难度：`beginner`/`intermediate`/`advanced` |
| `tone` | string | ❌ | 语气：`formal`/`casual`/`technical` |

---

## 🎯 核心特性

### 1️⃣ 真正的多Agent协作
- ✅ 5个独立Agent，各司其职
- ✅ Agent间通过Graph State共享上下文
- ✅ 每个Agent有独立的人格设定

### 2️⃣ 智能迭代优化
- ✅ 审核不通过自动重写
- ✅ 最多3次迭代
- ✅ 每次重写基于审核反馈改进

### 3️⃣ 上下文管理
- ✅ 研究摘要压缩（10000 tokens → 500 tokens）
- ✅ 大纲指导写作方向
- ✅ 审核反馈注入到重写过程

### 4️⃣ 可观察性
- ✅ 每个Agent的日志输出
- ✅ 各阶段耗时统计
- ✅ 完整的执行追踪

---

## 📊 性能指标

### 预期耗时（2000字文章）

| 阶段 | 耗时 | 说明 |
|------|------|------|
| 研究 | 20-30s | 搜索+总结 |
| 架构 | 10-15s | 设计大纲 |
| 写作 | 50-80s | 撰写正文 |
| 审核 | 15-25s | 质量评估 |
| 编辑 | 20-30s | 润色优化 |
| **总计** | **2-3分钟** | 首次通过 |

如果需要重写，每次+1-1.5分钟。

### Token消耗估算

| Agent | 输入 | 输出 | 总计 |
|-------|------|------|------|
| 研究员 | 2K | 4K | 6K |
| 架构师 | 5K | 2K | 7K |
| 作家 | 7K | 5K | 12K |
| 审核 | 12K | 3K | 15K |
| 编辑 | 15K | 5K | 20K |
| **总计** | - | - | **60K** |

重写一次 +17K tokens

---

## 🧪 测试用例

### 测试1: 基础技术文章
```json
{
  "topic": "Redis缓存策略详解",
  "requirements": "初学者友好，包含实际案例",
  "author_id": 1,
  "difficulty": "beginner"
}
```

### 测试2: 深度技术文章
```json
{
  "topic": "深入理解Kubernetes调度器原理",
  "requirements": "面向有经验的开发者，深入源码分析",
  "author_id": 1,
  "target_words": 3000,
  "difficulty": "advanced",
  "tone": "technical"
}
```

### 测试3: 实战教程
```json
{
  "topic": "从零搭建微服务监控系统",
  "requirements": "完整的实战教程，包含代码和配置",
  "author_id": 1,
  "tone": "casual"
}
```

---

## 🔍 日志示例

```
🚀 [CollaborativeWriting] 开始多Agent协作写作
📝 主题: 如何使用Docker部署Go应用

🔍 [Research] 研究员Agent开始工作
[ResearcherAgent] 提取关键词: [Docker, Go应用, 部署]
[ResearcherAgent] 找到 5 篇相关文章
[ResearcherAgent] 研究完成，生成摘要 487 字

🏗️ [Architect] 架构师Agent开始设计大纲
[ArchitectAgent] 大纲设计完成: 6个章节，目标2000字

✍️ [Write] 作家Agent开始撰写
[WriterAgent] 撰写章节 1: Docker基础概念
[WriterAgent] 撰写章节 2: Go应用容器化
...
[WriterAgent] 文章撰写完成: 2087字，3个引用

🔎 [Review] 审核Agent开始审核
[ReviewerAgent] ✅ 文章通过审核，得分: 87.5

💎 [Edit] 编辑Agent开始润色
[EditorAgent] 深度润色完成: 2087字 → 2145字

🎉 [Finalize] 多Agent协作完成
📊 总耗时: 135.2s | 字数: 2145 | 得分: 87.5 | 重写次数: 0
```

---

## 🐛 故障排查

### 问题1: 编译失败
```
go build error: package not found
```
**解决**: 确保所有文件都在正确的目录

### 问题2: 研究员Agent失败
```
search_article tool not found
```
**解决**: 确保blog-api服务正常运行，内部token配置正确

### 问题3: 响应超时
```
context deadline exceeded
```
**解决**: 增加超时时间，或减少target_words

---

## 🎯 下一步计划

### Phase 2: 反馈循环优化（1-2天）
- [ ] 优化审核反馈的注入方式
- [ ] 添加重写历史追踪
- [ ] 实现增量重写（只改有问题的章节）

### Phase 3: 角色分工并行（2天）
- [ ] 拆分为正文、代码、配图Agent
- [ ] 实现并发生成
- [ ] 添加组装Agent

### Phase 4: 运营闭环（2-3天）
- [ ] 添加选题Agent
- [ ] 添加SEO优化Agent
- [ ] 添加发布和监控Agent

---

## 📞 技术支持

- 实施计划: `MULTI_AGENT_PLAN.md`
- 源代码: `flow/collaborative_writing/`
- API接口: `api/collaborative_writing.go`

---

## 🎊 恭喜！

你已经掌握了真正的多Agent系统核心技术：
- ✅ Agent间协作
- ✅ 上下文管理
- ✅ 迭代优化
- ✅ 条件分支

继续加油，实现更多高级特性！🚀
