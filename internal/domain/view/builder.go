package view

import (
	"charm.land/lipgloss/v2"
	"github.com/Galdoba/gsheets-cli/internal/domain/render"
)

// BuildRenderConfig translates domain view configurations into a render.Config
func BuildRenderConfig(data render.DataTable, preset *Preset, rowCfg *RowConfig) render.Config {
	cfg := render.Config{
		Border:          preset.Charset.Border,
		CollapsedMarker: preset.Charset.Ellipsis,
		HeaderStyle:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Background(lipgloss.Color("236")),
		RowStyle:        lipgloss.NewStyle(),
	}

	// 1. Build Columns
	// We iterate through the physical columns in the DataTable.
	// If the Preset has a rule for this column, we use it. Otherwise, we use defaults.
	for i := 0; i < data.ColCount(); i++ {
		viewCol, hasRule := preset.Columns[i]

		var rCol render.ColumnConfig
		rCol.OriginalIndex = i

		if hasRule {
			rCol.Visibility = mapVisibility(viewCol.Visibility)
			rCol.WidthMode = mapWidthMode(viewCol.WidthMode)
			rCol.FixedWidth = viewCol.WidthValue

			// Map Alignment
			if viewCol.AlignRight {
				rCol.Align = lipgloss.Right
			} else {
				rCol.Align = lipgloss.Left
			}

			// Map Note Mode (Assuming you add NoteMode to view.ColumnConfig,
			// otherwise default to NoteHide for now)
			rCol.NoteMode = render.NoteHide
		} else {
			// Defaults for columns not explicitly configured in the Preset
			rCol.Visibility = render.Visible
			rCol.WidthMode = render.WidthMaxAll
			rCol.Align = lipgloss.Left
		}

		cfg.Columns = append(cfg.Columns, rCol)
	}

	// 2. Build Row Filter
	cfg.RowFilter = buildRowFilter(rowCfg)

	return cfg
}

// mapVisibility translates view.ColumnVisibility to render.Visibility
func mapVisibility(v ColumnVisibility) render.Visibility {
	switch v {
	case ColVisible:
		return render.Visible
	case ColHidden:
		return render.Hidden
	case ColCollapsedShort, ColCollapsedLong:
		return render.Collapsed
	default:
		return render.Visible
	}
}

// mapWidthMode translates view.WidthMode to render.WidthMode
func mapWidthMode(w WidthMode) render.WidthMode {
	switch w {
	case WidthMax:
		return render.WidthMaxAll
	case WidthMin:
		return render.WidthMinAll
	case WidthFixed:
		return render.WidthFixed
	default:
		return render.WidthMaxAll
	}
}

// buildRowFilter generates the predicate function for row filtering
func buildRowFilter(rowCfg *RowConfig) func(int, render.DataTable) bool {
	if rowCfg == nil {
		return nil // No filtering
	}

	return func(row int, data render.DataTable) bool {
		// 1. Check explicit row states
		if state, ok := rowCfg.States[row]; ok {
			if state == RowCollapsed { // Hidden inside a group
				return false
			}
		}

		// 2. Check if row belongs to a collapsed group
		for _, group := range rowCfg.Groups {
			if group.Collapsed {
				for _, containedRow := range group.ContainedRows {
					if containedRow == row {
						return false // Filter out rows in collapsed groups
					}
				}
			}
		}

		return true
	}
}
