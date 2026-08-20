package application

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	contract "github.com/example/event-contract-quality-gateway/internal/contract/domain"
	ing "github.com/example/event-contract-quality-gateway/internal/ingestion/domain"
)

type missingContracts struct{}
func (missingContracts) Find(context.Context, string, string) (contract.Contract, error) { return contract.Contract{}, fmt.Errorf("missing") }

func TestQuotaRejectsConcurrentOverflow(t *testing.T) {
	s := New(missingContracts{}, "", 1, func() string { return fmt.Sprintf("id-%d", time.Now().UnixNano()) })
	var wg sync.WaitGroup
	results := make(chan error, 2)
	start := make(chan struct{})
	for i := 0; i < 2; i++ { wg.Add(1); go func(n int) { defer wg.Done(); <-start; _, err := s.Publish(context.Background(), ing.Event{TenantID: "t", ContractID: "c", SchemaVersion: 1, IdempotencyKey: fmt.Sprintf("k-%d", n), Payload: map[string]any{"x": 1}}); results <- err }(i) }
	close(start); wg.Wait(); close(results)
	accepted := 0
	for err := range results { if err == nil { accepted++ } }
	if accepted != 1 { t.Fatalf("accepted %d events with quota 1", accepted) }
}
