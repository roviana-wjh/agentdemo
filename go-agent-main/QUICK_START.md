# 🚀 快速启动指南

## 醒来后的3步操作

### 第1步: 修复模型配置 (2分钟)

1. 打开火山引擎控制台
   - 网址: https://console.volcengine.com/ark

2. 获取正确的Endpoint ID
   - 找到DeepSeek推理服务
   - 复制endpoint ID (格式: ep-xxxxx-xxxxx)

3. 更新配置
   ```powershell
   notepad C:\Users\lenovo\Documents\vibe\go-agent-main\config.yaml
   ```
   
   修改这两处：
   ```yaml
   ds_think_chat_model:
     model: "ep-你的endpoint-ID"  # 修改这里
   
   ds_quick_chat_model:
     model: "ep-你的endpoint-ID"  # 修改这里
   ```

---

### 第2步: 重启服务 (1分钟)

```powershell
cd C:\Users\lenovo\Documents\vibe\go-agent-main

# 停止旧服务
Get-Process -Name "go-agent*" | Stop-Process -Force

# 启动新服务
Start-Process -FilePath ".\go-agent-simplified.exe" -WindowStyle Normal

# 等待15秒
Start-Sleep -Seconds 15
```

---

### 第3步: 测试生成文章 (3分钟)

```powershell
# 生成第一篇文章
$body = @{
    topic = "Redis缓存优化实战指南"
    author_id = 1
    target_words = 2000
} | ConvertTo-Json

$response = Invoke-RestMethod -Uri "http://localhost:8081/v1/collaborative/writing" `
    -Method Post -ContentType "application/json" -Body $body

# 保存文章
$response.content | Out-File -FilePath "first_article.md" -Encoding UTF8

# 查看结果
Write-Host "✅ 文章生成成功！"
Write-Host "字数: $($response.word_count)"
Write-Host "得分: $($response.review_score)"
Write-Host "文件: first_article.md"

# 打开文章
notepad first_article.md
```

---

## 🎉 完成！

如果看到文章内容，说明系统完全正常！

下一步：
- 设置定时任务
- 实现每天自动发布
- 享受自动化内容创作！

---

## ❌ 如果还有错误

查看详细报告：
```powershell
notepad C:\Users\lenovo\Documents\vibe\go-agent-main\FINAL_REPORT.md
```

查看工作日志：
```powershell
notepad C:\Users\lenovo\Documents\vibe\go-agent-main\WORK_LOG.md
```

---

## 📞 获取帮助

重新问我：
- "还是有错误，日志是..."
- "模型配置在哪里找？"
- "如何设置定时发布？"
