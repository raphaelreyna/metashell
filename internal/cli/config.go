package cli

import (
	"path/filepath"

	"github.com/raphaelreyna/metashell/internal/log"
)

type Config struct {
	socketPath  string
	pidFileName string
	workDir     string
	logFileName string
}

func (c Config) NewClient(rootDir string) *Client {
	c.socketPath = filepath.Join(rootDir, "daemon.socket")
	c.workDir = rootDir
	c.logFileName = log.OutFilePath()
	c.pidFileName = filepath.Join(rootDir, "daemon.pid")

	return &Client{config: c}
}
