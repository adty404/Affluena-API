-- 000030: Add color and icon to budgets, goals, installments, subscriptions,
-- and recurring transaction rules for richer UI metadata, mirroring the
-- existing wallet columns (000021/000029). Both are optional, backward
-- compatible (NOT NULL DEFAULT ''). The values are semantic strings owned by
-- the clients' icon catalog and color palette; the server does not validate
-- them (same as wallets).

ALTER TABLE category_budgets
    ADD COLUMN IF NOT EXISTS color text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS icon text NOT NULL DEFAULT '';

ALTER TABLE goals
    ADD COLUMN IF NOT EXISTS color text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS icon text NOT NULL DEFAULT '';

ALTER TABLE installments
    ADD COLUMN IF NOT EXISTS color text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS icon text NOT NULL DEFAULT '';

ALTER TABLE subscriptions
    ADD COLUMN IF NOT EXISTS color text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS icon text NOT NULL DEFAULT '';

ALTER TABLE recurring_transaction_rules
    ADD COLUMN IF NOT EXISTS color text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS icon text NOT NULL DEFAULT '';
