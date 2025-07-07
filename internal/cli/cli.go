package cli

import (
	"context"
	"errors"
	"os"
	"syscall"

	"github.com/raphaelreyna/metashell/internal/daemon"
	godaemon "github.com/sevlyar/go-daemon"
)

type Client struct {
	config *daemon.Config

	process *os.Process
}

func NewClient(config *daemon.Config) *Client {
	return &Client{
		config: config,
	}
}

func (c *Client) Connect(ctx context.Context) error {
	var (
		err   error
		cntxt = godaemon.Context{
			PidFileName: c.config.PidFileName,
			PidFilePerm: 0644,
			WorkDir:     c.config.WorkDir,
			Umask:       027,
		}
	)

	if len(os.Args) < 3 {
		return errors.New("no argument given")
	}

	c.process, err = cntxt.Search()
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) QuitDaemon(ctx context.Context) error {
	return c.process.Signal(syscall.SIGTERM)
}
