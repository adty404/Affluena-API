CREATE TABLE recurring_transaction_rules (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name text NOT NULL,
	type text NOT NULL CHECK (type IN ('income', 'expense', 'transfer', 'adjustment')),
	wallet_id uuid NOT NULL REFERENCES wallets(id) ON DELETE RESTRICT,
	to_wallet_id uuid REFERENCES wallets(id) ON DELETE RESTRICT,
	category_id uuid REFERENCES categories(id) ON DELETE RESTRICT,
	amount_minor bigint NOT NULL,
	frequency text NOT NULL CHECK (frequency IN ('weekly', 'monthly')),
	interval_count integer NOT NULL DEFAULT 1 CHECK (interval_count > 0),
	next_run_at timestamptz NOT NULL,
	end_at timestamptz,
	last_run_at timestamptz,
	status text NOT NULL CHECK (status IN ('active', 'paused', 'cancelled')),
	note text NOT NULL DEFAULT '',
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now(),
	CHECK (amount_minor <> 0),
	CHECK (
		(type IN ('income', 'expense') AND category_id IS NOT NULL AND to_wallet_id IS NULL)
		OR (type = 'transfer' AND category_id IS NULL AND to_wallet_id IS NOT NULL AND to_wallet_id <> wallet_id)
		OR (type = 'adjustment' AND category_id IS NULL AND to_wallet_id IS NULL)
	)
);

CREATE INDEX recurring_rules_due_idx ON recurring_transaction_rules(status, next_run_at)
	WHERE status = 'active';
CREATE INDEX recurring_rules_user_idx ON recurring_transaction_rules(user_id, status, next_run_at);

CREATE TABLE recurring_transaction_runs (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	rule_id uuid NOT NULL REFERENCES recurring_transaction_rules(id) ON DELETE CASCADE,
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	scheduled_for timestamptz NOT NULL,
	transaction_id uuid REFERENCES transactions(id) ON DELETE SET NULL,
	run_type text NOT NULL CHECK (run_type IN ('manual', 'scheduled')),
	created_at timestamptz NOT NULL DEFAULT now(),
	UNIQUE (rule_id, scheduled_for)
);

CREATE INDEX recurring_runs_user_created_idx ON recurring_transaction_runs(user_id, created_at DESC);
