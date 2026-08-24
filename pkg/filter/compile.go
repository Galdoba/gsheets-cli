package filter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type extractorType int

const (
	typeString extractorType = iota
	typeFloat
	typeBool
	typeTime
	typeStrings
)

// compiledRule holds the pre-validated and pre-parsed state of a rule
type compiledRule struct {
	extType   extractorType
	extractor any // The actual extraction function
	targetVal any // The pre-parsed target value (or *regexp.Regexp)
	op        Operator
	colIndex  int
	allCells  bool
}

// Valid operator maps for compile-time validation
var validStringOps = map[Operator]bool{OpEq: true, OpNeq: true, OpContains: true, OpRegex: true, OpEmpty: true, OpNotEmpty: true}
var validFloatOps = map[Operator]bool{OpEq: true, OpNeq: true, OpGt: true, OpGte: true, OpLt: true, OpLte: true, OpEmpty: true, OpNotEmpty: true}
var validBoolOps = map[Operator]bool{OpEq: true, OpNeq: true}
var validTimeOps = map[Operator]bool{OpEq: true, OpNeq: true, OpBefore: true, OpAfter: true, OpOlderThanDays: true, OpNewerThanDays: true, OpEmpty: true, OpNotEmpty: true}
var validStringsOps = map[Operator]bool{OpContainsAny: true, OpContainsAll: true, OpEmpty: true, OpNotEmpty: true}

// Compile translates a Group into a fast predicate function
func (fs *FilterSet[T]) Compile(g Group) (func(row []T) bool, error) {
	var compiledRules []compiledRule
	var nestedPredicates []func(row []T) bool

	isOrLogic := strings.ToLower(g.Logic) == "or"

	// Compile flat rules
	for _, rule := range g.Rules {
		cr, err := fs.compileRule(rule)
		if err != nil {
			return nil, fmt.Errorf("failed to compile rule: %w", err)
		}
		compiledRules = append(compiledRules, cr)
	}

	// Recursively compile nested groups
	for _, subGroup := range g.Groups {
		pred, err := fs.Compile(subGroup)
		if err != nil {
			return nil, err
		}
		nestedPredicates = append(nestedPredicates, pred)
	}

	// If no rules and no nested groups, always pass
	if len(compiledRules) == 0 && len(nestedPredicates) == 0 {
		return func(row []T) bool { return true }, nil
	}

	return func(row []T) bool {
		// Evaluate all flat rules
		for _, cr := range compiledRules {
			passes := fs.evaluateRule(row, cr)
			if isOrLogic && passes {
				return true
			}
			if !isOrLogic && !passes {
				return false
			}
		}

		// Evaluate all nested groups
		for _, pred := range nestedPredicates {
			passes := pred(row)
			if isOrLogic && passes {
				return true
			}
			if !isOrLogic && !passes {
				return false
			}
		}

		// If OR, all failed. If AND, all passed.
		return !isOrLogic
	}, nil
}

