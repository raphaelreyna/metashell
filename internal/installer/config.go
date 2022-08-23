package installer

import (
	"os"
	"path/filepath"
)

type Config struct {
	shellClientPath string
}

func (c Config) NewInstaller(_ string) *Installer {
	c.shellClientPath = os.Args[0]
	c.shellClientPath, _ = filepath.EvalSymlinks(c.shellClientPath)
	return &Installer{
		config: c,
	}
}
