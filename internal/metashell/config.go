package metashell

import "path/filepath"

type Config struct {
	ShellPath  string
	socketPath string
}

func (c Config) NewMetaShell(rootDir string) *MetaShell {
	c.socketPath = filepath.Join(rootDir, "daemon.socket")
	if c.ShellPath == "" {
		c.ShellPath = "/bin/bash"
	}
	return &MetaShell{config: c}
}

func DefaultConfig() Config {
	return Config{ShellPath: "/bin/bash"}
}