// compileRule handles the pre-parsing and validation of a single rule
func (fs *FilterSet[T]) compileRule(r Rule) (compiledRule, error) {
	attr := r.Attribute
	if attr == "" {
		attr = "value"
	}

	colIdx := r.ColumnIndex
	if !r.AllCells && r.ColumnName != "" {
		if fs.columnMap == nil {
			return compiledRule{}, fmt.Errorf("column name %q provided but no mapping is set", r.ColumnName)
		}
		idx, ok := fs.columnMap[r.ColumnName]
		if !ok {
			return compiledRule{}, fmt.Errorf("unknown column name %q", r.ColumnName)
		}
		colIdx = idx
	}

	if ext, ok := fs.stringExtractors[attr]; ok {
		if !validStringOps[r.Op] {
			return compiledRule{}, fmt.Errorf("unsupported operator %q for string attribute %q", r.Op, attr)
		}
		target, err := fs.parseStringTarget(r.Op, r.Value)
		if err != nil {
			return compiledRule{}, err
		}
		return compiledRule{typeString, ext, target, r.Op, colIdx, r.AllCells}, nil
	}

	if ext, ok := fs.floatExtractors[attr]; ok {
		if !validFloatOps[r.Op] {
			return compiledRule{}, fmt.Errorf("unsupported operator %q for float attribute %q", r.Op, attr)
		}
		target, err := fs.parseFloatTarget(r.Op, r.Value)
		if err != nil {
			return compiledRule{}, err
		}
		return compiledRule{typeFloat, ext, target, r.Op, colIdx, r.AllCells}, nil
	}

	if ext, ok := fs.boolExtractors[attr]; ok {
		if !validBoolOps[r.Op] {
			return compiledRule{}, fmt.Errorf("unsupported operator %q for bool attribute %q", r.Op, attr)
		}
		target, err := fs.parseBoolTarget(r.Op, r.Value)
		if err != nil {
			return compiledRule{}, err
		}
		return compiledRule{typeBool, ext, target, r.Op, colIdx, r.AllCells}, nil
	}

	if ext, ok := fs.timeExtractors[attr]; ok {
		if !validTimeOps[r.Op] {
			return compiledRule{}, fmt.Errorf("unsupported operator %q for time attribute %q", r.Op, attr)
		}
		target, err := fs.parseTimeTarget(r.Op, r.Value)
		if err != nil {
			return compiledRule{}, err
		}
		return compiledRule{typeTime, ext, target, r.Op, colIdx, r.AllCells}, nil
	}

	if ext, ok := fs.stringsExtractors[attr]; ok {
		if !validStringsOps[r.Op] {
			return compiledRule{}, fmt.Errorf("unsupported operator %q for strings attribute %q", r.Op, attr)
		}
		target, err := fs.parseStringsTarget(r.Op, r.Value)
		if err != nil {
			return compiledRule{}, err
		}
		return compiledRule{typeStrings, ext, target, r.Op, colIdx, r.AllCells}, nil
	}

	return compiledRule{}, fmt.Errorf("unknown attribute %q", attr)
}

// evaluateRule executes the compiled rule against a row
func (fs *FilterSet[T]) evaluateRule(row []T, cr compiledRule) bool {
	evalElem := func(elem T) bool {
		switch cr.extType {
		case typeString:
			ext := cr.extractor.(func(T) (string, error))
			val, err := ext(elem)
			if err != nil {
				return handleExtractorError(cr.op)
			}
			return compareString(val, cr.op, cr.targetVal)
		case typeFloat:
			ext := cr.extractor.(func(T) (float64, error))
			val, err := ext(elem)
			if err != nil {
				return handleExtractorError(cr.op)
			}
			return compareFloat(val, cr.op, cr.targetVal)
		case typeBool:
			ext := cr.extractor.(func(T) (bool, error))
			val, err := ext(elem)
			if err != nil {
				return handleExtractorError(cr.op)
			}
			return compareBool(val, cr.op, cr.targetVal)
		case typeTime:
			ext := cr.extractor.(func(T) (time.Time, error))
			val, err := ext(elem)
			if err != nil {
				return handleExtractorError(cr.op)
			}
			return compareTime(val, cr.op, cr.targetVal)
		case typeStrings:
			ext := cr.extractor.(func(T) ([]string, error))
			val, err := ext(elem)
			if err != nil {
				return handleExtractorError(cr.op)
			}
			return compareStrings(val, cr.op, cr.targetVal)
		}
		return false
	}

	if cr.allCells {
		if len(row) == 0 {
			return true // Vacuous truth
		}
		for _, elem := range row {
			if !evalElem(elem) {
				return false
			}
		}
		return true
	}

	if cr.colIndex < 0 || cr.colIndex >= len(row) {
		return handleExtractorError(cr.op)
	}

	return evalElem(row[cr.colIndex])
}

// --- Target Parsers ---

func (fs *FilterSet[T]) parseStringTarget(op Operator, val string) (any, error) {
	if op == OpRegex {
		re, err := regexp.Compile(val)
		if err != nil {
			return nil, fmt.Errorf("invalid regex %q: %w", val, err)
		}
		return re, nil
	}
	return val, nil
}

