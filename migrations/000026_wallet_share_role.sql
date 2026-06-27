-- Read-only sharing: a wallet_shares row now carries a role.
--   'member' = read + write (can record transactions in the shared wallet)
--   'viewer' = read only (can see the wallet + its transactions, cannot record)
-- Defaults to 'member' so every existing share keeps its current read+write
-- behavior; the access lifecycle (status: pending/joined/rejected) is unchanged.
ALTER TABLE wallet_shares
    ADD COLUMN IF NOT EXISTS role text NOT NULL DEFAULT 'member';

ALTER TABLE wallet_shares
    DROP CONSTRAINT IF EXISTS wallet_shares_role_check;
ALTER TABLE wallet_shares
    ADD CONSTRAINT wallet_shares_role_check CHECK (role IN ('viewer', 'member'));
