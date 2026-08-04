package main

import (
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/Galdoba/gsheets-cli/internal/domain/cell"
	"github.com/Galdoba/gsheets-cli/internal/domain/sheet"
	"github.com/Galdoba/gsheets-cli/internal/domain/view"
	"github.com/Galdoba/gsheets-cli/internal/infrastructure/tui"
)

func main() {
	// 1. Initialize Mock Data
	cache := sheet.New("Рабочая Таблица", "График работ 2.0")
	fmt.Println(cache.SheetName, cache.SpreadsheetTitle)
	for r, x := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11} {
		for c, y := range []string{"A", "B", "C", "D", "E", "F", "G"} {
			name := fmt.Sprintf("%s%d", y, x)
			cache.Cells[name] = cell.Cell{
				A1:        name,
				Row:       r,
				Col:       c,
				Value:     fmt.Sprintf("cell %q data", name),
				Note:      "",
				Format:    "",
				UpdatedAt: time.Now(),
			}
		}
	}
	// ... [Insert your mock data generation loop here] ...
	cache.UpdateDimentions()

	// 2. Initialize Domain View Configurations
	preset := view.NewDefault(cache.ColCount())

	// Let's manually tweak the preset to test the builder mapping
	// Hide Column C (Index 2)
	if col, ok := preset.Columns[2]; ok {
		col.Visibility = view.ColHidden
		preset.Columns[2] = col
	}

	// Collapse Columns D and E (Indices 3 and 4)
	if col, ok := preset.Columns[3]; ok {
		col.Visibility = view.ColCollapsedLong
		col.WidthMode = view.WidthFixed
		col.WidthValue = 4
		preset.Columns[3] = col
	}
	if col, ok := preset.Columns[4]; ok {
		col.Visibility = view.ColCollapsedLong
		col.WidthMode = view.WidthFixed
		col.WidthValue = 4
		preset.Columns[4] = col
	}

	// Initialize Row Config (e.g., hiding row 5)
	rowCfg := view.NewRowConfiguration()
	rowCfg.States[4] = view.RowCollapsed // 0-based index 4 is Row 5

	// 3. Translate Domain -> Render using the Builder
	renderCfg := view.BuildRenderConfig(cache, &preset, rowCfg)

	// 4. Launch TUI
	m := tui.NewModel(cache, renderCfg)
	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
