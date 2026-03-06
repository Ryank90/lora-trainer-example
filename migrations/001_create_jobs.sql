-- +migrate Up
CREATE TABLE IF NOT EXISTS jobs (
    id              TEXT PRIMARY KEY,
    status          TEXT NOT NULL DEFAULT 'pending',
    model_type      TEXT NOT NULL,
    request         JSONB NOT NULL,
    result          JSONB,
    error           TEXT,
    provider        TEXT,
    gpu_type        TEXT,
    idempotency_key TEXT UNIQUE,

    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    provisioned_at   TIMESTAMPTZ,
    download_start_at TIMESTAMPTZ,
    training_start_at TIMESTAMPTZ,
    upload_start_at  TIMESTAMPTZ,
    completed_at     TIMESTAMPTZ,
    failed_at        TIMESTAMPTZ,
    cancelled_at     TIMESTAMPTZ
);

CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_model_type ON jobs(model_type);
CREATE INDEX idx_jobs_created_at ON jobs(created_at DESC);
CREATE INDEX idx_jobs_idempotency_key ON jobs(idempotency_key) WHERE idempotency_key IS NOT NULL;

-- +migrate Down
DROP TABLE IF EXISTS jobs;
