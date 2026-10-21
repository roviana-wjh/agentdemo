@echo off
chcp 65001 >nul
echo ============================================================
echo 🧪 意图识别自动化测试
echo ============================================================
echo.

REM 检查服务是否运行
echo 📡 检查服务状态...
curl -s http://localhost:8081/health >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ 服务未启动！
    echo.
    echo 请先启动服务：
    echo   cd go-agent-main
    echo   go-agent.exe
    echo.
    pause
    exit /b 1
)
echo ✅ 服务正常运行
echo.

REM 运行测试
echo 🚀 开始执行测试...
echo.
cd /d "%~dp0"
go run test\intent_test.go

echo.
echo ============================================================
echo 📊 测试完成
echo ============================================================
echo.
echo 查看详细报告: test\intent_test_report.json
echo 查看测试说明: test\README.md
echo.
pause
