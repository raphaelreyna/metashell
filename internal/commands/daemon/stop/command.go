package daemonstop

import (
	"github.com/raphaelreyna/metashell/internal/cli"
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
		Use:   "stop",
		Short: "Stop the metashell daemon",
		Long:  "Stop the metashell daemon that manages plugins and commands.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			daemonClient := cli.NewClient(&c.config.Daemon)
			err := daemonClient.Connect(ctx)
			if err != nil {
				return err
			}
			return daemonClient.QuitDaemon(ctx)
		},
	}

	return c.command
}
