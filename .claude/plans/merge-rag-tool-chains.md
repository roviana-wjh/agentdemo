# 合并RAG链路和工具调用链路 - 实现计划

## 📋 项目背景

当前系统有两条独立的AI处理链路：
1. **RAG链路** (`RAGChatFlow`) - 基于向量检索的知识问答
2. **工具调用链路** (`BlogAssistant`) - 基于工具调用的博客数据查询

现在需要将这两条链路合并，通过**意图识别节点**作为统一入口进行路由。

---

## 🔍 现状分析

### 1. 现有链路架构

#### RAG链路 (`flow/rag_chat.go`)
```
RAGChatInput → PreProcess → Rewrite → QueryToMsgs → Retrieve (子图) → ConstructMessages → Chat → 输出
```

**特点：**
- 会话管理（SessionID、历史压缩）
- Query重写（基于历史上下文）
- 混合检索（Milvus + ES → RRF融合 → Rerank）
- 引用生成（Citation）
- 流式输出

#### 工具调用链路 (`flow/blog_assistant.go`)
```
[]*schema.Message → ReAct Agent (Tool Calling) → 输出
```

**特点：**
- 使用ReAct Agent模式
- 6个博客工具：搜索文章、获取详情、热门文章、评论、标签、统计
- 无会话管理
- 流式输出

#### 现有统一入口 (`flow/unified_assistant.go`)
```
Query → Classify (意图识别) → 路由到RAG或BlogAssistant
```

**问题：**
- 意图识别与链路执行**分离**在API层
- 工具链路**没有会话管理**
- **无法复用**RAG链路的会话压缩、历史管理能力

---

## 🎯 目标设计

### 架构图

```
                        ┌─────────────────────────┐
                        │  UnifiedChatInput       │
                        │  - Query                │
                        │  - SessionID            │
                        │  - CitationSink         │
                        └───────────┬─────────────┘
                                    │
                        ┌───────────▼─────────────┐
                        │  PreProcess             │
                        │  加载Session            │
                        └───────────┬─────────────┘
                                    │
                        ┌───────────▼─────────────┐
                        │  IntentClassify         │
                        │  意图识别节点            │
                        └───────────┬─────────────┘
                                    │
                    ┌───────────────┴───────────────┐
                    │                               │
        ┌───────────▼─────────┐       ┌───────────▼─────────┐
        │  KnowledgeRAGBranch │       │  BlogToolBranch     │
        │  (RAG检索+对话)      │       │  (ReAct工具调用)    │
        └───────────┬─────────┘       └───────────┬─────────┘
                    │                               │
                    └───────────────┬───────────────┘
                                    │
                        ┌───────────▼─────────────┐
                        │  PostProcess            │
                        │  保存Session            │
                        └───────────┬─────────────┘
                                    │
                                 输出
```

---

## 🛠️ 实现方案

### 方案选择：基于Graph的统一流程

**选择理由：**
1. ✅ 利用Eino Graph的状态管理能力
2. ✅ 统一会话管理和历史压缩
3. ✅ 保持流式输出能力
4. ✅ 可扩展（未来可添加更多意图分支）

---

## 📝 详细实现步骤

### Step 1: 创建统一的Graph状态结构

**文件：** `go-agent-main/flow/unified_chat.go` (新建)

```go
type UnifiedGraphState struct {
    Input       UnifiedChatInput
    Session     *memory.Session
    Intent      string           // knowledge_rag / blog_tool
    IntentReason string
    Docs        []*schema.Document  // RAG检索到的文档
    QueryRewritten string           // 重写后的查询
}

type UnifiedChatInput struct {
    Query        string
    SessionID    string
    CitationSink chan<- []Citation
}
```

---

### Step 2: 实现意图识别节点

**在同一文件中添加：**

```go
func buildIntentNode(chatModel model.BaseChatModel) compose.InvokableLambda {
    return compose.InvokableLambda(func(ctx context.Context, input UnifiedChatInput) (string, error) {
        // 使用轻量级模型进行意图分类
        response, err := chatModel.Generate(ctx, []*schema.Message{
            schema.SystemMessage(`意图分类器。输出JSON: {"intent":"knowledge_rag|blog_tool","reason":"原因"}
knowledge_rag: 询问知识、技术解释、文章内容
blog_tool: 查询热门文章、统计数据、评论、标签`),
            schema.UserMessage(input.Query),
        })
        
        // 解析JSON并保存到状态
        var result IntentResult
        json.Unmarshal([]byte(extractJSON(response.Content)), &result)
        
        compose.ProcessState[*UnifiedGraphState](ctx, func(ctx context.Context, state *UnifiedGraphState) error {
            state.Intent = result.Intent
            state.IntentReason = result.Reason
            return nil
        })
        
        return result.Intent, nil
    })
}
```

---

### Step 3: 构建RAG分支（复用现有逻辑）

