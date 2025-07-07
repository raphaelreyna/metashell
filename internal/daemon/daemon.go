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
	"github.com/raphaelreyna/metashell/internal/log"
	daemonproto "github.com/raphaelreyna/metashell/internal/rpc/go/daemon"
	"github.com/raphaelreyna/metashell/pkg/plugin/proto/proto"
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
	log.Info("handling termination on signal",
		"signal", sig.String(),
	)

	d.grpcServer.GracefulStop()
	log.Info("stopped gRPC server")
	if err := d.listener.Close(); err != nil {
		if !strings.Contains(err.Error(), "use of closed network connection") {
			log.Error("error closing unix socket listener", err)
		} else {
			log.Info("closed unix socket listener")
		}
	} else {
		log.Info("closed unix socket listener")
	}
	if err := d.plugins.Close(); err != nil {
		log.Error("error closing plugins", err)
	}
	return nil
}

func (d *Daemon) Run(ctx context.Context) error {
	d.cks = &cmdKeyService{}
	d.exitCodeStreamChans = make(map[string]chan exitCode)

	cntxt := &godaemon.Context{
		PidFileName: d.config.PidFileName,
		PidFilePerm: 0644,
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

	log.Info("starting daemon")

	// start of daemon
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		log.Info("started signal handlers")
		sig := <-sigChan
		d.termHandler(sig)
	}()

	if _, err := os.Stat(d.config.SocketPath); err == nil {
		if err := os.Remove(d.config.SocketPath); err != nil {
			log.Error("error removing old socket", err,
				"path", d.config.SocketPath,
			)
			return err
		}

		log.Debug("removed old socket",
			"path", d.config.SocketPath,
		)
	}

	d.plugins = &plugins.Plugins{
		PluginsDir: d.config.PluginsDir,
		ConfigsCallback: func() (map[string][]byte, error) {
			// TODO(raphaelreyna): do this in a more efficient way
			m := make(map[string][]byte, len(d.config.PluginConfigs))
			for k, v := range d.config.PluginConfigs {
				jsonData, err := json.Marshal(v)
				if err != nil {
					log.Error("error marshalling plugin config", err,
						"key", k,
					)
					return nil, err
				}
				m[k] = jsonData
			}
			return m, nil
		},
	}
	if err := d.plugins.Reload(ctx); err != nil {
		log.Error("error loading plugins", err,
			"path", d.plugins.PluginsDir,
		)
		return err
	}

	d.listener, err = net.Listen("unix", d.config.SocketPath)
	if err != nil {
		log.Error("error listening on unix socket", err,
			"path", d.config.SocketPath,
		)
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
	log.Info("registered new tty",
		"tty", tty,
	)

	for {
		select {
		case <-ctx.Done():
			return nil
		case ce := <-ceChan:
			err := server.Send(&daemonproto.CommandExitCode{
				Key:      ce.key,
				ExitCode: int32(ce.code),
			})
			if err != nil {
				log.Error("error sending command exit code", err)
				return err
			}
		}
	}
}

func (d *Daemon) RegisterCommandEntry(ctx context.Context, req *daemonproto.CommandEntry) (*daemonproto.CommandKey, error) {
	log.Debug("RegisterCommandEntry")

	key := d.cks.registerVector(&vector{
		command:   req.Command,
		tty:       req.Tty,
		timestamp: req.Timestamp,
	})

	return &daemonproto.CommandKey{Key: key}, nil
}

func (d *Daemon) PreRunQuery(ctx context.Context, req *daemonproto.PreRunQueryRequest) (*daemonproto.PreRunQueryResponse, error) {
	log.Debug("PreRunQuery")

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
	log.Debug("PostRunReport")

	if req.Uuid == "INIT" {
		log.Debug("got INIT")
		return &daemonproto.Empty{}, nil
	}

	v := d.cks.exchangeKey(req.Uuid)
	if v == nil {
		log.Warn("could not find vector for key",
			"key", req.Uuid,
		)
		return &daemonproto.Empty{}, nil
	}

	go func() {
		ecChan, exists := d.exitCodeStreamChans[v.tty]
		if !exists {
			log.Warn("got post run report for a non-connected tty",
				"tty", v.tty,
			)
			return
		}

		ecChan <- exitCode{
			key:  req.Uuid,
			code: int(req.ExitCode),
		}
	}()

	go func() {
		log.Debug("sending command report to plugins")
		d.plugins.CommandReport(context.TODO(), &proto.ReportCommandRequest{
			Command:   v.command,
			Tty:       v.tty,
			Timestamp: uint64(v.timestamp),
			ExitCode:  req.ExitCode,
		})
		log.Debug("sent command report to plugins")
	}()

	return &daemonproto.Empty{}, nil
}

func (d *Daemon) Metacommand(ctx context.Context, req *daemonproto.MetacommandRequest) (*daemonproto.MetacommandResponse, error) {
	log.Info("Metacommand")

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
