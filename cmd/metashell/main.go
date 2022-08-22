package main

import (
	"context"
	"fmt"
	"os"

	. "github.com/raphaelreyna/shelld/internal/log"
)

func main() {
	var ctx = context.Background()

	if err := run(ctx); err != nil {
		panic(err)
	}
}

func run(ctx context.Context) error {
	if len(os.Args) < 2 {
		return fmt.Errorf("subcommand needed")
	}

	var (
		subcommand = os.Args[1]
		err        error
	)

	config, err := parseConfig()
	if err != nil {
		return err
	}
	rootDir := config.rootDir

	if err := SetLog(rootDir, subcommand); err != nil {
		return fmt.Errorf("error initializing logging: %v", err)
	}

	switch subcommand {
	case "daemon":
		d := config.Daemon.NewDaemon(rootDir)
		err = d.Run(ctx)
	case "shellclient":
		sc := config.ShellClient.NewShellClient(rootDir)
		err = sc.Run(ctx)
	case "metashell":
		ms := config.MetaShell.NewMetaShell(rootDir)
		err = ms.Run(ctx)
	case "install":
		i := config.Installer.NewInstaller(rootDir)
		err = i.Run(ctx)
	case "client":
		c := config.Client.NewClient(rootDir)
		err = c.Run(ctx)
	default:
		err = fmt.Errorf("invalid subcommand")
	}

	return err
}
