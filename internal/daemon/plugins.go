package daemon

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/go-plugin"
	"github.com/raphaelreyna/shelld/pkg/plugin/proto"
	"github.com/raphaelreyna/shelld/pkg/plugin/proto/shared"

	. "github.com/raphaelreyna/shelld/internal/log"
)

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
		if os.IsNotExist(err) {
			return nil
		}
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

func (p *plugins) commandReport(ctx context.Context, rep *proto.CommandReport) error {
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
