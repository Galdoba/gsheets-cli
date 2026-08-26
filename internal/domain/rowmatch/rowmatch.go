package rowmatch

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
)

// CompareType defines how a rule compares old (local) and new (remote) values.
type CompareType string

const (
	CompareExact    CompareType = "exact"
	CompareContains CompareType = "contains"
	ComparePrefix   CompareType = "prefix"
	CompareSuffix   CompareType = "suffix"
	CompareRegexp   CompareType = "regexp"
	CompareNotEmpty CompareType = "not_empty"
)

// Rule describes a single comparison criterion for one column.
type Rule struct {
	Column  int         `json:"column"` // 1‑based column index
	Type    CompareType `json:"type"`
	Weight  float64     `json:"weight"`  // relative importance (0..1)
	Pattern string      `json:"pattern"` // used only by CompareRegexp
}

// Config holds the full matching configuration.
type Config struct {
	Rules            []Rule     `json:"rules"`
	PositionWeight   float64    `json:"position_weight"`   // influence of row distance (0 = ignore)
	UniquenessWeight float64    `json:"uniqueness_weight"` // influence of ambiguity (0 = ignore)
	Thresholds       Thresholds `json:"thresholds"`
}

// Thresholds decide the final action based on confidence score.
type Thresholds struct {
	AutoAccept float64 `json:"auto_accept"`
	Escalate   float64 `json:"escalate"`
}

// Fingerprint is the pre‑computed snapshot of a row’s relevant values.
// Values maps 1‑based column index to its string value.
type Fingerprint struct {
	RowIndex int            `json:"row_index"`
	Values   map[int]string `json:"values"`
}

// Candidate is a remote row that matched at least partially, along with its score.
type Candidate struct {
	Index            int     `json:"index"`
	RawScore         float64 `json:"raw_score"`
	PositionFactor   float64 `json:"position_factor"`
	UniquenessFactor float64 `json:"uniqueness_factor"`
	Score            float64 `json:"score"` // final confidence
}

// Decision tells what to do with the match result.
type Decision string

const (
	DecisionAutoAccept Decision = "auto_accept"
	DecisionEscalate   Decision = "escalate"
	DecisionReject     Decision = "reject"
)

// MatchResult is the outcome of matching one local row against all remote rows.
type MatchResult struct {
	Confidence float64     `json:"confidence"`
	Best       *Candidate  `json:"best"`
	Candidates []Candidate `json:"candidates"`
	Decision   Decision    `json:"decision"`
	Reason     string      `json:"reason"`
}

// DefaultConfig возвращает конфигурацию с четырьмя стабильными столбцами (1-4)
// и одним расширяемым столбцом (5). Все стабильные сравниваются точным совпадением,
// расширяемый – проверкой вхождения старого значения в новое (CompareContains).
// Веса равны, позиция и уникальность оказывают умеренное влияние.
func DefaultConfig() Config {
	return Config{
		Rules: []Rule{
			{Column: 1, Type: CompareExact, Weight: 0.2},
			{Column: 2, Type: CompareExact, Weight: 0.2},
			{Column: 3, Type: CompareExact, Weight: 0.2},
			{Column: 4, Type: CompareExact, Weight: 0.2},
			{Column: 5, Type: CompareContains, Weight: 0.2},
		},
		PositionWeight:   0.1,
		UniquenessWeight: 0.3,
		Thresholds: Thresholds{
			AutoAccept: 0.8,
			Escalate:   0.5,
		},
	}
}

// NewFingerprint создаёт отпечаток строки из карты значений столбцов (1‑based индексы).
func NewFingerprint(rowIndex int, values map[int]string) Fingerprint {
	return Fingerprint{
		RowIndex: rowIndex,
		Values:   values,
	}
}

