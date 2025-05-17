package shared

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"github.com/raphaelreyna/metashell/pkg/plugin/proto"
	"google.golang.org/grpc"
)

// Handshake is a common handshake that is shared by plugin and host.
var Handshake = plugin.HandshakeConfig{
	// This isn't required when using VersionedPlugins
	ProtocolVersion:  1,
	MagicCookieKey:   "BASIC_PLUGIN",
	MagicCookieValue: "hello",
}

// PluginMap is the map of plugins we can dispense.
var PluginMap = map[string]plugin.Plugin{
	"daemonPlugin": &DaemonPluginImplementation{},
}

type DaemonPlugin interface {
	ReportCommand(context.Context, *proto.ReportCommandRequest) error
	Metacommand(context.Context, *proto.MetacommandRequest) (*proto.MetacommandResponse, error)
	Info(context.Context) (*proto.PluginInfo, error)
}

type DaemonPluginImplementation struct {
	plugin.Plugin
	Impl DaemonPlugin
}

func (p *DaemonPluginImplementation) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterDaemonPluginServer(s, &DaemonPluginServer{Impl: p.Impl})
	return nil
}

func (p *DaemonPluginImplementation) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &DaemonPluginClient{client: proto.NewDaemonPluginClient(c)}, nil
}
