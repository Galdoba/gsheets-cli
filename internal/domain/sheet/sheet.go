package sheet

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Galdoba/gsheets-cli/internal/domain/cell"
	"github.com/Galdoba/gsheets-cli/internal/domain/render"
	"github.com/Galdoba/gsheets-cli/internal/domain/rowmatch"
	"google.golang.org/api/sheets/v4"
)

// SheetCache holds a local snapshot of a Google Sheet.
// Grid is a dense 2‑D slice of cells indexed by [row][col] (0‑based).
// An absent cell is represented by the zero value of cell.Cell (Row == 0).
type SheetCache struct {
	SpreadsheetTitle string           `json:"spreadsheet_title"`
	SheetName        string           `json:"sheet_name"`
	RevisionID       string           `json:"revision_id"`
	LastSync         time.Time        `json:"last_sync"`
	Rows             int              `json:"rows"`
	Cols             int              `json:"cols"`
	Grid             [][]cell.Cell    `json:"grid"`
	MatchRules       *rowmatch.Config `json:"match_rules"`
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

//HELPERS

// ExtractSheetID pulls the ID from a full Google Sheets URL or returns it as-is
func ExtractSheetID(input string) string {
	if _, after, ok := strings.Cut(input, "/d/"); ok {
		rest := after
		end := strings.Index(rest, "/")
		if end == -1 {
			end = len(rest)
		}
		return rest[:end]
	}
	return input
}

//safeguard related

func (sc *SheetCache) SetMatchConfig(cfg *rowmatch.Config) error {
	if cfg == nil || len(cfg.Rules) == 0 {
		return fmt.Errorf("match config is empty")
	}
	for _, r := range cfg.Rules {
		if r.Column < 1 {
			return fmt.Errorf("column must be >= 1")
		}
		if r.Weight < 0 || r.Weight > 1 {
			return fmt.Errorf("weight must be in [0,1]")
		}
	}
	sc.MatchRules = cfg
	return nil
}

func (sc *SheetCache) Fingerprint(rowIdx int, config *rowmatch.Config) (rowmatch.Fingerprint, error) {
	if rowIdx < 0 || rowIdx >= sc.Rows {
		return rowmatch.Fingerprint{}, fmt.Errorf("row index %d out of range (0..%d)", rowIdx, sc.Rows-1)
	}
	if config == nil {
		return rowmatch.Fingerprint{}, fmt.Errorf("nil config")
	}
	values := make(map[int]string)
	for _, rule := range config.Rules {
		if rule.Column < 1 || rule.Column > sc.Cols {
			return rowmatch.Fingerprint{}, fmt.Errorf("column %d out of range (1..%d)", rule.Column, sc.Cols)
		}
		c := sc.GetCell(rowIdx+1, rule.Column)
		val := c.Value
		if c.OriginalValue != "" {
			val = c.OriginalValue // берём старую версию для идентификации
		}
		values[rule.Column] = val
	}
	return rowmatch.NewFingerprint(rowIdx, values), nil
}

func (sc *SheetCache) ComputeFingerprints() ([]rowmatch.Fingerprint, error) {
	if sc.MatchRules == nil || len(sc.MatchRules.Rules) == 0 {
		return nil, fmt.Errorf("match config is empty")
	}
	fps := make([]rowmatch.Fingerprint, 0, sc.Rows)
	for rowIdx := 0; rowIdx < sc.Rows; rowIdx++ {
		fp, err := sc.Fingerprint(rowIdx, sc.MatchRules)
		if err != nil {
			return nil, fmt.Errorf("failed to compute fingerprint for row %d: %w", rowIdx+1, err)
		}
		fps = append(fps, fp)
	}
	return fps, nil
}

// Clone создаёт полную копию кэша (глубокая копия Grid).
func (sc *SheetCache) Clone() *SheetCache {
	clone := *sc
	clone.Grid = make([][]cell.Cell, len(sc.Grid))
	for i := range sc.Grid {
		clone.Grid[i] = make([]cell.Cell, len(sc.Grid[i]))
		copy(clone.Grid[i], sc.Grid[i])
	}
	if sc.MatchRules != nil {
		cfgCopy := *sc.MatchRules
		clone.MatchRules = &cfgCopy
	}
	return &clone
}

// CellChange описывает одно локальное изменение (для использования в sheet).
type CellChange struct {
	Row   int
	Col   int
	Field string // "value" или "note"
	Value string
}

// ApplyChanges применяет список изменений к кэшу. Изменяемые ячейки сохраняют
// исходные значения в OriginalValue/OriginalNote.
func (sc *SheetCache) ApplyChanges(changes []CellChange) error {
	for _, ch := range changes {
		if ch.Row < 1 || ch.Col < 1 {
			return fmt.Errorf("invalid cell position: row=%d col=%d", ch.Row, ch.Col)
		}
		c := sc.GetCell(ch.Row, ch.Col)
		if c.Row == 0 {
			return fmt.Errorf("cell %s does not exist", cell.PositionToA1(ch.Row, ch.Col))
		}
		switch strings.ToLower(ch.Field) {
		case "", "value":
			c.SetValue(ch.Value)
		case "note":
			c.SetNote(ch.Value)
		default:
			return fmt.Errorf("unsupported field %q", ch.Field)
		}
		sc.Grid[ch.Row-1][ch.Col-1] = c
	}
	return nil
}
