# Skill Learning Observability Stack

技能学习系统的完整可观测性解决方案。

## 快速开始

```bash
# 1. 启动观测栈
docker-compose up -d

# 2. 启动 Go Agent（输出日志到文件）
cd ../..
mkdir -p logs
./go-agent 2>&1 | tee logs/app.log

# 3. 访问 Grafana
# 浏览器打开 http://localhost:3000
# 查看 "Skill Learning System" 仪表盘
```

## 目录结构

```
observability/
├── docker-compose.yml          # 服务编排
├── loki-config.yaml            # Loki 配置
├── promtail-config.yaml        # Promtail 日志采集配置
├── grafana/
│   ├── provisioning/
│   │   ├── datasources/
│   │   │   └── loki.yaml       # Loki 数据源自动配置
│   │   └── dashboards/
│   │       └── dashboard.yaml  # Dashboard 自动加载配置
│   └── dashboards/
│       └── skill-learning.json # 技能学习仪表盘
└── README.md
```

## 服务端口

- Grafana: `3000`
- Loki: `3100`
- Promtail: `9080`

## 日志格式

应用输出的结构化日志示例：

```json
{
  "timestamp": "2024-01-01T12:00:00Z",
  "trace_id": "a1b2c3d4e5f6...",
  "event_type": "extraction",
  "status": "success",
  "skill_name": "query-hot-articles",
  "query": "查热门文章",
  "tool_calls": 4,
  "duration_ms": 1230,
  "metadata": {
    "tools_used": "search_article,get_hot_articles"
  }
}
```

## 支持的事件类型

| event_type | 说明 | status 枚举 |
|------------|------|-------------|
| extraction | 技能抽取 | started, success, failed, skipped |
| save | 技能保存 | success, failed |
| usage | 技能使用 | success, failed |
| cleanup | 定时清理 | success |

## 常用 LogQL 查询

```logql
# 查看所有技能事件
{job="go-agent"} |= `[SKILL_EVENT]` | json

# 成功学习的技能
{job="go-agent"} |= `[SKILL_EVENT]` | json | event_type="save" | status="success"

# 按 trace_id 追踪完整流程
{job="go-agent"} |= `trace_id="your-id"` | json

# 统计每小时学习数量
sum(count_over_time({job="go-agent"} |= `[SKILL_EVENT]` | json | event_type="save" | status="success" [1h]))
```

## 故障排查

### Grafana 看不到数据

```bash
# 1. 检查日志是否生成
tail -f ../../logs/app.log | grep SKILL_EVENT

# 2. 检查 Promtail 是否运行
docker-compose logs promtail

# 3. 测试 Loki 查询
curl -G "http://localhost:3100/loki/api/v1/query" \
  --data-urlencode 'query={job="go-agent"}' | jq
```

### 日志路径问题

确保 docker-compose.yml 中的卷映射正确：

```yaml
promtail:
  volumes:
    - ../../logs:/app/logs:ro  # 指向 Go Agent 的日志目录
```

## 停止服务

```bash
docker-compose down        # 停止并删除容器
docker-compose down -v     # 同时删除数据卷（清空所有数据）
```

## 更多文档

详细配置和高级用法请参考：
- [../../docs/observability_setup.md](../../docs/observability_setup.md) - 完整部署指南
- [../../docs/skill_tracing.md](../../docs/skill_tracing.md) - 追踪说明（已过时，请参考 observability_setup.md）
