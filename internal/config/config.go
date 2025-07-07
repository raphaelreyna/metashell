package config

import (
	"os"
	"path/filepath"

	"github.com/raphaelreyna/metashell/internal/daemon"
	"github.com/raphaelreyna/metashell/internal/metashell"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Daemon    daemon.Config    `yaml:"daemon"`
	MetaShell metashell.Config `yaml:"metashell"`
	LogLevel  string           `yaml:"log_level"`

	RootDir string
}

func ParseConfig() (*Config, error) {
	var c Config

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	c.RootDir = filepath.Join(homeDir, ".metashell")

	if err := metashell.EnsureDir(c.RootDir); err != nil {
		return nil, err
	}

	file, err := os.Open(filepath.Join(c.RootDir, "config.yaml"))
	if err != nil {
		if os.IsNotExist(err) {
			c.Daemon.SetDefaults(c.RootDir)
			c.MetaShell.SetDefaults(c.RootDir)
			c.LogLevel = "INFO"
		}
		return nil, err
	}
	defer file.Close()

	err = yaml.NewDecoder(file).Decode(&c)
	if err != nil {
		return nil, err
	}

	c.Daemon.SetDefaults(c.RootDir)
	c.MetaShell.SetDefaults(c.RootDir)
	if c.LogLevel == "" {
		c.LogLevel = "INFO"
	}

	return &c, err
}
