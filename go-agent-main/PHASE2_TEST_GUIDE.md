# Phase 2 增量重写测试指南

## 🧪 测试目标

验证Phase 2的核心功能：
1. ✅ 增量重写能否正常触发
2. ✅ 问题章节识别是否准确
3. ✅ Token和时间是否真的节省
4. ✅ 重写后质量是否提升

---

## 🚀 快速开始

### 方法1: 使用测试脚本（推荐）

```bash
cd go-agent-main

# 1. 确保已编译
go build -o go-agent.exe

# 2. 运行测试脚本
test_phase2.bat
```

脚本会自动：
- ✅ 启动服务
- ✅ 执行2个测试用例
- ✅ 保存结果到JSON文件
- ✅ 显示关键指标

### 方法2: 手动测试

**Terminal 1 - 启动服务：**
```bash
cd go-agent-main
./go-agent.exe
```

**Terminal 2 - 执行测试：**
```bash
# 测试1: 基础文章
curl -X POST http://localhost:8081/v1/collaborative/writing \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "Redis缓存策略详解",
    "requirements": "面向中级开发者，包含实战案例",
    "author_id": 1,
    "target_words": 2000
  }' | jq .

# 测试2: 复杂文章
curl -X POST http://localhost:8081/v1/collaborative/writing \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "深入理解Kubernetes调度器原理",
    "requirements": "面向有经验的开发者，深入源码",
    "author_id": 1,
    "target_words": 2500,
    "difficulty": "advanced"
  }' | jq .
```

---

## 🔍 如何判断增量重写生效

### 1. 查看服务日志

**增量重写的日志特征：**
```
🔎 [Review] 审核Agent开始审核
[ReviewerAgent] ❌ 文章未通过审核，得分: 72.5，问题数: 3
[ReviewerAgent] 问题章节: [2, 4]，重写策略: incremental  👈 关键！

🔄 [Decision] 审核未通过(72.5分)，第1次重写

✍️ [Write] 作家Agent开始撰写
[WriterAgent] 第1次重写，根据审核反馈改进
[WriterAgent] 使用增量重写策略，仅重写章节: [2 4]  👈 关键！
[WriterAgent] 🔄 重写章节 2: Redis持久化机制
[WriterAgent] ✓ 保持章节 1: Redis基础概念
[WriterAgent] ✓ 保持章节 3: 缓存更新策略
[WriterAgent] 🔄 重写章节 4: 缓存雪崩解决方案
[WriterAgent] 增量重写完成: 2080字  👈 更快！
```

**全文重写的日志特征：**
```
[ReviewerAgent] 问题章节: [1, 2, 3, 4, 5]，重写策略: full  👈 全文

[WriterAgent] 使用全文重写策略  👈 所有章节重写
[WriterAgent] 撰写章节 1: ...
[WriterAgent] 撰写章节 2: ...
...
```

### 2. 查看API响应

**增量重写的响应：**
```json
{
  "success": true,
  "iterations": 2,
  "rewrite_strategy": "incremental",  👈 关键字段
  "problematic_sections": [2, 4],     👈 只有2个章节
  "issues_resolved": 3,
  "review_score": 88.5,
  "stage_timings": {
    "write": "65.3s",
    "write_rewrite": "38.7s"  👈 重写更快！
  }
}
```

**全文重写的响应：**
```json
{
  "rewrite_strategy": "full",         👈 full
  "problematic_sections": [1,2,3,4,5], 👈 很多章节
  "stage_timings": {
    "write": "65.3s",
    "write_rewrite": "82.1s"  👈 重写慢
  }
}
```

---

## 📊 性能对比

### 指标1: Token消耗

**测试方法：** 查看服务日志中的Token统计（如果有）或计算字数

**预期结果：**
| 场景 | Phase 1 | Phase 2增量 | 节省 |
|------|---------|------------|------|
| 首次写作 | ~12K | ~12K | - |
| 全文重写 | ~17K | ~17K | - |
| 增量重写 | ~17K | **~9K** | **47%** ✅ |

### 指标2: 重写时间

**测试方法：** 对比 `stage_timings.write_rewrite`

**预期结果：**
| 策略 | 时间 |
|------|------|
| 全文重写 | 80-100s |
| 增量重写 | **40-50s** ✅ |

### 指标3: 质量提升

