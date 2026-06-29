-- Account-level "partner" links: a one-way, read-only grant. When the partner
-- accepts, they get viewer access to ALL of the owner's wallets -- current and
-- any created later -- via auto-managed wallet_shares rows. Direction is
-- one-way: owner_id's wallets become visible to partner_id, not vice versa.
CREATE TABLE partner_links (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    partner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status text NOT NULL CHECK (status IN ('pending', 'joined', 'rejected')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (owner_id, partner_id),
    CHECK (owner_id <> partner_id)
);

CREATE INDEX partner_links_partner_idx ON partner_links (partner_id, status);

-- Tag the origin of a wallet share so revoking a partner only removes the
-- shares it auto-created ('partner'), never a manually-invited share ('manual').
-- Existing rows default to 'manual', preserving current behavior.
ALTER TABLE wallet_shares
    ADD COLUMN IF NOT EXISTS source text NOT NULL DEFAULT 'manual';
ALTER TABLE wallet_shares
    DROP CONSTRAINT IF EXISTS wallet_shares_source_check;
ALTER TABLE wallet_shares
    ADD CONSTRAINT wallet_shares_source_check CHECK (source IN ('manual', 'partner'));
