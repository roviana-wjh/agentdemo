@echo off
chcp 65001 >nul
echo ============================================================
echo 🧪 Phase 2 增量重写测试
echo ============================================================
echo.

REM 检查可执行文件
if not exist "go-agent.exe" (
    echo ❌ go-agent.exe 不存在
    echo 请先编译: go build -o go-agent.exe
    pause
    exit /b 1
)

echo 📡 启动服务...
start "GoAgent" go-agent.exe

echo ⏳ 等待服务启动 (10秒)...
timeout /t 10 /nobreak >nul

REM 检查服务状态
curl -s http://localhost:8081/health >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ 服务启动失败
    pause
    exit /b 1
)

echo ✅ 服务已启动
echo.
echo ============================================================
echo 📝 测试用例1: 基础文章（容易触发增量重写）
echo ============================================================
echo.
echo 主题: Redis缓存策略
echo 预期: 2000字，可能需要1-2次重写
echo.

curl -X POST http://localhost:8081/v1/collaborative/writing ^
  -H "Content-Type: application/json" ^
  -d "{\"topic\":\"Redis缓存策略详解\",\"requirements\":\"面向中级开发者，包含实战案例\",\"author_id\":1,\"target_words\":2000}" ^
  -o test1_result.json

echo.
echo 结果已保存到: test1_result.json
echo.
type test1_result.json
echo.
echo.

echo ============================================================
echo 📝 测试用例2: 复杂文章（可能触发全文重写）
echo ============================================================
echo.
echo 主题: 深入Kubernetes调度器
echo 预期: 2500字，深度技术文章
echo.

curl -X POST http://localhost:8081/v1/collaborative/writing ^
  -H "Content-Type: application/json" ^
  -d "{\"topic\":\"深入理解Kubernetes调度器原理\",\"requirements\":\"面向有经验的开发者，深入源码\",\"author_id\":1,\"target_words\":2500,\"difficulty\":\"advanced\"}" ^
  -o test2_result.json

echo.
echo 结果已保存到: test2_result.json
echo.
type test2_result.json
echo.
echo.

echo ============================================================
echo 📊 测试完成
echo ============================================================
echo.
echo 查看结果:
echo   test1_result.json - 基础文章
echo   test2_result.json - 复杂文章
echo.
echo 关键指标:
echo   - iterations: 重写次数
echo   - rewrite_strategy: full/incremental
echo   - problematic_sections: 问题章节
echo   - review_score: 审核得分
echo.
pause
