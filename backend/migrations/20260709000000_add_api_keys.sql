-- CYB-3154: 统一鉴权 — SDK / API 调用方的 API Key。
--
-- 不透明 key,形如 `dbk_<prefix>_<secret>`:库里只存 secret 的哈希(明文仅
-- 创建时返回一次)。每个 key 自带 scopes 实现最小权限授权、owner 便于审计,
-- 可按 status='revoked' 单独吊销、不影响其他调用方。与旧静态 token(admin)
-- 和网页 JWT 会话并存,分派逻辑见 internal/middleware/auth.go。
--
-- 幂等(IF NOT EXISTS):部分应用后重跑也是 no-op。
CREATE TABLE IF NOT EXISTS "api_keys" (
  "id"           uuid NOT NULL DEFAULT gen_random_uuid(),
  "key_prefix"   text NOT NULL,
  "secret_hash"  text NOT NULL,
  "name"         text NOT NULL DEFAULT '',
  "owner"        text NOT NULL DEFAULT '',
  "scopes"       text[] NOT NULL DEFAULT '{}',
  "status"       text NOT NULL DEFAULT 'active',
  "expires_at"   timestamptz NULL,
  "last_used_at" timestamptz NULL,
  "created_by"   text NOT NULL DEFAULT '',
  "created_at"   timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "api_keys_status_check" CHECK (status = ANY (ARRAY['active'::text, 'revoked'::text]))
);
CREATE UNIQUE INDEX IF NOT EXISTS "api_keys_key_prefix_key" ON "api_keys" ("key_prefix");
CREATE INDEX IF NOT EXISTS "idx_api_keys_status" ON "api_keys" ("status");
