package render

import lipgloss "charm.land/lipgloss/v2"

// Visibility defines how a column should be presented
type Visibility int

const (
	Visible   Visibility = iota
	Hidden               // Completely removed from the table
	Collapsed            // Replaced by a marker (e.g., "…") to indicate hidden data
)

// WidthMode defines how the column width is calculated
type WidthMode int

const (
	WidthFixed       WidthMode = iota // Fixed width defined in config
	WidthMaxAll                       // Longest cell in the entire dataset
	WidthMinAll                       // Shortest cell in the entire dataset
	WidthMaxFiltered                  // Longest cell among filtered rows
	WidthMinFiltered                  // Shortest cell among filtered rows
	WidthMaxVisible                   // Longest cell among rows currently in viewport
	WidthMinVisible                   // Shortest cell among rows currently in viewport
)

// NoteMode defines how cell notes are presented to the user
type NoteMode int

const (
	NoteHide    NoteMode = iota // Show no indication
	NoteMark                    // Append a marker (e.g., "*") if note exists
	NoteAppend                  // Append note text (e.g., " [Note text]")
	NoteReplace                 // Replace cell value with note text
)

// ColumnConfig holds the rendering rules for a specific column
type ColumnConfig struct {
	OriginalIndex int // The 0-based index in the underlying DataTable
	Visibility    Visibility
	WidthMode     WidthMode
	FixedWidth    int // Used for WidthFixed and Collapsed modes

	// Presentation
	Align     lipgloss.Position
	CellStyle lipgloss.Style
	NoteMode  NoteMode
}

// Config is the master blueprint for rendering a specific view of the data
type Config struct {
	Columns         []ColumnConfig
	RowFilter       func(row int, data DataTable) bool
	CollapsedMarker string // Used for single collapsed columns

	// Styles
	HeaderStyle lipgloss.Style
	RowStyle    lipgloss.Style
	TableStyle  lipgloss.Style // Applied to the final table container
	Border      bool           // Toggles grid lines
}
