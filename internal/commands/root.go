package commands

import (
	"context"

	configcmd "github.com/raphaelreyna/metashell/internal/commands/config"
	"github.com/raphaelreyna/metashell/internal/commands/daemon"
	"github.com/raphaelreyna/metashell/internal/commands/install"
	"github.com/raphaelreyna/metashell/internal/commands/metashell"
	plugin "github.com/raphaelreyna/metashell/internal/commands/plugin"
	"github.com/raphaelreyna/metashell/internal/commands/shellclient"
	"github.com/raphaelreyna/metashell/internal/config"
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
	}

	c.command.AddCommand(
		c.subCommands(c.config)...,
	)

	return c.command
}

func (c *Cmd) subCommands(config *config.Config) []*cobra.Command {
	return []*cobra.Command{
		configcmd.New(config).Cobra(),
		daemon.New(config).Cobra(),
		install.New(config).Cobra(),
		metashell.New(config).Cobra(),
		shellclient.New(config).Cobra(),
		plugin.New(config).Cobra(),
	}
}

func (c *Cmd) Run(ctx context.Context) error {
	cmd := c.Cobra()
	if err := cmd.ExecuteContext(ctx); err != nil {
		return err
	}
	return nil
}
