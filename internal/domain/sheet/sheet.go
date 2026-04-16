package sheet

import (
	"encoding/json"
	"fmt"
	"gsheets-cli/internal/domain/cell"
	"maps"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/api/sheets/v4"
)

type SheetCache struct {
	SheetID    string               `json:"sheet_id"`
	SheetName  string               `json:"sheet_name"`
	RevisionID string               `json:"revision_id"`
	LastSync   time.Time            `json:"last_sync"`
	Rows       int                  `json:"rows"`
	Cols       int                  `json:"cols"`
	Cells      map[string]cell.Cell `json:"cells"`
}

func New(id, name string) *SheetCache {
	sc := SheetCache{
		SheetID:   id,
		SheetName: name,
		Cells:     make(map[string]cell.Cell),
	}
	return &sc
}

func (sc *SheetCache) UpdateGridData(sheet *sheets.Sheet) {
	fetched := parseGridData(sheet)
	switch len(fetched) < len(sc.Cells) {
	case false:
		sc.updateBy(fetched)
	case true:
		sc.populateBy(fetched)
	}
}

func (sc *SheetCache) updateBy(fetched map[string]cell.Cell) {
	maxRow := 0
	updated := 0
	for position, newCell := range fetched {
		oldCell := sc.Cells[position]
		maxRow = max(maxRow, newCell.Row)
		if cell.Equal(oldCell, newCell) {
			continue
		}
		newCell.UpdatedAt = time.Now()
		sc.Cells[position] = newCell
		updated++
	}
	sc.Rows = maxRow
	fmt.Println("cells updated:", updated)
}

func (sc *SheetCache) populateBy(fetched map[string]cell.Cell) {
	sc.Cells = make(map[string]cell.Cell)
	// cell.UpdatedAt = time.Now()
	maps.Copy(sc.Cells, fetched)
}

func parseGridData(sheet *sheets.Sheet) map[string]cell.Cell {
	cellMap := make(map[string]cell.Cell)
	for _, grid := range sheet.Data {
		startRow := grid.StartRow    // 0-based API index
		startCol := grid.StartColumn // 0-based API index

		for rowIdx, row := range grid.RowData {
			for colIdx, cellData := range row.Values {
				absRow := int(startRow) + rowIdx + 1 // Convert to 1-based
				absCol := int(startCol) + colIdx + 1

				cell := cell.Cell{
					A1:     fmt.Sprintf("%s", cell.PositionToA1(absRow, absCol)),
					Row:    absRow,
					Col:    absCol,
					Value:  extractValue(cellData),
					Note:   extractNote(cellData),
					Format: extractFormat(cellData),
				}
				cellMap[cell.A1] = cell
			}
		}
	}
	return cellMap
}

// extractValue: prefer formatted (display) value, fallback to raw
func extractValue(cell *sheets.CellData) string {
	if cell.FormattedValue != "" {
		return cell.FormattedValue
	}
	if cell.UserEnteredValue != nil {
		if cell.UserEnteredValue.StringValue != nil {
			return *cell.UserEnteredValue.StringValue
		}
		// Handle other types: number, bool, formula, error
		return fmt.Sprintf("%v", cell.UserEnteredValue)
	}
	return ""
}

// extractNote: yellow sticky note
func extractNote(cell *sheets.CellData) string {
	if cell.Note != "" {
		return cell.Note
	}
	return ""
}

// extractFormat: number format type (CURRENCY, DATE, TEXT, etc.)
func extractFormat(cell *sheets.CellData) string {
	if cell.EffectiveFormat != nil &&
		cell.EffectiveFormat.NumberFormat != nil &&
		cell.EffectiveFormat.NumberFormat.Type != "" {
		return cell.EffectiveFormat.NumberFormat.Type
	}
	return ""
}

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
	if err := os.WriteFile(path, data, 0666); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}
	return nil
}
