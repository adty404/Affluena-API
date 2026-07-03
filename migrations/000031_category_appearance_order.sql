-- 000031: Add icon, color, and position to categories.
--
-- icon/color add richer UI metadata, mirroring the existing wallet columns
-- (000021/000029/000030). Both are optional, backward compatible
-- (NOT NULL DEFAULT ''). The values are semantic strings owned by the
-- clients' icon catalog and color palette; the server does not validate
-- them (same as wallets).
--
-- position is a user-arrangeable, per-user sort order (0-based). It is
-- managed by PUT /api/v1/categories/reorder; new categories append at
-- MAX(position)+1 for the user. Existing rows are backfilled below with a
-- stable per-user sequence (created_at, then id) so current users get a
-- deterministic starting order.

ALTER TABLE categories
    ADD COLUMN IF NOT EXISTS icon text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS color text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS position integer NOT NULL DEFAULT 0;

-- Backfill runs after the column exists and before the index so populated
-- databases start from a deterministic order. Pure UPDATE on a fresh column;
-- no constraints depend on position, so statement order has no other
-- interplay.
UPDATE categories
SET position = ranked.rn
FROM (
    SELECT id, row_number() OVER (PARTITION BY user_id ORDER BY created_at, id) - 1 AS rn
    FROM categories
) ranked
WHERE categories.id = ranked.id;

CREATE INDEX IF NOT EXISTS idx_categories_user_position ON categories(user_id, position);
