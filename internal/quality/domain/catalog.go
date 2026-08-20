package domain

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// RuleSet is the versioned policy attached to a contract. Keeping policy
// versioning separate from the contract schema allows operators to roll out
// quality checks without publishing a new payload shape.
type RuleSet struct {
	ContractID string    `json:"contract_id"`
	Version    int       `json:"version"`
	Rules      []Rule    `json:"rules"`
	CreatedAt  time.Time `json:"created_at"`
	CreatedBy  string    `json:"created_by"`
	Active     bool      `json:"active"`
}

type RuleEvaluation struct {
	RuleID   string        `json:"rule_id"`
	Field    string        `json:"field"`
	Operator Operator      `json:"operator"`
	Passed   bool          `json:"passed"`
	Reason   string        `json:"reason,omitempty"`
	Duration time.Duration `json:"duration"`
}

type EvaluationSummary struct {
	Passed       bool             `json:"passed"`
	Total        int              `json:"total"`
	PassedCount  int              `json:"passed_count"`
	FailedCount  int              `json:"failed_count"`
	Evaluations  []RuleEvaluation `json:"evaluations"`
	Elapsed      time.Duration    `json:"elapsed"`
	FailureCodes []string         `json:"failure_codes,omitempty"`
}

// Validate checks policy shape and rejects duplicate rule IDs. Duplicate IDs
// are particularly dangerous because clients cannot reliably correlate a
// quality result to the configured rule.
func (s RuleSet) Validate() error {
	if strings.TrimSpace(s.ContractID) == "" {
		return fmt.Errorf("contract id is required")
	}
	if s.Version < 1 {
		return fmt.Errorf("rule set version must be positive")
	}
	seen := make(map[string]struct{}, len(s.Rules))
	for _, rule := range s.Rules {
		if err := rule.Validate(); err != nil {
			return err
		}
		if _, ok := seen[rule.ID]; ok {
			return fmt.Errorf("duplicate rule id %q", rule.ID)
		}
		seen[rule.ID] = struct{}{}
	}
	return nil
}

func (s RuleSet) Clone() RuleSet {
	s.Rules = append([]Rule(nil), s.Rules...)
	for i := range s.Rules {
		s.Rules[i].Value = strings.Clone(s.Rules[i].Value)
	}
	return s
}

// Evaluate executes enabled rules in stable ID order, making API responses
// deterministic and suitable for audit logs and replay comparisons.
func Evaluate(rules []Rule, fields map[string]any) EvaluationSummary {
	start := time.Now()
	ordered := append([]Rule(nil), rules...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	out := EvaluationSummary{Total: len(ordered), Evaluations: make([]RuleEvaluation, 0, len(ordered))}
	for _, rule := range ordered {
		if !rule.Enabled {
			continue
		}
		result := rule.Execute(fields)
		e := RuleEvaluation{RuleID: result.RuleID, Field: rule.Field, Operator: rule.Operator, Passed: result.Passed, Reason: result.Reason, Duration: result.Duration}
		out.Evaluations = append(out.Evaluations, e)
		if result.Passed {
			out.PassedCount++
		} else {
			out.FailedCount++
			out.FailureCodes = append(out.FailureCodes, rule.ID)
		}
	}
	out.Passed = out.FailedCount == 0
	out.Elapsed = time.Since(start)
	return out
}

// Catalog keeps a bounded in-memory history of policy versions. It is useful
// for deployments that need to explain which policy was active at ingestion.
type Catalog struct {
	mu      sync.RWMutex
	sets    map[string][]RuleSet
	latest  map[string]int
	maxKeep int
}

func NewCatalog(maxKeep int) *Catalog {
	if maxKeep < 1 {
		maxKeep = 10
	}
	return &Catalog{sets: make(map[string][]RuleSet), latest: make(map[string]int), maxKeep: maxKeep}
}

func (c *Catalog) Put(set RuleSet) error {
	if err := set.Validate(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	items := c.sets[set.ContractID]
	if set.Version <= c.latest[set.ContractID] {
		return fmt.Errorf("rule set version must be greater than current version")
	}
	for _, existing := range items {
		if existing.Version == set.Version {
			return fmt.Errorf("rule set version already exists")
		}
	}
	if set.Active {
		for i := range items {
			items[i].Active = false
		}
	}
	items = append(items, set.Clone())
	sort.Slice(items, func(i, j int) bool { return items[i].Version < items[j].Version })
	if len(items) > c.maxKeep {
		items = items[len(items)-c.maxKeep:]
	}
	c.sets[set.ContractID] = items
	c.latest[set.ContractID] = set.Version
	return nil
}

func (c *Catalog) NextVersion(contractID string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.latest[contractID] + 1
}

func (c *Catalog) Activate(contractID string, version int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	items := c.sets[contractID]
	found := false
	for i := range items {
		items[i].Active = items[i].Version == version
		found = found || items[i].Active
	}
	if !found {
		return fmt.Errorf("rule set version not found")
	}
	c.sets[contractID] = items
	return nil
}

func (c *Catalog) Active(contractID string) (RuleSet, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, set := range c.sets[contractID] {
		if set.Active {
			return set.Clone(), nil
		}
	}
	return RuleSet{}, fmt.Errorf("active rule set not found")
}

func (c *Catalog) List(contractID string) []RuleSet {
	c.mu.RLock()
	defer c.mu.RUnlock()
	items := c.sets[contractID]
	out := make([]RuleSet, len(items))
	for i, item := range items {
		out[i] = item.Clone()
	}
	return out
}
