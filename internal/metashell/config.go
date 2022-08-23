package metashell

import "path/filepath"

type Config struct {
	ShellPath  string
	PluginsDir string

	socketPath string
}

func (c Config) NewMetaShell(rootDir string) *MetaShell {
	c.socketPath = filepath.Join(rootDir, "daemon.socket")
	if c.ShellPath == "" {
		c.ShellPath = "/bin/bash"
	}
	if c.PluginsDir == "" {
		c.PluginsDir = filepath.Join(rootDir, "plugins", "metashell")
	}
	return &MetaShell{config: c}
}

func DefaultConfig(rootPath string) Config {
	return Config{
		ShellPath:  "/bin/bash",
		PluginsDir: filepath.Join(rootPath, "plugins", "metashell"),
	}
}
