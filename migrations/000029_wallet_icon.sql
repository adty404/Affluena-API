-- 000029: Add icon to wallets for richer UI metadata (a client-chosen icon
-- identifier, e.g. "bank", "cash", "gift"). Optional, backward compatible
-- (NOT NULL DEFAULT ''), mirroring the existing color column. The value is a
-- semantic string owned by the clients' icon catalog; the server does not
-- validate it (same as color).

ALTER TABLE wallets
    ADD COLUMN IF NOT EXISTS icon text NOT NULL DEFAULT '';
