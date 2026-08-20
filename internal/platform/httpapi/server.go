package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	contractapp "github.com/example/event-contract-quality-gateway/internal/contract/application"
	contract "github.com/example/event-contract-quality-gateway/internal/contract/domain"
	dlq "github.com/example/event-contract-quality-gateway/internal/deadletter/domain"
	ingestapp "github.com/example/event-contract-quality-gateway/internal/ingestion/application"
	ingest "github.com/example/event-contract-quality-gateway/internal/ingestion/domain"
	"github.com/example/event-contract-quality-gateway/internal/platform/config"
	"github.com/example/event-contract-quality-gateway/internal/platform/telemetry"
	quality "github.com/example/event-contract-quality-gateway/internal/quality/domain"
	"net/http"
	"net/http/pprof"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	config    config.Config
	contracts *contractapp.Service
	ingestion *ingestapp.Service
	metrics   *telemetry.Metrics
	mux       *http.ServeMux
	shutdownHook func(context.Context) error
}

func NewServer(c config.Config, contracts *contractapp.Service, ingestion *ingestapp.Service, metrics *telemetry.Metrics) *Server {
	s := &Server{config: c, contracts: contracts, ingestion: ingestion, metrics: metrics, mux: http.NewServeMux()}
	s.routes()
	return s
}
func (s *Server) Handler() http.Handler { return s.recover(s.requestLog(s.mux)) }
func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		JSON(w, 200, RequestID(r), map[string]string{"status": "ok"})
	})
	s.mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if !s.config.ReadinessEnabled {
			Fail(w, 503, RequestID(r), "not_ready", "readiness disabled", nil)
			return
		}
		JSON(w, 200, RequestID(r), map[string]string{"status": "ready"})
	})
	s.mux.HandleFunc("GET /metrics", s.metricsHandler)
	s.mux.HandleFunc("GET /debug/pprof/", pprof.Index)
	s.mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	s.mux.HandleFunc("POST /api/v1/contracts", s.createContract)
	s.mux.HandleFunc("GET /api/v1/contracts/{id}", s.getContract)
	s.mux.HandleFunc("POST /api/v1/contracts/{id}/versions", s.addVersion)
	s.mux.HandleFunc("POST /api/v1/contracts/{id}/versions/{version}/publish", s.publish)
	s.mux.HandleFunc("POST /api/v1/contracts/{id}/compatibility-check", s.compatibility)
	s.mux.HandleFunc("POST /api/v1/ingest/events", s.ingest)
	s.mux.HandleFunc("GET /api/v1/events", s.listEvents)
	s.mux.HandleFunc("GET /api/v1/events/{id}", s.getEvent)
	s.mux.HandleFunc("GET /api/v1/events/summary", s.eventSummary)
	s.mux.HandleFunc("POST /api/v1/contracts/{id}/quality-rules", s.setRules)
	s.mux.HandleFunc("GET /api/v1/contracts/{id}/quality-rules", s.getRules)
	s.mux.HandleFunc("POST /api/v1/contracts/{id}/quality-rules/{version}/activate", s.activateRules)
	s.mux.HandleFunc("GET /api/v1/events/{id}/lineage", s.lineage)
	s.mux.HandleFunc("GET /api/v1/events/{id}/quality-results", s.quality)
	s.mux.HandleFunc("GET /api/v1/events/{id}/quality-results/detailed", s.qualityDetailed)
	s.mux.HandleFunc("GET /api/v1/events/{id}/processing-attempts", s.attempts)
	s.mux.HandleFunc("GET /api/v1/dead-letters", s.listLetters)
	s.mux.HandleFunc("POST /api/v1/dead-letters/{id}/replay", s.replay)
	s.mux.HandleFunc("POST /api/v1/dead-letters/replay", s.replayMany)
}
func (s *Server) tenant(r *http.Request, w http.ResponseWriter, id string) (string, bool) {
	t, e := Tenant(r)
	if e != nil {
		Fail(w, 400, id, "invalid_tenant", e.Error(), nil)
		return "", false
	}
	return t, true
}
func (s *Server) createContract(w http.ResponseWriter, r *http.Request) {
	id := RequestID(r)
	t, ok := s.tenant(r, w, id)
	if !ok {
		return
	}
	var cmd contractapp.RegisterCommand
	if e := Decode(r, &cmd, s.config.MaxBodyBytes); e != nil {
		Fail(w, 400, id, "invalid_request", e.Error(), nil)
		return
	}
	v, e := s.contracts.Register(r.Context(), t, cmd)
	if e != nil {
		Fail(w, 422, id, "contract_rejected", e.Error(), nil)
		return
	}
	JSON(w, 201, id, v)
}
func (s *Server) getContract(w http.ResponseWriter, r *http.Request) {
	rid := RequestID(r)
	t, ok := s.tenant(r, w, rid)
	if !ok {
		return
	}
	v, e := s.contracts.Get(r.Context(), t, r.PathValue("id"))
	if e != nil {
		Fail(w, 404, rid, "not_found", e.Error(), nil)
		return
	}
	JSON(w, 200, rid, v)
}
func (s *Server) addVersion(w http.ResponseWriter, r *http.Request) {
	rid := RequestID(r)
	t, ok := s.tenant(r, w, rid)
	if !ok {
		return
	}
	var req struct {
		Schema contract.Schema `json:"schema"`
	}
	if e := Decode(r, &req, s.config.MaxBodyBytes); e != nil {
		Fail(w, 400, rid, "invalid_request", e.Error(), nil)
		return
	}
	v, e := s.contracts.AddVersion(r.Context(), t, r.PathValue("id"), req.Schema)
	if e != nil {
		Fail(w, 422, rid, "version_rejected", e.Error(), nil)
		return
	}
	JSON(w, 201, rid, v)
}
func (s *Server) publish(w http.ResponseWriter, r *http.Request) {
	rid := RequestID(r)
	t, ok := s.tenant(r, w, rid)
	if !ok {
		return
	}
	n, e := strconv.Atoi(r.PathValue("version"))
	if e != nil {
		Fail(w, 400, rid, "invalid_version", "version must be numeric", nil)
		return
	}
	v, e := s.contracts.Publish(r.Context(), t, r.PathValue("id"), n, r.Header.Get("If-Match"))
	if e != nil {
		status := 422
		if strings.Contains(e.Error(), "etag") {
			status = 412
		}
		Fail(w, status, rid, "publish_rejected", e.Error(), nil)
		return
	}
	JSON(w, 200, rid, v)
}
func (s *Server) compatibility(w http.ResponseWriter, r *http.Request) {
	rid := RequestID(r)
	t, ok := s.tenant(r, w, rid)
	if !ok {
		return
	}
	var req struct {
		Schema contract.Schema `json:"schema"`
	}
	if e := Decode(r, &req, s.config.MaxBodyBytes); e != nil {
		Fail(w, 400, rid, "invalid_request", e.Error(), nil)
		return
	}
	v, e := s.contracts.Check(r.Context(), t, r.PathValue("id"), req.Schema)
	if e != nil {
		Fail(w, 422, rid, "compatibility_failed", e.Error(), nil)
		return
	}
	JSON(w, 200, rid, v)
}
func (s *Server) ingest(w http.ResponseWriter, r *http.Request) {
	rid := RequestID(r)
	t, ok := s.tenant(r, w, rid)
	if !ok {
		return
	}
	var e ingest.Event
	if err := Decode(r, &e, s.config.MaxBodyBytes); err != nil {
		Fail(w, 400, rid, "invalid_request", err.Error(), nil)
		return
	}
	e.TenantID = t
	e.IdempotencyKey = r.Header.Get("Idempotency-Key")
	receipt, err := s.ingestion.Publish(r.Context(), e)
	if err != nil {
		status := 422
		if strings.Contains(err.Error(), "quota") {
			status = 429
		}
		Fail(w, status, rid, "ingestion_rejected", err.Error(), nil)
		return
	}
	if receipt.Status == ingest.Accepted {
		s.metrics.Accepted.Add(1)
		JSON(w, 202, rid, receipt)
		return
	}
	s.metrics.DeadLettered.Add(1)
	JSON(w, 202, rid, receipt)
}
func (s *Server) setRules(w http.ResponseWriter, r *http.Request) {
	rid := RequestID(r)
	if _, ok := s.tenant(r, w, rid); !ok {
		return
	}
	var req struct {
		Rules []quality.Rule `json:"rules"`
	}
	if e := Decode(r, &req, s.config.MaxBodyBytes); e != nil {
		Fail(w, 400, rid, "invalid_request", e.Error(), nil)
		return
	}
	if e := s.ingestion.SetRules(r.PathValue("id"), req.Rules); e != nil {
		Fail(w, 422, rid, "rules_rejected", e.Error(), nil)
		return
	}
	JSON(w, 200, rid, map[string]any{"count": len(req.Rules)})
}
func (s *Server) getRules(w http.ResponseWriter, r *http.Request) {
	rid := RequestID(r)
	if _, ok := s.tenant(r, w, rid); !ok {
		return
	}
	JSON(w, 200, rid, map[string]any{"contract_id": r.PathValue("id"), "rules": s.ingestion.Rules(r.PathValue("id")), "history": s.ingestion.RuleHistory(r.PathValue("id"))})
}
func (s *Server) activateRules(w http.ResponseWriter, r *http.Request) {
	rid := RequestID(r)
	if _, ok := s.tenant(r, w, rid); !ok {
		return
	}
	version, err := strconv.Atoi(r.PathValue("version"))
	if err != nil || version < 1 {
		Fail(w, 400, rid, "invalid_version", "rule version must be positive", nil)
		return
	}
	if err := s.ingestion.ActivateRuleVersion(r.PathValue("id"), version); err != nil {
		Fail(w, 422, rid, "activation_rejected", err.Error(), nil)
		return
	}
	JSON(w, 200, rid, map[string]any{"contract_id": r.PathValue("id"), "version": version, "rules": s.ingestion.Rules(r.PathValue("id"))})
}
func (s *Server) lineage(w http.ResponseWriter, r *http.Request) {
	rid := RequestID(r)
	v, e := s.ingestion.GetLineage(r.PathValue("id"))
	if e != nil {
		Fail(w, 404, rid, "not_found", e.Error(), nil)
		return
	}
	JSON(w, 200, rid, v)
}
func (s *Server) quality(w http.ResponseWriter, r *http.Request) {
	rid := RequestID(r)
	v, e := s.ingestion.Quality(r.PathValue("id"))
	if e != nil {
		Fail(w, 404, rid, "not_found", e.Error(), nil)
		return
	}
	JSON(w, 200, rid, map[string]any{"reasons": v, "passed": len(v) == 0})
}
func (s *Server) qualityDetailed(w http.ResponseWriter, r *http.Request) {
	rid := RequestID(r)
	v, e := s.ingestion.EvaluateEvent(r.Context(), r.PathValue("id"))
	if e != nil {
		Fail(w, 404, rid, "not_found", e.Error(), nil)
		return
	}
	JSON(w, 200, rid, v)
}
func (s *Server) attempts(w http.ResponseWriter, r *http.Request) {
	JSON(w, 200, RequestID(r), s.ingestion.Attempts(r.PathValue("id")))
}
func (s *Server) listEvents(w http.ResponseWriter, r *http.Request) {
	rid := RequestID(r)
	tenant, ok := s.tenant(r, w, rid)
	if !ok {
		return
	}
	filter := ingest.EventFilter{
		TenantID: tenant, ContractID: r.URL.Query().Get("contract_id"),
		PartitionKey: r.URL.Query().Get("partition_key"), Search: r.URL.Query().Get("search"),
		Status: ingest.Status(r.URL.Query().Get("status")),
	}
	filter.OccurredAfter = parseTime(r.URL.Query().Get("occurred_after"))
	filter.OccurredBefore = parseTime(r.URL.Query().Get("occurred_before"))
	filter.ReceivedAfter = parseTime(r.URL.Query().Get("received_after"))
	filter.ReceivedBefore = parseTime(r.URL.Query().Get("received_before"))
	limit := queryInt(r.URL.Query().Get("limit"), 50)
	page, err := s.ingestion.FindEvents(r.Context(), filter, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		Fail(w, 400, rid, "invalid_query", err.Error(), nil)
		return
	}
	JSON(w, 200, rid, page)
}
func (s *Server) getEvent(w http.ResponseWriter, r *http.Request) {
	rid := RequestID(r)
	tenant, ok := s.tenant(r, w, rid)
	if !ok {
		return
	}
	event, receipt, err := s.ingestion.GetEvent(r.Context(), tenant, r.PathValue("id"))
	if err != nil {
		Fail(w, 404, rid, "not_found", err.Error(), nil)
		return
	}
	JSON(w, 200, rid, map[string]any{"event": event, "receipt": receipt, "attempts": s.ingestion.Attempts(event.ID)})
}
func (s *Server) eventSummary(w http.ResponseWriter, r *http.Request) {
	rid := RequestID(r)
	tenant, ok := s.tenant(r, w, rid)
	if !ok {
		return
	}
	JSON(w, 200, rid, s.ingestion.Summary(r.Context(), tenant, r.URL.Query().Get("contract_id")))
}
func (s *Server) listLetters(w http.ResponseWriter, r *http.Request) {
	rid := RequestID(r)
	t, ok := s.tenant(r, w, rid)
	if !ok {
		return
	}
	state := dlq.State(r.URL.Query().Get("state"))
	JSON(w, 200, rid, s.ingestion.ListLetters(state, t))
}
func (s *Server) replay(w http.ResponseWriter, r *http.Request) {
	rid := RequestID(r)
	actor := r.Header.Get("X-Actor-ID")
	if actor == "" {
		actor = "unknown"
	}
	v, e := s.ingestion.Replay(context.Background(), r.PathValue("id"), actor)
	if e != nil {
		Fail(w, 422, rid, "replay_rejected", e.Error(), nil)
		return
	}
	JSON(w, 202, rid, v)
}
func (s *Server) replayMany(w http.ResponseWriter, r *http.Request) {
	rid := RequestID(r)
	tenant, ok := s.tenant(r, w, rid)
	if !ok {
		return
	}
	var req ingest.ReplayRequest
	if err := Decode(r, &req, s.config.MaxBodyBytes); err != nil {
		Fail(w, 400, rid, "invalid_request", err.Error(), nil)
		return
	}
	if len(req.LetterIDs) == 0 {
		Fail(w, 400, rid, "invalid_request", "letter_ids is required", nil)
		return
	}
	actor := req.Actor
	if actor == "" {
		actor = r.Header.Get("X-Actor-ID")
	}
	JSON(w, 202, rid, map[string]any{"tenant_id": tenant, "results": s.ingestion.ReplayManyForTenant(r.Context(), tenant, req.LetterIDs, actor)})
}
func (s *Server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = fmt.Fprintf(w, "ecqg_events_accepted_total %d\necqg_events_deadlettered_total %d\n", s.metrics.Accepted.Load(), s.metrics.DeadLettered.Load())
}
func (s *Server) requestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { next.ServeHTTP(w, r) })
}
func (s *Server) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				Fail(w, 500, RequestID(r), "internal_error", "unexpected server error", fmt.Sprint(v))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
func (s *Server) Shutdown(ctx context.Context) error {
	if s.shutdownHook == nil {
		return nil
	}
	return s.shutdownHook(ctx)
}

func parseTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return t
}

func queryInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}

var _ = json.RawMessage{}
var _ = time.Second
