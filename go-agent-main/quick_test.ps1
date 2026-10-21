# 快速测试脚本 - 编译完成后立即运行

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "快速测试：自动生成第一篇文章" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Step 1: 停止旧服务
Write-Host "Step 1: 停止旧服务..." -ForegroundColor Yellow
Get-Process -Name "go-agent*" -ErrorAction SilentlyContinue | Stop-Process -Force
Start-Sleep -Seconds 2

# Step 2: 启动新服务
Write-Host "Step 2: 启动新编译的服务..." -ForegroundColor Yellow
$process = Start-Process -FilePath ".\go-agent-phase123.exe" -WindowStyle Normal -PassThru
Write-Host "   进程ID: $($process.Id)"
Write-Host "   等待服务启动 (15秒)..."
Start-Sleep -Seconds 15

# Step 3: 检查服务状态
Write-Host "Step 3: 检查服务状态..." -ForegroundColor Yellow
try {
    $health = Invoke-WebRequest -Uri "http://localhost:8081/health" -UseBasicParsing -TimeoutSec 5
    Write-Host "   ✅ 服务运行正常 (状态码: $($health.StatusCode))" -ForegroundColor Green
} catch {
    Write-Host "   ❌ 服务未响应，再等10秒..." -ForegroundColor Red
    Start-Sleep -Seconds 10
    try {
        $health = Invoke-WebRequest -Uri "http://localhost:8081/health" -UseBasicParsing
        Write-Host "   ✅ 服务现在正常了" -ForegroundColor Green
    } catch {
        Write-Host "   ❌ 服务启动失败: $_" -ForegroundColor Red
        exit 1
    }
}

Write-Host ""

# Step 4: 测试生成文章
Write-Host "Step 4: 生成第一篇测试文章..." -ForegroundColor Yellow
Write-Host "   主题: Redis缓存优化实战指南"
Write-Host "   预计耗时: 2-3分钟"
Write-Host ""

$body = @{
    topic = "Redis缓存优化实战指南"
    author_id = 1
    target_words = 2000
} | ConvertTo-Json

$startTime = Get-Date

try {
    Write-Host "🚀 开始生成..." -ForegroundColor Cyan

    $response = Invoke-RestMethod -Uri "http://localhost:8081/v1/collaborative/writing" `
        -Method Post -ContentType "application/json" -Body $body -TimeoutSec 300

    $elapsed = (Get-Date) - $startTime

    Write-Host ""
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "✅ 测试成功！文章生成完毕！" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    Write-Host ""

    Write-Host "📊 结果统计:" -ForegroundColor Yellow
    Write-Host "   耗时: $([math]::Round($elapsed.TotalSeconds, 1)) 秒"
    Write-Host "   字数: $($response.word_count)"
    Write-Host "   质量得分: $($response.review_score)"
    Write-Host "   重写次数: $($response.iterations)"

    if ($response.rewrite_strategy) {
        Write-Host "   重写策略: $($response.rewrite_strategy)"
    }

    if ($response.stage_timings) {
        Write-Host ""
        Write-Host "⏱️  各阶段耗时:" -ForegroundColor Yellow
        $response.stage_timings.PSObject.Properties | ForEach-Object {
            Write-Host "   $($_.Name): $($_.Value)"
        }
    }

    # 保存文章
    $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
    $filename = "test_article_$timestamp.md"
    $response.content | Out-File -FilePath $filename -Encoding UTF8

    Write-Host ""
    Write-Host "💾 文章已保存到: $filename" -ForegroundColor Green

    # 显示文章预览
    Write-Host ""
    Write-Host "📝 文章预览（前800字）:" -ForegroundColor Cyan
    Write-Host "========================================" -ForegroundColor Cyan
    $preview = $response.content.Substring(0, [Math]::Min(800, $response.content.Length))
    Write-Host $preview
    if ($response.content.Length > 800) {
        Write-Host "..."
        Write-Host "(完整内容请查看文件: $filename)"
    }
    Write-Host "========================================" -ForegroundColor Cyan

    Write-Host ""
    Write-Host "🎉 恭喜！多Agent系统运行完美！" -ForegroundColor Green
    Write-Host ""
    Write-Host "下一步:" -ForegroundColor Yellow
    Write-Host "  1. 查看完整文章: notepad $filename"
    Write-Host "  2. 运行自动发布脚本: .\scripts\auto_publish.ps1"
    Write-Host "  3. 设置定时任务，实现每天自动发布"
    Write-Host ""

} catch {
    Write-Host ""
    Write-Host "❌ 生成失败: $_" -ForegroundColor Red
    Write-Host ""

    if ($_.Exception.Response) {
        $statusCode = $_.Exception.Response.StatusCode.value__
        Write-Host "   HTTP状态码: $statusCode"

        if ($statusCode -eq 404) {
            Write-Host "   原因: API端点未注册（可能服务启动时初始化失败）"
            Write-Host "   建议: 查看服务窗口的启动日志"
        }
    }
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
