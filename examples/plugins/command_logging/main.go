package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hashicorp/go-plugin"
	"github.com/raphaelreyna/shelld/pkg/plugin/proto"
	"github.com/raphaelreyna/shelld/pkg/plugin/proto/shared"
)

type handler struct {
	history []string
}

func (h *handler) ReportCommand(ctx context.Context, rep *proto.ReportCommandRequest) error {
	h.history = append(h.history, rep.Command)
	return nil
}

func (h *handler) Metacommand(ctx context.Context, req *proto.MetacommandRequest) (string, error) {
	if len(h.history) == 0 {
		return "", nil
	}

	switch req.MetaCommand {
	case "history":
		var width, height int
		parts := strings.Split(req.FormatArgs[0], "=")
		fmt.Sscanf(parts[1], "%dx%d", &width, &height)
		out := strings.Join(h.history, ", ")
		out = lipgloss.Place(width, height, lipgloss.Center, lipgloss.Top, out)
		return out, nil
	case "last":
		return h.history[len(h.history)-1], nil
	}

	return "", errors.New("unknown command")
}

func (h *handler) Info(ctx context.Context) (*proto.PluginInfo, error) {
	return &proto.PluginInfo{
		Name:                  "p1",
		Version:               "v0.0.1",
		AcceptsCommandReports: true,
		Metacommands: []*proto.MetacommandInfo{
			{
				Name:   "history",
				Format: proto.MetacommandResponseFormat_SCREEN,
			},
			{
				Name:   "last",
				Format: proto.MetacommandResponseFormat_SHELL,
			},
		},
	}, nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	out := &strings.Builder{}

	fmt.Fprint(out, "<h3>history:</h3>\n<br>\n<ul>\n")
	for _, item := range h.history {
		fmt.Fprintf(out, "\t<li>%s</li>\n", item)
	}
	fmt.Fprint(out, "</ul>")

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(out.String()))
}

func main() {
	h := &handler{}
	go http.ListenAndServe(":8080", h)
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.Handshake,
		Plugins: map[string]plugin.Plugin{
			"daemonPlugin": &shared.DaemonPluginImplementation{Impl: h},
		},
		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
