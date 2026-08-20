#!/bin/sh
set -eu
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"
ECQG_ADDRESS=:18080 go run ./cmd/gateway > /tmp/ecqg-smoke.log 2>&1 &
pid=$!
cleanup() { kill "$pid" 2>/dev/null || true; wait "$pid" 2>/dev/null || true; }
trap cleanup EXIT INT TERM
ready=false
for n in 1 2 3 4 5 6 7 8 9 10; do
  if curl -fsS http://127.0.0.1:18080/healthz >/dev/null; then
    ready=true
    break
  fi
  sleep 1
done
if [ "$ready" != true ]; then
  cat /tmp/ecqg-smoke.log >&2
  exit 1
fi
contract=$(curl -fsS -X POST http://127.0.0.1:18080/api/v1/contracts -H 'X-Tenant-ID: smoke' -H 'Content-Type: application/json' -d '{"name":"orders.created","compatibility":"backward"}')
id=$(printf '%s' "$contract" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')
version=$(curl -fsS -X POST "http://127.0.0.1:18080/api/v1/contracts/$id/versions" -H 'X-Tenant-ID: smoke' -H 'Content-Type: application/json' -d '{"schema":{"kind":"json-schema","fields":[{"name":"order_id","type":"string","required":true}]}}')
etag=$(printf '%s' "$version" | sed -n 's/.*"etag":"\([^"]*\)".*/\1/p')
curl -fsS -X POST "http://127.0.0.1:18080/api/v1/contracts/$id/versions/1/publish" -H 'X-Tenant-ID: smoke' -H "If-Match: $etag" >/dev/null
curl -fsS -X POST http://127.0.0.1:18080/api/v1/ingest/events -H 'X-Tenant-ID: smoke' -H 'Idempotency-Key: smoke-1' -H 'Content-Type: application/json' -d "{\"contract_id\":\"$id\",\"schema_version\":1,\"payload\":{\"order_id\":\"o-1\"}}" >/dev/null
echo 'smoke passed'
