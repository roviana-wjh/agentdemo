# 快速测试指南

## 🎯 一键测试

### Windows系统

1. **启动服务**（在一个终端中）
```bash
cd go-agent-main
go-agent.exe
```

2. **运行测试**（在另一个终端中）
```bash
cd C:\Users\lenovo\Documents\vibe\go-agent-main
run_test.bat
```

或手动运行：
```bash
go run test/intent_test.go
```

---

## 📊 测试数据集概览

| 测试类别 | 用例数 | 目标 |
|---------|--------|------|
| 关键词匹配-工具 | 7 | 验证快速关键词识别 |
| 关键词匹配-知识 | 6 | 验证知识问答关键词 |
| LLM语义分类 | 6 | 验证复杂语义理解 |
| 边界模糊查询 | 4 | 验证Fallback机制 |
| 混合场景 | 3 | 验证复合问题处理 |
| 上下文依赖 | 3 | 验证上下文处理 |
| **总计** | **33** | **覆盖全场景** |

---

## 📈 预期结果

✅ **准确率**: ≥85%  
✅ **响应时间**: ≤300ms  
✅ **关键词准确率**: ≥95%  
✅ **LLM准确率**: ≥80%  

---

## 🔍 典型测试用例

### 工具调用类（应识别为 blog_tool）
- ✅ "最近有哪些热门文章？" → keyword_match (0.95)
- ✅ "博客总共有多少篇文章？" → keyword_match (0.95)
- ✅ "给我推荐一些值得看的内容" → llm_classification (0.85)

### 知识问答类（应识别为 knowledge_rag）
- ✅ "什么是RESTful API？" → keyword_match (0.95)
- ✅ "如何实现JWT认证？" → keyword_match (0.95)
- ✅ "GraphQL相比REST有哪些好处" → llm_classification (0.85)

### Fallback场景（模糊问题）
- ⚠️ "告诉我更多" → fallback_default_safe (0.3)
- ⚠️ "这个怎么样" → fallback_default_safe (0.3)

---

## 📄 输出文件

执行测试后会生成：

1. **test/intent_test_report.json** - 详细测试报告（JSON格式）
2. **控制台输出** - 实时测试结果

---

## 🐛 故障排查

### 问题1: 服务连接失败
```
❌ 加载测试数据集失败: connection refused
```
**解决**: 确保 go-agent.exe 已启动，监听 8081 端口

### 问题2: 准确率过低（<70%）
**可能原因**:
- 意图模型配置错误
- 关键词列表不匹配实际场景
- LLM提示词需要优化

**解决步骤**:
1. 检查 config.yaml 中的 intent_model_type
2. 查看失败用例的详细错误信息
3. 调整 flow/unified_chat.go 中的关键词列表

### 问题3: 响应时间过长（>500ms）
**可能原因**:
- LLM调用延迟高
- 网络问题

**解决步骤**:
1. 检查模型配置，考虑使用更快的模型
2. 增加关键词覆盖率，减少LLM调用频率

---

## 📊 报告解读

### 关键指标

```json
{
  "summary": {
    "total_cases": 33,
    "passed": 29,           // ✅ 通过数量
    "failed": 4,            // ❌ 失败数量
    "accuracy": 87.9,       // 📊 准确率
    "avg_response_time": 178, // ⚡ 平均响应时间(ms)
    "failed_by_intent": {
      "blog_tool": 2,       // 工具调用识别失败2个
      "knowledge_rag": 2    // 知识问答识别失败2个
    },
    "failed_by_method": {
      "keyword_match": 0,         // 关键词匹配失败0个
      "llm_classification": 3,    // LLM分类失败3个
      "fallback_default_safe": 1  // Fallback失败1个
    }
  }
}
```

### 优化建议

如果 **failed_by_method.llm_classification** 较高：
- 优化LLM分类的系统提示词
- 增加典型问题的关键词覆盖

如果 **failed_by_method.keyword_match** 较高：
- 补充缺失的关键词
- 调整关键词优先级

---

## 🎯 下一步

1. **首次测试**: 运行 `run_test.bat`，建立基线
2. **分析结果**: 查看 `test/intent_test_report.json`
3. **优化改进**: 根据失败用例调整关键词或提示词
4. **回归测试**: 再次运行，验证改进效果
5. **达标验收**: 准确率≥85%，响应时间≤300ms

---

## 📞 需要帮助？

- 详细文档: `test/README.md`
- 测试脚本: `test/intent_test.go`
- 数据集: `test/intent_test_dataset.json`
- 实现代码: `flow/unified_chat.go`
