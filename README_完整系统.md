# 🚀 博客系统完整启动指南

## 📊 系统组成

```
你的完整博客系统：

1. blog-api (8080)          - 博客后端服务
2. blog-web (3000)          - 博客前端界面
3. go-agent-main (8081)     - AI写作服务 + 定时任务
```

---

## ⚡ 快速启动

### 方法1：一键启动脚本（推荐）

```powershell
cd C:\Users\lenovo\Documents\vibe
.\启动所有服务.ps1
```

然后手动启动前端：
```powershell
cd blog-web
npm run dev
```

### 方法2：手动启动

```powershell
# 1. 启动blog-api
cd C:\Users\lenovo\Documents\vibe\blog-api
.\blog-api.exe

# 2. 启动go-agent
cd C:\Users\lenovo\Documents\vibe\go-agent-main
.\go-agent-with-scheduler.exe

# 3. 启动前端
cd C:\Users\lenovo\Documents\vibe\blog-web
npm run dev
```

---

## 🔍 验证服务

```powershell
# 检查blog-api
Invoke-WebRequest http://localhost:8080/

# 检查go-agent
Invoke-WebRequest http://localhost:8081/health

# 检查前端
# 打开浏览器访问 http://localhost:3000
```

---

## 🛑 停止所有服务

```powershell
# 停止blog-api
Get-Process | Where-Object {$_.MainWindowTitle -match "blog-api"} | Stop-Process -Force

# 停止go-agent
Get-Process -Name "go-agent*" | Stop-Process -Force

# 停止前端（在运行npm的终端按 Ctrl+C）
```

---

## 📡 服务端口

| 服务 | 端口 | 用途 |
|------|------|------|
| blog-api | 8080 | 博客核心API |
| go-agent | 8081 | AI写作API |
| blog-web | 3000 | 前端界面 |

---

## 🎯 AI写作功能

### API调用
```javascript
// 在blog-web中调用
fetch('http://localhost:8081/v1/collaborative/writing', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    topic: "文章主题",
    author_id: 1,
    target_words: 2000
  })
})
```

### 定时任务
- **时间**: 每天早上 09:00
- **功能**: 自动生成文章
- **配置**: go-agent-main/config.yaml

---

## 📁 项目结构

```
C:\Users\lenovo\Documents\vibe\
├── blog-api/
│   └── blog-api.exe              # 博客后端服务
├── blog-web/
│   └── (前端代码)                # Vue/React前端
├── go-agent-main/
│   ├── go-agent-with-scheduler.exe   # AI写作服务
│   └── config.yaml                   # 配置文件
└── 启动所有服务.ps1              # 一键启动脚本
```

---

## 🔧 常用命令

### 查看日志
```powershell
# blog-api日志（查看窗口输出）

# go-agent日志
cd go-agent-main
Get-Content startup_error.log -Tail 20

# 查看定时任务
Get-Content startup_error.log | Select-String "AutoPublish"
```

### 重启服务
```powershell
# 重启go-agent
Get-Process -Name "go-agent*" | Stop-Process -Force
cd C:\Users\lenovo\Documents\vibe\go-agent-main
.\go-agent-with-scheduler.exe
```

---

## 📝 开发流程

### 添加AI写作功能到blog-web

1. **在前端添加按钮**
```vue
<button @click="generateArticle">AI生成文章</button>
```

2. **调用API**
```javascript
async generateArticle() {
  const response = await fetch('http://localhost:8081/v1/collaborative/writing', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      topic: this.topic,
      author_id: 1,
      target_words: 2000
    })
  });
  
  const data = await response.json();
  this.articleContent = data.content;
}
```

---

## ⏰ 自动化

**明天早上9点**：
- go-agent自动生成文章
- 保存为草稿
- 可在blog-web中查看和发布

---

## 🎉 完成清单

- [x] blog-api 运行在 8080
- [x] go-agent 运行在 8081
- [x] 定时任务已启用
- [x] 清理了测试文件
- [x] 只保留核心exe
- [x] DeepSeek API已配置
- [x] 文档完善

---

## 📞 问题排查

**Q: blog-api启动失败？**
A: 检查8080端口是否被占用，MySQL是否运行

**Q: go-agent API调用失败？**
A: 检查DeepSeek API Key是否有效

**Q: 前端无法访问API？**
A: 检查CORS配置，确保允许跨域

---

**最后更新**: 2026-09-09 19:30
**系统状态**: ✅ 全部运行中

祝你睡个好觉！明天早上9点系统会自动生成第一篇文章！🌙
