package view

type Preset struct {
	Columns map[int]ColumnConfig `json:"columns"` // 0-based
	Charset struct {
		Border   bool   `json:"border"`
		Padding  rune   `json:"padding"`
		Ellipsis string `json:"ellipsis"` // "…" or "..."
	} `json:"charset"`
}

func DefaultPreset() Preset {
	p := Preset{}
	p.Charset.Padding = ' '
	p.Charset.Ellipsis = "…"
	p.Charset.Border = true
	return p
}

func NewDefault(columns int) Preset {
	p := DefaultPreset()
	p.Columns = make(map[int]ColumnConfig, columns)
	for i := range columns {
		p.Columns[i] = ColumnConfig{
			Index:        i,
			NoteSfx:      false,
			CutSfx:       false,
			DirtySfx:     false,
			ViewPosition: i,
			Width:        0,
			Source:       SourceValues,
		}

	}

	return p
}
