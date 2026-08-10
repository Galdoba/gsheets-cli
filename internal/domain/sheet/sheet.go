package sheet

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Galdoba/gsheets-cli/internal/domain/cell"
	"github.com/Galdoba/gsheets-cli/internal/domain/render"
	"google.golang.org/api/sheets/v4"
)

// SheetCache holds a local snapshot of a Google Sheet.
// Grid is a dense 2‑D slice of cells indexed by [row][col] (0‑based).
// An absent cell is represented by the zero value of cell.Cell (Row == 0).
type SheetCache struct {
	SpreadsheetTitle string        `json:"spreadsheet_title"`
	SheetName        string        `json:"sheet_name"`
	RevisionID       string        `json:"revision_id"`
	LastSync         time.Time     `json:"last_sync"`
	Rows             int           `json:"rows"`
	Cols             int           `json:"cols"`
	Grid             [][]cell.Cell `json:"grid"`
}

// New creates an empty cache with no grid data.
func New(title, name string) *SheetCache {
	return &SheetCache{
		SpreadsheetTitle: title,
		SheetName:        name,
		Grid:             [][]cell.Cell{},
	}
}

// UpdateGridData fully replaces the local cache with the latest remote sheet.
// This matches your “always pull fresh before push” workflow.
func (sc *SheetCache) UpdateGridData(sheet *sheets.Sheet) {
	sc.Grid = parseGridData(sheet)
	sc.LastSync = time.Now()
	sc.UpdateDimentions()
}

// UpdateDimentions recalculates Rows and Cols from the current grid.
func (sc *SheetCache) UpdateDimentions() {
	if len(sc.Grid) == 0 {
		sc.Rows = 0
		sc.Cols = 0
		return
	}
	sc.Rows = len(sc.Grid)
	if sc.Rows > 0 {
		sc.Cols = len(sc.Grid[0])
	}
}

// ---------------------------------------------------------------------------
// Public CRUD (kept for programmatic local manipulation)
// ---------------------------------------------------------------------------

// CreateCell adds a new cell to the cache. Returns an error if the position
// is already occupied by a non‑absent cell.
// func (sc *SheetCache) CreateCell(c cell.Cell) error {
// 	if err := c.Validate(); err != nil {
// 		return fmt.Errorf("can't create invalid cell: %w", err)
// 	}
// 	rowIdx, colIdx := c.Row-1, c.Col-1
// 	if rowIdx < 0 || colIdx < 0 {
// 		return fmt.Errorf("invalid cell position")
// 	}
// 	// Expand grid if needed
// 	sc.ensureGridSize(c.Row, c.Col)
// 	if sc.Grid[rowIdx][colIdx].Row != 0 {
// 		return fmt.Errorf("cell %q already exists", c.A1)
// 	}
// 	sc.Grid[rowIdx][colIdx] = c
// 	sc.UpdateDimentions()
// 	return nil
// }

// ReadCell returns the cell at A1 notation and true if it exists.
// func (sc *SheetCache) ReadCell(a1 string) (cell.Cell, bool) {
// 	row, col, err := cell.A1ToPosition(a1)
// 	if err != nil {
// 		return cell.Cell{}, false
// 	}
// 	rowIdx, colIdx := row-1, col-1
// 	if rowIdx < len(sc.Grid) && colIdx < len(sc.Grid[rowIdx]) {
// 		c := sc.Grid[rowIdx][colIdx]
// 		if c.Row != 0 {
// 			return c, true
// 		}
// 	}
// 	return cell.Cell{}, false
// }

// UpdateCell modifies an existing cell. Returns an error if the cell does not
// exist at that position.
// func (sc *SheetCache) UpdateCell(c cell.Cell) error {
// 	if err := c.Validate(); err != nil {
// 		return fmt.Errorf("cell invalid: %w", err)
// 	}
// 	rowIdx, colIdx := c.Row-1, c.Col-1
// 	if rowIdx < 0 || colIdx < 0 {
// 		return fmt.Errorf("invalid cell position")
// 	}
// 	if rowIdx >= len(sc.Grid) || colIdx >= len(sc.Grid[rowIdx]) {
// 		return fmt.Errorf("cell %q does not exist", c.A1)
// 	}
// 	if sc.Grid[rowIdx][colIdx].Row == 0 {
// 		return fmt.Errorf("cell %q does not exist", c.A1)
// 	}
// 	sc.Grid[rowIdx][colIdx] = c
// 	return nil
// }

// // Delete removes a cell from the cache. After deletion the position is
// // considered empty. Returns an error if the cell was not present.
// func (sc *SheetCache) Delete(a1 string) error {
// 	row, col, err := cell.A1ToPosition(a1)
// 	if err != nil {
// 		return fmt.Errorf("cell %q does not exist", a1)
// 	}
// 	rowIdx, colIdx := row-1, col-1
// 	if rowIdx >= len(sc.Grid) || colIdx >= len(sc.Grid[rowIdx]) {
// 		return fmt.Errorf("cell %q does not exist", a1)
// 	}
// 	if sc.Grid[rowIdx][colIdx].Row == 0 {
// 		return fmt.Errorf("cell %q does not exist", a1)
// 	}
// 	sc.Grid[rowIdx][colIdx] = cell.Cell{} // mark as absent
// 	return nil
// }

// GetCell returns the cell at the given 1‑based row and column.
func (sc *SheetCache) GetCell(row, col int) cell.Cell {
	rowIdx, colIdx := row-1, col-1
	if rowIdx >= 0 && rowIdx < len(sc.Grid) && colIdx >= 0 && colIdx < len(sc.Grid[rowIdx]) {
		return sc.Grid[rowIdx][colIdx]
	}
	return cell.Cell{}
}

