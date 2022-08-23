package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

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

func (h *handler) Metacommand(ctx context.Context, cmd string) (string, error) {
	if len(h.history) == 0 {
		return "", nil
	}

	switch cmd {
	case "history":
		return strings.Join(h.history, ", "), nil
	case "last":
		return h.history[len(h.history)-1], nil
	}

	return "", nil
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
