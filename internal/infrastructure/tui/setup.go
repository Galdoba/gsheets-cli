package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/Galdoba/gsheets-cli/internal/domain/profile"
	"github.com/Galdoba/gsheets-cli/internal/domain/sheet"
	"github.com/Galdoba/gsheets-cli/internal/infrastructure/config"
	persistience "github.com/Galdoba/gsheets-cli/internal/infrastructure/persistence"
)

type Stage int

const (
	StageTableSelection Stage = iota
	StageProfileSelection
	StageDataFetch
	StageViewer
)

type TableOption struct {
	SpreadsheetID string
	SheetName     string
	Display       string
}

type SetupModel struct {
	cfg   config.Config
	repo  profile.Repository
	stage Stage

	// table selection state
	tables        []TableOption
	tableCursor   int
	selectedTable TableOption

	// profile selection state (filled later)
	profiles        []profile.Profile
	profileCursor   int
	selectedProfile *profile.Profile
}

// NewSetupModel creates the setup model and initialises the stage based on provided flags.
func NewSetupModel(cfg config.Config, initialTableID, initialSheetName, initialProfileName string) SetupModel {
	m := SetupModel{
		cfg:  cfg,
		repo: persistience.NewProfiles(),
	}

	if initialTableID != "" && initialSheetName != "" {
		m.selectedTable = TableOption{
			SpreadsheetID: sheet.ExtractSheetID(initialTableID),
			SheetName:     initialSheetName,
			Display:       fmt.Sprintf("%s / %s", sheet.ExtractSheetID(initialTableID), initialSheetName),
		}
		m.stage = StageProfileSelection
		// If profile flag is also provided, we could skip profile selection later.
		// For now, we still show profile selection (we'll handle in next step).
	} else {
		m.loadTables()
		m.stage = StageTableSelection
	}
	return m
}

// loadTables fills the tables slice from config.
func (m *SetupModel) loadTables() {
	// Start with last used table if present
	if m.cfg.Sheets.LastUsed.TableID != "" && m.cfg.Sheets.LastUsed.SheetName != "" {
		m.tables = append(m.tables, TableOption{
			SpreadsheetID: m.cfg.Sheets.LastUsed.TableID,
			SheetName:     m.cfg.Sheets.LastUsed.SheetName,
			Display:       fmt.Sprintf("Last used: %s / %s", m.cfg.Sheets.LastUsed.TableID, m.cfg.Sheets.LastUsed.SheetName),
		})
	}
	// Add tables from config aliases
	for alias, table := range m.cfg.Sheets.Tables {
		addr := table.Address
		// Extract ID from URL if needed
		id := sheet.ExtractSheetID(addr)
		for _, sheetName := range table.SheetsNames {
			m.tables = append(m.tables, TableOption{
				SpreadsheetID: id,
				SheetName:     sheetName,
				Display:       fmt.Sprintf("%s: %s / %s", alias, id, sheetName),
			})
		}
	}
	// If still empty, add a placeholder to indicate no tables found
	if len(m.tables) == 0 {
		m.tables = append(m.tables, TableOption{
			SpreadsheetID: "",
			SheetName:     "",
			Display:       "No tables configured. Please run fetch first or edit config.",
		})
	}
}

// Init is part of the tea.Model interface.
func (m SetupModel) Init() tea.Cmd {
	return nil
}

// Update handles messages depending on the current stage.
func (m SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.stage {
	case StageTableSelection:
		return m.updateTableSelection(msg)
	case StageProfileSelection:
		// Not implemented yet; we'll return a placeholder.
		return m, nil
	case StageDataFetch:
		return m, nil
	case StageViewer:
		return m, nil
	}
	return m, nil
}

// View renders the current stage.
func (m SetupModel) View() tea.View {
	s := ""
	switch m.stage {
	case StageTableSelection:
		s = m.renderTableSelection()
	case StageProfileSelection:
		s = "Profile selection not implemented yet.\nPress q to quit."
	case StageDataFetch:
		s = "Loading…"
	case StageViewer:
		s = "Viewer not implemented yet."
	}

	v := tea.NewView(s)
	v.AltScreen = true // Replaces tea.WithAltScreen()
	return v
}

// updateTableSelection handles input for the table selection stage.
func (m SetupModel) updateTableSelection(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.tableCursor > 0 {
				m.tableCursor--
			}
		case "down", "j":
			if m.tableCursor < len(m.tables)-1 {
				m.tableCursor++
			}
		case "enter", " ":
			if len(m.tables) > 0 {
				m.selectedTable = m.tables[m.tableCursor]
				m.stage = StageProfileSelection
				// We'll load profiles in the next step
			}
		}
	}
	return m, nil
}

// renderTableSelection displays the list of tables with the cursor.
func (m SetupModel) renderTableSelection() string {
	s := "Select a table:\n\n"
	for i, t := range m.tables {
		cursor := " "
		if i == m.tableCursor {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %s\n", cursor, t.Display)
	}
	s += "\n↑/↓ to navigate, Enter to select, q to quit."
	return s
}
