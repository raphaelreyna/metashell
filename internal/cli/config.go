package cli

import "path/filepath"

type Config struct {
	socketPath  string
	pidFileName string
	workDir     string
	logFileName string
}

func (c Config) NewClient(rootDir string) *Client {
	c.socketPath = filepath.Join(rootDir, "metashell.socket")
	c.workDir = rootDir
	c.logFileName = filepath.Join(rootDir, "logs", "daemon")
	c.pidFileName = filepath.Join(rootDir, "metashell-daemon.pid")

	return &Client{config: c}
}
