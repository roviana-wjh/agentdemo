package chat_model

import (
	"context"
	"go-agent/config"

	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/components/model"
)

func initQwen() {
	registerChatModel("qwen", func(ctx context.Context) (model.BaseChatModel, error) {
		baseURL := config.Cfg.QwenConf.BaseUrl
		if baseURL == "" {
			baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		}
		return qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
			BaseURL: baseURL,
			APIKey:  config.Cfg.QwenConf.QwenKey,
			Model:   config.Cfg.QwenConf.QwenChatModel,
		})
	})
}
