package domain

import "testing"

func TestCatalogListIsolation(t *testing.T) {
	c := NewCatalog(4)
	if err := c.Put(RuleSet{ContractID: "c", Version: 1, Rules: []Rule{{ID: "r", Field: "x", Operator: Equals, Value: "ok", Enabled: true}}, Active: true}); err != nil { t.Fatal(err) }
	items := c.List("c"); items[0].Active = false
	active, err := c.Active("c"); if err != nil { t.Fatal(err) }; if !active.Active { t.Fatal("list mutation changed catalog") }
}

func TestInvalidRegexDoesNotPanic(t *testing.T) {
	result := (Rule{ID: "bad", Field: "x", Operator: Regex, Value: "[", Enabled: true}).Execute(map[string]any{"x": "v"})
	if result.Passed { t.Fatal("invalid regex unexpectedly passed") }
}
