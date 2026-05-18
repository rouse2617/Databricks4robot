-- 018_add_saved_queries.sql — persisted Query IR objects for saved structured queries.

CREATE TABLE IF NOT EXISTS saved_queries (
    saved_query_id   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name             TEXT         NOT NULL,
    description      TEXT,
    resource         TEXT         NOT NULL DEFAULT 'assets',
    schema_version   TEXT         NOT NULL DEFAULT 'v1',
    query_ir_json    JSONB        NOT NULL,
    owner            TEXT,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_saved_queries_resource_updated
    ON saved_queries (resource, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_saved_queries_owner_updated
    ON saved_queries (owner, updated_at DESC)
    WHERE owner IS NOT NULL;

DROP TRIGGER IF EXISTS trg_saved_queries_updated_at ON saved_queries;
CREATE TRIGGER trg_saved_queries_updated_at
    BEFORE UPDATE ON saved_queries
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
