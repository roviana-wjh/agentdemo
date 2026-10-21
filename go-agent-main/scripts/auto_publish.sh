#!/bin/bash
# 简化版定时发布脚本
# 每天自动发布1-2篇文章

# 配置
API_URL="http://localhost:8081/v1/collaborative/writing"
AUTHOR_ID=1

# 主题池（每周更新）
TOPICS=(
    "Docker容器化部署最佳实践"
    "Kubernetes服务网格实战"
    "微服务架构设计模式"
    "Redis缓存优化策略"
    "MySQL性能调优指南"
    "Vue3组件开发实战"
    "React Hooks深入解析"
    "Go并发编程技巧"
    "Python数据分析实战"
    "DevOps自动化流程"
)

# 随机选择1个主题
TOPIC=${TOPICS[$RANDOM % ${#TOPICS[@]}]}

echo "=========================================="
echo "自动发布任务开始"
echo "时间: $(date '+%Y-%m-%d %H:%M:%S')"
echo "主题: $TOPIC"
echo "=========================================="

# 调用写作API
RESPONSE=$(curl -s -X POST "$API_URL" \
  -H "Content-Type: application/json" \
  -d "{
    \"topic\": \"$TOPIC\",
    \"author_id\": $AUTHOR_ID,
    \"target_words\": 2000
  }")

# 检查是否成功
SUCCESS=$(echo "$RESPONSE" | jq -r '.success')

if [ "$SUCCESS" = "true" ]; then
    WORD_COUNT=$(echo "$RESPONSE" | jq -r '.word_count')
    SCORE=$(echo "$RESPONSE" | jq -r '.review_score')
    ITERATIONS=$(echo "$RESPONSE" | jq -r '.iterations')

    echo "✅ 文章生成成功"
    echo "   字数: $WORD_COUNT"
    echo "   得分: $SCORE"
    echo "   迭代: $ITERATIONS 次"

    # 保存内容到文件
    FILENAME="article_$(date '+%Y%m%d_%H%M%S').md"
    echo "$RESPONSE" | jq -r '.content' > "$FILENAME"
    echo "   已保存: $FILENAME"

    # TODO: 这里添加实际的发布逻辑
    # 例如：调用博客API发布，或者提交到Git仓库

else
    echo "❌ 文章生成失败"
    echo "$RESPONSE" | jq '.'
fi

echo "=========================================="
