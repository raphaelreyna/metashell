package metamode

import (
	"encoding/json"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type listableItem struct {
	ItemTitle       string `json:"title"`
	ItemDescription string `json:"description"`
	ItemFilterValue string `json:"filter_value"`
	ItemValue       any    `json:"value,omitempty"`
}

func (l *listableItem) Title() string       { return l.ItemTitle }
func (l *listableItem) Description() string { return l.ItemDescription }
func (l *listableItem) FilterValue() string { return l.ItemFilterValue }
func (l *listableItem) Value() any          { return l.ItemValue }

type listData interface {
	[]list.Item | []byte
}

type listScreenInitData[D listData] struct {
	items      D
	nextScreen string
}

type listScreen struct {
	nextScreenName string
	next           func(string, any)
	size           func() (int, int)
	l              list.Model
}

func (s *listScreen) Name() string {
	return "list_screen"
}

func (s *listScreen) Init(rs rootScreen, data any) (tea.Cmd, error) {
	var (
		w, h      = rs.size()
		d         = list.NewDefaultDelegate()
		listItems []list.Item
	)

	d.UpdateFunc = s.delegatedUpdate
	s.next = rs.next
	s.size = rs.size

	switch data := data.(type) {
	case listScreenInitData[[]list.Item]:
		listItems = data.items
		s.nextScreenName = data.nextScreen
	case listScreenInitData[[]byte]:
		var items []*listableItem
		if err := json.Unmarshal(data.items, &items); err != nil {
			return nil, err
		}

		listItems = make([]list.Item, len(items))
		for i := 0; i < len(items); i++ {
			listItems[i] = items[i]
		}

		s.nextScreenName = data.nextScreen
	}

	s.l = list.New(listItems, d, w, h)

	return nil, nil
}

func (s *listScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	var cmd tea.Cmd
	s.l, cmd = s.l.Update(msg)
	return s, cmd
}

func (s *listScreen) View() string {
	var w, h = s.size()
	s.l.SetSize(w, h)
	return s.l.View()
}

func (s *listScreen) delegatedUpdate(msg tea.Msg, model *list.Model) tea.Cmd {
	type forwardableItem interface {
		Value() any
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			var (
				selectedItem = model.SelectedItem()
				item, ok     = selectedItem.(forwardableItem)
			)

			if s.nextScreenName != "" && ok && item != nil {
				s.next(s.nextScreenName, item.Value())
			}
		}
	}

	return nil
}
