package metamode

import (
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/raphaelreyna/metashell/internal/log"
	daemonproto "github.com/raphaelreyna/metashell/internal/rpc/go/daemon"
)

type rootScreen interface {
	next(string, any)
	size() (w, h int)
	daemon() daemonproto.MetashellDaemonClient
}

type screen interface {
	Name() string
	Init(rootScreen, any) (tea.Cmd, error)
	Update(tea.Msg) (screen, tea.Cmd)
	View() string
}

type Handler struct {
	w, h           int
	daemonClient   daemonproto.MetashellDaemonClient
	metaCommandOut string
	screens        map[string]screen

	activeScreen            screen
	newActiveScreen         string
	newActiveScreenInitData any
	sync.Mutex
}

func (m *Handler) Initialize(daemon daemonproto.MetashellDaemonClient, quit func()) error {
	m.daemonClient = daemon
	m.screens = map[string]screen{
		"main_screen": &mainScreen{
			prompt:          "> ",
			pluginNameDelim: "::",
		},
		"list_screen": &listScreen{},
	}
	m.activeScreen = m.screens["main_screen"]
	return nil
}

func (m *Handler) GetShellInjection() string {
	return m.metaCommandOut
}

func (m *Handler) Init() tea.Cmd {
	for _, scrn := range m.screens {
		if scrn != m.activeScreen {
			scrn.Init(m, nil)
		}
	}

	cmd, err := m.activeScreen.Init(m, nil)
	if err != nil {
		panic(err)
	}

	return cmd
}

func (m *Handler) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.Lock()
	defer m.Unlock()
	if m.activeScreen == nil {
		return m, tea.Quit
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch key := msg.String(); key {
		case "esc":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h = msg.Height
	}

	s, c := m.activeScreen.Update(msg)
	switch m.newActiveScreen {
	case "":
		m.activeScreen = s
	case "shell_injection":
		m.metaCommandOut = m.newActiveScreenInitData.(string)
		c = tea.Quit
	default:
		if s := m.screens[m.newActiveScreen]; s != nil {
			s.Init(m, m.newActiveScreenInitData)
			m.activeScreen = s
			m.newActiveScreen = ""
			m.newActiveScreenInitData = nil
		} else {
			log.Warn("could not find screen by name",
				"screen-name", m.newActiveScreen,
			)
		}
	}

	return m, c
}

func (m *Handler) View() string {
	if m.activeScreen == nil {
		return ""
	}
	return lipgloss.Place(m.w, m.h,
		lipgloss.Center, lipgloss.Center,
		m.activeScreen.View(),
	)
}

func (m *Handler) size() (w, h int) {
	return m.w, m.h
}

func (m *Handler) next(sn string, data any) {
	m.newActiveScreen = sn
	m.newActiveScreenInitData = data
}

func (m *Handler) daemon() daemonproto.MetashellDaemonClient {
	return m.daemonClient
}
