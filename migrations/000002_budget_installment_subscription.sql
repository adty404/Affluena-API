CREATE TABLE category_budgets (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	category_id uuid NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
	month date NOT NULL,
	limit_minor bigint NOT NULL CHECK (limit_minor > 0),
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now(),
	UNIQUE (user_id, category_id, month),
	CHECK (date_trunc('month', month)::date = month)
);

CREATE INDEX category_budgets_user_month_idx ON category_budgets(user_id, month);

CREATE TABLE installments (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name text NOT NULL,
	wallet_id uuid NOT NULL REFERENCES wallets(id) ON DELETE RESTRICT,
	category_id uuid NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
	total_amount_minor bigint NOT NULL CHECK (total_amount_minor > 0),
	monthly_amount_minor bigint NOT NULL CHECK (monthly_amount_minor > 0),
	tenor_months integer NOT NULL CHECK (tenor_months > 0),
	remaining_months integer NOT NULL CHECK (remaining_months >= 0),
	start_date date NOT NULL,
	due_day integer NOT NULL CHECK (due_day BETWEEN 1 AND 31),
	status text NOT NULL CHECK (status IN ('active', 'paid_off', 'cancelled')),
	note text NOT NULL DEFAULT '',
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now(),
	CHECK (remaining_months <= tenor_months)
);

CREATE INDEX installments_user_status_idx ON installments(user_id, status);

CREATE TABLE subscriptions (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name text NOT NULL,
	wallet_id uuid NOT NULL REFERENCES wallets(id) ON DELETE RESTRICT,
	category_id uuid NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
	amount_minor bigint NOT NULL CHECK (amount_minor > 0),
	billing_cycle text NOT NULL CHECK (billing_cycle IN ('weekly', 'monthly')),
	next_due_date date NOT NULL,
	status text NOT NULL CHECK (status IN ('active', 'paused', 'cancelled')),
	note text NOT NULL DEFAULT '',
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX subscriptions_user_status_due_idx ON subscriptions(user_id, status, next_due_date);
