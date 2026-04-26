package view

import (
	"fmt"
	"io"
)

type RenderMode string

const (
	RenderModeUnset RenderMode = ""
	RenderModeASCII RenderMode = "ascii"
	RenderModeUTF8  RenderMode = "utf-8"
	RenderModeRaw   RenderMode = "bytes"
	Stderr                     = "<Stderr>" //used by tui
	Stdout                     = "<Stdout>" //used by piping
)

type renderer struct {
	colConfigs  []*ColumnConfig
	rowConfig   *RowConfig
	renderMode  RenderMode
	destination io.Writer
	// viewport Viewport //tobe implemented
}

func (r *renderer) Render(rc *RowConfig, cc []*ColumnConfig) ([]byte, error) {
	/*
	   RENDERING ALGORITHM:

	   1. RESOLVE COLUMNS
	      ├─ Filter out ColHidden
	      ├─ Replace contiguous ColCollapsed with single marker
	      ├─ Compute ComputedWidth for each (Fixed or Auto)
	      └─ Order: Frozen cols first, then remaining

	   2. RESOLVE ROWS
	      ├─ Apply ActiveFilters → mark RowHidden
	      ├─ Apply Groups → mark DetailRows as RowCollapsed/Visible
	      ├─ Collect visible row indices (including FrozenRows at top)
	      └─ Slice to Viewport.RowStart..RowEnd

	   3. CALCULATE HEADER
	      ├─ Column letters + NoteInHeader markers
	      ├─ Right-align/center as per config
	      └─ Generate separator line: `─` × TotalWidth

	   4. RENDER ROWS
	      FOR each visible row:
	        ├─ Build cell strings using ColumnConfig rules
	        ├─ Truncate, pad, append NoteInCell markers
	        ├─ Join with `│` (ASCII `|` if monochrome)
	        └─ Write to buffer

	   5. OUTPUT
	      ├─ Join lines with `\n`
	      ├─ Strip ANSI (pure `.txt` mode)
	      └─ Return string / write to file

	*/
	return nil, fmt.Errorf("render is not implemented")
}

//render manager?
