package application

import (
	"context"
	"testing"

	"github.com/example/event-contract-quality-gateway/internal/contract/domain"
	"github.com/example/event-contract-quality-gateway/internal/platform/store"
)

func TestGetPreservesCompleteVersionFields(t *testing.T) {
	repo := store.NewMemoryContracts()
	s := New(repo, func() string { return "c1" })
	if _, err := s.Register(context.Background(), "tenant", RegisterCommand{Name: "orders"}); err != nil { t.Fatal(err) }
	if _, err := s.AddVersion(context.Background(), "tenant", "c1", domain.Schema{Kind: domain.JSONSchema, Fields: []domain.Field{{Name: "id", Type: "string"}, {Name: "state", Type: "string"}}}); err != nil { t.Fatal(err) }
	got, err := s.Get(context.Background(), "tenant", "c1")
	if err != nil { t.Fatal(err) }
	if len(got.Versions) != 1 || len(got.Versions[0].Schema.Fields) != 2 { t.Fatalf("version fields truncated: %#v", got.Versions) }
}