// ---------------------------------------------------------------------------
// Render support (render.DataTable)
// ---------------------------------------------------------------------------

var _ render.DataTable = (*SheetCache)(nil)

func (sc *SheetCache) RowCount() int { return sc.Rows }
func (sc *SheetCache) ColCount() int { return sc.Cols }

func (sc *SheetCache) CellValue(row, col int) string {
	return sc.GetCell(row+1, col+1).Value
}

func (sc *SheetCache) CellNote(row, col int) string {
	return sc.GetCell(row+1, col+1).Note
}

func (sc *SheetCache) ColName(col int) string {
	return cell.ColIndexToLetter(col)
}

// RowCells returns all cells in the given 0‑based row.
// Zero-allocation: returns a slice header pointing to the underlying grid.
func (sc *SheetCache) RowCells(row int) []cell.Cell {
	if row < 0 || row >= len(sc.Grid) {
		return nil
	}
	return sc.Grid[row]
}

func (sc *SheetCache) Column(col int) []cell.Cell {
	if len(sc.Grid) <= 0 || col < 0 {
		return []cell.Cell{}
	}
	column := make([]cell.Cell, len(sc.Grid[0]))
	for _, row := range sc.Grid {
		for c, cell := range row {
			if c != col {
				continue
			}
			column = append(column, cell)
		}
	}
	return column
}

// ---------------------------------------------------------------------------
// Persistence
// ---------------------------------------------------------------------------

func (sc *SheetCache) SaveAs(path string) error {
	data, err := json.MarshalIndent(sc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate file: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Google Sheets API helpers
// ---------------------------------------------------------------------------

// parseGridData builds a dense 2‑D grid from a *sheets.Sheet.
// It uses GridProperties to allocate the correct dimensions and fills in
// only the cells returned by the API. All absent cells remain zero values.
// Function gracefully handles missing GridProperties by calculating dimensions from the payload.
func parseGridData(sheet *sheets.Sheet) [][]cell.Cell {
	var rows, cols int

	// 1. Attempt to read dimensions from GridProperties if available
	if sheet.Properties != nil && sheet.Properties.GridProperties != nil {
		rows = int(sheet.Properties.GridProperties.RowCount)
		cols = int(sheet.Properties.GridProperties.ColumnCount)
	}

	// 2. If GridProperties was missing (due to API field restrictions) or returned 0,
	// calculate the max bounds dynamically from the returned data blocks.
	if rows == 0 || cols == 0 {
		for _, data := range sheet.Data {
			startRow := int(data.StartRow)
			startCol := int(data.StartColumn)

			// Calculate max rows needed
			if numRows := startRow + len(data.RowData); numRows > rows {
				rows = numRows
			}

			// Calculate max cols needed
			for _, row := range data.RowData {
				if numCols := startCol + len(row.Values); numCols > cols {
					cols = numCols
				}
			}
		}
	}

	// If the sheet is genuinely empty, return an empty grid
	if rows == 0 || cols == 0 {
		return [][]cell.Cell{}
	}

	// 3. Allocate the dense 2D grid
	grid := make([][]cell.Cell, rows)
	for i := range grid {
		grid[i] = make([]cell.Cell, cols)
	}

	// 4. Populate the grid
	for _, data := range sheet.Data {
		startRow := int(data.StartRow)
		startCol := int(data.StartColumn)
		for rowIdx, row := range data.RowData {
			absRow := startRow + rowIdx + 1 // 1-based
			if absRow > rows {
				continue
			}
			for colIdx, cellData := range row.Values {
				absCol := startCol + colIdx + 1 // 1-based
				if absCol > cols {
					continue
				}
				grid[absRow-1][absCol-1] = cell.Cell{
					A1:     cell.PositionToA1(absRow, absCol),
					Row:    absRow,
					Col:    absCol,
					Value:  extractValue(cellData),
					Note:   extractNote(cellData),
					Format: extractFormat(cellData),
				}
			}
		}
	}
	return grid
}

// extractValue prefers the formatted display value, falls back to raw.
func extractValue(cell *sheets.CellData) string {
	if cell.FormattedValue != "" {
		return cell.FormattedValue
	}
	if cell.UserEnteredValue != nil {
		if cell.UserEnteredValue.StringValue != nil {
			return *cell.UserEnteredValue.StringValue
		}
		return fmt.Sprintf("%v", cell.UserEnteredValue)
	}
	return ""
}

func extractNote(cell *sheets.CellData) string {
	if cell.Note != "" {
		return cell.Note
	}
	return ""
}

func extractFormat(cell *sheets.CellData) string {
	if cell.EffectiveFormat != nil &&
		cell.EffectiveFormat.NumberFormat != nil &&
		cell.EffectiveFormat.NumberFormat.Type != "" {
		return cell.EffectiveFormat.NumberFormat.Type
	}
	return ""
}

// ---------------------------------------------------------------------------
// A1 notation helpers (internal)
// ---------------------------------------------------------------------------

// ensureGridSize is a tiny helper used only by CreateCell to grow the grid
// when a cell is added beyond the current bounds.
// func (sc *SheetCache) ensureGridSize(row, col int) {
// 	// Expand columns in all existing rows if necessary
// 	for len(sc.Grid[0]) < col {
// 		for i := range sc.Grid {
// 			sc.Grid[i] = append(sc.Grid[i], cell.Cell{})
// 		}
// 	}
// 	// Add missing rows
// 	for len(sc.Grid) < row {
// 		newRow := make([]cell.Cell, max(col, sc.Cols))
// 		sc.Grid = append(sc.Grid, newRow)
// 	}
// 	// Keep Columns consistent
// 	if sc.Cols < col {
// 		sc.Cols = col
// 	}
// }
