package daemon

import (
	daemonstart "github.com/raphaelreyna/metashell/internal/commands/daemon/start"
	daemonstop "github.com/raphaelreyna/metashell/internal/commands/daemon/stop"
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
		Use:   "daemon",
		Short: "Run the metashell daemon",
		Long:  "Run the metashell daemon to manage plugins and commands.",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			log.SetLog(c.config.LogLevel, c.config.RootDir, "daemon")
			return nil
		},
	}

	c.command.AddCommand(
		c.subCommands(c.config)...,
	)

	return c.command
}

func (c *Cmd) subCommands(config *config.Config) []*cobra.Command {
	return []*cobra.Command{
		daemonstart.New(config).Cobra(),
		daemonstop.New(config).Cobra(),
	}
}
