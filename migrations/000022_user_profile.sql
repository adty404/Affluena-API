-- 000022: Add user profile fields + session metadata for auth endpoints.
-- All optional, backward compatible.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS name text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS avatar_url text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS password_changed_at timestamptz NOT NULL DEFAULT now();

ALTER TABLE refresh_tokens
    ADD COLUMN IF NOT EXISTS user_agent text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS ip_address text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_used_at timestamptz;
