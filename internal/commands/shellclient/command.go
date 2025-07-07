package shellclient

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/raphaelreyna/metashell/internal/config"
	"github.com/raphaelreyna/metashell/internal/log"
	daemonproto "github.com/raphaelreyna/metashell/internal/rpc/go/daemon"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Cmd struct {
	command *cobra.Command
	config  *config.Config

	tty      string
	cmd      string
	cmdKey   string
	exitCode int
}

func New(config *config.Config) *Cmd {
	return &Cmd{
		config: config,
	}
}

func (c *Cmd) Cobra() *cobra.Command {
	if c.command != nil {
		return c.command
	}

	c.command = &cobra.Command{
		Use:   "shellclient",
		Short: "Run the metashell shell client",
		Long:  "Run the metashell shell client to interact with the metashell daemon.",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			log.SetLog(c.config.LogLevel, c.config.RootDir, "shellclient")
			return nil
		},
		RunE: c.Run,
	}

	fs := c.command.Flags()
	fs.StringVar(&c.tty, "tty", "", "internal")
	fs.StringVar(&c.cmd, "cmd", "", "internal")
	fs.StringVar(&c.cmdKey, "cmdKey", "", "internal")
	fs.IntVar(&c.exitCode, "exit-code", -1, "internal")

	return c.command
}

func (sc *Cmd) Run(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

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

func (r *Cmd) recordExitCode(ctx context.Context) error {
	if r.exitCode < 0 {
		panic("exit code not set")
	}

	conn, err := grpc.Dial("unix://"+r.config.Daemon.SocketPath,
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

func (r *Cmd) requestID(ctx context.Context) error {
	conn, err := grpc.Dial("unix://"+r.config.Daemon.SocketPath,
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
