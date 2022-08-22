package cli

import (
	"context"
	"errors"
	"os"
	"syscall"

	godaemon "github.com/sevlyar/go-daemon"
)

type Client struct {
	config Config

	process *os.Process
}

func (c *Client) Run(ctx context.Context) error {
	var (
		err   error
		arg   string
		cntxt = godaemon.Context{
			PidFileName: c.config.pidFileName,
			PidFilePerm: 0644,
			LogFileName: c.config.logFileName,
			LogFilePerm: 0640,
			WorkDir:     c.config.workDir,
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

func (c *Client) quitDaemon(ctx context.Context) error {
	return c.process.Signal(syscall.SIGTERM)
}