func (fs *FilterSet[T]) parseFloatTarget(op Operator, val string) (any, error) {
	if op == OpEmpty || op == OpNotEmpty {
		return nil, nil
	}
	f, err := fs.parsers.Float(val)
	if err != nil {
		return nil, fmt.Errorf("invalid float %q: %w", val, err)
	}
	return f, nil
}

func (fs *FilterSet[T]) parseBoolTarget(op Operator, val string) (any, error) {
	b, err := fs.parsers.Bool(val)
	if err != nil {
		return nil, fmt.Errorf("invalid bool %q: %w", val, err)
	}
	return b, nil
}

func (fs *FilterSet[T]) parseTimeTarget(op Operator, val string) (any, error) {
	if op == OpOlderThanDays || op == OpNewerThanDays {
		days, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid days %q: %w", val, err)
		}
		return days, nil
	}
	t, err := fs.parsers.Time(val)
	if err != nil {
		return nil, fmt.Errorf("invalid time %q: %w", val, err)
	}
	return t, nil
}

func (fs *FilterSet[T]) parseStringsTarget(op Operator, val string) (any, error) {
	if op == OpEmpty || op == OpNotEmpty {
		return nil, nil
	}
	s, err := fs.parsers.Strings(val)
	if err != nil {
		return nil, fmt.Errorf("invalid strings %q: %w", val, err)
	}
	return s, nil
}

// --- Comparators ---

func handleExtractorError(op Operator) bool {
	if op == OpEmpty {
		return true
	}
	if op == OpNotEmpty {
		return false
	}
	return false
}

func compareString(val string, op Operator, target any) bool {
	switch op {
	case OpEq:
		return val == target.(string)
	case OpNeq:
		return val != target.(string)
	case OpContains:
		return strings.Contains(val, target.(string))
	case OpRegex:
		return target.(*regexp.Regexp).MatchString(val)
	case OpEmpty:
		return val == ""
	case OpNotEmpty:
		return val != ""
	}
	return false
}

func compareFloat(val float64, op Operator, target any) bool {
	if op == OpEmpty {
		return false
	}
	if op == OpNotEmpty {
		return true
	}
	t := target.(float64)
	switch op {
	case OpEq:
		return val == t
	case OpNeq:
		return val != t
	case OpGt:
		return val > t
	case OpGte:
		return val >= t
	case OpLt:
		return val < t
	case OpLte:
		return val <= t
	}
	return false
}

func compareBool(val bool, op Operator, target any) bool {
	t := target.(bool)
	switch op {
	case OpEq:
		return val == t
	case OpNeq:
		return val != t
	}
	return false
}

func compareTime(val time.Time, op Operator, target any) bool {
	if op == OpEmpty {
		return val.IsZero()
	}
	if op == OpNotEmpty {
		return !val.IsZero()
	}
	switch op {
	case OpEq:
		return val.Equal(target.(time.Time))
	case OpNeq:
		return !val.Equal(target.(time.Time))
	case OpBefore:
		return val.Before(target.(time.Time))
	case OpAfter:
		return val.After(target.(time.Time))
	case OpOlderThanDays:
		days := target.(int)
		cutoff := time.Now().AddDate(0, 0, -days)
		return val.Before(cutoff)
	case OpNewerThanDays:
		days := target.(int)
		cutoff := time.Now().AddDate(0, 0, -days)
		return val.After(cutoff)
	}
	return false
}

func compareStrings(val []string, op Operator, target any) bool {
	if op == OpEmpty {
		return len(val) == 0
	}
	if op == OpNotEmpty {
		return len(val) > 0
	}
	t := target.([]string)
	switch op {
	case OpContainsAny:
		for _, s := range t {
			if containsString(val, s) {
				return true
			}
		}
		return false
	case OpContainsAll:
		for _, s := range t {
			if !containsString(val, s) {
				return false
			}
		}
		return true
	}
	return false
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
