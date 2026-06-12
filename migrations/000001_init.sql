CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	email text NOT NULL UNIQUE,
	password_hash text NOT NULL,
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE refresh_tokens (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	token_hash text NOT NULL UNIQUE,
	expires_at timestamptz NOT NULL,
	revoked_at timestamptz,
	created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE wallets (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name text NOT NULL,
	type text NOT NULL CHECK (type IN ('cash', 'bank', 'e_wallet', 'investment')),
	currency_code text NOT NULL DEFAULT 'IDR',
	balance_minor bigint NOT NULL DEFAULT 0,
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now(),
	UNIQUE (user_id, name)
);

CREATE TABLE categories (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name text NOT NULL,
	type text NOT NULL CHECK (type IN ('income', 'expense')),
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now(),
	UNIQUE (user_id, name, type)
);

CREATE TABLE transactions (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	type text NOT NULL CHECK (type IN ('income', 'expense', 'transfer', 'adjustment')),
	wallet_id uuid NOT NULL REFERENCES wallets(id) ON DELETE RESTRICT,
	to_wallet_id uuid REFERENCES wallets(id) ON DELETE RESTRICT,
	category_id uuid REFERENCES categories(id) ON DELETE RESTRICT,
	amount_minor bigint NOT NULL,
	transaction_at timestamptz NOT NULL,
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

CREATE INDEX transactions_user_date_idx ON transactions(user_id, transaction_at DESC);
CREATE INDEX transactions_wallet_idx ON transactions(wallet_id);

CREATE TABLE quick_entry_templates (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name text NOT NULL,
	type text NOT NULL CHECK (type IN ('income', 'expense', 'transfer', 'adjustment')),
	wallet_id uuid NOT NULL REFERENCES wallets(id) ON DELETE RESTRICT,
	to_wallet_id uuid REFERENCES wallets(id) ON DELETE RESTRICT,
	category_id uuid REFERENCES categories(id) ON DELETE RESTRICT,
	amount_minor bigint NOT NULL,
	note text NOT NULL DEFAULT '',
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now(),
	UNIQUE (user_id, name),
	CHECK (amount_minor <> 0),
	CHECK (
		(type IN ('income', 'expense') AND category_id IS NOT NULL AND to_wallet_id IS NULL)
		OR (type = 'transfer' AND category_id IS NULL AND to_wallet_id IS NOT NULL AND to_wallet_id <> wallet_id)
		OR (type = 'adjustment' AND category_id IS NULL AND to_wallet_id IS NULL)
	)
);
