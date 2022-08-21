package main

import (
	"context"
	"encoding/json"
	"os"

	"github.com/hashicorp/go-plugin"
	"github.com/raphaelreyna/shelld/pkg/plugin/proto"
	"github.com/raphaelreyna/shelld/pkg/plugin/proto/shared"
)

type handler struct {
}

func (h *handler) ReportCommand(ctx context.Context, rep *proto.CommandReport) error {
	bytes, err := json.Marshal(*rep)
	if err != nil {
		return err
	}
	return os.WriteFile("/home/rr/report.out", []byte(string(bytes)+"\n"), os.ModeAppend|os.ModePerm)
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
