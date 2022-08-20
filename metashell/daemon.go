package metashell

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/golang/protobuf/ptypes/empty"
	daemonproto "github.com/raphaelreyna/shelld/rpc/go/daemon"
	godaemon "github.com/sevlyar/go-daemon"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const SubCommandDaemon = "daemon"

type PostRunReport struct {
	TTY       string
	Command   string
	Timestamp int64
}

type PostRunReportHandlerFunc func(context.Context, *PostRunReport) error

type exitCode struct {
	key  string
	code int
	time int64
}

type daemon struct {
	socketPath           string
	postRunReportHandler PostRunReportHandlerFunc
	pidFileName          string
	logFileName          string
	workDir              string

	listener   net.Listener
	grpcServer *grpc.Server

	cks                 *cmdKeyService
	exitCodeStreamChans map[string]chan exitCode

	daemonproto.UnimplementedMetashellDaemonServer
	daemonproto.UnimplementedShellclientDaemonServer
}

func (d *daemon) termHandler(sig os.Signal) error {
	log.Info().
		Str("signal", sig.String()).
		Msg("handling termination on signal")
	d.grpcServer.GracefulStop()
	log.Info().Msg("stopped gRPC server")
	if err := d.listener.Close(); err != nil {
		if !strings.Contains(err.Error(), "use of closed network connection") {
			log.Error().
				Err(err).
				Msg("error closing unix socket listener")
		} else {
			log.Info().Msg("closed unix socket listener")
		}
	} else {
		log.Info().Msg("closed unix socket listener")
	}
	return nil
}

func (d *daemon) run(ctx context.Context) error {
	d.cks = &cmdKeyService{}
	d.exitCodeStreamChans = make(map[string]chan exitCode)

	cntxt := &godaemon.Context{
		PidFileName: d.pidFileName,
		PidFilePerm: 0644,
		LogFileName: d.logFileName,
		LogFilePerm: 0640,
		WorkDir:     d.workDir,
		Umask:       027,
	}

	dd, err := cntxt.Reborn()
	if err != nil {
		return err
	}
	if dd != nil {
		return nil
	}
	defer cntxt.Release()

	// start of daemon
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		d.termHandler(sig)
	}()

	if _, err := os.Stat(d.socketPath); err == nil {
		if err := os.Remove(d.socketPath); err != nil {
			return err
		}
	}

	if _, err := os.Stat(d.socketPath); err == nil {
		if err := os.Remove(d.socketPath); err != nil {
			return err
		}
	}

	d.listener, err = net.Listen("unix", d.socketPath)
	if err != nil {
		return err
	}

	d.grpcServer = grpc.NewServer()
	daemonproto.RegisterShellclientDaemonServer(d.grpcServer, d)
	daemonproto.RegisterMetashellDaemonServer(d.grpcServer, d)
	return d.grpcServer.Serve(d.listener)
}

func (d *daemon) NewExitCodeStream(_ *empty.Empty, server daemonproto.MetashellDaemon_NewExitCodeStreamServer) error {
	ctx := server.Context()

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return fmt.Errorf("no metadata found")
	}
	tty := md.Get("TTY")[0]
	ceChan := make(chan exitCode, 1)

	if tty == "" {
		return fmt.Errorf("no tty given in metadata")
	}

	log.Info().
		Str("tty", tty).
		Msg("registered new tty")
	d.exitCodeStreamChans[tty] = ceChan

	for {
		select {
		case <-ctx.Done():
			return nil
		case ce := <-ceChan:
			log.Info().
				Msg("got exit code via chan, sending via grpc")
			err := server.Send(&daemonproto.CommandExitCode{
				Key:      ce.key,
				ExitCode: int32(ce.code),
			})
			if err != nil {
				log.Error().Err(err).
					Msg("error sending command exit code")
				return err
			}
			log.Info().
				Msg("sent exit code over grpc")
		}
	}
}

func (d *daemon) RegisterCommandEntry(ctx context.Context, req *daemonproto.CommandEntry) (*daemonproto.CommandKey, error) {
	log.Info().Msg("RegisterCommandEntry")

	key := d.cks.registerVector(&vector{
		command:   req.Command,
		tty:       req.Tty,
		timestamp: req.Timestamp,
	})

	return &daemonproto.CommandKey{Key: key}, nil
}

func (d *daemon) PreRunQuery(ctx context.Context, req *daemonproto.PreRunQueryRequest) (*daemonproto.PreRunQueryResponse, error) {
	log.Info().Msg("PreRunQuery")

	k := d.cks.getKey(&vector{
		command:   req.Command,
		tty:       req.Tty,
		timestamp: req.Timestamp,
	})

	if k == "" {
		k = "INIT"
	}

	return &daemonproto.PreRunQueryResponse{Uuid: k}, nil
}

func (d *daemon) PostRunReport(ctx context.Context, req *daemonproto.PostRunReportRequest) (*empty.Empty, error) {
	log.Info().Msg("PostRunReport")

	if req.Uuid == "INIT" {
		log.Debug().Msg("got INIT")
		return &empty.Empty{}, nil
	}

	v := d.cks.exchangeKey(req.Uuid)
	if v == nil {
		log.Warn().
			Str("key", req.Uuid).
			Msg("could not find vector for key")
		return &empty.Empty{}, nil
	}

	go func() {
		ecChan, exists := d.exitCodeStreamChans[v.tty]
		if !exists {
			log.Warn().
				Str("tty", v.tty).
				Msg("got post run report for a non-connected tty")
			return
		}

		ecChan <- exitCode{
			key:  req.Uuid,
			code: int(req.ExitCode),
		}
	}()

	go func() {
		ctx = context.WithValue(ctx, stdoutKey{}, log)
		log.Debug().Msg("running hook")
		d.postRunReportHandler(ctx, &PostRunReport{
			TTY:       v.tty,
			Command:   v.command,
			Timestamp: v.timestamp,
		})
		log.Debug().Msg("ran hook")
	}()

	return &empty.Empty{}, nil
}

type stdoutKey struct{}

func GetStdout(ctx context.Context) io.Writer {
	w, _ := ctx.Value(stdoutKey{}).(io.Writer)
	return w
}
