package store

import (
	"context"
	"testing"

	"github.com/example/event-contract-quality-gateway/internal/contract/domain"
)

func TestContractCloneDeepCopiesEnums(t *testing.T) {
	m := NewMemoryContracts()
	c := domain.Contract{ID: "c1", OrganizationID: "tenant", Versions: []domain.Version{{Number: 1, Schema: domain.Schema{Fields: []domain.Field{{Name: "state", Type: "string", Enum: []string{"ready", "done"}}}}}}}
	if err := m.Create(context.Background(), c); err != nil { t.Fatal(err) }
	got, err := m.Find(context.Background(), "tenant", "c1")
	if err != nil { t.Fatal(err) }
	got.Versions[0].Schema.Fields[0].Enum[0] = "corrupt"
	again, err := m.Find(context.Background(), "tenant", "c1")
	if err != nil { t.Fatal(err) }
	if again.Versions[0].Schema.Fields[0].Enum[0] != "ready" { t.Fatalf("stored enum was mutated: %#v", again.Versions[0].Schema.Fields[0].Enum) }
}
