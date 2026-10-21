# 一键启动所有服务

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "启动完整博客系统" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# 1. 启动blog-api (8080)
Write-Host "1. 启动 blog-api..." -ForegroundColor Yellow
cd C:\Users\lenovo\Documents\vibe\blog-api

$blogApi = Get-Process | Where-Object { $_.MainWindowTitle -match "blog-api" }
if (-not $blogApi) {
    Start-Process -FilePath ".\blog-api.exe" -WindowStyle Normal
    Write-Host "   blog-api 已启动" -ForegroundColor Green
} else {
    Write-Host "   blog-api 已在运行" -ForegroundColor Gray
}

Start-Sleep -Seconds 3

# 2. 启动go-agent-main (8081)
Write-Host ""
Write-Host "2. 启动 go-agent (AI写作)..." -ForegroundColor Yellow
cd C:\Users\lenovo\Documents\vibe\go-agent-main

$goAgent = Get-Process -Name "go-agent*" -ErrorAction SilentlyContinue
if (-not $goAgent) {
    Start-Process -FilePath ".\go-agent-with-scheduler.exe" -WindowStyle Normal
    Write-Host "   go-agent 已启动" -ForegroundColor Green
} else {
    Write-Host "   go-agent 已在运行" -ForegroundColor Gray
}

Start-Sleep -Seconds 3

# 3. 提示启动前端
Write-Host ""
Write-Host "3. 启动 blog-web (前端)..." -ForegroundColor Yellow
Write-Host "   需要手动执行:" -ForegroundColor Cyan
Write-Host "   cd C:\Users\lenovo\Documents\vibe\blog-web" -ForegroundColor White
Write-Host "   npm run dev" -ForegroundColor White

Write-Host ""
Write-Host "等待服务初始化 (10秒)..." -ForegroundColor Cyan
Start-Sleep -Seconds 10

# 验证服务
Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "服务状态检查" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""

try {
    Invoke-WebRequest -Uri "http://localhost:8080/" -UseBasicParsing -TimeoutSec 3 | Out-Null
    Write-Host "✅ blog-api:8080 运行正常" -ForegroundColor Green
} catch {
    Write-Host "❌ blog-api:8080 未响应" -ForegroundColor Red
}

try {
    Invoke-WebRequest -Uri "http://localhost:8081/health" -UseBasicParsing -TimeoutSec 3 | Out-Null
    Write-Host "✅ go-agent:8081 运行正常" -ForegroundColor Green
} catch {
    Write-Host "❌ go-agent:8081 未响应" -ForegroundColor Red
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "访问地址" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "博客后端:   http://localhost:8080" -ForegroundColor White
Write-Host "AI写作:     http://localhost:8081" -ForegroundColor White
Write-Host "博客前端:   http://localhost:3000 (启动后)" -ForegroundColor White
Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "后端服务已全部启动！" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
