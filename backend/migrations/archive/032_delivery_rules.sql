-- CYB-1020: delivery_rules for pre-delivery compliance checks.

CREATE TABLE IF NOT EXISTS delivery_rules (
    rule_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    owner         TEXT NOT NULL,
    customer_id   TEXT REFERENCES customers(customer_id),
    query_dsl     JSONB NOT NULL,
    dsl_version   TEXT NOT NULL DEFAULT 'v1',
    enforce_mode  TEXT NOT NULL DEFAULT 'block'
                  CHECK (enforce_mode IN ('block', 'warn', 'tag_only')),
    rating_scope  TEXT NOT NULL DEFAULT 'current'
                  CHECK (rating_scope IN ('current', 'logical')),
    is_active     BOOLEAN NOT NULL DEFAULT true,
    version       BIGINT NOT NULL DEFAULT 1,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_drules_customer ON delivery_rules(customer_id, is_active);
CREATE INDEX IF NOT EXISTS idx_drules_active ON delivery_rules(is_active) WHERE is_active = true;
