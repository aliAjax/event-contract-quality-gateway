package domain

import "testing"

func TestRegexRule(t *testing.T) {
	r := Rule{ID: "email", Field: "email", Operator: Regex, Value: `^[^@]+@[^@]+$`, Enabled: true}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	if result := r.Execute(map[string]any{"email": "a@example.test"}); !result.Passed {
		t.Fatalf("unexpected failure: %s", result.Reason)
	}
}
