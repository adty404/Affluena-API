-- 000021: Add color and description to wallets for richer UI metadata.
-- Both optional, backward compatible (NOT NULL DEFAULT '').

ALTER TABLE wallets
    ADD COLUMN IF NOT EXISTS color text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS description text NOT NULL DEFAULT '';
