package plugins

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-plugin"
	"github.com/raphaelreyna/shelld/pkg/plugin/proto"
	"github.com/raphaelreyna/shelld/pkg/plugin/proto/shared"

	. "github.com/raphaelreyna/shelld/internal/log"
)

type PluginInfo struct {
	Name           string
	AcceptsReports bool
	MetaCommands   map[string]int
}

type Plugins struct {
	PluginsDir string

	clients       []*plugin.Client
	daemonPlugins map[string]shared.DaemonPlugin
	info          map[string]PluginInfo
}

func (p *Plugins) GetPluginInfoMatches(pluginName string) []PluginInfo {
	info, found := p.info[pluginName]
	if found {
		return []PluginInfo{info}
	}

	var infos = make([]PluginInfo, 0)
	for pn, pi := range p.info {
		if strings.HasPrefix(pn, pluginName) {
			infos = append(infos, pi)
			break
		}
	}

	return infos
}

func (p *Plugins) GetMetacommandPluginInfoMatches(pluginName string) []PluginInfo {
	info, found := p.info[pluginName]
	if found {
		return []PluginInfo{info}
	}

	var infos = make([]PluginInfo, 0)
	for pn, pi := range p.info {
		if strings.HasPrefix(pn, pluginName) && 0 < len(pi.MetaCommands) {
			infos = append(infos, pi)
			break
		}
	}

	return infos
}

func (p *Plugins) Reload(ctx context.Context) error {
	if err := p.Close(); err != nil {
		return err
	}

	p.clients = make([]*plugin.Client, 0)
	p.daemonPlugins = make(map[string]shared.DaemonPlugin)
	p.info = make(map[string]PluginInfo)

	entries, err := os.ReadDir(p.PluginsDir)
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

		path := filepath.Join(p.PluginsDir, entry.Name())

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

		iface, err := cc.Dispense("daemonPlugin")
		if err != nil {
			Log.Warn().
				Str("path", path).
				Err(err).
				Msg("unable to dispense daemonPlugin, skipping")
			continue
		}

		h, ok := iface.(shared.DaemonPlugin)
		if !ok {
			Log.Warn().
				Str("path", path).
				Msg("could not cast plugin as daemonPlugin")
			continue
		}

		info, err := h.Info(ctx)
		if err != nil {
			Log.Warn().
				Str("path", path).
				Err(err).
				Msg("unable to get plugin info, skipping")
		}

		pi := PluginInfo{
			Name:           info.Name,
			AcceptsReports: info.AcceptsCommandReports,
			MetaCommands:   make(map[string]int),
		}
		for _, mc := range info.Metacommands {
			pi.MetaCommands[mc.Name] = int(mc.Format)
		}

		p.clients = append(p.clients, client)
		p.daemonPlugins[info.Name] = h
		p.info[info.Name] = pi

		Log.Info().
			Str("path", path).
			Interface("info", info).
			Msg("loaded plugin")
	}

	return nil
}

func (p *Plugins) CommandReport(ctx context.Context, rep *proto.ReportCommandRequest) error {
	if len(p.daemonPlugins) == 0 {
		Log.Info().Msg("no daemonPlugin plugins")
		return nil
	}

	for name, plugin := range p.daemonPlugins {
		if info := p.info[name]; !info.AcceptsReports {
			continue
		}
		if err := plugin.ReportCommand(ctx, rep); err != nil {
			Log.Error().Err(err).
				Msg("daemonPlugin plugin error")
		}
	}

	return nil
}

func (p *Plugins) Metacommand(ctx context.Context, pluginName, cmd string, args []string) (string, error) {
	h := p.daemonPlugins[pluginName]
	if h == nil {
		return "", fmt.Errorf("plugin %s not found", pluginName)
	}

	return h.Metacommand(ctx, &proto.MetacommandRequest{
		MetaCommand: cmd,
		Args:        args,
	})
}

func (p *Plugins) Close() error {
	if len(p.clients) == 0 {
		return nil
	}

	for _, c := range p.clients {
		c.Kill()
	}

	return nil
}
