-- Payment history for installments & subscriptions (mirrors debt_payments).
-- transaction_id is ON DELETE CASCADE (not RESTRICT like debt_payments):
-- tracker payment transactions have always been deletable via
-- DELETE /api/v1/transactions/:id, and deleting one reverses the wallet
-- balance — so its history row must disappear with it instead of blocking
-- the delete.
CREATE TABLE installment_payments (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	installment_id uuid NOT NULL REFERENCES installments(id) ON DELETE CASCADE,
	transaction_id uuid NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
	amount_minor bigint NOT NULL CHECK (amount_minor > 0),
	paid_at timestamptz NOT NULL,
	note text NOT NULL DEFAULT '',
	created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX installment_payments_installment_paid_idx ON installment_payments(installment_id, paid_at DESC);
CREATE INDEX installment_payments_transaction_idx ON installment_payments(transaction_id);

CREATE TABLE subscription_payments (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	subscription_id uuid NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
	transaction_id uuid NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
	amount_minor bigint NOT NULL CHECK (amount_minor > 0),
	paid_at timestamptz NOT NULL,
	note text NOT NULL DEFAULT '',
	created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX subscription_payments_subscription_paid_idx ON subscription_payments(subscription_id, paid_at DESC);
CREATE INDEX subscription_payments_transaction_idx ON subscription_payments(transaction_id);
