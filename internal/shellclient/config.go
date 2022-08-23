package shellclient

import "path/filepath"

type Config struct {
	socketPath string
}

func (c Config) NewShellClient(rootDir string) *ShellClient {
	c.socketPath = filepath.Join(rootDir, "daemon.socket")
	return &ShellClient{config: c}
}
