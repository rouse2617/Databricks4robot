-- CYB-1014: customers table + deliveries.customer_id FK (backfill placeholders).

CREATE TABLE IF NOT EXISTS customers (
    customer_id      TEXT PRIMARY KEY,
    display_name     TEXT NOT NULL,
    legal_name       TEXT,
    status           TEXT NOT NULL DEFAULT 'active'
                     CHECK (status IN ('active', 'trial', 'suspended', 'offboarded')),
    region           TEXT,
    sla_tier         TEXT NOT NULL DEFAULT 'standard'
                     CHECK (sla_tier IN ('standard', 'premium', 'enterprise')),
    account_owner    TEXT,
    compliance_tags  JSONB NOT NULL DEFAULT '[]'::jsonb,
    exclude_tags     JSONB NOT NULL DEFAULT '[]'::jsonb,
    metadata         JSONB NOT NULL DEFAULT '{}'::jsonb,
    extra            JSONB NOT NULL DEFAULT '{}'::jsonb,
    onboarded_at     TIMESTAMPTZ,
    offboarded_at    TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    row_version      BIGINT NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_customers_status ON customers(status);
CREATE INDEX IF NOT EXISTS idx_customers_sla ON customers(sla_tier);
CREATE INDEX IF NOT EXISTS idx_customers_account_owner
    ON customers(account_owner) WHERE account_owner IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_customers_region ON customers(region) WHERE region IS NOT NULL;

-- Placeholder rows for every customer_id already referenced by deliveries.
INSERT INTO customers (customer_id, display_name, status, metadata)
SELECT DISTINCT TRIM(d.customer_id),
       TRIM(d.customer_id),
       'active',
       jsonb_build_object('migrated', true, 'source', '029_customers_backfill')
FROM deliveries d
WHERE TRIM(d.customer_id) <> ''
ON CONFLICT (customer_id) DO NOTHING;

CREATE INDEX IF NOT EXISTS idx_deliveries_customer_created
    ON deliveries (customer_id, created_at DESC);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_deliveries_customer'
    ) THEN
        ALTER TABLE deliveries
            ADD CONSTRAINT fk_deliveries_customer
            FOREIGN KEY (customer_id) REFERENCES customers(customer_id);
    END IF;
END $$;
