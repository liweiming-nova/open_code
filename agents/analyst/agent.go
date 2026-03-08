package analyst

import (
	"context"
	"log"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/liweiming-nova/open_code/agents/common"
	"github.com/liweiming-nova/open_code/chatmodel"
	"github.com/liweiming-nova/open_code/config"
	"github.com/liweiming-nova/open_code/enums"
	"github.com/liweiming-nova/open_code/pkg/logger"
	"github.com/liweiming-nova/open_code/tools/doc"
	"github.com/liweiming-nova/open_code/tools/excel"
	"github.com/liweiming-nova/open_code/tools/file"
	"github.com/liweiming-nova/open_code/tools/script/uv"
	"github.com/liweiming-nova/open_code/tools/todo"
	"github.com/liweiming-nova/open_code/variable"
	"go.uber.org/zap"
)

var analystAgent config.Agent

func NewAgent(ctx context.Context) adk.Agent {

	prompt, err := loadPrompt(ctx)
	if err != nil {
		logger.Fatal(ctx, "load prompt failed", zap.Error(err))
	}

	baseDir, ok := variable.GlobalRegistry.Get(enums.SystemVariableWorkspaceDir.String())
	if !ok {
		logger.Fatal(ctx, "无法获取工作目录系统变量")
	}
	analystSafeDir, err := baseDir.Getter(ctx)
	if err != nil {
		logger.Fatal(ctx, "无法获取工作目录", zap.Error(err))
	}

	// 初始化 Todo 工具
	todoToolsInstance, err := todo.NewTodoTools(analystSafeDir)
	if err != nil {
		log.Fatalf("初始化Todo工具失败: %v", err)
	}

	todoTools, err := todoToolsInstance.GetTools()
	if err != nil {
		log.Fatal("创建 Todo 工具失败:", err)
	}

	// 初始化 Excel 工具
	excelToolsInstance, err := excel.NewExcelTools(analystSafeDir)
	if err != nil {
		log.Fatalf("初始化Excel工具失败: %v", err)
	}
	excelTools, err := excelToolsInstance.GetTools()
	if err != nil {
		log.Fatal("创建 Excel 工具失败:", err)
	}

	// 初始化文件工具
	fileToolsInstance, err := file.NewFileTools(analystSafeDir)
	if err != nil {
		log.Fatalf("初始化文件工具失败: %v", err)
	}
	fileTools, err := fileToolsInstance.GetTools()
	if err != nil {
		log.Fatal("创建文件工具失败:", err)
	}

	// 初始化 uv 工具
	uvToolsInstance, err := uv.NewUVTools(analystSafeDir)
	if err != nil {
		log.Fatal("初始化uv工具失败:", err)
	}
	uvTools, err := uvToolsInstance.GetTools()
	if err != nil {
		log.Fatal("创建uv工具失败:", err)
	}

	// 初始化文档工具
	docTools, err := doc.GetTools()
	if err != nil {
		log.Fatal("创建文档工具失败:", err)
	}

	var toolList []tool.BaseTool
	toolList = append(toolList, todoTools...)
	toolList = append(toolList, excelTools...)
	toolList = append(toolList, fileTools...)
	toolList = append(toolList, uvTools...)
	toolList = append(toolList, docTools...)

	instruction := prompt[0].Content

	a, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "分析师",
		Description:   "分析师，擅长使用 Excel、Python 脚本和文档处理工具，从复杂数据和文档中提取有价值的信息并提供专业洞察。",
		Instruction:   instruction,
		Model:         chatmodel.NewChatModel(ctx),
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
		logger.Fatal(ctx, "create agent failed", zap.Error(err))
	}

	return a
}
