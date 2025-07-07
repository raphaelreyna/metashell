package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	json "encoding/json"

	"github.com/hashicorp/go-plugin"
	"github.com/raphaelreyna/metashell/pkg/plugin/log"
	"github.com/raphaelreyna/metashell/pkg/plugin/proto/proto"
	"github.com/raphaelreyna/metashell/pkg/plugin/proto/shared"
)

type handler struct {
	history []string
	stderr  *bytes.Buffer
}

func (h *handler) ReportCommand(ctx context.Context, rep *proto.ReportCommandRequest) error {
	log.Info("called ReportCommand")

	h.history = append(h.history, rep.Command)
	return nil
}

func (h *handler) Metacommand(ctx context.Context, req *proto.MetacommandRequest) (*proto.MetacommandResponse, error) {
	log.Info("called Metacommand")

	var (
		resp proto.MetacommandResponse
		err  error
	)

	if len(h.history) == 0 {
		return &resp, nil
	}

	switch req.MetaCommand {
	case "history":
		var items = []map[string]string{}
		for _, item := range h.history {
			i := map[string]string{
				"title":        item,
				"filter_value": item,
				"value":        item,
			}
			items = append(items, i)
		}
		resp.Data, err = json.Marshal(items)
	case "last":
		resp.Data = []byte(h.history[len(h.history)-1])
	default:
		resp.Error = "unknown command"
		err = errors.New(resp.Error)
	}

	log.Info("metacommand response",
		"response", string(resp.Data),
	)

	if err != nil {
		log.Error("metacommand error", err)
	}

	return &resp, err
}

func (h *handler) Info(ctx context.Context) (*proto.PluginInfo, error) {
	return &proto.PluginInfo{
		Name:                  "logging",
		Version:               "v0.0.1",
		AcceptsCommandReports: true,
		Metacommands: []*proto.MetacommandInfo{
			{
				Name:   "history",
				Format: proto.MetacommandResponseFormat_SHELL_INJECTION_LIST,
			},
			{
				Name:   "last",
				Format: proto.MetacommandResponseFormat_SHELL_INJECTION,
			},
		},
	}, nil
}

func (h *handler) Init(ctx context.Context, config *proto.PluginConfig) error {
	log.Init(config)
	log.Info("called Init",
		"config", config.Data,
	)
	return nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	switch path = strings.TrimSuffix(path, "/"); path {
	case "":
		out := &strings.Builder{}

		fmt.Fprint(out, "<h3>history:</h3>\n<br>\n<ul>\n")
		for _, item := range h.history {
			fmt.Fprintf(out, "\t<li>%s</li>\n", item)
		}
		fmt.Fprint(out, "</ul>")

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(out.String()))
	case "logs":
		w.Write(h.stderr.Bytes())
	case "logs/clear":
		h.stderr.Reset()
		w.Write([]byte("cleared"))
	}
}

func main() {
	h := &handler{stderr: bytes.NewBuffer(nil)}

	go http.ListenAndServe(":8086", h)
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.Handshake,
		Plugins: map[string]plugin.Plugin{
			"daemonPlugin": &shared.DaemonPluginImplementation{Impl: h},
		},
		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
		Logger:     log.GetLogger(),
	})
}
