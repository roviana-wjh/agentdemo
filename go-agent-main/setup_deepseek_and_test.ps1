# 一键配置并测试
# 使用DeepSeek官方API

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "一键配置DeepSeek API并测试" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# 1. 备份配置文件
Write-Host "Step 1: 备份配置文件..." -ForegroundColor Yellow
Copy-Item "config.yaml" "config.yaml.bak" -Force
Write-Host "✅ 已备份为 config.yaml.bak" -ForegroundColor Green
Write-Host ""

# 2. 修改配置
Write-Host "Step 2: 更新API配置..." -ForegroundColor Yellow
$config = Get-Content "config.yaml" -Raw

# 替换模型配置
$config = $config -replace 'base_url: "https://ark.cn-beijing.volces.com/api/v3"', 'base_url: "https://api.deepseek.com"'
$config = $config -replace 'model: "deepseek-v3-2-251201"', 'model: "deepseek-chat"'
$config = $config -replace 'api_key: "f6c0af0e-d859-4f46-b20f-cebfedd69a02"', 'api_key: "sk-a5c25bcb1c504e65b681475198ff3204"'

$config | Out-File "config.yaml" -Encoding UTF8

Write-Host "✅ 配置已更新为DeepSeek官方API" -ForegroundColor Green
Write-Host ""

# 3. 重启服务
Write-Host "Step 3: 重启服务..." -ForegroundColor Yellow
Get-Process -Name "go-agent*" -ErrorAction SilentlyContinue | Stop-Process -Force
Start-Sleep -Seconds 2

Start-Process -FilePath ".\go-agent-simplified.exe" -WindowStyle Normal

Write-Host "等待服务启动 (20秒)..."
Start-Sleep -Seconds 20

Write-Host "✅ 服务已启动" -ForegroundColor Green
Write-Host ""

# 4. 测试生成文章
Write-Host "Step 4: 测试生成文章..." -ForegroundColor Yellow
Write-Host "主题: Redis缓存优化实战指南"
Write-Host "预计耗时: 2-3分钟"
Write-Host ""

$body = @{
    topic = "Redis缓存优化实战指南"
    author_id = 1
    target_words = 2000
} | ConvertTo-Json

try {
    $startTime = Get-Date
    Write-Host "开始生成... $(Get-Date -Format 'HH:mm:ss')" -ForegroundColor Cyan

    $response = Invoke-RestMethod -Uri "http://localhost:8081/v1/collaborative/writing" `
        -Method Post -ContentType "application/json" -Body $body -TimeoutSec 300

    $elapsed = (Get-Date) - $startTime

    Write-Host ""
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "🎉🎉🎉 成功！文章生成完成！" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "📊 统计信息:" -ForegroundColor Yellow
    Write-Host "  ⏱️  总耗时: $([math]::Round($elapsed.TotalSeconds, 1)) 秒"
    Write-Host "  📝 字数: $($response.word_count) 字"
    Write-Host "  ⭐ 质量得分: $($response.review_score) 分"
    Write-Host "  🔄 迭代次数: $($response.iterations) 次"
    Write-Host ""

    # 保存文章
    $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
    $filename = "article_redis_$timestamp.md"
    $response.content | Out-File -FilePath $filename -Encoding UTF8

    Write-Host "💾 文章已保存: $filename" -ForegroundColor Green
    Write-Host ""
    Write-Host "📄 预览前600字:" -ForegroundColor Cyan
    Write-Host "========================================" -ForegroundColor Gray
    $preview = $response.content.Substring(0, [Math]::Min(600, $response.content.Length))
    Write-Host $preview
    Write-Host "..."
    Write-Host "========================================" -ForegroundColor Gray
    Write-Host ""
    Write-Host "🎊 大功告成！多Agent系统完全可用！" -ForegroundColor Green
    Write-Host ""
    Write-Host "下一步:" -ForegroundColor Yellow
    Write-Host "  1. 查看完整文章: notepad $filename"
    Write-Host "  2. 设置定时任务: .\scripts\setup_schedule.ps1"
    Write-Host "  3. 开始自动发布！"
    Write-Host ""

    # 询问是否打开文章
    $open = Read-Host "是否打开文章查看？(Y/N)"
    if ($open -eq "Y" -or $open -eq "y") {
        notepad $filename
    }

} catch {
    Write-Host ""
    Write-Host "❌ 生成失败: $_" -ForegroundColor Red
    Write-Host ""

    if ($_.ErrorDetails.Message) {
        Write-Host "详细错误:" -ForegroundColor Yellow
        Write-Host $_.ErrorDetails.Message
    }

    Write-Host ""
    Write-Host "请检查:" -ForegroundColor Yellow
    Write-Host "  1. API Key是否有效"
    Write-Host "  2. 网络连接是否正常"
    Write-Host "  3. DeepSeek账户是否有余额"
    Write-Host ""
    Write-Host "查看服务日志: Get-Content startup_error.log -Tail 20"
}

Write-Host ""
Write-Host "按任意键退出..."
pause | Out-Null
