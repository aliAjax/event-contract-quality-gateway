package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

type Operator string

const (
	Required Operator = "required"
	Regex    Operator = "regex"
	Min      Operator = "min"
	Max      Operator = "max"
	Equals   Operator = "equals"
)

type Rule struct {
	ID       string        `json:"id"`
	Field    string        `json:"field"`
	Operator Operator      `json:"operator"`
	Value    string        `json:"value,omitempty"`
	Enabled  bool          `json:"enabled"`
	Timeout  time.Duration `json:"timeout"`
}
type Result struct {
	RuleID   string        `json:"rule_id"`
	Passed   bool          `json:"passed"`
	Reason   string        `json:"reason,omitempty"`
	Duration time.Duration `json:"duration"`
}

func (r Rule) Validate() error {
	if r.ID == "" || r.Field == "" {
		return fmt.Errorf("rule id and field required")
	}
	switch r.Operator {
	case Required, Regex, Min, Max, Equals:
	default:
		return fmt.Errorf("unsupported rule operator")
	}
	if r.Operator == Regex {
		if _, e := regexp.Compile(r.Value); e != nil {
			return fmt.Errorf("invalid regex: %w", e)
		}
	}
	return nil
}
func (r Rule) Execute(fields map[string]any) Result {
	start := time.Now()
	v, exists := fields[r.Field]
	res := Result{RuleID: r.ID, Passed: true}
	switch r.Operator {
	case Required:
		res.Passed = exists && v != nil && strings.TrimSpace(fmt.Sprint(v)) != ""
		if !res.Passed {
			res.Reason = "value is required"
		}
	case Equals:
		res.Passed = exists && fmt.Sprint(v) == r.Value
		if !res.Passed {
			res.Reason = "value does not equal expected value"
		}
	case Regex:
		res.Passed = exists && regexp.MustCompile(r.Value).MatchString(fmt.Sprint(v))
		if !res.Passed {
			res.Reason = "value does not match regex"
		}
	case Min:
		var n float64
		_, e := fmt.Sscan(fmt.Sprint(v), &n)
		var min float64
		_, me := fmt.Sscan(r.Value, &min)
		res.Passed = e == nil && me == nil && n >= min
		if !res.Passed {
			res.Reason = "value is below minimum"
		}
	case Max:
		var n float64
		_, e := fmt.Sscan(fmt.Sprint(v), &n)
		var max float64
		_, me := fmt.Sscan(r.Value, &max)
		res.Passed = e == nil && me == nil && n <= max
		if !res.Passed {
			res.Reason = "value exceeds maximum"
		}
	}
	res.Duration = time.Since(start)
	return res
}
