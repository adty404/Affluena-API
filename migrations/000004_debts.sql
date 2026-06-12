CREATE TABLE debts (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	type text NOT NULL CHECK (type IN ('receivable', 'payable')),
	counterparty_name text NOT NULL,
	wallet_id uuid NOT NULL REFERENCES wallets(id) ON DELETE RESTRICT,
	disbursement_category_id uuid NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
	payment_category_id uuid NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
	origination_transaction_id uuid NOT NULL REFERENCES transactions(id) ON DELETE RESTRICT,
	principal_amount_minor bigint NOT NULL CHECK (principal_amount_minor > 0),
	paid_amount_minor bigint NOT NULL DEFAULT 0 CHECK (paid_amount_minor >= 0),
	opened_at timestamptz NOT NULL,
	due_date date,
	status text NOT NULL CHECK (status IN ('open', 'partial', 'paid_off', 'cancelled')),
	note text NOT NULL DEFAULT '',
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now(),
	CHECK (paid_amount_minor <= principal_amount_minor)
);

CREATE INDEX debts_user_status_due_idx ON debts(user_id, status, due_date);
CREATE INDEX debts_user_opened_idx ON debts(user_id, opened_at DESC);
CREATE INDEX debts_wallet_idx ON debts(wallet_id);

CREATE TABLE debt_payments (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	debt_id uuid NOT NULL REFERENCES debts(id) ON DELETE CASCADE,
	transaction_id uuid NOT NULL REFERENCES transactions(id) ON DELETE RESTRICT,
	amount_minor bigint NOT NULL CHECK (amount_minor > 0),
	paid_at timestamptz NOT NULL,
	note text NOT NULL DEFAULT '',
	created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX debt_payments_debt_paid_idx ON debt_payments(debt_id, paid_at DESC);
CREATE INDEX debt_payments_user_created_idx ON debt_payments(user_id, created_at DESC);
