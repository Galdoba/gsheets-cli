package view

import (
	"gsheets-cli/internal/domain/cell"
	"gsheets-cli/internal/domain/sheet"
	"slices"

	"github.com/mattn/go-runewidth"
)

const (
	OverreachSfx = ".."
	NoteSfx      = "^"
	DirtyPrefix  = "*"
)

type Alignment string

type Column struct {
	Cells      []cell.Cell
	Title      string
	Code       string
	MaxWidth   int
	FixedWidth int
	Index      int
	Position   int
}

func GetColumn(index int, cache *sheet.SheetCache, whitelist ...int) *Column {
	c := Column{
		Index:    index,
		Position: index,
	}
	list := len(whitelist) > 0
	col := cache.GetCol(index)
	for i, cel := range col.Cells {
		if i == 0 {
			c.Title = cel.Value
			c.Code = cell.ColIndexToLetter(index)
		}
		if list && !slices.Contains(whitelist, index) {
			continue
		}

		c.MaxWidth = max(c.MaxWidth, runewidth.StringWidth(cel.Value))
	}
	return &c
}

func (c *Column) WithWidth(w int) *Column {
	c.FixedWidth = w
	return c
}

func (c *Column) WithPosition(p int) *Column {
	c.Position = p
	return c
}
