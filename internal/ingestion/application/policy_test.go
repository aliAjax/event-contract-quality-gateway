package application

import (
	"testing"
	quality "github.com/example/event-contract-quality-gateway/internal/quality/domain"
)

func TestActivateRuleVersionLoadsActiveRules(t *testing.T) {
	s := New(nil, "", 10, func() string { return "id" })
	if err := s.SetRules("c", []quality.Rule{{ID: "r1", Field: "x", Operator: quality.Required, Enabled: true}}); err != nil { t.Fatal(err) }
	if err := s.ActivateRuleVersion("c", 1); err != nil { t.Fatal(err) }
	if got := s.Rules("c"); len(got) != 1 || got[0].ID != "r1" { t.Fatalf("rules=%#v", got) }
}
