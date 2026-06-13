ALTER TABLE wallets
	ADD CONSTRAINT wallets_user_id_id_unique UNIQUE (user_id, id);

ALTER TABLE categories
	ADD CONSTRAINT categories_user_id_id_unique UNIQUE (user_id, id);

ALTER TABLE transactions
	ADD CONSTRAINT transactions_user_id_id_unique UNIQUE (user_id, id);

ALTER TABLE recurring_transaction_rules
	ADD CONSTRAINT recurring_rules_user_id_id_unique UNIQUE (user_id, id);

ALTER TABLE debts
	ADD CONSTRAINT debts_user_id_id_unique UNIQUE (user_id, id);

ALTER TABLE transactions
	ADD CONSTRAINT transactions_user_wallet_fk
		FOREIGN KEY (user_id, wallet_id) REFERENCES wallets(user_id, id) ON DELETE RESTRICT,
	ADD CONSTRAINT transactions_user_to_wallet_fk
		FOREIGN KEY (user_id, to_wallet_id) REFERENCES wallets(user_id, id) ON DELETE RESTRICT,
	ADD CONSTRAINT transactions_user_category_fk
		FOREIGN KEY (user_id, category_id) REFERENCES categories(user_id, id) ON DELETE RESTRICT;

ALTER TABLE quick_entry_templates
	ADD CONSTRAINT quick_entry_templates_user_wallet_fk
		FOREIGN KEY (user_id, wallet_id) REFERENCES wallets(user_id, id) ON DELETE RESTRICT,
	ADD CONSTRAINT quick_entry_templates_user_to_wallet_fk
		FOREIGN KEY (user_id, to_wallet_id) REFERENCES wallets(user_id, id) ON DELETE RESTRICT,
	ADD CONSTRAINT quick_entry_templates_user_category_fk
		FOREIGN KEY (user_id, category_id) REFERENCES categories(user_id, id) ON DELETE RESTRICT;

ALTER TABLE category_budgets
	ADD CONSTRAINT category_budgets_user_category_fk
		FOREIGN KEY (user_id, category_id) REFERENCES categories(user_id, id) ON DELETE CASCADE;

ALTER TABLE installments
	ADD CONSTRAINT installments_user_wallet_fk
		FOREIGN KEY (user_id, wallet_id) REFERENCES wallets(user_id, id) ON DELETE RESTRICT,
	ADD CONSTRAINT installments_user_category_fk
		FOREIGN KEY (user_id, category_id) REFERENCES categories(user_id, id) ON DELETE RESTRICT;

ALTER TABLE subscriptions
	ADD CONSTRAINT subscriptions_user_wallet_fk
		FOREIGN KEY (user_id, wallet_id) REFERENCES wallets(user_id, id) ON DELETE RESTRICT,
	ADD CONSTRAINT subscriptions_user_category_fk
		FOREIGN KEY (user_id, category_id) REFERENCES categories(user_id, id) ON DELETE RESTRICT;

ALTER TABLE recurring_transaction_rules
	ADD CONSTRAINT recurring_rules_user_wallet_fk
		FOREIGN KEY (user_id, wallet_id) REFERENCES wallets(user_id, id) ON DELETE RESTRICT,
	ADD CONSTRAINT recurring_rules_user_to_wallet_fk
		FOREIGN KEY (user_id, to_wallet_id) REFERENCES wallets(user_id, id) ON DELETE RESTRICT,
	ADD CONSTRAINT recurring_rules_user_category_fk
		FOREIGN KEY (user_id, category_id) REFERENCES categories(user_id, id) ON DELETE RESTRICT;

ALTER TABLE recurring_transaction_runs
	ADD CONSTRAINT recurring_runs_user_rule_fk
		FOREIGN KEY (user_id, rule_id) REFERENCES recurring_transaction_rules(user_id, id) ON DELETE CASCADE,
	ADD CONSTRAINT recurring_runs_user_transaction_fk
		FOREIGN KEY (user_id, transaction_id) REFERENCES transactions(user_id, id) ON DELETE SET NULL (transaction_id);

ALTER TABLE debts
	ADD CONSTRAINT debts_user_wallet_fk
		FOREIGN KEY (user_id, wallet_id) REFERENCES wallets(user_id, id) ON DELETE RESTRICT,
	ADD CONSTRAINT debts_user_disbursement_category_fk
		FOREIGN KEY (user_id, disbursement_category_id) REFERENCES categories(user_id, id) ON DELETE RESTRICT,
	ADD CONSTRAINT debts_user_payment_category_fk
		FOREIGN KEY (user_id, payment_category_id) REFERENCES categories(user_id, id) ON DELETE RESTRICT,
	ADD CONSTRAINT debts_user_origination_transaction_fk
		FOREIGN KEY (user_id, origination_transaction_id) REFERENCES transactions(user_id, id) ON DELETE RESTRICT;

ALTER TABLE debt_payments
	ADD CONSTRAINT debt_payments_user_debt_fk
		FOREIGN KEY (user_id, debt_id) REFERENCES debts(user_id, id) ON DELETE CASCADE,
	ADD CONSTRAINT debt_payments_user_transaction_fk
		FOREIGN KEY (user_id, transaction_id) REFERENCES transactions(user_id, id) ON DELETE RESTRICT;
