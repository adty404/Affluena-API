ALTER TABLE installments
	DROP CONSTRAINT installments_user_wallet_fk;

ALTER TABLE subscriptions
	DROP CONSTRAINT subscriptions_user_wallet_fk;
