# 🚀 Go服务启动指南

## 当前服务状态

✅ **Go后端服务**: 正在运行
✅ **端口**: 8081
✅ **定时任务**: 已启用（每天09:00）

---

## 🔧 服务管理

### 启动服务
```powershell
cd C:\Users\lenovo\Documents\vibe\go-agent-main
.\go-agent-with-scheduler.exe
```

### 停止服务
```powershell
Get-Process -Name "go-agent*" | Stop-Process -Force
```

### 重启服务
```powershell
# 停止
Get-Process -Name "go-agent*" | Stop-Process -Force
Start-Sleep -Seconds 2

# 启动
.\go-agent-with-scheduler.exe
```

---

## 📡 API端点

### 多Agent协作写作
```
POST http://localhost:8081/v1/collaborative/writing
Content-Type: application/json

{
  "topic": "Redis缓存优化实战指南",
  "author_id": 1,
  "target_words": 2000
}
```

**响应**:
```json
{
  "success": true,
  "content": "文章内容...",
  "word_count": 2156,
  "review_score": 87.5,
  "iterations": 1,
  "total_time": "2m30s"
}
```

---

## ⏰ 定时任务

**配置文件**: config.yaml
```yaml
auto_publish:
  enabled: true
  cron: "09:00"  # 每天早上9点
```

**修改时间**:
1. 编辑 config.yaml 中的 `cron` 字段
2. 重启服务

---

## 🔍 查看日志

```powershell
# 查看最新20行
Get-Content startup_error.log -Tail 20

# 实时监控
Get-Content startup_error.log -Wait
```

---

## 🧪 测试API

```powershell
# 测试脚本
.\setup_deepseek_and_test.ps1
```

或者直接测试：
```powershell
$body = @{
    topic = "测试文章"
    author_id = 1
    target_words = 2000
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8081/v1/collaborative/writing" `
    -Method Post -ContentType "application/json" -Body $body
```

---

## 📝 博客前端集成

你的博客前端（blog-web）可以直接调用API：

```javascript
// 示例代码
fetch('http://localhost:8081/v1/collaborative/writing', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    topic: '文章主题',
    author_id: 1,
    target_words: 2000
  })
})
.then(res => res.json())
.then(data => {
  console.log('生成成功:', data.content);
});
```

---

## ⚙️ 配置说明

### DeepSeek API (已配置)
```yaml
ds_think_chat_model:
  api_key: "sk-a5c25bcb1c504e65b681475198ff3204"
  base_url: "https://api.deepseek.com"
  model: "deepseek-chat"
```

### 自动发布
```yaml
auto_publish:
  enabled: true   # 启用/禁用
  cron: "09:00"   # 每天执行时间
```

---

## 🎯 功能说明

### 多Agent协作流程
```
1. 研究员Agent   - 检索相关文章
2. 架构师Agent   - 设计大纲结构
3. 作家Agent     - 撰写文章内容
4. 审核Agent     - 质量评估打分
5. 编辑Agent     - 润色优化
```

### 自动化特性
- ✅ 每天自动生成文章
- ✅ 随机选择主题（20个预设）
- ✅ 质量评分保证
- ✅ 自动保存草稿

---

## ❓ 常见问题

**Q: 服务无法启动？**
A: 检查端口8081是否被占用

**Q: API调用失败？**
A: 检查DeepSeek API Key是否有效

**Q: 定时任务不执行？**
A: 确认config.yaml中 `enabled: true`

**Q: 如何查看定时任务状态？**
A: 查看日志中的 `[AutoPublish]` 相关信息

---

## 📞 维护

**日常操作**: 不需要任何操作，服务会自动运行

**定期检查**: 
- 查看日志确认定时任务执行
- 检查DeepSeek API余额
- 查看生成的文章质量

---

最后更新: 2026-09-09
