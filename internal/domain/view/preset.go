package view

// Preset — полный набор правил для рендеринга
type Preset struct {
	Columns map[int]ColumnConfig `json:"columns"` // key: col index (0-based)
	Charset struct {
		Border   bool   `json:"border"`   // рисовать ли сетку
		Padding  rune   `json:"padding"`  // обычно ' '
		Ellipsis string `json:"ellipsis"` // "…" или "..."
	} `json:"charset"`
}

// DefaultPreset возвращает настройки по умолчанию
func DefaultPreset() Preset {
	p := Preset{}
	p.Charset.Padding = ' '
	p.Charset.Ellipsis = "…"
	p.Charset.Border = true
	return p
}
