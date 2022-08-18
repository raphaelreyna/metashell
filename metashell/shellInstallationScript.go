package metashell

import (
	"context"
	"fmt"
	"os"
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
			METASHELL_CMD_KEY=$($EXEC -tty $TTY -cmd "$BASH_COMMAND")
	esac
}

__postRun() {
	$EXEC -cmdKey $METASHELL_CMD_KEY -exit-code $?
}
`
)

type installer struct {
	shellClientPath string
}

func (i *installer) run(ctx context.Context) error {
	_, err := fmt.Fprintf(os.Stdout, bashSource, i.shellClientPath)
	return err
}