**子图节点：**

```go
// RAG分支：Rewrite → Retrieve → ConstructRAGMessages
ragBranch := compose.NewGraph[string, []*schema.Message](...)

// 节点1: Query重写
ragBranch.AddLambdaNode("Rewrite", ...)

// 节点2: 混合检索（复用retrieverSubGraph）
ragBranch.AddGraphNode("Retrieve", retrieverSubGraph)

// 节点3: 构建RAG Prompt
ragBranch.AddLambdaNode("ConstructRAGMessages", func(docs []*schema.Document) {
    // 从状态获取Session历史
    // 构建：System Prompt + 背景摘要 + 历史 + 参考知识 + 当前问题
    // 发送Citations
})
```

---

### Step 4: 构建工具调用分支（集成ReAct）

**子图节点：**

```go
// Tool分支：ConstructToolMessages → ReActAgent
toolBranch := compose.NewGraph[string, []*schema.Message](...)

// 节点1: 构建工具调用Prompt
toolBranch.AddLambdaNode("ConstructToolMessages", func(query string) {
    // 从状态获取Session历史
    // 构建：System Persona + 背景摘要 + 历史 + 当前问题
})

// 节点2: ReAct Agent（复用BlogAssistant的agent）
// 需要将BlogAssistant.agent暴露为可复用组件
toolBranch.AddReactNode("ReActAgent", blogAssistant.agent)
```

---

### Step 5: 实现分支路由逻辑

```go
g.AddBranch("IntentRouter", compose.NewGraphBranch(
    func(ctx context.Context, intent string) (string, error) {
        var targetNode string
        compose.ProcessState[*UnifiedGraphState](ctx, func(ctx context.Context, state *UnifiedGraphState) error {
            if state.Intent == IntentKnowledgeRAG {
                targetNode = "RAGBranch"
            } else {
                targetNode = "ToolBranch"
            }
            return nil
        })
        return targetNode, nil
    },
    map[string]bool{
        "RAGBranch":  true,
        "ToolBranch": true,
    },
))
```

---

### Step 6: 统一Chat节点（支持流式输出）

```go
g.AddChatModelNode("UnifiedChat", chatModel,
    compose.WithStreamStatePostHandler(func(ctx context.Context, stream *schema.StreamReader[*schema.Message], state *UnifiedGraphState) {
        // 流式读取
        // 累积完整回复
        // 更新Session历史
        // 异步保存Session（压缩+存储）
    }),
)
```

---

### Step 7: 修改API层

**文件：** `go-agent-main/api/unified_chat_stream.go` (新建或替换 `unified_assistant.go`)

```go
func UnifiedChatStream(c *gin.Context) {
    var req struct {
        Query     string `json:"query" binding:"required"`
        SessionID string `json:"session_id"`
        RequestID string `json:"request_id,omitempty"`
    }
    // ... 参数校验
    
    writer := NewSSEWriter(...)
    writer.Start()
    
    // 调用统一Graph
    unifiedFlow, err := flow.GetUnifiedChatFlow()
    citationSink := make(chan []flow.Citation, 1)
    
    stream, err := unifiedFlow.Stream(ctx, flow.UnifiedChatInput{
        Query:        req.Query,
        SessionID:    req.SessionID,
        CitationSink: citationSink,
    })
    
    // 流式输出（根据意图发送不同的event）
    // - Route event: 发送意图分类结果
    // - Delta event: 发送内容增量
    // - Citation event: RAG时发送引用
    // - Done event: 结束
}
```

---

### Step 8: 修改初始化逻辑

**文件：** `go-agent-main/main.go`

```go
// 替换原有的InitRAGChatFlow和InitBlogAssistant
err = flow.InitUnifiedChatFlow(ctx, memStore, taskModel)
if err != nil {
    log.Printf("warning: UnifiedChatFlow init failed: %v", err)
} else {
    log.Println("UnifiedChatFlow compiled and cached")
}
```

---

## 🔧 技术细节

### 会话管理统一

**问题：** 原BlogAssistant无会话管理

**方案：** 
- 在PreProcess节点统一加载Session
- 在PostProcess节点统一保存Session
- 两条分支都从状态中获取历史

```go
// 在RAG分支和Tool分支的消息构建中统一处理
messages := []*schema.Message{
    schema.SystemMessage(systemPrompt),
}
if state.Session.Summary != "" {
    messages = append(messages, schema.SystemMessage("背景摘要: "+state.Session.Summary))
}
messages = append(messages, state.Session.History...)
messages = append(messages, schema.UserMessage(state.Input.Query))
```

---

### 流式输出处理

**挑战：** Graph中间节点如何传递流式数据

**方案：**
1. 意图识别 → 发送Route事件（非流式）
2. RAG/Tool分支 → 返回构建好的Messages（非流式）
3. 统一Chat节点 → 流式输出（使用StreamStatePostHandler）

