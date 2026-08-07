package profile

import (
	"github.com/Galdoba/gsheets-cli/pkg/filter"
)

// Profile represents a complete, saved rendering configuration for a specific table.
type Profile struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SheetID   string `json:"sheet_id"`
	TableName string `json:"table_name"`

	Layout  LayoutConfig `json:"layout"`
	Filters filter.Group `json:"filters"`
}

// LayoutConfig dictates the structural presentation of columns.
// The order of the Columns slice implicitly defines the visual column order.
type LayoutConfig struct {
	Columns []ColumnState `json:"columns"`
}

// ColumnState defines the rendering rules for a specific physical column.
type ColumnState struct {
	OriginalIndex int        `json:"original_index"`
	Visibility    Visibility `json:"visibility"`
	WidthMode     WidthMode  `json:"width_mode"`
	FixedWidth    int        `json:"fixed_width,omitempty"`
	AlignRight    bool       `json:"align_right"`
}

// Visibility defines how a column should be presented
type Visibility int

const (
	Visible Visibility = iota
	Hidden
	Collapsed
)

// WidthMode defines how the column width is calculated
type WidthMode int

const (
	WidthAuto WidthMode = iota
	WidthFixed
)
