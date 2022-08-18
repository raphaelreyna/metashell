package metashell

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type InitBundle struct {
	RootDir string
	Shell   string

	PostRunReportHandlerFunc PostRunReportHandlerFunc
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
	if err := initLogging(rootDir, subcommand); err != nil {
		panic(err)
	}

	socketPath := filepath.Join(rootDir, "daemon", "daemon.socket")

	switch subcommand {
	case "daemon":
		d := daemon{
			socketPath:           socketPath,
			postRunReportHandler: ib.PostRunReportHandlerFunc,
			logFileName:          log.out.Name(),
			workDir:              filepath.Join(rootDir, "daemon"),
			pidFileName:          filepath.Join(rootDir, "daemon", "daemon.pid"),
		}
		log.Info("starting daemon")
		err = d.Run(ctx)
	case "shellclient":
		sc := shellClient{
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
		ms := metaShell{
			shellPath:  ib.Shell,
			socketPath: socketPath,
			execPath:   execPath,
		}
		err = ms.run(ctx)
	case "install":
		execPath, err := os.Executable()
		if err != nil {
			return err
		}
		execPath, err = filepath.EvalSymlinks(execPath)
		if err != nil {
			return err
		}
		i := installer{
			shellClientPath: execPath,
		}
		err = i.run(ctx)
	default:
		err = fmt.Errorf("invalid subcommand")
	}

	return err
}
