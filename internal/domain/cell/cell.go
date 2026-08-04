package cell

import (
	"fmt"
	"strings"
	"time"
)

// Cell represents single cell from excel of google spreadsheet
type Cell struct {
	A1        string    `json:"a1"`
	Row       int       `json:"row"`
	Col       int       `json:"col"`
	Value     string    `json:"value"`
	Note      string    `json:"note,omitempty"`
	Format    string    `json:"format,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewFromRowCol creates cell with 1-based row-col position
func NewFromRowCol(row, col int) Cell {
	a1 := PositionToA1(row, col)
	c := Cell{
		A1:     a1,
		Row:    row,
		Col:    col,
		Value:  "",
		Note:   "",
		Format: "",
	}
	return c
}

// NewFromA1 creates cell with a1 notation
func NewFromA1(a1 string) (Cell, error) {
	a1 = strings.ToUpper(a1)
	row, col, err := A1ToPosition(a1)
	if err != nil {
		return Cell{}, fmt.Errorf("failed to convert a1 notation: %w", err)
	}
	c := Cell{
		A1:        a1,
		Row:       row,
		Col:       col,
		Value:     "",
		Note:      "",
		Format:    "",
		UpdatedAt: time.Time{},
	}
	return c, nil
}

func PositionToA1(row, col int) string {
	return fmt.Sprintf("%s%d", colToLetter(col), row)
}

// A1ToPosition converts an A1 notation string (e.g., "A1", "AB42") into
// (row, column) indices (both 1-based). Returns an error if the format is invalid.
func A1ToPosition(a1 string) (int, int, error) {
	if a1 == "" {
		return 0, 0, fmt.Errorf("empty A1 reference")
	}
	a1 = strings.ToUpper(a1)
	letters := ""
	digits := ""
	for _, ch := range a1 {
		if ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z' {
			letters += string(ch)
		} else if ch >= '0' && ch <= '9' {
			digits += string(ch)
		} else {
			return 0, 0, fmt.Errorf("invalid character '%c' in A1 reference", ch)
		}
	}

	if letters == "" {
		return 0, 0, fmt.Errorf("missing column letters in A1 reference")
	}
	if digits == "" {
		return 0, 0, fmt.Errorf("missing row number in A1 reference")
	}

	col, err := letterToColumn(letters)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid column part: %w", err)
	}

	var row int
	_, err = fmt.Sscanf(digits, "%d", &row)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid row number: %w", err)
	}
	if row < 1 {
		return 0, 0, fmt.Errorf("row number must be >= 1, got %d", row)
	}

	return row, col, nil
}

// colToLetter converts 1-based column index to Excel-style A1 notation
func colToLetter(n int) string {
	if n <= 0 {
		return ""
	}
	letter := ""
	for n > 0 {
		n--
		letter = string(rune('A'+n%26)) + letter
		n /= 26
	}
	return letter
}

// letterToColumn converts Excel column letters (e.g., "A", "Z", "AA", "AB") to a 1‑based column index.
// Returns an error if the input is empty or contains non-letter characters.
func letterToColumn(letters string) (int, error) {
	if letters == "" {
		return 0, fmt.Errorf("empty column letters")
	}
	col := 0
	for _, ch := range letters {
		if ch < 'A' || ch > 'Z' {
			// Allow lowercase and convert to uppercase
			if ch >= 'a' && ch <= 'z' {
				ch = ch - 'a' + 'A'
			} else {
				return 0, fmt.Errorf("invalid character '%c' in column letters", ch)
			}
		}
		val := int(ch - 'A' + 1)
		col = col*26 + val
	}
	return col, nil
}

func (c Cell) Validate() error {
	if c.A1 != strings.ToUpper(c.A1) {
		return fmt.Errorf("cell name %q, must be in upper resgister", c.A1)
	}
	row, col, err := A1ToPosition(c.A1)
	if err != nil {
		return err
	}
	if c.Row != row || c.Col != col {
		return fmt.Errorf("actual row, col (%d,%d) do not match projected (%d,%d)", c.Row, c.Col, row, col)
	}
	pos := PositionToA1(row, col)
	if c.A1 != pos {
		return fmt.Errorf("actual cell name (%q) do not match projected (%q)", c.A1, pos)
	}
	return nil
}
func Equal(c1, c2 Cell) bool {
	for _, match := range []bool{
		c1.A1 == c2.A1,
		c1.Format == c2.Format,
		c1.Value == c2.Value,
		c1.Note == c2.Note,
		c1.Col == c2.Col,
		c1.Row == c2.Row,
	} {
		if !match {
			return false
		}
	}
	return true
}

func ColIndexToLetter(col int) string {
	return colToLetter(col + 1)
}
