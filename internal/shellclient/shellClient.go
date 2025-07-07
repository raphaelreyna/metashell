package shellclient

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/raphaelreyna/metashell/internal/log"
	daemonproto "github.com/raphaelreyna/metashell/internal/rpc/go/daemon"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const SubCommandShellClient = "shellclient"

type ShellClient struct {
	config Config

	tty      string
	cmd      string
	cmdKey   string
	exitCode int
}

func (sc *ShellClient) parseFlags() {
	fs := flag.NewFlagSet("shellclient", flag.PanicOnError)
	fs.StringVar(&sc.tty, "tty", "", "internal")
	fs.StringVar(&sc.cmd, "cmd", "", "internal")
	fs.StringVar(&sc.cmdKey, "cmdKey", "", "internal")
	fs.IntVar(&sc.exitCode, "exit-code", -1, "internal")

	fs.Parse(os.Args[2:])
}

func (sc *ShellClient) Run(ctx context.Context) error {
	sc.parseFlags()

	logEvent := log.With("args", os.Args)

	var (
		runRecordExitCode     = -1 < sc.exitCode && sc.cmdKey != ""
		runPreRunQueryRequest = sc.tty != "" && sc.cmd != ""
	)

	switch {
	case runPreRunQueryRequest && runRecordExitCode:
		break
	case runPreRunQueryRequest:
		logEvent.Info("ran", "mode", "preRunQuery")
		return sc.requestID(ctx)
	case runRecordExitCode:
		logEvent.Info("ran", "mode", "postRunReport")
		return sc.recordExitCode(ctx)
	}

	return fmt.Errorf("invalid flag combination")
}

func (r *ShellClient) recordExitCode(ctx context.Context) error {
	if r.exitCode < 0 {
		panic("exit code not set")
	}

	conn, err := grpc.Dial("unix://"+r.config.socketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := daemonproto.NewShellclientDaemonClient(conn)
	_, err = client.PostRunReport(ctx, &daemonproto.PostRunReportRequest{
		Uuid:     r.cmdKey,
		ExitCode: int32(r.exitCode),
	})
	return err
}

func (r *ShellClient) requestID(ctx context.Context) error {
	conn, err := grpc.Dial("unix://"+r.config.socketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}
	defer conn.Close()
	client := daemonproto.NewShellclientDaemonClient(conn)
	resp, err := client.PreRunQuery(ctx, &daemonproto.PreRunQueryRequest{
		Command:   r.cmd,
		Tty:       r.tty,
		Timestamp: time.Now().Unix(),
	})
	if err != nil {
		return err
	}

	fmt.Fprint(os.Stdout, resp.Uuid)
	return err
}
