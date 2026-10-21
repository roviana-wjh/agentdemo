# 设置Windows定时任务
# 用途：配置每天自动运行发布脚本

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "配置定时任务" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# 检查管理员权限
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

if (-not $isAdmin) {
    Write-Host "❌ 需要管理员权限！" -ForegroundColor Red
    Write-Host ""
    Write-Host "请右键点击PowerShell，选择'以管理员身份运行'" -ForegroundColor Yellow
    Write-Host ""
    pause
    exit 1
}

# 配置参数
$taskName = "BlogAutoPublish"
$scriptPath = "C:\Users\lenovo\Documents\vibe\go-agent-main\scripts\auto_publish.ps1"
$time = "09:00"  # 每天9点执行

Write-Host "任务名称: $taskName"
Write-Host "脚本路径: $scriptPath"
Write-Host "执行时间: 每天 $time"
Write-Host ""

# 检查脚本是否存在
if (-not (Test-Path $scriptPath)) {
    Write-Host "❌ 脚本文件不存在: $scriptPath" -ForegroundColor Red
    pause
    exit 1
}

# 删除已存在的任务（如果有）
$existingTask = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
if ($existingTask) {
    Write-Host "⚠️  发现已存在的任务，删除中..." -ForegroundColor Yellow
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false
    Write-Host "✅ 旧任务已删除" -ForegroundColor Green
    Write-Host ""
}

# 创建动作
Write-Host "创建任务..." -ForegroundColor Yellow
$action = New-ScheduledTaskAction -Execute "PowerShell.exe" `
    -Argument "-ExecutionPolicy Bypass -WindowStyle Hidden -File `"$scriptPath`""

# 创建触发器（每天9点）
$trigger = New-ScheduledTaskTrigger -Daily -At $time

# 创建设置
$settings = New-ScheduledTaskSettingsSet `
    -AllowStartIfOnBatteries `
    -DontStopIfGoingOnBatteries `
    -StartWhenAvailable `
    -RunOnlyIfNetworkAvailable

# 创建主体（使用当前用户）
$principal = New-ScheduledTaskPrincipal -UserId $env:USERNAME -RunLevel Highest

# 注册任务
Register-ScheduledTask -TaskName $taskName `
    -Action $action `
    -Trigger $trigger `
    -Settings $settings `
    -Principal $principal `
    -Description "每天自动生成并发布一篇技术博客文章" | Out-Null

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "✅ 定时任务设置成功！" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "任务详情:" -ForegroundColor Cyan
Write-Host "  任务名称: $taskName"
Write-Host "  执行时间: 每天 $time"
Write-Host "  执行脚本: $scriptPath"
Write-Host "  任务状态: 已启用"
Write-Host ""
Write-Host "📅 下次运行时间:" -ForegroundColor Yellow

# 获取下次运行时间
$task = Get-ScheduledTask -TaskName $taskName
$nextRun = (Get-ScheduledTaskInfo -TaskName $taskName).NextRunTime
Write-Host "  $nextRun"
Write-Host ""

Write-Host "管理命令:" -ForegroundColor Cyan
Write-Host "  查看任务: Get-ScheduledTask -TaskName '$taskName'"
Write-Host "  启动任务: Start-ScheduledTask -TaskName '$taskName'"
Write-Host "  禁用任务: Disable-ScheduledTask -TaskName '$taskName'"
Write-Host "  删除任务: Unregister-ScheduledTask -TaskName '$taskName'"
Write-Host ""

Write-Host "🎉 大功告成！从明天开始，系统会每天自动发布文章！" -ForegroundColor Green
Write-Host ""

# 询问是否立即测试
$test = Read-Host "是否立即测试运行一次？(Y/N)"
if ($test -eq "Y" -or $test -eq "y") {
    Write-Host ""
    Write-Host "开始测试..." -ForegroundColor Yellow
    Write-Host ""
    Start-ScheduledTask -TaskName $taskName
    Write-Host "✅ 任务已启动，请查看 articles 目录" -ForegroundColor Green
}

Write-Host ""
Write-Host "按任意键退出..."
pause | Out-Null
