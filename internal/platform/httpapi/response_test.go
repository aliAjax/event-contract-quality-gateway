package httpapi

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestFailKeepsStructuredDetails(t *testing.T) {
	r := httptest.NewRecorder(); Fail(r, 422, "req-1", "bad", "invalid", map[string]any{"field": "state"})
	var envelope Envelope; if err := json.Unmarshal(r.Body.Bytes(), &envelope); err != nil { t.Fatal(err) }
	if envelope.Error == nil || envelope.Error.Details == nil { t.Fatalf("details were dropped: %#v", envelope.Error) }
}
