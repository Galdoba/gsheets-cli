package view

import (
	"gsheets-cli/internal/domain/cell"
)

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
			Index:          i,
			RenderPosition: i,
			Letter:         cell.ColIndexToLetter(i),
			Visibility:     ColVisible,
			WidthMode:      WidthMax,
			WidthValue:     0,
			AlignRight:     false,
			FormatHint:     "",
			NoteHint:       "",
			Frosen:         false,
			GroupID:        "",
		}

	}

	return p
}
