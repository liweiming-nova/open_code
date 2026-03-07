package consts

type SystemVariable string

const (
	SystemVariableWorkspaceDir SystemVariable = "workspace_dir"
	SystemVariableCurrentTime  SystemVariable = "current_time"
	SystemVariableOs           SystemVariable = "os"
)

func (s SystemVariable) String() string {
	return string(s)
}
