package metamode

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	daemonproto "github.com/raphaelreyna/shelld/internal/rpc/go/daemon"

	. "github.com/raphaelreyna/shelld/internal/log"
)

var (
	inputBorder = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2)
)

type mainScreen struct {
	prompt          string
	pluginNameDelim string

	input textinput.Model

	next   func(string, any)
	size   func() (int, int)
	daemon daemonproto.MetashellDaemonClient

	completionData map[string][]string
	formats        map[string]map[string]daemonproto.MetacommandResponseFormat
}

func (ms *mainScreen) Name() string {
	return "main_screen"
}

func (ms *mainScreen) Init(r rootScreen, data any) (tea.Cmd, error) {
	initData, _ := data.(string)

	ms.next = r.next
	ms.daemon = r.daemon()
	ms.size = r.size

	ms.input = textinput.New()
	ms.input.Prompt = ms.prompt
	ms.input.SetValue(initData)
	ms.input.Focus()

	ms.completionData = make(map[string][]string)
	ms.formats = make(map[string]map[string]daemonproto.MetacommandResponseFormat)

	if err := ms.updatePlugins(context.TODO()); err != nil {
		return nil, err
	}

	return textinput.Blink, nil
}

func (ms *mainScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch key := msg.String(); key {
		case "tab":
			pn, mn, _ := ms.parsedInput()
			ms.next("list_screen", listScreenInitData[[]list.Item]{
				nextScreen: ms.Name(),
				items:      ms.createCompletionList(pn, mn),
			})

			return ms, nil
		case "enter":
			var pn, mn, args = ms.parsedInput()
			if err := ms.execMetacommand(context.TODO(), pn, mn, args); err != nil {
				Log.Error().Err(err).
					Msg("error executing metacommand")
			}
			return ms, nil
		}
	}

	var cmd tea.Cmd
	ms.input, cmd = ms.input.Update(msg)

	return ms, cmd
}

func (ms *mainScreen) View() string {
	r := ms.input.View()
	return inputBorder.Render(r)
}

func (ms *mainScreen) parsedInput() (string, string, []string) {
	var (
		topLevelParts = strings.Split(ms.input.Value(), " ")
		parts         = strings.Split(topLevelParts[0], ms.pluginNameDelim)
		args          = topLevelParts[1:]
		plName        string
		mcName        string
	)

	if 0 < len(parts) {
		plName = parts[0]
	}

	if 1 < len(parts) {
		mcName = parts[1]
	}

	return plName, mcName, args
}

func (ms *mainScreen) createCompletionList(plugin, metacommand string) []list.Item {
	var items = make([]list.Item, 0)

	for pluginName, mcNames := range ms.completionData {
		if !strings.HasPrefix(pluginName, plugin) {
			continue
		}
		for _, mcn := range mcNames {
			if strings.HasPrefix(mcn, metacommand) {
				items = append(items, &listableItem{
					ItemTitle:       pluginName + " - " + mcn,
					ItemDescription: "description",
					ItemFilterValue: pluginName + "::" + mcn,
					ItemValue:       pluginName + "::" + mcn,
				})
			}
		}
	}

	return items
}

func (ms *mainScreen) execMetacommand(ctx context.Context, plugin, metacommand string, args []string) error {
	var (
		pf     = ms.formats[plugin]
		format = pf[metacommand]

		req = daemonproto.MetacommandRequest{
			PluginName:  plugin,
			MetaCommand: metacommand,
		}
	)

	switch format {
	case daemonproto.MetacommandResponseFormat_SCREEN:
		var w, h = ms.size()
		req.FormatArgs = []string{fmt.Sprintf("size=%dx%d", w, h)}
	}

	resp, err := ms.daemon.Metacommand(ctx, &req)
	if err != nil {
		return err
	}

	switch format {
	case daemonproto.MetacommandResponseFormat_SHELL_INJECTION:
		ms.next("shell_injection", string(resp.Data))
	case daemonproto.MetacommandResponseFormat_SHELL_INJECTION_LIST:
		ms.next("list_screen", listScreenInitData[[]byte]{
			nextScreen: "shell_injection",
			items:      resp.Data,
		})
	case daemonproto.MetacommandResponseFormat_SCREEN:
		ms.next("fullscreen", string(resp.Data))
	}

	return nil
}

func (ms *mainScreen) updatePlugins(ctx context.Context) error {
	resp, err := ms.daemon.GetPluginInfo(ctx, &daemonproto.GetPluginInfoRequest{
		PluginName:      "",
		MetacommandName: "",
	})
	if err != nil {
		return err
	}

	for _, plugin := range resp.Plugins {
		var (
			mcs     = make([]string, len(plugin.Metacommands))
			formats = make(map[string]daemonproto.MetacommandResponseFormat)
		)
		for idx, mc := range plugin.Metacommands {
			mcs[idx] = mc.Name
			formats[mc.Name] = mc.Format
		}
		ms.completionData[plugin.Name] = mcs
		ms.formats[plugin.Name] = formats
	}

	return nil
}
