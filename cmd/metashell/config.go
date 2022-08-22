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
	RootDir     string
	Daemon      daemon.Config
	Installer   installer.Config
	MetaShell   metashell.Config
	Client      cli.Config
	ShellClient shellclient.Config
}

func parseConfig(path string) (*Config, error) {
	var c Config
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	err = yaml.NewDecoder(file).Decode(&c)
	if err != nil {
		return nil, err
	}

	if c.RootDir == "" {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			return nil, err
		}
		c.RootDir = filepath.Join(cacheDir, "metashell")
	}

	return &c, err
}
