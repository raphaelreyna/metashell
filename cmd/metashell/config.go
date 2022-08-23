package main

import (
	"os"
	"path/filepath"

	"github.com/raphaelreyna/shelld/internal/cli"
	"github.com/raphaelreyna/shelld/internal/daemon"
	"github.com/raphaelreyna/shelld/internal/installer"
	"github.com/raphaelreyna/shelld/internal/metashell"
	"github.com/raphaelreyna/shelld/internal/shellclient"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Daemon      daemon.Config
	Installer   installer.Config
	MetaShell   metashell.Config
	Client      cli.Config
	ShellClient shellclient.Config

	rootDir string
}

func parseConfig() (*Config, error) {
	var c Config

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	c.rootDir = filepath.Join(homeDir, ".metashell")

	if err := metashell.EnsureDir(c.rootDir); err != nil {
		return nil, err
	}

	file, err := os.Open(filepath.Join(c.rootDir, "config.yaml"))
	if err != nil {
		if os.IsNotExist(err) {
			return defaultConfig(c.rootDir)
		}
		return nil, err
	}
	defer file.Close()

	err = yaml.NewDecoder(file).Decode(&c)
	if err != nil {
		return nil, err
	}

	return &c, err
}

func defaultConfig(rootDir string) (*Config, error) {
	return &Config{
		Daemon:    daemon.DefaultConfig(rootDir),
		MetaShell: metashell.DefaultConfig(rootDir),
		rootDir:   rootDir,
	}, nil
}
