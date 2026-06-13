CREATE TABLE wallet_shares (
    wallet_id uuid NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status text NOT NULL CHECK (status IN ('pending', 'joined', 'rejected')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (wallet_id, user_id)
);

-- Drop the composite constraints that prevent users from inserting transactions into shared wallets
ALTER TABLE transactions
    DROP CONSTRAINT transactions_user_wallet_fk,
    DROP CONSTRAINT transactions_user_to_wallet_fk;
