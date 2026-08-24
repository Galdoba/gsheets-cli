package profile

import (
	"fmt"
	"strings"
	"time"
)

// parseCellDate attempts to parse a date string using a set of common layouts.
// It returns the parsed time or an error if none match.
func parseCellDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	layouts := []string{
		time.RFC3339,       // "2006-01-02T15:04:05Z07:00"
		"02.01.2006",       // "16.08.2026"
		"02/01/2006",       // "16/08/2026"
		"02-01-2006",       // "16-08-2026"
		"2006-01-02",       // "2026-08-16"
		"01/02/2006",       // "08/16/2026" (US style)
		"02.01.2006 15:04", // with time
		"02/01/2006 15:04",
		"02-01-2006 15:04",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %q", value)
}
