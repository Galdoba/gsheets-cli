package main

import (
	"fmt"
	"gsheets-cli/internal/domain/cell"
	"gsheets-cli/internal/domain/render"
	"gsheets-cli/internal/domain/sheet"
	"gsheets-cli/internal/domain/view"
	"time"
)

func main() {
	// 1. Создаём тестовые данные
	data := &sheet.SheetCache{
		SpreadsheetTitle: "Demo Sheet",
		SheetName:        "Sheet1",
		RevisionID:       "v1",
		LastSync:         time.Now(),
		Rows:             5,
		Cols:             3,
		Cells: map[string]cell.Cell{
			"A1": {A1: "A1", Row: 0, Col: 0, Value: "Name"},
			"B1": {A1: "B1", Row: 0, Col: 1, Value: "Age"},
			"C1": {A1: "C1", Row: 0, Col: 2, Value: "City"},
			"A2": {A1: "A2", Row: 1, Col: 0, Value: "Alice"},
			"B2": {A1: "B2", Row: 1, Col: 1, Value: "30"},
			"C2": {A1: "C2", Row: 1, Col: 2, Value: "New York"},
			"A3": {A1: "A3", Row: 2, Col: 0, Value: "Bob"},
			"B3": {A1: "B3", Row: 2, Col: 1, Value: "25"},
			"C3": {A1: "C3", Row: 2, Col: 2, Value: "London"},
		},
	}

	// 2. Создаём пресет
	preset := view.DefaultPreset()
	preset.Columns = map[int]view.ColumnConfig{
		// 0: {Visibility: view.ColVisible, WidthMode: view.WidthMax, AlignRight: false},
		// 1: {Visibility: view.ColVisible, WidthMode: view.WidthFixed, WidthValue: 5, AlignRight: true},
		// 2: {Visibility: view.ColVisible, WidthMode: view.WidthMax, AlignRight: false},
	}

	// 3. Рендерим полный канвас
	canvas := render.Render(data, &preset)

	// 4. Сохраняем в файл
	err := render.SaveToFile(canvas, "output.txt")
	if err != nil {
		fmt.Printf("Error saving file: %v\n", err)
		return
	}

	// 5. Для отладки: выводим в консоль
	fmt.Println("=== Full Canvas ===")
	fmt.Print(canvas.String())

	// 6. Пример использования Viewport: выводим только первые 2 колонки, 3 строки
	fmt.Println("\n=== Viewport (x=0, y=0, w=15, h=4) ===")
	fmt.Print(canvas.Viewport(0, 0, 15, 4))
}
