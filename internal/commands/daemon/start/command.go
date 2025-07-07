package daemonstart

import (
	"github.com/raphaelreyna/metashell/internal/config"
	"github.com/spf13/cobra"
)

type Cmd struct {
	command *cobra.Command
	config  *config.Config
}

func New(config *config.Config) *Cmd {
	cmd := &Cmd{
		config: config,
	}

	return cmd
}

func (c *Cmd) Cobra() *cobra.Command {
	if c.command != nil {
		return c.command
	}

	c.command = &cobra.Command{
		Use:   "start",
		Short: "Start the metashell daemon",
		Long:  "Start the metashell daemon to manage plugins and commands.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			d := c.config.Daemon.NewDaemon(c.config.RootDir)
			return d.Run(ctx)
		},
	}

	return c.command
}