// Match compares a local fingerprint against all remote fingerprints using the
// provided configuration and returns a decision (auto‑accept, escalate, reject)
// together with the best candidate and a sorted list of all candidates.
func Match(local Fingerprint, remote []Fingerprint, config *Config) MatchResult {
	if config == nil {
		return MatchResult{
			Confidence: 0,
			Decision:   DecisionReject,
			Reason:     "nil match config",
		}
	}
	// 1. Compute raw score for every remote row.
	totalWeight := 0.0
	for _, rule := range config.Rules {
		if rule.Type == CompareNotEmpty {
			totalWeight += rule.Weight
			continue
		}
		if _, ok := local.Values[rule.Column]; ok {
			totalWeight += rule.Weight
		}
	}
	if totalWeight == 0 {
		return MatchResult{
			Confidence: 0,
			Decision:   DecisionReject,
			Reason:     "no valid rules configured or no local values for rule columns",
		}
	}

	type scoredRemote struct {
		idx       int
		rawScore  float64
		posFactor float64
	}
	var scored []scoredRemote

	for _, rem := range remote {
		raw := 0.0
		for _, rule := range config.Rules {
			newVal, hasNew := rem.Values[rule.Column]
			if !hasNew {
				continue
			}

			switch rule.Type {
			case CompareNotEmpty:
				if newVal != "" {
					raw += rule.Weight
				}
			default:
				oldVal, hasOld := local.Values[rule.Column]
				if !hasOld {
					continue // rule not checked because local value missing
				}
				if compareValues(oldVal, newVal, rule) {
					raw += rule.Weight
				}
			}
		}

		// Normalize raw score to [0,1] using totalWeight.
		normRaw := raw / totalWeight
		if normRaw < 0 {
			normRaw = 0
		} else if normRaw > 1 {
			normRaw = 1
		}

		// Position factor.
		posFactor := 1.0
		if config.PositionWeight > 0 {
			dist := math.Abs(float64(local.RowIndex - rem.RowIndex))
			posFactor = 1.0 / (1.0 + config.PositionWeight*dist)
		}

		scored = append(scored, scoredRemote{
			idx:       rem.RowIndex,
			rawScore:  normRaw,
			posFactor: posFactor,
		})
	}

	// 2. If no candidates have rawScore > 0, reject.
	var candidates []Candidate
	for _, s := range scored {
		if s.rawScore > 0 {
			candidates = append(candidates, Candidate{
				Index:          s.idx,
				RawScore:       s.rawScore,
				PositionFactor: s.posFactor,
			})
		}
	}
	if len(candidates) == 0 {
		return MatchResult{
			Confidence: 0,
			Decision:   DecisionReject,
			Reason:     "no remote row matched any rule",
		}
	}

	// 3. Compute uniqueness factor for each candidate based on the best and second‑best raw scores.
	//    Sort candidates by rawScore descending first.
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].RawScore > candidates[j].RawScore
	})

	bestRaw := candidates[0].RawScore
	secondRaw := 0.0
	if len(candidates) > 1 {
		secondRaw = candidates[1].RawScore
	}

	for i := range candidates {
		uniqueness := 1.0
		if bestRaw > 0 && config.UniquenessWeight > 0 && len(candidates) > 1 {
			uniqueness = 1.0 - config.UniquenessWeight*(secondRaw/bestRaw)
			if uniqueness < 0 {
				uniqueness = 0
			}
		}
		candidates[i].UniquenessFactor = uniqueness
		candidates[i].Score = candidates[i].RawScore * candidates[i].PositionFactor * uniqueness
		if candidates[i].Score > 1 {
			candidates[i].Score = 1
		}
	}

	// Re‑sort by final score (descending).
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	best := candidates[0]
	decision := DecisionReject
	reason := ""

	if best.Score >= config.Thresholds.AutoAccept {
		decision = DecisionAutoAccept
	} else if best.Score >= config.Thresholds.Escalate {
		decision = DecisionEscalate
		reason = "confidence below auto-accept threshold; manual selection required"
	} else {
		reason = "confidence too low for escalation"
	}

	if decision == DecisionAutoAccept {
		// Additional check: if there is a close second candidate, we might still want to escalate.
		// This can be tuned with UniquenessWeight; for safety, we can require that secondScore < bestScore * (1 - uniquenessFactor) or similar.
		// Here we keep it simple: if uniqueness < 0.5, escalate.
		if best.UniquenessFactor < 0.5 {
			decision = DecisionEscalate
			reason = "ambiguous match (low uniqueness)"
		}
	}

	return MatchResult{
		Confidence: best.Score,
		Best:       &best,
		Candidates: candidates,
		Decision:   decision,
		Reason:     reason,
	}
}

// compareValues evaluates a single rule against old (local) and new (remote) values.
func compareValues(old, new string, rule Rule) bool {
	switch rule.Type {
	case CompareExact:
		return old == new
	case CompareContains:
		return strings.Contains(new, old)
	case ComparePrefix:
		return strings.HasPrefix(new, old)
	case CompareSuffix:
		return strings.HasSuffix(new, old)
	case CompareRegexp:
		re, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return false
		}
		return re.MatchString(new)
	default:
		return false
	}
}

// String representations for debugging.
func (c Candidate) String() string {
	return fmt.Sprintf("row=%d score=%.3f (raw=%.3f pos=%.3f uniq=%.3f)",
		c.Index, c.Score, c.RawScore, c.PositionFactor, c.UniquenessFactor)
}

func (m MatchResult) String() string {
	return fmt.Sprintf("decision=%s confidence=%.3f reason=%q", m.Decision, m.Confidence, m.Reason)
}
