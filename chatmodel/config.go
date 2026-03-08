package chatmodel

import (
	"context"
	"os"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/liweiming-nova/open_code/pkg/logger"
	"go.uber.org/zap"
)

type ChatModelConfig struct {
	OpenAI *openai.ChatModelConfig
	Ark    *ark.ChatModelConfig
}

func (cfg *ChatModelConfig) NewChatModel(ctx context.Context) model.ToolCallingChatModel {
	if cfg == nil {
		return newDefaultChatModel(ctx)
	}

	if cfg.OpenAI != nil {
		return newOpenAIChatModel(ctx, cfg.OpenAI)
	}

	if cfg.Ark != nil {
		return newArkChatModel(ctx, cfg.Ark)
	}

	return newDefaultChatModel(ctx)
}

// NewChatModel 返回系统默认 chat model，供内置 agent 使用。
func NewChatModel(ctx context.Context) model.ToolCallingChatModel {
	return GetDefaultChatModel(ctx)
}

func GetDefaultChatModel(ctx context.Context) model.ToolCallingChatModel {
	return newDefaultChatModel(ctx)
}

// 默认模型改为 Ark。
func newDefaultChatModel(ctx context.Context) model.ToolCallingChatModel {
	cm, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey:  os.Getenv("OPEN_CODE_ARK_API_KEY"),
		BaseURL: os.Getenv("OPEN_CODE_ARK_BASE_URL"),
		Model:   os.Getenv("OPEN_CODE_ARK_MODEL"),
	})
	if err != nil {
		logger.Fatal(ctx, "[newDefaultChatModel]", zap.Error(err))
	}
	return cm
}

func newOpenAIChatModel(ctx context.Context, cfg *openai.ChatModelConfig) model.ToolCallingChatModel {
	cm, err := openai.NewChatModel(ctx, cfg)
	if err != nil {
		logger.Fatal(ctx, "[newOpenAIChatModel]", zap.Error(err))
	}

	return cm
}

func newArkChatModel(ctx context.Context, cfg *ark.ChatModelConfig) model.ToolCallingChatModel {
	cm, err := ark.NewChatModel(ctx, cfg)
	if err != nil {
		logger.Fatal(ctx, "[newArkChatModel]", zap.Error(err))
	}

	return cm
}
