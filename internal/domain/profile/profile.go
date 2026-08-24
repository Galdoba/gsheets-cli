package profile

import (
	"fmt"
	"strings"

	"github.com/Galdoba/gsheets-cli/pkg/filter"
)

// Profile represents a complete, saved rendering configuration for a specific table.
type Profile struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Layout      LayoutConfig `json:"layout"`
	Filters     filter.Group `json:"filters"`
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

// Validate ensures the profile is safe to save and semantically coherent.
func (p *Profile) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("profile name cannot be empty")
	}
	if strings.ContainsAny(p.Name, `/\`) {
		return fmt.Errorf("profile name cannot contain path separators")
	}
	if p.Name == "." || p.Name == ".." {
		return fmt.Errorf("profile name cannot be '.' or '..'")
	}

	for i, col := range p.Layout.Columns {
		if col.OriginalIndex < 0 {
			return fmt.Errorf("column %d: OriginalIndex must be >= 0", i)
		}
		switch col.Visibility {
		case Visible, Hidden, Collapsed:
			// ok
		default:
			return fmt.Errorf("column %d: invalid visibility value", i)
		}
		switch col.WidthMode {
		case WidthAuto, WidthFixed:
			// ok
		default:
			return fmt.Errorf("column %d: invalid width mode", i)
		}
		if col.WidthMode == WidthFixed && col.FixedWidth <= 0 {
			return fmt.Errorf("column %d: FixedWidth must be > 0 when WidthMode is Fixed", i)
		}
	}
	return nil
}

func DefaultProfile() *Profile {
	return &Profile{
		Name:        "default",
		Description: "This is a default profile with no filters attached",
		Layout:      LayoutConfig{Columns: []ColumnState{}},
		Filters:     filter.Group{Logic: "and", Rules: []filter.Rule{}},
	}
}
