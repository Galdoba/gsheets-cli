package filter

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// FilterSet is a generic filter builder for a specific type T
type FilterSet[T any] struct {
	stringExtractors  map[string]func(T) (string, error)
	floatExtractors   map[string]func(T) (float64, error)
	boolExtractors    map[string]func(T) (bool, error)
	timeExtractors    map[string]func(T) (time.Time, error)
	stringsExtractors map[string]func(T) ([]string, error)

	parsers   Parsers
	columnMap map[string]int // Maps column names to indices
}

// NewFilterSet creates a new FilterSet with default parsers
func NewFilterSet[T any]() *FilterSet[T] {
	return &FilterSet[T]{
		stringExtractors:  make(map[string]func(T) (string, error)),
		floatExtractors:   make(map[string]func(T) (float64, error)),
		boolExtractors:    make(map[string]func(T) (bool, error)),
		timeExtractors:    make(map[string]func(T) (time.Time, error)),
		stringsExtractors: make(map[string]func(T) ([]string, error)),
		parsers:           defaultParsers(),
	}
}

// RegisterStringExtractor adds a string extractor for a given attribute name
func (fs *FilterSet[T]) RegisterStringExtractor(name string, fn func(T) (string, error)) error {
	if fs.hasExtractor(name) {
		return fmt.Errorf("extractor for attribute %q already registered", name)
	}
	fs.stringExtractors[name] = fn
	return nil
}

// RegisterFloatExtractor adds a float extractor for a given attribute name
func (fs *FilterSet[T]) RegisterFloatExtractor(name string, fn func(T) (float64, error)) error {
	if fs.hasExtractor(name) {
		return fmt.Errorf("extractor for attribute %q already registered", name)
	}
	fs.floatExtractors[name] = fn
	return nil
}

// RegisterBoolExtractor adds a bool extractor for a given attribute name
func (fs *FilterSet[T]) RegisterBoolExtractor(name string, fn func(T) (bool, error)) error {
	if fs.hasExtractor(name) {
		return fmt.Errorf("extractor for attribute %q already registered", name)
	}
	fs.boolExtractors[name] = fn
	return nil
}

// RegisterTimeExtractor adds a time extractor for a given attribute name
func (fs *FilterSet[T]) RegisterTimeExtractor(name string, fn func(T) (time.Time, error)) error {
	if fs.hasExtractor(name) {
		return fmt.Errorf("extractor for attribute %q already registered", name)
	}
	fs.timeExtractors[name] = fn
	return nil
}

// RegisterStringsExtractor adds a string slice extractor for a given attribute name
func (fs *FilterSet[T]) RegisterStringsExtractor(name string, fn func(T) ([]string, error)) error {
	if fs.hasExtractor(name) {
		return fmt.Errorf("extractor for attribute %q already registered", name)
	}
	fs.stringsExtractors[name] = fn
	return nil
}

// SetParsers sets custom parsers. If not called, default parsers are used.
func (fs *FilterSet[T]) SetParsers(p Parsers) {
	fs.parsers = p
}

// SetColumnNameMapping defines a mapping from column names to indices.
func (fs *FilterSet[T]) SetColumnNameMapping(mapping map[string]int) {
	fs.columnMap = mapping
}

// hasExtractor checks if an attribute name is already in use across all types
func (fs *FilterSet[T]) hasExtractor(name string) bool {
	if _, ok := fs.stringExtractors[name]; ok {
		return true
	}
	if _, ok := fs.floatExtractors[name]; ok {
		return true
	}
	if _, ok := fs.boolExtractors[name]; ok {
		return true
	}
	if _, ok := fs.timeExtractors[name]; ok {
		return true
	}
	if _, ok := fs.stringsExtractors[name]; ok {
		return true
	}
	return false
}

// defaultParsers returns the standard library parsers
func defaultParsers() Parsers {
	return Parsers{
		String: func(s string) (string, error) { return strings.TrimSpace(s), nil },
		Float:  func(s string) (float64, error) { return strconv.ParseFloat(s, 64) },
		Bool:   strconv.ParseBool,
		Time:   func(s string) (time.Time, error) { return time.Parse(time.RFC3339, s) },
		Strings: func(s string) ([]string, error) {
			parts := strings.Split(s, ",")
			res := make([]string, 0, len(parts))
			for _, p := range parts {
				trimmed := strings.TrimSpace(p)
				if trimmed != "" {
					res = append(res, trimmed)
				}
			}
			return res, nil
		},
	}
}
