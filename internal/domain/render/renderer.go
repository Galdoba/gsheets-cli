package render

import (
	"gsheets-cli/internal/domain/cell"
	"gsheets-cli/internal/domain/sheet"
	"gsheets-cli/internal/domain/view"
	"strings"
)

// Render строит полный Canvas из данных и правил.
// Функция чистая, детерминированная и идемпотентная.
//
// Важное соглашение об индексах:
//   - Входные данные (SheetCache) используют 1-базированную нумерацию (A1, Row=1, Col=1).
//   - Внутренняя логика Render использует 0-базированные индексы для итерации.
//   - При доступе к данным происходит явная конвертация: 0→1.
func Render(data *sheet.SheetCache, preset *view.Preset) Canvas {
	if preset == nil {
		p := view.DefaultPreset()
		preset = &p
	}

	// ========================================================================
	// PHASE 1: RESOLUTION — сбор видимых колонок и строк
	// ========================================================================

	type visibleCol struct {
		Index         int    // 0-based индекс колонки во внутреннем цикле
		Letter        string // Буквенное обозначение для заголовка (A, B, C...)
		Config        view.ColumnConfig
		ComputedWidth int // Финальная ширина в экранных позициях
	}
	var visibleCols []visibleCol

	// Итерируемся по колонкам исходных данных (0-based цикл)
	// data.Cols содержит количество колонок, индексы 0..Cols-1
	for col := 0; col < data.Cols; col++ {
		cfg, ok := preset.Columns[col]
		if !ok {
			// Если правила нет, используем дефолт: видимая, авто-ширина
			cfg = view.ColumnConfig{Visibility: view.ColVisible, WidthMode: view.WidthMax}
		}
		if cfg.Visibility == view.ColHidden {
			continue
		}
		visibleCols = append(visibleCols, visibleCol{
			Index:  col,
			Letter: cell.ColIndexToLetter(col + 1), // FIX: конвертация 0→1 для заголовка
			Config: cfg,
		})
	}

	// Сбор видимых строк (пока без фильтрации, берём все)
	// visibleRows хранит 0-базированные индексы строк для итерации
	visibleRows := make([]int, data.Rows)
	for i := range visibleRows {
		visibleRows[i] = i
	}

	// ========================================================================
	// PHASE 2: LAYOUT — вычисление геометрии (ширины колонок)
	// ========================================================================

	for i := range visibleCols {
		vc := &visibleCols[i]
		switch vc.Config.WidthMode {
		case view.WidthFixed:
			// Фиксированная ширина из пресета
			vc.ComputedWidth = vc.Config.WidthValue
		case view.WidthMax, view.WidthMin:
			// Сканируем данные для определения ширины
			// Минимум — ширина заголовка
			maxW := stringWidth(vc.Letter)
			for _, rowIdx := range visibleRows {
				// FIX: конвертация индексов при доступе к данным: 0→1
				cellVal := data.GetCell(rowIdx+1, vc.Index+1)
				w := stringWidth(cellVal.Value)
				if w > maxW {
					maxW = w
				}
			}
			vc.ComputedWidth = maxW
		default:
			vc.ComputedWidth = 10 // fallback
		}
		// Гарантируем минимальную ширину 1
		if vc.ComputedWidth < 1 {
			vc.ComputedWidth = 1
		}
	}

	// Вычисляем общую ширину канваса с учётом разделителей
	separatorWidth := 0
	if preset.Charset.Border {
		separatorWidth = len(visibleCols) + 1 // |col1|col2| → N+1 вертикальная черта
	}
	totalWidth := separatorWidth
	for _, vc := range visibleCols {
		totalWidth += vc.ComputedWidth
	}

	// Высота: заголовок + (опционально) разделитель + строки данных
	headerHeight := 1
	separatorHeight := 0
	if preset.Charset.Border {
		separatorHeight = 1
	}
	totalHeight := headerHeight + separatorHeight + len(visibleRows)

	// ========================================================================
	// PHASE 3: RASTERIZATION — создание и заполнение буфера
	// ========================================================================

	canvas := NewCanvas(totalWidth, totalHeight, preset.Charset.Padding)

	// ------------------------------------------------------------------------
	// 3.1 Отрисовка заголовка (буквы колонок)
	// ------------------------------------------------------------------------
	xPos := 0
	if preset.Charset.Border {
		canvas.Set(xPos, 0, '│')
		xPos++
	}
	for _, vc := range visibleCols {
		header := vc.Letter
		// FIX: используем stringWidth для расчёта паддинга
		contentWidth := stringWidth(header)
		padding := vc.ComputedWidth - contentWidth
		if padding < 0 {
			padding = 0
		}

		if vc.Config.AlignRight {
			header = strings.Repeat(string(preset.Charset.Padding), padding) + header
		} else {
			header = header + strings.Repeat(string(preset.Charset.Padding), padding)
		}
		canvas.BlitString(xPos, 0, header)
		xPos += vc.ComputedWidth
		if preset.Charset.Border {
			canvas.Set(xPos, 0, '│')
			xPos++
		}
	}

	// ------------------------------------------------------------------------
	// 3.2 Отрисовка разделителя под заголовком
	// ------------------------------------------------------------------------
	if preset.Charset.Border {
		y := 1
		// Горизонтальная линия
		for x := 0; x < totalWidth; x++ {
			canvas.Set(x, y, '─')
		}
		// Восстанавливаем пересечения (углы)
		xPos = 0
		canvas.Set(xPos, y, '├')
		xPos++
		for _, vc := range visibleCols {
			for w := 0; w < vc.ComputedWidth; w++ {
				canvas.Set(xPos+w, y, '─')
			}
			xPos += vc.ComputedWidth
			canvas.Set(xPos, y, '┼')
			xPos++
		}
	}

	// ------------------------------------------------------------------------
	// 3.3 Отрисовка данных
	// ------------------------------------------------------------------------
	dataStartY := headerHeight + separatorHeight

	for rowIdx, row := range visibleRows {
		y := dataStartY + rowIdx
		xPos = 0
		if preset.Charset.Border {
			canvas.Set(xPos, y, '│')
			xPos++
		}
		for _, vc := range visibleCols {
			// FIX: конвертация 0→1 при доступе к данным
			cellVal := data.GetCell(row+1, vc.Index+1)
			text := cellVal.Value

			// Обрезка + эллипсис если текст не влезает
			if stringWidth(text) > vc.ComputedWidth {
				ellipsis := preset.Charset.Ellipsis
				ellipsisW := stringWidth(ellipsis)
				if ellipsisW < vc.ComputedWidth {
					text = trimToWidth(text, vc.ComputedWidth-ellipsisW) + ellipsis
				} else {
					text = trimToWidth(text, vc.ComputedWidth)
				}
			}

			// Выравнивание
			// FIX: используем stringWidth и защиту от отрицательного паддинга
			contentWidth := stringWidth(text)
			padding := vc.ComputedWidth - contentWidth
			if padding < 0 {
				padding = 0
			}
			if vc.Config.AlignRight {
				text = strings.Repeat(string(preset.Charset.Padding), padding) + text
			} else {
				text = text + strings.Repeat(string(preset.Charset.Padding), padding)
			}

			canvas.BlitString(xPos, y, text)
			xPos += vc.ComputedWidth
			if preset.Charset.Border {
				canvas.Set(xPos, y, '│')
				xPos++
			}
		}
	}

	return canvas
}

// trimToWidth обрезает строку до заданной ширины в экранных позициях.
// Учитывает multi-byte руны через runeWidth.
func trimToWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	w := 0
	for i, r := range s {
		if w+runeWidth(r) > maxWidth {
			return s[:i]
		}
		w += runeWidth(r)
	}
	return s
}
