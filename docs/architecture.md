# Architecture and Operations

## Data dictionary

`contracts` owns tenant-scoped event names, compatibility mode, revision and
timestamps. `contract_versions` owns the canonical schema fingerprint and
lifecycle. `ingestion_attempts` retains validation outcomes without copying
secret material. `outbox` is the durable handoff boundary for notifications.
Dead letters include original payload only in encrypted durable storage in a
production adapter; the memory adapter is explicitly development-only.

## Capacity model and SLO

At 1,000 requests/s with 1 KiB payloads the gateway needs approximately 1
MiB/s ingress. Validation is O(fields + rules); gateway admission limits body
size before decoding and uses fixed per-tenant minute windows. Target SLO is
99.9% successful accepted-event responses under 100 ms and 99.99% durability
for an enabled durable outbox. Scale ingestion partitions by partition key;
one partition is consumed in order and a worker lease fences duplicate owners.

## Threat model

Actors include a tenant producer, consumer, platform operator, compromised
API key, and an untrusted payload. Mitigations: tenant boundary at every
repository key, HMAC verification with secrets injected by reference, bounded
payloads, fixed rule DSL without code execution, timestamp replay window,
idempotency, redacted structured logs, mTLS for gRPC deployments, immutable
audit entries and least-privilege database roles. Payload fields must never be
used as log attributes. Keys rotate by overlapping secret references.

## Failure exercises

1. Stop an outbox worker while producing events; after restart verify lease
   expiry and idempotent delivery using the outbox identifier.
2. Send an event with a missing mandatory field; verify dead-letter lineage,
   replay audit and poison transition after its max replay count.
3. Deploy a schema that removes a required field; compatibility check must
   reject it before publish. Retry its publish with stale ETag; expect 412.
4. Flood a tenant beyond quota; expect 429 while other tenant traffic
   continues. Capture pprof only under an authorized incident window.