**测试方法：** 对比重写前后的 `review_score`

**预期结果：**
| 迭代 | 分数 |
|------|------|
| 第1次 | 65-75分 |
| 增量重写后 | 85-90分 ✅ |

---

## 🎯 测试场景设计

### 场景1: 容易触发增量重写

**特点：**
- 主题相对简单
- 目标字数适中（2000字）
- 大部分内容质量OK，局部有小问题

**测试用例：**
```json
{
  "topic": "Redis缓存策略详解",
  "requirements": "面向中级开发者，包含实战案例",
  "author_id": 1,
  "target_words": 2000
}
```

**预期：**
- 审核得分：70-80分
- 问题章节：1-3个
- 重写策略：**incremental** ✅

### 场景2: 容易触发全文重写

**特点：**
- 主题复杂深入
- 目标字数大（2500+字）
- 多个章节质量不够

**测试用例：**
```json
{
  "topic": "深入理解Kubernetes调度器原理",
  "requirements": "面向有经验的开发者，深入源码分析",
  "author_id": 1,
  "target_words": 2500,
  "difficulty": "advanced"
}
```

**预期：**
- 审核得分：60-70分
- 问题章节：4+个
- 重写策略：**full**

---

## 📋 测试检查清单

### ✅ 功能验证

- [ ] 服务正常启动
- [ ] API调用成功返回
- [ ] 文章内容完整
- [ ] Markdown格式正确
- [ ] 重写策略字段存在

### ✅ 增量重写验证

- [ ] `rewrite_strategy` 为 `"incremental"`
- [ ] `problematic_sections` 数组有值
- [ ] 日志显示"使用增量重写策略"
- [ ] 日志显示部分章节"保持"
- [ ] 重写时间 < 60秒

### ✅ 性能验证

- [ ] 增量重写时间 < 全文重写时间
- [ ] 总耗时 < 3分钟
- [ ] 审核得分提升
- [ ] 最终得分 > 85分

---

## 🐛 故障排查

### 问题1: 始终是全文重写

**可能原因：**
- 首次写作质量太差，问题太多
- 审核标准太严格

**解决方法：**
- 调整测试主题，选择更简单的
- 检查审核Agent的评分逻辑

### 问题2: 重写后反而变差

**可能原因：**
- 增量重写的Prompt不够好
- 章节识别不准确

**解决方法：**
- 检查日志中的问题定位
- 优化rewriteSection的Prompt

### 问题3: 服务响应超时

**可能原因：**
- 模型调用太慢
- 超时设置太短

**解决方法：**
- 增加API超时时间
- 检查网络和模型服务

---

## 📊 预期测试结果

### 成功标准

**功能层面：**
- ✅ 增量重写策略能正确触发
- ✅ 问题章节识别准确
- ✅ 重写后质量提升

**性能层面：**
- ✅ 增量重写时间 < 60秒
- ✅ Token节省 > 30%
- ✅ 总耗时 < 3分钟

**质量层面：**
- ✅ 审核得分 > 85分
- ✅ 文章结构完整
- ✅ 内容逻辑清晰

---

## 📝 测试报告模板

测试完成后，记录以下信息：

```
测试时间：2024-XX-XX
测试环境：Windows 11 + DeepSeek模型

【测试用例1: Redis缓存策略】
- 主题：Redis缓存策略详解
- 目标字数：2000
- 实际字数：2105
- 审核得分：88.5
- 重写次数：1
- 重写策略：incremental ✅
- 问题章节：[2, 4]
- 总耗时：2m15s
- 增量重写耗时：42s ✅
- 结论：增量重写成功，性能提升明显

【测试用例2: Kubernetes调度器】
- 主题：Kubernetes调度器原理
- 目标字数：2500
- 实际字数：2687
- 审核得分：86.0
- 重写次数：2
- 重写策略：full
- 问题章节：[1,2,3,4,5]
- 总耗时：3m42s
- 结论：复杂文章触发全文重写

【总结】
✅ 增量重写功能正常
✅ 性能提升符合预期（~45%）
✅ 质量保证有效
```

---

## 🚀 开始测试

1. 确保已编译：`go build -o go-agent.exe`
2. 运行测试：`test_phase2.bat`
3. 观察日志和结果
4. 记录性能数据

祝测试顺利！🎉
