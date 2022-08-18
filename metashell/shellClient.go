package metashell

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	daemonproto "github.com/raphaelreyna/shelld/rpc/go/daemon"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const SubCommandShellClient = "shellclient"

type shellClient struct {
	SocketPath string

	tty      string
	cmd      string
	cmdKey   string
	exitCode int
}

func (sc *shellClient) parseFlags() {
	fs := flag.NewFlagSet("shellclient", flag.PanicOnError)
	fs.StringVar(&sc.tty, "tty", "", "internal")
	fs.StringVar(&sc.cmd, "cmd", "", "internal")
	fs.StringVar(&sc.cmdKey, "cmdKey", "", "internal")
	fs.IntVar(&sc.exitCode, "exit-code", -1, "internal")

	fs.Parse(os.Args[2:])
}

func (sc *shellClient) Run(ctx context.Context) error {
	sc.parseFlags()

	log.Infof("shell client running with: %+v\n", *sc)
	log.Infof("shell client running with os args: %+v\n", os.Args)

	var (
		runRecordExitCode     = -1 < sc.exitCode && sc.cmdKey != ""
		runPreRunQueryRequest = sc.tty != "" && sc.cmd != ""
	)

	switch {
	case runPreRunQueryRequest && runRecordExitCode:
		break
	case runPreRunQueryRequest:
		log.Info("prerunquery")
		return sc.requestID(ctx)
	case runRecordExitCode:
		log.Info("report")
		return sc.recordExitCode(ctx)
	}

	return fmt.Errorf("invalid flag combination")
}

func (r *shellClient) recordExitCode(ctx context.Context) error {
	if r.exitCode < 0 {
		panic("exit code not set")
	}

	conn, err := grpc.Dial("unix://"+r.SocketPath,
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

func (r *shellClient) requestID(ctx context.Context) error {
	conn, err := grpc.Dial("unix://"+r.SocketPath,
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
