-- Rebrand the account-level "partner" link to "Berbagi Dompet" (wallet-history
-- sharing): an owner invites up to 5 people to view ALL their wallets, read
-- only. Drop every "partner" name from the schema. (The HTTP endpoints and Go
-- package stay under /partners for client compatibility — already-shipped apps
-- call /partners; renaming those needs a coordinated client release.)

ALTER TABLE partner_links RENAME TO wallet_share_links;
-- partner_id is the person granted view access -> viewer_id. (The UNIQUE and
-- CHECK constraints on this column follow the rename automatically.)
ALTER TABLE wallet_share_links RENAME COLUMN partner_id TO viewer_id;
ALTER INDEX partner_links_partner_idx RENAME TO wallet_share_links_viewer_idx;

-- Also drop "partner" from the auto-generated constraint names. RENAME
-- CONSTRAINT on a PK/UNIQUE also renames its backing index, so no separate
-- ALTER INDEX is needed for those two.
ALTER TABLE wallet_share_links RENAME CONSTRAINT partner_links_pkey TO wallet_share_links_pkey;
ALTER TABLE wallet_share_links RENAME CONSTRAINT partner_links_owner_id_partner_id_key TO wallet_share_links_owner_viewer_key;
ALTER TABLE wallet_share_links RENAME CONSTRAINT partner_links_owner_id_fkey TO wallet_share_links_owner_fkey;
ALTER TABLE wallet_share_links RENAME CONSTRAINT partner_links_partner_id_fkey TO wallet_share_links_viewer_fkey;
ALTER TABLE wallet_share_links RENAME CONSTRAINT partner_links_status_check TO wallet_share_links_status_check;
ALTER TABLE wallet_share_links RENAME CONSTRAINT partner_links_check TO wallet_share_links_owner_viewer_check;

-- wallet_shares.source tags auto-created shares. 'partner' (made by a share
-- link) becomes 'link'; 'manual' (a per-wallet invite) is unchanged. Drop the
-- old CHECK FIRST so the UPDATE to the new value is allowed, then re-add it.
ALTER TABLE wallet_shares DROP CONSTRAINT IF EXISTS wallet_shares_source_check;
UPDATE wallet_shares SET source = 'link' WHERE source = 'partner';
ALTER TABLE wallet_shares ADD CONSTRAINT wallet_shares_source_check CHECK (source IN ('manual', 'link'));
