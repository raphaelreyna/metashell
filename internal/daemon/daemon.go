package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/raphaelreyna/metashell/internal/daemon/plugins"
	daemonproto "github.com/raphaelreyna/metashell/internal/rpc/go/daemon"
	"github.com/raphaelreyna/metashell/pkg/plugin/proto/proto"
	godaemon "github.com/sevlyar/go-daemon"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	. "github.com/raphaelreyna/metashell/internal/log"
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
	config Config

	listener   net.Listener
	grpcServer *grpc.Server

	cks                 *cmdKeyService
	exitCodeStreamChans map[string]chan exitCode

	plugins *plugins.Plugins

	daemonproto.UnimplementedMetashellDaemonServer
	daemonproto.UnimplementedShellclientDaemonServer
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
		PidFileName: d.config.PidFileName,
		PidFilePerm: 0644,
		LogFileName: d.config.LogFileName,
		LogFilePerm: 0640,
		WorkDir:     d.config.WorkDir,
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

	Log.Info().
		Msg("starting daemon")

	// start of daemon
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		Log.Info().
			Msg("started signal handlers")
		sig := <-sigChan
		d.termHandler(sig)
	}()

	if _, err := os.Stat(d.config.SocketPath); err == nil {
		if err := os.Remove(d.config.SocketPath); err != nil {
			Log.Error().Err(err).
				Str("path", d.config.SocketPath).
				Msg("error removing old socket")
			return err
		}

		Log.Debug().
			Str("path", d.config.SocketPath).
			Msg("removed old socket")
	}

	d.plugins = &plugins.Plugins{
		PluginsDir: d.config.PluginsDir,
		ConfigsCallback: func() (map[string][]byte, error) {
			// TODO(raphaelreyna): do this in a more efficient way
			m := make(map[string][]byte, len(d.config.PluginConfigs))
			for k, v := range d.config.PluginConfigs {
				jsonData, err := json.Marshal(v)
				if err != nil {
					Log.Error().Err(err).
						Str("key", k).
						Msg("error marshalling plugin config")
					return nil, err
				}
				m[k] = jsonData
			}
			return m, nil
		},
	}
	if err := d.plugins.Reload(ctx); err != nil {
		Log.Error().Err(err).
			Str("path", d.plugins.PluginsDir).
			Msg("error loading plugins")
		return err
	}

	d.listener, err = net.Listen("unix", d.config.SocketPath)
	if err != nil {
		Log.Error().Err(err).
			Str("path", d.config.SocketPath).
			Msg("error listening on unix socket")
		return err
	}

	d.grpcServer = grpc.NewServer()
	daemonproto.RegisterShellclientDaemonServer(d.grpcServer, d)
	daemonproto.RegisterMetashellDaemonServer(d.grpcServer, d)
	return d.grpcServer.Serve(d.listener)
}

func (d *Daemon) NewExitCodeStream(_ *daemonproto.Empty, server daemonproto.MetashellDaemon_NewExitCodeStreamServer) error {
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

	d.exitCodeStreamChans[tty] = ceChan
	Log.Info().
		Str("tty", tty).
		Msg("registered new tty")

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

func (d *Daemon) PostRunReport(ctx context.Context, req *daemonproto.PostRunReportRequest) (*daemonproto.Empty, error) {
	Log.Info().Msg("PostRunReport")

	if req.Uuid == "INIT" {
		Log.Debug().Msg("got INIT")
		return &daemonproto.Empty{}, nil
	}

	v := d.cks.exchangeKey(req.Uuid)
	if v == nil {
		Log.Warn().
			Str("key", req.Uuid).
			Msg("could not find vector for key")
		return &daemonproto.Empty{}, nil
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
		Log.Debug().Msg("sending command report to plugins")
		d.plugins.CommandReport(context.TODO(), &proto.ReportCommandRequest{
			Command:   v.command,
			Tty:       v.tty,
			Timestamp: uint64(v.timestamp),
			ExitCode:  req.ExitCode,
		})
		Log.Debug().Msg("sent command report to plugins")
	}()

	return &daemonproto.Empty{}, nil
}

func (d *Daemon) Metacommand(ctx context.Context, req *daemonproto.MetacommandRequest) (*daemonproto.MetacommandResponse, error) {
	Log.Info().Msg("Metacommand")

	resp1, err := d.plugins.Metacommand(ctx, req.PluginName, req.MetaCommand, req.Args)
	resp2 := &daemonproto.MetacommandResponse{}
	if err != nil {
		resp2.Error = err.Error()
	}
	if resp1 != nil {
		resp2.Data = resp1.Data
	}

	return resp2, err
}

func (d *Daemon) GetPluginInfo(ctx context.Context, req *daemonproto.GetPluginInfoRequest) (*daemonproto.GetPluginInfoResponse, error) {
	var plugins = make([]*daemonproto.PluginInfo, 0)

	for _, info := range d.plugins.GetMetacommandPluginInfoMatches(req.PluginName) {
		var mcs = make([]*daemonproto.MetacommandInfo, 0)

		for mcName, mcFormat := range info.MetaCommands {
			if strings.HasPrefix(mcName, req.MetacommandName) {
				mcs = append(mcs, &daemonproto.MetacommandInfo{
					Name:   mcName,
					Format: daemonproto.MetacommandResponseFormat(mcFormat),
				})
			}
		}

		if 0 < len(mcs) {
			plugins = append(plugins, &daemonproto.PluginInfo{
				Name:                  info.Name,
				AcceptsCommandReports: info.AcceptsReports,
				Metacommands:          mcs,
			})
		}
	}

	return &daemonproto.GetPluginInfoResponse{
		Plugins: plugins,
	}, nil
}
