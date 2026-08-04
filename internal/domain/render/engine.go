package render

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"fmt"
	"github.com/mattn/go-runewidth"
)

// Render generates the final TUI string for the given viewport
func Render(data DataTable, cfg Config, viewportStart, viewportHeight int) string {
	// 1. Filter Rows
	var filteredRows []int
	for i := 0; i < data.RowCount(); i++ {
		if cfg.RowFilter == nil || cfg.RowFilter(i, data) {
			filteredRows = append(filteredRows, i)
		}
	}

	// 2. Apply Viewport Boundaries
	end := viewportStart + viewportHeight
	if end > len(filteredRows) {
		end = len(filteredRows)
	}
	if viewportStart > len(filteredRows) {
		viewportStart = len(filteredRows)
	}
	if viewportStart > end {
		viewportStart = end
	}
	visibleRows := filteredRows[viewportStart:end]

	// 3. RESOLUTION PHASE: Group contiguous collapsed columns
	resolved := resolveColumns(cfg)

	// 4. Calculate Geometry using resolved columns
	widths := computeColumnWidths(data, resolved, cfg, filteredRows, visibleRows)

	// 5. Initialize Lipgloss Table
	t := table.New()
	if cfg.Border {
		t.Border(lipgloss.NormalBorder())
	} else {
		t.Border(lipgloss.Border{})
	}

	// 6. Build Header Row
	var headerCells []string
	for i, rc := range resolved {
		var content string
		if rc.Visibility == Collapsed {
			if rc.GroupSize == 1 && cfg.CollapsedMarker != "" {
				content = cfg.CollapsedMarker
			} else {
				content = fmt.Sprintf("+%d", rc.GroupSize)
			}
		} else {
			content = data.ColName(rc.OriginalIndex)
		}

		style := cfg.HeaderStyle.Width(widths[i]).Align(rc.Config.Align)
		headerCells = append(headerCells, style.Render(truncate(content, widths[i])))
	}
	t.Row(headerCells...)

	// 7. Build Data Rows
	for _, rIdx := range visibleRows {
		var cells []string
		for i, rc := range resolved {
			var content string

			if rc.Visibility == Collapsed {
				// Render grouped/empty data cells subtly
				content = "·" // Or leave empty "" if you prefer
			} else {
				content = data.CellValue(rIdx, rc.OriginalIndex)
				note := data.CellNote(rIdx, rc.OriginalIndex)
				content = processNote(content, note, rc.Config.NoteMode)
			}

			style := cfg.RowStyle.Width(widths[i]).Align(rc.Config.Align)

			// Apply specific cell styling if defined
			// (Note: Lipgloss style inheritance can be tricky, explicit overrides are safer)
			if rc.Config.CellStyle.GetBackground() != lipgloss.Color("") || rc.Config.CellStyle.GetForeground() != lipgloss.Color("") {
				style = style.Inherit(rc.Config.CellStyle)
			}

			cells = append(cells, style.Render(truncate(content, widths[i])))
		}
		t.Row(cells...)
	}

	// 8. Final Assembly
	return cfg.TableStyle.Render(t.String())
}

// truncate safely cuts strings to fit exact terminal widths, adding an ellipsis if needed
func truncate(s string, maxWidth int) string {
	if runewidth.StringWidth(s) <= maxWidth {
		return s
	}
	if maxWidth <= 1 {
		return runewidth.Truncate(s, maxWidth, "")
	}
	return runewidth.Truncate(s, maxWidth-1, "…")
}
