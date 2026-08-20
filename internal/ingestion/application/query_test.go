package application

import (
	"context"
	"testing"
	"time"

	ing "github.com/example/event-contract-quality-gateway/internal/ingestion/domain"
)

func testQueryService() *Service {
	return &Service{events: map[string]ing.Event{"e1": {ID: "e1", TenantID: "t", ReceivedAt: time.Unix(1, 0), Payload: map[string]any{"state": "new"}}, "e2": {ID: "e2", TenantID: "t", ReceivedAt: time.Unix(2, 0), Payload: map[string]any{"state": "ready"}}}, seen: map[string]ing.Receipt{}, attempts: map[string][]ing.Attempt{}}
}

func TestFindEventsReturnsPayloadSnapshot(t *testing.T) {
	s := testQueryService(); page, err := s.FindEvents(context.Background(), ing.EventFilter{TenantID: "t"}, 1, "")
	if err != nil { t.Fatal(err) }; page.Items[0].Payload["state"] = "changed"
	again, err := s.FindEvents(context.Background(), ing.EventFilter{TenantID: "t"}, 1, "")
	if err != nil { t.Fatal(err) }; if again.Items[0].Payload["state"] != "new" { t.Fatalf("event payload was aliased: %#v", again.Items[0].Payload) }
}
