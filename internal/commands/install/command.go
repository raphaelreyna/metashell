package install

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/raphaelreyna/metashell/internal/config"
	"github.com/raphaelreyna/metashell/internal/log"
	"github.com/spf13/cobra"
)

const (
	bashSource = `
PROMPT_COMMAND=__postRun
EXEC="%s shellclient"
export METASHELL_CMD_KEY=INIT

trap __preRun DEBUG

__preRun() {
	case "$BASH_COMMAND" in
		$PROMPT_COMMAND)
			;;
		*)
			TTY=$(tty)
			METASHELL_CMD_KEY=$($EXEC --tty $TTY --cmd "$BASH_COMMAND")
	esac
}

__postRun() {
	$EXEC --cmdKey $METASHELL_CMD_KEY --exit-code $?
}
`
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
		Use:   "install",
		Short: "Install a plugin",
		Long:  "Install a plugin to the metashell daemon.",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			log.SetLog(c.config.LogLevel, c.config.RootDir, "installer")
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := os.Args[0]
			path, _ = filepath.EvalSymlinks(path)
			_, err := fmt.Fprintf(os.Stdout, bashSource, path)
			return err
		},
	}

	return c.command
}
