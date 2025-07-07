package daemon

import (
	"path/filepath"

	"github.com/raphaelreyna/metashell/internal/log"
)

type Config struct {
	SocketPath    string         `yaml:"socket_path"`
	PidFileName   string         `yaml:"pid_file_name"`
	WorkDir       string         `yaml:"work_dir"`
	LogFileName   string         `yaml:"log_file_name"`
	PluginsDir    string         `yaml:"plugins_dir"`
	PluginConfigs map[string]any `yaml:"plugin_configs"`
}

func (c *Config) NewDaemon(rootDir string) *Daemon {
	if c.SocketPath == "" {
		c.SocketPath = filepath.Join(rootDir, "daemon.socket")
	}
	if c.PidFileName == "" {
		c.PidFileName = filepath.Join(rootDir, "daemon.pid")
	}
	if c.WorkDir == "" {
		c.WorkDir = rootDir
	}
	if c.LogFileName == "" {
		c.LogFileName = log.OutFilePath()
	}
	if c.PluginsDir == "" {
		c.PluginsDir = filepath.Join(rootDir, "plugins", "daemon")
	}
	if c.PluginConfigs == nil {
		c.PluginConfigs = make(map[string]any)
	}

	return NewDaemon(*c)
}

func DefaultConfig(rootDir string) Config {
	return Config{
		SocketPath:    filepath.Join(rootDir, "daemon.socket"),
		PidFileName:   filepath.Join(rootDir, "daemon.pid"),
		WorkDir:       rootDir,
		LogFileName:   log.OutFilePath(),
		PluginsDir:    filepath.Join(rootDir, "plugins", "daemon"),
		PluginConfigs: make(map[string]any),
	}
}

func NewDaemon(c Config) *Daemon {
	return &Daemon{config: c}
}
