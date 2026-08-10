package render

import (
	lipgloss "charm.land/lipgloss/v2"
	"fmt"
)

// renderColumn represents the resolved state of a column for the rendering engine
type renderColumn struct {
	OriginalIndex int
	Visibility    Visibility
	IsGrouped     bool
	GroupSize     int
	Config        ColumnConfig
}

// resolveColumns scans the raw config and groups contiguous Collapsed columns
func resolveColumns(cfg Config) []renderColumn {
	var resolved []renderColumn

	for i := 0; i < len(cfg.Columns); i++ {
		col := cfg.Columns[i]

		if col.Visibility == Hidden {
			continue // Completely skip hidden columns
		}

		if col.Visibility == Collapsed {
			// Count contiguous collapsed columns
			count := 1
			for j := i + 1; j < len(cfg.Columns); j++ {
				if cfg.Columns[j].Visibility == Collapsed {
					count++
				} else {
					break
				}
			}

			// Create a single grouped column
			resolved = append(resolved, renderColumn{
				Visibility: Collapsed,
				IsGrouped:  true,
				GroupSize:  count,
				Config:     col, // Inherit styling from the first collapsed column
			})

			// Advance the loop index to skip the columns we just grouped
			i += (count - 1)
		} else {
			// Normal visible column
			resolved = append(resolved, renderColumn{
				OriginalIndex: col.OriginalIndex,
				Visibility:    Visible,
				Config:        col,
			})
		}
	}

	return resolved
}

// computeColumnWidths now operates on the resolved columns
// func computeColumnWidths(data DataTable, resolved []renderColumn, cfg Config, filteredRows, visibleRows []int) []int {
// 	widths := make([]int, len(resolved))

// 	for i, rc := range resolved {
// 		if rc.Visibility == Collapsed {
// 			// Handle width for collapsed/grouped columns
// 			if rc.Config.WidthMode == WidthFixed && rc.Config.FixedWidth > 0 {
// 				widths[i] = rc.Config.FixedWidth
// 			} else {
// 				// Default to "+N" marker, or the single CollapsedMarker if size is 1
// 				marker := fmt.Sprintf("+%d", rc.GroupSize)
// 				if rc.GroupSize == 1 && cfg.CollapsedMarker != "" {
// 					marker = cfg.CollapsedMarker
// 				}
// 				widths[i] = lipgloss.Width(marker)
// 			}
// 			if widths[i] < 1 {
// 				widths[i] = 1
// 			}
// 			continue
// 		}

// 		// --- Existing logic for Visible columns ---
// 		if rc.Config.WidthMode == WidthFixed {
// 			widths[i] = rc.Config.FixedWidth
// 			continue
// 		}

// 		var scanRows []int
// 		switch rc.Config.WidthMode {
// 		case WidthMaxAll, WidthMinAll:
// 			scanRows = makeRange(0, data.RowCount())
// 		case WidthMaxFiltered, WidthMinFiltered:
// 			scanRows = filteredRows
// 		case WidthMaxVisible, WidthMinVisible:
// 			scanRows = visibleRows
// 		}

// 		maxW, minW := 0, 1000000
// 		for _, r := range scanRows {
// 			// 1. Fetch raw data
// 			val := data.CellValue(r, rc.OriginalIndex)
// 			note := data.CellNote(r, rc.OriginalIndex)

// 			// 2. Process notes/markers BEFORE measuring width
// 			finalContent := processNote(val, note, rc.Config.NoteMode)

// 			// 3. Measure the final rendered string
// 			w := lipgloss.Width(finalContent)
// 			if w > maxW {
// 				maxW = w
// 			}
// 			if w < minW {
// 				minW = w
// 			}
// 		}

// 		switch rc.Config.WidthMode {
// 		case WidthMaxAll, WidthMaxFiltered, WidthMaxVisible:
// 			widths[i] = maxW
// 		case WidthMinAll, WidthMinFiltered, WidthMinVisible:
// 			widths[i] = minW
// 		}

// 		if widths[i] < 1 {
// 			widths[i] = 1
// 		}
// 	}
// 	return widths
// }

func computeColumnWidths(data DataTable, resolved []renderColumn, cfg Config, filteredRows, visibleRows []int) []int {
	widths := make([]int, len(resolved))

	for i, rc := range resolved {
		if rc.Visibility == Collapsed {
			if rc.Config.WidthMode == WidthFixed && rc.Config.FixedWidth > 0 {
				widths[i] = rc.Config.FixedWidth
			} else {
				marker := fmt.Sprintf("+%d", rc.GroupSize)
				if rc.GroupSize == 1 && cfg.CollapsedMarker != "" {
					marker = cfg.CollapsedMarker
				}
				widths[i] = lipgloss.Width(marker)
			}
			if widths[i] < 1 {
				widths[i] = 1
			}
			continue
		}

		if rc.Config.WidthMode == WidthFixed {
			widths[i] = rc.Config.FixedWidth
			continue
		}

		maxW, minW := 0, 1000000

		// Helper closure to measure a single row (inlined for performance)
		measure := func(r int) {
			val := data.CellValue(r, rc.OriginalIndex)
			note := data.CellNote(r, rc.OriginalIndex)
			finalContent := processNote(val, note, rc.Config.NoteMode)
			w := lipgloss.Width(finalContent)
			if w > maxW {
				maxW = w
			}
			if w < minW {
				minW = w
			}
		}

		// Iterate directly without allocating index slices
		switch rc.Config.WidthMode {
		case WidthMaxAll, WidthMinAll:
			for r := 0; r < data.RowCount(); r++ {
				measure(r)
			}
		case WidthMaxFiltered, WidthMinFiltered:
			for _, r := range filteredRows {
				measure(r)
			}
		case WidthMaxVisible, WidthMinVisible:
			for _, r := range visibleRows {
				measure(r)
			}
		}

		switch rc.Config.WidthMode {
		case WidthMaxAll, WidthMaxFiltered, WidthMaxVisible:
			widths[i] = maxW
		case WidthMinAll, WidthMinFiltered, WidthMinVisible:
			widths[i] = minW
		}

		if widths[i] < 1 {
			widths[i] = 1
		}
	}
	return widths
}

func makeRange(min, max int) []int {
	a := make([]int, max-min)
	for i := range a {
		a[i] = min + i
	}
	return a
}
