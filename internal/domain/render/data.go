package render

import "github.com/Galdoba/gsheets-cli/internal/domain/cell"

// DataTable abstracts the local storage layer for the renderer
type DataTable interface {
	RowCount() int
	ColCount() int
	CellValue(row, col int) string
	CellNote(row, col int) string
	ColName(col int) string // Returns "A", "B", "C", etc.
	RowCells(row int) []cell.Cell
}
