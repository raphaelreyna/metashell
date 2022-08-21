package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/raphaelreyna/shelld/internal/cli"
	"github.com/raphaelreyna/shelld/internal/daemon"
	"github.com/raphaelreyna/shelld/internal/installer"
	"github.com/raphaelreyna/shelld/internal/metashell"
	"github.com/raphaelreyna/shelld/internal/shellclient"

	. "github.com/raphaelreyna/shelld/internal/log"
)

func main() {
	var (
		ctx    = context.Background()
		bundle = InitBundle{
			RootDir: "/home/rr/Projects/metashell/_test",
			Shell:   "/bin/bash",
		}
	)

	if err := Init(ctx, &bundle); err != nil {
		panic(err)
	}
}

type InitBundle struct {
	RootDir string
	Shell   string
}

func Init(ctx context.Context, ib *InitBundle) error {
	if len(os.Args) < 2 {
		return fmt.Errorf("subcommand needed")
	}

	var (
		subcommand = os.Args[1]
		rootDir    = filepath.Join(ib.RootDir, "metashell")
		err        error
	)
	if err := SetLog(rootDir, subcommand); err != nil {
		return fmt.Errorf("error initializing logging: %v", err)
	}

	socketPath := filepath.Join(rootDir, "daemon", "daemon.socket")

	switch subcommand {
	case "daemon":
		d := daemon.Daemon{
			SocketPath:  socketPath,
			LogFileName: Log.OutFilePath(),
			WorkDir:     filepath.Join(rootDir, "daemon"),
			PidFileName: filepath.Join(rootDir, "daemon", "daemon.pid"),
		}
		Log.Info().Msg("starting daemon")
		err = d.Run(ctx)
	case "shellclient":
		sc := shellclient.ShellClient{
			SocketPath: socketPath,
		}
		err = sc.Run(ctx)
	case "metashell":
		execPath, err := os.Executable()
		if err != nil {
			return err
		}
		execPath, err = filepath.EvalSymlinks(execPath)
		if err != nil {
			return err
		}
		ms := metashell.MetaShell{
			ShellPath:  ib.Shell,
			SocketPath: socketPath,
			ExecPath:   execPath,
		}
		err = ms.Run(ctx)
	case "install":
		execPath, err := os.Executable()
		if err != nil {
			return err
		}
		execPath, err = filepath.EvalSymlinks(execPath)
		if err != nil {
			return err
		}
		i := installer.Installer{
			ShellClientPath: execPath,
		}
		err = i.Run(ctx)
	case "client":
		d := cli.Client{
			SocketPath:  socketPath,
			LogFileName: Log.OutFilePath(),
			WorkDir:     filepath.Join(rootDir, "daemon"),
			PidFileName: filepath.Join(rootDir, "daemon", "daemon.pid"),
		}
		err = d.Run(ctx)
	default:
		err = fmt.Errorf("invalid subcommand")
	}

	return err
}
