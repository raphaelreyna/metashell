package metashell

import (
	"context"
	"errors"
	"os"
	"syscall"

	godaemon "github.com/sevlyar/go-daemon"
)

type client struct {
	socketPath           string
	postRunReportHandler PostRunReportHandlerFunc
	pidFileName          string
	logFileName          string
	workDir              string

	process *os.Process
}

func (c *client) run(ctx context.Context) error {
	var (
		err   error
		arg   string
		cntxt = godaemon.Context{
			PidFileName: c.pidFileName,
			PidFilePerm: 0644,
			LogFileName: c.logFileName,
			LogFilePerm: 0640,
			WorkDir:     c.workDir,
			Umask:       027,
		}
	)

	if len(os.Args) < 3 {
		return errors.New("no argument given")
	}
	arg = os.Args[2]

	c.process, err = cntxt.Search()
	if err != nil {
		return err
	}

	switch arg {
	case "quit-daemon":
		return c.quitDaemon(ctx)
	}

	return nil
}

func (c *client) quitDaemon(ctx context.Context) error {
	return c.process.Signal(syscall.SIGTERM)
}
