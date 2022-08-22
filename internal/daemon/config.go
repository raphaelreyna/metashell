package daemon

import "path/filepath"

type Config struct {
	PluginsDir string

	socketPath  string
	pidFileName string
	workDir     string
	logFileName string
}

func (c Config) NewDaemon(rootDir string) *Daemon {
	c.socketPath = filepath.Join(rootDir, "metashell.socket")
	c.workDir = rootDir
	c.logFileName = filepath.Join(rootDir, "logs", "daemon")
	c.pidFileName = filepath.Join(rootDir, "metashell-daemon.pid")

	if c.PluginsDir == "" {
		c.PluginsDir = filepath.Join(rootDir, "plugins", "daemon")
	}

	return &Daemon{config: c}
}
