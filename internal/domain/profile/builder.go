package profile

import (
	"fmt"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/Galdoba/gsheets-cli/internal/domain/cell"
	"github.com/Galdoba/gsheets-cli/internal/domain/render"
	"github.com/Galdoba/gsheets-cli/pkg/filter"
)

// BuildRenderConfig translates a saved, table-independent Profile into an
// actionable render.Config for a given DataTable.
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

		_ = fs.RegisterTimeExtractor("value_date", func(c cell.Cell) (time.Time, error) {
			if c.Value == "" {
				return time.Time{}, fmt.Errorf("empty value")
			}
			t, err := parseCellDate(c.Value)
			return t, err
		})
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

// ---------------------------------------------------------------------------
// Profile Builder and Editor
// ---------------------------------------------------------------------------

// New creates a new profile with a default "show all" layout and no filters.
// The returned profile is ready for further modification via the fluent methods.
func New(name string) *Profile {
	return &Profile{
		Name:        name,
		Description: "",
		Layout:      LayoutConfig{Columns: []ColumnState{}},
		Filters:     filter.Group{Logic: "and", Rules: []filter.Rule{}},
	}
}

// WithDescription sets the profile description and returns the profile for chaining.
func (p *Profile) WithDescription(desc string) *Profile {
	p.Description = desc
	return p
}

// ColumnOption is a functional option for configuring a ColumnState.
type ColumnOption func(*ColumnState)

// WithVisibility sets the visibility of a column.
func WithVisibility(v Visibility) ColumnOption {
	return func(c *ColumnState) { c.Visibility = v }
}

// WithWidthMode sets the width mode of a column.
func WithWidthMode(w WidthMode) ColumnOption {
	return func(c *ColumnState) { c.WidthMode = w }
}

// WithFixedWidth sets a fixed width for a column (only meaningful when WidthMode is WidthFixed).
func WithFixedWidth(width int) ColumnOption {
	return func(c *ColumnState) { c.FixedWidth = width }
}

// WithAlignRight toggles right alignment for a column.
func WithAlignRight(align bool) ColumnOption {
	return func(c *ColumnState) { c.AlignRight = align }
}

// UpsertColumn adds a new column state, or updates an existing one with the same
// OriginalIndex. It applies any given ColumnOption functions to the column.
func (p *Profile) UpsertColumn(originalIndex int, opts ...ColumnOption) *Profile {
	col := ColumnState{
		OriginalIndex: originalIndex,
		Visibility:    Visible,
		WidthMode:     WidthAuto,
	}
	for _, opt := range opts {
		opt(&col)
	}
	// Replace if exists
	for i := range p.Layout.Columns {
		if p.Layout.Columns[i].OriginalIndex == originalIndex {
			p.Layout.Columns[i] = col
			return p
		}
	}
	// Append new
	p.Layout.Columns = append(p.Layout.Columns, col)
	return p
}

// RemoveColumn deletes the column with the given OriginalIndex from the layout.
func (p *Profile) RemoveColumn(originalIndex int) *Profile {
	for i, col := range p.Layout.Columns {
		if col.OriginalIndex == originalIndex {
			p.Layout.Columns = append(p.Layout.Columns[:i], p.Layout.Columns[i+1:]...)
			break
		}
	}
	return p
}

// SetColumnOrder reorders the visual columns according to the given sequence of
// OriginalIndex values. It returns an error if any index is missing or if the
// length does not match the current number of columns.
func (p *Profile) SetColumnOrder(order []int) error {
	if len(order) != len(p.Layout.Columns) {
		return fmt.Errorf("order must contain exactly %d indices", len(p.Layout.Columns))
	}
	colMap := make(map[int]ColumnState, len(p.Layout.Columns))
	for _, col := range p.Layout.Columns {
		colMap[col.OriginalIndex] = col
	}
	newOrder := make([]ColumnState, 0, len(order))
	for _, idx := range order {
		col, ok := colMap[idx]
		if !ok {
			return fmt.Errorf("column with OriginalIndex %d not found", idx)
		}
		newOrder = append(newOrder, col)
	}
	p.Layout.Columns = newOrder
	return nil
}

// SetColumnVisibility is a convenience wrapper for UpsertColumn to set only visibility.
func (p *Profile) SetColumnVisibility(originalIndex int, v Visibility) *Profile {
	return p.UpsertColumn(originalIndex, WithVisibility(v))
}

// SetColumnWidthMode is a convenience wrapper for UpsertColumn to set only width mode.
func (p *Profile) SetColumnWidthMode(originalIndex int, w WidthMode) *Profile {
	return p.UpsertColumn(originalIndex, WithWidthMode(w))
}

// SetColumnFixedWidth is a convenience wrapper for UpsertColumn to set fixed width.
func (p *Profile) SetColumnFixedWidth(originalIndex int, width int) *Profile {
	return p.UpsertColumn(originalIndex, WithWidthMode(WidthFixed), WithFixedWidth(width))
}

// SetColumnAlignRight toggles right alignment for the given column.
func (p *Profile) SetColumnAlignRight(originalIndex int, alignRight bool) *Profile {
	return p.UpsertColumn(originalIndex, WithAlignRight(alignRight))
}

// AddRule appends a new filter rule to the profile's filter group.
func (p *Profile) AddRule(rule filter.Rule) *Profile {
	p.Filters.Rules = append(p.Filters.Rules, rule)
	return p
}

// RemoveRule removes a rule by its index in the Rules slice.
func (p *Profile) RemoveRule(index int) *Profile {
	if index >= 0 && index < len(p.Filters.Rules) {
		p.Filters.Rules = append(p.Filters.Rules[:index], p.Filters.Rules[index+1:]...)
	}
	return p
}

// SetLogic sets the logical operator for the filter group ("and" or "or").
func (p *Profile) SetLogic(logic string) *Profile {
	p.Filters.Logic = logic
	return p
}

// ClearRules removes all filter rules from the profile.
func (p *Profile) ClearRules() *Profile {
	p.Filters.Rules = []filter.Rule{}
	return p
}
