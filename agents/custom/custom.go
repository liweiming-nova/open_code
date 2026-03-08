package custom

import (
	"context"
	"log"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/liweiming-nova/open_code/agents/common"
	"github.com/liweiming-nova/open_code/chatmodel"
	"github.com/liweiming-nova/open_code/config"
	"github.com/liweiming-nova/open_code/pkg/logger"
	"github.com/liweiming-nova/open_code/tools/mcp"
	"go.uber.org/zap"
)

func NewAgent(ctx context.Context, cfg config.Agent) adk.Agent {

	formattedPrompt, err := cfg.SystemPrompt.Format(ctx)
	if err != nil {
		logger.Fatal(ctx, "NewCusuomAgent", zap.Error(err))
	}

	var toolList []tool.BaseTool
	for _, toolName := range cfg.Tools {
		baseTools, err := mcp.GetToolsByName(toolName)
		if err != nil {
			log.Fatal(err)
		}
		toolList = append(toolList, baseTools...)
	}

	model := chatmodel.NewChatModel(ctx)
	if cfg.ChatModel.OpenAI != nil || cfg.ChatModel.Ark != nil {
		model = cfg.ChatModel.NewChatModel(ctx)
	}

	a, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          cfg.Name,
		Description:   cfg.Desc,
		Instruction:   formattedPrompt,
		Model:         model,
		MaxIterations: common.MaxIterations,
		ModelRetryConfig: &adk.ModelRetryConfig{
			MaxRetries:  common.MaxRetries,
			IsRetryAble: common.IsRetryAble,
		},
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: toolList,
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	return a
}
