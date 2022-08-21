package main

import (
	"context"
	"io/fs"
	"os"

	"github.com/hashicorp/go-plugin"
	"github.com/raphaelreyna/shelld/plugin/proto/shared"
)

type handler struct {
}

func (h *handler) ReportCommand(ctx context.Context, cmd string) error {
	return os.WriteFile("/home/rr/report.out", []byte(cmd+"\n"), fs.ModeAppend)
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.Handshake,
		Plugins: map[string]plugin.Plugin{
			"commandReportHandler": &shared.CommandReportHandlerPlugin{Impl: &handler{}},
		},
		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
