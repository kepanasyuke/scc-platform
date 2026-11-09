CREATE EXTENSION IF NOT EXISTS timescaledb;
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

CREATE SCHEMA IF NOT EXISTS telephony;
CREATE SCHEMA IF NOT EXISTS ops;

CREATE TABLE IF NOT EXISTS telephony.cdr (
    id BIGSERIAL,
    call_id UUID NOT NULL,
    direction TEXT NOT NULL,
    trunk TEXT NOT NULL,
    queue TEXT NOT NULL,
    agent_ext TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    answered_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    ring_seconds NUMERIC(10, 3) NOT NULL DEFAULT 0,
    talk_seconds NUMERIC(10, 3) NOT NULL DEFAULT 0,
    hangup_cause TEXT NOT NULL DEFAULT 'NORMAL_CLEARING',
    sla_met BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (id, started_at)
) PARTITION BY RANGE (started_at);

CREATE TABLE IF NOT EXISTS telephony.cdr_2026_11 PARTITION OF telephony.cdr
    FOR VALUES FROM ('2026-11-01') TO ('2026-12-01');

CREATE TABLE IF NOT EXISTS telephony.cdr_2026_12 PARTITION OF telephony.cdr
    FOR VALUES FROM ('2026-12-01') TO ('2027-01-01');

CREATE TABLE IF NOT EXISTS telephony.cdr_default PARTITION OF telephony.cdr DEFAULT;

CREATE INDEX IF NOT EXISTS cdr_queue_started_idx ON telephony.cdr (queue, started_at DESC);
CREATE INDEX IF NOT EXISTS cdr_trunk_started_idx ON telephony.cdr (trunk, started_at DESC);
CREATE INDEX IF NOT EXISTS cdr_answered_idx ON telephony.cdr (started_at DESC) WHERE answered_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS cdr_hangup_cause_idx ON telephony.cdr USING gin (to_tsvector('simple', hangup_cause));

CREATE TABLE IF NOT EXISTS telephony.alerts (
    id BIGSERIAL PRIMARY KEY,
    rule TEXT NOT NULL,
    severity TEXT NOT NULL,
    subject TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    acknowledged_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS alerts_open_idx ON telephony.alerts (created_at DESC) WHERE acknowledged_at IS NULL;
CREATE INDEX IF NOT EXISTS alerts_payload_idx ON telephony.alerts USING gin (payload jsonb_path_ops);

CREATE TABLE IF NOT EXISTS telephony.agents (
    ext TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    team TEXT NOT NULL,
    paused_until TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS ops.query_samples (
    id BIGSERIAL PRIMARY KEY,
    captured_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    queryid BIGINT NOT NULL,
    fingerprint TEXT NOT NULL,
    calls BIGINT NOT NULL,
    mean_ms NUMERIC(12, 3) NOT NULL,
    total_ms NUMERIC(14, 3) NOT NULL,
    rows_read BIGINT NOT NULL,
    plan JSONB
);

CREATE INDEX IF NOT EXISTS query_samples_fingerprint_idx ON ops.query_samples (fingerprint, captured_at DESC);
