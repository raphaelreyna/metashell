package daemon

import (
	"path/filepath"
)

type Config struct {
	SocketPath    string         `yaml:"socket_path"`
	PidFileName   string         `yaml:"pid_file_name"`
	WorkDir       string         `yaml:"work_dir"`
	PluginsDir    string         `yaml:"plugins_dir"`
	PluginConfigs map[string]any `yaml:"plugin_configs"`
}

func (c *Config) SetDefaults(rootDir string) {
	if c.SocketPath == "" {
		c.SocketPath = filepath.Join(rootDir, "daemon.socket")
	}
	if c.PidFileName == "" {
		c.PidFileName = filepath.Join(rootDir, "daemon.pid")
	}
	if c.WorkDir == "" {
		c.WorkDir = rootDir
	}
	if c.PluginsDir == "" {
		c.PluginsDir = filepath.Join(rootDir, "plugins", "daemon")
	}
	if c.PluginConfigs == nil {
		c.PluginConfigs = make(map[string]any)
	}
}

func (c *Config) NewDaemon(rootDir string) *Daemon {
	return &Daemon{config: *c}
}
