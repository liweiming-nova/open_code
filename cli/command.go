package cli

import (
	"strings"

	"github.com/liweiming-nova/open_code/enums"
	"github.com/pterm/pterm"
)

// CommandHandler 命令处理器a
type CommandHandler struct {
	modeSwitcher    ModeSwitcher
	commandHandlers map[enums.CommandName]func(rawInput string) enums.CommandResult
}

// ModeSwitcher 工作模式切换器接口
type ModeSwitcher interface {
	SwitchMode() (newMode string, err error)
}

// NewCommandHandler 创建命令处理器
func NewCommandHandler(modeSwitcher ModeSwitcher) *CommandHandler {
	h := &CommandHandler{
		modeSwitcher: modeSwitcher,
	}
	h.commandHandlers = h.buildCommandHandlers()
	return h
}

// Handle 处理命令，返回命令执行结果
func (h *CommandHandler) Handle(input string) enums.CommandResult {
	cmd := h.resolveCommand(input)
	handler, ok := h.commandHandlers[cmd]
	if !ok {
		return enums.ResultNotFound
	}
	return handler(input)
}

func (h *CommandHandler) buildCommandHandlers() map[enums.CommandName]func(rawInput string) enums.CommandResult {
	return map[enums.CommandName]func(rawInput string) enums.CommandResult{
		enums.CommandHelp:         h.handleHelp,
		enums.CommandQuit:         h.handleQuit,
		enums.CommandListAgents:   h.handleListAgents,
		enums.CommandListSchedule: h.handleListSchedule,
		enums.CommandSwitchMode:   h.handleSwitchMode,
	}
}

func (h *CommandHandler) resolveCommand(input string) enums.CommandName {
	key := strings.ToLower(strings.TrimSpace(input))
	if cmd, ok := enums.CommandAliasMap[key]; ok {
		return cmd
	}
	return enums.CommandUnknown
}

// 以下为预留命令执行逻辑。
func (h *CommandHandler) handleHelp(_ string) enums.CommandResult {
	pterm.Println("=== Open Code 命令帮助 ===")
	pterm.Println()
	pterm.Println("基本操作:")
	pterm.Println("  help                             显示此帮助信息")
	pterm.Println("  q, quit, Enter                   退出程序")
	pterm.Println()
	pterm.Println("智能体切换:")
	pterm.Println("  list_agents                     列出所有可用的智能体")
	pterm.Println("  @智能体名 [查询内容]              切换到指定智能体并可选执行查询")
	pterm.Println()
	pterm.Println("文件引用:")
	pterm.Println("  #文件路径                        快速引用工作目录中的文件或文件夹")
	pterm.Println()
	pterm.Println("聊天历史管理:")
	pterm.Println("  list_chat_history                        列出所有可用的聊天历史会话")
	pterm.Println("  load_chat_history <session_id>            加载指定的聊天历史会话")
	pterm.Println("  save_chat_history                        保存聊天历史到当前会话文件")
	pterm.Println("  clear_chat_history              清空当前聊天历史")
	pterm.Println("  save_chat_history_to_markdown   导出聊天历史为 Markdown 文件")
	pterm.Println("  save_chat_history_to_html       导出聊天历史为 HTML 文件")
	pterm.Println()
	pterm.Println("任务管理:")
	pterm.Println("  clear_todo                      清空所有待办事项")
	pterm.Println("  list_schedule                   列出所有定时任务")
	pterm.Println("  cancel_schedule <id>            取消指定的定时任务")
	pterm.Println()
	pterm.Println("模式切换:")
	pterm.Println("  switch_work_mode               切换当前工作模式")
	pterm.Println()
	pterm.Println("其他操作:")
	pterm.Println("  直接输入问题                     与智能体团队对话")
	return enums.ResultHandled
}

func (h *CommandHandler) handleQuit(_ string) enums.CommandResult {
	pterm.Info.Println("谢谢使用，再见！")
	return enums.ResultExit
}

func (h *CommandHandler) handleListAgents(_ string) enums.CommandResult {
	pterm.Println()
	pterm.Println("=== 可用智能体列表 ===")
	pterm.Println()
	pterm.Println("使用方式: 输入 #智能体名 [查询内容] 即可切换到该智能体")
	pterm.Println()

	for _, agent := range agents.GetRegistry() {
		pterm.Printf("  #%s\n", agent.Name)
		pterm.Printf("    描述: %s\n", agent.Description)
		pterm.Println()
	}

	pterm.Println("提示: 输入 # 后会自动提示可用的智能体")
	pterm.Println()
	return enums.ResultHandled
}

func (h *CommandHandler) handleListSchedule(_ string) enums.CommandResult {
	return enums.ResultHandled
}

func (h *CommandHandler) handleSwitchMode(_ string) enums.CommandResult {
	return enums.ResultHandled
}
