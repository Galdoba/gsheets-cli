package view

type WidthMode int

const (
	WidthUnset WidthMode = iota
	WidthMin             //column width equal shortest non empty cell of the column
	WidthMax             //column width equal longest cell in the column
	WidthFixed           //column width is fixed. longer values are trimmed
)

// ColumnVisibility defines visibility of the column
type ColumnVisibility int

const (
	VisibilityUnset   ColumnVisibility = iota
	ColVisible                         //normal view
	ColHidden                          //column is completely skipped: eg. |col1|col5|
	ColCollapsedShort                  //hidden columns indicated with no marker: eg. |col1||col5|
	ColCollapsedLong                   //marker is indicating hidden columns: eg. |col1|+3|col5|
)

type NotePresentationMode int

const (
	Unset   NotePresentationMode = iota
	Hide                         //show no indication if Note is present
	Mark                         //add marker if note is not empty: eg. |col1|col2*|col3|
	Append                       //append note: eg. |col1|col2[note text]|col3|
	Replace                      //replace cell value with note text: eg. |col1|[note text]|col3|
)

// ColumnConfig explains to Renderer how to present column content
type ColumnConfig struct {
	Index          int              `json:"-"`
	RenderPosition int              `json:"render_position"`
	Letter         string           `json:"letter"`
	Visibility     ColumnVisibility `json:"visibility"`
	WidthMode      WidthMode        `json:"width_mode"`
	WidthValue     int
	AlignRight     bool   `json:"align_right"`
	FormatHint     string `json:"format_hint"`
	NoteHint       string `json:"note_hint"`
	Frosen         bool   `json:"frosen"`
	GroupID        string `json:"group_id"`
	// computedWidth  int    `json:"-"`
}
