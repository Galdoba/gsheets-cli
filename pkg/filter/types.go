package filter

import "time"

// Operator defines the logical comparison for a filter rule
type Operator string

const (
	// String operators
	OpEq       Operator = "eq"
	OpNeq      Operator = "neq"
	OpContains Operator = "contains"
	OpRegex    Operator = "regex"
	OpEmpty    Operator = "empty"
	OpNotEmpty Operator = "not_empty"

	// Numeric operators
	OpGt  Operator = "gt"
	OpGte Operator = "gte"
	OpLt  Operator = "lt"
	OpLte Operator = "lte"

	// Time operators
	OpBefore        Operator = "before"
	OpAfter         Operator = "after"
	OpOlderThanDays Operator = "older_than_days"
	OpNewerThanDays Operator = "newer_than_days"

	// Slice operators
	OpContainsAny Operator = "contains_any"
	OpContainsAll Operator = "contains_all"
)

// Rule is a declarative, JSON-serializable filter instruction
type Rule struct {
	ColumnIndex int      `json:"column_index,omitempty"`
	ColumnName  string   `json:"column_name,omitempty"`
	Attribute   string   `json:"attribute,omitempty"` // defaults to "value"
	Op          Operator `json:"op"`
	Value       string   `json:"value"`
	AllCells    bool     `json:"all_cells,omitempty"`
}

// Group represents a collection of rules combined with logical operators
type Group struct {
	Logic  string  `json:"logic"` // "and" (default) or "or"
	Rules  []Rule  `json:"rules"`
	Groups []Group `json:"groups,omitempty"` // nested groups
}

// Parsers define how to convert string values from rules into typed values
type Parsers struct {
	String  func(string) (string, error)
	Float   func(string) (float64, error)
	Bool    func(string) (bool, error)
	Time    func(string) (time.Time, error)
	Strings func(string) ([]string, error)
}
