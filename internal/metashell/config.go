package metashell

import "path/filepath"

type Config struct {
	ShellPath  string
	PluginsDir string

	socketPath string
}

func (c Config) NewMetaShell(rootDir string) *MetaShell {
	return &MetaShell{config: c}
}

func (c *Config) SetDefaults(rootDir string) {
	if c.ShellPath == "" {
		c.ShellPath = "/bin/bash"
	}
	if c.PluginsDir == "" {
		c.PluginsDir = filepath.Join(rootDir, "plugins", "metashell")
	}
	c.socketPath = filepath.Join(rootDir, "daemon.socket")
}
