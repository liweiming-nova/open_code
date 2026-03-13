package knowledge

import (
	"context"
	"fmt"

	"github.com/liweiming-nova/open_code/config"
	knowledgeplugin "github.com/liweiming-nova/open_code/knowledge"
	"github.com/pterm/pterm"
	ucli "github.com/urfave/cli/v3"
)

func KnowledgeCommand() *ucli.Command {
	return &ucli.Command{
		Name:  "knowledge",
		Usage: "knowledge plugin commands",
		Commands: []*ucli.Command{
			{
				Name:  "vectorize",
				Usage: "vectorize one markdown file",
				Flags: []ucli.Flag{
					&ucli.StringFlag{
						Name:     "file",
						Aliases:  []string{"f"},
						Required: true,
						Usage:    "markdown file path (absolute or relative to plugins.knowledge.dir)",
					},
				},
				Action: func(ctx context.Context, cmd *ucli.Command) error {
					cfg, err := config.Get()
					if err != nil {
						return err
					}
					plugin, err := knowledgeplugin.NewPlugin(cfg.Plugins.Knowledge)
					if err != nil {
						return err
					}

					record, err := plugin.VectorizeFile(ctx, cmd.String("file"))
					if err != nil {
						return err
					}

					pterm.Success.Printfln(
						"vectorized file: %s chunks=%d dim=%d",
						record.FilePath,
						record.ChunkCount,
						record.VectorDim,
					)
					return nil
				},
			},
			{
				Name:  "list",
				Usage: "list vectorized markdown files from local json state",
				Action: func(ctx context.Context, cmd *ucli.Command) error {
					cfg, err := config.Get()
					if err != nil {
						return err
					}
					plugin, err := knowledgeplugin.NewPlugin(cfg.Plugins.Knowledge)
					if err != nil {
						return err
					}

					files := plugin.ListFiles()
					if len(files) == 0 {
						pterm.Info.Println("no vectorized files found")
						return nil
					}

					for _, f := range files {
						pterm.Println(fmt.Sprintf(
							"- %s | chunks=%d | dim=%d | at=%s",
							f.FilePath, f.ChunkCount, f.VectorDim, f.VectorizedAt.Format("2006-01-02 15:04:05"),
						))
					}
					return nil
				},
			},
		},
	}
}
