package installer

import (
	"os"
	"path/filepath"
)

type Config struct {
	shellClientPath string
}

func (c Config) NewInstaller(_ string) *Installer {
	return &Installer{
		config: c,
	}
}

func (c *Config) SetDefaults(rootDir string) {
	if c.shellClientPath == "" {
		c.shellClientPath = os.Args[0]
		c.shellClientPath, _ = filepath.EvalSymlinks(c.shellClientPath)
	}
}
