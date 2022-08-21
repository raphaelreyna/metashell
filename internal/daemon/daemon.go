package daemon

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/hashicorp/go-plugin"
	daemonproto "github.com/raphaelreyna/shelld/internal/rpc/go/daemon"
	"github.com/raphaelreyna/shelld/pkg/plugin/proto"
	"github.com/raphaelreyna/shelld/pkg/plugin/proto/shared"
	godaemon "github.com/sevlyar/go-daemon"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	. "github.com/raphaelreyna/shelld/internal/log"
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

type Daemon struct {
	SocketPath  string
	PidFileName string
	LogFileName string
	WorkDir     string

	listener   net.Listener
	grpcServer *grpc.Server

	cks                 *cmdKeyService
	exitCodeStreamChans map[string]chan exitCode

	plugins *plugins

	daemonproto.UnimplementedMetashellDaemonServer
	daemonproto.UnimplementedShellclientDaemonServer
}

type plugins struct {
	clients               []*plugin.Client
	commandReportHandlers []shared.CommandReportHandler
}

func (p *plugins) init(ctx context.Context, pluginDir string) error {
	Log.Info().
		Str("dir", pluginDir).
		Msg("loading plugins")

	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		return fmt.Errorf("error reading plugin dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(pluginDir, entry.Name())

		Log.Info().
			Str("path", path).
			Msg("checking plugin file")

		client := plugin.NewClient(&plugin.ClientConfig{
			HandshakeConfig: shared.Handshake,
			Plugins:         shared.PluginMap,
			Cmd:             exec.Command(path),
			AllowedProtocols: []plugin.Protocol{
				plugin.ProtocolGRPC,
			},
		})

		cc, err := client.Client()
		if err != nil {
			return fmt.Errorf("error opening plugin client: %s: %w", path, err)
		}

		iface, err := cc.Dispense("commandReportHandler")
		if err != nil {
			Log.Info().
				Str("path", path).
				Err(err).
				Msg("skipping plugin")
			continue
		}

		h, ok := iface.(shared.CommandReportHandler)
		if !ok {
			Log.Info().
				Str("path", path).
				Msg("could not cast plugin as commandReportHandler")
			continue
		}

		p.clients = append(p.clients, client)
		p.commandReportHandlers = append(p.commandReportHandlers, h)

		Log.Info().
			Str("path", path).
			Msg("loaded plugin")
	}

	Log.Info().
		Int("client_count", len(p.clients)).
		Int("commandReportHandler_count", len(p.commandReportHandlers)).
		Msg("finished loading plugins")

	return nil
}

func (p *plugins) CommandReport(ctx context.Context, rep *proto.CommandReport) error {
	if len(p.commandReportHandlers) == 0 {
		Log.Info().Msg("no commandReportHandler plugins")
		return nil
	}

	for _, h := range p.commandReportHandlers {
		if err := h.ReportCommand(ctx, rep); err != nil {
			Log.Error().Err(err).
				Msg("commandReportHandler plugin error")
		}
	}

	return nil
}

func (p *plugins) Close() error {
	if len(p.clients) == 0 {
		return nil
	}

	for _, c := range p.clients {
		c.Kill()
	}

	return nil
}

func (d *Daemon) termHandler(sig os.Signal) error {
	Log.Info().
		Str("signal", sig.String()).
		Msg("handling termination on signal")
	d.grpcServer.GracefulStop()
	Log.Info().Msg("stopped gRPC server")
	if err := d.listener.Close(); err != nil {
		if !strings.Contains(err.Error(), "use of closed network connection") {
			Log.Error().
				Err(err).
				Msg("error closing unix socket listener")
		} else {
			Log.Info().Msg("closed unix socket listener")
		}
	} else {
		Log.Info().Msg("closed unix socket listener")
	}
	if err := d.plugins.Close(); err != nil {
		Log.Error().Err(err).
			Msg("error closing plugins")
	}
	return nil
}

func (d *Daemon) Run(ctx context.Context) error {
	d.cks = &cmdKeyService{}
	d.exitCodeStreamChans = make(map[string]chan exitCode)

	cntxt := &godaemon.Context{
		PidFileName: d.PidFileName,
		PidFilePerm: 0644,
		LogFileName: d.LogFileName,
		LogFilePerm: 0640,
		WorkDir:     d.WorkDir,
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

	if _, err := os.Stat(d.SocketPath); err == nil {
		if err := os.Remove(d.SocketPath); err != nil {
			return err
		}
	}

	if _, err := os.Stat(d.SocketPath); err == nil {
		if err := os.Remove(d.SocketPath); err != nil {
			return err
		}
	}

	d.plugins = &plugins{}
	if err := d.plugins.init(ctx, "/home/rr/plugins"); err != nil {
		return err
	}

	d.listener, err = net.Listen("unix", d.SocketPath)
	if err != nil {
		return err
	}

	d.grpcServer = grpc.NewServer()
	daemonproto.RegisterShellclientDaemonServer(d.grpcServer, d)
	daemonproto.RegisterMetashellDaemonServer(d.grpcServer, d)
	return d.grpcServer.Serve(d.listener)
}

func (d *Daemon) NewExitCodeStream(_ *empty.Empty, server daemonproto.MetashellDaemon_NewExitCodeStreamServer) error {
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

	Log.Info().
		Str("tty", tty).
		Msg("registered new tty")
	d.exitCodeStreamChans[tty] = ceChan

	for {
		select {
		case <-ctx.Done():
			return nil
		case ce := <-ceChan:
			Log.Info().
				Msg("got exit code via chan, sending via grpc")
			err := server.Send(&daemonproto.CommandExitCode{
				Key:      ce.key,
				ExitCode: int32(ce.code),
			})
			if err != nil {
				Log.Error().Err(err).
					Msg("error sending command exit code")
				return err
			}
			Log.Info().
				Msg("sent exit code over grpc")
		}
	}
}

func (d *Daemon) RegisterCommandEntry(ctx context.Context, req *daemonproto.CommandEntry) (*daemonproto.CommandKey, error) {
	Log.Info().Msg("RegisterCommandEntry")

	key := d.cks.registerVector(&vector{
		command:   req.Command,
		tty:       req.Tty,
		timestamp: req.Timestamp,
	})

	return &daemonproto.CommandKey{Key: key}, nil
}

func (d *Daemon) PreRunQuery(ctx context.Context, req *daemonproto.PreRunQueryRequest) (*daemonproto.PreRunQueryResponse, error) {
	Log.Info().Msg("PreRunQuery")

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

func (d *Daemon) PostRunReport(ctx context.Context, req *daemonproto.PostRunReportRequest) (*empty.Empty, error) {
	Log.Info().Msg("PostRunReport")

	if req.Uuid == "INIT" {
		Log.Debug().Msg("got INIT")
		return &empty.Empty{}, nil
	}

	v := d.cks.exchangeKey(req.Uuid)
	if v == nil {
		Log.Warn().
			Str("key", req.Uuid).
			Msg("could not find vector for key")
		return &empty.Empty{}, nil
	}

	go func() {
		ecChan, exists := d.exitCodeStreamChans[v.tty]
		if !exists {
			Log.Warn().
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
		Log.Debug().Msg("running hook")
		d.plugins.CommandReport(context.TODO(), &proto.CommandReport{
			Command:   v.command,
			Tty:       v.tty,
			Timestamp: uint64(v.timestamp),
			ExitCode:  req.ExitCode,
		})
		Log.Debug().Msg("ran hook")
	}()

	return &empty.Empty{}, nil
}
