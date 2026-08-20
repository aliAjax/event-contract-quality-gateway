package application

import (
	"context"
	"fmt"
	"sort"

	quality "github.com/example/event-contract-quality-gateway/internal/quality/domain"
)

// Rules returns a defensive copy so callers cannot mutate live validation
// policy while an event is being processed.
func (s *Service) Rules(contractID string) []quality.Rule {
	s.mu.Lock()
	defer s.mu.Unlock()
	rules := append([]quality.Rule(nil), s.rules[contractID]...)
	sort.Slice(rules, func(i, j int) bool { return rules[i].ID < rules[j].ID })
	return rules
}

func (s *Service) RuleHistory(contractID string) []quality.RuleSet {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.catalog.List(contractID)
}

func (s *Service) ActivateRuleVersion(contractID string, version int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.catalog.Activate(contractID, version); err != nil {
		return err
	}
	set, err := s.catalog.Active(contractID)
	if err != nil {
		return err
	}
	s.rules[contractID] = append([]quality.Rule(nil), set.Rules...)
	return nil
}

func (s *Service) EvaluateEvent(_ context.Context, eventID string) (quality.EvaluationSummary, error) {
	s.mu.Lock()
	event, ok := s.events[eventID]
	rules := append([]quality.Rule(nil), s.rules[event.ContractID]...)
	s.mu.Unlock()
	if !ok {
		return quality.EvaluationSummary{}, fmt.Errorf("event not found")
	}
	return quality.Evaluate(rules, event.Payload), nil
}

func (s *Service) RuleResults(eventID string) ([]quality.RuleEvaluation, error) {
	summary, err := s.EvaluateEvent(context.Background(), eventID)
	if err != nil {
		return nil, err
	}
	return append([]quality.RuleEvaluation(nil), summary.Evaluations...), nil
}
