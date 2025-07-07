package config

import (
	"github.com/raphaelreyna/metashell/internal/config"
	"github.com/raphaelreyna/metashell/internal/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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
		Use:   "config",
		Short: "Manage metashell configuration",
		Long:  "Manage the configuration of the metashell CLI and daemon.",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			log.SetLog(c.config.LogLevel, c.config.RootDir, "config")
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return yaml.NewEncoder(cmd.OutOrStdout()).Encode(c.config)
		},
	}

	return c.command
}
