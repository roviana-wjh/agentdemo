# 自动发布脚本
# 用途：每天自动生成并发布一篇技术文章

param(
    [string]$Topic = "",
    [int]$TargetWords = 2000
)

# 主题列表（如果没有指定主题，随机选择一个）
$topics = @(
    "Redis缓存优化实战指南",
    "MySQL索引优化技巧",
    "Docker容器化部署指南",
    "Nginx反向代理配置",
    "Git分支管理最佳实践",
    "RESTful API设计原则",
    "微服务架构模式详解",
    "JWT身份认证实战",
    "SQL注入防护技术",
    "高并发系统设计",
    "分布式事务解决方案",
    "消息队列应用场景",
    "ElasticSearch搜索优化",
    "MongoDB数据建模",
    "Python异步编程指南",
    "JavaScript闭包深入解析",
    "Vue3组件设计模式",
    "React Hooks实战技巧",
    "TypeScript类型体操",
    "Webpack构建优化"
)

# 如果没有指定主题，随机选择
if ([string]::IsNullOrEmpty($Topic)) {
    $Topic = $topics | Get-Random
}

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "自动发布系统" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "主题: $Topic"
Write-Host "目标字数: $TargetWords"
Write-Host "时间: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
Write-Host ""

# 1. 生成文章
Write-Host "Step 1: 生成文章..." -ForegroundColor Yellow

$body = @{
    topic = $Topic
    author_id = 1
    target_words = $TargetWords
} | ConvertTo-Json

try {
    $startTime = Get-Date

    $response = Invoke-RestMethod -Uri "http://localhost:8081/v1/collaborative/writing" `
        -Method Post -ContentType "application/json" -Body $body -TimeoutSec 300

    $elapsed = (Get-Date) - $startTime

    Write-Host "✅ 文章生成成功" -ForegroundColor Green
    Write-Host "   耗时: $([math]::Round($elapsed.TotalSeconds, 1)) 秒"
    Write-Host "   字数: $($response.word_count)"
    Write-Host "   得分: $($response.review_score)"
    Write-Host ""

    # 2. 保存到文件
    Write-Host "Step 2: 保存文章..." -ForegroundColor Yellow

    $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
    $safeTitle = $Topic -replace '[\\/:*?"<>|]', '_'
    $filename = "articles\${timestamp}_${safeTitle}.md"

    # 确保目录存在
    if (-not (Test-Path "articles")) {
        New-Item -ItemType Directory -Path "articles" | Out-Null
    }

    $response.content | Out-File -FilePath $filename -Encoding UTF8

    Write-Host "✅ 文章已保存" -ForegroundColor Green
    Write-Host "   文件: $filename"
    Write-Host ""

    # 3. 记录日志
    Write-Host "Step 3: 记录日志..." -ForegroundColor Yellow

    $logEntry = @"
[$timestamp] 文章发布成功
主题: $Topic
字数: $($response.word_count)
得分: $($response.review_score)
文件: $filename
---
"@

    $logEntry | Out-File -FilePath "publish_log.txt" -Append -Encoding UTF8

    Write-Host "✅ 日志已记录" -ForegroundColor Green
    Write-Host ""

    # 4. 统计信息
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "发布完成！" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    Write-Host ""

    # 统计总发布数量
    $articleCount = (Get-ChildItem "articles" -Filter "*.md").Count
    Write-Host "📊 统计:" -ForegroundColor Cyan
    Write-Host "   累计发布: $articleCount 篇"
    Write-Host "   本次得分: $($response.review_score) 分"
    Write-Host "   本次字数: $($response.word_count) 字"
    Write-Host ""

    Write-Host "🎉 今日任务完成！" -ForegroundColor Green

} catch {
    Write-Host ""
    Write-Host "❌ 发布失败: $_" -ForegroundColor Red
    Write-Host ""

    # 记录错误日志
    $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
    $errorLog = @"
[$timestamp] 文章发布失败
主题: $Topic
错误: $_
---
"@

    $errorLog | Out-File -FilePath "publish_error.txt" -Append -Encoding UTF8

    exit 1
}