```go
compose.WithStreamStatePostHandler(func(ctx context.Context, stream *schema.StreamReader[*schema.Message], state *UnifiedGraphState) {
    reader, writer := schema.Pipe[*schema.Message](1)
    go func() {
        defer writer.Close()
        var answer strings.Builder
        for {
            chunk, err := stream.Recv()
            if err == io.EOF {
                // 保存会话
                state.Session.History = append(state.Session.History, 
                    schema.UserMessage(state.Input.Query),
                    schema.AssistantMessage(answer.String(), nil))
                queueSessionSave(state.Session, summarizer, store)
                return
            }
            answer.WriteString(chunk.Content)
            writer.Send(chunk, nil)
        }
    }()
    return reader, nil
})
```

---

### Citation处理

**仅RAG分支需要：**

```go
// 在RAG分支的ConstructRAGMessages节点中
compose.ProcessState[*UnifiedGraphState](ctx, func(ctx context.Context, state *UnifiedGraphState) error {
    citations := BuildCitations(docs)
    if state.Input.CitationSink != nil {
        select {
        case state.Input.CitationSink <- citations:
        default:
        }
    }
    return nil
})
```

---

## 📂 文件变更清单

### 新建文件
1. ✅ `go-agent-main/flow/unified_chat.go` - 统一Chat Graph实现
2. ✅ `go-agent-main/api/unified_chat_stream.go` - 统一API endpoint

### 修改文件
1. ✅ `go-agent-main/main.go` - 替换初始化逻辑
2. ✅ `go-agent-main/api/router.go` - 更新路由注册
3. ✅ `go-agent-main/flow/blog_assistant.go` - 暴露agent供复用

### 可选保留（向后兼容）
- `api/rag_chat_stream.go` - 可标记为deprecated
- `api/blog_chat_stream.go` - 可标记为deprecated
- `api/unified_assistant.go` - 可删除（被新的unified_chat_stream替代）

---

## 🧪 测试计划

### 1. 单元测试
- [ ] 意图识别准确性测试
- [ ] 会话状态管理测试
- [ ] RAG分支独立测试
- [ ] Tool分支独立测试

### 2. 集成测试
- [ ] 端到端流式输出测试
- [ ] 会话连续性测试（多轮对话）
- [ ] Citation生成测试（RAG场景）
- [ ] 工具调用正确性测试（Tool场景）

### 3. 性能测试
- [ ] 意图识别延迟（目标<200ms）
- [ ] 整体响应时间对比（不应显著增加）
- [ ] 并发会话测试

---

## ⚠️ 风险与注意事项

### 1. 向后兼容性
- 旧的`/v1/rag/chat/stream`和`/v1/blog/chat/stream`端点可保留
- 新端点使用`/v1/unified/chat/stream`

### 2. 意图识别准确性
- 需要足够的测试样例验证
- 考虑添加fallback机制（默认RAG）
- 可添加置信度阈值

### 3. 性能影响
- 增加一次意图识别LLM调用（~100-300ms）
- 使用轻量级模型减少延迟
- 可考虑缓存相似Query的意图

### 4. 状态管理复杂度
- Graph状态需要正确传递
- 注意并发安全（Session克隆）
- 流式处理中的状态更新时序

---

## 📊 优势总结

### ✅ 架构优势
1. **统一入口** - 一个endpoint处理所有场景
2. **会话复用** - 两条链路共享会话管理
3. **智能路由** - 自动选择最佳处理路径
4. **易扩展** - 可轻松添加新的意图分支

### ✅ 用户体验优势
1. 无需手动选择RAG或工具模式
2. 多轮对话保持上下文
3. 流式输出体验一致
4. 错误处理统一

---

## 🚀 实施建议

### Phase 1: 核心实现（2-3天）
1. 实现unified_chat.go的Graph结构
2. 集成RAG和Tool分支
3. 基本流式输出

### Phase 2: 完善功能（1-2天）
1. 会话管理集成
2. Citation处理
3. API层适配

### Phase 3: 测试与优化（1-2天）
1. 编写测试用例
2. 性能调优
3. 文档更新

---

## ❓ 待确认问题

1. **意图识别模型选择？**
   - 使用当前的IntentModelType配置？
   - 还是固定使用快速模型？

2. **旧API端点处理？**
   - 完全删除还是标记deprecated？
   - 是否需要重定向到新端点？

3. **会话ID生成策略？**
   - 前端传入还是后端生成？
   - 匿名用户如何处理？

4. **意图分类失败fallback？**
   - 默认走RAG还是返回错误？
   - 是否支持用户手动指定意图？

---

## 📚 参考资料

- Eino Graph文档：状态管理、分支路由
- 现有代码：`rag_chat.go`, `blog_assistant.go`, `unified_assistant.go`
- ReAct模式：工具调用与推理循环
