-- CDR Ingest: партиционирование по дням, дедупликация, партиции «на будущее».
-- Выполняется ролью scc при старте сервиса (идемпотентно).

CREATE TABLE IF NOT EXISTS telephony.cdr (
    id BIGSERIAL,
    call_id UUID NOT NULL,
    direction TEXT NOT NULL CHECK (direction IN ('inbound', 'outbound')),
    trunk TEXT NOT NULL,
    queue TEXT NOT NULL,
    agent_ext TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    answered_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ NOT NULL,
    ring_seconds NUMERIC(10, 3) NOT NULL DEFAULT 0 CHECK (ring_seconds >= 0),
    talk_seconds NUMERIC(10, 3) NOT NULL DEFAULT 0 CHECK (talk_seconds >= 0),
    hangup_cause TEXT NOT NULL DEFAULT 'NORMAL_CLEARING',
    sla_met BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (id, started_at),
    UNIQUE (call_id, started_at)
) PARTITION BY RANGE (started_at);

-- Идемпотентная запись батча: конфликт по call_id не рвёт транзакцию.
CREATE OR REPLACE FUNCTION telephony.upsert_cdr_batch(payload JSONB)
RETURNS TABLE (accepted INT, duplicates INT) AS $$
BEGIN
    RETURN QUERY
    WITH source AS (
        SELECT *
        FROM jsonb_to_recordset(payload) AS r(
            call_id UUID, direction TEXT, trunk TEXT, queue TEXT, agent_ext TEXT,
            started_at TIMESTAMPTZ, answered_at TIMESTAMPTZ, ended_at TIMESTAMPTZ,
            ring_seconds NUMERIC, talk_seconds NUMERIC, hangup_cause TEXT, sla_met BOOLEAN
        )
    ), inserted AS (
        INSERT INTO telephony.cdr (call_id, direction, trunk, queue, agent_ext, started_at,
                                   answered_at, ended_at, ring_seconds, talk_seconds, hangup_cause, sla_met)
        SELECT call_id, direction, trunk, queue, agent_ext, started_at,
               answered_at, ended_at, ring_seconds, talk_seconds, hangup_cause, sla_met
        FROM source
        ON CONFLICT (call_id, started_at) DO NOTHING
        RETURNING 1
    )
    SELECT (SELECT count(*) FROM inserted)::INT,
           (SELECT count(*) FROM source)::INT - (SELECT count(*) FROM inserted)::INT;
END;
$$ LANGUAGE plpgsql;

-- Партиция на сутки: имя детерминированное, создание идемпотентное.
CREATE OR REPLACE FUNCTION telephony.ensure_cdr_partition(day DATE)
RETURNS TEXT AS $$
DECLARE
    partition_name TEXT := format('cdr_%s', to_char(day, 'YYYY_MM_DD'));
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = partition_name) THEN
        EXECUTE format(
            'CREATE TABLE telephony.%I PARTITION OF telephony.cdr FOR VALUES FROM (%L) TO (%L)',
            partition_name, day, day + 1
        );
    END IF;
    RETURN partition_name;
END;
$$ LANGUAGE plpgsql;

-- Порядок захвата блокировок фиксирован (trunk -> queue), поэтому deadlock
-- между инжестом и распределением вызовов не возникает: сначала строка агента,
-- затем строка очереди, ровно в этом порядке.
CREATE OR REPLACE FUNCTION telephony.assign_agent(p_queue TEXT, p_ext TEXT)
RETURNS BOOLEAN AS $$
DECLARE
    agent_row telephony.agents%ROWTYPE;
BEGIN
    SELECT * INTO agent_row FROM telephony.agents WHERE ext = p_ext FOR UPDATE SKIP LOCKED;
    IF agent_row.ext IS NULL THEN
        RETURN false;
    END IF;
    UPDATE telephony.agents SET paused_until = NULL WHERE ext = p_ext;
    RETURN true;
END;
$$ LANGUAGE plpgsql;

CREATE INDEX IF NOT EXISTS cdr_started_brin ON telephony.cdr USING brin (started_at) WITH (pages_per_range = 32);
CREATE INDEX IF NOT EXISTS cdr_queue_started_idx ON telephony.cdr (queue, started_at DESC);
CREATE INDEX IF NOT EXISTS cdr_answered_partial_idx ON telephony.cdr (queue, started_at DESC)
    WHERE answered_at IS NOT NULL AND sla_met = false;
CREATE INDEX IF NOT EXISTS cdr_cause_expression_idx ON telephony.cdr (lower(hangup_cause), started_at DESC);
CREATE INDEX IF NOT EXISTS cdr_payload_gin ON telephony.cdr USING gin (to_tsvector('simple', trunk || ' ' || queue));
