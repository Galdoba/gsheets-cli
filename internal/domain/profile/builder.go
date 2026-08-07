package profile

import (
	"fmt"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/Galdoba/gsheets-cli/internal/domain/cell"
	"github.com/Galdoba/gsheets-cli/internal/domain/render"
	"github.com/Galdoba/gsheets-cli/pkg/filter"
)

// BuildRenderConfig translates a saved Profile into an actionable render.Config.
func BuildRenderConfig(data render.DataTable, p *Profile) (render.Config, error) {
	cfg := render.Config{
		Border:          true, // Can be moved to Profile settings later
		CollapsedMarker: "…",
		HeaderStyle:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Background(lipgloss.Color("236")),
		RowStyle:        lipgloss.NewStyle(),
	}

	// 1. Map Layout (Column Order, Visibility, Widths)
	// If the profile has no columns defined (e.g., a fresh profile), fallback to showing all columns.
	if len(p.Layout.Columns) == 0 {
		for i := 0; i < data.ColCount(); i++ {
			cfg.Columns = append(cfg.Columns, render.ColumnConfig{
				OriginalIndex: i,
				Visibility:    render.Visible,
				WidthMode:     render.WidthMaxAll,
				Align:         lipgloss.Left,
			})
		}
	} else {
		for _, colState := range p.Layout.Columns {
			rCol := render.ColumnConfig{
				OriginalIndex: colState.OriginalIndex,
				Visibility:    mapVisibility(colState.Visibility),
				WidthMode:     mapWidthMode(colState.WidthMode),
				FixedWidth:    colState.FixedWidth,
				Align:         lipgloss.Left,
			}
			if colState.AlignRight {
				rCol.Align = lipgloss.Right
			}
			cfg.Columns = append(cfg.Columns, rCol)
		}
	}

	// 2. Compile Filters
	if len(p.Filters.Rules) > 0 {
		fs := filter.NewFilterSet[cell.Cell]()

		// Register extractors for the cell.Cell domain type
		_ = fs.RegisterStringExtractor("value", func(c cell.Cell) (string, error) { return c.Value, nil })
		_ = fs.RegisterStringExtractor("note", func(c cell.Cell) (string, error) { return c.Note, nil })
		_ = fs.RegisterStringExtractor("format", func(c cell.Cell) (string, error) { return c.Format, nil })
		_ = fs.RegisterStringExtractor("a1", func(c cell.Cell) (string, error) { return c.A1, nil })
		_ = fs.RegisterTimeExtractor("updated_at", func(c cell.Cell) (time.Time, error) { return c.UpdatedAt, nil })

		// Provide column name mapping (e.g., "A" -> 0, "B" -> 1)
		colMap := make(map[string]int)
		for i := 0; i < data.ColCount(); i++ {
			colMap[data.ColName(i)] = i
		}
		fs.SetColumnNameMapping(colMap)

		// Compile the declarative rules into a fast predicate
		rowPredicate, err := fs.Compile(p.Filters)
		if err != nil {
			return render.Config{}, fmt.Errorf("failed to compile filters: %w", err)
		}

		// 3. The Adapter Bridge:
		// Convert func([]cell.Cell) bool -> func(int, render.DataTable) bool
		cfg.RowFilter = func(row int, data render.DataTable) bool {
			cells := data.RowCells(row)
			return rowPredicate(cells)
		}
	}

	return cfg, nil
}

// mapVisibility translates domain Visibility to render Visibility
func mapVisibility(v Visibility) render.Visibility {
	switch v {
	case Hidden:
		return render.Hidden
	case Collapsed:
		return render.Collapsed
	default:
		return render.Visible
	}
}

// mapWidthMode translates domain WidthMode to render WidthMode
func mapWidthMode(w WidthMode) render.WidthMode {
	switch w {
	case WidthFixed:
		return render.WidthFixed
	default:
		return render.WidthMaxAll // Default to Auto (MaxAll)
	}
}
