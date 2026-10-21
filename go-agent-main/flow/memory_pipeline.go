package flow

import (
	"context"
	"log"

	"go-agent/tool/memory"

	"github.com/cloudwego/eino/schema"
)

// 长期记忆 pipeline 的包级依赖。由 main.go 在初始化时通过 SetMemoryPipeline 注入。
// 未注入时（MySQL 不可用或未启用），extractFactsFromDropped 静默跳过，系统退回纯 session 模式。
var (
	factExtractor   *memory.FactExtractor
	userMemoryStore memory.UserMemoryStore
)

// SetMemoryPipeline 注入长期记忆依赖。传 nil 表示禁用长期记忆。
func SetMemoryPipeline(extractor *memory.FactExtractor, store memory.UserMemoryStore) {
	factExtractor = extractor
	userMemoryStore = store
}

// memoryPipelineEnabled 报告长期记忆是否已启用。
func memoryPipelineEnabled() bool {
	return factExtractor != nil && userMemoryStore != nil
}

// extractFactsFromDropped 对本次压缩被丢弃的旧消息抽取长期记忆并写库。
// dropped 为空（未发生压缩）时直接返回。调用方应在后台 goroutine 中执行。
func extractFactsFromDropped(ctx context.Context, sessionID string, dropped []*schema.Message) {
	if !memoryPipelineEnabled() || len(dropped) == 0 {
		return
	}

	userID := memory.ParseUserID(sessionID)
	facts, err := factExtractor.Extract(ctx, userID, dropped)
	if err != nil {
		log.Printf("warning: fact extract failed for user %s: %v", userID, err)
		return
	}
	if len(facts) == 0 {
		return
	}
	if err := userMemoryStore.Append(ctx, facts); err != nil {
		log.Printf("warning: user memory append failed for user %s: %v", userID, err)
		return
	}
	log.Printf("info: extracted %d long-term memories for user %s", len(facts), userID)
}

// loadUserFacts 加载某会话对应用户的长期记忆，供注入 prompt 使用。
// 未启用或出错时返回 nil，不影响主流程。
func loadUserFacts(ctx context.Context, sessionID string, limit int) []memory.UserMemory {
	if !memoryPipelineEnabled() {
		return nil
	}
	userID := memory.ParseUserID(sessionID)
	facts, err := userMemoryStore.List(ctx, userID, limit)
	if err != nil {
		log.Printf("warning: load user memory failed for user %s: %v", userID, err)
		return nil
	}
	return facts
}

// buildUserFactsMessage 把长期记忆渲染成一条 SystemMessage；无记忆时返回 nil。
func buildUserFactsMessage(facts []memory.UserMemory) *schema.Message {
	if len(facts) == 0 {
		return nil
	}
	text := "已知的用户长期信息（供参考，如与当前对话冲突以当前对话为准）：\n"
	for _, f := range facts {
		text += "- " + f.Content + "\n"
	}
	return schema.SystemMessage(text)
}
