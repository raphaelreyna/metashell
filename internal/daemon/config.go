package daemon

import (
	"path/filepath"

	. "github.com/raphaelreyna/metashell/internal/log"
)

type Config struct {
	PluginsDir string

	socketPath  string
	pidFileName string
	workDir     string
	logFileName string
}

func (c Config) NewDaemon(rootDir string) *Daemon {
	c.socketPath = filepath.Join(rootDir, "daemon.socket")
	c.workDir = rootDir
	c.logFileName = Log.OutFilePath()
	c.pidFileName = filepath.Join(rootDir, "daemon.pid")

	if c.PluginsDir == "" {
		c.PluginsDir = filepath.Join(rootDir, "plugins", "daemon")
	}

	return &Daemon{config: c}
}

func DefaultConfig(rootPath string) Config {
	return Config{
		PluginsDir: filepath.Join(rootPath, "plugins", "daemon"),
	}
}
