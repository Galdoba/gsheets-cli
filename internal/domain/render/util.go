package render

import "fmt"

// processNote applies the configured NoteMode to the cell content
func processNote(value, note string, mode NoteMode) string {
	if note == "" {
		return value
	}

	switch mode {
	case NoteHide:
		return value
	case NoteMark:
		return value + "*" // Or use an icon like "🗨️"
	case NoteAppend:
		return fmt.Sprintf("%s [%s]", value, note)
	case NoteReplace:
		return note
	default:
		return value
	}
}
