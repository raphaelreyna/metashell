package metashell

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

type metamodehandler struct {
	input  *textinput.Model
	w, h   int
	daemon daemonproto.MetashellDaemonClient

	metaCommandOut       string
	metaCommandScreenOut string
	completionViewData   map[string][]string

	completionList *list.Model

	completionData map[string][]string
	formats        map[string]map[string]daemonproto.CommandResponseFormat
}

func newMetamodeHandler(daemon daemonproto.MetashellDaemonClient) (*metamodehandler, error) {
	var mh metamodehandler
	mh.daemon = daemon
	mh.completionData = make(map[string][]string)
	mh.formats = make(map[string]map[string]daemonproto.CommandResponseFormat)

	input := textinput.New()
	mh.input = &input
	mh.input.Focus()
	mh.input.Prompt = ""

	resp, err := mh.daemon.GetPluginInfo(context.TODO(), &daemonproto.GetPluginInfoRequest{
		PluginName:      "",
		MetacommandName: "",
	})
	if err != nil {
		return nil, err
	}

	for _, plugin := range resp.Plugins {
		var (
			mcs     = make([]string, len(plugin.Metacommands))
			formats = make(map[string]daemonproto.CommandResponseFormat)
		)
		for idx, mc := range plugin.Metacommands {
			mcs[idx] = mc.Name
			formats[mc.Name] = mc.Format
		}
		mh.completionData[plugin.Name] = mcs
		mh.formats[plugin.Name] = formats
	}

	return &mh, nil
}

func (m *metamodehandler) Init() tea.Cmd {
	return textinput.Blink
}

func (m *metamodehandler) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch input := msg.String(); input {
		case "esc":
			return m, tea.Quit
		case "tab":
			var (
				parts  = strings.Split(input, "::")
				plName string
				mcName string

				data = make(map[string][]string, 0)
			)

			if 0 < len(parts) {
				plName = parts[0]
			}

			if 1 < len(parts) {
				mcName = parts[1]
			}

			for pluginName, mcNames := range m.completionData {
				var mcs = make([]string, 0)
				if !strings.HasPrefix(plName, pluginName) {
					continue
				}
				for _, mcn := range mcNames {
					if strings.HasPrefix(mcn, mcName) {
						mcs = append(mcs, mcn)
					}
				}
				if 0 < len(mcs) {
					data[pluginName] = mcs
				}
			}

			if 0 < len(data) {
				m.completionViewData = data
				m.input = nil
			}
		case "enter":
			var (
				input      = m.input.Value()
				parts      = strings.Split(input, "::")
				pluginName = parts[0]
				mcName     = parts[1]

				pf     = m.formats[pluginName]
				format = pf[mcName]

				req = daemonproto.MetacommandRequest{
					PluginName:  pluginName,
					MetaCommand: mcName,
				}
			)

			switch format {
			case daemonproto.CommandResponseFormat_SCREEN:
				req.FormatArgs = []string{fmt.Sprintf("size=%dx%d", m.w, m.h)}
			}

			resp, err := m.daemon.Metacommand(context.TODO(), &req)
			if err != nil {
				Log.Error().Err(err).
					Msg("error running metacommand")
			}

			switch format {
			case daemonproto.CommandResponseFormat_SHELL:
				m.metaCommandOut = resp.Out
			case daemonproto.CommandResponseFormat_SCREEN:
				m.metaCommandScreenOut = resp.Out
			}

			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h = msg.Height
	}

	var cmd tea.Cmd
	if m.completionList != nil {
		var model list.Model
		model, cmd = m.completionList.Update(msg)
		m.completionList = &model
	} else {
		var model textinput.Model
		model, cmd = m.input.Update(msg)
		m.input = &model
	}
	return m, cmd
}

func (m *metamodehandler) View() string {
	if s := m.metaCommandScreenOut; s != "" {
		m.metaCommandScreenOut = ""
		return s
	}
	if cvd := m.completionViewData; 0 < len(cvd) {
		return m.completionList.View()
	}
	var style = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2)

	out := fmt.Sprintf("Metacommand: %s", m.input.View())
	out = style.Render(out)
	out = lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, out)
	return out
}

type completionItem struct {
	pluginName      string
	metaCommandName string
}

func (c completionItem) Title() string       { return c.pluginName + "::" + c.metaCommandName }
func (c completionItem) Description() string { return c.metaCommandName }
func (c completionItem) FilterValue() string { return c.pluginName + "::" + c.metaCommandName }
