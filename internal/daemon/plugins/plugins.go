package plugins

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-plugin"
	"github.com/raphaelreyna/metashell/pkg/plugin/proto/proto"
	"github.com/raphaelreyna/metashell/pkg/plugin/proto/shared"

	"github.com/raphaelreyna/metashell/internal/log"
)

type PluginInfo struct {
	Name           string
	AcceptsReports bool
	MetaCommands   map[string]int
}

type Plugins struct {
	PluginsDir      string
	ConfigsCallback func() (map[string][]byte, error)

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
	err := p.Close()
	if err != nil {
		return err
	}

	p.clients = make([]*plugin.Client, 0)
	p.daemonPlugins = make(map[string]shared.DaemonPlugin)
	p.info = make(map[string]PluginInfo)

	configs, err := p.ConfigsCallback()
	if err != nil {
		return fmt.Errorf("error getting plugin configs: %w", err)
	}

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

		log.Info("checking plugin file",
			"path", path)

		client := plugin.NewClient(&plugin.ClientConfig{
			HandshakeConfig: shared.Handshake,
			Plugins:         shared.PluginMap,
			Cmd:             exec.Command(path),
			Logger:          log.GetLogger(),
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
			log.Warn("unable to dispense daemonPlugin, skipping",
				"error", err,
				"path", path,
			)
			continue
		}

		h, ok := iface.(shared.DaemonPlugin)
		if !ok {
			log.Warn("plugin does not implement daemonPlugin interface, skipping",
				"path", path,
			)
			continue
		}

		info, err := h.Info(ctx)
		if err != nil {
			log.Warn("unable to get plugin info, skipping",
				"err", err,
				"path", path,
			)
		}

		// Validate plugin info
		if info == nil {
			log.Warn("plugin info is nil, skipping",
				"path", path,
			)
			continue
		}
		if info.Name == "" {
			log.Warn("plugin info name is empty, skipping",
				"path", path,
			)
			continue
		}
		// TODO(raphaelreyna): Validate plugin info version

		if err := h.Init(ctx, &proto.PluginConfig{
			Data:     configs[info.Name],
			LogLevel: log.GetLogLevel(),
			LogName:  info.Name,
		}); err != nil {
			log.Warn("error initializing plugin, skipping",
				"path", path,
				"error", err,
			)
			continue
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

		log.Info("loaded plugin",
			"name", info.Name,
			"info", info,
		)
	}

	return nil
}

func (p *Plugins) CommandReport(ctx context.Context, rep *proto.ReportCommandRequest) error {
	if len(p.daemonPlugins) == 0 {
		log.Info("no daemonPlugin plugins")
		return nil
	}

	for name, plugin := range p.daemonPlugins {
		if info := p.info[name]; !info.AcceptsReports {
			continue
		}
		if err := plugin.ReportCommand(ctx, rep); err != nil {
			log.Error("daemonPlugin plugin error", err,
				"plugin", name,
			)
		}
	}

	return nil
}

func (p *Plugins) Metacommand(ctx context.Context, pluginName, cmd string, args []string) (*proto.MetacommandResponse, error) {
	h := p.daemonPlugins[pluginName]
	if h == nil {
		return nil, fmt.Errorf("plugin %s not found", pluginName)
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
