CREATE TABLE goals (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name text NOT NULL,
	target_amount_minor bigint NOT NULL CHECK (target_amount_minor > 0),
	deadline date,
	status text NOT NULL CHECK (status IN ('active', 'achieved', 'cancelled')),
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX goals_user_idx ON goals(user_id);

CREATE TABLE goal_members (
	goal_id uuid NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	status text NOT NULL CHECK (status IN ('pending', 'joined', 'rejected')),
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now(),
	PRIMARY KEY (goal_id, user_id)
);

ALTER TABLE wallets DROP CONSTRAINT wallets_type_check;
ALTER TABLE wallets ADD CONSTRAINT wallets_type_check CHECK (type IN ('cash', 'bank', 'e_wallet', 'investment', 'goal'));
ALTER TABLE wallets ADD COLUMN goal_id uuid REFERENCES goals(id) ON DELETE CASCADE;

CREATE INDEX wallets_goal_idx ON wallets(goal_id);
