package filter

import (
	"fmt"
	"testing"
	"time"
)

// mockCell is a generic type used for testing the filter package
type mockCell struct {
	Value  string
	Score  float64
	Active bool
	Date   time.Time
	Tags   []string
	Broken bool // Simulates an extractor error
}

// setupFilterSet creates a FilterSet with all necessary extractors registered
func setupFilterSet() *FilterSet[mockCell] {
	fs := NewFilterSet[mockCell]()

	_ = fs.RegisterStringExtractor("value", func(c mockCell) (string, error) {
		if c.Broken {
			return "", fmt.Errorf("broken cell")
		}
		return c.Value, nil
	})

	_ = fs.RegisterFloatExtractor("score", func(c mockCell) (float64, error) {
		if c.Broken {
			return 0, fmt.Errorf("broken cell")
		}
		return c.Score, nil
	})

	_ = fs.RegisterBoolExtractor("active", func(c mockCell) (bool, error) {
		return c.Active, nil
	})

	_ = fs.RegisterTimeExtractor("date", func(c mockCell) (time.Time, error) {
		return c.Date, nil
	})

	_ = fs.RegisterStringsExtractor("tags", func(c mockCell) ([]string, error) {
		return c.Tags, nil
	})

	return fs
}

// TestStringOperators verifies all string-based operations
func TestStringOperators(t *testing.T) {
	fs := setupFilterSet()
	row := []mockCell{{Value: "apple"}}

	tests := []struct {
		name     string
		op       Operator
		value    string
		expected bool
	}{
		{"Equals", OpEq, "apple", true},
		{"Not Equals", OpNeq, "apple", false},
		{"Contains", OpContains, "app", true},
		{"Contains Case Sensitive", OpContains, "APP", false},
		{"Regex Match", OpRegex, "^app.*$", true},
		{"Regex Fail", OpRegex, "^b.*$", false},
		{"Empty", OpEmpty, "", false},
		{"Not Empty", OpNotEmpty, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := Group{Logic: "and", Rules: []Rule{{Attribute: "value", Op: tt.op, Value: tt.value}}}
			pred, err := fs.Compile(g)
			if err != nil {
				t.Fatalf("unexpected compile error: %v", err)
			}
			if got := pred(row); got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

// TestFloatOperators verifies all numeric-based operations
func TestFloatOperators(t *testing.T) {
	fs := setupFilterSet()
	row := []mockCell{{Score: 10.5}}

	tests := []struct {
		name     string
		op       Operator
		value    string
		expected bool
	}{
		{"Equals", OpEq, "10.5", true},
		{"Not Equals", OpNeq, "10.5", false},
		{"Greater Than", OpGt, "10", true},
		{"Greater Than Equal", OpGte, "10.5", true},
		{"Less Than", OpLt, "11", true},
		{"Less Than Equal", OpLte, "10", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := Group{Logic: "and", Rules: []Rule{{Attribute: "score", Op: tt.op, Value: tt.value}}}
			pred, err := fs.Compile(g)
			if err != nil {
				t.Fatalf("unexpected compile error: %v", err)
			}
			if got := pred(row); got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

// TestTimeOperators verifies time comparisons and relative day calculations
func TestTimeOperators(t *testing.T) {
	fs := setupFilterSet()
	baseTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	twoDaysAgo := time.Now().AddDate(0, 0, -2)

	tests := []struct {
		name     string
		cellTime time.Time
		op       Operator
		value    string
		expected bool
	}{
		{"Before", baseTime, OpBefore, "2024-01-01T00:00:00Z", true},
		{"After", baseTime, OpAfter, "2023-01-01T00:00:00Z", true},
		{"Older Than Days", twoDaysAgo, OpOlderThanDays, "1", true},
		{"Newer Than Days", twoDaysAgo, OpNewerThanDays, "1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := []mockCell{{Date: tt.cellTime}}
			g := Group{Logic: "and", Rules: []Rule{{Attribute: "date", Op: tt.op, Value: tt.value}}}
			pred, err := fs.Compile(g)
			if err != nil {
				t.Fatalf("unexpected compile error: %v", err)
			}
			if got := pred(row); got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

// TestStringsOperators verifies slice operations (contains_any, contains_all)
func TestStringsOperators(t *testing.T) {
	fs := setupFilterSet()
	row := []mockCell{{Tags: []string{"red", "blue", "green"}}}

	tests := []struct {
		name     string
		op       Operator
		value    string
		expected bool
	}{
		{"Contains Any (Match)", OpContainsAny, "blue,yellow", true},
		{"Contains Any (No Match)", OpContainsAny, "yellow,purple", false},
		{"Contains All (Match)", OpContainsAll, "red,green", true},
		{"Contains All (No Match)", OpContainsAll, "red,purple", false},
		{"Empty", OpEmpty, "", false},
		{"Not Empty", OpNotEmpty, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := Group{Logic: "and", Rules: []Rule{{Attribute: "tags", Op: tt.op, Value: tt.value}}}
			pred, err := fs.Compile(g)
			if err != nil {
				t.Fatalf("unexpected compile error: %v", err)
			}
			if got := pred(row); got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

// TestLogicGroups verifies AND/OR combinations and short-circuiting
func TestLogicGroups(t *testing.T) {
	fs := setupFilterSet()
	row := []mockCell{{Value: "apple", Score: 10}}

	tests := []struct {
		name     string
		logic    string
		rules    []Rule
		expected bool
	}{
		{"AND (All True)", "and", []Rule{
			{Attribute: "value", Op: OpEq, Value: "apple"},
			{Attribute: "score", Op: OpGt, Value: "5"},
		}, true},
		{"AND (One False)", "and", []Rule{
			{Attribute: "value", Op: OpEq, Value: "apple"},
			{Attribute: "score", Op: OpGt, Value: "15"},
		}, false},
		{"OR (One True)", "or", []Rule{
			{Attribute: "value", Op: OpEq, Value: "banana"},
			{Attribute: "score", Op: OpGt, Value: "5"},
		}, true},
		{"OR (All False)", "or", []Rule{
			{Attribute: "value", Op: OpEq, Value: "banana"},
			{Attribute: "score", Op: OpGt, Value: "15"},
		}, false},
		{"Empty Group", "and", []Rule{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := Group{Logic: tt.logic, Rules: tt.rules}
			pred, err := fs.Compile(g)
			if err != nil {
				t.Fatalf("unexpected compile error: %v", err)
			}
			if got := pred(row); got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

// TestAllCells verifies the AllCells=true feature
func TestAllCells(t *testing.T) {
	fs := setupFilterSet()
	row := []mockCell{{Value: "a"}, {Value: "b"}, {Value: "c"}}

	tests := []struct {
		name     string
		op       Operator
		value    string
		expected bool
	}{
		{"All Match Regex", OpRegex, "^[a-c]$", true},
		{"Not All Match Eq", OpEq, "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := Group{Logic: "and", Rules: []Rule{{Attribute: "value", Op: tt.op, Value: tt.value, AllCells: true}}}
			pred, err := fs.Compile(g)
			if err != nil {
				t.Fatalf("unexpected compile error: %v", err)
			}
			if got := pred(row); got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

// TestColumnNameMapping verifies mapping string names to column indices
func TestColumnNameMapping(t *testing.T) {
	fs := setupFilterSet()
	fs.SetColumnNameMapping(map[string]int{"Name": 0, "Score": 1})

	row := []mockCell{{Value: "apple"}, {Score: 10}}

	g := Group{Logic: "and", Rules: []Rule{
		// Target Column 0 ("Name"), extract the "value" attribute (string)
		{ColumnName: "Name", Attribute: "value", Op: OpEq, Value: "apple"},

		// Target Column 1 ("Score"), extract the "score" attribute (float)
		{ColumnName: "Score", Attribute: "score", Op: OpGt, Value: "5"},
	}}

	pred, err := fs.Compile(g)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}

	if !pred(row) {
		t.Errorf("expected column mapping to pass, but it failed")
	}
}

// TestCompilationErrors verifies that invalid rules fail at compile time
func TestCompilationErrors(t *testing.T) {
	fs := setupFilterSet()
	fs.SetColumnNameMapping(map[string]int{"Name": 0})

	tests := []struct {
		name string
		rule Rule
	}{
		{"Unknown Attribute", Rule{Attribute: "unknown", Op: OpEq, Value: "x"}},
		{"Unsupported Operator", Rule{Attribute: "value", Op: OpGt, Value: "x"}},
		{"Invalid Float Parsing", Rule{Attribute: "score", Op: OpEq, Value: "not_a_number"}},
		{"Invalid Regex", Rule{Attribute: "value", Op: OpRegex, Value: "["}},
		{"Invalid Column Name", Rule{ColumnName: "DoesNotExist", Op: OpEq, Value: "x"}},
		{"Missing Column Mapping", Rule{ColumnName: "Name", Op: OpEq, Value: "x"}}, // fs has mapping, but testing nil mapping below
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Special case for missing mapping
			if tt.name == "Missing Column Mapping" {
				fsNoMap := setupFilterSet()
				_, err := fsNoMap.Compile(Group{Rules: []Rule{tt.rule}})
				if err == nil {
					t.Errorf("expected error for missing column mapping, got nil")
				}
				return
			}

			_, err := fs.Compile(Group{Rules: []Rule{tt.rule}})
			if err == nil {
				t.Errorf("expected compilation error, got nil")
			}
		})
	}
}

// TestExtractorErrors verifies behavior when extractors fail at runtime
func TestExtractorErrors(t *testing.T) {
	fs := setupFilterSet()
	row := []mockCell{{Broken: true}} // This cell will trigger extractor errors

	tests := []struct {
		name     string
		op       Operator
		value    string
		expected bool
	}{
		{"Empty returns true on error", OpEmpty, "", true},
		{"Not Empty returns false on error", OpNotEmpty, "", false},
		{"Eq returns false on error", OpEq, "anything", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := Group{Logic: "and", Rules: []Rule{{Attribute: "value", Op: tt.op, Value: tt.value}}}
			pred, err := fs.Compile(g)
			if err != nil {
				t.Fatalf("unexpected compile error: %v", err)
			}
			if got := pred(row); got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}
