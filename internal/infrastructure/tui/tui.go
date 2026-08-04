package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/Galdoba/gsheets-cli/internal/domain/render"
)

type Model struct {
	data           render.DataTable
	cfg            render.Config
	viewportStart  int
	viewportHeight int
}

func NewModel(data render.DataTable, cfg render.Config) Model {
	return Model{
		data:           data,
		cfg:            cfg,
		viewportStart:  0,
		viewportHeight: 20, // Safe default before WindowSizeMsg arrives
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// In v2, KeyMsg is an interface. We use KeyPressMsg for standard key presses.
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "j", "down":
			maxScroll := max(m.data.RowCount()-m.viewportHeight, 0)
			if m.viewportStart < maxScroll {
				m.viewportStart++
			}
		case "k", "up":
			if m.viewportStart > 0 {
				m.viewportStart--
			}
		}

	case tea.WindowSizeMsg:
		// Reserve 2 lines for the status bar at the bottom
		m.viewportHeight = max(msg.Height-2, 1)
	}
	return m, nil
}

func (m Model) View() tea.View {
	// 1. Delegate the heavy lifting entirely to your render engine
	tableStr := render.Render(m.data, m.cfg, m.viewportStart, m.viewportHeight)

	// 2. Add a simple status bar to verify viewport math
	status := fmt.Sprintf("\n Total Rows: %d | Viewing: %d-%d | Use ↑/↓ to scroll, 'q' to quit ",
		m.data.RowCount(),
		m.viewportStart+1,
		m.viewportStart+m.viewportHeight)

	// 3. Declare the view and terminal state declaratively
	v := tea.NewView(tableStr + status)
	v.AltScreen = true // Replaces tea.WithAltScreen()

	return v
}
