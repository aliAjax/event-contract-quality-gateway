package domain

import (
	"testing"
	contract "github.com/example/event-contract-quality-gateway/internal/contract/domain"
)

func TestObjectValidationHandlesMapValues(t *testing.T) {
	issues := ValidatePayload(map[string]any{"meta": map[string]any{"x": 1}}, []contract.Field{{Name: "meta", Type: "object"}})
	if len(issues) != 0 { t.Fatalf("unexpected issues: %#v", issues) }
}
