-- CYB-3680: content-addressed runtime-config blobs — the DB source of truth
-- behind the shared, owner-less runtime-config ConfigMaps. The in-cluster CM
-- is a disposable projection: self-heal layers rebuild it from these rows,
-- so the TTL janitor deleting a CM is never data loss.
CREATE TABLE "runtime_config_blobs" (
  "hash" text NOT NULL,
  "files" jsonb NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "last_used_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("hash")
);
