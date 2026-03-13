package cmd

import (
	"github.com/liweiming-nova/open_code/cmd/cinit"
	"github.com/liweiming-nova/open_code/cmd/knowledge"
	"github.com/liweiming-nova/open_code/cmd/update"
	"github.com/liweiming-nova/open_code/version"

	ucli "github.com/urfave/cli/v3"
)

// Root 创建根命令
func Root() *ucli.Command {
	return &ucli.Command{
		Name:    "Open Code",
		Usage:   "智能AI助手",
		Version: version.Get().String(),
		Commands: []*ucli.Command{
			cinit.InitCommand(),
			knowledge.KnowledgeCommand(),
			update.UpdateCommand(),
		},
		Flags: []ucli.Flag{},
	}
}
