package enums

type SystemVariable string

const (
	SystemVariableWorkspaceDir SystemVariable = "workspace_dir"
	SystemVariableCurrentTime  SystemVariable = "current_time"
	SystemVariableOs           SystemVariable = "os"
)

func (s SystemVariable) String() string {
	return string(s)
}

// CommandResult 命令执行结果
type CommandResult int

const (
	ResultContinue CommandResult = iota // 继续处理
	ResultHandled                       // 命令已处理，继续循环
	ResultExit                          // 退出程序
	ResultNotFound                      // 命令未找到
)

// CommandName 命令名半枚举
type CommandName string

const (
	CommandUnknown      CommandName = ""
	CommandHelp         CommandName = "help"
	CommandQuit         CommandName = "quit"
	CommandListAgents   CommandName = "list_agents"
	CommandListSchedule CommandName = "list_schedule"
	CommandSwitchMode   CommandName = "switch_work_mode"
)

// CommandAliasMap CLI 输入字符串到命令名的别名映射
var CommandAliasMap = map[string]CommandName{
	"help":             CommandHelp,
	"q":                CommandQuit,
	"quit":             CommandQuit,
	"":                 CommandQuit,
	"list_agents":      CommandListAgents,
	"list_schedule":    CommandListSchedule,
	"switch_work_mode": CommandSwitchMode,
}
