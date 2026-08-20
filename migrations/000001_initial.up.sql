CREATE TABLE IF NOT EXISTS contracts (
 id TEXT PRIMARY KEY, organization_id TEXT NOT NULL, name TEXT NOT NULL,
 compatibility_mode TEXT NOT NULL, revision BIGINT NOT NULL, created_at TIMESTAMPTZ NOT NULL,
 updated_at TIMESTAMPTZ NOT NULL, UNIQUE (organization_id, name)
);
CREATE TABLE IF NOT EXISTS contract_versions (
 contract_id TEXT NOT NULL REFERENCES contracts(id), version_number INTEGER NOT NULL,
 fingerprint TEXT NOT NULL, lifecycle TEXT NOT NULL, schema_document JSONB NOT NULL,
 created_at TIMESTAMPTZ NOT NULL, PRIMARY KEY (contract_id, version_number)
);
CREATE TABLE IF NOT EXISTS ingestion_attempts (
 id TEXT PRIMARY KEY, event_id TEXT NOT NULL, tenant_id TEXT NOT NULL, status TEXT NOT NULL,
 reasons JSONB NOT NULL, occurred_at TIMESTAMPTZ NOT NULL, duration_ms BIGINT NOT NULL
);
CREATE TABLE IF NOT EXISTS outbox (
 id TEXT PRIMARY KEY, aggregate_type TEXT NOT NULL, aggregate_id TEXT NOT NULL,
 event_type TEXT NOT NULL, payload JSONB NOT NULL, created_at TIMESTAMPTZ NOT NULL, published_at TIMESTAMPTZ
);
