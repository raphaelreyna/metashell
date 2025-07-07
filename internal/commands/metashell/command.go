package metashell

import (
	"github.com/raphaelreyna/metashell/internal/config"
	"github.com/raphaelreyna/metashell/internal/log"
	"github.com/spf13/cobra"
)

type Cmd struct {
	command *cobra.Command
	config  *config.Config
}

func New(config *config.Config) *Cmd {
	return &Cmd{
		config: config,
	}
}

func (c *Cmd) Cobra() *cobra.Command {
	if c.command != nil {
		return c.command
	}

	c.command = &cobra.Command{
		Use:   "metashell",
		Short: "Run the metashell CLI",
		Long:  "Run the metashell CLI to manage plugins and commands.",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			log.SetLog(c.config.LogLevel, c.config.RootDir, "metashell")
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			m := c.config.MetaShell.NewMetaShell(c.config.RootDir)
			return m.Run(ctx)
		},
	}

	return c.command
}
