package metashell

import "path/filepath"

type Config struct {
	ShellPath  string
	socketPath string
}

func (c Config) NewMetaShell(rootDir string) *MetaShell {
	c.socketPath = filepath.Join(rootDir, "metashell.socket")
	if c.ShellPath == "" {
		c.ShellPath = "/bin/sh"
	}
	return &MetaShell{config: c}
}
