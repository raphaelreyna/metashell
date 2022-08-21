package shared

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"github.com/raphaelreyna/shelld/plugin/proto"
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
	"commandReportHandler": &CommandReportHandlerPlugin{},
}

type CommandReportHandler interface {
	ReportCommand(context.Context, string) error
}

type CommandReportHandlerPlugin struct {
	plugin.Plugin
	Impl CommandReportHandler
}

func (p *CommandReportHandlerPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterCommandReportHandlerServer(s, &CommandReportHandlerServer{Impl: p.Impl})
	return nil
}

func (p *CommandReportHandlerPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &CommandReportHandlerClient{client: proto.NewCommandReportHandlerClient(c)}, nil
}
